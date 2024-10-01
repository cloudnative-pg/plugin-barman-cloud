package operator

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
)

// ReconcilerImplementation implements the Reconciler capability
type ReconcilerImplementation struct {
	reconciler.UnimplementedReconcilerHooksServer
}

// GetCapabilities implements the Reconciler interface
func (r ReconcilerImplementation) GetCapabilities(
	_ context.Context,
	_ *reconciler.ReconcilerHooksCapabilitiesRequest,
) (*reconciler.ReconcilerHooksCapabilitiesResult, error) {
	return &reconciler.ReconcilerHooksCapabilitiesResult{
		ReconcilerCapabilities: []*reconciler.ReconcilerHooksCapability{
			{
				Kind: reconciler.ReconcilerHooksCapability_KIND_CLUSTER,
			},
			{
				Kind: reconciler.ReconcilerHooksCapability_KIND_BACKUP,
			},
		},
	}, nil
}

// Pre implements the reconciler interface
func (r ReconcilerImplementation) Pre(
	_ context.Context,
	_ *reconciler.ReconcilerHooksRequest,
) (*reconciler.ReconcilerHooksResult, error) {
	return &reconciler.ReconcilerHooksResult{
		Behavior: reconciler.ReconcilerHooksResult_BEHAVIOR_CONTINUE,
	}, nil
}

// Post implements the reconciler interface
func (r ReconcilerImplementation) Post(
	_ context.Context,
	_ *reconciler.ReconcilerHooksRequest,
) (*reconciler.ReconcilerHooksResult, error) {
	return &reconciler.ReconcilerHooksResult{
		Behavior: reconciler.ReconcilerHooksResult_BEHAVIOR_CONTINUE,
	}, nil
}
