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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NifiRegistrySpec defines the desired state of NifiRegistry
type NifiRegistrySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file

	// +kubebuilder:validation:Minimum=1
	// Size is the size of the NiFi Registry cluster.
	Size int32 `json:"size"`

	// Image defines the container image used for NiFi Registry.
	Image ImageSpec `json:"image,omitempty"`

	// Resources defines the resource requests and limits for the NiFi Registry Pod.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// FlowStorage defines persistent storage configuration for flow definitions.
	FlowStorage FlowStorageSpec `json:"flowStorage,omitempty"`

	// Database defines the database connection details for NiFi Registry.
	Database DatabaseSpec `json:"database,omitempty"`

	// PostgreSQL defines the specification for an optional embedded PostgreSQL instance.
	PostgreSQL PostgresSpec `json:"postgreSQL,omitempty"`

	// Tls defines the TLS/SSL configuration for NiFi Registry.
	Tls TlsSpec `json:"tls,omitempty"` // <-- ВОССТАНОВЛЕНО
}

// ImageSpec defines the image repository and tag.
type ImageSpec struct {
	// Repository is the image repository (e.g., apache/nifi-registry).
	Repository string `json:"repository,omitempty"`
	// Tag is the image tag (e.g., 1.25.0).
	Tag string `json:"tag,omitempty"`
}

// FlowStorageSpec defines persistent storage configuration for flow definitions.
type FlowStorageSpec struct {
	// Enabled determines whether to use persistent storage for flows.
	Enabled bool `json:"enabled,omitempty"`
	// StorageSize defines the size of the PersistentVolumeClaim (e.g., 1Gi).
	StorageSize string `json:"storageSize,omitempty"` // Имя поля, используемое в pvc_helpers.go
	// StorageClassName defines the StorageClass to use for the PVC.
	StorageClassName string `json:"storageClassName,omitempty"` // Имя поля, используемое в pvc_helpers.go
}

// DatabaseSpec defines the connection details for the database.
type DatabaseSpec struct {
	// Enabled determines whether to use an external database.
	Enabled bool `json:"enabled,omitempty"`
	// Url is the JDBC connection URL for the database (e.g., jdbc:postgresql://host:port/dbname).
	Url string `json:"url,omitempty"`
	// DriverClass is the JDBC driver class (e.g., org.postgresql.Driver).
	DriverClass string `json:"driverClass,omitempty"`
	// Username is the database username.
	Username string `json:"username,omitempty"`
	// SecretName will be used to hardcode the password string into the Deployment Environment Variable.
	SecretName string `json:"secretName,omitempty"`
}

// PostgresSpec defines the specification for an optional embedded PostgreSQL instance.
type PostgresSpec struct {
	// Enabled determines whether to deploy an embedded PostgreSQL instance.
	Enabled bool `json:"enabled,omitempty"`
	// Image defines the PostgreSQL container image (e.g., postgres:15-alpine).
	Image string `json:"image,omitempty"`
	// Size defines the size of the PersistentVolumeClaim for PostgreSQL data (e.g., 2Gi).
	Size string `json:"size,omitempty"`
	// StorageClass defines the StorageClass to use for the PostgreSQL PVC.
	StorageClass string `json:"storageClass,omitempty"` // <-- ДОБАВЛЕНО для postgresql_helpers.go
}

// TlsSpec defines the TLS/SSL configuration.
type TlsSpec struct {
	// Enabled determines whether to enable TLS/SSL for the NiFi Registry service.
	Enabled bool `json:"enabled,omitempty"`
} // <-- ВОССТАНОВЛЕНО

// NifiRegistryStatus defines the observed state of NifiRegistry
type NifiRegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS - define the observed state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
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
