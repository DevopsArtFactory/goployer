package builder

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-test/deep"
)

func TestCheckValidationConfig(t *testing.T) {
	b := Builder{
		Config: Config{
			Manifest: "config/hello.yaml",
			Timeout:  DEFAULT_DEPLOYMENT_TIMEOUT,
		},
	}

	if err := b.CheckValidation(); err == nil || err.Error() != "you should choose at least one stack" {
		t.Errorf("validation failed: stack")
	}
	b.Config.Stack = "artd"

	b.Config.Ami = "ami-test"
	if err := b.CheckValidation(); err == nil || fmt.Sprintf("%s", err.Error()) != fmt.Sprintf("ami id cannot be used in different regions : %s", b.Config.Ami) {
		t.Errorf("validation failed: global ami")
	}
	b.Config.Region = "ap-northeast-2"

	b.Config.ReleaseNotesBase64 = "test-base64"
	b.Config.ReleaseNotes = "test"
	if err := b.CheckValidation(); err == nil || err.Error() != "you cannot specify the release-notes and release-notes-base64 at the same time" {
		t.Errorf("validation failed: release notes")
	}

	b.Config.ReleaseNotesBase64 = ""
	b.Config.PollingInterval = MIN_POLLING_INTERVAL - 1
	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("polling interval cannot be smaller than %.0f sec", MIN_POLLING_INTERVAL.Seconds()) {
		t.Errorf("validation failed: min polling interval")
	}
	b.Config.PollingInterval = MIN_POLLING_INTERVAL

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

	if err := b.CheckValidation(); err == nil || err.Error() != fmt.Sprintf("no %s file exists", METRIC_YAML_PATH) {
		t.Errorf("validation failed: metric file")
	}
	b.Config.DisableMetrics = true
}

func TestCheckValidationStack(t *testing.T) {
	b := Builder{
		Config: Config{
			Stack:           "artd",
			Manifest:        "config/hello.yaml",
			Timeout:         DEFAULT_DEPLOYMENT_TIMEOUT,
			PollingInterval: DEFAULT_POLLING_INTERVAL,
			DisableMetrics:  true,
		},
		MetricConfig: MetricConfig{
			Enabled: true,
			Region:  "ap-northeast-2",
			Storage: Storage{
				Name: "goployer-test",
				Type: "dynamodb",
			},
		},
		Stacks: []Stack{
			{
				Stack:   "artd",
				Account: "dev",
				Env:     "dev",
				Autoscaling: []ScalePolicy{
					{
						Name:              "",
						AdjustmentType:    "",
						ScalingAdjustment: 0,
						Cooldown:          0,
					},
				},
				Alarms: []AlarmConfigs{
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
		},
	}

	if err := b.CheckValidation(); err == nil || err.Error() != "autoscaling policy doesn't have a name" {
		t.Errorf("validation failed: stack-autoscaling")
	}
	b.Stacks[0].Autoscaling[0].Name = "test"

	if err := b.CheckValidation(); err == nil || err.Error() != "cloudwatch alarm doesn't have a name" {
		t.Errorf("validation failed: stack-cloudwatch alarm")
	}
	b.Stacks[0].Alarms[0].Name = "test"
	b.Stacks[0].Alarms[0].AlarmActions = []string{"test2"}

	if err := b.CheckValidation(); err == nil || err.Error() != "no scaling action exists : test2" {
		t.Errorf("validation failed: stack-cloudwatch alarm mapping")
	}
	b.Stacks[0].Alarms[0].AlarmActions = []string{"test"}

	b.Stacks[0].InstanceMarketOptions = &InstanceMarketOptions{
		MarketType:  "not spot",
		SpotOptions: SpotOptions{},
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

	b.Stacks[0].BlockDevices = []BlockDevice{
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

	b.Stacks[0].BlockDevices[0].VolumeType = "st1"
	b.Stacks[0].BlockDevices[0].VolumeSize = 100
	if err := b.CheckValidation(); err == nil || err.Error() != "volume size of st1 type should be larger than 500GiB" {
		t.Errorf("validation failed: volume size")
	}
	b.Stacks[0].BlockDevices[0].VolumeSize = 500

	b.Stacks[0].BlockDevices = append(b.Stacks[0].BlockDevices, BlockDevice{
		DeviceName: "/dev/xvda",
		VolumeSize: 100,
		VolumeType: "gp2",
	})
	if err := b.CheckValidation(); err == nil || err.Error() != "device names are duplicated : /dev/xvda" {
		t.Errorf("validation failed: duplicate volume name")
	}
	b.Stacks[0].BlockDevices[1].DeviceName = "/dev/xvdb"

	b.Stacks[0].LifecycleHooks = LifecycleHooks{
		LaunchTransition: []LifecycleHookSpecification{
			{
				LifecycleHookName:     "test",
				NotificationTargetARN: "arn:test",
			},
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "role_arn is needed if notification_target_arn is not empty : test" {
		t.Errorf("validation failed: lifecycle hook notification")
	}

	b.Stacks[0].LifecycleHooks = LifecycleHooks{
		LaunchTransition: []LifecycleHookSpecification{
			{
				LifecycleHookName: "test",
				RoleARN:           "arn:test",
			},
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "notification_target_arn is needed if role_arn is not empty : test" {
		t.Errorf("validation failed: lifecycle hook role")
	}
	b.Stacks[0].LifecycleHooks = LifecycleHooks{}

	b.Stacks[0].Regions = []RegionConfig{
		{
			Region: "ap-northeast-2",
			AmiId:  "",
		},
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to specify at least one ami id" {
		t.Errorf("validation failed: ami")
	}
	b.Stacks[0].Regions[0].AmiId = "ami-test"

	if err := b.CheckValidation(); err == nil || err.Error() != "you have to specify the instance type" {
		t.Errorf("validation failed: instance type")
	}
	b.Stacks[0].Regions[0].InstanceType = "t3.large"

	b.Stacks[0].Regions[0].HealthcheckTargetGroup = "test-tg"
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to add healthcheck_target_group to target_groups" {
		t.Errorf("validation failed: target groups missing")
	}
	b.Stacks[0].Regions[0].HealthcheckTargetGroup = ""

	b.Stacks[0].Regions[0].TargetGroups = []string{"test-tg"}
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to choose one target group as healthcheck_target_group" {
		t.Errorf("validation failed: healthcheck target group missing")
	}

	b.Stacks[0].Regions[0].HealthcheckTargetGroup = "test-tg2"
	if err := b.CheckValidation(); err == nil || err.Error() != "you have to choose the healthcheck_target_group from the target_groups" {
		t.Errorf("validation failed: wrong healthcheck target group")
	}
	b.Stacks[0].Regions[0].HealthcheckTargetGroup = "test-tg"

	b.Stacks[0].MixedInstancesPolicy = MixedInstancesPolicy{
		Enabled:                true,
		SpotAllocationStrategy: "capacity-optimized",
		SpotInstancePools:      1,
	}
	if err := b.CheckValidation(); err == nil || err.Error() != "you can only set spot_instance_pools with lowest-price spot_allocation_strategy" {
		t.Errorf("validation failed: spot allocation strategy")
	}
	b.Stacks[0].MixedInstancesPolicy.SpotAllocationStrategy = DEFAULT_SPOT_ALLOCATION_STRATEGY

	if err := b.CheckValidation(); err == nil || err.Error() != "you have to set at least one instance type to use in override" {
		t.Errorf("validation failed: override")
	}
	b.Stacks[0].MixedInstancesPolicy.Override = []string{"t3.large"}

	if err := b.CheckValidation(); err != nil {
		t.Errorf("validation failed: no error")
	}
}

func TestRefineConfig(t *testing.T) {
	type TestData struct {
		input  Config
		output Config
	}

	testData := []TestData{
		{
			input: Config{
				Timeout: 5,
			},
			output: Config{
				Timeout: 5 * time.Minute,
			},
		},
		{
			input: Config{
				PollingInterval: 5,
			},
			output: Config{
				PollingInterval: 5 * time.Second,
			},
		},
	}

	for _, td := range testData {
		r := RefineConfig(td.input)
		td.output.StartTimestamp = r.StartTimestamp
		if diff := deep.Equal(r, td.output); diff != nil {
			t.Error(diff)
		}
	}
}
