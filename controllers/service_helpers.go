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

	serviceName := fmt.Sprintf("%s-service", nifiRegistry.Name)

	servicePorts := []corev1.ServicePort{}

	// --- ОПРЕДЕЛЕНИЕ ПОРТОВ (Без nifiRegistry.Spec.Port) ---

	if nifiRegistry.Spec.Tls.Enabled {
		// Если TLS включен, используем порт из TlsSpec для HTTPS
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

	} else {
		// Если TLS выключен, используем порт по умолчанию 18080 для HTTP
		httpPort := int32(18080)

		servicePorts = append(servicePorts, corev1.ServicePort{
			Port:       httpPort,
			TargetPort: intstr.FromInt(int(httpPort)),
			Protocol:   corev1.ProtocolTCP,
			Name:       "http",
		})
	}

	// --- Создание Service ---

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
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
