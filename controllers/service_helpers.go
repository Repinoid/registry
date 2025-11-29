// controllers/service_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// serviceForNifiRegistry генерирует Service для NiFi Registry
func serviceForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.Service {
	labels := map[string]string{"app": nifiRegistry.Name}

	// ИСПРАВЛЕНИЕ: Используем уникальное имя Service с суффиксом
	serviceName := fmt.Sprintf("%s-service", nifiRegistry.Name)

	// ИСПРАВЛЕНИЕ: Гарантируем, что порт > 0
	httpPort := int32(8080)
	if nifiRegistry.Spec.Port != 0 {
		httpPort = nifiRegistry.Spec.Port
	}

	servicePorts := []corev1.ServicePort{
		{
			Port:       httpPort,
			TargetPort: intstr.FromInt(int(httpPort)),
			Protocol:   corev1.ProtocolTCP,
			Name:       "http",
		},
	}

	// Добавляем порт HTTPS, если TLS включен
	if nifiRegistry.Spec.Tls.Enabled {
		tlsPort := int32(8443)
		if nifiRegistry.Spec.Tls.Port != 0 {
			tlsPort = nifiRegistry.Spec.Tls.Port
		}
		servicePorts = append(servicePorts, corev1.ServicePort{
			Port:       tlsPort,
			TargetPort: intstr.FromInt(int(tlsPort)),
			Protocol:   corev1.ProtocolTCP,
			Name:       "https",
		})
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName, // ИСПРАВЛЕНО
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    servicePorts,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	ctrl.SetControllerReference(nifiRegistry, svc, scheme)
	return svc
}
