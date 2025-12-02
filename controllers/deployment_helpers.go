// Filename: controllers/deployment_helpers.go
// Changes:
//          1. ИСПРАВЛЕНИЕ: Исправлен вызов initContainersForNifiRegistry в Шаге 2.1.
//             Переданы правильные 4 аргумента: nifiRegistry, volumeMounts (слайс), confMountPath, externalLibMountPath.
// ----------------------------------------------------------------------------------------------------------------

package controllers

import (
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

	// =======================================
	// 1. Определение констант и имен
	// =======================================

	// Имена томов
	flowStorageVolumeName := nifiRegistry.Name + "-flow"
	libStorageVolumeName := nifiRegistry.Name + "-lib"
	confVolumeName := "nifi-registry-conf" // EmptyDir для конфигурации (для InitContainer)
	tlsSecretVolumeName := "tls-keystore"  // Том для Secret TLS

	// НОВЫЕ ТОМЫ ДЛЯ KEYCLOAK CONFIGMAP
	identityConfVolumeName := nifiRegistry.Name + "-identity-cm"
	authorizersConfVolumeName := nifiRegistry.Name + "-authorizers-cm"

	// ПУТИ
	externalLibMountPath := "/opt/nifi-registry/external_lib"
	confMountPath := "/opt/nifi-registry/nifi-registry-current/conf"
	// confMountDest := "/mnt/conf" // ЭТА ПЕРЕМЕННАЯ НЕ НУЖНА В ЭТОМ ФАЙЛЕ

	// ПУТИ МОНТИРОВАНИЯ KEYCLOAK
	identityConfMountPath := confMountPath + "/identity-providers.xml"
	authorizersConfMountPath := confMountPath + "/authorizers.xml"

	// =======================================
	// 2. Сборка компонентов
	// =======================================

	// 2.3 Volumes (из volume_helpers.go)
	volumes := volumesForNifiRegistry(nifiRegistry, flowStorageVolumeName, libStorageVolumeName, confVolumeName, tlsSecretVolumeName)

	// 2.4 VolumeMounts (из volume_helpers.go)
	volumeMounts := volumeMountsForNifiRegistry(nifiRegistry, flowStorageVolumeName, libStorageVolumeName, confVolumeName, tlsSecretVolumeName, confMountPath, externalLibMountPath)

	// ДОБАВЛЕНИЕ ТОМОВ KEYCLOAK
	if nifiRegistry.Spec.Keycloak.Enabled {
		// Том для identity-providers.xml
		volumes = append(volumes, corev1.Volume{
			Name: identityConfVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-identity-providers-cm",
					},
				},
			},
		})
		// Том для authorizers.xml
		volumes = append(volumes, corev1.Volume{
			Name: authorizersConfVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-authorizers-cm",
					},
				},
			},
		})

		// ДОБАВЛЕНИЕ МОНТИРОВАНИЯ KEYCLOAK
		// Монтирование identity-providers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      identityConfVolumeName,
			MountPath: identityConfMountPath,
			SubPath:   "identity-providers.xml",
			ReadOnly:  true,
		})
		// Монтирование authorizers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      authorizersConfVolumeName,
			MountPath: authorizersConfMountPath,
			SubPath:   "authorizers.xml",
			ReadOnly:  true,
		})
	}

	// 2.1 InitContainers (из initcontainer_helpers.go)
	// ИСПРАВЛЕНИЕ ВЫЗОВА: 4 аргумента правильных типов:
	initContainers := initContainersForNifiRegistry(
		nifiRegistry,
		volumeMounts, // <-- ИСПОЛЬЗУЕМ СЛАЙС volumeMounts, определенный выше
		confMountPath,
		externalLibMountPath,
	)

	// 2.2 EnvVars и Ports (из envvars_helpers.go)
	envVars, containerPorts := envVarsAndPortsForNifiRegistry(nifiRegistry, confMountPath, externalLibMountPath)

	// =======================================
	// 3. Создание Deployment
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
							// Command и Args убраны (больше не в DEBUG-режиме)

							Ports: containerPorts,
							Env:   envVars,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  int64Ptr(1000),
								RunAsGroup: int64Ptr(1000),
							},
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
