/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package specs

import (
	"fmt"
	"slices"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/stringset"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

// BuildRole builds the Role object for this cluster
func BuildRole(
	cluster *cnpgv1.Cluster,
	barmanObjects []barmancloudv1.ObjectStore,
) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetRBACName(cluster.Name),
			Labels:    BuildLabels(cluster),
		},
		Rules: BuildRoleRules(barmanObjects),
	}
}

// BuildRoleRules builds the RBAC PolicyRules for the given ObjectStores.
func BuildRoleRules(barmanObjects []barmancloudv1.ObjectStore) []rbacv1.PolicyRule {
	secretsSet := stringset.New()
	barmanObjectsSet := stringset.New()

	for _, barmanObject := range barmanObjects {
		barmanObjectsSet.Put(barmanObject.Name)
		for _, secret := range CollectSecretNamesFromCredentials(&barmanObject.Spec.Configuration.BarmanCredentials) {
			secretsSet.Put(secret)
		}
	}

	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				barmancloudv1.GroupVersion.Group,
			},
			Verbs: []string{
				"get",
				"watch",
				"list",
			},
			Resources: []string{
				"objectstores",
			},
			ResourceNames: barmanObjectsSet.ToSortedList(),
		},
		{
			APIGroups: []string{
				barmancloudv1.GroupVersion.Group,
			},
			Verbs: []string{
				"update",
			},
			Resources: []string{
				"objectstores/status",
			},
			ResourceNames: barmanObjectsSet.ToSortedList(),
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
			ResourceNames: secretsSet.ToSortedList(),
		},
	}
}

// ObjectStoreNamesFromRole extracts the ObjectStore names referenced
// by a plugin-managed Role. It finds the objectstores rule
// semantically (by APIGroup and Resource, not by index) and returns
// a copy of its ResourceNames. Returns nil if no matching rule is
// found.
func ObjectStoreNamesFromRole(role *rbacv1.Role) []string {
	for _, rule := range role.Rules {
		if len(rule.APIGroups) == 1 &&
			rule.APIGroups[0] == barmancloudv1.GroupVersion.Group &&
			len(rule.Resources) == 1 &&
			rule.Resources[0] == "objectstores" {
			return slices.Clone(rule.ResourceNames)
		}
	}

	return nil
}

// BuildRoleBinding builds the role binding object for this cluster
func BuildRoleBinding(
	cluster *cnpgv1.Cluster,
) *rbacv1.RoleBinding {
	clusterServiceAccountName := cluster.Name
	if cluster.Spec.ServiceAccountName != "" {
		clusterServiceAccountName = cluster.Spec.ServiceAccountName
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetRBACName(cluster.Name),
			Labels:    BuildLabels(cluster),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      clusterServiceAccountName,
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
