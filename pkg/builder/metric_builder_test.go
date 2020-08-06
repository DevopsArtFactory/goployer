package builder

import (
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/go-test/deep"
	"testing"
)

var (
	TEST_METRIC_YAML_PATH = "../../test/metrics_test.yaml"
)

func TestParseMetricConfig(t *testing.T) {

	input, _ := ParseMetricConfig(false, TEST_METRIC_YAML_PATH)
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
