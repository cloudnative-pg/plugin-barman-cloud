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

package config

import (
	"strings"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/machinery/pkg/stringset"
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

	// AdditionalBarmanObjectNames lists the object stores that a Backup
	// resource is allowed to request on top of the ones used by the cluster
	// itself. Nothing is written to them unless a Backup asks for one, but
	// they take part in the RBAC and in the certificates of the instances.
	AdditionalBarmanObjectNames []string
}

// GetBarmanObjectKey gets the namespaced name of the barman object
func (config *PluginConfiguration) GetBarmanObjectKey() types.NamespacedName {
	return types.NamespacedName{
		Namespace: config.Cluster.Namespace,
		Name:      config.BarmanObjectName,
	}
}

// ApplyBackupParameters overrides the object store selection with the
// parameters of the Backup resource. The operator relays them in the
// BackupRequest, and without this a Backup asking for a different object
// store is silently written to the cluster one.
func (config *PluginConfiguration) ApplyBackupParameters(parameters map[string]string) {
	if len(parameters) == 0 {
		return
	}

	if value := parameters["barmanObjectName"]; len(value) > 0 {
		config.BarmanObjectName = value
	}

	if value := parameters["serverName"]; len(value) > 0 {
		config.ServerName = value
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
		Name:      config.ReplicaSourceBarmanObjectName,
	}
}

// HasAnyBarmanObjectStore reports whether any barman object store is configured.
func (config *PluginConfiguration) HasAnyBarmanObjectStore() bool {
	return len(config.BarmanObjectName) > 0 ||
		len(config.RecoveryBarmanObjectName) > 0 ||
		len(config.ReplicaSourceBarmanObjectName) > 0
}

// GetReferredBarmanObjectsKey gets the list of barman objects referred by this
// plugin configuration
func (config *PluginConfiguration) GetReferredBarmanObjectsKey() []types.NamespacedName {
	objectNames := stringset.New()
	if len(config.BarmanObjectName) > 0 {
		objectNames.Put(config.BarmanObjectName)
	}
	if len(config.RecoveryBarmanObjectName) > 0 {
		objectNames.Put(config.RecoveryBarmanObjectName)
	}
	if len(config.ReplicaSourceBarmanObjectName) > 0 {
		objectNames.Put(config.ReplicaSourceBarmanObjectName)
	}
	for _, name := range config.AdditionalBarmanObjectNames {
		objectNames.Put(name)
	}

	result := make([]types.NamespacedName, 0, 4)
	for _, name := range objectNames.ToSortedList() {
		result = append(result, types.NamespacedName{
			Name:      name,
			Namespace: config.Cluster.Namespace,
		})
	}

	return result
}

// NewFromClusterJSON decodes a JSON representation of a cluster.
func NewFromClusterJSON(clusterJSON []byte) (*PluginConfiguration, error) {
	var result cnpgv1.Cluster

	if err := decoder.DecodeObjectLenient(clusterJSON, &result); err != nil {
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
		if len(replicaSourceServerName) == 0 {
			replicaSourceServerName = cluster.Name
		}
	}

	result := &PluginConfiguration{
		Cluster: cluster,
		// used for the backup/archive
		BarmanObjectName: helper.Parameters["barmanObjectName"],
		ServerName:       serverName,
		// reachable by a Backup resource requesting them explicitly
		AdditionalBarmanObjectNames: parseObjectNameList(helper.Parameters["additionalBarmanObjectNames"]),
		// used for restore and wal_restore during backup recovery
		RecoveryServerName:       recoveryServerName,
		RecoveryBarmanObjectName: recoveryBarmanObjectName,
		// used for wal_restore in the designed primary of a replica cluster
		ReplicaSourceServerName:       replicaSourceServerName,
		ReplicaSourceBarmanObjectName: replicaSourceBarmanObjectName,
	}

	return result
}

// parseObjectNameList splits a comma separated list of object store names,
// dropping the empty entries
func parseObjectNameList(value string) []string {
	if len(value) == 0 {
		return nil
	}

	var result []string
	for _, name := range strings.Split(value, ",") {
		if name = strings.TrimSpace(name); len(name) > 0 {
			result = append(result, name)
		}
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

	if !config.HasAnyBarmanObjectStore() {
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
