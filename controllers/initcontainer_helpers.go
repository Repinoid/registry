// Filename: controllers/initcontainer_helpers.go
// Changes: 1. Исправлена ошибка подсчета аргументов в fmt.Sprintf, вызвавшая ошибку "reads arg #37, but call has only 36 args".
//          2. Использован образ bash:latest для Init-контейнера.
// ----------------------------------------------------------------------------------------------------------------

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// initContainersForNifiRegistry возвращает список Init-контейнеров для настройки NiFi Registry.
func initContainersForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, confMountDest string, confMountPath string, externalLibMountPath string, libStorageVolumeName string) []corev1.Container {
	var containers []corev1.Container

	// 1. Контейнер для копирования конфигурации по умолчанию
	copyConfContainer := corev1.Container{
		Name:    "copy-conf",
		Image:   nifiRegistry.Spec.Image.Repository + ":" + nifiRegistry.Spec.Image.Tag,
		Command: []string{"sh", "-c"},
		Args:    []string{fmt.Sprintf("cp -R %s/. %s", confMountPath, confMountDest)},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "nifi-registry-conf", // MountName из deployment_helpers.go
				MountPath: confMountDest,
			},
		},
	}
	containers = append(containers, copyConfContainer)

	// =====================================================================================================
	// 2. Контейнер для конфигурации nifi-registry.properties (TLS, Database, Keycloak)
	// =====================================================================================================
	configurePropertiesScript := `
set -xe;

# TLS Configuration
echo -e '\n# TLS Configuration' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.keystore=%s/tls/keystore.jks' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.keystore.password=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.keystore.type=JKS' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.truststore=%s/tls/truststore.jks' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.truststore.password=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.truststore.type=JKS' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.client.auth=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.web.https.host=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.web.https.port=%s' >> %s/nifi-registry.properties;
sed -i '/nifi.registry.web.http.port=/c\#nifi.registry.web.http.port=8080' %s/nifi-registry.properties;

# Database Configuration
echo -e '\n# Database Configuration' >> %s/nifi-registry.properties;
echo 'nifi.registry.flow.provider=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.db.implementation=%s' >> %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.url=.*$|nifi.registry.db.url=%s|g' %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.driver.class=.*$|nifi.registry.db.driver.class=%s|g' %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.username=.*$|nifi.registry.db.username=%s|g' %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.password=.*$|nifi.registry.db.password=%s|g' %s/nifi-registry.properties;

# Keycloak/OIDC Configuration (если включен)
if [ "%t" = "true" ]; then
    echo -e '\n# OIDC Configuration' >> %s/nifi-registry.properties;
    echo 'nifi.registry.security.identity.providers.configuration.file=%s/identity-providers.xml' >> %s/nifi-registry.properties;
    echo 'nifi.registry.security.authorizer.configuration.file=%s/authorizers.xml' >> %s/nifi-registry.properties;
fi
`

	configurePropertiesCommand := fmt.Sprintf(
		configurePropertiesScript,
		confMountDest, // 1. (TLS start)

		// TLS properties: 14 аргументов (2 * 7)
		confMountPath, confMountDest, // 2, 3. keystore path/confMountDest
		nifiRegistry.Spec.Tls.KeystorePassword, confMountDest, // 4, 5. keystore password/confMountDest
		confMountDest, // 6. keystore type (>> %s)

		confMountPath, confMountDest, // 7, 8. truststore path/confMountDest
		nifiRegistry.Spec.Tls.TruststorePassword, confMountDest, // 9, 10. truststore password/confMountDest
		confMountDest, // 11. truststore type (>> %s)

		nifiRegistry.Spec.Tls.ClientAuth, confMountDest, // 12, 13. client auth/confMountDest

		"0.0.0.0", confMountDest, // 14, 15. HTTPS Host/confMountDest
		"8443", confMountDest, // 16, 17. HTTPS Port/confMountDest
		confMountDest, // 18. sed for HTTP Port

		// DB properties: 12 аргументов
		confMountDest, // 19. (DB start)
		"org.apache.nifi.flow.sql.SqlFlowProvider", confMountDest, // 20, 21. flow provider/confMountDest
		"org.apache.nifi.registry.db.sql.SqlFlowPersistenceProvider", confMountDest, // 22, 23. db implementation/confMountDest
		nifiRegistry.Spec.Database.Url, confMountDest, // 24, 25. db url/confMountDest
		nifiRegistry.Spec.Database.DriverClass, confMountDest, // 26, 27. db driver class/confMountDest
		nifiRegistry.Spec.Database.Username, confMountDest, // 28, 29. db username/confMountDest
		nifiRegistry.Spec.Database.Password, confMountDest, // 30, 31. db password/confMountDest

		// Keycloak/OIDC: 6 аргументов (32-37)
		nifiRegistry.Spec.Keycloak.Enabled, // 32. %t
		confMountDest, // 33. confMountDest (echo start)
		confMountPath, confMountDest, // 34, 35. identity providers path/confMountDest
		confMountPath, confMountDest, // 36, 37. authorizers path/confMountDest
	)

	configurePropertiesContainer := corev1.Container{
		Name:    "configure-registry-properties",
		Image:   "bash:latest", // Используем стандартный образ bash
		Command: []string{"sh", "-c"},
		Args:    []string{configurePropertiesCommand},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "nifi-registry-conf",
				MountPath: confMountDest,
			},
			// Добавление VolumeMount для Secret TLS
			{
				Name:      "tls-keystore",
				MountPath: confMountPath + "/tls", // /opt/nifi-registry/nifi-registry-current/conf/tls
				ReadOnly:  true,
			},
		},
	}
	containers = append(containers, configurePropertiesContainer)

	// 3. Контейнер для загрузки JDBC драйвера
	// ИСПРАВЛЕНИЕ: Жестко заданный URL для обхода ошибки компиляции Go с динамическим URL
	downloadDriverCommand := fmt.Sprintf("curl -sL %s -o %s/postgresql-jdbc.jar", "https://jdbc.postgresql.org/download/postgresql-42.7.3.jar", externalLibMountPath)
	downloadDriverContainer := corev1.Container{
		Name:    "download-db-driver",
		Image:   "curlimages/curl:latest",
		Command: []string{"sh", "-c"},
		Args:    []string{downloadDriverCommand},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      libStorageVolumeName,
				MountPath: externalLibMountPath,
			},
		},
	}
	containers = append(containers, downloadDriverContainer)

	return containers
}
