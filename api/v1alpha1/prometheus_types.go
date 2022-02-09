/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusSpec defines the desired state of Prometheus
type PrometheusSpec struct {

	// Image represent the spec of Prometheus image/version
	Image ImageSpec `json:"image"`

	// Replica number of replicas to run
	Replicas int32 `json:"replicas"`

	// Compute Resources for Prometheus.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// VolumeClaimTemplate the claim that Prometheus reference.
	// +immutable
	VolumeClaimTemplate corev1.PersistentVolumeClaim `json:"volumeClaimTemplate"`

	// Targets Prometheus scraping targets
	// +optional
	Targets []PrometheusTarget `json:"targets,omitempty"`

	// AdditionalScrapeConfigs Prometheus scraping configs
	// +optional
	AdditionalScrapeConfig []ScrapeConfig `json:"additionalScrapeConfigs,omitempty"`
}

type ImageSpec struct {

	// +optional
	// +kubebuilder:default=prom/prometheus
	Repository *string `json:"repository,omitempty"`

	// Version of Prometheus
	Version string `json:"version"`
}

// Prometheus defines the spec of Prometheus targets
type PrometheusTarget struct {
	Targets []string `json:"targets,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

// ScrapeConfig
type ScrapeConfig struct {
	JobName string `json:"jobName"`

	// +optional
	Scheme string `json:"scheme,omitempty"`
	// +optional
	TlsConfig TLSConfig `json:"tlsConfig,omitempty"`
	// +optional
	BearerTokenFile string `json:"bearerTokenFile,omitempty"`

	StaticConfigs []StaticConfig `json:"staticConfigs"`
}

type StaticConfig struct {
	Targets []string `json:"targets"`
}

type TLSConfig struct {
	// +kubebuilder:default=true
	InsecureSkipVerify bool `json:"insecureSkipVerify"`
}

// PrometheusStatus defines the observed state of Prometheus
type PrometheusStatus struct {

	// ReadyReplicas number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas",description="Total number of ready instances."
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Prometheus"

// Prometheus is the Schema for the prometheuses API
type Prometheus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrometheusSpec   `json:"spec,omitempty"`
	Status PrometheusStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PrometheusList contains a list of Prometheus
type PrometheusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Prometheus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Prometheus{}, &PrometheusList{})
}
