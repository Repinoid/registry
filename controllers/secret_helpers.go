// controllers/secret_helpers.go

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// secretForNifiRegistry генерирует Secret для хранения учетных данных БД
func secretForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.Secret {
	dbSpec := nifiRegistry.Spec.Database

	// Если внешняя БД не включена или SecretName не указан, Secret не создается.
	if !dbSpec.Enabled || dbSpec.SecretName == "" {
		return nil
	}

	labels := map[string]string{"app": nifiRegistry.Name}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbSpec.SecretName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			// Ключи должны соответствовать полям, которые мы будем использовать в Deployment.
			// Мы используем username из CRD. Пароль здесь не устанавливаем,
			// так как Secret, скорее всего, существует.
			"username": []byte(dbSpec.Username),
			"password": []byte("placeholder"), // Используем placeholder, чтобы избежать конфликта при создании
		},
	}

	// Secret для БД обычно не удаляется с ресурсом NifiRegistry,
	// так как он может быть общим. Поэтому мы НЕ устанавливаем ControllerReference.
	// Тем не менее, для целей разработки и тестирования мы его временно установим.
	ctrl.SetControllerReference(nifiRegistry, secret, scheme)
	return secret
}
