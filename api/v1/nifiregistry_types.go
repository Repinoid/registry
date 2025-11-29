// api/v1/nifiregistry_types.go

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- Вспомогательные структуры ---

// TlsSpec определяет настройки TLS
type TlsSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=8443
	Port int32 `json:"port,omitempty"`

	Host string `json:"host,omitempty"`

	KeystorePassword   string `json:"keystorePassword,omitempty"`
	TruststorePassword string `json:"truststorePassword,omitempty"`

	AdminIdentity string `json:"adminIdentity,omitempty"`
}

// KeycloakSpec определяет настройки Keycloak OIDC
type KeycloakSpec struct {
	DiscoveryUrl         string `json:"discoveryUrl,omitempty"`
	ClientId             string `json:"clientId,omitempty"`
	ClaimIdentifyingUser string `json:"claimIdentifyingUser,omitempty"`
	ClientSecretName     string `json:"clientSecretName,omitempty"`
}

// DatabaseSpec определяет настройки внешней БД (для подключения NiFi Registry)
type DatabaseSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	Url         string `json:"url,omitempty"`
	DriverClass string `json:"driverClass,omitempty"`
	Username    string `json:"username,omitempty"`
	SecretName  string `json:"secretName,omitempty"`
}

// FlowStorageSpec определяет спецификацию Persistent Volume Claim (PVC)
type FlowStorageSpec struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// +kubebuilder:validation:Required
	Size resource.Quantity `json:"size"`

	StorageClass string `json:"storageClass,omitempty"`
}

// ImageSpec определяет репозиторий и тег образа NiFi Registry
type ImageSpec struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// PostgreSQLDeploySpec определяет настройки для развертывания внутреннего PostgreSQL
type PostgreSQLDeploySpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	Image string `json:"image,omitempty"`

	// +kubebuilder:default="1Gi"
	Size string `json:"size,omitempty"`

	StorageClass string `json:"storageClass,omitempty"`
}

// --- Основная структура ---

// NifiRegistrySpec определяет желаемое состояние NifiRegistry
type NifiRegistrySpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Size int32 `json:"size,omitempty"`

	// Image, Repository, Tag
	Image ImageSpec `json:"image,omitempty"`

	// Resources определяет ограничения ресурсов (CPU/Memory) для контейнера NiFi Registry.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Tls TlsSpec `json:"tls,omitempty"`

	Keycloak KeycloakSpec `json:"keycloak,omitempty"`

	// Конфигурация подключения к внешней БД (используется NiFi Registry)
	Database DatabaseSpec `json:"database,omitempty"`

	// Конфигурация для развертывания PostgreSQL (управляется оператором)
	PostgreSQL PostgreSQLDeploySpec `json:"postgresql,omitempty"`

	FlowStorage FlowStorageSpec `json:"flowStorage,omitempty"`
}

// NifiRegistryStatus определяет наблюдаемое состояние NifiRegistry
type NifiRegistryStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NifiRegistry — это Custom Resource для деплоя NiFi Registry.
type NifiRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NifiRegistrySpec   `json:"spec,omitempty"`
	Status NifiRegistryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NifiRegistryList содержит список NifiRegistry
type NifiRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NifiRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NifiRegistry{}, &NifiRegistryList{})
}
