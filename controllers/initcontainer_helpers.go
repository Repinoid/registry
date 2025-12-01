// Filename: controllers/initcontainer_helpers.go
// Changes: 1. ИСПРАВЛЕНО: Устранена ошибка компилятора 'DriverDownloadUrl undefined' путем замены переменной на жестко заданный URL драйвера PostgreSQL (postgresql-42.7.3.jar).
//          2. Добавлена корректная обработка символа новой строки (\n) в начале блоков echo для TLS и DB.
// ----------------------------------------------------------------------------------------------------------------

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// initContainersForNifiRegistry создает и возвращает список InitContainers для пода NiFi Registry.
func initContainersForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, confMountDest, confMountPath, externalLibMountPath, libStorageVolumeName string) []corev1.Container {
	var initContainers []corev1.Container

	// 1. Контейнер: Копирует конфигурацию из образа в EmptyDir (делает ее доступной для записи)
	initContainers = append(initContainers, corev1.Container{
		Name:  "copy-conf",
		Image: nifiRegistry.Spec.Image.Repository + ":" + nifiRegistry.Spec.Image.Tag, // Используем основной образ
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf("cp -R /opt/nifi-registry/nifi-registry-current/conf/. %s", confMountDest),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "nifi-registry-conf", // confVolumeName
				MountPath: confMountDest,
			},
		},
	})

	// ************************************************************************************
	// 2. Контейнер: Обновляет nifi-registry.properties с помощью значений TLS и DB
	// ************************************************************************************
	var updateCommand string

	// --- A. Логика TLS ---
	if nifiRegistry.Spec.Tls.Enabled {
		keystorePath := confMountPath + "/tls/keystore.jks" // Путь в рабочем контейнере
		truststorePath := confMountPath + "/tls/truststore.jks"

		// 1. Добавление/Обновление TLS свойств (Keystore/Truststore)
		// Используем echo -e '\n...' для добавления первой строки с новой строки, чтобы избежать синтаксической ошибки
		updateCommand += fmt.Sprintf(`echo -e '\nnifi.registry.security.keystore.path=%s' >> %s/nifi-registry.properties; `, keystorePath, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.security.keystore.password=%s' >> %s/nifi-registry.properties; `, nifiRegistry.Spec.Tls.KeystorePassword, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.security.keystore.type=JKS' >> %s/nifi-registry.properties; `, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.security.truststore.path=%s' >> %s/nifi-registry.properties; `, truststorePath, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.security.truststore.password=%s' >> %s/nifi-registry.properties; `, nifiRegistry.Spec.Tls.TruststorePassword, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.security.truststore.type=JKS' >> %s/nifi-registry.properties; `, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.security.client.auth=REQUIRED' >> %s/nifi-registry.properties; `, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.web.https.host=0.0.0.0' >> %s/nifi-registry.properties; `, confMountDest)
		updateCommand += fmt.Sprintf(`echo 'nifi.registry.web.https.port=%s' >> %s/nifi-registry.properties; `, "8443", confMountDest)

		// 2. Закомментировать HTTP порт
		updateCommand += fmt.Sprintf(`sed -i '/nifi.registry.web.http.port=/c\#nifi.registry.web.http.port=8080' %s/nifi-registry.properties; `, confMountDest)
	}

	// --- B. Логика Database ---
	if nifiRegistry.Spec.Database.Enabled {
		dbUrl := nifiRegistry.Spec.Database.Url
		driverClass := nifiRegistry.Spec.Database.DriverClass
		dbUsername := nifiRegistry.Spec.Database.Username
		dbPassword := nifiRegistry.Spec.Database.Password

		// 1. Добавление/Обновление Flow Persistence (новые свойства, которых нет по умолчанию)
		// Используем echo -e '\n...' для добавления первой строки с новой строки, чтобы избежать синтаксической ошибки
		if !nifiRegistry.Spec.Tls.Enabled {
			// Если TLS не включен, добавляем новую строку здесь
			updateCommand += fmt.Sprintf(`echo -e '\nnifi.registry.flow.provider=org.apache.nifi.flow.sql.SqlFlowProvider' >> %s/nifi-registry.properties; `, confMountDest)
		} else {
			// Если TLS включен, новая строка уже была добавлена в TLS-блоке, начинаем сразу с 'echo'
			updateCommand += fmt.Sprintf(`echo 'nifi.registry.flow.provider=org.apache.nifi.flow.sql.SqlFlowProvider' >> %s/nifi-registry.properties; `, confMountDest)
		}

		updateCommand += fmt.Sprintf(`echo 'nifi.registry.db.implementation=org.apache.nifi.db.sql.SqlFlowPersistenceProvider' >> %s/nifi-registry.properties; `, confMountDest)

		// 2. Обновление существующих свойств БД (Url, Class, User, Password) с помощью sed
		updateCommand += fmt.Sprintf(`sed -i 's|^nifi.registry.db.url=.*$|nifi.registry.db.url=%s|g' %s/nifi-registry.properties; `, dbUrl, confMountDest)
		updateCommand += fmt.Sprintf(`sed -i 's|^nifi.registry.db.driver.class=.*$|nifi.registry.db.driver.class=%s|g' %s/nifi-registry.properties; `, driverClass, confMountDest)
		updateCommand += fmt.Sprintf(`sed -i 's|^nifi.registry.db.username=.*$|nifi.registry.db.username=%s|g' %s/nifi-registry.properties; `, dbUsername, confMountDest)
		updateCommand += fmt.Sprintf(`sed -i 's|^nifi.registry.db.password=.*$|nifi.registry.db.password=%s|g' %s/nifi-registry.properties; `, dbPassword, confMountDest)
	}

	// 3. Создаем Init Container для конфигурации, если есть команды для обновления
	if updateCommand != "" {
		finalCommand := fmt.Sprintf("set -xe; %s", updateCommand)

		initContainers = append(initContainers, corev1.Container{
			Name:  "configure-registry-properties",
			Image: "busybox", // Легкий образ с sh/sed/echo
			Command: []string{
				"sh",
				"-c",
				finalCommand,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "nifi-registry-conf", // confVolumeName
					MountPath: confMountDest,
				},
			},
		})
	}

	// 4. Контейнер: Скачивает драйвер PostgreSQL (если БД включена)
	if nifiRegistry.Spec.Database.Enabled {
		initContainers = append(initContainers, corev1.Container{
			Name:  "download-db-driver",
			Image: "curlimages/curl:latest", // Используем легкий образ с curl для скачивания
			Command: []string{
				"sh",
				"-c",
				// ИСПРАВЛЕНО: Заменена переменная на жестко заданную ссылку
				"curl -sL https://jdbc.postgresql.org/download/postgresql-42.7.3.jar -o " + externalLibMountPath + "/postgresql-jdbc.jar",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      libStorageVolumeName,
					MountPath: externalLibMountPath, // Монтируем PVC Lib Storage
				},
			},
		})
	}

	return initContainers
}
