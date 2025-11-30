// Filename: controllers/configmap_helpers.go
// Changes: Added functions configMapIdentityProvidersForNifiRegistry and configMapAuthorizersForNifiRegistry
// to support Keycloak OIDC configuration (Identity Providers and Initial Admin Identity).

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Константа, содержащая ИСПРАВЛЕННОЕ содержимое providers.xml (Оставлено по запросу пользователя)
const registryProvidersXml = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<providers>
	<flowPersistenceProvider>
		<class>org.apache.nifi.registry.provider.flow.DatabaseFlowPersistenceProvider</class>
	</flowPersistenceProvider>

	<extensionBundlePersistenceProvider>
		<class>org.apache.nifi.registry.provider.extension.FileSystemBundlePersistenceProvider</class>
		<property name="Extension Bundle Storage Directory">./extension_bundles</property>
	</extensionBundlePersistenceProvider>
	</providers>`

// configMapForNifiRegistry генерирует ConfigMap для NiFi Registry (Оставлено по запросу пользователя)
func configMapForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}
	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"providers.xml": registryProvidersXml,
		},
	}

	ctrl.SetControllerReference(nifiRegistry, cm, scheme)
	return cm
}

// ====================================================================================
// НОВЫЕ ФУНКЦИИ ДЛЯ KEYCLOAK (OIDC)
// ====================================================================================

// configMapIdentityProvidersForNifiRegistry генерирует ConfigMap для identity-providers.xml (OIDC Keycloak)
func configMapIdentityProvidersForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}

	// Используем имя CRD + "-identity-config" для нового ConfigMap
	configMapName := fmt.Sprintf("%s-identity-config", nifiRegistry.Name)

	// Константа для пути обратного вызова OIDC
	const oidcCallbackPath = "/nifi-registry-api/access/oidc/callback"

	// 1. Формируем Discovery URL и Redirect URL
	// Пример: https://registry.k8c.ru/realms/nifier/.well-known/openid-configuration
	discoveryURL := fmt.Sprintf("%s/realms/%s", nifiRegistry.Spec.Keycloak.ExternalURL, nifiRegistry.Spec.Keycloak.Realm)
	redirectURL := fmt.Sprintf("%s%s", nifiRegistry.Spec.Keycloak.ExternalURL, oidcCallbackPath)

	// 2. Генерируем содержимое XML
	identityProvidersXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<identityProviders>
    <provider>
        <id>oidc-keycloak-provider</id>
        <class>org.apache.nifi.registry.security.identity.OidcIdentityProvider</class>
        <property name="OIDC Provider Discovery URL">%s</property>
        <property name="Client ID">%s</property>
        <property name="Redirect URL">%s</property>
        <property name="Claim Identifying User">%s</property>
        <property name="Claim Identifying User Group"></property>
        <property name="Callback Path">%s</property>
    </provider>
</identityProviders>`,
		discoveryURL,
		nifiRegistry.Spec.Keycloak.ClientId,
		redirectURL,
		nifiRegistry.Spec.Keycloak.ClaimIdentifyingUser,
		oidcCallbackPath,
	)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"identity-providers.xml": identityProvidersXml,
		},
	}

	ctrl.SetControllerReference(nifiRegistry, cm, scheme)
	return cm
}

// configMapAuthorizersForNifiRegistry генерирует ConfigMap для authorizers.xml
func configMapAuthorizersForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}
	configMapName := fmt.Sprintf("%s-authorizers-config", nifiRegistry.Name)

	// Получаем имя первого администратора из CRD
	initialAdminIdentity := nifiRegistry.Spec.Keycloak.InitialAdminIdentity

	// 1. Генерируем содержимое XML
	authorizersXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<authorizers>
    <authorizer>
        <identifier>managed-authorizer</identifier>
        <class>org.apache.nifi.registry.security.authorization.ConfigurableAccessPolicyProvider</class>
        <property name="User Group Provider">file-user-group-provider</property>
        <property name="Initial Admin Identity">%s</property>
        <property name="Authorization Access Policy Provider">file-access-policy-provider</property>
        <property name="Access Policy Provider">file-access-policy-provider</property>
        <property name="Access Policy Provider Implementation">org.apache.nifi.registry.security.authorization.FileSystemAccessPolicyProvider</class>
        <property name="Authorizations File">./conf/authorizations.xml</property>
    </authorizer>

    <userGroupProvider>
        <identifier>file-user-group-provider</identifier>
        <class>org.apache.nifi.registry.security.authorization.FileUserGroupProvider</class>
        <property name="Users File">./conf/users.xml</property>
        <property name="Legacy Authorized Users File"></property>
    </userGroupProvider>

    <accessPolicyProvider>
        <identifier>file-access-policy-provider</identifier>
        <class>org.apache.nifi.registry.security.authorization.FileSystemAccessPolicyProvider</class>
        <property name="Authorizations File">./conf/authorizations.xml</property>
        <property name="Initial Admin Identity">%s</property>
    </accessPolicyProvider>
</authorizers>`,
		initialAdminIdentity,
		initialAdminIdentity,
	)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"authorizers.xml": authorizersXml,
		},
	}

	ctrl.SetControllerReference(nifiRegistry, cm, scheme)
	return cm
}
