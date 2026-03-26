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
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/rbac"
)

// ObjectStoreReconciler reconciles a ObjectStore object.
type ObjectStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;list;get;watch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=backups,verbs=get;list;watch
// +kubebuilder:rbac:groups=barmancloud.cnpg.io,resources=objectstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=barmancloud.cnpg.io,resources=objectstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=barmancloud.cnpg.io,resources=objectstores/finalizers,verbs=update

// Reconcile ensures that the RBAC Role for each Cluster referencing
// this ObjectStore is up to date with the current ObjectStore spec.
func (r *ObjectStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	contextLogger := log.FromContext(ctx).WithValues(
		"objectStoreName", req.Name,
		"namespace", req.Namespace,
	)
	ctx = log.IntoContext(ctx, contextLogger)

	contextLogger.Info("ObjectStore reconciliation start")

	// List all Clusters in the same namespace
	var clusterList cnpgv1.ClusterList
	if err := r.List(ctx, &clusterList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("while listing clusters: %w", err)
	}

	// For each Cluster that references this ObjectStore, reconcile the Role
	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		pluginConfiguration := config.NewFromCluster(cluster)
		referredObjects := pluginConfiguration.GetReferredBarmanObjectsKey()

		if !referencesObjectStore(referredObjects, req.NamespacedName) {
			continue
		}

		contextLogger.Info("Reconciling RBAC for cluster",
			"clusterName", cluster.Name)

		if err := r.reconcileRBACForCluster(ctx, cluster, referredObjects); err != nil {
			return ctrl.Result{}, fmt.Errorf("while reconciling RBAC for cluster %s: %w", cluster.Name, err)
		}
	}

	contextLogger.Info("ObjectStore reconciliation completed")
	return ctrl.Result{}, nil
}

// reconcileRBACForCluster ensures the Role for the given Cluster is
// up to date with the current ObjectStore specs.
func (r *ObjectStoreReconciler) reconcileRBACForCluster(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	referredObjectKeys []client.ObjectKey,
) error {
	contextLogger := log.FromContext(ctx)
	barmanObjects := make([]barmancloudv1.ObjectStore, 0, len(referredObjectKeys))
	for _, key := range referredObjectKeys {
		var barmanObject barmancloudv1.ObjectStore
		if err := r.Get(ctx, key, &barmanObject); err != nil {
			if apierrs.IsNotFound(err) {
				contextLogger.Info("ObjectStore not found, skipping",
					"objectStoreName", key.Name)
				continue
			}
			return fmt.Errorf("while getting ObjectStore %s: %w", key, err)
		}
		barmanObjects = append(barmanObjects, barmanObject)
	}

	return rbac.EnsureRole(ctx, r.Client, cluster, barmanObjects)
}

// referencesObjectStore checks if the given ObjectStore is in the list
// of referred barman objects.
func referencesObjectStore(
	referredObjects []client.ObjectKey,
	objectStore client.ObjectKey,
) bool {
	for _, ref := range referredObjects {
		if ref.Name == objectStore.Name && ref.Namespace == objectStore.Namespace {
			return true
		}
	}
	return false
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
