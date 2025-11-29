// controllers/deployment_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deploymentForNifiRegistry генерирует Deployment для NiFi Registry
func deploymentForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name}

	replicas := nifiRegistry.Spec.Size

	fullImage := fmt.Sprintf("%s:%s", nifiRegistry.Spec.Image.Repository, nifiRegistry.Spec.Image.Tag)

	// Инициализация списков для Volumes, VolumeMounts и InitContainers
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initContainers := []corev1.Container{}

	// Общие переменные
	flowStorageVolumeName := "flow-storage-volume"
	flowStorageMountPath := "/opt/nifi-registry/nifi-registry-current/flow_storage"

	// 1. Volumes, Volume Mounts и Init Container для Flow Storage

	if nifiRegistry.Spec.FlowStorage.Enabled {
		// Имя PVC должно совпадать с тем, что мы создаем в pvcForNifiRegistry
		pvcName := fmt.Sprintf("%s-flow-storage", nifiRegistry.Name) 

		// Добавляем PVC Volume
		volumes = append(volumes, corev1.Volume{
			Name: flowStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
					ReadOnly:  false,
				},
			},
		})

		// Volume Mounts для Flow Storage
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      flowStorageVolumeName,
			MountPath: flowStorageMountPath,
		})

		// Init Container для chown
		initContainers = append(initContainers, corev1.Container{
			Name:    "init-data-chown",
			Image:   "busybox",
			Command: []string{"sh", "-c", "chown -R 1000:1000 " + flowStorageMountPath},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      flowStorageVolumeName,
					MountPath: flowStorageMountPath,
				},
			},
		})
	}

	// 2. Environment Variables (Настройка сети и внешней БД)
	envVars := []corev1.EnvVar{
		{
			Name:  "NIFI_REGISTRY_WEB_HTTP_HOST",
			Value: "0.0.0.0",
		},
		{
			Name:  "NIFI_REGISTRY_WEB_HTTP_PORT",
			Value: "18080",
		},
	}

	// Если внешняя БД включена, добавляем соответствующие переменные окружения
	if nifiRegistry.Spec.Database.Enabled {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "NIFI_REGISTRY_DB_IMPLEMENTATION",
			Value: "postgresql", 
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "NIFI_REGISTRY_DB_URL",
			Value: nifiRegistry.Spec.Database.Url,
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "NIFI_REGISTRY_DB_DRIVER_CLASS",
			Value: nifiRegistry.Spec.Database.DriverClass,
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "NIFI_REGISTRY_DB_USERNAME",
			Value: nifiRegistry.Spec.Database.Username,
		})

		// ПРЯМОЙ ПАРОЛЬ: Берем значение из SecretName (теперь это поле пароля)
		if nifiRegistry.Spec.Database.SecretName != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name: "NIFI_REGISTRY_DB_PASSWORD",
				// Пароль хардкодится прямо в переменную окружения
				Value: nifiRegistry.Spec.Database.SecretName,
			})
		}
	}

	// Основной контейнер NiFi Registry
	nifiRegistryContainer := corev1.Container{
		Name:    "nifi-registry",
		Image:   fullImage,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 18080,
				Name:          "http-port",
			},
		},
		VolumeMounts: volumeMounts,
		Resources:    nifiRegistry.Spec.Resources,
		Env:          envVars,
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
					InitContainers: initContainers,
					Containers: []corev1.Container{
						nifiRegistryContainer,
					},
					Volumes: volumes,
				},
			},
		},
	}

	ctrl.SetControllerReference(nifiRegistry, dep, scheme)
	return dep
}
