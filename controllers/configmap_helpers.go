// # Filename: controllers/configmap_helpers.go
// # Changes: Added the corrected providers.xml content to the ConfigMap data to switch flow persistence
// # to DatabaseFlowPersistenceProvider, resolving the H2 driver conflict.

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Константа, содержащая ИСПРАВЛЕННОЕ содержимое providers.xml
const registryProvidersXml = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<providers>
	<flowPersistenceProvider>
		<class>org.apache.nifi.registry.provider.flow.DatabaseFlowPersistenceProvider</class>
	</flowPersistenceProvider>

	<extensionBundlePersistenceProvider>
		<class>org.apache.nifi.registry.provider.extension.FileSystemBundlePersistenceProvider</class>
		<property name="Extension Bundle Storage Directory">./extension_bundles</property>
	</extensionBundlePersistenceProvider>
	</providers>`

// configMapForNifiRegistry генерирует ConfigMap для NiFi Registry
func configMapForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}
	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"providers.xml": registryProvidersXml,
		},
	}

	ctrl.SetControllerReference(nifiRegistry, cm, scheme)
	return cm
}
