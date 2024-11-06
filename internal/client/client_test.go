package client

import (
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ExtendedClient Get", func() {
	var (
		extendedClient *ExtendedClient
		secretInClient *corev1.Secret
	)

	BeforeEach(func() {
		secretInClient = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-secret",
			},
		}
		baseClient := fake.NewClientBuilder().WithObjects(secretInClient).Build()
		extendedClient = NewExtendedClient(baseClient, 60).(*ExtendedClient)
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
