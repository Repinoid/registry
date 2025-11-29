// controllers/deployment_helpers.go

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// deploymentForNifiRegistry возвращает Deployment для NiFi Registry.
func deploymentForNifiRegistry(nifiRegistry *registryv1.NifiRegistry) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name}
	replicas := int32(nifiRegistry.Spec.Size)

	// Имя секрета для пароля базы данных
	dbSecretName := nifiRegistry.Spec.Database.SecretName

	// Объем для данных NiFi Registry (flow storage)
	flowStorageVolumeName := "nifi-registry-flow-storage"

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
					InitContainers: []corev1.Container{
						{
							Name:  "copy-postgres-driver",
							Image: "curlimages/curl:latest",
							Command: []string{
								"sh",
								"-c",
								"curl -sL https://jdbc.postgresql.org/download/postgresql-42.7.3.jar -o /opt/nifi-registry/nifi-registry-current/lib/postgresql-jdbc.jar",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      flowStorageVolumeName,
									MountPath: "/opt/nifi-registry/nifi-registry-current/lib",
								},
							},
						},
					},

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
							Env: []corev1.EnvVar{
								{
									Name:  "NIFI_REGISTRY_DATABASE_URL",
									Value: nifiRegistry.Spec.Database.Url,
								},
								{
									Name:  "NIFI_REGISTRY_DATABASE_DRIVER_CLASS",
									Value: nifiRegistry.Spec.Database.DriverClass,
								},
								{
									Name:  "NIFI_REGISTRY_DATABASE_DRIVER_LIB_DIR",
									Value: "/opt/nifi-registry/nifi-registry-current/lib",
								},
								{
									Name:  "NIFI_REGISTRY_DATABASE_USERNAME",
									Value: nifiRegistry.Spec.Database.Username,
								},
								{
									Name: "NIFI_REGISTRY_DATABASE_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: dbSecretName,
											},
											Key: "password",
										},
									},
								},
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      flowStorageVolumeName,
									MountPath: "/opt/nifi-registry/nifi-registry-current/lib",
								},
								{
									Name:      flowStorageVolumeName,
									MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage",
									SubPath:   "flow",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: flowStorageVolumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: nifiRegistry.Name + "-flow",
								},
							},
						},
					},
				},
			},
		},
	}

	return dep
}