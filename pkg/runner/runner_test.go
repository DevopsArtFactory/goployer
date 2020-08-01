package runner

import (
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
