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
	replicas := int32(1)

	// Имя ConfigMap
	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)
	// Имя PVC
	pvcName := fmt.Sprintf("%s-flow", nifiRegistry.Name)

	const flowStorageMountPath = "/opt/nifi-registry/nifi-registry-current/flow_storage"

	// Контейнер Init для копирования конфигурации (log4j2.xml и nifi-registry.properties)
	initConfigCopyContainer := corev1.Container{
		Name:    "init-config-copy",
		Image:   "busybox",
		Command: []string{"sh", "-c", "cp /config-source/* /opt/nifi-registry/nifi-registry-current/conf/"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config-volume",
				MountPath: "/config-source",
			},
			{
				Name:      "config-target-volume",
				MountPath: "/opt/nifi-registry/nifi-registry-current/conf",
			},
		},
	}

	// КОНТЕЙНЕР INIT ДЛЯ ПРАВ ДОСТУПА (УСИЛЕННЫЙ)
	// chown -R 1000:1000 /opt/nifi-registry/nifi-registry-current (даем права на всю домашнюю папку)
	// chmod -R 700 /opt/nifi-registry/nifi-registry-current/flow_storage (даем права на PVC)
	initDataChownContainer := corev1.Container{
		Name:    "init-data-chown",
		Image:   "busybox",
		Command: []string{"sh", "-c", fmt.Sprintf("chown -R 1000:1000 /opt/nifi-registry/nifi-registry-current && chmod -R 700 %s", flowStorageMountPath)},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "flow-storage-volume",
				MountPath: flowStorageMountPath,
			},
		},
	}

	// Основной контейнер NiFi Registry
	nifiRegistryContainer := corev1.Container{
		Name:  "nifi-registry",
		Image: nifiRegistry.Spec.Image, // Используем образ из CR
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 18080,
				Name:          "http-port",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config-target-volume",
				MountPath: "/opt/nifi-registry/nifi-registry-current/conf",
			},
			{
				Name:      "flow-storage-volume",
				MountPath: flowStorageMountPath,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "NIFI_REGISTRY_HOME",
				Value: "/opt/nifi-registry/nifi-registry-current",
			},
		},
		Resources: nifiRegistry.Spec.Resources, // <-- ИСПОЛЬЗУЕМ НОВОЕ ПОЛЕ
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
					// Init-контейнеры
					InitContainers: []corev1.Container{
						initConfigCopyContainer,
						initDataChownContainer,
					},
					// Основные контейнеры
					Containers: []corev1.Container{
						nifiRegistryContainer,
					},
					// Volumes (Тома)
					Volumes: []corev1.Volume{
						// ConfigMap Volume
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
						// EmptyDir для записи ConfigMap в папку conf
						{
							Name: "config-target-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						// PVC Volume для хранения потоков
						{
							Name: "flow-storage-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(nifiRegistry, dep, scheme)
	return dep
}
