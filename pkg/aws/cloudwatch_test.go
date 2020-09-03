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
