package instance

import (
	"context"
	"fmt"

	"github.com/cloudnative-pg/cnpg-i/pkg/identity"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

// IdentityImplementation implements IdentityServer
type IdentityImplementation struct {
	identity.UnimplementedIdentityServer
	BarmanObjectKey client.ObjectKey
	Client          client.Client
}

// GetPluginMetadata implements IdentityServer
func (i IdentityImplementation) GetPluginMetadata(
	_ context.Context,
	_ *identity.GetPluginMetadataRequest,
) (*identity.GetPluginMetadataResponse, error) {
	return &metadata.Data, nil
}

// GetPluginCapabilities implements IdentityServer
func (i IdentityImplementation) GetPluginCapabilities(
	_ context.Context,
	_ *identity.GetPluginCapabilitiesRequest,
) (*identity.GetPluginCapabilitiesResponse, error) {
	return &identity.GetPluginCapabilitiesResponse{
		Capabilities: []*identity.PluginCapability{
			{
				Type: &identity.PluginCapability_Service_{
					Service: &identity.PluginCapability_Service{
						Type: identity.PluginCapability_Service_TYPE_WAL_SERVICE,
					},
				},
			},
			{
				Type: &identity.PluginCapability_Service_{
					Service: &identity.PluginCapability_Service{
						Type: identity.PluginCapability_Service_TYPE_BACKUP_SERVICE,
					},
				},
			},
		},
	}, nil
}

// Probe implements IdentityServer
func (i IdentityImplementation) Probe(
	ctx context.Context,
	_ *identity.ProbeRequest,
) (*identity.ProbeResponse, error) {
	var obj barmancloudv1.ObjectStore
	if err := i.Client.Get(ctx, i.BarmanObjectKey, &obj); err != nil {
		return nil, fmt.Errorf("while fetching object store %s: %w", i.BarmanObjectKey.Name, err)
	}

	return &identity.ProbeResponse{
		Ready: true,
	}, nil
}
