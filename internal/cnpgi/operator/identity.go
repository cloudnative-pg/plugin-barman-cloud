package operator

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/identity"
)

type IdentityImplementation struct {
	identity.UnimplementedIdentityServer
}

func (i IdentityImplementation) GetPluginMetadata(
	ctx context.Context,
	request *identity.GetPluginMetadataRequest,
) (*identity.GetPluginMetadataResponse, error) {
	return &Data, nil
}

func (i IdentityImplementation) GetPluginCapabilities(
	ctx context.Context,
	request *identity.GetPluginCapabilitiesRequest,
) (*identity.GetPluginCapabilitiesResponse, error) {
	return &identity.GetPluginCapabilitiesResponse{
		Capabilities: []*identity.PluginCapability{
			{
				Type: &identity.PluginCapability_Service_{
					Service: &identity.PluginCapability_Service{
						Type: identity.PluginCapability_Service_TYPE_RECONCILER_HOOKS,
					},
				},
			},
			{
				Type: &identity.PluginCapability_Service_{
					Service: &identity.PluginCapability_Service{
						Type: identity.PluginCapability_Service_TYPE_LIFECYCLE_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (i IdentityImplementation) Probe(
	ctx context.Context,
	request *identity.ProbeRequest,
) (*identity.ProbeResponse, error) {
	return &identity.ProbeResponse{
		Ready: true,
	}, nil
}
