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

// InstanceSidecarConfiguration defines the configuration for the sidecar that runs in the instance pods.
type InstanceSidecarConfiguration struct {
	// The expiration time of the cache entries not managed by the informers. Expressed in seconds.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3600
	// +kubebuilder:default=180
	CacheTTL *int `json:"cacheTTL,omitempty"`

	// The environment to be explicitely passed to the sidecar
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// GetCacheTTL returns the cache TTL value, defaulting to 180 seconds if not set.
func (i InstanceSidecarConfiguration) GetCacheTTL() int {
	if i.CacheTTL == nil {
		return 180
	}
	return *i.CacheTTL
}

// ObjectStoreSpec defines the desired state of ObjectStore.
type ObjectStoreSpec struct {
	Configuration barmanapi.BarmanObjectStoreConfiguration `json:"configuration"`

	// +optional
	InstanceSidecarConfiguration InstanceSidecarConfiguration `json:"instanceSidecarConfiguration,omitempty"`
}

// ObjectStoreStatus defines the observed state of ObjectStore.
type ObjectStoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +kubebuilder:storageversion

// ObjectStore is the Schema for the objectstores API.
type ObjectStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ObjectStoreSpec `json:"spec"`
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
