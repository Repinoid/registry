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
					InitContainers: createInitContainers(imageName),
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
func createInitContainers(imageName string) []corev1.Container {
	const nifiConfigPath = "/opt/nifi-registry/nifi-registry-current/conf"

	return []corev1.Container{
		{
			Name:  "init-config-copy",
			Image: imageName,
			Command: []string{
				"sh", "-c",
				fmt.Sprintf("cp -LR %s/. /conf-writable/ && cp /config-source/*.properties /conf-writable/ && cp /config-source/*.xml /conf-writable/",
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
				"chown -R 1000:1000 /data-flow && chown -R 1000:1000 /data-db && chown -R 1000:1000 /data-ext",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "flow-storage-volume", MountPath: "/data-flow"},
				{Name: "database-volume", MountPath: "/data-db"},
				{Name: "extension-bundles-volume", MountPath: "/data-ext"},
			},
		},
	}
}

// createMainContainer создает основной контейнер
func createMainContainer(nifiRegistry *registryv1.NifiRegistry, imageName string) corev1.Container {
	return corev1.Container{
		Image: imageName,
		Name:  "nifi-registry",

		Command: []string{
			"/opt/nifi-registry/nifi-registry-current/bin/nifi-registry.sh",
			"run",
			"--foreground",
		},

		Ports: []corev1.ContainerPort{
			{ContainerPort: 8080, Name: "http"},
			{ContainerPort: nifiRegistry.Spec.Tls.Port, Name: "https"},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/nifi-registry",
					Port: intstr.FromInt(8080),
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
					Port: intstr.FromInt(8080),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       5,
			FailureThreshold:    3,
		},
		VolumeMounts: createVolumeMounts(),
	}
}

// createVolumeMounts создает VolumeMounts для контейнера
func createVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: "conf-writable", MountPath: "/opt/nifi-registry/nifi-registry-current/conf"},
		{Name: "logs-volume", MountPath: "/opt/nifi-registry/nifi-registry-current/logs"},
		{Name: "flow-storage-volume", MountPath: "/opt/nifi-registry/nifi-registry-current/flow_storage"},
		{Name: "database-volume", MountPath: "/opt/nifi-registry/nifi-registry-current/database"},
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
		{Name: "database-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
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
