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

package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/cloudnative-pg/machinery/pkg/log"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/rbac"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/specs"
)

// ObjectStoreReconciler reconciles a ObjectStore object.
type ObjectStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;list;get;watch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=backups,verbs=get;list;watch
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=barmancloud.cnpg.io,resources=objectstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=barmancloud.cnpg.io,resources=objectstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=barmancloud.cnpg.io,resources=objectstores/finalizers,verbs=update

// Reconcile ensures that the RBAC Role for each Cluster referencing
// this ObjectStore is up to date with the current ObjectStore spec.
// It discovers affected Roles by listing plugin-managed Roles and
// inspecting their rules, without needing access to Cluster objects.
func (r *ObjectStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	contextLogger := log.FromContext(ctx).WithValues(
		"objectStoreName", req.Name,
		"namespace", req.Namespace,
	)
	ctx = log.IntoContext(ctx, contextLogger)

	contextLogger.Info("ObjectStore reconciliation start")

	var roleList rbacv1.RoleList
	if err := r.List(ctx, &roleList,
		client.InNamespace(req.Namespace),
		client.HasLabels{metadata.ClusterLabelName},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("while listing roles: %w", err)
	}

	var errs []error
	for i := range roleList.Items {
		role := &roleList.Items[i]

		objectStoreNames := specs.ObjectStoreNamesFromRole(role)
		if !slices.Contains(objectStoreNames, req.Name) {
			continue
		}

		contextLogger.Info("Reconciling RBAC for role",
			"roleName", role.Name)

		if err := r.reconcileRoleRules(ctx, role, objectStoreNames); err != nil {
			contextLogger.Error(err, "Failed to reconcile RBAC for role",
				"roleName", role.Name, "namespace", role.Namespace)
			errs = append(errs, fmt.Errorf("while reconciling role %s: %w", role.Name, err))
		}
	}

	contextLogger.Info("ObjectStore reconciliation completed")
	return ctrl.Result{}, errors.Join(errs...)
}

// reconcileRoleRules fetches the ObjectStores referenced by the
// Role and patches its rules to match the current specs.
func (r *ObjectStoreReconciler) reconcileRoleRules(
	ctx context.Context,
	role *rbacv1.Role,
	objectStoreNames []string,
) error {
	contextLogger := log.FromContext(ctx)
	barmanObjects := make([]barmancloudv1.ObjectStore, 0, len(objectStoreNames))

	for _, name := range objectStoreNames {
		var barmanObject barmancloudv1.ObjectStore
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: role.Namespace,
			Name:      name,
		}, &barmanObject); err != nil {
			if apierrs.IsNotFound(err) {
				contextLogger.Info("ObjectStore not found, skipping",
					"objectStoreName", name)
				continue
			}
			return fmt.Errorf("while getting ObjectStore %s: %w", name, err)
		}
		barmanObjects = append(barmanObjects, barmanObject)
	}

	return rbac.EnsureRoleRules(ctx, r.Client, client.ObjectKeyFromObject(role), barmanObjects)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObjectStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&barmancloudv1.ObjectStore{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
	if err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	return nil
}
