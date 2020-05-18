package application

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
     "encoding/base64"
	"time"
)

var (
	NO_MANIFEST_EXISTS="Manifest file does not exist"
)

type UserdataProvider interface {
	provide() string
}

type LocalProvider struct {
	Path string
}

type S3Provider struct {
	Path string
}

func (l LocalProvider) provide() string {
	if l.Path == "" {
		error_logging("Please specify userdata script path")
	}
	if ! fileExists(l.Path) {
		error_logging(fmt.Sprintf("File does not exist in %s", l.Path))
	}

	userdata, err := ioutil.ReadFile(l.Path)
	if err != nil {
		error_logging("Error reading userdata file")
	}

	return base64.StdEncoding.EncodeToString(userdata)
}

func (s S3Provider) provide() string  {
	return ""
}

type Builder struct {
	Config 		Config		// Config from command
	AwsConfig 	AWSConfig 	// Common Config
	Stacks 		Stacks 		// Stack Config
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
}


type YamlConfig struct {
	Name 			string
	Userdata 		Userdata 	`yaml:"userdata"`
	Tags 		 	[]string 	`yaml:"tags"`
	Stacks			[]Stack 	`yaml:"stacks"`
}

type AWSConfig struct {
	Name 			string
	Userdata 		Userdata
	Tags 		 	[]string
}

type Stacks struct {
	Stacks 	[]Stack
}

type Userdata struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type AutoscalingPolicy struct {
	ScaleUp 	ScalePolicy `yaml:"scale_up"`
	ScaleDown 	ScalePolicy `yaml:"scale_down"`
}

type ScalePolicy struct {
	AdjustmentType 		string `yaml:"adjustment_type"`
	ScalingAdjustment 	int `yaml:"scaling_adjustment"`
	Cooldown 			string `yaml:"cooldown"`
}

type Alarms struct {
	ScaleUpOnUtil 	AlarmConfigs `yaml:"scale_up_on_util"`
	ScaleDownOnUtil	AlarmConfigs `yaml:"scale_down_on_util"`
}

type AlarmConfigs struct {
	Namespace 			string
	Metric 				string
	Statistic 			string
	Comparison 			string
	Threshold 			int
	Period 				int
	EvaluationPeriods 	int `yaml:"evaluation_periods"`
	AlarmActions 		[]string `yaml:"alarm_actions"`
}

type Stack struct {
	Stack 				string `yaml:"stack"`
	Account 			string `yaml:"account"`
	Env 				string `yaml:"env"`
	ReplacementType 	string `yaml:"replacement_type"`
	InstanceType 		string `yaml:"instance_type"`
	SshKey 				string `yaml:"ssh_key"`
	Userdata 			Userdata `yaml:"userdata"`
	IamInstanceProfile 	string `yaml:"iam_instance_profile"`
	AnsibleTags 		string `yaml:"ansible_tags"`
	AssumeRole 			string `yaml:"assume_role"`
	EbsOptimized 		bool   `yaml:"ebs_optimized"`
	BlockDevices 		[]BlockDevice `yaml:"block_devices"`
	ExtraVars 			string `yaml:"extra_vars"`
	Capacity 			Capacity `yaml:"capacity"`
	Autoscaling 		AutoscalingPolicy
	Alarms 				Alarms
	LifecycleCallbacks 	LifecycleCallbacks `yaml:"lifecycle_callbacks"`
	Regions 			[]RegionConfig
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
	Region 					string `yaml:"region"`
	UsePublicSubnets		bool `yaml:"use_public_subnets"`
	VPC 					string `yaml:"vpc"`
	SecurityGroups 			[]string `yaml:"security_groups"`
	HealthcheckLB 			string `yaml:"healthcheck_load_balancer"`
	HealthcheckTargetGroup 	string `yaml:"healthcheck_target_group"`
	TargetGroups 			[]string `yaml:"target_groups"`
	LoadBalancers 			[]string `yaml:"loadbalancers"`
	AvailabilityZones 		[]string `yaml:"availability_zones"`
}

type Capacity struct {
	Min 	int64 `yaml:"min"`
	Max 	int64 `yaml:"max"`
	Desired int64 `yaml:"desired"`
}

func NewBuilder() Builder {
	// Parsing Argument
	config := _argument_parsing()
	awsConfig, Stacks := _parsingManifestFile(config.Manifest)

	// Get New Builder
	builder := Builder{
		Config: config,
		AwsConfig: awsConfig,
		Stacks: Stacks,
	}

	return builder
}

// Validation Check
func (b Builder) CheckValidation()  {
	//Check manifest file
	if len(b.Config.Manifest) == 0 || ! fileExists(b.Config.Manifest) {
		error_logging(NO_MANIFEST_EXISTS)
	}
}

// Print Summary
func (b Builder) PrintSummary() {
	formatting := `
============================================================
Target Stack Deployment Information
============================================================
name       : %s
ami        : %s
region     : %s
timeout    : %d
============================================================
Stacks
============================================================`
	summary := fmt.Sprintf(formatting, b.AwsConfig.Name, b.Config.Ami, b.Config.Region, b.Config.Timeout)
	fmt.Println(summary)

	for _, stack := range b.Stacks.Stacks {
		_print_environment(stack)
	}
}

func _print_environment(stack Stack)  {
	formatting := `[ %s ]
Environment             : %s
Environment             : %s
Instance type           : %s
SSH key                 : %s
IAM Instance Profile    : %s
Ansible tags            : %s 
Extra vars              : %s 
Capacity                : %+v 
Block_devices           : %+v
============================================================
	`
	summary := fmt.Sprintf(formatting, stack.Stack, stack.Account, stack.Env, stack.InstanceType, stack.SshKey, stack.IamInstanceProfile, stack.AnsibleTags, stack.ExtraVars, stack.Capacity, stack.BlockDevices)
	fmt.Println(summary)
}

// Parsing Manifest File
func  _parsingManifestFile(manifest string) (AWSConfig, Stacks) {
    yamlConfig := YamlConfig{}
	yamlFile, err := ioutil.ReadFile(manifest)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		error_logging("Please check the yaml file")
	}

	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		_fatal_error(err)
	}

	awsConfig := AWSConfig{
		Name:     yamlConfig.Name,
		Userdata: yamlConfig.Userdata,
		Tags:     yamlConfig.Tags,
	}

	Stacks := Stacks{Stacks: yamlConfig.Stacks}

	return awsConfig, Stacks
}

// Parsing Config from command
func _argument_parsing() Config {
	manifest := flag.String("manifest", "", "The manifest configuration file to use.")
	ami := flag.String("ami", "", "The AMI to use for the servers.")
	env := flag.String("env", "", "The environment that is being deployed into.")
	stack := flag.String("stack", "", "An ordered, comma-delimited list of stacks that should be deployed.")
	assume_role := flag.String("assume_role", "", "The Role ARN to assume into")
	timeout := flag.Int64("timeout", 60, "Time in minutes to wait for deploy to finish before timing out")
	region := flag.String("region", "us-east-2", "The region to deploy into, if undefined, then the deployment will run against all regions for the given environment.")
	confirm := flag.Bool("confirm", true, "Suppress confirmation prompt")

	flag.Parse()

	config := Config{
		Manifest: *manifest,
		Ami: *ami,
		Env: *env,
		Stack: *stack,
		Region: *region,
		AssumeRole: *assume_role,
		Timeout: *timeout,
		StartTimestamp: time.Now().Unix(),
		Confirm: *confirm,
	}

	return config
}

// Set Userdata provider
func _set_userdata_provider(userdata Userdata, default_userdata Userdata) UserdataProvider {

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

func (s Stacks) GetTargetStack(config Config) (bool, Stack, RegionConfig) {
	for _, stack := range s.Stacks {
		if stack.Stack == config.Stack {
			for _, region := range stack.Regions {
				if region.Region == config.Region {
					return true, stack, region
				}
			}
		}
	}
	// Null Value
	return false, Stack{}, RegionConfig{}
}

