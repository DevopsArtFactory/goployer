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

package runner

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

func TestFilterS3Path(t *testing.T) {
	path := "s3://goployer/test.yaml"
	type TestData struct {
		Bucket   string
		FilePath string
	}
	b, f := FilterS3Path(path)
	input := TestData{
		Bucket:   b,
		FilePath: f,
	}

	expected := TestData{
		Bucket:   "goployer",
		FilePath: "test.yaml",
	}

	if diff := deep.Equal(input, expected); diff != nil {
		t.Error(diff)
	}
}

func TestCheckUpdateInformation(t *testing.T) {
	type TestData struct {
		old      schemas.Capacity
		now      schemas.Capacity
		expected bool
	}
	testData := []TestData{
		{
			old:      makeCapacityStruct(1, 1, 1),
			now:      makeCapacityStruct(2, 1, 1),
			expected: false,
		},
		{
			old:      makeCapacityStruct(1, 1, 1),
			now:      makeCapacityStruct(2, 2, 1),
			expected: false,
		},
		{
			old:      makeCapacityStruct(1, 1, 1),
			now:      makeCapacityStruct(1, 2, 3),
			expected: false,
		},
		{
			old:      makeCapacityStruct(1, 1, 1),
			now:      makeCapacityStruct(1, 2, 2),
			expected: true,
		},
		{
			old:      makeCapacityStruct(1, 1, 1),
			now:      makeCapacityStruct(1, 1, 1),
			expected: false,
		},
	}

	for _, td := range testData {
		if (CheckUpdateInformation(td.old, td.now) == nil) != td.expected {
			t.Errorf("validation error")
		}
	}
}
