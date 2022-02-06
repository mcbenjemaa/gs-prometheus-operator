package controllers

type PrometheusConfigFile struct {
	ScrapeConfigs []PrometheusScrapeConfig `yaml:"scrape_configs"`
}

type PrometheusScrapeConfig struct {
	JobName       string                   `yaml:"job_name"`
	FileSdConfigs []PrometheusFileSdConfig `yaml:"file_sd_configs"`
}

type PrometheusFileSdConfig struct {
	Files []string `yaml:"files"`
}
