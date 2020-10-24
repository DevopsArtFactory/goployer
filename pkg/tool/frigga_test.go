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

package tool

import "testing"

func TestParseTargetGroupVersion(t *testing.T) {
	testData := []struct {
		Input    string
		Expected int
	}{
		{
			Input:    "test_dev-useast1-canary-v000",
			Expected: 0,
		},
		{
			Input:    "test_dev-useast1-canary-v001",
			Expected: 1,
		},
		{
			Input:    "test_dev-useast1-canary-v012",
			Expected: 12,
		},
		{
			Input:    "test_dev-useast1-canary-v100",
			Expected: 100,
		},
	}

	for _, td := range testData {
		if output := ParseTargetGroupVersion(td.Input); output != td.Expected {
			t.Errorf("expected: %d, output: %d", td.Expected, output)
		}
	}
}
