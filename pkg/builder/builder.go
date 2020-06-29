package builder

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"time"
)

var (
	NO_MANIFEST_EXISTS="Manifest file does not exist"
	DFEAULT_SPOT_ALLOCATION_STRATEGY="lowest-price"
	availableBlockTypes=[]string{"io1", "gp2", "st1", "sc1"}
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

type Builder struct {
	Config    Config    // Config from command
	AwsConfig AWSConfig // Common Config
	Stacks    []Stack   // Stack Config
}

type Config struct {
	Manifest 		string
	Ami  	 		string
	Env  	 		string
	Stack  	 		string
	AssumeRole 		string
	Timeout  		int64
	StartTimestamp 	int64
	Region   		string
	Confirm  		bool
	SlackOff  		bool
	LogLevel  		string
	ExtraTags		string
}


type YamlConfig struct {
	Name     string   `yaml:"name"`
	Userdata Userdata `yaml:"userdata"`
	Tags     []string `yaml:"tags"`
	Stacks   []Stack  `yaml:"stacks"`
}

type AWSConfig struct {
	Name     string
	Userdata Userdata
	Tags     []string
}

type Userdata struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type ScalePolicy struct {
	Name				string `yaml:"name"`
	AdjustmentType 		string `yaml:"adjustment_type"`
	ScalingAdjustment 	int64 `yaml:"scaling_adjustment"`
	Cooldown 			int64 `yaml:"cooldown"`
}

type AlarmConfigs struct {
	Name				string
	Namespace 			string
	Metric 				string
	Statistic 			string
	Comparison 			string
	Threshold 			float64
	Period 				int64
	EvaluationPeriods 	int64 	 `yaml:"evaluation_periods"`
	AlarmActions 		[]string `yaml:"alarm_actions"`
}

type Stack struct {
	Stack                 string                `yaml:"stack"`
	Account               string                `yaml:"account"`
	Env                   string                `yaml:"env"`
	ReplacementType       string                `yaml:"replacement_type"`
	Userdata              Userdata              `yaml:"userdata"`
	IamInstanceProfile    string                `yaml:"iam_instance_profile"`
	AnsibleTags           string                `yaml:"ansible_tags"`
	AssumeRole            string                `yaml:"assume_role"`
	EbsOptimized          bool                  `yaml:"ebs_optimized"`
	InstanceMarketOptions InstanceMarketOptions `yaml:"instance_market_options"`
	MixedInstancesPolicy  MixedInstancesPolicy 	`yaml:"mixed_instances_policy,omitempty"`
	BlockDevices          []BlockDevice         `yaml:"block_devices"`
	extraTags             string                `yaml:"extra_vars"`
	Capacity              Capacity              `yaml:"capacity"`
	Autoscaling           []ScalePolicy         `yaml:"autoscaling"`
	Alarms                []AlarmConfigs        `yaml:alarms`
	LifecycleCallbacks    LifecycleCallbacks    `yaml:"lifecycle_callbacks"`
	Regions               []RegionConfig        `yaml:"regions"`
}

type InstanceMarketOptions struct {
	MarketType  string      `yaml:"market_type"`
	SpotOptions SpotOptions `yaml:"spot_options"`
}

type MixedInstancesPolicy struct {
	Enabled					bool		`yaml:"enabled"`
	Override 				[]string 	`yaml:"override_instance_types"`
	OnDemandPercentage  	int64      	`yaml:"on_demand_percentage"`
	SpotAllocationStrategy  string      `yaml:"spot_allocation_strategy"`
	SpotInstancePools		int64		`yaml:"spot_instance_pools"`
	SpotMaxPrice 			string 		`yaml:"spot_max_price,omitempty"`
}

type SpotOptions struct {
	BlockDurationMinutes int64 `yaml:"block_duration_minutes"`
	InstanceInterruptionBehavior string `yaml:"instance_interruption_behavior"`
	MaxPrice string `yaml:"max_price"`
	SpotInstanceType string `yaml:"spot_instance_type"`
}

type BlockDevice struct {
	DeviceName string `yaml:"device_name"`
	VolumeSize int64 `yaml:"volume_size"`
	VolumeType string `yaml:"volume_type"`
}

type LifecycleCallbacks struct {
	PreTerminatePastClusters []string `yaml:"pre_terminate_past_clusters"`
}

type RegionConfig struct {
	Region 					string 		`yaml:"region"`
	UsePublicSubnets		bool 		`yaml:"use_public_subnets"`
	InstanceType 			string 		`yaml:"instance_type"`
	SshKey 					string 		`yaml:"ssh_key"`
	AmiId 					string 		`yaml:"ami_id"`
	VPC 					string 		`yaml:"vpc"`
	SecurityGroups 			[]string 	`yaml:"security_groups"`
	HealthcheckLB 			string 		`yaml:"healthcheck_load_balancer"`
	HealthcheckTargetGroup 	string 		`yaml:"healthcheckTargetGroup"`
	TargetGroups 			[]string 	`yaml:"target_groups"`
	LoadBalancers 			[]string 	`yaml:"loadbalancers"`
	AvailabilityZones 		[]string 	`yaml:"availability_zones"`
}

type Capacity struct {
	Min 	int64 `yaml:"min"`
	Max 	int64 `yaml:"max"`
	Desired int64 `yaml:"desired"`
}


func (l LocalProvider) Provide() string {
	if l.Path == "" {
		tool.ErrorLogging("Please specify userdata script path")
	}
	if ! tool.FileExists(l.Path) {
		tool.ErrorLogging(fmt.Sprintf("File does not exist in %s", l.Path))
	}

	userdata, err := ioutil.ReadFile(l.Path)
	if err != nil {
		tool.ErrorLogging("Error reading userdata file")
	}

	return base64.StdEncoding.EncodeToString(userdata)
}

func (s S3Provider) Provide() string  {
	return ""
}

func NewBuilder() (Builder, error) {
	builder := Builder{}

	// Parsing Argument
	config := argumentParsing()

	//Check manifest file
	if len(config.Manifest) == 0 || ! tool.FileExists(config.Manifest) {
		return builder, fmt.Errorf(NO_MANIFEST_EXISTS)
	}


	// Set config
	builder.Config = config

	return builder.SetStacks(), nil
}

// SetStacks set stack information
func (b Builder) SetStacks() Builder {

	awsConfig, Stacks := parsingManifestFile(b.Config.Manifest)

	b.AwsConfig = awsConfig
	b.Stacks = Stacks

	if b.Config.Env == "" {
		for _, stack := range Stacks {
			if b.Config.Stack == stack.Stack {
				b.Config.Env = stack.Env
			}
		}
	}

	return b
}

// Validation Check
func (b Builder) CheckValidation() error {
	target_ami := b.Config.Ami
	target_region := b.Config.Region

	// Check stack
	if len(b.Config.Stack) == 0 {
		return fmt.Errorf("you should choose at least one stack.")
	}

	// Global AMI check
	if len(target_region) == 0 && len(target_ami) != 0 && strings.HasPrefix(target_ami, "ami-") {
		// One ami id cannot be used in different regions
		return fmt.Errorf("one ami id cannot be used in different regions : %s", target_ami)
	}

	// check validations in each stack
	for _, stack := range b.Stacks {
		// Check AMI
		// Check Autoscaling and Alarm setting
		if len(stack.Autoscaling) != 0 && len(stack.Alarms) != 0 {
			policies := []string{}
			for _, scaling := range stack.Autoscaling {
				if len(scaling.Name) == 0 {
					return fmt.Errorf("autoscaling policy doesn't have a name.")
				}
				policies = append(policies, scaling.Name)
			}
			for _, alarm := range stack.Alarms {
				for _, action := range alarm.AlarmActions {
					if !tool.IsStringInArray(action, policies) {
						return fmt.Errorf("no scaling action exists : %s", action)
					}
				}
			}
		}

		// Check Spot Options
		if len(stack.InstanceMarketOptions.MarketType) != 0 {
			if stack.InstanceMarketOptions.MarketType != "spot" {
				return fmt.Errorf("no valid market type : %s", stack.InstanceMarketOptions.MarketType)
			}

			if stack.InstanceMarketOptions.SpotOptions.BlockDurationMinutes % 60 != 0 || stack.InstanceMarketOptions.SpotOptions.BlockDurationMinutes > 360 {
				return fmt.Errorf("block_duration_minutes should be one of [ 60, 120, 180, 240, 300, 360 ]")
			}

			if stack.InstanceMarketOptions.SpotOptions.SpotInstanceType == "persistent" && stack.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior == "terminate" {
				return fmt.Errorf("persistent type is not allowed with termiante behavior.")
			}
		}

		// Check block device setting
		if len(stack.BlockDevices) > 0 {
			dNames  := []string{}
			for _, block := range stack.BlockDevices {
				if len(block.DeviceName) == 0 {
					return fmt.Errorf("name of device is required.")
				}

				if ! tool.IsStringInArray(block.VolumeType, availableBlockTypes) {
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

		for _, region := range stack.Regions {
			//Check ami id
			if len(target_ami) == 0 && len(region.AmiId) == 0 {
				return fmt.Errorf("you have to specify at least one ami id.")
			}

			//Check instance type
			if len(region.InstanceType) == 0 {
				return fmt.Errorf("you have to specify the instance type.")
			}
		}

		// check mixed instances policy
		if stack.MixedInstancesPolicy.Enabled {
			if len(stack.MixedInstancesPolicy.SpotAllocationStrategy) ==0 {
				stack.MixedInstancesPolicy.SpotAllocationStrategy = DFEAULT_SPOT_ALLOCATION_STRATEGY
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
name       : %s
env        : %s
timeout    : %d
============================================================
Stacks
============================================================`
	summary = append(summary, fmt.Sprintf(formatting, b.AwsConfig.Name, b.Config.Env, b.Config.Timeout))

	for _, stack := range b.Stacks {
		if stack.Stack == target_stack {
			summary = append(summary, printEnvironment(stack))
		}
	}

	return strings.Join(summary, "\n")
}

func printEnvironment(stack Stack) string {
	formatting := `[ %s ]
Account             	: %s
Environment             : %s
IAM Instance Profile    : %s
Ansible tags            : %s 
Extra vars              : %s 
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
		stack.AnsibleTags,
		stack.extraTags,
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
func  parsingManifestFile(manifest string) (AWSConfig, []Stack) {
    yamlConfig := YamlConfig{}
	yamlFile, err := ioutil.ReadFile(manifest)
	if err != nil {
		Logger.Errorf("Error reading YAML file: %s\n", err)
		return AWSConfig{}, nil
	}

	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		tool.FatalError(err)
	}

	awsConfig := AWSConfig{
		Name:     yamlConfig.Name,
		Userdata: yamlConfig.Userdata,
		Tags:     yamlConfig.Tags,
	}

	Stacks := yamlConfig.Stacks

	return awsConfig, Stacks
}

// Parsing Config from command
func argumentParsing() Config {
	manifest := flag.String("manifest", "", "The manifest configuration file to use.")
	ami := flag.String("ami", "", "The AMI to use for the servers.")
	env := flag.String("env", "", "The environment that is being deployed into.")
	stack := flag.String("stack", "", "An ordered, comma-delimited list of stacks that should be deployed.")
	assumeRole := flag.String("assume_role", "", "The Role ARN to assume into")
	timeout := flag.Int64("timeout", 60, "Time in minutes to wait for deploy to finish before timing out")
	region := flag.String("region", "", "The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.")
	confirm := flag.Bool("confirm", true, "Suppress confirmation prompt")
	slackOff := flag.Bool("slack-off", false, "Turn off slack alarm")
	logLevel := flag.String("log-level", "info", "log level")
	extraTags := flag.String("extra-tags", "", "Extra tags to add to autoscaling group tags")

	flag.Parse()

	config := Config{
		Manifest: *manifest,
		Ami: *ami,
		Env: *env,
		Stack: *stack,
		Region: *region,
		AssumeRole: *assumeRole,
		Timeout: *timeout,
		StartTimestamp: time.Now().Unix(),
		Confirm: *confirm,
		SlackOff: *slackOff,
		LogLevel: *logLevel,
		ExtraTags: *extraTags,
	}

	return config
}

// Set Userdata provider
func SetUserdataProvider(userdata Userdata, default_userdata Userdata) UserdataProvider {

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
