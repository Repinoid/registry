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
// ИСПРАВЛЕНИЕ: Убираем лишние аргументы (string, string, string)
func configMapForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *corev1.ConfigMap {
	labels := map[string]string{"app": nifiRegistry.Name}
	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)

	configMapData := map[string]string{
		"log4j2.xml": `
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
		`,
		// Дополнительные файлы конфигурации, если нужны, можно добавить здесь
		"nifi-registry.properties": `
# NiFi Registry Properties
# Эти настройки будут переопределены переменными окружения в deployment_helpers.go

# Web Properties
nifi.registry.web.http.host=0.0.0.0
nifi.registry.web.http.port=18080
nifi.registry.web.context.path=/nifi-registry

# Flow Persistence Provider
nifi.registry.flow.persistence.provider.implementation=org.apache.nifi.registry.flow.keyvalue.KeyValueFlowPersistenceProvider
nifi.registry.flow.keyvalue.flow.storage.directory=./flow_storage

# Extension Bundle Persistence Provider
nifi.registry.extension.bundle.persistence.provider.implementation=org.apache.nifi.registry.extension.bundle.FileSystemExtensionBundlePersistenceProvider
nifi.registry.extension.bundle.file.system.storage.directory=./extension_bundles

# Database Configuration (если не используется env var)
#nifi.registry.db.url=
#nifi.registry.db.driver.class=
#nifi.registry.db.username=
#nifi.registry.db.password=

# Security
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
