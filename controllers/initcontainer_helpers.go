// Filename: controllers/initcontainer_helpers.go
// Changes:
//          1. ИСПРАВЛЕНИЕ: Восстановлено определение переменной 'cmd' в функции downloadDbDriverInitContainer,
//             чтобы устранить ошибку компиляции "undefined: cmd".
//          2. Сохранены изменения по настройке logback на STDOUT (Шаг 250).
// ----------------------------------------------------------------------------------------------------------------

package controllers

import (
	"fmt"
	"strconv"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// configureRegistryPropertiesInitContainer создает InitContainer для настройки nifi-registry.properties.
func configureRegistryPropertiesInitContainer(nifiRegistry *registryv1.NifiRegistry, volumeMounts []corev1.VolumeMount, confMountPath string, externalLibMountPath string) corev1.Container {
	propertiesFile := "nifi-registry.properties"

	// ----------------------------------------------------------------------
	// 1. ЛОГИКА ДЛЯ БД: ГЕНЕРАЦИЯ КОНФИГУРАЦИИ
	// ----------------------------------------------------------------------
	dbSettings := ""
	if nifiRegistry.Spec.Database.Enabled {
		// Включаем настройки БД в nifi-registry.properties
		dbSettings = fmt.Sprintf(`
			# Database Configuration (PostgreSQL)
			sed -i 's|nifi.registry.flow.persistence.provider.implementation.class=.*|nifi.registry.flow.persistence.provider.implementation.class=org.apache.nifi.registry.flow.sql.SqlFlowProvider|g' %s/%s;
			sed -i 's|nifi.registry.flow.persistence.connect.timeout=.*|nifi.registry.flow.persistence.connect.timeout=10s|g' %s/%s;
			
			# Установка пустых значений для предотвращения конфликтов
			sed -i 's|nifi.registry.db.url=.*|nifi.registry.db.url=|g' %s/%s;
			sed -i 's|nifi.registry.db.driver.class=.*|nifi.registry.db.driver.class=|g' %s/%s;
			sed -i 's|nifi.registry.db.driver.directory=.*|nifi.registry.db.driver.directory=|g' %s/%s;
			sed -i 's|nifi.registry.db.username=.*|nifi.registry.db.username=|g' %s/%s;
			sed -i 's|nifi.registry.db.password=.*|nifi.registry.db.password=|g' %s/%s;
		`,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
		)
	}

	// ----------------------------------------------------------------------
	// 2. ЛОГИКА ДЛЯ TLS: ГЕНЕРАЦИЯ КОНФИГУРАЦИИ
	// ----------------------------------------------------------------------
	tlsSettings := ""
	tlsEnabled := "false"
	// NOTE: Используем nifiRegistry.Spec.Tls.Enabled, а не nifiRegistry.Spec.Tls.Port, для проверки включения TLS
	if nifiRegistry.Spec.Tls.Enabled {
		tlsEnabled = "true"
		// Указываем, что keystore и truststore должны находиться в поддиректории /tls каталога conf
		tlsSettings = fmt.Sprintf(`
			sed -i 's|nifi.registry.security.keystore=.*|nifi.registry.security.keystore=%s/tls/keystore.jks|g' %s/%s;
			sed -i 's|nifi.registry.security.keystorePasswd=.*|nifi.registry.security.keystorePasswd=%s|g' %s/%s;
			sed -i 's|nifi.registry.security.truststore=.*|nifi.registry.security.truststore=%s/tls/truststore.jks|g' %s/%s;
			sed -i 's|nifi.registry.security.truststorePasswd=.*|nifi.registry.security.truststorePasswd=%s|g' %s/%s;
			sed -i 's|nifi.registry.security.truststoreType=.*|nifi.registry.security.truststoreType=JKS|g' %s/%s;
			sed -i 's|nifi.registry.security.keystoreType=.*|nifi.registry.security.keystoreType=JKS|g' %s/%s;
			sed -i 's|nifi.registry.security.needClientAuth=.*|nifi.registry.security.needClientAuth=%s|g' %s/%s;
		`,
			confMountPath, confMountPath, propertiesFile,
			nifiRegistry.Spec.Tls.KeystorePassword, confMountPath, propertiesFile,
			confMountPath, confMountPath, propertiesFile,
			nifiRegistry.Spec.Tls.TruststorePassword, confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			confMountPath, propertiesFile,
			nifiRegistry.Spec.Tls.ClientAuth, confMountPath, propertiesFile, // ИСПОЛЬЗУЕМ ClientAuth из CRD
		)
	}

	// ----------------------------------------------------------------------
	// 3. ЛОГИКА ДЛЯ LOGBACK.XML: ПЕРЕКЛЮЧЕНИЕ НА STDOUT
	// ----------------------------------------------------------------------
	// Заменяем имя файла в Appender с nifi-registry-app.log на <target>STDOUT</target>
	// И меняем класс Appender на ConsoleAppender
	logbackPatch := fmt.Sprintf("sed -i 's|<file>logs/nifi-registry-app.log</file>|<target>STDOUT</target>|g' %s/logback.xml && ", confMountPath) +
		fmt.Sprintf("sed -i 's|ch.qos.logback.core.rolling.RollingFileAppender|ch.qos.logback.core.ConsoleAppender|g' %s/logback.xml", confMountPath)

	// ----------------------------------------------------------------------
	// 4. ГЛАВНАЯ КОМАНДА СКРИПТА
	// ----------------------------------------------------------------------
	cmd := fmt.Sprintf(`
		set -e;

		# 1. Настройка TLS/Security:
		%s
		sed -i 's|nifi.registry.security.authorizer=.*|nifi.registry.security.authorizer=managed-authorizer|g' %s/%s;
		sed -i 's|nifi.registry.security.secure.flow.management.actions=.*|nifi.registry.security.secure.flow.management.actions=%s|g' %s/%s;

		# 2. Настройка базы данных:
		%s
		
		# 3. Настройка HTTP/HTTPS портов:
		sed -i 's|nifi.registry.web.http.port=.*|nifi.registry.web.http.port=%s|g' %s/%s;
		sed -i 's|nifi.registry.web.https.port=.*|nifi.registry.web.https.port=%s|g' %s/%s;
		
		# 4. Настройка логирования в STDOUT для Kubernetes:
		%s
		
		# 5. Установка режима DEBUG (ВЫСОКИЙ ПРИОРИТЕТ)
		sed -i 's|<logger name="org.apache.nifi" level="INFO"/>|<logger name="org.apache.nifi" level="DEBUG"/>|g' %s/logback.xml;

		# 6. Вывод финальной конфигурации (DEBUG):
		echo "--- Final %s ---";
		cat %s/%s;
		echo "--- Final logback.xml ---";
		cat %s/logback.xml;
	`,
		// 1. TLS/Security
		tlsSettings,
		confMountPath, propertiesFile,
		tlsEnabled, confMountPath, propertiesFile,

		// 2. Database
		dbSettings,

		// 3. Ports
		strconv.Itoa(18080), confMountPath, propertiesFile, // HTTP PORT
		strconv.Itoa(8443), confMountPath, propertiesFile, // HTTPS PORT

		// 4. Logback Patch
		logbackPatch,

		// 5. DEBUG Logging
		confMountPath,

		// 6. Final Output
		propertiesFile,
		confMountPath, propertiesFile,
		confMountPath,
	)

	return corev1.Container{
		Name:            "configure-registry-properties",
		Image:           nifiRegistry.Spec.InitImage, // Теперь InitImage должен быть определен в CRD (Шаг 252)
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/bash", "-c"},
		Args:            []string{cmd},
		VolumeMounts:    volumeMounts,
	}
}

// copyConfInitContainer создает InitContainer для копирования конфигурации (conf).
func copyConfInitContainer(nifiRegistry *registryv1.NifiRegistry, volumeMounts []corev1.VolumeMount, confMountPath string) corev1.Container {
	return corev1.Container{
		Name:            "copy-conf",
		Image:           nifiRegistry.Spec.InitImage, // Используем InitImage
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"cp"},
		Args:            []string{"-R", "/opt/nifi-registry/nifi-registry-current/conf/.", confMountPath},
		VolumeMounts:    volumeMounts,
	}
}

// downloadDbDriverInitContainer создает InitContainer для загрузки драйвера базы данных.
func downloadDbDriverInitContainer(nifiRegistry *registryv1.NifiRegistry, volumeMounts []corev1.VolumeMount) corev1.Container {
	// --- ИСПРАВЛЕНИЕ: ВОССТАНОВЛЕНА ПЕРЕМЕННАЯ cmd ---
	cmd := fmt.Sprintf(`
		set -e;
		echo "--- Downloading JDBC Driver ---";
		# Проверка: существует ли уже драйвер?
		if [ -f /opt/nifi-registry/external-lib/driver.jar ]; then
			echo "Driver already exists. Skipping download.";
		else
			# Скачивание:
			wget -q -O /opt/nifi-registry/external-lib/driver.jar "%s";
		fi
		ls -l /opt/nifi-registry/external-lib;
		echo "--- Download finished ---";
	`,
		nifiRegistry.Spec.Database.DriverDownloadURL,
	)
	// --- КОНЕЦ ИСПРАВЛЕНИЯ ---

	return corev1.Container{
		Name:            "download-db-driver",
		Image:           nifiRegistry.Spec.InitImage, // Используем InitImage
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/bash", "-c"},
		Args:            []string{cmd},
		VolumeMounts:    volumeMounts,
	}
}

// initContainersForNifiRegistry создает список InitContainers.
func initContainersForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, volumeMounts []corev1.VolumeMount, confMountPath string, externalLibMountPath string) []corev1.Container {

	initContainers := []corev1.Container{
		copyConfInitContainer(nifiRegistry, volumeMounts, confMountPath),
		configureRegistryPropertiesInitContainer(nifiRegistry, volumeMounts, confMountPath, externalLibMountPath),
	}

	if nifiRegistry.Spec.Database.Enabled {
		initContainers = append(initContainers, downloadDbDriverInitContainer(nifiRegistry, volumeMounts))
	}

	return initContainers
}
