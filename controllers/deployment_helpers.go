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
	initContainers := []corev1.Container{} // <-- Инициализация списка Init-контейнеров

	// Общие переменные для томов NiFi Registry
	flowStorageVolumeName := "flow-storage-volume"
	flowStorageMountPath := "/opt/nifi-registry/nifi-registry-current/flow_storage"

	// Если Flow Storage включен, добавляем PVC volume, mount и Init-контейнер <-- ВОЗВРАЩЕНО
	if nifiRegistry.Spec.FlowStorage.Enabled {
		pvcName := fmt.Sprintf("%s-flow", nifiRegistry.Name)

		// 1. Volumes
		volumes = append(volumes, corev1.Volume{
			Name: flowStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
					ReadOnly:  false,
				},
			},
		})

		// 2. Volume Mounts для основного контейнера
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      flowStorageVolumeName,
			MountPath: flowStorageMountPath,
		})

		// 3. Init Container для chown (изменение прав доступа) <-- ДОБАВЛЕНО
		initContainers = append(initContainers, corev1.Container{
			Name:    "init-data-chown",
			Image:   "busybox", // Используем легковесный образ
			Command: []string{"sh", "-c", "chown -R 1000:1000 " + flowStorageMountPath},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      flowStorageVolumeName,
					MountPath: flowStorageMountPath,
				},
			},
		})
	}

	// 4. Environment Variables (Добавляем массив переменных окружения)
	envVars := []corev1.EnvVar{
		{
			Name:  "NIFI_REGISTRY_WEB_HTTP_HOST",
			Value: "0.0.0.0", // Исправляем ошибку Connection Refused
		},
	}

	// Основной контейнер NiFi Registry
	nifiRegistryContainer := corev1.Container{
		Name:  "nifi-registry",
		Image: fullImage,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 18080,
				Name:          "http-port",
			},
		},
		VolumeMounts: volumeMounts,
		Resources:    nifiRegistry.Spec.Resources,
		Env:          envVars, // <-- ПРИМЕНЯЕМ массив envVars к контейнеру
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
					// Init-контейнеры ДОЛЖНЫ быть в PodSpec
					InitContainers: initContainers, // <-- Используем восстановленный список
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
