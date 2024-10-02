package operator

import (
	"context"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilerImplementation implements the Reconciler capability
type ReconcilerImplementation struct {
	Client client.Client
	reconciler.UnimplementedReconcilerHooksServer
}

// GetCapabilities implements the Reconciler interface
func (r ReconcilerImplementation) GetCapabilities(
	ctx context.Context,
	request *reconciler.ReconcilerHooksCapabilitiesRequest,
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

	pluginConfiguration, err := config.NewFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	if err := r.ensureRole(ctx, cluster, pluginConfiguration.BarmanObjectName); err != nil {
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
	ctx context.Context,
	request *reconciler.ReconcilerHooksRequest,
) (*reconciler.ReconcilerHooksResult, error) {
	return &reconciler.ReconcilerHooksResult{
		Behavior: reconciler.ReconcilerHooksResult_BEHAVIOR_CONTINUE,
	}, nil
}

func (r ReconcilerImplementation) ensureRole(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	barmanObjectName string,
) error {
	var role rbacv1.Role
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      getRBACName(cluster.Name),
	}, &role); err != nil {
		if apierrs.IsNotFound(err) {
			return r.createRole(ctx, cluster, barmanObjectName)
		}
		return err
	}

	// TODO: patch existing role
	return nil
}

func (r ReconcilerImplementation) ensureRoleBinding(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
) error {
	var role rbacv1.RoleBinding
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      getRBACName(cluster.Name),
	}, &role); err != nil {
		if apierrs.IsNotFound(err) {
			return r.createRoleBinding(ctx, cluster)
		}
		return err
	}

	// TODO: patch existing role binding
	return nil
}

func (r ReconcilerImplementation) createRole(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	barmanObjectName string,
) error {
	role := buildRole(cluster, barmanObjectName)
	if err := ctrl.SetControllerReference(cluster, role, r.Client.Scheme()); err != nil {
		return err
	}
	return r.Client.Create(ctx, role)
}

func (r ReconcilerImplementation) createRoleBinding(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
) error {
	roleBinding := buildRoleBinding(cluster)
	if err := ctrl.SetControllerReference(cluster, roleBinding, r.Client.Scheme()); err != nil {
		return err
	}
	return r.Client.Create(ctx, roleBinding)
}

func buildRole(
	cluster *cnpgv1.Cluster,
	barmanObjectName string,
) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      getRBACName(cluster.Name),
		},

		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"barmancloud.cnpg.io",
				},
				Verbs: []string{
					"get",
					"watch",
					"list",
				},
				Resources: []string{
					"objectstores",
				},
				ResourceNames: []string{
					barmanObjectName,
				},
			},
		},
	}
}

func buildRoleBinding(
	cluster *cnpgv1.Cluster,
) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      getRBACName(cluster.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     getRBACName(cluster.Name),
		},
	}
}

// getRBACName returns the name of the RBAC entities for the
// barman cloud plugin
func getRBACName(clusterName string) string {
	return fmt.Sprintf("%s-barman", clusterName)
}
