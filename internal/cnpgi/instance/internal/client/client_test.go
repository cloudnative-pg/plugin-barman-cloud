package client

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var scheme = buildScheme()

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	return scheme
}

var _ = Describe("ExtendedClient Get", func() {
	var (
		extendedClient *ExtendedClient
		secretInClient *corev1.Secret
		objectStore    *v1.ObjectStore
	)

	BeforeEach(func() {
		secretInClient = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-secret",
			},
		}
		objectStore = &v1.ObjectStore{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-object-store",
			},
			Spec: v1.ObjectStoreSpec{},
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

		// manually add the secret to the cache, this is not present in the fake client so we are sure it is from the
		// cache
		extendedClient.cachedObjects = []cachedEntry{
			{
				entry:         secretNotInClient,
				fetchUnixTime: time.Now().Unix(),
			},
		}

		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretNotInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(secretNotInClient).To(Equal(extendedClient.cachedObjects[0].entry))
	})

	It("fetches secret from base client if cache is expired", func(ctx SpecContext) {
		extendedClient.cachedObjects = []cachedEntry{
			{
				entry:         secretInClient.DeepCopy(),
				fetchUnixTime: time.Now().Add(-2 * time.Minute).Unix(),
			},
		}

		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())
	})

	It("fetches secret from base client if not in cache", func(ctx SpecContext) {
		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())
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
})
