package config

import (
	"strings"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"

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
	BarmanObjectName         string
	ServerName               string
	RecoveryBarmanObjectName string
	RecoveryServerName       string
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

	result := &PluginConfiguration{
		// used for the backup/archive
		BarmanObjectName: helper.Parameters["barmanObjectName"],
		ServerName:       serverName,
		// used for restore/wal_restore
		RecoveryServerName:       recoveryServerName,
		RecoveryBarmanObjectName: recoveryBarmanObjectName,
	}

	return result
}

func getRecoveryParameters(cluster *cnpgv1.Cluster) map[string]string {
	recoveryPluginConfiguration := cluster.GetRecoverySourcePlugin()
	if recoveryPluginConfiguration == nil {
		return nil
	}

	if recoveryPluginConfiguration.Name != metadata.PluginName {
		return nil
	}

	return recoveryPluginConfiguration.Parameters
}

// Validate checks if the barmanObjectName is set
func (p *PluginConfiguration) Validate() error {
	err := NewConfigurationError()

	if len(p.BarmanObjectName) == 0 && len(p.RecoveryBarmanObjectName) == 0 {
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
