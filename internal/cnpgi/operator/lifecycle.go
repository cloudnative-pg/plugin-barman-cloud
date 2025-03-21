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
						Type: lifecycle.OperatorOperationType_TYPE_PATCH,
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

	certificates, err := impl.collectAdditionalCertificates(ctx, cluster.Namespace, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	return reconcileJob(ctx, cluster, request, env, certificates)
}

func reconcileJob(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	env []corev1.EnvVar,
	certificates []corev1.VolumeProjection,
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

	if getCNPGJobRole(&job) != "full-recovery" {
		contextLogger.Debug("job is not a recovery job, skipping")
		return nil, nil
	}

	mutatedJob := job.DeepCopy()

	if err := reconcilePodSpec(
		cluster,
		&mutatedJob.Spec.Template.Spec,
		"full-recovery",
		corev1.Container{
			Args: []string{"restore"},
		},
		env,
		certificates,
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

	certificates, err := impl.collectAdditionalCertificates(ctx, cluster.Namespace, pluginConfiguration)
	if err != nil {
		return nil, err
	}

	return reconcilePod(ctx, cluster, request, pluginConfiguration, env, certificates)
}

func reconcilePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	pluginConfiguration *config.PluginConfiguration,
	env []corev1.EnvVar,
	certificates []corev1.VolumeProjection,
) (*lifecycle.OperatorLifecycleResponse, error) {
	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	contextLogger := log.FromContext(ctx).WithName("plugin-barman-cloud-lifecycle").
		WithValues("podName", pod.Name)

	mutatedPod := pod.DeepCopy()

	if len(pluginConfiguration.BarmanObjectName) != 0 {
		if err := reconcilePodSpec(
			cluster,
			&mutatedPod.Spec,
			"postgres",
			corev1.Container{
				Args: []string{"instance"},
			},
			env,
			certificates,
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
	sidecarConfig corev1.Container,
	additionalEnvs []corev1.EnvVar,
	certificates []corev1.VolumeProjection,
) error {
	envs := []corev1.EnvVar{
		{
			Name:  "NAMESPACE",
			Value: cluster.Namespace,
		},
		{
			Name:  "CLUSTER_NAME",
			Value: cluster.Name,
		},
		{
			// TODO: should we really use this one?
			// should we mount an emptyDir volume just for that?
			Name:  "SPOOL_DIRECTORY",
			Value: "/controller/wal-restore-spool",
		},
		{
			Name:  "CUSTOM_CNPG_GROUP",
			Value: cluster.GetObjectKind().GroupVersionKind().Group,
		},
		{
			Name:  "CUSTOM_CNPG_VERSION",
			Value: cluster.GetObjectKind().GroupVersionKind().Version,
		},
	}

	envs = append(envs, additionalEnvs...)

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
	sidecarConfig.Name = "plugin-barman-cloud"
	sidecarConfig.Image = viper.GetString("sidecar-image")
	sidecarConfig.ImagePullPolicy = cluster.Spec.ImagePullPolicy
	sidecarConfig.StartupProbe = baseProbe.DeepCopy()

	// merge the main container envs if they aren't already set
	for _, container := range spec.Containers {
		if container.Name == mainContainerName {
			for _, env := range container.Env {
				found := false
				for _, existingEnv := range sidecarConfig.Env {
					if existingEnv.Name == env.Name {
						found = true
						break
					}
				}
				if !found {
					sidecarConfig.Env = append(sidecarConfig.Env, env)
				}
			}
			break
		}
	}

	// merge the default envs if they aren't already set
	for _, env := range envs {
		found := false
		for _, existingEnv := range sidecarConfig.Env {
			if existingEnv.Name == env.Name {
				found = true
				break
			}
		}
		if !found {
			sidecarConfig.Env = append(sidecarConfig.Env, env)
		}
	}

	if err := injectPluginSidecarPodSpec(spec, &sidecarConfig, mainContainerName); err != nil {
		return err
	}

	// inject the volume containing the certificates if needed
	if !volumeListHasVolume(spec.Volumes, barmanCertificatesVolumeName) {
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: barmanCertificatesVolumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: certificates,
				},
			},
		})
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

	spec.Volumes = append(spec.Volumes, corev1.Volume{
		Name: pluginVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	for i := range spec.Containers {
		if spec.Containers[i].Name == mainContainerName {
			spec.Containers[i].VolumeMounts = append(
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
//
// If the "injectMainContainerVolumes" flag is true, this will append all the volume
// mounts that are used in the instance manager Pod to the passed sidecar
// container, granting it superuser access to the PostgreSQL instance.
func injectPluginSidecarPodSpec(
	spec *corev1.PodSpec,
	sidecar *corev1.Container,
	mainContainerName string,
) error {
	sidecar = sidecar.DeepCopy()
	InjectPluginVolumePodSpec(spec, mainContainerName)

	var volumeMounts []corev1.VolumeMount
	sidecarContainerFound := false
	mainContainerFound := false
	for i := range spec.Containers {
		if spec.Containers[i].Name == mainContainerName {
			volumeMounts = spec.Containers[i].VolumeMounts
			mainContainerFound = true
		}
	}

	if !mainContainerFound {
		return errors.New("main container not found")
	}

	for i := range spec.InitContainers {
		if spec.InitContainers[i].Name == sidecar.Name {
			sidecarContainerFound = true
		}
	}

	if sidecarContainerFound {
		// The sidecar container was already added
		return nil
	}

	// Do not modify the passed sidecar definition
	sidecar.VolumeMounts = append(
		sidecar.VolumeMounts,
		corev1.VolumeMount{
			Name:      barmanCertificatesVolumeName,
			MountPath: metadata.BarmanCertificatesPath,
		})
	sidecar.VolumeMounts = append(sidecar.VolumeMounts, volumeMounts...)
	sidecar.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
	spec.InitContainers = append(spec.InitContainers, *sidecar)

	return nil
}

// volumeListHasVolume check if a volume with a known name exists
// in the volume list
func volumeListHasVolume(volumes []corev1.Volume, name string) bool {
	for i := range volumes {
		if volumes[i].Name == name {
			return true
		}
	}

	return false
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
