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
	"fmt"

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

	return patchRole(ctx, c, roleKey, newRole.Rules, specs.BuildLabels(cluster))
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

// EnsureRoleBinding ensures the RoleBinding for the given Cluster
// is present and carries the recommended labels.
//
// This function is called from the Pre hook (gRPC). It creates the
// RoleBinding if it does not exist, then reconciles labels and
// Subjects:
//   - Labels are written per-key. Keys the plugin manages overwrite
//     existing values; unrelated keys (anything outside the desired
//     set) are left alone.
//   - Subjects are additive. The plugin guarantees its own Subject
//     is bound, but never removes Subjects added by other actors —
//     a Subject is a grant of access, and silently revoking access
//     someone else granted is the wrong default.
//
// RoleRef is immutable in Kubernetes. If the existing RoleBinding
// points to a different Role, the plugin fails loudly so the
// operator notices and recreates the object.
func EnsureRoleBinding(ctx context.Context, c client.Client, cluster *cnpgv1.Cluster) error {
	desiredRoleBinding := specs.BuildRoleBinding(cluster)
	if err := specs.SetControllerReference(cluster, desiredRoleBinding); err != nil {
		return err
	}

	roleBinding, err := getOrCreateRoleBinding(ctx, c, desiredRoleBinding)
	if err != nil || roleBinding == nil {
		// Either an error, or we just created the object with the
		// desired state — nothing to patch.
		return err
	}

	return reconcileRoleBinding(ctx, c, roleBinding, desiredRoleBinding)
}

// getOrCreateRoleBinding returns the existing RoleBinding when it
// is already present on the API server, or nil after a successful
// Create when the just-created object already carries the desired
// state (so the caller can skip the patch path).
//
// On a stale-informer-cache race during plugin pod startup, where
// Get returns NotFound but Create returns AlreadyExists, the
// function re-Gets to return the racing winner — the caller then
// falls through to reconciliation against that real object.
func getOrCreateRoleBinding(
	ctx context.Context,
	c client.Client,
	desired *rbacv1.RoleBinding,
) (*rbacv1.RoleBinding, error) {
	contextLogger := log.FromContext(ctx)

	rb := &rbacv1.RoleBinding{}
	err := c.Get(ctx, client.ObjectKeyFromObject(desired), rb)
	if err == nil {
		return rb, nil
	}
	if !apierrs.IsNotFound(err) {
		return nil, err
	}

	createErr := c.Create(ctx, desired)
	switch {
	case createErr == nil:
		contextLogger.Info("Created RoleBinding",
			"name", desired.Name, "namespace", desired.Namespace)
		// Just-created with the desired state — caller skips patch.
		return nil, nil
	case apierrs.IsAlreadyExists(createErr):
		contextLogger.Debug(
			"RoleBinding already exists, likely a stale informer cache; re-fetching",
			"name", desired.Name, "namespace", desired.Namespace)
		// Re-Get to return the racing winner so the caller can
		// reconcile against the real existing object.
		fetched := &rbacv1.RoleBinding{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(desired), fetched); err != nil {
			return nil, err
		}
		return fetched, nil
	default:
		return nil, createErr
	}
}

// reconcileRoleBinding brings an existing RoleBinding's labels and
// Subjects into alignment with desired. It re-Gets the object on
// conflict-retry so each attempt observes fresh server state, and
// uses optimistic locking so a competing writer's patch is rejected
// with 409 instead of silently last-write-winning.
//
// On the first attempt the function uses the existing object passed
// in by the caller, avoiding a second Get on the steady-state path.
func reconcileRoleBinding(
	ctx context.Context,
	c client.Client,
	existing, desired *rbacv1.RoleBinding,
) error {
	contextLogger := log.FromContext(ctx)
	first := true
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var roleBinding *rbacv1.RoleBinding
		if first {
			roleBinding = existing
			first = false
		} else {
			roleBinding = &rbacv1.RoleBinding{}
			if err := c.Get(ctx, client.ObjectKeyFromObject(desired), roleBinding); err != nil {
				return err
			}
		}

		// RoleRef is immutable in Kubernetes; we cannot patch it.
		// Divergence at the canonical name is corruption regardless
		// of who wrote the existing object — fail loudly so the
		// operator notices and deletes the RoleBinding, and the
		// next Pre call recreates it correctly.
		if !equality.Semantic.DeepEqual(roleBinding.RoleRef, desired.RoleRef) {
			return fmt.Errorf(
				"RoleBinding %s/%s has divergent immutable RoleRef "+
					"(existing=%+v, desired=%+v); delete the RoleBinding to allow recreation",
				roleBinding.Namespace, roleBinding.Name,
				roleBinding.RoleRef, desired.RoleRef)
		}

		if !roleBindingNeedsUpdate(roleBinding, desired) {
			return nil
		}

		contextLogger.Info("Patching role binding",
			"name", roleBinding.Name, "namespace", roleBinding.Namespace)

		oldRoleBinding := roleBinding.DeepCopy()
		roleBinding.Labels = mergeLabels(roleBinding.Labels, desired.Labels)
		roleBinding.Subjects = mergeSubjects(roleBinding.Subjects, desired.Subjects)

		return c.Patch(ctx, roleBinding,
			client.MergeFromWithOptions(oldRoleBinding, client.MergeFromWithOptimisticLock{}))
	})
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
		role.Labels = mergeLabels(role.Labels, desiredLabels)

		return c.Patch(ctx, &role,
			client.MergeFromWithOptions(oldRole, client.MergeFromWithOptimisticLock{}))
	})
}

// mergeLabels writes the desired labels onto existing per-key.
// Keys in desired overwrite the existing value; keys not in desired
// (any unrelated label a user may have set) are left alone.
func mergeLabels(existing, desired map[string]string) map[string]string {
	if len(desired) == 0 {
		return existing
	}
	if existing == nil {
		existing = make(map[string]string, len(desired))
	}
	for k, v := range desired {
		existing[k] = v
	}
	return existing
}

// labelsNeedUpdate returns true if a Patch is required to bring
// existing labels into the state mergeLabels would produce, i.e.
// any desired key is missing or has a different value in existing.
func labelsNeedUpdate(existing, desired map[string]string) bool {
	for k, v := range desired {
		if existing[k] != v {
			return true
		}
	}
	return false
}

// containsSubject reports whether subjects contains an element that
// is semantically equal to subject.
func containsSubject(subjects []rbacv1.Subject, subject rbacv1.Subject) bool {
	for _, s := range subjects {
		if equality.Semantic.DeepEqual(s, subject) {
			return true
		}
	}
	return false
}

// mergeSubjects appends desired Subjects that are not already
// present in existing.
//
// This is intentionally asymmetric to mergeLabels: labels are
// metadata, so replacing stale plugin-set values is safe. A
// Subject is a grant of access, so removing a Subject silently
// revokes permissions an external operator chose to grant. The
// plugin only requires that ITS Subject is present, not that it
// is exclusive.
func mergeSubjects(existing, desired []rbacv1.Subject) []rbacv1.Subject {
	for _, d := range desired {
		if !containsSubject(existing, d) {
			existing = append(existing, d)
		}
	}
	return existing
}

// roleBindingNeedsUpdate returns true if a Patch is required to
// bring existing into alignment with desired — any desired Subject
// missing (see mergeSubjects), or any desired label key missing
// or holding a stale value (see mergeLabels).
func roleBindingNeedsUpdate(existing, desired *rbacv1.RoleBinding) bool {
	for _, s := range desired.Subjects {
		if !containsSubject(existing.Subjects, s) {
			return true
		}
	}

	if labelsNeedUpdate(existing.Labels, desired.Labels) {
		return true
	}

	return false
}
