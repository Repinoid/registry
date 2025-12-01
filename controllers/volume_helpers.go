// Filename: controllers/volume_helpers.go
// Changes: НОВЫЙ ФАЙЛ. Выделение всей логики создания Volumes и VolumeMounts.
// ----------------------------------------------------------------------------------------------------------------

package controllers

import (
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// volumeMountsForNifiRegistry создает и возвращает список VolumeMounts для NiFi Registry.
func volumeMountsForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, flowStorageVolumeName, libStorageVolumeName, confVolumeName, tlsSecretVolumeName, confMountPath, externalLibMountPath string) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}

	// 1. Монтирование flow storage
	if nifiRegistry.Spec.FlowStorage.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      flowStorageVolumeName,
			MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage",
		})
	}

	// 2. Монтирование lib storage (для драйвера PostgreSQL, если БД включена)
	if nifiRegistry.Spec.LibStorage.Enabled || nifiRegistry.Spec.Database.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      libStorageVolumeName,
			MountPath: externalLibMountPath, // Монтируем внешний Lib в изолированный каталог
		})
	}

	// 3. Монтирование тома /conf (EmptyDir) - для доступа Init-контейнеров
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      confVolumeName,
		MountPath: confMountPath,
	})

	// 4. Монтирование TLS Secret (в conf/tls)
	if nifiRegistry.Spec.Tls.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      tlsSecretVolumeName,
			MountPath: confMountPath + "/tls", // Монтируем Secret в поддиректорию conf/tls
			ReadOnly:  true,
		})
	}

	// 5. Монтирование ConfigMaps для Keycloak (в conf)
	if nifiRegistry.Spec.Keycloak.Enabled {
		// providers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      nifiRegistry.Name + "-providers-cm",
			MountPath: confMountPath + "/providers.xml",
			SubPath:   "providers.xml",
			ReadOnly:  true,
		})
		// identity-providers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      nifiRegistry.Name + "-identity-providers-cm",
			MountPath: confMountPath + "/identity-providers.xml",
			SubPath:   "identity-providers.xml",
			ReadOnly:  true,
		})
		// authorizers.xml
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      nifiRegistry.Name + "-authorizers-cm",
			MountPath: confMountPath + "/authorizers.xml",
			SubPath:   "authorizers.xml",
			ReadOnly:  true,
		})
	}

	return volumeMounts
}

// volumesForNifiRegistry создает и возвращает список Volumes для NiFi Registry.
func volumesForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, flowStorageVolumeName, libStorageVolumeName, confVolumeName, tlsSecretVolumeName string) []corev1.Volume {
	volumes := []corev1.Volume{}
	tlsSecretName := nifiRegistry.Name + "-tls-secret"

	// 1. Том для Flow Storage
	if nifiRegistry.Spec.FlowStorage.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: flowStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: nifiRegistry.Name + "-flow",
				},
			},
		})
	}

	// 2. Том для Lib Storage
	if nifiRegistry.Spec.LibStorage.Enabled || nifiRegistry.Spec.Database.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: libStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: nifiRegistry.Name + "-lib",
				},
			},
		})
	}

	// 3. Том EmptyDir для конфигурации (для доступа на запись InitContainers)
	volumes = append(volumes, corev1.Volume{
		Name: confVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// 4. Том для Secret (TLS Keystore/Truststore)
	if nifiRegistry.Spec.Tls.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: tlsSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsSecretName, // Используем имя Secret по умолчанию
					Items: []corev1.KeyToPath{
						{
							Key:  "keystore.jks",
							Path: "keystore.jks", // Монтируется в conf/tls/keystore.jks
						},
						{
							Key:  "truststore.jks",
							Path: "truststore.jks", // Монтируется в conf/tls/truststore.jks
						},
					},
				},
			},
		})
	}

	// 4.1. Том для Secret (Keycloak Client Secret)
	if nifiRegistry.Spec.Keycloak.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Spec.Keycloak.ClientSecretName, // Используем имя Secret из CRD
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: nifiRegistry.Spec.Keycloak.ClientSecretName,
				},
			},
		})
	}

	// 5. Тома для ConfigMaps (Keycloak/OIDC)
	if nifiRegistry.Spec.Keycloak.Enabled {
		// providers.xml
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Name + "-providers-cm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-providers-cm",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "providers.xml",
							Path: "providers.xml", // Монтируется в /conf/providers.xml
						},
					},
				},
			},
		})
		// identity-providers.xml
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Name + "-identity-providers-cm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-identity-providers-cm",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "identity-providers.xml",
							Path: "identity-providers.xml", // Монтируется в /conf/identity-providers.xml
						},
					},
				},
			},
		})
		// authorizers.xml
		volumes = append(volumes, corev1.Volume{
			Name: nifiRegistry.Name + "-authorizers-cm",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nifiRegistry.Name + "-authorizers-cm",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "authorizers.xml",
							Path: "authorizers.xml", // Монтируется в /conf/authorizers.xml
						},
					},
				},
			},
		})
	}

	return volumes
}
