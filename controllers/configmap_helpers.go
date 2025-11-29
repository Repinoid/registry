package controllers

import (
	"fmt"
	"net/url"
	"strings"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// getOidcBaseURL извлекает базовый URL Keycloak из DiscoveryURL
func getOidcBaseURL(discoveryURL string) (string, error) {
	u, err := url.Parse(discoveryURL)
	if err != nil {
		return "", err
	}

	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) > 3 && pathParts[len(pathParts)-3] == ".well-known" {
		u.Path = strings.Join(pathParts[:len(pathParts)-3], "/")
	} else {
		lastSlash := strings.LastIndex(u.Path, "/")
		if lastSlash != -1 {
			u.Path = u.Path[:lastSlash]
		}
	}

	return u.Scheme + "://" + u.Host + u.Path, nil
}


// generateIdentityProvidersXML генерирует identity-providers.xml
func generateIdentityProvidersXML(nifiRegistry *registryv1.NifiRegistry, clientSecret string) string {
	keycloakSpec := nifiRegistry.Spec.Keycloak
	baseURL, _ := getOidcBaseURL(keycloakSpec.DiscoveryUrl)

	claimForIdentity := keycloakSpec.ClaimIdentifyingUser
	if claimForIdentity == "" {
		claimForIdentity = "preferred_username"
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<identityProviders>
	<provider>
		<id>keycloak</id>
		<class>org.apache.nifi.registry.security.identity.OidcIdentityProvidersConfigurationContext</class>
		<property name="Oidc User Authentication Provider">org.apache.nifi.registry.security.identity.OidcIdentityProvidersConfigurationContext</property>
		<property name="Client ID">%s</property>
		<property name="Client Secret">%s</property>
		<property name="Token Endpoint">%s/protocol/openid-connect/token</property>
		<property name="Authorize Endpoint">%s/protocol/openid-connect/auth</property>
		<property name="User Info Endpoint">%s/protocol/openid-connect/userinfo</property>
		<property name="JWKS URL">%s/protocol/openid-connect/certs</property>
		<property name="Claim for Identity">%s</property>
		<property name="Claim for Groups"></property>
		<property name="Disable Wildcard Redirect URL">false</property>
	</provider>
</identityProviders>`,
		keycloakSpec.ClientId, clientSecret, baseURL, baseURL, baseURL, baseURL, claimForIdentity)
}

// generateAuthorizersXML генерирует authorizers.xml
func generateAuthorizersXML(nifiRegistry *registryv1.NifiRegistry) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<authorizers>
	<userGroupProvider>
		<id>file-user-group-provider</id>
		<class>org.apache.nifi.registry.security.authorization.file.FileUserGroupProvider</class>
		<property name="Users File">./conf/users.xml</property>
		<property name="Groups File">./conf/groups.xml</property>
		<property name="Initial User Identity 1"></property>
	</userGroupProvider>

	<authorizer>
		<id>file-access-policy-provider</id>
		<class>org.apache.nifi.registry.security.authorization.file.FileAccessPolicyProvider</class>
		<property name="User Group Provider">file-user-group-provider</property>
		<property name="Authorizations File">./conf/authorizations.xml</property>
		<property name="Initial Admin Identity">%s</property> 
		<property name="Node Identity"></property>
		<property name="Node Group Identity"></property>
	</authorizer>
</authorizers>`, nifiRegistry.Spec.Tls.AdminIdentity)
}

// configMapForNifiRegistry генерирует ConfigMap с конфигурационными файлами
func configMapForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, name string, clientSecret string, dbPassword string, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: 		 name,
			Namespace: nifiRegistry.Namespace,
			Labels: 	 labels,
		},
		Data: map[string]string{
			// КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: УДАЛЕНА настройка nifi-registry.properties.
			// Теперь Pod будет использовать дефолтный файл из образа, который работает с H2.
			// "nifi-registry.properties": generateNifiRegistryProperties(nifiRegistry, dbPassword), 
			
			"identity-providers.xml": 	generateIdentityProvidersXML(nifiRegistry, clientSecret),
			"authorizers.xml": 				generateAuthorizersXML(nifiRegistry),
			"logback.xml": 						generateLogbackXML(),
		},
	}

	ctrl.SetControllerReference(nifiRegistry, configMap, scheme)
	return configMap
}

// generateLogbackXML генерирует logback.xml
func generateLogbackXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<appender name="CONSOLE" class="ch.qos.logback.core.ConsoleAppender">
		<encoder>
			<pattern>%date %level [%thread] %logger{40} %msg%n</pattern>
		</encoder>
	</appender>

	<logger name="org.apache.nifi" level="DEBUG"/>
	<logger name="org.apache.nifi.authorization" level="DEBUG"/>
	<logger name="org.apache.nifi.authentication" level="DEBUG"/>
	<logger name="org.apache.nifi.web.security" level="DEBUG"/>
	<logger name="org.apache.nifi.security" level="DEBUG"/>
	<logger name="org.apache.nifi.cluster" level="DEBUG"/>
	<logger name="org.springframework" level="DEBUG"/>
	<logger name="org.eclipse.jetty" level="DEBUG"/>

	<root level="INFO">
		<appender-ref ref="CONSOLE"/>
	</root>
</configuration>`
}