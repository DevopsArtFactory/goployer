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

package builder

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

func TestCheckValidationConfig(t *testing.T) {
	b := Builder{
		Config: schemas.Config{
			Stack:    "artd",
			Manifest: "config/hello.yaml",
			Timeout:  constants.DefaultDeploymentTimeout,
		},
		Stacks: []schemas.Stack{
			{
				Stack:   "artp",
				Account: "dev",
				Env:     "dev",
				Regions: []schemas.RegionConfig{
					{
						Region:       "ap-northeast-2",
						AmiID:        "ami-test",
						InstanceType: "t3.small",
						ScheduledActions: []string{
							"fake_action",
						},
					},
				},
			},
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "stack does not exist: artd" {
		t.Errorf("validation failed: stack existence check")
	}
	b.Stacks[0].Stack = "artd"

	b.Config.Ami = "ami-test"
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("ami id cannot be used in different regions : %s", b.Config.Ami) {
		t.Errorf("validation failed: global ami")
	}
	b.Config.Region = "ap-northeast-2"

	b.Config.ReleaseNotesBase64 = "test-base64"
	b.Config.ReleaseNotes = constants.TestString
	if err := b.CheckValidation(); err == nil || err.Error() != "you cannot specify the release-notes and release-notes-base64 at the same time" {
		t.Errorf("validation failed: release notes")
	}

	b.Config.ReleaseNotesBase64 = ""
	b.Config.PollingInterval = constants.MinPollingInterval - 1
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("polling interval cannot be smaller than %.0f sec", constants.MinPollingInterval.Seconds()) {
		t.Errorf("validation failed: min polling interval")
	}
	b.Config.PollingInterval = constants.MinPollingInterval

	b.Config.PollingInterval = b.Config.Timeout + 1
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("polling interval should be lower than %.0f min", b.Config.Timeout.Minutes()) {
		t.Errorf("validation failed: max polling interval")
	}
	b.Config.PollingInterval = b.Config.Timeout - 1

	b.MetricConfig.Region = ""
	if err := b.CheckValidation(); err == nil || err.Error() != "you do not specify the region for metrics" {
		t.Errorf("validation failed: metric region")
	}
	b.MetricConfig.Region = "ap-northeast-2"

	b.MetricConfig.Storage.Name = ""
	if err := b.CheckValidation(); err == nil || err.Error() != "you do not specify the name of storage for metrics" {
		t.Errorf("validation failed: metric name")
	}
	b.MetricConfig.Storage.Name = "goployer-test"

	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("no %s file exists", constants.MetricYamlPath) {
		t.Errorf("validation failed: metric file")
	}
	b.Config.DisableMetrics = true
	b.Config.OverrideSpotType = "t3.small,t2.large|m5.small"
	if err := b.CheckValidation(); err == nil || err.Error() != "you must using delimiter '|'" {
		t.Errorf("validation failed: OverrideSpotType")
	}

	b.Config.OverrideSpotType = "t3.small|c6g.medium|t2.large"
	if err := b.CheckValidation(); err == nil || err.Error() != "you can only use same type of spot instance type(arm64 and intel_x86 type)" {
		t.Errorf("validation failed: OverrideSpotInstanceType")
	}
}

func TestCheckValidationScheduledAction(t *testing.T) {
	scheduledActionName := "scale_in_during_weekend"
	b := Builder{
		AwsConfig: schemas.AWSConfig{
			Name: "hello",
			ScheduledActions: []schemas.ScheduledAction{
				{},
			},
		},
		Config: schemas.Config{
			Stack:           "artd",
			Manifest:        "config/hello.yaml",
			Timeout:         constants.DefaultDeploymentTimeout,
			PollingInterval: constants.DefaultPollingInterval,
			DisableMetrics:  true,
		},
		MetricConfig: schemas.MetricConfig{
			Enabled: true,
			Region:  "ap-northeast-2",
			Storage: schemas.Storage{
				Name: "goployer-test",
				Type: "dynamodb",
			},
		},
		Stacks: []schemas.Stack{
			{
				Stack:   "artd",
				Account: "dev",
				Env:     "dev",
				Regions: []schemas.RegionConfig{
					{
						Region:       "ap-northeast-2",
						AmiID:        "ami-test",
						InstanceType: "t3.small",
						ScheduledActions: []string{
							"fake_action",
						},
					},
				},
			},
		},
	}

	if err := b.CheckValidation(); err == nil || err.Error() != "you have to set name of scheduled action" {
		t.Errorf("validation failed: scheduled action name")
	}
	b.AwsConfig.ScheduledActions[0].Name = scheduledActionName

	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("recurrence is required field: %s", scheduledActionName) {
		t.Errorf("validation failed: scheduled action recurrence")
	}
	b.AwsConfig.ScheduledActions[0].Recurrence = "30 0 1 1,6,12 *"

	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("capacity is required field: %s", scheduledActionName) {
		t.Errorf("validation failed: scheduled action capacity")
	}
	b.AwsConfig.ScheduledActions[0].Capacity = &schemas.Capacity{
		Min:     1,
		Desired: 1,
		Max:     1,
	}

	b.APITestTemplates = []*schemas.APITestTemplate{
		{},
	}

	b.APITestTemplates[0].Name = ""
	if err := b.CheckValidation(); err == nil || err.Error() != "name of API test is required" {
		t.Errorf("validation failed: api-test name")
	}
	b.APITestTemplates[0].Name = "test"

	b.APITestTemplates[0].Duration = 999 * time.Millisecond
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("duration for api test cannot be smaller than %.0f seconds", constants.MinAPITestDuration.Seconds()) {
		t.Errorf("validation failed: api-test duration")
	}
	b.APITestTemplates[0].Duration = 2 * time.Second

	b.APITestTemplates[0].RequestPerSecond = 0
	if err := b.CheckValidation(); err == nil || err.Error() != "request per second should be specified" {
		t.Errorf("validation failed: api-test request-per-second")
	}
	b.APITestTemplates[0].RequestPerSecond = 5

	b.APITestTemplates[0].APIs = []*schemas.APIManifest{
		{
			Method: "TEST",
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "api is not allowed: TEST" {
		t.Errorf("validation failed: api-test method")
	}
	b.APITestTemplates[0].APIs[0].Method = "GET"

	b.APITestTemplates[0].APIs[0].Body = []string{"key=value"}
	if err := b.CheckValidation(); err == nil || err.Error() != "api with GET request cannot have body" {
		t.Errorf("validation failed: api-test get-body mismatching")
	}
	b.APITestTemplates[0].APIs[0].Method = "POST"

	if err := b.CheckValidation(); err == nil || err.Error() != "scheduled action is not defined: fake_action" {
		t.Errorf("validation failed: scheduled action existence")
	}
}

func TestCheckValidationStack(t *testing.T) {
	b := Builder{
		Config: schemas.Config{
			Stack:           "artd",
			Manifest:        "config/hello.yaml",
			Timeout:         constants.DefaultDeploymentTimeout,
			PollingInterval: constants.DefaultPollingInterval,
			DisableMetrics:  true,
		},
		MetricConfig: schemas.MetricConfig{
			Enabled: true,
			Region:  "ap-northeast-2",
			Storage: schemas.Storage{
				Name: "goployer-test",
				Type: "dynamodb",
			},
		},
		Stacks: []schemas.Stack{
			{
				Stack:   "artd",
				Account: "dev",
				Env:     "dev",
				Autoscaling: []schemas.ScalePolicy{
					{
						Name:              "",
						AdjustmentType:    "",
						ScalingAdjustment: 0,
						Cooldown:          0,
					},
				},
				Alarms: []schemas.AlarmConfigs{
					{
						Name:              "",
						Namespace:         "",
						Metric:            "",
						Statistic:         "",
						Comparison:        "",
						Threshold:         0,
						Period:            0,
						EvaluationPeriods: 0,
						AlarmActions:      nil,
					},
				},
			},
			{
				Stack:   "artd",
				Account: "dev",
				Env:     "dev",
			},
		},
		APITestTemplates: []*schemas.APITestTemplate{
			{
				Name:             "api-test",
				Duration:         time.Second * 2,
				RequestPerSecond: 30,
				APIs: []*schemas.APIManifest{
					{
						Method: "GET",
						URL:    "api-test.com",
					},
				},
			},
		},
	}

	if err := b.CheckValidation(); err == nil || err.Error() != "duplicated stack key between stacks : artd" {
		t.Errorf("validation failed: duplicated stack key")
	}
	b.Stacks[1].Stack = "artd2"

	if err := b.CheckValidation(); err == nil || err.Error() != "duplicated env between stacks : dev" {
		t.Errorf("validation failed: duplicated env")
	}
	b.Stacks = b.Stacks[:1]

	if err := b.CheckValidation(); err == nil || err.Error() != "autoscaling policy doesn't have a name" {
		t.Errorf("validation failed: stack-autoscaling")
	}
	b.Stacks[0].Autoscaling[0].Name = constants.TestString

	if err := b.CheckValidation(); err == nil || err.Error() != "cloudwatch alarm doesn't have a name" {
		t.Errorf("validation failed: stack-cloudwatch alarm")
	}
	b.Stacks[0].Alarms[0].Name = constants.TestString
	b.Stacks[0].Alarms[0].AlarmActions = []string{"test2"}

	if err := b.CheckValidation(); err == nil || err.Error() != "no scaling action exists : test2" {
		t.Errorf("validation failed: stack-cloudwatch alarm mapping")
	}
	b.Stacks[0].Alarms[0].AlarmActions = []string{constants.TestString}

	b.Stacks[0].InstanceMarketOptions = &schemas.InstanceMarketOptions{
		MarketType:  "not spot",
		SpotOptions: schemas.SpotOptions{},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "no valid market type : not spot" {
		t.Errorf("validation failed: stack-instance market options type")
	}

	b.Stacks[0].InstanceMarketOptions.MarketType = "spot"
	b.Stacks[0].InstanceMarketOptions.SpotOptions.BlockDurationMinutes = 10
	if err := b.CheckValidation(); err == nil || err.Error() != "block_duration_minutes should be one of [ 60, 120, 180, 240, 300, 360 ]" {
		t.Errorf("validation failed: stack-instance market options block_duration_minutes")
	}

	b.Stacks[0].InstanceMarketOptions.SpotOptions.BlockDurationMinutes = 370
	if err := b.CheckValidation(); err == nil || err.Error() != "block_duration_minutes should be one of [ 60, 120, 180, 240, 300, 360 ]" {
		t.Errorf("validation failed: stack-instance market options block_duration_minutes")
	}

	b.Stacks[0].InstanceMarketOptions.SpotOptions.BlockDurationMinutes = 60
	b.Stacks[0].InstanceMarketOptions.SpotOptions.SpotInstanceType = "persistent"
	b.Stacks[0].InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior = "terminate"

	if err := b.CheckValidation(); err == nil || err.Error() != "persistent type is not allowed with terminate behavior" {
		t.Errorf("validation failed: stack-instance market options spot type")
	}
	b.Stacks[0].InstanceMarketOptions = nil

	b.Stacks[0].BlockDevices = []schemas.BlockDevice{
		{
			DeviceName: "",
			VolumeSize: 0,
			VolumeType: "",
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "name of device is required" {
		t.Errorf("validation failed: block device")
	}

	b.Stacks[0].BlockDevices[0].DeviceName = "/dev/xvda"
	b.Stacks[0].BlockDevices[0].VolumeType = "test"
	if err := b.CheckValidation(); err == nil || err.Error() != "not available volume type : test" {
		t.Errorf("validation failed: volume type")
	}

	b.Stacks[0].BlockDevices[0].VolumeType = "gp2"
	b.Stacks[0].BlockDevices[0].VolumeSize = 0
	if err := b.CheckValidation(); err == nil || err.Error() != "volume size of gp2 or gp3 type should be larger than 1GiB" {
		t.Errorf("validation failed: volume size - gp2")
	}

	b.Stacks[0].BlockDevices[0].VolumeType = "st1"
	b.Stacks[0].BlockDevices[0].VolumeSize = 100
	if err := b.CheckValidation(); err == nil || err.Error() != "volume size of st1 type should be larger than 500GiB" {
		t.Errorf("validation failed: volume size - st1")
	}

	b.Stacks[0].BlockDevices[0].VolumeType = "io1"
	b.Stacks[0].BlockDevices[0].VolumeSize = 1
	if err := b.CheckValidation(); err == nil || err.Error() != "volume size of io1 and io2 type should be larger than 4GiB" {
		t.Errorf("validation failed: volume size - io1")
	}

	b.Stacks[0].BlockDevices[0].VolumeType = "io2"
	b.Stacks[0].BlockDevices[0].VolumeSize = 1
	if err := b.CheckValidation(); err == nil || err.Error() != "volume size of io1 and io2 type should be larger than 4GiB" {
		t.Errorf("validation failed: volume size - io2")
	}

	b.Stacks[0].BlockDevices[0].VolumeSize = 4
	b.Stacks[0].BlockDevices[0].Iops = 0
	if err := b.CheckValidation(); err == nil || err.Error() != "iops of io1 and io2 type should be larger than 100" {
		t.Errorf("validation failed: iops - io2")
	}
	b.Stacks[0].BlockDevices[0].Iops = 100

	b.Stacks[0].BlockDevices = append(b.Stacks[0].BlockDevices, schemas.BlockDevice{
		DeviceName: "/dev/xvda",
		VolumeSize: 100,
		VolumeType: "gp2",
	})
	if err := b.CheckValidation(); err == nil || err.Error() != "device names are duplicated : /dev/xvda" {
		t.Errorf("validation failed: duplicate volume name")
	}
	b.Stacks[0].BlockDevices[1].DeviceName = "/dev/xvdb"

	b.Stacks[0].LifecycleHooks = &schemas.LifecycleHooks{
		LaunchTransition: []schemas.LifecycleHookSpecification{
			{
				LifecycleHookName:     "test",
				NotificationTargetARN: "arn:test",
			},
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "role_arn is needed if notification_target_arn is not empty : test" {
		t.Errorf("validation failed: lifecycle hook notification")
	}

	b.Stacks[0].LifecycleHooks = &schemas.LifecycleHooks{
		LaunchTransition: []schemas.LifecycleHookSpecification{
			{
				LifecycleHookName: "test",
				RoleARN:           "arn:test",
			},
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "notification_target_arn is needed if role_arn is not empty: test" {
		t.Errorf("validation failed: lifecycle hook role")
	}
	b.Stacks[0].LifecycleHooks = nil

	b.Stacks[0].ReplacementType = constants.BlueGreenDeployment
	b.Stacks[0].TerminationDelayRate = 101
	if err := b.CheckValidation(); err == nil || err.Error() != "termination_delay_rate cannot exceed 100. It should be 0<=x<=100" {
		t.Errorf("validation failed: termination delay rate exceed 100")
	}

	b.Stacks[0].TerminationDelayRate = -1
	if err := b.CheckValidation(); err == nil || err.Error() != "termination_delay_rate cannot be negative. It should be 0<=x<=100" {
		t.Errorf("validation failed: termination delay rate negative")
	}
	b.Stacks[0].TerminationDelayRate = 0

	b.Stacks[0].Regions = []schemas.RegionConfig{
		{
			Region: "ap-northeast-2",
			AmiID:  "",
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to specify at least one ami id" {
		t.Errorf("validation failed: ami")
	}
	b.Stacks[0].Regions[0].AmiID = "ami-test"

	if err := b.CheckValidation(); err == nil || err.Error() != "you have to specify the instance type" {
		t.Errorf("validation failed: instance type")
	}
	b.Stacks[0].Regions[0].InstanceType = "t3.large"

	b.Stacks[0].Regions[0].HealthcheckTargetGroup = ""
	b.Stacks[0].Regions[0].TargetGroups = []string{"test-tg"}
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to choose one target group as healthcheck_target_group" {
		t.Errorf("validation failed: healthcheck target group missing")
	}
	b.Stacks[0].Regions[0].HealthcheckTargetGroup = "test-tg"

	b.Stacks[0].Regions[0].HealthcheckLB = ""
	b.Stacks[0].Regions[0].LoadBalancers = []string{"test-lb"}
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to choose one load balancer as healthcheck_load_balancer" {
		t.Errorf("validation failed: healthcheck load balancer missing")
	}
	b.Stacks[0].Regions[0].HealthcheckLB = "test-lb"

	if err := b.CheckValidation(); err == nil || err.Error() != "you cannot use healthcheck_load_balancer with target_groups" {
		t.Errorf("validation failed: mixing healthcheck load balancer and target groups")
	}
	b.Stacks[0].Regions[0].TargetGroups = nil

	if err := b.CheckValidation(); err == nil || err.Error() != "you cannot use healthcheck_target_group and healthcheck_load_balancer at the same time" {
		t.Errorf("validation failed: mixing healthcheck target group and healthcheck load balancer")
	}
	b.Stacks[0].Regions[0].HealthcheckLB = ""
	b.Stacks[0].Regions[0].LoadBalancers = nil

	b.Stacks[0].Userdata = schemas.Userdata{
		Type: "local",
		Path: "script/cannotfindpath.yaml",
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "script file does not exists" {
		t.Errorf("validation failed: stack script check")
	}
	b.Stacks[0].Userdata = schemas.Userdata{}

	b.Stacks[0].MixedInstancesPolicy = schemas.MixedInstancesPolicy{
		Enabled:                true,
		SpotAllocationStrategy: "capacity-optimized",
		SpotInstancePools:      1,
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "you can only set spot_instance_pools with lowest-price spot_allocation_strategy" {
		t.Errorf("validation failed: spot allocation strategy")
	}
	b.Stacks[0].MixedInstancesPolicy.SpotAllocationStrategy = constants.DefaultSpotAllocationStrategy

	if err := b.CheckValidation(); err == nil || err.Error() != "you have to set at least one instance type to use in override" {
		t.Errorf("validation failed: override")
	}
	b.Stacks[0].MixedInstancesPolicy.Override = []string{"t3.large"}

	b.Stacks[0].APITestEnabled = true
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("you have to specify the name of template for api test: %s", b.Stacks[0].Stack) {
		t.Errorf("validation failed: stack api_test_enabled but no manifest")
	}

	b.Stacks[0].APITestTemplate = "api-test-fake"
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("template does not exist in the list: %s", b.Stacks[0].APITestTemplate) {
		t.Errorf("validation failed: stack wrong api_test_template")
	}
	b.Stacks[0].APITestTemplate = "api-test"

	if err := b.CheckValidation(); err != nil {
		t.Errorf("validation failed: no error")
	}
}

func TestRefineConfig(t *testing.T) {
	type TestData struct {
		input  schemas.Config
		output schemas.Config
	}

	testData := []TestData{
		{
			input: schemas.Config{
				Timeout: 5,
				Region:  "ap-northeast-2",
			},
			output: schemas.Config{
				Timeout: 5 * time.Minute,
				Region:  "ap-northeast-2",
			},
		},
		{
			input: schemas.Config{
				PollingInterval: 5,
				Region:          "ap-northeast-2",
			},
			output: schemas.Config{
				PollingInterval: 5 * time.Second,
				Region:          "ap-northeast-2",
			},
		},
	}

	for _, td := range testData {
		r, _ := RefineConfig(td.input)
		td.output.StartTimestamp = r.StartTimestamp
		if diff := deep.Equal(r, td.output); diff != nil {
			t.Error(diff)
		}
	}

	regionTest := TestData{
		input: schemas.Config{
			Timeout: 5,
		},
		output: schemas.Config{
			Timeout: 5 * time.Minute,
			Region:  "us-east-2",
		},
	}

	os.Setenv("AWS_DEFAULT_REGION", "us-east-2")
	r, _ := RefineConfig(regionTest.input)
	regionTest.output.StartTimestamp = r.StartTimestamp
	if diff := deep.Equal(r, regionTest.output); diff != nil {
		t.Error(diff)
	}
}

func TestHasProhibited(t *testing.T) {
	type TestData struct {
		input  []string
		output bool
	}

	testData := []TestData{
		{
			input:  []string{"test=test"},
			output: false,
		},
		{
			input:  []string{"Name=test"},
			output: true,
		},
		{
			input:  []string{"name=test"},
			output: false,
		},
		{
			input:  []string{"name=test", "app=test"},
			output: false,
		},
	}

	for _, td := range testData {
		if HasProhibited(td.input) != td.output {
			t.Errorf("wrong validation: %s/%t", strings.Join(td.input, ","), td.output)
		}
	}
}

func TestValidCronExpression(t *testing.T) {
	testData := []struct {
		input    string
		expected bool
	}{
		{
			input:    "* * * * *",
			expected: true,
		},
		{
			input:    "* * * * * *",
			expected: false,
		},
		{
			input:    "1 * * * *",
			expected: true,
		},
		{
			input:    "-1 * * * *",
			expected: false,
		},
		{
			input:    "59 * * * *",
			expected: true,
		},
		{
			input:    "60 * * * *",
			expected: false,
		},
		{
			input:    "* 1 * * *",
			expected: true,
		},
		{
			input:    "* -1 * * *",
			expected: false,
		},
		{
			input:    "* 23 * * *",
			expected: true,
		},
		{
			input:    "* 24 * * *",
			expected: false,
		},
		{
			input:    "* * 1 * *",
			expected: true,
		},
		{
			input:    "* * 0 * *",
			expected: false,
		},
		{
			input:    "* * 31 * *",
			expected: true,
		},
		{
			input:    "* * 32 * *",
			expected: false,
		},
		{
			input:    "* * -1 * *",
			expected: false,
		},
		{
			input:    "* * * 1 *",
			expected: true,
		},
		{
			input:    "* * * 12 *",
			expected: true,
		},
		{
			input:    "* * * 0 *",
			expected: false,
		},
		{
			input:    "* * * 13 *",
			expected: false,
		},
		{
			input:    "* * * 1,12 *",
			expected: true,
		},
		{
			input:    "* * * 0,12 *",
			expected: false,
		},
		{
			input:    "* * * * MON-TUE",
			expected: true,
		},
		{
			input:    "* * * * TUE-FRI",
			expected: true,
		},
		{
			input:    "* * * * TUE-SAT",
			expected: true,
		},
		{
			input:    "* * * * TUE,SUN",
			expected: true,
		},
		{
			input:    "* * * * SUN,SAT,MON-WED",
			expected: true,
		},
		{
			input:    "* * * * SUN,SAT,MONDAY",
			expected: false,
		},
		{
			input:    "* * * * MON-SAT-SUN",
			expected: false,
		},
		{
			input:    "* * * * 0",
			expected: true,
		},
		{
			input:    "* * * * 0-3",
			expected: true,
		},
		{
			input:    "* * * * 0-8",
			expected: false,
		},
	}

	for _, td := range testData {
		if result, _ := ValidCronExpression(td.input); result != td.expected {
			t.Errorf("error occurred with input: %s", td.input)
		}
	}
}
