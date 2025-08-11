package instance

import (
	"context"
	"encoding/json"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"k8s.io/utils/ptr"
	"time"

	"github.com/cloudnative-pg/cnpg-i/pkg/metrics"
	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Metrics Collect method", func() {
	const (
		clusterName     = "test-cluster"
		namespace       = "test-ns"
		objectStoreName = "test-object-store"
	)

	var (
		fakeClient  client.Client
		m           metricsImpl
		ctx         context.Context
		req         *metrics.CollectMetricsRequest
		objectStore *barmancloudv1.ObjectStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(barmancloudv1.AddToScheme(scheme)).To(Succeed())

		// Timestamps for the test
		firstRecoverabilityPoint := metav1.NewTime(time.Now().Add(-24 * time.Hour))
		lastSuccessfulBackupTime := metav1.NewTime(time.Now())

		// Create a fake ObjectStore with a status
		objectStore = &barmancloudv1.ObjectStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      objectStoreName,
				Namespace: namespace,
			},
			Status: barmancloudv1.ObjectStoreStatus{
				ServerRecoveryWindow: map[string]barmancloudv1.RecoveryWindow{
					clusterName: {
						FirstRecoverabilityPoint: &firstRecoverabilityPoint,
						LastSuccessfulBackupTime: &lastSuccessfulBackupTime,
						LastSuccussfulBackupTime: &lastSuccessfulBackupTime,
					},
				},
			},
		}

		// Create a fake client with the ObjectStore
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&barmancloudv1.ObjectStore{}).
			WithObjects(objectStore).
			Build()

		m = metricsImpl{Client: fakeClient}

		// Create a minimal cluster definition
		clusterDefinition := cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
			Spec: cnpgv1.ClusterSpec{
				Plugins: []cnpgv1.PluginConfiguration{
					{
						Name:    metadata.PluginName,
						Enabled: ptr.To(true),
						Parameters: map[string]string{
							"serverName":       clusterName,
							"barmanObjectName": objectStoreName,
						},
					},
				},
			},
		}
		clusterJSON, err := json.Marshal(clusterDefinition)
		Expect(err).ToNot(HaveOccurred())

		req = &metrics.CollectMetricsRequest{
			ClusterDefinition: clusterJSON,
		}
	})

	It("should collect metrics successfully", func() {
		res, err := m.Collect(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())
		Expect(res.Metrics).To(HaveLen(2))

		// Verify the metrics
		metricsMap := make(map[string]float64)
		for _, metric := range res.Metrics {
			metricsMap[metric.FqName] = metric.Value
		}

		// Check timestamp metrics
		expectedFirstPoint, _ := metricsMap[firstRecoverabilityPointMetricName]
		Expect(expectedFirstPoint).To(BeNumerically("~", float64(objectStore.Status.ServerRecoveryWindow[clusterName].FirstRecoverabilityPoint.Unix()), 1))

		expectedLastBackup, _ := metricsMap[lastAvailableBackupTimestampMetricName]
		Expect(expectedLastBackup).To(BeNumerically("~", float64(objectStore.Status.ServerRecoveryWindow[clusterName].LastSuccessfulBackupTime.Unix()), 1))
	})

	It("should return an error if the object store is not found", func() {
		// Use a client without any objects
		m.Client = fake.NewClientBuilder().Build()
		_, err := m.Collect(ctx, req)
		Expect(err).To(HaveOccurred())
	})
})
