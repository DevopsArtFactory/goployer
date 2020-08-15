package runner

import (
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/go-test/deep"
	"testing"
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
