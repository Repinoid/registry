// controllers/pvc_helpers.go

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// pvcForPostgreSQL возвращает PVC для базы данных PostgreSQL.
func pvcForPostgreSQL(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	name := nifiRegistry.Name + "-postgres"
	labels := map[string]string{"app": name}
	size := nifiRegistry.Spec.PostgreSQL.Size
	storageClass := nifiRegistry.Spec.PostgreSQL.StorageClass

	if size == "" {
		size = "1Gi" // Размер по умолчанию
	}

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
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
			StorageClassName: &storageClass,
		},
	}
	return pvc
}

// pvcForFlowStorage возвращает PVC для каталога flows NiFi Registry.
func pvcForFlowStorage(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	name := nifiRegistry.Name + "-flow"
	labels := map[string]string{"app": nifiRegistry.Name + "-registry"}
	size := nifiRegistry.Spec.FlowStorage.Size
	storageClass := nifiRegistry.Spec.FlowStorage.StorageClass

	if size == "" {
		size = "1Gi" // Размер по умолчанию
	}

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
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
			StorageClassName: &storageClass,
		},
	}
	return pvc
}

// pvcForLibStorage возвращает PVC для каталога lib NiFi Registry. <--- ДОБАВЛЕНО
func pvcForLibStorage(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {
	name := nifiRegistry.Name + "-lib"
	labels := map[string]string{"app": nifiRegistry.Name + "-registry"}
	size := nifiRegistry.Spec.LibStorage.Size
	storageClass := nifiRegistry.Spec.LibStorage.StorageClass

	if size == "" {
		size = "500Mi" // Размер по умолчанию для lib
	}

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
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
			StorageClassName: &storageClass,
		},
	}
	return pvc
}
