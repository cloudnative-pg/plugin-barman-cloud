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

// Package rbac contains utilities to reconcile RBAC resources
// for the barman-cloud plugin.
package rbac

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/specs"
)

// EnsureRole ensures the RBAC Role for the given Cluster matches
// the desired state derived from the given ObjectStores. On creation,
// the Cluster is set as the owner of the Role for garbage collection.
//
// This function is called from both the Pre hook (gRPC) and the
// ObjectStore controller. To handle concurrent modifications
// gracefully, AlreadyExists on Create and Conflict on Patch are
// retried once rather than returned as errors.
func EnsureRole(
	ctx context.Context,
	c client.Client,
	cluster *cnpgv1.Cluster,
	barmanObjects []barmancloudv1.ObjectStore,
) error {
	newRole := specs.BuildRole(cluster, barmanObjects)

	roleKey := client.ObjectKey{
		Namespace: newRole.Namespace,
		Name:      newRole.Name,
	}

	var role rbacv1.Role
	err := c.Get(ctx, roleKey, &role)

	switch {
	case apierrs.IsNotFound(err):
		role, err := createRole(ctx, c, cluster, newRole)
		if err != nil {
			return err
		}
		if role == nil {
			// Created successfully, nothing else to do.
			return nil
		}
		// AlreadyExists: fall through to patch with the re-read role.
		return patchRoleRules(ctx, c, newRole.Rules, role)

	case err != nil:
		return err

	default:
		return patchRoleRules(ctx, c, newRole.Rules, &role)
	}
}

// createRole attempts to create the Role. If another writer created
// it concurrently (AlreadyExists), it re-reads and returns the
// existing Role for the caller to patch. On success it returns nil.
func createRole(
	ctx context.Context,
	c client.Client,
	cluster *cnpgv1.Cluster,
	newRole *rbacv1.Role,
) (*rbacv1.Role, error) {
	contextLogger := log.FromContext(ctx)

	if err := controllerutil.SetControllerReference(cluster, newRole, c.Scheme()); err != nil {
		return nil, err
	}

	contextLogger.Info("Creating role",
		"name", newRole.Name, "namespace", newRole.Namespace)

	createErr := c.Create(ctx, newRole)
	if createErr == nil {
		return nil, nil
	}
	if !apierrs.IsAlreadyExists(createErr) {
		return nil, createErr
	}

	contextLogger.Info("Role was created concurrently, checking rules")

	var role rbacv1.Role
	if err := c.Get(ctx, client.ObjectKeyFromObject(newRole), &role); err != nil {
		return nil, err
	}

	return &role, nil
}

// patchRoleRules patches the Role's rules if they differ from the
// desired state. On Conflict (concurrent modification), it retries
// once with a fresh read.
func patchRoleRules(
	ctx context.Context,
	c client.Client,
	desiredRules []rbacv1.PolicyRule,
	role *rbacv1.Role,
) error {
	if equality.Semantic.DeepEqual(desiredRules, role.Rules) {
		return nil
	}

	contextLogger := log.FromContext(ctx)
	contextLogger.Info("Patching role",
		"name", role.Name, "namespace", role.Namespace, "rules", desiredRules)

	oldRole := role.DeepCopy()
	role.Rules = desiredRules

	patchErr := c.Patch(ctx, role, client.MergeFrom(oldRole))
	if patchErr == nil || !apierrs.IsConflict(patchErr) {
		return patchErr
	}

	// Conflict: re-read and retry once.
	contextLogger.Info("Role was modified concurrently, retrying patch")
	if err := c.Get(ctx, client.ObjectKeyFromObject(role), role); err != nil {
		return err
	}
	if equality.Semantic.DeepEqual(desiredRules, role.Rules) {
		return nil
	}

	oldRole = role.DeepCopy()
	role.Rules = desiredRules

	return c.Patch(ctx, role, client.MergeFrom(oldRole))
}
