// api/v1/nifiregistry_types.go

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags at least on the top level struct.

// NifiRegistrySpec defines the desired state of NifiRegistry
type NifiRegistrySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make manifests" to regenerate code after modifying this file

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:default=1
	Size int32 `json:"size,omitempty"`

	// Image defines the Nifi Registry container image to use
	Image ImageSpec `json:"image,omitempty"`

	// Resources defines the compute resources for the Nifi Registry Pod
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tls defines the TLS settings for the Nifi Registry
	Tls TlsSpec `json:"tls,omitempty"`

	// Keycloak defines the OIDC settings for Keycloak authentication
	Keycloak KeycloakSpec `json:"keycloak,omitempty"`

	// FlowStorage defines the settings for Nifi Registry flow persistence
	FlowStorage FlowStorageSpec `json:"flowStorage,omitempty"`

	// Database defines the external database connection settings
	Database DatabaseSpec `json:"database,omitempty"`

	// PostgreSQL defines the settings for the managed PostgreSQL instance
	PostgreSQL PostgreSQLSpec `json:"postgreSQL,omitempty"`
}

// ImageSpec defines the container image repository and tag
type ImageSpec struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// TlsSpec defines the TLS configuration
type TlsSpec struct {
	Enabled            bool   `json:"enabled,omitempty"`
	Port               int32  `json:"port,omitempty"`
	Host               string `json:"host,omitempty"`
	KeystorePassword   string `json:"keystorePassword,omitempty"`
	TruststorePassword string `json:"truststorePassword,omitempty"`
	AdminIdentity      string `json:"adminIdentity,omitempty"`
}

// KeycloakSpec defines the OIDC settings for Keycloak
type KeycloakSpec struct {
	DiscoveryUrl         string `json:"discoveryUrl,omitempty"`
	ClientId             string `json:"clientId,omitempty"`
	ClaimIdentifyingUser string `json:"claimIdentifyingUser,omitempty"`
	ClientSecretName     string `json:"clientSecretName,omitempty"`
}

// FlowStorageSpec определяет настройки для хранилища NiFi Registry
type FlowStorageSpec struct {
	Enabled      bool   `json:"enabled,omitempty"`
	Size         string `json:"size,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
}

// DatabaseSpec defines the external database connection settings
type DatabaseSpec struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Url         string `json:"url,omitempty"`
	DriverClass string `json:"driverClass,omitempty"`
	Username    string `json:"username,omitempty"`
	SecretName  string `json:"secretName,omitempty"`
}

// PostgreSQLSpec defines the settings for the managed PostgreSQL instance
type PostgreSQLSpec struct {
	Enabled      bool      `json:"enabled,omitempty"`
	Image        ImageSpec `json:"image,omitempty"`
	Database     string    `json:"database,omitempty"`
	Username     string    `json:"username,omitempty"`
	Password     string    `json:"password,omitempty"`
	Size         string    `json:"size,omitempty"`
	StorageClass string    `json:"storageClass,omitempty"`
}

// NifiRegistryStatus defines the observed state of NifiRegistry
type NifiRegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS - define status fields
	// Important: Run "make manifests" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NifiRegistry is the Schema for the nifiregistries API
type NifiRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NifiRegistrySpec   `json:"spec,omitempty"`
	Status NifiRegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NifiRegistryList contains a list of NifiRegistry
type NifiRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NifiRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NifiRegistry{}, &NifiRegistryList{})
}
