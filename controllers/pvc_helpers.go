// controllers/pvc_helpers.go

package controllers

import (
	"fmt"
	"os"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// persistentVolumeClaimForFlowStorage генерирует PVC для хранения данных NiFi Registry
func persistentVolumeClaimForFlowStorage(nifiRegistry *registryv1.NifiRegistry) *corev1.PersistentVolumeClaim {

	// Используем значение из CR, которое является resource.Quantity.
	storageSize := nifiRegistry.Spec.FlowStorage.Size

	// Проверяем, если размер равен нулю (не задан в CRD), устанавливаем значение по умолчанию 1Gi
	// Это решает проблему типизации.
	if storageSize.IsZero() {
		defaultSize, err := resource.ParseQuantity("1Gi")
		if err != nil {
			// Это не должно произойти
			fmt.Fprintf(os.Stderr, "FATAL: Could not parse default storage size: %v\n", err)
			storageSize = resource.MustParse("1Gi")
		} else {
			storageSize = defaultSize
		}
	}

	pvcName := fmt.Sprintf("%s-flow", nifiRegistry.Name)

	// StorageClassName УДАЛЕНО: Чтобы избежать ошибки 'undefined', мы исключаем это поле,
	// поскольку его нет в вашей структуре FlowStorageSpec.

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: nifiRegistry.Namespace,
			Labels:    map[string]string{"app": nifiRegistry.Name},
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
			// StorageClassName: nil // Оставляем nil, чтобы использовался класс по умолчанию,
			// или не указываем его явно
		},
	}

	return pvc
}
