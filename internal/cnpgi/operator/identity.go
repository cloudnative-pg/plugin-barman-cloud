package operator

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/identity"
)

// IdentityImplementation is the implementation of the CNPG-i
// Identity entrypoint
type IdentityImplementation struct {
	identity.UnimplementedIdentityServer
}

// GetPluginMetadata implements Identity
func (i IdentityImplementation) GetPluginMetadata(
	_ context.Context,
	_ *identity.GetPluginMetadataRequest,
) (*identity.GetPluginMetadataResponse, error) {
	return &Data, nil
}

// GetPluginCapabilities implements identity
func (i IdentityImplementation) GetPluginCapabilities(
	_ context.Context,
	_ *identity.GetPluginCapabilitiesRequest,
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

// Probe implements Identity
func (i IdentityImplementation) Probe(
	_ context.Context,
	_ *identity.ProbeRequest,
) (*identity.ProbeResponse, error) {
	return &identity.ProbeResponse{
		Ready: true,
	}, nil
}
