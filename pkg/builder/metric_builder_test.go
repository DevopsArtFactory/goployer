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
	"testing"

	"github.com/go-test/deep"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

func TestParseMetricConfig(t *testing.T) {
	input, _ := ParseMetricConfig(false, constants.TestMetricYamlPath)
	expected := schemas.MetricConfig{
		Enabled: true,
		Region:  "ap-northeast-2",
		Storage: schemas.Storage{
			Type: "dynamodb",
			Name: "goployer-metrics-test",
		},
		Metrics: schemas.Metrics{BaseTimezone: "UTC"},
	}

	if diff := deep.Equal(input, expected); diff != nil {
		t.Error(diff)
	}
}
