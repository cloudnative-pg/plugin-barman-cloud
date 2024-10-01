package operator

import (
	"context"
	"errors"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/common"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
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
		},
	}, nil
}

// LifecycleHook is called when creating Kubernetes services
func (impl LifecycleImplementation) LifecycleHook(
	ctx context.Context,
	request *lifecycle.OperatorLifecycleRequest,
) (*lifecycle.OperatorLifecycleResponse, error) {
	contextLogger := log.FromContext(ctx).WithName("plugin-barman-cloud-lifecycle")

	operation := request.GetOperationType().GetType().Enum()
	if operation == nil {
		return nil, errors.New("no operation set")
	}

	cluster, err := decoder.DecodeClusterJSON(request.GetClusterDefinition())
	if err != nil {
		return nil, err
	}

	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	helper := common.NewPlugin(
		*cluster,
		metadata.PluginName,
	)

	// TODO: Validation of the plugin configuration

	mutatedPod := pod.DeepCopy()
	err = object.InjectPluginSidecar(mutatedPod, &corev1.Container{
		Name:  "plugin-barman-cloud",
		Image: viper.GetString("sidecar-image"),
		Env: []corev1.EnvVar{
			{
				Name:  "BARMAN_OBJECT_NAME",
				Value: helper.Parameters["barmanObjectStore"],
			},
			{
				// TODO: should we really use this one?
				// should we mount an emptyDir volume just for that?
				Name:  "SPOOL_DIRECTORY",
				Value: "/controller/wal-restore-spool",
			},
		},
	}, true)
	if err != nil {
		return nil, err
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
