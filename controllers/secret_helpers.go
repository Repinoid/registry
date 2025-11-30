// Filename: controllers/secret_helpers.go
// Changes: Added functions to generate Kubernetes Secrets for TLS keystores and Keycloak Client Secret.

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// !!! ВНИМАНИЕ: Это ЗАГЛУШКИ. В реальном операторе эти данные должны
// быть СГЕНЕРИРОВАНЫ (например, с помощью InitContainer) или взяты
// из внешнего источника, например, Vault.
const tlsKeystoreData = "PLACEHOLDER_JKS_KEYSTORE_DATA"
const tlsTruststoreData = "PLACEHOLDER_JKS_TRUSTSTORE_DATA"

// tlsSecretForNifiRegistry генерирует Secret для Keystore и Truststore (TLS).
// Secret требуется для монтирования файлов .jks в файловую систему Pod.
func tlsSecretForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.Secret {
	labels := map[string]string{"app": nifiRegistry.Name}
	secretName := fmt.Sprintf("%s-tls-secret", nifiRegistry.Name)

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			// Файлы, которые будут смонтированы в /conf/tls/
			"keystore.jks":  []byte(tlsKeystoreData),
			"truststore.jks": []byte(tlsTruststoreData),
		},
	}

	ctrl.SetControllerReference(nifiRegistry, sec, scheme)
	return sec
}

// keycloakSecretForNifiRegistry генерирует Secret, содержащий Client Secret Keycloak,
// используя ссылку на имя Secret из CRD (ClientSecretName).
// Для простоты, здесь мы используем заглушку, поскольку оператор не управляет содержимым
// этого Secret, а только ссылается на него.
func keycloakSecretForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.Secret {
	labels := map[string]string{"app": nifiRegistry.Name}
	
	// Используем имя Secret, указанное в CRD
	secretName := nifiRegistry.Spec.Keycloak.ClientSecretName
	
	// ВНИМАНИЕ: Если ClientSecretName пуст, это может вызвать ошибку.
	// В рабочем коде здесь должна быть проверка. Предполагаем, что оно задано.
	if secretName == "" {
		secretName = fmt.Sprintf("%s-keycloak-secret-default", nifiRegistry.Name)
	}
	
	// Здесь мы не можем получить Client Secret, потому что он не хранится в CRD.
	// Мы предполагаем, что Secret с именем nifiRegistry.Spec.Keycloak.ClientSecretName
	// уже существует в кластере и содержит ключ "client-secret".
	
	// Для целей генерации ресурса Secret, нам нужен Secret, чтобы оператор мог
	// проверить его существование. Поскольку мы не знаем его содержимого, мы
	// создадим пустой Secret с нужным именем, чтобы избежать ошибок, 
	// если вы захотите, чтобы оператор его создал.
	
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			// Требуется наличие ключа "client-secret" для монтирования в ENV
			"client-secret": []byte("REPLACE_ME_WITH_REAL_KEYCLOAK_SECRET"), 
		},
	}

	ctrl.SetControllerReference(nifiRegistry, sec, scheme)
	return sec
}
