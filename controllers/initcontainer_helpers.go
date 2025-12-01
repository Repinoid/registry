// Filename: controllers/initcontainer_helpers.go
// Changes: 1. Исправлены имена свойств TLS: удален суффикс ".path" (nifi.registry.security.keystore.path -> nifi.registry.security.keystore)
//          2. Заменен URL для JDBC-драйвера PostgreSQL на жестко заданный URL (из-за ошибки компиляции Go).
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

	// 2. Контейнер для конфигурации nifi-registry.properties (TLS и Database)
	// Этот скрипт выполняет:
	// 1. Настройку TLS
	// 2. Отключение HTTP и включение HTTPS
	// 3. Настройку Flow Provider на SQL
	// 4. Настройку JDBC
	configurePropertiesScript := `
set -xe;
echo -e '\n# TLS Configuration' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.keystore=%s/tls/keystore.jks' >> %s/nifi-registry.properties; # <-- ИСПРАВЛЕНО
echo 'nifi.registry.security.keystore.password=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.keystore.type=JKS' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.truststore=%s/tls/truststore.jks' >> %s/nifi-registry.properties; # <-- ИСПРАВЛЕНО
echo 'nifi.registry.security.truststore.password=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.truststore.type=JKS' >> %s/nifi-registry.properties;
echo 'nifi.registry.security.client.auth=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.web.https.host=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.web.https.port=%s' >> %s/nifi-registry.properties;
sed -i '/nifi.registry.web.http.port=/c\#nifi.registry.web.http.port=8080' %s/nifi-registry.properties;
echo -e '\n# Database Configuration' >> %s/nifi-registry.properties;
echo 'nifi.registry.flow.provider=%s' >> %s/nifi-registry.properties;
echo 'nifi.registry.db.implementation=%s' >> %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.url=.*$|nifi.registry.db.url=%s|g' %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.driver.class=.*$|nifi.registry.db.driver.class=%s|g' %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.username=.*$|nifi.registry.db.username=%s|g' %s/nifi-registry.properties;
sed -i 's|^nifi.registry.db.password=.*$|nifi.registry.db.password=%s|g' %s/nifi-registry.properties;
`

	configurePropertiesCommand := fmt.Sprintf(
		configurePropertiesScript,
		confMountDest, // 1, 6, 10, 14, 17, 20, 23, 26, 29, 32
		// TLS
		confMountPath, nifiRegistry.Spec.Tls.KeystorePassword,
		confMountDest, nifiRegistry.Spec.Tls.KeystorePassword,
		confMountDest,
		confMountPath, nifiRegistry.Spec.Tls.TruststorePassword,
		confMountDest, nifiRegistry.Spec.Tls.TruststorePassword,
		confMountDest,
		nifiRegistry.Spec.Tls.ClientAuth, confMountDest,
		"0.0.0.0", confMountDest, // HTTPS Host
		"8443", confMountDest, // HTTPS Port
		confMountDest, // sed for HTTP Port
		// DB
		confMountDest,
		"org.apache.nifi.flow.sql.SqlFlowProvider", confMountDest,
		"org.apache.nifi.db.sql.SqlFlowPersistenceProvider", confMountDest,
		"jdbc:postgresql://postgres-service:5432/nifiregistry", confMountDest,
		"org.postgresql.Driver", confMountDest,
		"nifiregistry", confMountDest,
		"password", confMountDest,
	)

	configurePropertiesContainer := corev1.Container{
		Name:    "configure-registry-properties",
		Image:   "busybox",
		Command: []string{"sh", "-c"},
		Args:    []string{configurePropertiesCommand},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "nifi-registry-conf",
				MountPath: confMountDest,
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
