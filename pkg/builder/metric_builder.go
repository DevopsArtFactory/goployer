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

func ParseMetricConfig(disabledMetrics bool, filename string) (MetricConfig, error) {
	if disabledMetrics {
		return MetricConfig{Enabled: false}, nil
	}

	metricConfig := MetricConfig{Enabled: true}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		Logger.Errorf("Error reading YAML file: %s\n", err)
		return metricConfig, err
	}

	err = yaml.Unmarshal(yamlFile, &metricConfig)
	if err != nil {
		Logger.Errorf(err.Error())
		return metricConfig, err
	}

	if len(metricConfig.Metrics.BaseTimezone) <= 0 {
		metricConfig.Metrics.BaseTimezone = "UTC"
	}

	return metricConfig, nil
}
