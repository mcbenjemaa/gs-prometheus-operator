package controllers

import (
	"fmt"

	monitoringv1alpha1 "github.com/mcbenjemaa/gs-prometheus-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	prometheusPort                   = 9090
	prometheusConfigMapTargetsSuffix = "-targets"
	prometheusConfigMapSuffix        = "-config"
	prometheusConfig                 = `
		scrape_configs:
		 - job_name: 'gs'
		   file_sd_configs:
			 - files:
			    - /etc/targets/targets.yaml
	`
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
		Ports:     []corev1.ContainerPort{{ContainerPort: 9090}},
		Resources: *p.Spec.Resources.DeepCopy(),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/ready",
					Port: intstr.FromInt(prometheusPort),
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      30,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/healthy",
					Port: intstr.FromInt(prometheusPort),
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
				Name:      p.Name,
				MountPath: "/data",
				SubPath:   "",
			},
		},
	}
}

func volumes(n string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: n + prometheusConfigMapSuffix,
					},
				},
			},
		},
		{
			Name: "targets-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: n + prometheusConfigMapTargetsSuffix,
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

func volumeClaimTemplate(p *monitoringv1alpha1.Prometheus) corev1.PersistentVolumeClaim {
	if p.Spec.VolumeClaimTemplate.ObjectMeta.Name == "" {
		p.Spec.VolumeClaimTemplate.ObjectMeta = metav1.ObjectMeta{
			Name:   p.Name,
			Labels: labels(),
		}
	}
	return p.Spec.VolumeClaimTemplate
}

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
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{volumeClaimTemplate(p)},
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

func desiredService(p *monitoringv1alpha1.Prometheus) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: p.Name, Namespace: p.Namespace, Labels: labels()},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:       "http",
					Port:       prometheusPort,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(prometheusPort),
				},
			},
			SessionAffinity: corev1.ServiceAffinityClientIP,
			Selector:        labels(),
		},
	}
}

func desiredPrometheusConfigMap(p *monitoringv1alpha1.Prometheus) (corev1.ConfigMap, error) {

	cfg := PrometheusConfigFile{
		ScrapeConfigs: []PrometheusScrapeConfig{
			PrometheusScrapeConfig{
				JobName: "gs",
				FileSdConfigs: []PrometheusFileSdConfig{
					PrometheusFileSdConfig{
						Files: []string{
							"/etc/targets/targets.yaml",
						},
					},
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(&cfg)

	if err != nil {
		return corev1.ConfigMap{}, fmt.Errorf("Error while Marshaling. %v", err)
	}

	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: p.Name + prometheusConfigMapSuffix, Namespace: p.Namespace, Labels: labels()},
		Data: map[string]string{
			"prometheus.yml": string(yamlData),
		},
	}, nil
}

func desiredTargetsConfigMap(p *monitoringv1alpha1.Prometheus) (corev1.ConfigMap, error) {

	jsonStr, err := yaml.Marshal(p.Spec.Targets)
	if err != nil {
		return corev1.ConfigMap{}, fmt.Errorf("unable to Marshal 'targets', %v", err)
	}
	data := map[string]string{
		"targets.yaml": string(jsonStr),
	}

	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: p.Name + prometheusConfigMapTargetsSuffix, Namespace: p.Namespace, Labels: labels()},
		Data:       data,
	}, nil
}
