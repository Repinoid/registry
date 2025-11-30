// controllers/postgresql_helpers.go

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// serviceForPostgreSQL возвращает Service для PostgreSQL.
func serviceForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.Service {
	labels := map[string]string{"app": nifiRegistry.Name + "-postgres"}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name + "-postgres",
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port: 5432,
					Name: "db-port",
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	return svc
}

// deploymentForPostgreSQL возвращает Deployment для PostgreSQL.
func deploymentForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name + "-postgres"}
	replicas := int32(1)
	dbPassword := nifiRegistry.Spec.PostgreSQL.Password

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name + "-postgres",
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
					Containers: []corev1.Container{
						{
							Name:  "postgres",
							Image: nifiRegistry.Spec.PostgreSQL.Image.Repository + ":" + nifiRegistry.Spec.PostgreSQL.Image.Tag,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
									Name:          "db-port",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "POSTGRES_DB",
									Value: nifiRegistry.Spec.PostgreSQL.Database,
								},
								{
									Name:  "POSTGRES_USER",
									Value: nifiRegistry.Spec.PostgreSQL.Username,
								},
								{
									Name:  "POSTGRES_PASSWORD",
									Value: dbPassword,
								},
								{
									Name:  "PGDATA", // <--- ДОБАВЛЕНО: Указываем PGDATA
									Value: "/var/lib/postgresql/data/pgdata",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nifiRegistry.Name + "-postgres-pvc",
									MountPath: "/var/lib/postgresql/data", // <--- ИСПРАВЛЕНО: Монтируем в /var/lib/postgresql/data, но данные будут в подкаталоге pgdata
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: nifiRegistry.Name + "-postgres-pvc",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: nifiRegistry.Name + "-postgres",
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
