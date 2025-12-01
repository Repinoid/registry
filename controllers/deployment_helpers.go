// Filename: controllers/deployment_helpers.go
// Changes: 1. Удалено ВРЕМЕННОЕ ИЗМЕНЕНИЕ: удалена команда "sleep infinity", чтобы NiFi Registry начал запуск.
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

	// ПУТИ
	externalLibMountPath := "/opt/nifi-registry/external_lib"
	confMountPath := "/opt/nifi-registry/nifi-registry-current/conf"
	confMountDest := "/mnt/conf" // Путь в Init-контейнерах

	// =======================================
	// 2. Сборка компонентов
	// =======================================

	// 2.1 InitContainers (из initcontainer_helpers.go)
	initContainers := initContainersForNifiRegistry(nifiRegistry, confMountDest, confMountPath, externalLibMountPath, libStorageVolumeName)

	// 2.2 EnvVars и Ports (из envvars_helpers.go)
	envVars, containerPorts := envVarsAndPortsForNifiRegistry(nifiRegistry, confMountPath, externalLibMountPath)

	// 2.3 Volumes (из volume_helpers.go)
	volumes := volumesForNifiRegistry(nifiRegistry, flowStorageVolumeName, libStorageVolumeName, confVolumeName, tlsSecretVolumeName)

	// 2.4 VolumeMounts (из volume_helpers.go)
	volumeMounts := volumeMountsForNifiRegistry(nifiRegistry, flowStorageVolumeName, libStorageVolumeName, confVolumeName, tlsSecretVolumeName, confMountPath, externalLibMountPath)

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
							// **************************************************************************************
							// * ВРЕМЕННЫЙ DEBUG-РЕЖИМ: sleep infinity для предотвращения CrashLoopBackOff 			*
							// * и возможности просмотра внутреннего лога через kubectl exec. 						*
							// **************************************************************************************
							// Command: []string{"/bin/sh", "-c"},
							// Args:    []string{"sleep infinity"},

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
