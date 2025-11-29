// controllers/deployment_helpers.go

package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deploymentForNifiRegistry генерирует Deployment для NiFi Registry
func deploymentForNifiRegistry(nifiRegistry *registryv1.NifiRegistry, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := map[string]string{"app": nifiRegistry.Name, "app.kubernetes.io/name": nifiRegistry.Name}
	replicas := nifiRegistry.Spec.Size
	imageName := fmt.Sprintf("%s:%s", nifiRegistry.Spec.Image.Repository, nifiRegistry.Spec.Image.Tag)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiRegistry.Name,
			Namespace: nifiRegistry.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: func() *int64 { i := int64(1000); return &i }(),
					},
					InitContainers: createInitContainers(nifiRegistry, imageName),
					Containers:     []corev1.Container{createMainContainer(nifiRegistry, imageName)},
					Volumes:        createVolumes(nifiRegistry),
				},
			},
		},
	}
	ctrl.SetControllerReference(nifiRegistry, dep, scheme)
	return dep
}

// createInitContainers создает init-контейнеры
func createInitContainers(nifiRegistry *registryv1.NifiRegistry, imageName string) []corev1.Container {
	const nifiConfigPath = "/opt/nifi-registry/nifi-registry-current/conf"

	initContainers := []corev1.Container{
		{
			Name:  "init-config-copy",
			Image: imageName,

			Command: []string{
				"sh", "-c",
				// Копируем .properties (включая nifi-registry.properties) и .xml (Keycloak конфиг)
				fmt.Sprintf("cp -LR %s/. /conf-writable/ && cp /config-source/*.properties /conf-writable/ || true && cp /config-source/*.xml /conf-writable/",
					nifiConfigPath),
			},

			VolumeMounts: []corev1.VolumeMount{
				{Name: "config-source", MountPath: "/config-source"},
				{Name: "conf-writable", MountPath: "/conf-writable"},
			},
		},
		{
			Name:  "init-data-chown",
			Image: "busybox:1.36",
			Command: []string{
				"sh", "-c",
				"chown -R 1000:1000 /data-flow && chown -R 1000:1000 /data-ext",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "flow-storage-volume", MountPath: "/data-flow"},
				{Name: "extension-bundles-volume", MountPath: "/data-ext"},
			},
		},
	}

	return initContainers
}

// createEnvVars создает список переменных окружения для основного контейнера.
func createEnvVars(nifiRegistry *registryv1.NifiRegistry) []corev1.EnvVar {
	// Основные настройки (заменяют nifi-registry.properties)
	envVars := []corev1.EnvVar{
		// Web Settings
		{Name: "NIFI_REGISTRY_WEB_HTTP_HOST", Value: "0.0.0.0"},
		{Name: "NIFI_REGISTRY_WEB_HTTP_PORT", Value: "8080"},
		{Name: "NIFI_REGISTRY_WEB_CONTEXT_PATH", Value: "/nifi-registry"},

		// Security Settings: УБРАНО

		// Flow Persistence Provider Settings (Используем настройки по умолчанию для H2)
		{Name: "NIFI_REGISTRY_FLOW_PROVIDER_IMPLEMENTATION_ORG_APACHE_NIFI_REGISTRY_FLOW_KEYVALUE_KEYVALUEFLOWPROVIDER_FLOW_STORAGE_DIRECTORY", Value: "./flow_storage"},

		// Registry Version
		{Name: "NIFI_REGISTRY_VERSION", Value: "1.24.0"},
	}

	return envVars
}

// createMainContainer создает основной контейнер
func createMainContainer(nifiRegistry *registryv1.NifiRegistry, imageName string) corev1.Container {
	volumeMounts := createVolumeMounts()

	// ИСПРАВЛЕНИЕ: Гарантируем, что порты > 0

	// Определяем HTTP порт
	httpPort := int32(8080)
	if nifiRegistry.Spec.Port != 0 {
		httpPort = nifiRegistry.Spec.Port
	}

	containerPorts := []corev1.ContainerPort{
		{ContainerPort: httpPort, Name: "http"},
	}

	if nifiRegistry.Spec.Tls.Enabled {
		tlsPort := int32(8443)
		if nifiRegistry.Spec.Tls.Port != 0 {
			tlsPort = nifiRegistry.Spec.Tls.Port
		}

		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: tlsPort,
			Name:          "https",
		})
	}

	return corev1.Container{
		Image: imageName,
		Name:  "nifi-registry",

		Env: createEnvVars(nifiRegistry),

		Command: []string{
			"/opt/nifi-registry/nifi-registry-current/bin/nifi-registry.sh",
			"run",
			"--foreground",
		},

		Ports: containerPorts, // Используем исправленные порты

		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/nifi-registry",
					Port: intstr.FromInt(int(httpPort)),
				},
			},
			InitialDelaySeconds: 60,
			PeriodSeconds:       10,
			FailureThreshold:    5,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/nifi-registry",
					Port: intstr.FromInt(int(httpPort)),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       5,
			FailureThreshold:    3,
		},
		VolumeMounts: volumeMounts,
	}
}

// createVolumeMounts создает VolumeMounts для контейнера
func createVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: "conf-writable", MountPath: "/opt/nifi-registry/nifi-registry-current/conf"},
		{Name: "logs-volume", MountPath: "/opt/nifi-registry/nifi-registry-current/logs"},
		{Name: "flow-storage-volume", MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage"},
		{Name: "extension-bundles-volume", MountPath: "/opt/nifi-registry/nifi-registry-current/extension_bundles"},
	}
}

// createVolumes создает тома для Pod
func createVolumes(nifiRegistry *registryv1.NifiRegistry) []corev1.Volume {
	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)

	volumes := []corev1.Volume{
		{
			Name: "config-source",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				},
			},
		},
		{Name: "conf-writable", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "logs-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "extension-bundles-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}

	// Flow storage volume
	if nifiRegistry.Spec.FlowStorage.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: "flow-storage-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-flow", nifiRegistry.Name),
				},
			},
		})
	} else {
		volumes = append(volumes, corev1.Volume{
			Name:         "flow-storage-volume",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
	}

	return volumes
}
