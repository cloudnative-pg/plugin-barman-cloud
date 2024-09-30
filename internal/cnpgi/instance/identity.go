package instance

import (
	"context"
	"fmt"

	"github.com/cloudnative-pg/cnpg-i/pkg/identity"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

type IdentityImplementation struct {
	identity.UnimplementedIdentityServer
	BarmanObjectKey client.ObjectKey
	Client          client.Client
}

func (i IdentityImplementation) GetPluginMetadata(
	_ context.Context,
	_ *identity.GetPluginMetadataRequest,
) (*identity.GetPluginMetadataResponse, error) {
	return &Data, nil
}

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
