// Filename: controllers/envvars_helpers.go
// Changes: 1. Добавление опции -Xmx к NIFI_REGISTRY_JAVA_OPTS.
// ----------------------------------------------------------------------------------------------------------------

package controllers

import (
	"strconv"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// envVarsAndPortsForNifiRegistry создает и возвращает список EnvVars и ContainerPorts.
func envVarsAndPortsForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, confMountPath, externalLibMountPath string) ([]corev1.EnvVar, []corev1.ContainerPort) {
	// Переменные окружения NiFi Registry
	envVars := []corev1.EnvVar{
		{
			Name:  "NIFI_REGISTRY_WEB_HTTP_PORT",
			Value: strconv.Itoa(18080),
		},
		{
			Name:  "NIFI_REGISTRY_WEB_HTTP_HOST",
			Value: "0.0.0.0",
		},
	}

	containerPorts := []corev1.ContainerPort{}

	// =======================================
	// 1. ЛОГИКА TLS
	// =======================================

	if nifiRegistry.Spec.Tls.Enabled {
		// Изменяем порт для HTTP
		envVars[0].Value = "" // Удаляем NIFI_REGISTRY_WEB_HTTP_PORT

		// Добавляем ENV для HTTPS
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_WEB_HTTPS_HOST",
				Value: "0.0.0.0",
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_WEB_HTTPS_PORT",
				Value: strconv.Itoa(8443),
			},
			// Указываем, что keystore и truststore должны находиться в поддиректории /tls каталога conf
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_KEYSTORE_PATH",
				Value: confMountPath + "/tls/keystore.jks",
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_KEYSTORE_PASSWORD",
				Value: nifiRegistry.Spec.Tls.KeystorePassword,
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_KEYSTORE_TYPE",
				Value: "JKS",
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_TRUSTSTORE_PATH",
				Value: confMountPath + "/tls/truststore.jks",
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_TRUSTSTORE_PASSWORD",
				Value: nifiRegistry.Spec.Tls.TruststorePassword,
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_TRUSTSTORE_TYPE",
				Value: "JKS",
			},
			corev1.EnvVar{
				Name:  "NIFI_REGISTRY_CLIENT_AUTH",
				Value: "REQUIRED",
			},
		)

		// Добавляем порт HTTPS
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: 8443,
			Name:          "https-port",
		})
	} else {
		// Добавляем порт HTTP
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: 18080,
			Name:          "web-port",
		})
	}

	// =======================================
	// 1.1 ЛОГИКА KEYCLOAK (OIDC)
	// =======================================
	if nifiRegistry.Spec.Keycloak.Enabled {
		// Добавление Client Secret для OIDC (берется из Secret)
		envVars = append(envVars, corev1.EnvVar{
			Name: "NIFI_REGISTRY_OIDC_CLIENT_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Spec.Keycloak.ClientSecretName, // ИСПОЛЬЗУЕМ ИМЯ ИЗ CRD
					},
					Key: "client-secret", // Ключ из Secret (см. secret_helpers.go)
				},
			},
		})
	}

	// Если включена внешняя БД, добавляем все переменные окружения для подключения к БД
	if nifiRegistry.Spec.Database.Enabled {
		// Установка стандартных ENV (переопределяют nifi-registry.properties)
		dbEnv := []corev1.EnvVar{
			// Flow Persistence Provider Settings
			{
				Name:  "NIFI_REGISTRY_FLOW_PROVIDER",
				Value: "org.apache.nifi.registry.flow.sql.SqlFlowProvider",
			},
			// Database Configuration (PostgreSQL)
			{
				Name:  "NIFI_REGISTRY_DB_IMPLEMENTATION",
				Value: "org.apache.nifi.registry.db.sql.SqlFlowPersistenceProvider",
			},
			{
				Name:  "NIFI_REGISTRY_DB_URL",
				Value: nifiRegistry.Spec.Database.Url,
			},
			{
				Name:  "NIFI_REGISTRY_DB_DRIVER_CLASS",
				Value: nifiRegistry.Spec.Database.DriverClass,
			},
			{
				Name:  "NIFI_REGISTRY_DB_USERNAME",
				Value: nifiRegistry.Spec.Database.Username,
			},
			// ПАРОЛЬ ОТКРЫТЫМ ТЕКСТОМ (для тестового стенда)
			{
				Name:  "NIFI_REGISTRY_DB_PASSWORD",
				Value: nifiRegistry.Spec.Database.Password,
			},
			{
				Name:  "NIFI_REGISTRY_DB_DRIVER_LIB_DIR",
				Value: externalLibMountPath, // Указываем изолированный путь для драйвера
			},
		}
		envVars = append(envVars, dbEnv...)

		// ДОБАВЛЕНИЕ NIFI_REGISTRY_JAVA_OPTS для принудительной установки драйвера в Flyway/Spring Boot
		dbUrl := nifiRegistry.Spec.Database.Url
		driverClass := nifiRegistry.Spec.Database.DriverClass
		dbUsername := nifiRegistry.Spec.Database.Username
		dbPassword := nifiRegistry.Spec.Database.Password

		// ОГРАНИЧЕНИЕ ПАМЯТИ JVM: берем Request Memory (512Mi)
		memoryRequest := nifiRegistry.Spec.Resources.Requests.Memory().String()

		// Новые опции с включением логина, пароля, принудительным классом драйвера Flyway и loader.path
		javaOptsValue := "-Xmx" + memoryRequest +
			" -Dspring.datasource.driver-class-name=" + driverClass +
			" -Dspring.datasource.url=" + dbUrl +
			" -Dspring.datasource.username=" + dbUsername +
			" -Dspring.datasource.password=" + dbPassword +
			" -Dspring.flyway.driver-class-name=" + driverClass +
			" -Dloader.path=" + externalLibMountPath // <--- ИСПОЛЬЗУЕТ ИЗОЛИРОВАННЫЙ CLASSPATH

		javaOptsEnv := corev1.EnvVar{
			Name:  "NIFI_REGISTRY_JAVA_OPTS",
			Value: javaOptsValue,
		}
		envVars = append(envVars, javaOptsEnv)
	}

	return envVars, containerPorts
}
