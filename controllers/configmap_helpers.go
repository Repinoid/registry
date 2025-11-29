// controllers/configmap_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// configMapForNifiRegistry генерирует ConfigMap для NiFi Registry
func configMapForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}
	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)

	configMapData := map[string]string{
		// 1. log4j2.xml (Корректный минимальный XML)
		"log4j2.xml": `<?xml version="1.0" encoding="UTF-8"?>
<Configuration status="WARN" name="NiFiRegistry" packages="org.apache.nifi.registry.util">
    <Appenders>
        <Console name="STDOUT" target="SYSTEM_OUT">
            <PatternLayout pattern="%d{yyyy-MM-dd HH:mm:ss,SSS} %-5p [%t] %c %M (%L) - %m%n"/>
        </Console>
    </Appenders>
    <Loggers>
        <Root level="INFO">
            <AppenderRef ref="STDOUT"/>
        </Root>
    </Loggers>
</Configuration>
		`,
		// 2. nifi-registry.properties (Минимальная рабочая конфигурация)
		"nifi-registry.properties": `
# NiFi Registry Properties
nifi.registry.web.http.host=0.0.0.0
nifi.registry.web.http.port=18080
nifi.registry.web.context.path=/nifi-registry

# Flow Persistence Provider: Используем встроенное KeyValue хранилище
nifi.registry.flow.persistence.provider.implementation=org.apache.nifi.registry.flow.keyvalue.KeyValueFlowPersistenceProvider
nifi.registry.flow.keyvalue.flow.storage.directory=./flow_storage

# Extension Bundle Persistence Provider
nifi.registry.extension.bundle.persistence.provider.implementation=org.apache.nifi.registry.extension.bundle.FileSystemExtensionBundlePersistenceProvider
nifi.registry.extension.bundle.file.system.storage.directory=./extension_bundles

# Database Configuration (Отключено - используем Derby)
#nifi.registry.db.url=...

# Security (Отключено)
#nifi.registry.security.user.login.identity.provider=
`,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Data: configMapData,
	}

	ctrl.SetControllerReference(nifiRegistry, cm, scheme)
	return cm
}
