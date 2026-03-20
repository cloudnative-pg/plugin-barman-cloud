/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// LifecycleImplementation is the implementation of the lifecycle handler
type LifecycleImplementation struct {
	lifecycle.UnimplementedOperatorLifecycleServer
	Client client.Client
}

// GetCapabilities exposes the lifecycle capabilities
func (impl LifecycleImplementation) GetCapabilities(
	_ context.Context,
	_ *lifecycle.OperatorLifecycleCapabilitiesRequest,
) (*lifecycle.OperatorLifecycleCapabilitiesResponse, error) {
	return &lifecycle.OperatorLifecycleCapabilitiesResponse{
		LifecycleCapabilities: []*lifecycle.OperatorLifecycleCapabilities{
			{
				Group: "",
				Kind:  "Pod",
				OperationTypes: []*lifecycle.OperatorOperationType{
					{
						Type: lifecycle.OperatorOperationType_TYPE_CREATE,
					},
					{
						Type: lifecycle.OperatorOperationType_TYPE_EVALUATE,
					},
				},
			},
			{
				Group: batchv1.GroupName,
				Kind:  "Job",
				OperationTypes: []*lifecycle.OperatorOperationType{
					{
						Type: lifecycle.OperatorOperationType_TYPE_CREATE,
					},
				},
			},
		},
	}, nil
}

// LifecycleHook is called when creating Kubernetes services
func (impl LifecycleImplementation) LifecycleHook(
	ctx context.Context,
	request *lifecycle.OperatorLifecycleRequest,
) (*lifecycle.OperatorLifecycleResponse, error) {
	contextLogger := log.FromContext(ctx).WithName("lifecycle")
	contextLogger.Info("Lifecycle hook reconciliation start")
	operation := request.GetOperationType().GetType().Enum()
	if operation == nil {
		return nil, errors.New("no operation set")
	}

	kind, err := object.GetKind(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	var cluster cnpgv1.Cluster
	if err := decoder.DecodeObjectLenient(
		request.GetClusterDefinition(),
		&cluster,
	); err != nil {
		return nil, err
	}

	pluginConfiguration := config.NewFromCluster(&cluster)

	// barman object is required for both the archive and restore process
	if err := pluginConfiguration.Validate(); err != nil {
		contextLogger.Info("pluginConfiguration invalid, skipping lifecycle", "error", err)
		return nil, nil
	}

	switch kind {
	case "Pod":
		contextLogger.Info("Reconciling pod")
		return impl.reconcilePod(ctx, &cluster, request, pluginConfiguration)
	case "Job":
		contextLogger.Info("Reconciling job")
		return impl.reconcileJob(ctx, &cluster, request, pluginConfiguration)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func (impl LifecycleImplementation) reconcileJob(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	pluginConfiguration *config.PluginConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {
	env, err := impl.collectAdditionalEnvs(ctx, cluster.Namespace, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	certificates, err := impl.collectAdditionalCertificates(ctx, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	resources, err := impl.collectSidecarResourcesForRecoveryJob(ctx, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	return reconcileJob(ctx, cluster, request, sidecarConfiguration{
		env:          env,
		certificates: certificates,
		resources:    resources,
	})
}

type sidecarConfiguration struct {
	env            []corev1.EnvVar
	certificates   []corev1.VolumeProjection
	resources      corev1.ResourceRequirements
	additionalArgs []string
}

func reconcileJob(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	config sidecarConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {
	contextLogger := log.FromContext(ctx).WithName("lifecycle")
	if pluginConfig := cluster.GetRecoverySourcePlugin(); pluginConfig == nil || pluginConfig.Name != metadata.PluginName {
		contextLogger.Debug("cluster does not use the this plugin for recovery, skipping")
		return nil, nil
	}

	var job batchv1.Job
	if err := decoder.DecodeObjectStrict(
		request.GetObjectDefinition(),
		&job,
		batchv1.SchemeGroupVersion.WithKind("Job"),
	); err != nil {
		contextLogger.Error(err, "failed to decode job")
		return nil, err
	}

	contextLogger = log.FromContext(ctx).WithName("plugin-barman-cloud-lifecycle").
		WithValues("jobName", job.Name)
	contextLogger.Debug("starting job reconciliation")

	jobRole := getCNPGJobRole(&job)
	if jobRole != "full-recovery" &&
		jobRole != "snapshot-recovery" {
		contextLogger.Debug("job is not a recovery job, skipping")
		return nil, nil
	}

	mutatedJob := job.DeepCopy()

	if err := reconcilePodSpec(
		cluster,
		&mutatedJob.Spec.Template.Spec,
		jobRole,
		corev1.Container{
			Args: []string{"restore"},
		},
		config,
	); err != nil {
		return nil, fmt.Errorf("while reconciling pod spec for job: %w", err)
	}

	patch, err := object.CreatePatch(mutatedJob, &job)
	if err != nil {
		return nil, err
	}

	contextLogger.Debug("generated patch", "content", string(patch))
	return &lifecycle.OperatorLifecycleResponse{
		JsonPatch: patch,
	}, nil
}

func (impl LifecycleImplementation) reconcilePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	pluginConfiguration *config.PluginConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {
	env, err := impl.collectAdditionalEnvs(ctx, cluster.Namespace, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	certificates, err := impl.collectAdditionalCertificates(ctx, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	resources, err := impl.collectSidecarResourcesForPod(ctx, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	additionalArgs, err := impl.collectAdditionalInstanceArgs(ctx, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	return reconcileInstancePod(ctx, cluster, request, pluginConfiguration, sidecarConfiguration{
		env:            env,
		certificates:   certificates,
		resources:      resources,
		additionalArgs: additionalArgs,
	})
}

func (impl LifecycleImplementation) collectAdditionalInstanceArgs(
	ctx context.Context,
	pluginConfiguration *config.PluginConfiguration,
) ([]string, error) {
	collectTypedAdditionalArgs := func(store *barmancloudv1.ObjectStore) []string {
		if store == nil {
			return nil
		}

		var args []string
		if len(store.Spec.InstanceSidecarConfiguration.LogLevel) > 0 {
			args = append(args, fmt.Sprintf("--log-level=%s", store.Spec.InstanceSidecarConfiguration.LogLevel))
		}

		return args
	}

	// Prefer the cluster object store (backup/archive). If not set, fallback to the recovery object store.
	// If neither is configured, no additional args are provided.
	if len(pluginConfiguration.BarmanObjectName) > 0 {
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, pluginConfiguration.GetBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, fmt.Errorf("while getting barman object store %s: %w",
				pluginConfiguration.GetBarmanObjectKey().String(), err)
		}
		args := barmanObjectStore.Spec.InstanceSidecarConfiguration.AdditionalContainerArgs
		args = append(
			args,
			collectTypedAdditionalArgs(&barmanObjectStore)...,
		)
		return args, nil
	}

	if len(pluginConfiguration.RecoveryBarmanObjectName) > 0 {
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, pluginConfiguration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, fmt.Errorf("while getting recovery barman object store %s: %w",
				pluginConfiguration.GetRecoveryBarmanObjectKey().String(), err)
		}
		args := barmanObjectStore.Spec.InstanceSidecarConfiguration.AdditionalContainerArgs
		args = append(
			args,
			collectTypedAdditionalArgs(&barmanObjectStore)...,
		)
		return args, nil
	}

	return nil, nil
}

func reconcileInstancePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	pluginConfiguration *config.PluginConfiguration,
	config sidecarConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {
	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	contextLogger := log.FromContext(ctx).WithName("plugin-barman-cloud-lifecycle").
		WithValues("podName", pod.Name)

	mutatedPod := pod.DeepCopy()

	if len(pluginConfiguration.BarmanObjectName) != 0 ||
		len(pluginConfiguration.ReplicaSourceBarmanObjectName) != 0 {
		if err := reconcilePodSpec(
			cluster,
			&mutatedPod.Spec,
			"postgres",
			corev1.Container{
				Args: []string{"instance"},
			},
			config,
		); err != nil {
			return nil, fmt.Errorf("while reconciling pod spec for pod: %w", err)
		}
	} else {
		contextLogger.Debug("No need to mutate instance with no backup & archiving configuration")
	}

	patch, err := object.CreatePatch(mutatedPod, pod)
	if err != nil {
		return nil, err
	}

	contextLogger.Debug("generated patch", "content", string(patch))
	return &lifecycle.OperatorLifecycleResponse{
		JsonPatch: patch,
	}, nil
}

func reconcilePodSpec(
	cluster *cnpgv1.Cluster,
	spec *corev1.PodSpec,
	mainContainerName string,
	sidecarTemplate corev1.Container,
	config sidecarConfiguration,
) error {
	envs := make([]corev1.EnvVar, 0, 5+len(config.env))
	envs = append(envs,
		corev1.EnvVar{
			Name:  "NAMESPACE",
			Value: cluster.Namespace,
		},
		corev1.EnvVar{
			Name:  "CLUSTER_NAME",
			Value: cluster.Name,
		},
		corev1.EnvVar{
			// TODO: should we really use this one?
			// should we mount an emptyDir volume just for that?
			Name:  "SPOOL_DIRECTORY",
			Value: "/controller/wal-restore-spool",
		},
		corev1.EnvVar{
			Name:  "CUSTOM_CNPG_GROUP",
			Value: cluster.GetObjectKind().GroupVersionKind().Group,
		},
		corev1.EnvVar{
			Name:  "CUSTOM_CNPG_VERSION",
			Value: cluster.GetObjectKind().GroupVersionKind().Version,
		},
	)

	envs = append(envs, config.env...)

	baseProbe := &corev1.Probe{
		FailureThreshold: 10,
		TimeoutSeconds:   10,
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/manager", "healthcheck", "unix"},
			},
		},
	}

	// fixed values
	sidecarTemplate.Name = "plugin-barman-cloud"
	sidecarTemplate.Image = viper.GetString("sidecar-image")
	sidecarTemplate.ImagePullPolicy = cluster.Spec.ImagePullPolicy
	sidecarTemplate.StartupProbe = baseProbe.DeepCopy()
	sidecarTemplate.SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
		Privileged:               ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
	sidecarTemplate.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
	sidecarTemplate.Resources = config.resources
	sidecarTemplate.Args = append(sidecarTemplate.Args, config.additionalArgs...)

	// merge the main container envs if they aren't already set
	for _, container := range spec.Containers {
		if container.Name == mainContainerName {
			for _, env := range container.Env {
				found := false
				for _, existingEnv := range sidecarTemplate.Env {
					if existingEnv.Name == env.Name {
						found = true
						break
					}
				}
				if !found {
					sidecarTemplate.Env = append(sidecarTemplate.Env, env)
				}
			}
			break
		}
	}

	// merge the default envs if they aren't already set
	for _, env := range envs {
		found := false
		for _, existingEnv := range sidecarTemplate.Env {
			if existingEnv.Name == env.Name {
				found = true
				break
			}
		}
		if !found {
			sidecarTemplate.Env = append(sidecarTemplate.Env, env)
		}
	}

	if len(config.certificates) > 0 {
		sidecarTemplate.VolumeMounts = ensureVolumeMount(
			sidecarTemplate.VolumeMounts,
			corev1.VolumeMount{
				Name:      barmanCertificatesVolumeName,
				MountPath: metadata.BarmanCertificatesPath,
			})

		spec.Volumes = ensureVolume(spec.Volumes, corev1.Volume{
			Name: barmanCertificatesVolumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: config.certificates,
				},
			},
		})
	} else {
		sidecarTemplate.VolumeMounts = removeVolumeMount(sidecarTemplate.VolumeMounts, barmanCertificatesVolumeName)
		spec.Volumes = removeVolume(spec.Volumes, barmanCertificatesVolumeName)
	}

	if err := injectPluginSidecarPodSpec(spec, &sidecarTemplate, mainContainerName); err != nil {
		return err
	}

	return nil
}

// TODO: move to machinery once the logic is finalized

// InjectPluginVolumePodSpec injects the plugin volume into a CNPG Pod spec.
func InjectPluginVolumePodSpec(spec *corev1.PodSpec, mainContainerName string) {
	const (
		pluginVolumeName = "plugins"
		pluginMountPath  = "/plugins"
	)

	foundPluginVolume := false
	for i := range spec.Volumes {
		if spec.Volumes[i].Name == pluginVolumeName {
			foundPluginVolume = true
		}
	}

	if foundPluginVolume {
		return
	}

	spec.Volumes = ensureVolume(spec.Volumes, corev1.Volume{
		Name: pluginVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	for i := range spec.Containers {
		if spec.Containers[i].Name == mainContainerName {
			spec.Containers[i].VolumeMounts = ensureVolumeMount(
				spec.Containers[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      pluginVolumeName,
					MountPath: pluginMountPath,
				},
			)
		}
	}
}

// injectPluginSidecarPodSpec injects a plugin sidecar into a CNPG Pod spec.
func injectPluginSidecarPodSpec(
	spec *corev1.PodSpec,
	sidecar *corev1.Container,
	mainContainerName string,
) error {
	sidecar = sidecar.DeepCopy()
	InjectPluginVolumePodSpec(spec, mainContainerName)

	sidecarContainerFound := false
	mainContainerFound := false
	for i := range spec.Containers {
		if spec.Containers[i].Name == mainContainerName {
			sidecar.VolumeMounts = ensureVolumeMount(sidecar.VolumeMounts, spec.Containers[i].VolumeMounts...)
			mainContainerFound = true
		}
	}

	if !mainContainerFound {
		return errors.New("main container not found")
	}

	for i := range spec.InitContainers {
		if spec.InitContainers[i].Name == sidecar.Name {
			sidecarContainerFound = true
			spec.InitContainers[i] = *sidecar
		}
	}

	if !sidecarContainerFound {
		spec.InitContainers = append(spec.InitContainers, *sidecar)
	}

	return nil
}

// ensureVolume makes sure the passed volume is present in the list of volumes.
// If the volume is already present, it is updated.
func ensureVolume(volumes []corev1.Volume, volume corev1.Volume) []corev1.Volume {
	volumeFound := false
	for i := range volumes {
		if volumes[i].Name == volume.Name {
			volumeFound = true
			volumes[i] = volume
		}
	}

	if !volumeFound {
		volumes = append(volumes, volume)
	}

	return volumes
}

// ensureVolumeMount makes sure the passed volume mounts are present in the list of volume mounts.
// If a volume mount is already present, it is updated.
func ensureVolumeMount(mounts []corev1.VolumeMount, volumeMounts ...corev1.VolumeMount) []corev1.VolumeMount {
	for _, mount := range volumeMounts {
		mountFound := false
		for i := range mounts {
			if mounts[i].Name == mount.Name {
				mountFound = true
				mounts[i] = mount
				break
			}
		}

		if !mountFound {
			mounts = append(mounts, mount)
		}
	}

	return mounts
}

// removeVolume removes a volume with a known name from a list of volumes.
func removeVolume(volumes []corev1.Volume, name string) []corev1.Volume {
	var filteredVolumes []corev1.Volume
	for _, volume := range volumes {
		if volume.Name != name {
			filteredVolumes = append(filteredVolumes, volume)
		}
	}
	return filteredVolumes
}

func removeVolumeMount(mounts []corev1.VolumeMount, name string) []corev1.VolumeMount {
	var filteredMounts []corev1.VolumeMount
	for _, mount := range mounts {
		if mount.Name != name {
			filteredMounts = append(filteredMounts, mount)
		}
	}
	return filteredMounts
}

// getCNPGJobRole gets the role associated to a CNPG job
func getCNPGJobRole(job *batchv1.Job) string {
	const jobRoleLabelSuffix = "/jobRole"
	for k, v := range job.Spec.Template.Labels {
		if strings.HasSuffix(k, jobRoleLabelSuffix) {
			return v
		}
	}

	return ""
}
