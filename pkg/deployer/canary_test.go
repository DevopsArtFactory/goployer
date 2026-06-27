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

package deployer

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

func TestCheckCanaryVersion(t *testing.T) {
	region := constants.DefaultRegion
	regionShard := strings.ReplaceAll(region, "-", "")
	testData := []struct {
		Input    []string
		Expected int
	}{
		{
			Input: []string{
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s/xxxxxx", region, regionShard),
			},
			Expected: 0,
		},
		{
			Input: []string{
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s/xxxxxx", region, regionShard),
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s-canary-v001/xxxxxx", region, regionShard),
			},
			Expected: 1,
		},
		{
			Input: []string{
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s/xxxxxx", region, regionShard),
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s-canary-v001/xxxxxx", region, regionShard),
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s-canary-v002/xxxxxx", region, regionShard),
			},
			Expected: 2,
		},
	}

	for _, td := range testData {
		if output := CheckCanaryVersion(td.Input, region); output != td.Expected {
			t.Errorf("expected: %d, output: %d", td.Expected, output)
		}
	}
}

func TestCanaryWeights(t *testing.T) {
	region := schemas.RegionConfig{
		Canary: schemas.CanaryConfig{
			Weight: 25,
		},
	}

	stable, canary := CanaryWeights(region, false)
	if stable != 75 || canary != 25 {
		t.Fatalf("expected 75/25 canary weights, got %d/%d", stable, canary)
	}

	stable, canary = CanaryWeights(region, true)
	if stable != 100 || canary != 0 {
		t.Fatalf("expected 100/0 complete weights, got %d/%d", stable, canary)
	}
}

func TestCanaryWeightsDefault(t *testing.T) {
	stable, canary := CanaryWeights(schemas.RegionConfig{}, false)
	if stable != 90 || canary != 10 {
		t.Fatalf("expected default 90/10 canary weights, got %d/%d", stable, canary)
	}
}

func TestCanaryBakeTimeConfig(t *testing.T) {
	region := schemas.RegionConfig{
		Canary: schemas.CanaryConfig{
			BakeTime: 10 * time.Minute,
		},
	}

	if region.Canary.BakeTime != 10*time.Minute {
		t.Fatalf("expected 10m bake time, got %s", region.Canary.BakeTime)
	}
}

func TestShouldRollbackCanary(t *testing.T) {
	c := Canary{
		Deployer: &Deployer{
			StepStatus: map[int64]bool{constants.StepDeploy: true},
		},
	}

	if !c.ShouldRollbackCanary(schemas.Config{}) {
		t.Fatal("expected rollback after canary deploy step")
	}

	if c.ShouldRollbackCanary(schemas.Config{CompleteCanary: true}) {
		t.Fatal("did not expect rollback during complete canary")
	}

	c.StepStatus[constants.StepDeploy] = false
	if c.ShouldRollbackCanary(schemas.Config{}) {
		t.Fatal("did not expect rollback before canary deploy step")
	}
}

func TestCanaryAutoScalingGroupNamePrefersActiveDeployment(t *testing.T) {
	c := Canary{
		Deployer: &Deployer{
			AsgNames:  map[string]string{constants.DefaultRegion: "demo-canary-v002"},
			LatestAsg: map[string]string{constants.DefaultRegion: "demo-stable-v001"},
		},
	}

	if got := c.CanaryAutoScalingGroupName(constants.DefaultRegion); got != "demo-canary-v002" {
		t.Fatalf("expected active canary ASG, got %s", got)
	}
}

func TestCanaryAutoScalingGroupNameFallsBackToLatest(t *testing.T) {
	c := Canary{
		Deployer: &Deployer{
			AsgNames:  map[string]string{},
			LatestAsg: map[string]string{constants.DefaultRegion: "demo-canary-v002"},
		},
	}

	if got := c.CanaryAutoScalingGroupName(constants.DefaultRegion); got != "demo-canary-v002" {
		t.Fatalf("expected latest ASG fallback, got %s", got)
	}
}

func TestValidateCanaryDeploymentAllowsListenerLookup(t *testing.T) {
	c := Canary{
		Deployer: &Deployer{
			DeploymentFlag: map[string]string{},
			Stack: schemas.Stack{
				Regions: []schemas.RegionConfig{
					{
						Region: constants.DefaultRegion,
						Canary: schemas.CanaryConfig{
							LoadBalancer: "demoapp-xyzdapne2-ext",
							ListenerPort: 443,
						},
					},
				},
			},
		},
	}

	if err := c.ValidateCanaryDeployment(schemas.Config{}, constants.DefaultRegion); err != nil {
		t.Fatalf("unexpected validation error: %s", err)
	}
}

func TestValidateCanaryDeploymentRequiresListenerConfig(t *testing.T) {
	c := Canary{
		Deployer: &Deployer{
			DeploymentFlag: map[string]string{},
			Stack: schemas.Stack{
				Regions: []schemas.RegionConfig{{Region: constants.DefaultRegion}},
			},
		},
	}

	if err := c.ValidateCanaryDeployment(schemas.Config{}, constants.DefaultRegion); err == nil {
		t.Fatal("expected listener validation error")
	}
}

func TestValidateCanaryDeploymentRejectsUnsafeWeight(t *testing.T) {
	c := Canary{
		Deployer: &Deployer{
			DeploymentFlag: map[string]string{},
			Stack: schemas.Stack{
				Regions: []schemas.RegionConfig{
					{
						Region: constants.DefaultRegion,
						Canary: schemas.CanaryConfig{
							LoadBalancer: "demoapp-xyzdapne2-ext",
							ListenerPort: 443,
							Weight:       91,
						},
					},
				},
			},
		},
	}

	if err := c.ValidateCanaryDeployment(schemas.Config{}, constants.DefaultRegion); err == nil {
		t.Fatal("expected canary weight validation error")
	}
}

func TestCanaryRunAPITestDelegatesToDeployer(t *testing.T) {
	logger := Logger.New()
	logger.SetOutput(io.Discard)

	c := Canary{
		Deployer: &Deployer{
			Stack:      schemas.Stack{APITestEnabled: false},
			Logger:     logger,
			StepStatus: map[int64]bool{constants.StepGatherMetrics: true, constants.StepCleanChecking: true},
		},
	}

	if err := c.RunAPITest(schemas.Config{CompleteCanary: true}); err != nil {
		t.Fatalf("unexpected API test error: %s", err)
	}
}
