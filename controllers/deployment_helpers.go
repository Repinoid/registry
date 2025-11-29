// controllers/deployment_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deploymentForNifiRegistry генерирует Deployment для NiFi Registry.
func deploymentForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name}
	name := nifiRegistry.Name

	// 1. Volumes
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "conf",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "state",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		// Добавляем PVC для FlowStorage
		{
			Name: "flow-storage-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-flow", nifiRegistry.Name),
				},
			},
		},
	}

	// 2. Volume Mounts
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/opt/nifi-registry/nifi-registry-current/conf",
		},
		{
			Name:      "conf",
			MountPath: "/opt/nifi-registry/nifi-registry-current/conf.bak",
		},
		{
			Name:      "state",
			MountPath: "/opt/nifi-registry/nifi-registry-current/state",
		},
		{
			Name:      "flow-storage-volume",
			MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage",
		},
	}

	// 3. Init Containers
	initContainers := []corev1.Container{
		// Init-контейнер для chown (как в стандартном образе)
		{
			Name:  "init-data-chown",
			// Используем busybox, так как в nifi-registry может не быть нужных утилит
			Image: "busybox",
			// Команда sh -c должна быть корректной для busybox
			Command: []string{"sh", "-c", "chown -R 1000:1000 /opt/nifi-registry/nifi-registry-current/flow_storage"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "flow-storage-volume",
					MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage",
				},
			},
		},
	}

	// 4. Environment Variables
	envVars := []corev1.EnvVar{
		{
			Name:  "NIFI_REGISTRY_WEB_HTTP_PORT",
			Value: "18080",
		},
		{ // ИСПРАВЛЕНИЕ: ПРИВЯЗКА К 0.0.0.0
			Name:  "NIFI_REGISTRY_WEB_HTTP_HOST",
			Value: "0.0.0.0",
		},
	}

	// 5. Deployment Spec
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &nifiRegistry.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					// Добавляем Init Containers
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  "nifi-registry",
							Image: nifiRegistry.Spec.Image.Repository + ":" + nifiRegistry.Spec.Image.Tag,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 18080,
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: nifiRegistry.Spec.Resources,
							VolumeMounts: volumeMounts,
							Env:          envVars,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/nifi-registry/",
										Port:   intstr.FromInt(18080),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 20,
								TimeoutSeconds:      1,
								PeriodSeconds:       5,
								FailureThreshold:    6,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/nifi-registry/",
										Port:   intstr.FromInt(18080),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								FailureThreshold:    3,
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	// Устанавливаем владельца
	ctrl.SetControllerReference(nifiRegistry, deployment, scheme)
	return deployment
}
