// controllers/postgresql_helpers.go

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// serviceForPostgreSQL генерирует Service для PostgreSQL
func serviceForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.Service {
	// Имя сервиса должно совпадать с тем, что мы хардкодили в NIFI_REGISTRY_DB_URL: postgres-service
	name := "postgres-service"
	labels := map[string]string{"app": nifiRegistry.Name, "db": "postgresql"}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels, // Селектор для поиска подов, которые будут запущены Deployment-ом PostgreSQL
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
					Name:       "postgres",
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Устанавливаем владельца
	ctrl.SetControllerReference(nifiRegistry, svc, nil)
	return svc
}
