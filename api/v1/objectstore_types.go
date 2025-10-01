/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProbeConfig holds configuration for probe timing and thresholds
// This is a subset of the corev1.Probe type, with only the fields that we want to expose as configuration.
type ProbeConfig struct {
	// InitialDelaySeconds is the number of seconds after the container has started before startup probes are initiated.
	// +kubebuilder:default:=0
	// +optional
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`

	// TimeoutSeconds is the number of seconds after which the probe times out.
	// +kubebuilder:default:=10
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// PeriodSeconds is how often (in seconds) to perform the probe.
	// +kubebuilder:default:=10
	// +optional
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`

	// SuccessThreshold is the minimum consecutive successes for the probe to be considered successful.
	// +kubebuilder:default:=1
	// +optional
	SuccessThreshold int32 `json:"successThreshold,omitempty"`

	// FailureThreshold is the minimum consecutive failures for the probe to be considered failed.
	// +kubebuilder:default:=10
	// +optional
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
}

// InstanceSidecarConfiguration defines the configuration for the sidecar that runs in the instance pods.
type InstanceSidecarConfiguration struct {
	// The environment to be explicitly passed to the sidecar
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// The retentionCheckInterval defines the frequency at which the
	// system checks and enforces retention policies.
	// +kubebuilder:default:=1800
	// +optional
	RetentionPolicyIntervalSeconds int `json:"retentionPolicyIntervalSeconds,omitempty"`

	// Resources define cpu/memory requests and limits for the sidecar that runs in the instance pods.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// StartupProbe defines the configuration for the startup probe of the sidecar container.
	// +optional
	StartupProbe *ProbeConfig `json:"startupProbe,omitempty"`

	// LivenessProbe defines the configuration for the liveness probe of the sidecar container.
	// +optional
	LivenessProbe *ProbeConfig `json:"livenessProbe,omitempty"`

	// ReadinessProbe defines the configuration for the readiness probe of the sidecar container.
	// +optional
	ReadinessProbe *ProbeConfig `json:"readinessProbe,omitempty"`
	// AdditionalContainerArgs is an optional list of command-line arguments
	// to be passed to the sidecar container when it starts.
	// The provided arguments are appended to the container’s default arguments.
	// +kubebuilder:validation:XValidation:rule="!self.exists(a, a.startsWith('--log-level'))",reason="FieldValueForbidden",message="do not set --log-level in additionalContainerArgs; use spec.instanceSidecarConfiguration.logLevel"
	// +optional
	AdditionalContainerArgs []string `json:"additionalContainerArgs,omitempty"`

	// The log level for PostgreSQL instances. Valid values are: `error`, `warning`, `info` (default), `debug`, `trace`
	// +kubebuilder:default:=info
	// +kubebuilder:validation:Enum:=error;warning;info;debug;trace
	// +optional
	LogLevel string `json:"logLevel,omitempty"`
}

// ObjectStoreSpec defines the desired state of ObjectStore.
type ObjectStoreSpec struct {
	// The configuration for the barman-cloud tool suite
	// +kubebuilder:validation:XValidation:rule="!has(self.serverName)",fieldPath=".serverName",reason="FieldValueForbidden",message="use the 'serverName' plugin parameter in the Cluster resource"
	Configuration barmanapi.BarmanObjectStoreConfiguration `json:"configuration"`

	// RetentionPolicy is the retention policy to be used for backups
	// and WALs (i.e. '60d'). The retention policy is expressed in the form
	// of `XXu` where `XX` is a positive integer and `u` is in `[dwm]` -
	// days, weeks, months.
	// +kubebuilder:validation:Pattern=^[1-9][0-9]*[dwm]$
	// +optional
	RetentionPolicy string `json:"retentionPolicy,omitempty"`

	// The configuration for the sidecar that runs in the instance pods
	// +optional
	InstanceSidecarConfiguration InstanceSidecarConfiguration `json:"instanceSidecarConfiguration,omitempty"`
}

// ObjectStoreStatus defines the observed state of ObjectStore.
type ObjectStoreStatus struct {
	// ServerRecoveryWindow maps each server to its recovery window
	ServerRecoveryWindow map[string]RecoveryWindow `json:"serverRecoveryWindow,omitempty"`
}

// RecoveryWindow represents the time span between the first
// recoverability point and the last successful backup of a PostgreSQL
// server, defining the period during which data can be restored.
type RecoveryWindow struct {
	// The first recoverability point in a PostgreSQL server refers to
	// the earliest point in time to which the database can be
	// restored.
	FirstRecoverabilityPoint *metav1.Time `json:"firstRecoverabilityPoint,omitempty"`

	// The last successful backup time
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`

	// The last failed backup time
	LastFailedBackupTime *metav1.Time `json:"lastFailedBackupTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +kubebuilder:storageversion

// ObjectStore is the Schema for the objectstores API.
type ObjectStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Specification of the desired behavior of the ObjectStore.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec ObjectStoreSpec `json:"spec"`
	// Most recently observed status of the ObjectStore. This data may not be up to
	// date. Populated by the system. Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status ObjectStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObjectStoreList contains a list of ObjectStore.
type ObjectStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ObjectStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ObjectStore{}, &ObjectStoreList{})
}
