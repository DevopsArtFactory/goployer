package builder

import (
	Logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

var (
	METRIC_YAML_PATH            = "metrics.yaml"
	DEFAULT_METRIC_STORAGE_TYPE = "dynamodb"
)

type MetricBuilder struct {
	MetricConfig MetricConfig
}

type MetricConfig struct {
	Enabled bool
	Region  string  `yaml:"region"`
	Storage Storage `yaml:"storage"`
}

type Storage struct {
	Type string `yaml:"type"`
	Name string `yaml:"name"`
}

func ParseMetricConfig(disabledMetrics bool) (MetricConfig, error) {
	if disabledMetrics {
		return MetricConfig{Enabled: false}, nil
	}

	metricConfig := MetricConfig{Enabled: true}
	yamlFile, err := ioutil.ReadFile(METRIC_YAML_PATH)
	if err != nil {
		Logger.Errorf("Error reading YAML file: %s\n", err)
		return metricConfig, err
	}

	err = yaml.Unmarshal(yamlFile, &metricConfig)
	if err != nil {
		Logger.Errorf(err.Error())
		return metricConfig, err
	}

	return metricConfig, nil
}
