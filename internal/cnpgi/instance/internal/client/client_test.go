package client

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"time"

	v1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
			Spec: v1.ObjectStoreSpec{
				InstanceSidecarConfiguration: v1.InstanceSidecarConfiguration{
					CacheTTL: ptr.To(60),
				},
			},
		}

		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secretInClient, objectStore).Build()
		extendedClient = NewExtendedClient(baseClient, client.ObjectKeyFromObject(objectStore)).(*ExtendedClient)
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
		extendedClient.cachedSecrets = []*cachedSecret{
			{
				secret:        secretNotInClient,
				fetchUnixTime: time.Now().Unix(),
			},
		}

		err := extendedClient.Get(ctx, client.ObjectKeyFromObject(secretNotInClient), secretInClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(secretNotInClient).To(Equal(extendedClient.cachedSecrets[0].secret))
	})

	It("fetches secret from base client if cache is expired", func(ctx SpecContext) {
		extendedClient.cachedSecrets = []*cachedSecret{
			{
				secret:        secretInClient.DeepCopy(),
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
		Expect(extendedClient.cachedSecrets).To(BeEmpty())
	})
})
