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

package instance

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/cloudnative-pg/barman-cloud/pkg/catalog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

var _ = Describe("recovery window updates", func() {
	It("retries conflicts and preserves concurrent status fields", func(ctx context.Context) {
		scheme := runtime.NewScheme()
		barmancloudv1.AddKnownTypes(scheme)

		failedAt := metav1.NewTime(time.Now().Add(-time.Hour))
		objectStore := &barmancloudv1.ObjectStore{
			ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "default"},
			Status: barmancloudv1.ObjectStoreStatus{
				ServerRecoveryWindow: map[string]barmancloudv1.RecoveryWindow{
					"cluster": {},
				},
			},
		}

		var attempts atomic.Int32
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&barmancloudv1.ObjectStore{}).
			WithObjects(objectStore).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(
					ctx context.Context,
					c client.Client,
					subResourceName string,
					obj client.Object,
					opts ...client.SubResourceUpdateOption,
				) error {
					if subResourceName == "status" && attempts.Add(1) == 1 {
						var concurrent barmancloudv1.ObjectStore
						Expect(c.Get(ctx, client.ObjectKeyFromObject(objectStore), &concurrent)).To(Succeed())
						window := concurrent.Status.ServerRecoveryWindow["cluster"]
						window.LastFailedBackupTime = &failedAt
						concurrent.Status.ServerRecoveryWindow["cluster"] = window
						Expect(c.Status().Update(ctx, &concurrent)).To(Succeed())
						return apierrors.NewConflict(
							schema.GroupResource{Group: "barmancloud.cnpg.io", Resource: "objectstores"},
							objectStore.Name,
							errors.New("simulated concurrent update"),
						)
					}
					return c.Status().Update(ctx, obj, opts...)
				},
			}).
			Build()

		Expect(updateRecoveryWindow(ctx, fakeClient, &catalog.Catalog{}, objectStore, "cluster")).To(Succeed())
		Expect(attempts.Load()).To(Equal(int32(2)))

		var updated barmancloudv1.ObjectStore
		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(objectStore), &updated)).To(Succeed())
		Expect(updated.Status.ServerRecoveryWindow["cluster"].LastFailedBackupTime.Unix()).To(Equal(failedAt.Unix()))
	})
})
