package config

import (
	"strings"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

// ConfigurationError represents a mistake in the plugin configuration
type ConfigurationError struct {
	messages []string
}

// Error implements the error interface
func (e *ConfigurationError) Error() string {
	return strings.Join(e.messages, ",")
}

// NewConfigurationError creates a new empty configuration error
func NewConfigurationError() *ConfigurationError {
	return &ConfigurationError{}
}

// WithMessage adds a new error message to a potentially empty
// ConfigurationError
func (e *ConfigurationError) WithMessage(msg string) *ConfigurationError {
	if e == nil {
		return &ConfigurationError{
			messages: []string{msg},
		}
	}

	return &ConfigurationError{
		messages: append(e.messages, msg),
	}
}

// IsEmpty returns true if there's no error messages
func (e *ConfigurationError) IsEmpty() bool {
	return len(e.messages) == 0
}

// PluginConfiguration is the configuration of the plugin
type PluginConfiguration struct {
	Cluster *cnpgv1.Cluster

	BarmanObjectName string
	ServerName       string

	RecoveryBarmanObjectName string
	RecoveryServerName       string

	ReplicaSourceBarmanObjectName string
	ReplicaSourceServerName       string
}

// GetBarmanObjectKey gets the namespaced name of the barman object
func (config *PluginConfiguration) GetBarmanObjectKey() types.NamespacedName {
	return types.NamespacedName{
		Namespace: config.Cluster.Namespace,
		Name:      config.BarmanObjectName,
	}
}

// GetRecoveryBarmanObjectKey gets the namespaced name of the recovery barman object
func (config *PluginConfiguration) GetRecoveryBarmanObjectKey() types.NamespacedName {
	return types.NamespacedName{
		Namespace: config.Cluster.Namespace,
		Name:      config.RecoveryBarmanObjectName,
	}
}

// GetReplicaSourceBarmanObjectKey gets the namespaced name of the replica source barman object
func (config *PluginConfiguration) GetReplicaSourceBarmanObjectKey() types.NamespacedName {
	return types.NamespacedName{
		Namespace: config.Cluster.Namespace,
		Name:      config.ReplicaSourceServerName,
	}
}

// GetReferredBarmanObjectsKey gets the list of barman objects referred by this
// plugin configuration
func (config *PluginConfiguration) GetReferredBarmanObjectsKey() []types.NamespacedName {
	result := make([]types.NamespacedName, 0, 3)

	if len(config.BarmanObjectName) > 0 {
		result = append(result, config.GetBarmanObjectKey())
	}
	if len(config.RecoveryBarmanObjectName) > 0 {
		result = append(result, config.GetRecoveryBarmanObjectKey())
	}
	if len(config.ReplicaSourceBarmanObjectName) > 0 {
		result = append(result, config.GetReplicaSourceBarmanObjectKey())
	}

	return result
}

func getClusterGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   cnpgv1.GroupVersion.Group,
		Version: cnpgv1.GroupVersion.Version,
		Kind:    cnpgv1.ClusterKind,
	}
}

// NewFromClusterJSON decodes a JSON representation of a cluster.
func NewFromClusterJSON(clusterJSON []byte) (*PluginConfiguration, error) {
	var result cnpgv1.Cluster

	if err := decoder.DecodeObject(clusterJSON, &result, getClusterGVK()); err != nil {
		return nil, err
	}

	return NewFromCluster(&result), nil
}

// NewFromCluster extracts the configuration from the cluster
func NewFromCluster(cluster *cnpgv1.Cluster) *PluginConfiguration {
	helper := NewPlugin(
		*cluster,
		metadata.PluginName,
	)

	serverName := cluster.Name
	for _, plugin := range cluster.Spec.Plugins {
		if plugin.IsEnabled() && plugin.Name == metadata.PluginName {
			if pluginServerName, ok := plugin.Parameters["serverName"]; ok {
				serverName = pluginServerName
			}
		}
	}

	recoveryServerName := ""
	recoveryBarmanObjectName := ""
	if recoveryParameters := getRecoveryParameters(cluster); recoveryParameters != nil {
		recoveryBarmanObjectName = recoveryParameters["barmanObjectName"]
		recoveryServerName = recoveryParameters["serverName"]
		if len(recoveryServerName) == 0 {
			recoveryServerName = cluster.Name
		}
	}

	replicaSourceServerName := ""
	replicaSourceBarmanObjectName := ""
	if replicaSourceParameters := getReplicaSourceParameters(cluster); replicaSourceParameters != nil {
		replicaSourceBarmanObjectName = replicaSourceParameters["barmanObjectName"]
		replicaSourceServerName = replicaSourceParameters["serverName"]
		if len(recoveryServerName) == 0 {
			replicaSourceServerName = cluster.Name
		}
	}

	result := &PluginConfiguration{
		Cluster: cluster,
		// used for the backup/archive
		BarmanObjectName: helper.Parameters["barmanObjectName"],
		ServerName:       serverName,
		// used for restore and wal_restore during backup recovery
		RecoveryServerName:       recoveryServerName,
		RecoveryBarmanObjectName: recoveryBarmanObjectName,
		// used for wal_restore in the designed primary of a replica cluster
		ReplicaSourceServerName:       replicaSourceServerName,
		ReplicaSourceBarmanObjectName: replicaSourceBarmanObjectName,
	}

	return result
}

func getRecoveryParameters(cluster *cnpgv1.Cluster) map[string]string {
	recoveryPluginConfiguration := getRecoverySourcePlugin(cluster)
	if recoveryPluginConfiguration == nil {
		return nil
	}

	if recoveryPluginConfiguration.Name != metadata.PluginName {
		return nil
	}

	return recoveryPluginConfiguration.Parameters
}

func getReplicaSourceParameters(cluster *cnpgv1.Cluster) map[string]string {
	replicaSourcePluginConfiguration := getReplicaSourcePlugin(cluster)
	if replicaSourcePluginConfiguration == nil {
		return nil
	}

	if replicaSourcePluginConfiguration.Name != metadata.PluginName {
		return nil
	}

	return replicaSourcePluginConfiguration.Parameters
}

// getRecoverySourcePlugin returns the configuration of the plugin being
// the recovery source of the cluster. If no such plugin have been configured,
// nil is returned
func getRecoverySourcePlugin(cluster *cnpgv1.Cluster) *cnpgv1.PluginConfiguration {
	if cluster.Spec.Bootstrap == nil || cluster.Spec.Bootstrap.Recovery == nil {
		return nil
	}

	recoveryConfig := cluster.Spec.Bootstrap.Recovery
	if len(recoveryConfig.Source) == 0 {
		// Plugin-based recovery is supported only with
		// An external cluster definition
		return nil
	}

	recoveryExternalCluster, found := cluster.ExternalCluster(recoveryConfig.Source)
	if !found {
		// This error should have already been detected
		// by the validating webhook.
		return nil
	}

	return recoveryExternalCluster.PluginConfiguration
}

// getRecoverySourcePlugin returns the configuration of the plugin being
// the recovery source of the cluster. If no such plugin have been configured,
// nil is returned
func getReplicaSourcePlugin(cluster *cnpgv1.Cluster) *cnpgv1.PluginConfiguration {
	if cluster.Spec.ReplicaCluster == nil || len(cluster.Spec.ReplicaCluster.Source) == 0 {
		return nil
	}

	recoveryExternalCluster, found := cluster.ExternalCluster(cluster.Spec.ReplicaCluster.Source)
	if !found {
		// This error should have already been detected
		// by the validating webhook.
		return nil
	}

	return recoveryExternalCluster.PluginConfiguration
}

// Validate checks if the barmanObjectName is set
func (config *PluginConfiguration) Validate() error {
	err := NewConfigurationError()

	if len(config.BarmanObjectName) == 0 && len(config.RecoveryBarmanObjectName) == 0 {
		return err.WithMessage("no reference to barmanObjectName have been included")
	}

	return nil
}

// Plugin represents a plugin with its associated cluster and parameters.
type Plugin struct {
	Cluster *cnpgv1.Cluster
	// Parameters are the configuration parameters of this plugin
	Parameters  map[string]string
	PluginIndex int
}

// NewPlugin creates a new Plugin instance for the given cluster and plugin name.
func NewPlugin(cluster cnpgv1.Cluster, pluginName string) *Plugin {
	result := &Plugin{Cluster: &cluster}

	result.PluginIndex = -1
	for idx, cfg := range result.Cluster.Spec.Plugins {
		if cfg.Name == pluginName {
			result.PluginIndex = idx
			result.Parameters = cfg.Parameters
		}
	}

	return result
}
