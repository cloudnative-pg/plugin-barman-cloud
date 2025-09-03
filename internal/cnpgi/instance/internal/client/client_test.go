package client

import (
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
		extendedClient = NewExtendedClient(baseClient).(*ExtendedClient)
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
		})
})
