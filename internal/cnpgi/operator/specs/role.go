package specs

import (
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

// BuildRole builds the Role object for this cluster
func BuildRole(
	cluster *cnpgv1.Cluster,
	barmanObject *barmancloudv1.ObjectStore,
) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetRBACName(cluster.Name),
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
					barmanObject.Name,
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
				Verbs: []string{
					"get",
					"watch",
					"list",
				},
				ResourceNames: collectSecretNames(barmanObject),
			},
		},
	}
}

// BuildRoleBinding builds the role binding object for this cluster
func BuildRoleBinding(
	cluster *cnpgv1.Cluster,
) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetRBACName(cluster.Name),
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
			Name:     GetRBACName(cluster.Name),
		},
	}
}

// GetRBACName returns the name of the RBAC entities for the
// barman cloud plugin
func GetRBACName(clusterName string) string {
	return fmt.Sprintf("%s-barman-cloud", clusterName)
}
