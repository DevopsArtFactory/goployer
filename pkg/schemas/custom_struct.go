package schemas

import vegeta "github.com/tsenart/vegeta/lib"

type MetricResult struct {
	URL  string
	Data vegeta.Metrics
}
