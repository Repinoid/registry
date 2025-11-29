// controllers/pvc_helpers.go

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// pvcForPostgreSQL возвращает PVC для PostgreSQL.
func pvcForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": nifiRegistry.Name + "-postgres"}
	storageClass := nifiRegistry.Spec.PostgreSQL.StorageClass
	storageSize := resource.MustParse(nifiRegistry.Spec.PostgreSQL.Size)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name + "-postgres",
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	return pvc
}

// pvcForNifiRegistry возвращает PVC для Flow Storage NiFi Registry.
func pvcForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": nifiRegistry.Name}
	storageClass := nifiRegistry.Spec.FlowStorage.StorageClass 
	storageSize := resource.MustParse(nifiRegistry.Spec.FlowStorage.Size) 

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name + "-flow",
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	return pvc
}

// pvcForNifiRegistryLib возвращает PVC для lib NiFi Registry.
func pvcForNifiRegistryLib(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": nifiRegistry.Name}
	storageClass := nifiRegistry.Spec.FlowStorage.StorageClass 
	
	// Используем размер FlowStorage, так как отдельного поля для lib PVC нет.
	storageSize := resource.MustParse(nifiRegistry.Spec.FlowStorage.Size) 

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name + "-lib",
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	return pvc
}
