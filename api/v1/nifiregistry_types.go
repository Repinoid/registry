package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- Структуры, скопированные из Nifi Operator ---

// ImageSpec defines the container image properties
type ImageSpec struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

// KeycloakSpec defines the OIDC connection properties using Keycloak
type KeycloakSpec struct {
	DiscoveryUrl         string `json:"discoveryUrl"`
	ClientId             string `json:"clientId"`
	ClientSecretName     string `json:"clientSecretName"` // <-- ИМЯ SECRET
	ClaimIdentifyingUser string `json:"claimIdentifyingUser,omitempty"`
}

// TlsSpec defines the TLS properties for security configuration
type TlsSpec struct {
	Enabled            bool   `json:"enabled"`
	ClientAuth         string `json:"clientAuth,omitempty"`
	KeystorePassword   string `json:"keystorePassword"`
	TruststorePassword string `json:"truststorePassword"`
	AdminIdentity      string `json:"adminIdentity"`
	Host               string `json:"host"`
	Port               int32  `json:"port"`
}

// PersistenceSpec defines the volume persistence properties
type PersistenceSpec struct {
	Enabled bool `json:"enabled"`
	// Используем resource.Quantity, как в рабочем NiFi Operator
	Size         resource.Quantity `json:"size"`
	StorageClass string            `json:"storageClass,omitempty"`
}

// DatabaseSpec defines the external PostgreSQL database connection
type DatabaseSpec struct {
	Enabled     bool   `json:"enabled"`
	DriverClass string `json:"driverClass"`
	Url         string `json:"url"`
	Username    string `json:"username"`
	SecretName  string `json:"secretName"`
}

// --- Структура NifiRegistrySpec ---

// NifiRegistrySpec defines the desired state of NifiRegistry
type NifiRegistrySpec struct {
	Size        int32           `json:"size"`
	Image       ImageSpec       `json:"image"`
	Tls         TlsSpec         `json:"tls"`
	Keycloak    KeycloakSpec    `json:"keycloak"`
	FlowStorage PersistenceSpec `json:"flowStorage"`
	Database    DatabaseSpec    `json:"database"`
}

// NifiRegistryStatus defines the observed state of NifiRegistry
type NifiRegistryStatus struct {
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type NifiRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NifiRegistrySpec   `json:"spec,omitempty"`
	Status            NifiRegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type NifiRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NifiRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NifiRegistry{}, &NifiRegistryList{})
}
