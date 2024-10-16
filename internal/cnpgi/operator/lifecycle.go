package operator

import (
	"context"
	"errors"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// LifecycleImplementation is the implementation of the lifecycle handler
type LifecycleImplementation struct {
	lifecycle.UnimplementedOperatorLifecycleServer
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
	operation := request.GetOperationType().GetType().Enum()
	if operation == nil {
		return nil, errors.New("no operation set")
	}

	kind, err := object.GetKind(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	cluster, err := decoder.DecodeClusterJSON(request.GetClusterDefinition())
	if err != nil {
		return nil, err
	}

	pluginConfiguration := config.NewFromCluster(cluster)

	switch kind {
	case "Pod":
		return reconcilePod(ctx, cluster, request, pluginConfiguration)
	case "Job":
		return reconcileJob(ctx, cluster, request, pluginConfiguration)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func reconcileJob(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	pluginConfiguration *config.PluginConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {
	var job batchv1.Job
	if err := decoder.DecodeObject(
		request.GetObjectDefinition(),
		&job,
		batchv1.SchemeGroupVersion.WithKind("Job"),
	); err != nil {
		return nil, err
	}

	contextLogger := log.FromContext(ctx).WithName("plugin-barman-cloud-lifecycle").
		WithValues("jobName", job.Name)

	if job.Spec.Template.Labels[utils.JobRoleLabelName] != "full-recovery" {
		contextLogger.Debug("job is not a recovery job, skipping")
		return nil, nil
	}

	if err := pluginConfiguration.ValidateBackupObjectName(); err != nil {
		return nil, err
	}

	mutatedJob := job.DeepCopy()

	if err := reconcilePodSpec(
		pluginConfiguration,
		cluster,
		&job.Spec.Template.Spec,
		"full-recovery",
		corev1.Container{
			Args: []string{"restore"},
		},
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

func reconcilePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	pluginConfiguration *config.PluginConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {
	if err := pluginConfiguration.ValidateBarmanObjectName(); err != nil {
		return nil, err
	}

	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	contextLogger := log.FromContext(ctx).WithName("plugin-barman-cloud-lifecycle").
		WithValues("podName", pod.Name)

	mutatedPod := pod.DeepCopy()

	if err := reconcilePodSpec(pluginConfiguration, cluster, &mutatedPod.Spec, "postgres", corev1.Container{
		Args: []string{"instance"},
	}); err != nil {
		return nil, fmt.Errorf("while reconciling pod spec for pod: %w", err)
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
	cfg *config.PluginConfiguration,
	cluster *cnpgv1.Cluster,
	spec *corev1.PodSpec,
	mainContainerName string,
	sidecarConfig corev1.Container,
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
			Name:  "BARMAN_OBJECT_NAME",
			Value: cfg.BarmanObjectName,
		},
		{
			Name:  "BACKUP_OBJECT_NAME",
			Value: cfg.BackupObjectName,
		},
		{
			// TODO: should we really use this one?
			// should we mount an emptyDir volume just for that?
			Name:  "SPOOL_DIRECTORY",
			Value: "/controller/wal-restore-spool",
		},
	}

	// fixed values
	sidecarConfig.Name = "plugin-barman-cloud"
	sidecarConfig.Image = viper.GetString("sidecar-image")
	sidecarConfig.ImagePullPolicy = cluster.Spec.ImagePullPolicy

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

	if err := object.InjectPluginSidecarPodSpec(spec, &sidecarConfig, mainContainerName, true); err != nil {
		return err
	}

	return nil
}
