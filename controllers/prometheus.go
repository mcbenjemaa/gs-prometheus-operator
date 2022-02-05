package controllers

import (
	monitoringv1alpha1 "github.com/mcbenjemaa/gs-prometheus-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func labels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "gs-prometheus",
		"app.kubernetes.io/component": "prometheus",
	}
}

func sidecarContainer() corev1.Container {
	return corev1.Container{
		Name:  "configmap-reload",
		Image: "jimmidyson/configmap-reload:v0.6.1",
		Args: []string{"--volume-dir=/etc/targets",
			"--volume-dir=/etc/config",
			"--webhook-url=http://127.0.0.1:9090/-/reload",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "targets-volume",
				MountPath: "/etc/targets",
				ReadOnly:  true,
			},
			{
				Name:      "config-volume",
				MountPath: "/etc/config/",
				ReadOnly:  true,
			},
		},
	}
}

func prometheusContainer(p *monitoringv1alpha1.Prometheus) corev1.Container {
	return corev1.Container{
		Name:            "prometheus",
		Image:           *p.Spec.Image.Repository + ":" + p.Spec.Image.Version,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{"--config.file=/etc/config/prometheus.yml",
			"--storage.tsdb.path=/data",
			"--web.enable-lifecycle",
		},
		Ports:     []corev1.ContainerPort{corev1.ContainerPort{ContainerPort: 9090}},
		Resources: *p.Spec.Resources.DeepCopy(),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/ready",
					Port: intstr.FromInt(9090),
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      30,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/healthy",
					Port: intstr.FromInt(9090),
				},
			},

			InitialDelaySeconds: 30,
			TimeoutSeconds:      30,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "targets-volume",
				MountPath: "/etc/targets",
			},
			{
				Name:      "config-volume",
				MountPath: "/etc/config/",
			},
			{
				Name:      "gs-prometheus-data",
				MountPath: "/data",
				SubPath:   "",
			},
		},
	}
}

func volumes(n string) []corev1.Volume {
	return []corev1.Volume{
		corev1.Volume{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: n + "-config",
					},
				},
			},
		},
		corev1.Volume{
			Name: "targets-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: n + "-targets",
					},
				},
			},
		},
	}
}

// func affinity(n string) corev1.Affinity {
// 	return  corev1.PodAffinity{
// 		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
// 			corev1.PodAffinityTerm{
// 				LabelSelector: []metav1.LabelSelectorRequirement{
// 					metav1.LabelSelectorRequirement{
// 						Key: "app.kubernetes.io/name",
// 						Operator: metav1.LabelSelectorOpIn,
// 						Values: []string{
// 							"gs-prometheus",
// 						},
// 					},
// 				},
// 				TopologyKey: "kubernetes.io/hostname",
// 			},
// 		},
// 	  }

// }

func desiredStatefulSet(p *monitoringv1alpha1.Prometheus) appsv1.StatefulSet {
	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: p.Name, Namespace: p.Namespace, Labels: labels()},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &p.Spec.Replicas,
			UpdateStrategy:      appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels(),
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{p.Spec.VolumeClaimTemplate},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels(),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						sidecarContainer(),
						prometheusContainer(p),
					},
					Volumes: volumes(p.Name),
					// Affinity:                      affinity(),
				},
			},
		},
	}
}
