// Filename: controllers/deployment_helpers.go
// Changes: Fixed the undefined field error 'nifiRegistry.Spec.Tls.SecretName'
//          by defaulting the SecretName to '<nifiRegistry.Name>-tls-secret',
//          which is the expected name for the generated Secret.

package controllers

import (
	"strconv"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Вспомогательная функция (обязательна для *int64)
func int64Ptr(val int64) *int64 {
	return &val
}

// deploymentForNifiRegistry возвращает Deployment для NiFi Registry.
func deploymentForNifiRegistry(nifiRegistry *registryv1.NifiRegistry) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name}
	replicas := int32(nifiRegistry.Spec.Size)

	// Имена томов
	flowStorageVolumeName := nifiRegistry.Name + "-flow"
	libStorageVolumeName := nifiRegistry.Name + "-lib"
	confVolumeName := "nifi-registry-conf" // EmptyDir для конфигурации (для InitContainer)
	tlsSecretVolumeName := "tls-keystore"  // Том для Secret TLS
	
	// ПУТЬ ДЛЯ ВНЕШНИХ ДРАЙВЕРОВ
	externalLibMountPath := "/opt/nifi-registry/external_lib"
	
	// ПУТЬ ДЛЯ ФАЙЛОВ КОНФИГУРАЦИИ
	confMountPath := "/opt/nifi-registry/nifi-registry-current/conf"
    
    // Имя TLS Secret по умолчанию (если SecretName не определено в API, используем это имя)
    // SecretName по умолчанию = <имя ресурса>-tls-secret
    tlsSecretName := nifiRegistry.Name + "-tls-secret"


	// Определяем InitContainers
	var initContainers []corev1.Container

	// InitContainers нужны, только если используется внешняя БД (PostgreSQL)
	if nifiRegistry.Spec.Database.Enabled {
		initContainers = []corev1.Container{
			// Контейнер 0: Копирует конфигурацию из образа в EmptyDir (делает ее доступной для записи)
			{
				Name:    "copy-conf",
				Image: nifiRegistry.Spec.Image.Repository + ":" + nifiRegistry.Spec.Image.Tag, // Используем основной образ
				Command: []string{
					"sh",
					"-c",
					"cp -R /opt/nifi-registry/nifi-registry-current/conf/. /mnt/conf",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      confVolumeName,
						MountPath: "/mnt/conf",
					},
				},
			},
			// Контейнер 1: Скачивает драйвер PostgreSQL
			{
				Name:    "download-db-driver",
				Image: "curlimages/curl:latest", // Используем легкий образ с curl для скачивания
				Command: []string{
					"sh",
					"-c",
					// Скачиваем драйвер в смонтированный PVC
					"curl -sL https://jdbc.postgresql.org/download/postgresql-42.7.3.jar -o " + externalLibMountPath + "/postgresql-jdbc.jar",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      libStorageVolumeName,
						MountPath: externalLibMountPath, // Монтируем PVC Lib Storage
					},
				},
			},
		}
	}

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

	// =======================================
	// 1. ЛОГИКА TLS
	// =======================================
	containerPorts := []corev1.ContainerPort{}

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
			// Указываем, что keystore и truststore находятся в поддиректории /tls каталога conf
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
				Value: externalLibMountPath, // Указываем путь для драйвера
			},
		}
		envVars = append(envVars, dbEnv...)

		// ДОБАВЛЕНИЕ NIFI_REGISTRY_JAVA_OPTS для принудительной установки драйвера в Flyway/Spring Boot
		dbUrl := nifiRegistry.Spec.Database.Url
		driverClass := nifiRegistry.Spec.Database.DriverClass
		dbUsername := nifiRegistry.Spec.Database.Username
		dbPassword := nifiRegistry.Spec.Database.Password

		// Новые опции с включением логина, пароля и принудительным классом драйвера Flyway
		javaOptsValue := "-Dspring.datasource.driver-class-name=" + driverClass +
			" -Dspring.datasource.url=" + dbUrl +
			" -Dspring.datasource.username=" + dbUsername +
			" -Dspring.datasource.password=" + dbPassword +
			" -Dspring.flyway.driver-class-name=" + driverClass

		javaOptsEnv := corev1.EnvVar{
			Name:  "NIFI_REGISTRY_JAVA_OPTS",
			Value: javaOptsValue,
		}
		envVars = append(envVars, javaOptsEnv)
	}

	// =======================================
	// 2. VolumeMounts (добавление TLS и Keycloak)
	// =======================================

	volumeMounts := []corev1.VolumeMount{}

	// 1. Монтирование flow storage
	if nifiRegistry.Spec.FlowStorage.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      flowStorageVolumeName,
			MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage",
		})
	}

	// 2. Монтирование lib storage (для драйвера PostgreSQL, если БД включена)
	if nifiRegistry.Spec.LibStorage.Enabled || nifiRegistry.Spec.Database.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      libStorageVolumeName,
			MountPath: externalLibMountPath, // Монтируем внешний Lib в /external_lib
		})
	}

	// 3. Монтирование тома /conf (EmptyDir)
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      confVolumeName,
		MountPath: confMountPath, // ПРАВИЛЬНЫЙ ПУТЬ
	})

	// 4. Монтирование TLS Secret (в conf/tls)
	if nifiRegistry.Spec.Tls.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      tlsSecretVolumeName,
			MountPath: confMountPath + "/tls", // Монтируем Secret в поддиректорию conf/tls
			ReadOnly:  true,
		})
	}

	// 5. Монтирование ConfigMaps для Keycloak (в conf)
	if nifiRegistry.Spec.Keycloak.Enabled {
		// providers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      nifiRegistry.Name + "-providers-cm",
			MountPath: confMountPath + "/providers.xml",
			SubPath:   "providers.xml",
			ReadOnly:  true,
		})
		// identity-providers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      nifiRegistry.Name + "-identity-providers-cm",
			MountPath: confMountPath + "/identity-providers.xml",
			SubPath:   "identity-providers.xml",
			ReadOnly:  true,
		})
		// authorizers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      nifiRegistry.Name + "-authorizers-cm",
			MountPath: confMountPath + "/authorizers.xml",
			SubPath:   "authorizers.xml",
			ReadOnly:  true,
		})
	}


	// =======================================
	// 3. Volumes (добавление TLS и Keycloak)
	// =======================================

	volumes := []corev1.Volume{}

	// 1. Том для Flow Storage
	if nifiRegistry.Spec.FlowStorage.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: flowStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: nifiRegistry.Name + "-flow",
				},
			},
		})
	}

	// 2. Том для Lib Storage
	if nifiRegistry.Spec.LibStorage.Enabled || nifiRegistry.Spec.Database.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: libStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: nifiRegistry.Name + "-lib",
				},
			},
		})
	}

	// 3. Том EmptyDir для конфигурации (для доступа на запись InitContainers)
	volumes = append(volumes, corev1.Volume{
		Name: confVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	
	// 4. Том для Secret (TLS Keystore/Truststore)
	if nifiRegistry.Spec.Tls.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: tlsSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsSecretName, // <-- ИСПРАВЛЕНО: Используем имя Secret по умолчанию
					Items: []corev1.KeyToPath{
						{
							Key:  "keystore.jks",
							Path: "keystore.jks", // Монтируется в conf/tls/keystore.jks
						},
						{
							Key:  "truststore.jks",
							Path: "truststore.jks", // Монтируется в conf/tls/truststore.jks
						},
					},
				},
			},
		})
	}

	// 5. Тома для ConfigMaps (Keycloak/OIDC)
	if nifiRegistry.Spec.Keycloak.Enabled {
		// providers.xml
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Name + "-providers-cm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-providers-cm",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "providers.xml",
							Path: "providers.xml", // Монтируется в /conf/providers.xml
						},
					},
				},
			},
		})
		// identity-providers.xml
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Name + "-identity-providers-cm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-identity-providers-cm",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "identity-providers.xml",
							Path: "identity-providers.xml", // Монтируется в /conf/identity-providers.xml
						},
					},
				},
			},
		})
		// authorizers.xml
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Name + "-authorizers-cm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-authorizers-cm",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "authorizers.xml",
							Path: "authorizers.xml", // Монтируется в /conf/authorizers.xml
						},
					},
				},
			},
		})
	}


	// =======================================
	// 4. Создание Deployment
	// =======================================

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: int64Ptr(1000),
					},
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  nifiRegistry.Name,
							Image: nifiRegistry.Spec.Image.Repository + ":" + nifiRegistry.Spec.Image.Tag,
							// ******************************************************************************
							// * ИСПРАВЛЕНИЕ: ЯВНОЕ ОПРЕДЕЛЕНИЕ COMMAND И ARGS ДЛЯ УДЕРЖАНИЯ ПРОЦЕССА *
							// ******************************************************************************
							Command: []string{"/opt/nifi-registry/nifi-registry-current/bin/nifi-registry.sh"},
							Args:    []string{"run"},
							// ******************************************************************************
							Ports: containerPorts, // Используем обновленный список портов
							Env:   envVars,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(nifiRegistry.Spec.Resources.Requests.Cpu().String()),
									corev1.ResourceMemory: resource.MustParse(nifiRegistry.Spec.Resources.Requests.Memory().String()),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(nifiRegistry.Spec.Resources.Limits.Cpu().String()),
									corev1.ResourceMemory: resource.MustParse(nifiRegistry.Spec.Resources.Limits.Memory().String()),
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return dep
}