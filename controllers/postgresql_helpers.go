// controllers/postgresql_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// ДОБАВЛЕН ИМПОРТ intstr
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	postgresName = "postgresql"
)

// deploymentForPostgreSQL возвращает Deployment для встроенного PostgreSQL.
func deploymentForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name, "component": postgresName}
	replicas := int32(1)

	// Пароль для PostgreSQL (используем значение из spec.database.secretName)
	pgPassword := nifiRegistry.Spec.Database.SecretName

	envVars := []corev1.EnvVar{
		{
			Name:  "POSTGRES_USER",
			Value: nifiRegistry.Spec.Database.Username,
		},
		{
			Name:  "POSTGRES_PASSWORD",
			Value: pgPassword,
		},
		{
			Name:  "POSTGRES_DB",
			Value: "nifiregistry",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", nifiRegistry.Name, postgresName),
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
							Name:  postgresName,
							Image: nifiRegistry.Spec.PostgreSQL.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
									Name:          "tcp-port",
								},
							},
							Env: envVars,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf("%s-postgres-data", nifiRegistry.Name),
									MountPath: "/var/lib/postgresql/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: fmt.Sprintf("%s-postgres-data", nifiRegistry.Name),
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-postgres-data", nifiRegistry.Name),
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

// serviceForPostgreSQL возвращает Service для встроенного PostgreSQL.
func serviceForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.Service {
	labels := map[string]string{"app": nifiRegistry.Name, "component": postgresName}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-service", // Статичное имя для подключения NiFi Registry
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp-port",
					Protocol: corev1.ProtocolTCP,
					Port:     5432,
					// ИСПРАВЛЕНО: используем intstr для IntOrString и Int
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 5432,
					},
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return svc
}

// pvcForPostgreSQL возвращает PVC для данных PostgreSQL.
func pvcForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": nifiRegistry.Name, "component": postgresName}
	name := fmt.Sprintf("%s-postgres-data", nifiRegistry.Name)
	postgresSpec := nifiRegistry.Spec.PostgreSQL

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(postgresSpec.Size),
				},
			},
		},
	}

	// Используем StorageClass, который мы только что вернули в types.go
	if postgresSpec.StorageClass != "" {
		pvc.Spec.StorageClassName = &postgresSpec.StorageClass
	}

	return pvc
}
