// controllers/postgresql_helpers.go

package controllers

import (
	"fmt"
	
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	PostgresServiceName = "postgres-service"
	PostgresPort        = 5432
)

// pvcForPostgreSQL генерирует PersistentVolumeClaim для PostgreSQL data
func pvcForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	name := fmt.Sprintf("%s-postgres-data", nifiRegistry.Name)
	labels := map[string]string{"app": nifiRegistry.Name, "db": "postgresql"}

	// ВАЖНО: Мы используем ресурсный объект resource.Quantity, который требует соответствующего импорта (k8s.io/apimachinery/pkg/api/resource)
	size, err := resource.ParseQuantity(nifiRegistry.Spec.PostgreSQL.Size)
	if err != nil {
		size = resource.MustParse("1Gi") // Значение по умолчанию
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}

	if nifiRegistry.Spec.PostgreSQL.StorageClass != "" {
		pvc.Spec.StorageClassName = &nifiRegistry.Spec.PostgreSQL.StorageClass
	}

	ctrl.SetControllerReference(nifiRegistry, pvc, nil)
	return pvc
}

// serviceForPostgreSQL генерирует Service для PostgreSQL
func serviceForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.Service {
	// Имя сервиса должно совпадать с тем, что мы хардкодили в NIFI_REGISTRY_DB_URL
	name := PostgresServiceName 
	labels := map[string]string{"app": nifiRegistry.Name, "db": "postgresql"}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels, 
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     PostgresPort,
					TargetPort: intstr.FromInt(PostgresPort),
					Name:     "postgres",
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	ctrl.SetControllerReference(nifiRegistry, svc, nil)
	return svc
}

// deploymentForPostgreSQL генерирует Deployment для PostgreSQL
func deploymentForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name, "db": "postgresql"}
	replicas := int32(1)
	
	dbSpec := nifiRegistry.Spec.Database
	postgresImage := nifiRegistry.Spec.PostgreSQL.Image
	
	// Определяем переменные окружения для PostgreSQL
	envVars := []corev1.EnvVar{
		{
			Name: "POSTGRES_USER",
			Value: dbSpec.Username,
		},
		{
			Name: "POSTGRES_DB",
			// Название БД берем "nifiregistry", что соответствует URL подключения NiFi Registry
			Value: "nifiregistry", 
		},
	}
	
	// Пароль берем из Secret
	if dbSpec.SecretName != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "POSTGRES_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dbSpec.SecretName,
					},
					Key: "password", // Используем ключ "password" из Secret
				},
			},
		})
	}
	
	// Контейнер PostgreSQL
	postgresContainer := corev1.Container{
		Name:  "postgresql",
		Image: postgresImage,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: PostgresPort,
				Name:          "postgres",
			},
		},
		Env: envVars,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "postgres-data",
				MountPath: "/var/lib/postgresql/data",
			},
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-postgresql", nifiRegistry.Name),
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
						postgresContainer,
					},
					Volumes: []corev1.Volume{
						{
							Name: "postgres-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-postgres-data", nifiRegistry.Name),
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(nifiRegistry, dep, nil)
	return dep
}
