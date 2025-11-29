/*
Copyright 2024 Repinoid.

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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NifiRegistrySpec defines the desired state of NifiRegistry
type NifiRegistrySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file

	// Size is the number of desired NiFi Registry replicas
	Size int32 `json:"size"`

	// Port defines the primary HTTP port for the NiFi Registry service (default is 8080).
	// +optional
	// +kubebuilder:default:=8080
	Port int32 `json:"port,omitempty"`

	// Image definition for NiFi Registry
	Image ImageSpec `json:"image"`

	// TLS configuration
	Tls TlsSpec `json:"tls"`

	// Keycloak OIDC configuration
	Keycloak KeycloakSpec `json:"keycloak"`

	// Flow Storage configuration
	FlowStorage FlowStorageSpec `json:"flowStorage"`

	// Database configuration
	Database DatabaseSpec `json:"database"`
}

// ImageSpec defines the container image properties
type ImageSpec struct {
	Repository string            `json:"repository"`
	Tag        string            `json:"tag"`
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// TlsSpec defines TLS configuration for the NiFi Registry
type TlsSpec struct {
	Enabled            bool   `json:"enabled"`
	Port               int32  `json:"port"`
	Host               string `json:"host"`
	KeystorePassword   string `json:"keystorePassword"`
	TruststorePassword string `json:"truststorePassword"`
	AdminIdentity      string `json:"adminIdentity"`
}

// KeycloakSpec defines the Keycloak OIDC settings
type KeycloakSpec struct {
	DiscoveryUrl         string `json:"discoveryUrl"`
	ClientId             string `json:"clientId"`
	ClaimIdentifyingUser string `json:"claimIdentifyingUser"`
	ClientSecretName     string `json:"clientSecretName"`
}

// FlowStorageSpec defines the persistent storage for flow versions
type FlowStorageSpec struct {
	Enabled      bool              `json:"enabled"`
	Size         resource.Quantity `json:"size"`
	StorageClass string            `json:"storageClass"`
}

// DatabaseSpec defines the external database configuration
type DatabaseSpec struct {
	Enabled     bool   `json:"enabled"`
	Url         string `json:"url"`
	DriverClass string `json:"driverClass"`
	Username    string `json:"username"`
	SecretName  string `json:"secretName"`
}

// NifiRegistryStatus defines the observed state of NifiRegistry
type NifiRegistryStatus struct {
	// Represents the observations of a NifiRegistry's current state.
	// Known type-specific conditions are defined below.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMapKey:"type"`

	// State represents the current high-level state of the NiFi Registry resource.
	// For example: Creating, Running, Failed, Upgrading.
	// +optional
	State string `json:"state,omitempty"` // <--- ИСПРАВЛЕНИЕ: ДОБАВЛЕНО ПОЛЕ STATE
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NifiRegistry is the Schema for the nifiregistries API
type NifiRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NifiRegistrySpec   `json:"spec,omitempty"`
	Status NifiRegistryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NifiRegistryList contains a list of NifiRegistry
type NifiRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NifiRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NifiRegistry{}, &NifiRegistryList{})
}
