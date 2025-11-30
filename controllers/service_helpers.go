// Filename: controllers/service_helpers.go
// Changes: REMOVED the redundant function serviceForPostgreSQL, as it is correctly defined in
//          controllers/postgresql_helpers.go, resolving the code duplication issue.

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	// !!! ВОТ ОН, НЕДОСТАЮЩИЙ ИМПОРТ !!!
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// serviceForNifiRegistry возвращает Service для NiFi Registry.
func serviceForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.Service {
	labels := map[string]string{"app": nifiRegistry.Name}

	// Конфигурация порта зависит от TLS
	var svcPort corev1.ServicePort

	if nifiRegistry.Spec.Tls.Enabled { // ПРОВЕРЯЕМ: Tls.Enabled
		svcPort = corev1.ServicePort{
			Name:     "https-port",
			Protocol: corev1.ProtocolTCP,
			Port:     8443,
			TargetPort: intstr.IntOrString{ // ИСПОЛЬЗУЕМ: intstr.IntOrString
				Type:   intstr.Int, // ИСПОЛЬЗУЕМ: intstr.Int
				IntVal: 8443,
			},
		}
	} else {
		svcPort = corev1.ServicePort{
			Name:     "http-port",
			Protocol: corev1.ProtocolTCP,
			Port:     18080,
			TargetPort: intstr.IntOrString{ // ИСПОЛЬЗУЕМ: intstr.IntOrString
				Type:   intstr.Int, // ИСПОЛЬЗУЕМ: intstr.Int
				IntVal: 18080,
			},
		}
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{svcPort},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	return svc
}
