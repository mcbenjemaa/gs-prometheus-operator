package controllers

import monitoringv1alpha1 "github.com/mcbenjemaa/gs-prometheus-operator/api/v1alpha1"

type PrometheusConfigFile struct {
	ScrapeConfigs []PrometheusScrapeConfig `yaml:"scrape_configs"`
}

type PrometheusScrapeConfig struct {
	JobName string `yaml:"job_name"`

	Scheme          string                   `yaml:"scheme,omitempty"`
	TlsConfig       TLSConfig                `yaml:"tls_config,omitempty"`
	BearerTokenFile string                   `yaml:"bearer_token_file,omitempty"`
	StaticConfigs   []StaticConfig           `yaml:"static_configs,omitempty"`
	FileSdConfigs   []PrometheusFileSdConfig `yaml:"file_sd_configs,omitempty"`
}

type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

type TLSConfig struct {
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}
type PrometheusFileSdConfig struct {
	Files []string `yaml:"files"`
}

func getPrometheusScrapeConfig(s []monitoringv1alpha1.ScrapeConfig) []PrometheusScrapeConfig {
	r := make([]PrometheusScrapeConfig, 0)

	if s != nil {
		for _, i := range s {

			ts := make([]StaticConfig, 0)
			for _, sc := range i.StaticConfigs {
				ts = append(ts, StaticConfig{
					Targets: sc.Targets})
			}
			psc := PrometheusScrapeConfig{
				JobName:         i.JobName,
				Scheme:          i.Scheme,
				TlsConfig:       TLSConfig{InsecureSkipVerify: i.TlsConfig.InsecureSkipVerify},
				BearerTokenFile: i.BearerTokenFile,
				StaticConfigs:   ts,
			}
			r = append(r, psc)
		}
	}

	r = append(r, PrometheusScrapeConfig{
		JobName: "gs",
		FileSdConfigs: []PrometheusFileSdConfig{
			PrometheusFileSdConfig{
				Files: []string{
					"/etc/targets/targets.yaml",
				},
			},
		},
	})

	return r
}
