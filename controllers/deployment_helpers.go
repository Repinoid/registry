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
	
    // ПУТЬ ДЛЯ ВНЕШНИХ ДРАЙВЕРОВ
    externalLibMountPath := "/opt/nifi-registry/external_lib"


	// Определяем, нужен ли InitContainer для копирования JDBC драйвера.
	var initContainers []corev1.Container

	// InitContainer нужен, только если используется внешняя БД (PostgreSQL)
	if nifiRegistry.Spec.Database.Enabled {
		initContainers = []corev1.Container{
			{
				Name:  "copy-postgres-driver",
				Image: "curlimages/curl:latest",
				Command: []string{
					"sh",
					"-c",
					// Копируем драйвер в новый, неперекрывающий каталог: /external_lib
					"curl -sL https://jdbc.postgresql.org/download/postgresql-42.7.3.jar -o " + externalLibMountPath + "/postgresql-jdbc.jar",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      libStorageVolumeName,
						MountPath: externalLibMountPath, // <--- ИЗМЕНЕН ПУТЬ МОНТИРОВАНИЯ
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

	// Если включена внешняя БД, добавляем все переменные окружения для подключения к БД
	if nifiRegistry.Spec.Database.Enabled {
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
				Value: externalLibMountPath, // <--- ИСПРАВЛЕНО: Указываем новый путь для драйвера
			},
		}
		envVars = append(envVars, dbEnv...)
	}

	// VolumeMounts
	volumeMounts := []corev1.VolumeMount{}
	if nifiRegistry.Spec.FlowStorage.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      flowStorageVolumeName,
			MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage",
		})
	}
	// LibStorage нужен, если включен FlowStorage ИЛИ Database.
	if nifiRegistry.Spec.LibStorage.Enabled || nifiRegistry.Spec.Database.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      libStorageVolumeName,
			MountPath: externalLibMountPath, // <--- ИЗМЕНЕН ПУТЬ МОНТИРОВАНИЯ
		})
	}

	// Volumes
	volumes := []corev1.Volume{}
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
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 18080,
									Name:          "web-port",
								},
							},
							Env: envVars,
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
