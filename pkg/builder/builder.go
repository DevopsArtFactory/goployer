package builder

import (
	"encoding/base64"
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
)

type Builder struct { // Do not add comments for this struct
	// Config from command
	Config Config

	// AWS related Configuration
	AwsConfig schemas.AWSConfig

	// Configuration for metrics
	MetricConfig schemas.MetricConfig

	// Stack configuration
	Stacks []schemas.Stack
}

type Config struct { // Do not add comments for this struct
	Manifest              string        `json:"manifest"`
	ManifestS3Region      string        `json:"manifest_s3_region"`
	Ami                   string        `json:"ami"`
	Env                   string        `json:"env"`
	Stack                 string        `json:"stack"`
	AssumeRole            string        `json:"assume_role"`
	Timeout               time.Duration `json:"timeout"`
	Region                string        `json:"region"`
	SlackOff              bool          `json:"slack_off"`
	LogLevel              string        `json:"log_level"`
	ExtraTags             string        `json:"extra_tags"`
	AnsibleExtraVars      string        `json:"ansible_extra_vars"`
	OverrideInstanceType  string        `json:"override_instance_type"`
	DisableMetrics        bool          `json:"disable_metrics"`
	ReleaseNotes          string        `json:"release_notes"`
	ReleaseNotesBase64    string        `json:"release_notes_base64"`
	ForceManifestCapacity bool          `json:"force_manifest_capacity"`
	PollingInterval       time.Duration `json:"polling_interval"`
	AutoApply             bool          `json:"auto-apply"`
	Application           string        `,inline`
	StartTimestamp        int64         `,inline`
}

var (
	NO_MANIFEST_EXISTS               = "Manifest file does not exist"
	DEFAULT_SPOT_ALLOCATION_STRATEGY = "lowest-price"
	DEFAULT_DEPLOYMENT_TIMEOUT       = 60 * time.Minute
	DEFAULT_POLLING_INTERVAL         = 60 * time.Second
	MIN_POLLING_INTERVAL             = 5 * time.Second
	S3_PREFIX                        = "s3://"
	availableBlockTypes              = []string{"io1", "gp2", "st1", "sc1"}
	timeFields                       = []string{"timeout", "polling-interval"}
	prohibitedTags                   = []string{"Name", "stack"}
)

type UserdataProvider interface {
	Provide() string
}

type LocalProvider struct {
	Path string
}

type S3Provider struct {
	Path string
}

func (l LocalProvider) Provide() string {
	if l.Path == "" {
		tool.ErrorLogging("Please specify userdata script path")
	}
	if !tool.FileExists(l.Path) {
		tool.ErrorLogging(fmt.Sprintf("File does not exist in %s", l.Path))
	}

	userdata, err := ioutil.ReadFile(l.Path)
	if err != nil {
		tool.ErrorLogging("Error reading userdata file")
	}

	return base64.StdEncoding.EncodeToString(userdata)
}

func (s S3Provider) Provide() string {
	return ""
}

func NewBuilder(config *Config) (Builder, error) {
	builder := Builder{}

	// parsing argument
	if config == nil {
		c := argumentParsing()
		config = &c
	}

	// set config
	builder.Config = *config

	return builder, nil
}

func (b Builder) SetManifestConfig() Builder {
	awsConfig, stacks := ParsingManifestFile(b.Config.Manifest)
	b.AwsConfig = awsConfig

	return b.SetStacks(stacks)
}

func (b Builder) SetManifestConfigWithS3(fileBytes []byte) Builder {
	awsConfig, stacks := buildStructFromYaml(fileBytes)
	b.AwsConfig = awsConfig

	return b.SetStacks(stacks)
}

// SetStacks set stack information
func (b Builder) SetStacks(stacks []schemas.Stack) Builder {

	if len(b.Config.AssumeRole) > 0 {
		for i, _ := range stacks {
			stacks[i].AssumeRole = b.Config.AssumeRole
		}
	}

	var deployStack schemas.Stack
	for _, stack := range stacks {
		if b.Config.Stack == stack.Stack {
			deployStack = stack
			break
		}
	}

	b.Stacks = stacks

	if len(b.Config.Env) == 0 {
		b.Config.Env = deployStack.Env
	}

	if b.Config.PollingInterval == 0 {
		if deployStack.PollingInterval > 0 {
			b.Config.PollingInterval = deployStack.PollingInterval
		} else {
			b.Config.PollingInterval = DEFAULT_POLLING_INTERVAL
		}
	}

	return b
}

// Validation Check
func (b Builder) CheckValidation() error {
	target_ami := b.Config.Ami
	target_region := b.Config.Region

	// check configurations
	// check stack
	if len(b.Config.Stack) == 0 {
		return fmt.Errorf("you should choose at least one stack")
	}

	if len(b.AwsConfig.Tags) > 0 && HasProhibited(b.AwsConfig.Tags) {
		return fmt.Errorf("you cannot use prohibited tags : %s", strings.Join(prohibitedTags, ","))
	}

	if len(b.Config.ExtraTags) > 0 && HasProhibited(strings.Split(b.Config.ExtraTags, ",")) {
		return fmt.Errorf("you cannot use prohibited tags : %s", strings.Join(prohibitedTags, ","))
	}

	// global AMI check
	if len(target_region) == 0 && len(target_ami) != 0 && strings.HasPrefix(target_ami, "ami-") {
		return fmt.Errorf("ami id cannot be used in different regions : %s", target_ami)
	}

	// check release notes
	if len(b.Config.ReleaseNotes) > 0 && len(b.Config.ReleaseNotesBase64) > 0 {
		return fmt.Errorf("you cannot specify the release-notes and release-notes-base64 at the same time")
	}

	// check polling interval
	if b.Config.PollingInterval < MIN_POLLING_INTERVAL {
		return fmt.Errorf("polling interval cannot be smaller than %.0f sec", MIN_POLLING_INTERVAL.Seconds())
	}

	if b.Config.PollingInterval >= b.Config.Timeout {
		return fmt.Errorf("polling interval should be lower than %.0f min", b.Config.Timeout.Minutes())
	}

	// Check Configuration about metrics
	if !b.Config.DisableMetrics {
		if len(b.MetricConfig.Region) <= 0 {
			return fmt.Errorf("you do not specify the region for metrics")
		}

		if len(b.MetricConfig.Storage.Name) <= 0 {
			return fmt.Errorf("you do not specify the name of storage for metrics")
		}

		if !tool.FileExists(METRIC_YAML_PATH) {
			return fmt.Errorf("no %s file exists", METRIC_YAML_PATH)
		}
	}

	// check validations in each stack
	for _, stack := range b.Stacks {
		if stack.Stack != b.Config.Stack {
			continue
		}

		if len(stack.Tags) > 0 && HasProhibited(stack.Tags) {
			return fmt.Errorf("you cannot use prohibited tags : %s", strings.Join(prohibitedTags, ","))
		}

		// Check AMI
		// Check Autoscaling and Alarm setting
		if len(stack.Autoscaling) != 0 && len(stack.Alarms) != 0 {
			policies := []string{}
			for _, scaling := range stack.Autoscaling {
				if len(scaling.Name) == 0 {
					return fmt.Errorf("autoscaling policy doesn't have a name")
				}
				policies = append(policies, scaling.Name)
			}
			for _, alarm := range stack.Alarms {
				if len(alarm.Name) == 0 {
					return fmt.Errorf("cloudwatch alarm doesn't have a name")
				}
				for _, action := range alarm.AlarmActions {
					if !tool.IsStringInArray(action, policies) {
						return fmt.Errorf("no scaling action exists : %s", action)
					}
				}
			}
		}

		// Check Spot Options
		if stack.InstanceMarketOptions != nil {
			if stack.InstanceMarketOptions.MarketType != "spot" {
				return fmt.Errorf("no valid market type : %s", stack.InstanceMarketOptions.MarketType)
			}

			if stack.InstanceMarketOptions.SpotOptions.BlockDurationMinutes%60 != 0 || stack.InstanceMarketOptions.SpotOptions.BlockDurationMinutes > 360 {
				return fmt.Errorf("block_duration_minutes should be one of [ 60, 120, 180, 240, 300, 360 ]")
			}

			if stack.InstanceMarketOptions.SpotOptions.SpotInstanceType == "persistent" && stack.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior == "terminate" {
				return fmt.Errorf("persistent type is not allowed with terminate behavior")
			}
		}

		// Check block device setting
		if len(stack.BlockDevices) > 0 {
			dNames := []string{}
			for _, block := range stack.BlockDevices {
				if len(block.DeviceName) == 0 {
					return fmt.Errorf("name of device is required")
				}

				if !tool.IsStringInArray(block.VolumeType, availableBlockTypes) {
					return fmt.Errorf("not available volume type : %s", block.VolumeType)
				}

				if block.VolumeType == "st1" && block.VolumeSize < 500 {
					return fmt.Errorf("volume size of st1 type should be larger than 500GiB")
				}

				if tool.IsStringInArray(block.DeviceName, dNames) {
					return fmt.Errorf("device names are duplicated : %s", block.DeviceName)
				} else {
					dNames = append(dNames, block.DeviceName)
				}
			}
		}

		if &stack.LifecycleHooks != nil {
			if len(stack.LifecycleHooks.LaunchTransition) > 0 {
				for _, l := range stack.LifecycleHooks.LaunchTransition {
					if len(l.NotificationTargetARN) > 0 && len(l.RoleARN) == 0 {
						return fmt.Errorf("role_arn is needed if notification_target_arn is not empty : %s", l.LifecycleHookName)
					}

					if len(l.RoleARN) > 0 && len(l.NotificationTargetARN) == 0 {
						return fmt.Errorf("notification_target_arn is needed if role_arn is not empty : %s", l.LifecycleHookName)
					}

					if l.HeartbeatTimeout == 0 {
						Logger.Warnf("you didn't specify the heartbeat timeout. you might have to wait too long time.")
					}
				}
			}

			if len(stack.LifecycleHooks.TerminateTransition) > 0 {
				for _, l := range stack.LifecycleHooks.TerminateTransition {
					if len(l.NotificationTargetARN) > 0 && len(l.RoleARN) == 0 {
						return fmt.Errorf("role_arn is needed if notification_target_arn is not empty : %s", l.LifecycleHookName)
					}

					if len(l.RoleARN) > 0 && len(l.NotificationTargetARN) == 0 {
						return fmt.Errorf("notification_target_arn is needed if role_arn is not empty  : %s", l.LifecycleHookName)
					}

					if l.HeartbeatTimeout == 0 {
						Logger.Warnf("you didn't specify the heartbeat timeout. you might have to wait too long time.")
					}
				}
			}
		}

		for _, region := range stack.Regions {
			// Check ami id
			if len(target_ami) == 0 && len(region.AmiId) == 0 {
				return fmt.Errorf("you have to specify at least one ami id")
			}

			// Check instance type
			if len(region.InstanceType) == 0 {
				return fmt.Errorf("you have to specify the instance type")
			}

			// Check target group
			if len(region.TargetGroups) > 0 && region.HealthcheckTargetGroup == "" {
				return fmt.Errorf("you have to choose one target group as healthcheck_target_group")
			}

			// Check load balancer
			if len(region.LoadBalancers) > 0 && region.HealthcheckLB == "" {
				return fmt.Errorf("you have to choose one load balancer as healthcheck_load_balancer")
			}

			// Check load balancer
			if region.HealthcheckLB != "" && len(region.TargetGroups) > 0 {
				return fmt.Errorf("you cannot use healthcheck_load_balancer with target_groups")
			}

			// Check load balancer and target group
			if region.HealthcheckLB != "" && region.HealthcheckTargetGroup != "" {
				return fmt.Errorf("you cannot use healthcheck_target_group and healthcheck_load_balancer at the same time")
			}

			// Check userdata
			if stack.Userdata.Type == "local" && len(stack.Userdata.Path) > 0 && !tool.FileExists(stack.Userdata.Path) {
				return fmt.Errorf("script file does not exists")
			}
		}

		if stack.MixedInstancesPolicy.Enabled {
			if len(stack.MixedInstancesPolicy.SpotAllocationStrategy) == 0 {
				stack.MixedInstancesPolicy.SpotAllocationStrategy = DEFAULT_SPOT_ALLOCATION_STRATEGY
			}

			if stack.MixedInstancesPolicy.SpotAllocationStrategy != "lowest-price" && stack.MixedInstancesPolicy.SpotInstancePools > 0 {
				return fmt.Errorf("you can only set spot_instance_pools with lowest-price spot_allocation_strategy")
			}

			if len(stack.MixedInstancesPolicy.Override) <= 0 {
				return fmt.Errorf("you have to set at least one instance type to use in override")
			}
		}
	}

	return nil
}

// Print Summary
func (b Builder) MakeSummary(target_stack string) string {
	summary := []string{}
	formatting := `
============================================================
Target Stack Deployment Information
============================================================
name             	: %s
env              	: %s
timeout          	: %.0f min
polling-interval 	: %.0f sec
assume role      	: %s
ansible-extra-vars  	: %s
extra tags       	: %s
============================================================
Stack
============================================================`
	summary = append(summary, fmt.Sprintf(formatting, b.AwsConfig.Name, b.Config.Env, b.Config.Timeout.Minutes(), b.Config.PollingInterval.Seconds(), b.Config.AssumeRole, b.Config.AnsibleExtraVars, b.Config.ExtraTags))

	for _, stack := range b.Stacks {
		if stack.Stack == target_stack {
			summary = append(summary, printEnvironment(stack))
		}
	}

	return strings.Join(summary, "\n")
}

func printEnvironment(stack schemas.Stack) string {
	formatting := `[ %s ]
Account             	: %s
Environment             : %s
IAM Instance Profile    : %s
tags              	: %s 
Capacity                : %+v
MixedInstancesPolicy
- Enabled 			: %t
- Override 			: %+v
- OnDemandPercentage  		: %d
- SpotAllocationStrategy 	: %s
- SpotInstancePools 		: %d
- SpotMaxPrice 			: %s
	
============================================================`
	summary := fmt.Sprintf(formatting,
		stack.Stack,
		stack.Account,
		stack.Env,
		stack.IamInstanceProfile,
		strings.Join(stack.Tags, ","),
		stack.Capacity,
		stack.MixedInstancesPolicy.Enabled,
		stack.MixedInstancesPolicy.Override,
		stack.MixedInstancesPolicy.OnDemandPercentage,
		stack.MixedInstancesPolicy.SpotAllocationStrategy,
		stack.MixedInstancesPolicy.SpotInstancePools,
		stack.MixedInstancesPolicy.SpotMaxPrice,
	)
	return summary
}

// Parsing Manifest File
func ParsingManifestFile(manifest string) (schemas.AWSConfig, []schemas.Stack) {
	var yamlFile []byte
	var err error

	yamlFile, err = ioutil.ReadFile(manifest)
	if err != nil {
		Logger.Errorf("Error reading YAML file: %s\n", err)
		return schemas.AWSConfig{}, nil
	}

	return buildStructFromYaml(yamlFile)
}

func buildStructFromYaml(yamlFile []byte) (schemas.AWSConfig, []schemas.Stack) {

	yamlConfig := schemas.YamlConfig{}
	err := yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		tool.FatalError(err)
	}

	awsConfig := schemas.AWSConfig{
		Name:     yamlConfig.Name,
		Userdata: yamlConfig.Userdata,
		Tags:     yamlConfig.Tags,
	}

	Stacks := yamlConfig.Stacks

	return awsConfig, Stacks
}

// Parsing Config from command
func argumentParsing() Config {
	keys := viper.AllKeys()
	config := Config{}

	val := reflect.ValueOf(&config).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		key := strings.ReplaceAll(typeField.Tag.Get("json"), "_", "-")
		if tool.IsStringInArray(key, keys) {
			t := val.FieldByName(typeField.Name)
			if t.CanSet() {
				switch t.Kind() {
				case reflect.String:
					t.SetString(viper.GetString(key))
				case reflect.Int64: // should use int64 not, int
					if tool.IsStringInArray(key, timeFields) {
						t.SetInt(int64(viper.GetDuration(key)))
					} else {
						t.SetInt(viper.GetInt64(key))
					}
				case reflect.Bool:
					t.SetBool(viper.GetBool(key))
				}
			}
		}
	}

	return RefineConfig(config)
}

// Set Userdata provider
func SetUserdataProvider(userdata schemas.Userdata, default_userdata schemas.Userdata) UserdataProvider {
	//Set default if no userdata exists in the stack
	if userdata.Type == "" {
		userdata.Type = default_userdata.Type
	}

	if userdata.Path == "" {
		userdata.Path = default_userdata.Path
	}

	if userdata.Type == "s3" {
		return S3Provider{Path: userdata.Path}
	}

	return LocalProvider{
		Path: userdata.Path,
	}
}

func (b Builder) PreConfigValidation() error {
	// check manifest file
	if len(b.Config.Manifest) <= 0 {
		return fmt.Errorf("you should specify manifest file")
	}

	if strings.HasPrefix(b.Config.Manifest, S3_PREFIX) && len(b.Config.ManifestS3Region) <= 0 {
		return fmt.Errorf("you have to specify region of s3 bucket: --manifest-s3-region")
	}

	if len(b.Config.Manifest) <= 0 || (!strings.HasPrefix(b.Config.Manifest, S3_PREFIX) && !tool.FileExists(b.Config.Manifest)) {
		return fmt.Errorf(NO_MANIFEST_EXISTS)
	}

	return nil
}

// RefineConfig refines the values for clear setting
func RefineConfig(config Config) Config {
	if config.Timeout < time.Minute {
		config.Timeout = config.Timeout * time.Minute
	}

	if config.PollingInterval < time.Second {
		config.PollingInterval = config.PollingInterval * time.Second
	}

	config.StartTimestamp = time.Now().Unix()

	return config
}

func HasProhibited(tags []string) bool {
	for _, t := range tags {
		arr := strings.Split(t, "=")
		k := arr[0]

		if tool.IsStringInArray(k, prohibitedTags) {
			return true
		}
	}

	return false
}
