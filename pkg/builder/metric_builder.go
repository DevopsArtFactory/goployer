/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package builder

import (
	"os"

	Logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

func ParseMetricConfig(disabledMetrics bool, filename string) (schemas.MetricConfig, error) {
	if disabledMetrics {
		return schemas.MetricConfig{Enabled: false}, nil
	}

	if !tool.CheckFileExists(filename) {
		return schemas.MetricConfig{Enabled: false}, nil
	}

	metricConfig := schemas.MetricConfig{Enabled: true}
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		Logger.Errorf("Error reading YAML file: %s\n", err)
		return metricConfig, err
	}

	err = yaml.Unmarshal(yamlFile, &metricConfig)
	if err != nil {
		Logger.Error(err.Error())
		return metricConfig, err
	}

	if len(metricConfig.Metrics.BaseTimezone) == 0 {
		metricConfig.Metrics.BaseTimezone = "UTC"
	}

	return metricConfig, nil
}
