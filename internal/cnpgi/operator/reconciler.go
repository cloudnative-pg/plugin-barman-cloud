package operator

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/specs"
)

// ReconcilerImplementation implements the Reconciler capability
type ReconcilerImplementation struct {
	Client client.Client
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
	ctx context.Context,
	request *reconciler.ReconcilerHooksRequest,
) (*reconciler.ReconcilerHooksResult, error) {
	contextLogger := log.FromContext(ctx)

	reconciledKind, err := object.GetKind(request.GetResourceDefinition())
	if err != nil {
		return nil, err
	}
	if reconciledKind != "Cluster" {
		return &reconciler.ReconcilerHooksResult{
			Behavior: reconciler.ReconcilerHooksResult_BEHAVIOR_CONTINUE,
		}, nil
	}

	cluster, err := decoder.DecodeClusterJSON(request.GetResourceDefinition())
	if err != nil {
		return nil, err
	}

	contextLogger = contextLogger.WithValues("name", cluster.Name, "namespace", cluster.Namespace)
	ctx = log.IntoContext(ctx, contextLogger)

	pluginConfiguration, err := config.NewFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	var barmanObject barmancloudv1.ObjectStore
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      pluginConfiguration.BarmanObjectName,
	}, &barmanObject); err != nil {
		if apierrs.IsNotFound(err) {
			contextLogger.Info("Not found barman object configuration, requeuing")
			return &reconciler.ReconcilerHooksResult{
				Behavior: reconciler.ReconcilerHooksResult_BEHAVIOR_REQUEUE,
			}, nil
		}
	}

	if err := r.ensureRole(ctx, cluster, &barmanObject); err != nil {
		return nil, err
	}

	if err := r.ensureRoleBinding(ctx, cluster); err != nil {
		return nil, err
	}

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

func (r ReconcilerImplementation) ensureRole(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	barmanObject *barmancloudv1.ObjectStore,
) error {
	contextLogger := log.FromContext(ctx)
	newRole := specs.BuildRole(cluster, barmanObject)

	var role rbacv1.Role
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: newRole.Namespace,
		Name:      newRole.Name,
	}, &role); err != nil {
		if !apierrs.IsNotFound(err) {
			return err
		}

		contextLogger.Info(
			"Creating role",
			"name", newRole.Name,
			"namespace", newRole.Namespace,
		)

		if err := ctrl.SetControllerReference(
			cluster,
			newRole,
			r.Client.Scheme(),
		); err != nil {
			return err
		}

		return r.Client.Create(ctx, newRole)
	}

	if equality.Semantic.DeepEqual(newRole.Rules, role.Rules) {
		// There's no need to hit the API server again
		return nil
	}

	contextLogger.Info(
		"Patching role",
		"name", newRole.Name,
		"namespace", newRole.Namespace,
		"rules", newRole.Rules,
	)

	return r.Client.Patch(ctx, newRole, client.MergeFrom(&role))
}

func (r ReconcilerImplementation) ensureRoleBinding(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
) error {
	var role rbacv1.RoleBinding
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      specs.GetRBACName(cluster.Name),
	}, &role); err != nil {
		if apierrs.IsNotFound(err) {
			return r.createRoleBinding(ctx, cluster)
		}
		return err
	}

	// TODO: this assumes role bindings never change.
	// Is that true? Should we relax this assumption?
	return nil
}

func (r ReconcilerImplementation) createRoleBinding(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
) error {
	roleBinding := specs.BuildRoleBinding(cluster)
	if err := ctrl.SetControllerReference(cluster, roleBinding, r.Client.Scheme()); err != nil {
		return err
	}
	return r.Client.Create(ctx, roleBinding)
}
