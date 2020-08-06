package builder

import (
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

var (
	METRIC_YAML_PATH            = "metrics.yaml"
	DEFAULT_METRIC_STORAGE_TYPE = "dynamodb"
)

func ParseMetricConfig(disabledMetrics bool, filename string) (schemas.MetricConfig, error) {
	if disabledMetrics {
		return schemas.MetricConfig{Enabled: false}, nil
	}

	if ! tool.FileExists(filename) {
		return schemas.MetricConfig{Enabled: false}, nil
	}

	metricConfig := schemas.MetricConfig{Enabled: true}
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
