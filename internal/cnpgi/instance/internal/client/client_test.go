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

package client

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var scheme = buildScheme()

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = barmancloudv1.AddToScheme(scheme)

	return scheme
}

func addToCache(c *ExtendedClient, obj client.Object, fetchUnixTime int64) {
	ce := cachedEntry{
		entry:         obj.DeepCopyObject().(client.Object),
		fetchUnixTime: fetchUnixTime,
		ttlSeconds:    DefaultTTLSeconds,
	}
	ce.entry.SetResourceVersion("from cache")
	c.cachedObjects = append(c.cachedObjects, ce)
}

var _ = Describe("ExtendedClient Get", func() {
	var (
		extendedClient *ExtendedClient
		secretInClient *corev1.Secret
		objectStore    *barmancloudv1.ObjectStore
		cancelCtx      context.CancelFunc
	)

	BeforeEach(func() {
		secretInClient = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-secret",
			},
		}
		objectStore = &barmancloudv1.ObjectStore{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-object-store",
			},
			Spec: barmancloudv1.ObjectStoreSpec{},
		}

		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secretInClient, objectStore).Build()
		ctx, cancel := context.WithCancel(context.Background())
		cancelCtx = cancel
		extendedClient = NewExtendedClient(ctx, baseClient).(*ExtendedClient)
	})

	AfterEach(func() {
		// Cancel the context to stop the cleanup routine
		cancelCtx()
	})

	It("returns secret from cache if not expired", func(ctx SpecContext) {
		secretNotInClient := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-secret-not-in-client",
			},
		}

		// manually add the secret to the cache, this is not present in the fake client,
		// so we are sure it is from the cache
		addToCache(extendedClient, secretNotInClient, time.Now().Unix())

		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretNotInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(secretInClient).To(Equal(extendedClient.cachedObjects[0].entry))
		Expect(secretInClient.GetResourceVersion()).To(Equal("from cache"))
	})

	It("fetches secret from base client if cache is expired", func(ctx SpecContext) {
		addToCache(extendedClient, secretInClient, time.Now().Add(-2*time.Minute).Unix())

		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(secretInClient.GetResourceVersion()).NotTo(Equal("from cache"))

		// the cache is updated with the new value
		Expect(extendedClient.cachedObjects).To(HaveLen(1))
		Expect(extendedClient.cachedObjects[0].entry.GetResourceVersion()).NotTo(Equal("from cache"))
	})

	It("fetches secret from base client if not in cache", func(ctx SpecContext) {
		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())

		// the cache is updated with the new value
		Expect(extendedClient.cachedObjects).To(HaveLen(1))
	})

	It("does not cache non-secret objects", func(ctx SpecContext) {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-configmap",
			},
		}
		err := extendedClient.Create(ctx, configMap)
		Expect(err).ToNot(HaveOccurred())

		err = extendedClient.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)
		Expect(err).NotTo(HaveOccurred())
		Expect(extendedClient.cachedObjects).To(BeEmpty())
	})

	It("returns the correct object from cache when multiple objects with the same object key are cached",
		func(ctx SpecContext) {
			secretNotInClient := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "common-name",
				},
			}
			objectStoreNotInClient := &barmancloudv1.ObjectStore{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "common-name",
				},
			}

			// manually add the objects to the cache, these are not present in the fake client,
			// so we are sure they are from the cache
			addToCache(extendedClient, secretNotInClient, time.Now().Unix())
			addToCache(extendedClient, objectStoreNotInClient, time.Now().Unix())

			err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretNotInClient), secretInClient)
			Expect(err).NotTo(HaveOccurred())
			err = extendedClient.Get(ctx, client.ObjectKeyFromObject(objectStoreNotInClient), objectStore)
			Expect(err).NotTo(HaveOccurred())

			Expect(secretInClient.GetResourceVersion()).To(Equal("from cache"))
			Expect(objectStore.GetResourceVersion()).To(Equal("from cache"))
		})
})

var _ = Describe("ExtendedClient Cache Cleanup", func() {
	var (
		extendedClient *ExtendedClient
		cancelCtx      context.CancelFunc
	)

	BeforeEach(func() {
		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		ctx, cancel := context.WithCancel(context.Background())
		cancelCtx = cancel
		extendedClient = NewExtendedClient(ctx, baseClient).(*ExtendedClient)
	})

	AfterEach(func() {
		cancelCtx()
	})

	It("cleans up expired entries", func(ctx SpecContext) {
		// Add some expired entries
		expiredSecret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "expired-secret-1",
			},
		}
		expiredSecret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "expired-secret-2",
			},
		}
		validSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "valid-secret",
			},
		}

		// Add expired entries (2 minutes ago)
		addToCache(extendedClient, expiredSecret1, time.Now().Add(-2*time.Minute).Unix())
		addToCache(extendedClient, expiredSecret2, time.Now().Add(-2*time.Minute).Unix())
		// Add valid entry (just now)
		addToCache(extendedClient, validSecret, time.Now().Unix())

		Expect(extendedClient.cachedObjects).To(HaveLen(3))

		// Trigger cleanup
		extendedClient.cleanupExpiredEntries(ctx)

		// Only the valid entry should remain
		Expect(extendedClient.cachedObjects).To(HaveLen(1))
		Expect(extendedClient.cachedObjects[0].entry.GetName()).To(Equal("valid-secret"))
	})

	It("does nothing when all entries are valid", func(ctx SpecContext) {
		validSecret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "valid-secret-1",
			},
		}
		validSecret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "valid-secret-2",
			},
		}

		addToCache(extendedClient, validSecret1, time.Now().Unix())
		addToCache(extendedClient, validSecret2, time.Now().Unix())

		Expect(extendedClient.cachedObjects).To(HaveLen(2))

		// Trigger cleanup
		extendedClient.cleanupExpiredEntries(ctx)

		// Both entries should remain
		Expect(extendedClient.cachedObjects).To(HaveLen(2))
	})

	It("does nothing when cache is empty", func(ctx SpecContext) {
		Expect(extendedClient.cachedObjects).To(BeEmpty())

		// Trigger cleanup
		extendedClient.cleanupExpiredEntries(ctx)

		Expect(extendedClient.cachedObjects).To(BeEmpty())
	})

	It("removes all entries when all are expired", func(ctx SpecContext) {
		expiredSecret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "expired-secret-1",
			},
		}
		expiredSecret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "expired-secret-2",
			},
		}

		addToCache(extendedClient, expiredSecret1, time.Now().Add(-2*time.Minute).Unix())
		addToCache(extendedClient, expiredSecret2, time.Now().Add(-2*time.Minute).Unix())

		Expect(extendedClient.cachedObjects).To(HaveLen(2))

		// Trigger cleanup
		extendedClient.cleanupExpiredEntries(ctx)

		Expect(extendedClient.cachedObjects).To(BeEmpty())
	})

	It("stops cleanup routine when context is cancelled", func() {
		// Create a new client with a short cleanup interval for testing
		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		ctx, cancel := context.WithCancel(context.Background())
		ec := NewExtendedClient(ctx, baseClient).(*ExtendedClient)
		ec.cleanupInterval = 10 * time.Millisecond

		// Cancel the context immediately
		cancel()

		// Give the goroutine time to stop
		time.Sleep(50 * time.Millisecond)

		// The goroutine should have stopped gracefully (no panic or hanging)
		// This test mainly verifies the cleanup routine respects context cancellation
	})
})
