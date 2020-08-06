package deployer

import (
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"testing"
)

func TestGetStackName(t *testing.T) {
	b := BlueGreen{Deployer{Stack: schemas.Stack{Stack: "Test"}}}

	input := b.GetStackName()
	expected := "Test"

	if input != expected {
		t.Error(input)
	}
}

func TestCheckRegionExist(t *testing.T) {
	target := "ap-northeast-2"
	regionList := []schemas.RegionConfig{
		schemas.RegionConfig{
			Region: "us-east-1",
		},
	}
	input := CheckRegionExist(target, regionList)
	expected := false

	if input != expected {
		t.Error(regionList, target)
	}

	regionList = append(regionList, schemas.RegionConfig{
		Region: target,
	})

	input = CheckRegionExist(target, regionList)
	expected = true

	if input != expected {
		t.Error(regionList, target)
	}
}
