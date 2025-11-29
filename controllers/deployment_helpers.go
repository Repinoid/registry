// controllers/deployment_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deploymentForNifiRegistry генерирует Deployment для NiFi Registry
func deploymentForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name}

	// Используем поле Size из CRD
	replicas := nifiRegistry.Spec.Size

	// Формируем полный образ из структуры Spec.Image
	fullImage := fmt.Sprintf("%s:%s", nifiRegistry.Spec.Image.Repository, nifiRegistry.Spec.Image.Tag)

	// Основной контейнер NiFi Registry
	nifiRegistryContainer := corev1.Container{
		Name:  "nifi-registry",
		Image: fullImage,
		Ports: []corev1.ContainerPort{
			{
				// Порт берется из Service (18080 или 8443)
				ContainerPort: 18080,
				Name:          "http-port",
			},
		},
		VolumeMounts: nil, // УДАЛЕНО: Больше нет монтирования томов
		Env:          nil, // УДАЛЕНО: Больше нет переменных окружения (NIFI_REGISTRY_HOME)
		Resources:    nifiRegistry.Spec.Resources,
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: nil, // УДАЛЕНО: Init-контейнеры
					Containers: []corev1.Container{
						nifiRegistryContainer,
					},
					Volumes: nil, // УДАЛЕНО: Тома
				},
			},
		},
	}

	ctrl.SetControllerReference(nifiRegistry, dep, scheme)
	return dep
}
