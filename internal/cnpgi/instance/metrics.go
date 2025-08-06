package instance

import (
	"context"
	"fmt"
	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/cloudnative-pg/cnpg-i/pkg/metrics"
	"github.com/cloudnative-pg/machinery/pkg/log"
)

var (
	// Sanitize the plugin name to be a valid Prometheus metric namespace
	metricsDomain = strings.NewReplacer(".", "_", "-", "_").Replace(metadata.PluginName)
)

type metricsImpl struct {
	// important the client should be one with a underlying cache
	Client client.Client
	metrics.UnimplementedMetricsServer
}

func buildFqName(name string) string {
	// Build the fully qualified name for the metric
	return fmt.Sprintf("%s_%s", metricsDomain, strings.NewReplacer(".", "_", "-", "_").Replace(name))
}

var (
	firstRecoverabilityPointMetricName     = buildFqName("first_recoverability_point")
	lastAvailableBackupTimestampMetricName = buildFqName("last_available_backup_timestamp")
	testMetricName                         = buildFqName("test_metric")
)

func (m metricsImpl) GetCapabilities(
	ctx context.Context,
	_ *metrics.MetricsCapabilitiesRequest,
) (*metrics.MetricsCapabilitiesResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Trace("metrics capabilities call received")

	return &metrics.MetricsCapabilitiesResult{
		Capabilities: []*metrics.MetricsCapability{
			{
				Type: &metrics.MetricsCapability_Rpc{
					Rpc: &metrics.MetricsCapability_RPC{
						Type: metrics.MetricsCapability_RPC_TYPE_METRICS,
					},
				},
			},
		},
	}, nil
}

func (m metricsImpl) Define(
	ctx context.Context,
	_ *metrics.DefineMetricsRequest,
) (*metrics.DefineMetricsResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Trace("metrics define call received")

	return &metrics.DefineMetricsResult{
		Metrics: []*metrics.Metric{
			{
				FqName:    firstRecoverabilityPointMetricName,
				Help:      "The first point of recoverability for the cluster as a unix timestamp",
				ValueType: &metrics.MetricType{Type: metrics.MetricType_TYPE_GAUGE},
			},
			{
				FqName:    lastAvailableBackupTimestampMetricName,
				Help:      "The last available backup as a unix timestamp",
				ValueType: &metrics.MetricType{Type: metrics.MetricType_TYPE_GAUGE},
			},
		},
	}, nil
}

func (m metricsImpl) Collect(
	ctx context.Context,
	req *metrics.CollectMetricsRequest,
) (*metrics.CollectMetricsResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Trace("metrics collect call received")

	configuration, err := config.NewFromClusterJSON(req.ClusterDefinition)
	if err != nil {
		contextLogger.Error(err, "while creating configuration from cluster definition")
		return nil, fmt.Errorf("while creating configuration from cluster definition: %w", err)
	}

	var objectStore barmancloudv1.ObjectStore
	if err := m.Client.Get(ctx, configuration.GetBarmanObjectKey(), &objectStore); err != nil {
		contextLogger.Error(err, "while getting object store", "key", configuration.GetRecoveryBarmanObjectKey())
		return nil, err
	}

	x, ok := objectStore.Status.ServerRecoveryWindow[configuration.ServerName]
	if !ok {
		return &metrics.CollectMetricsResult{
			Metrics: []*metrics.CollectMetric{
				{
					FqName: firstRecoverabilityPointMetricName,
					Value:  0,
				},
				{
					FqName: lastAvailableBackupTimestampMetricName,
					Value:  0,
				},
			},
		}, nil
	}

	var firstRecoverabilityPoint float64
	var lastAvailableBackup float64
	if x.FirstRecoverabilityPoint != nil {
		firstRecoverabilityPoint = float64(x.FirstRecoverabilityPoint.Unix())
	}
	if x.LastSuccessfulBackupTime != nil {
		lastAvailableBackup = float64(x.LastSuccessfulBackupTime.Unix())
	}

	return &metrics.CollectMetricsResult{
		Metrics: []*metrics.CollectMetric{
			{
				FqName: firstRecoverabilityPointMetricName,
				Value:  firstRecoverabilityPoint,
			},
			{
				FqName: lastAvailableBackupTimestampMetricName,
				Value:  lastAvailableBackup,
			},
		},
	}, nil
}
