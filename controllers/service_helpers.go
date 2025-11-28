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

// persistentVolumeClaimForFlowStorage генерирует PVC для Flow Storage
func persistentVolumeClaimForFlowStorage(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": nifiRegistry.Name}
	flowSpec := nifiRegistry.Spec.FlowStorage

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-flow", nifiRegistry.Name),
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: flowSpec.Size},
			},
			StorageClassName: &flowSpec.StorageClass,
		},
	}

	if flowSpec.StorageClass == "" {
		pvc.Spec.StorageClassName = nil
	}

	return pvc
}

// serviceForNifiRegistry генерирует Service для NiFi Registry
func serviceForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.Service {
	labels := map[string]string{"app": nifiRegistry.Name, "app.kubernetes.io/name": nifiRegistry.Name}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": nifiRegistry.Name},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "https",
					Port:       nifiRegistry.Spec.Tls.Port,
					TargetPort: intstr.FromInt(int(nifiRegistry.Spec.Tls.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	ctrl.SetControllerReference(nifiRegistry, svc, scheme)
	return svc
}
