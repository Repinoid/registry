// controllers/pvc_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// pvcForNifiRegistry генерирует PVC для NiFi Registry Flow Storage
func pvcForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": nifiRegistry.Name}
	pvcName := fmt.Sprintf("%s-flow", nifiRegistry.Name)
	// Используем 'standard', как было ранее
	storageClassName := "standard"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClassName,
			// ИСПРАВЛЕНО: используем corev1.ResourceRequirements
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: nifiRegistry.Spec.FlowStorage.Size,
				},
			},
		},
	}

	ctrl.SetControllerReference(nifiRegistry, pvc, scheme)
	return pvc
}
