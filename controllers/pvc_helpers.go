// controllers/pvc_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// pvcForNifiRegistry возвращает PVC для NiFi Registry Flow Storage.
func pvcForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.PersistentVolumeClaim {
	flowStorage := nifiRegistry.Spec.FlowStorage
	labels := map[string]string{"app": nifiRegistry.Name}
	name := fmt.Sprintf("%s-flow-storage", nifiRegistry.Name)

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
					corev1.ResourceStorage: resource.MustParse(flowStorage.StorageSize), // ИСПОЛЬЗУЕМ: StorageSize
				},
			},
		},
	}

	if flowStorage.StorageClassName != "" { // ИСПОЛЬЗУЕМ: StorageClassName
		pvc.Spec.StorageClassName = &flowStorage.StorageClassName
	}

	return pvc
}
