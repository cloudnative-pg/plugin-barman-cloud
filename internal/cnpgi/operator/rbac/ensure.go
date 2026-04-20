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

package rbac

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/specs"
)

// EnsureRole ensures the RBAC Role for the given Cluster matches
// the desired state derived from the given ObjectStores. On creation,
// the Cluster is set as the owner of the Role for garbage collection.
//
// This function is called from the Pre hook (gRPC). It creates the
// Role if it does not exist, then patches rules and labels to match
// the desired state.
//
// Note: the ObjectStore controller (EnsureRoleRules) can patch the
// same Role concurrently. Both paths use RetryOnConflict but compute
// desired rules from their own view of ObjectStores. If the Pre hook
// reads stale ObjectStore data from the informer cache, it may
// briefly revert a fresher update. This is self-healing: the next
// ObjectStore reconcile restores the correct state.
func EnsureRole(
	ctx context.Context,
	c client.Client,
	cluster *cnpgv1.Cluster,
	barmanObjects []barmancloudv1.ObjectStore,
) error {
	newRole := specs.BuildRole(cluster, barmanObjects)
	roleKey := client.ObjectKeyFromObject(newRole)

	if err := ensureRoleExists(ctx, c, cluster, newRole); err != nil {
		return err
	}

	return patchRole(ctx, c, roleKey, newRole.Rules, specs.GetRequiredLabels(cluster))
}

// EnsureRoleRules updates the rules of an existing Role to match
// the desired state derived from the given ObjectStores. Unlike
// EnsureRole, this function does not create Roles or set owner
// references — it only patches rules on Roles that already exist.
// It is intended for the ObjectStore controller path where no
// Cluster object is available. Returns nil if the Role does not
// exist (the Pre hook has not created it yet).
func EnsureRoleRules(
	ctx context.Context,
	c client.Client,
	roleKey client.ObjectKey,
	barmanObjects []barmancloudv1.ObjectStore,
) error {
	err := patchRole(ctx, c, roleKey, specs.BuildRoleRules(barmanObjects), nil)
	if apierrs.IsNotFound(err) {
		log.FromContext(ctx).Debug("Role not found, skipping rule update",
			"name", roleKey.Name, "namespace", roleKey.Namespace)
		return nil
	}

	return err
}

// EnsureRoleBinding ensures the RoleBinding for the given Cluster matches
// the desired state.
//
// This function is called from the Pre hook (gRPC). It creates the RoleBinding
// if it does not exist, otherwise it patches RoleRef, Subjects, and labels to match
// the desired state.
func EnsureRoleBinding(ctx context.Context, c client.Client, cluster *cnpgv1.Cluster) error {
	contextLogger := log.FromContext(ctx)

	desiredRoleBinding := specs.BuildRoleBinding(cluster)
	if err := specs.SetControllerReference(cluster, desiredRoleBinding); err != nil {
		return err
	}

	roleBinding := &rbacv1.RoleBinding{}

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      specs.GetRBACName(cluster.Name),
	}, roleBinding); err != nil {
		if apierrs.IsNotFound(err) {
			contextLogger.Info("Creating RoleBinding", "name", desiredRoleBinding.Name,
				"namespace", desiredRoleBinding.Namespace)
			return c.Create(ctx, desiredRoleBinding)
		}
	}

	if !roleBindingNeedsUpdate(roleBinding, desiredRoleBinding) {
		return nil
	}

	contextLogger.Info("Patching role binding",
		"name", roleBinding.Name, "namespace", roleBinding.Namespace)

	oldRoleBinding := roleBinding.DeepCopy()
	roleBinding.Labels = desiredRoleBinding.Labels
	roleBinding.RoleRef = desiredRoleBinding.RoleRef
	roleBinding.Subjects = desiredRoleBinding.Subjects

	return c.Patch(ctx, roleBinding, client.MergeFrom(oldRoleBinding))
}

// ensureRoleExists creates the Role if it does not exist. Returns
// nil on success and nil on AlreadyExists (another writer created
// it concurrently). The caller always follows up with patchRole.
func ensureRoleExists(
	ctx context.Context,
	c client.Client,
	cluster *cnpgv1.Cluster,
	newRole *rbacv1.Role,
) error {
	contextLogger := log.FromContext(ctx)

	var existing rbacv1.Role
	err := c.Get(ctx, client.ObjectKeyFromObject(newRole), &existing)
	if err == nil {
		return nil
	}
	if !apierrs.IsNotFound(err) {
		return err
	}

	if err := specs.SetControllerReference(cluster, newRole); err != nil {
		return err
	}

	contextLogger.Info("Creating role",
		"name", newRole.Name, "namespace", newRole.Namespace)

	createErr := c.Create(ctx, newRole)
	if createErr == nil || apierrs.IsAlreadyExists(createErr) {
		return nil
	}

	return createErr
}

// patchRole patches the Role's rules and optionally its labels to
// match the desired state. When desiredLabels is nil, labels are
// not modified. Uses retry.RetryOnConflict for concurrent
// modification handling.
func patchRole(
	ctx context.Context,
	c client.Client,
	roleKey client.ObjectKey,
	desiredRules []rbacv1.PolicyRule,
	desiredLabels map[string]string,
) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var role rbacv1.Role
		if err := c.Get(ctx, roleKey, &role); err != nil {
			return err
		}

		rulesMatch := equality.Semantic.DeepEqual(desiredRules, role.Rules)
		labelsMatch := desiredLabels == nil || !labelsNeedUpdate(role.Labels, desiredLabels)

		if rulesMatch && labelsMatch {
			return nil
		}

		contextLogger := log.FromContext(ctx)
		contextLogger.Info("Patching role",
			"name", role.Name, "namespace", role.Namespace)

		oldRole := role.DeepCopy()
		role.Rules = desiredRules

		if desiredLabels != nil {
			if role.Labels == nil {
				role.Labels = make(map[string]string, len(desiredLabels))
			}
			for k, v := range desiredLabels {
				role.Labels[k] = v
			}
		}

		return c.Patch(ctx, &role, client.MergeFrom(oldRole))
	})
}

// labelsNeedUpdate returns true if any key in desired is missing
// or has a different value in existing.
func labelsNeedUpdate(existing, desired map[string]string) bool {
	for k, v := range desired {
		if existing[k] != v {
			return true
		}
	}
	return false
}

// roleBindingNeedsUpdate returns true if the existing RoleBinding's
// RoleRef or Subjects differ from the desired, or if labels need update.
func roleBindingNeedsUpdate(existing, desired *rbacv1.RoleBinding) bool {
	if existing == nil || desired == nil {
		return existing != desired
	}

	if !equality.Semantic.DeepEqual(existing.RoleRef, desired.RoleRef) {
		return true
	}

	if !equality.Semantic.DeepEqual(existing.Subjects, desired.Subjects) {
		return true
	}

	if labelsNeedUpdate(existing.Labels, desired.Labels) {
		return true
	}

	return false
}
