package aws

import (
	"testing"
	"time"
)

func TestCheckMetricTimeValidation(t *testing.T) {
	baseTime := time.Now()
	testData := []map[string]interface{}{
		{
			"start":    baseTime.Add(-1 * time.Second),
			"end":      baseTime.Add(1 * time.Second),
			"expected": true,
		},
		{
			"start":    baseTime.Add(-1 * time.Second),
			"end":      baseTime.Add(-2 * time.Second),
			"expected": false,
		},
		{
			"start":    baseTime.Add(1 * time.Second),
			"end":      baseTime.Add(1 * time.Second),
			"expected": false,
		},
	}

	for _, td := range testData {
		if CheckMetricTimeValidation(td["start"].(time.Time), td["end"].(time.Time)) != td["expected"].(bool) {
			t.Error("CheckMetricTimeValidation failed")
		}
	}
}
