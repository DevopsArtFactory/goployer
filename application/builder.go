package application

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

var (
	NO_MANIFEST_EXISTS="Manifest file does not exist"
)

type Builder struct {
	Config Config
	AwsConfig AWSConfig
}

type Config struct {
	Manifest 	string
	Ami  	 	string
	Env  	 	string
	Stack  	 	string
	AssumeRole 	string
	Timeout  	int
	Region   	string
	Confirm  	bool
}

type AWSConfig struct {
	Name string
	ReplacementType string `yaml:"replacement_type"`
	Environments []Environment
}

type AutoscalingPolicy struct {
	ScaleUp ScalePolicy `yaml:"scale_up"`
	ScaleDown ScalePolicy `yaml:"scale_down"`
}

type ScalePolicy struct {
	AdjustmentType string `yaml:"adjustment_type"`
	ScalingAdjustment int `yaml:"scaling_adjustment"`
	Cooldown string `yaml:"cooldown"`
}

type Alarms struct {
	ScaleUpOnUtil 	AlarmConfigs `yaml:"scale_up_on_util"`
	ScaleDownOnUtil	AlarmConfigs `yaml:"scale_down_on_util"`
}

type AlarmConfigs struct {
	Namespace string
	Metric string
	Statistic string
	Comparison string
	Threshold int
	Period int
	EvaluationPeriods int `yaml:"evaluation_periods"`
	AlarmActions []string `yaml:"alarm_actions"`
}

type Environment struct {
	Stack string
	Account string
	InstanceType string `yaml:"instance_type"`
	SshKey string `yaml:"ssh_key"`
	IamInstanceProfile string `yaml:"iam_instance_profile"`
	AnsibleTags string `yaml:"ansible_tags"`
	BlockDevices []BlockDevice `yaml:"block_devices"`
	ExtraVars string `yaml:"extra_vars"`
	Capacity Capacity `yaml:"capacity"`
	Autoscaling AutoscalingPolicy
	Alarms Alarms
	LifecycleCallbacks LifecycleCallbacks `yaml:"lifecycle_callbacks"`
	Regions []RegionConfig
}

type BlockDevice struct {
	DeviceName string `yaml:"device_name"`
	VolumeSize string `yaml:"volume_size"`
	VolumeType string `yaml:"volume_type"`
}

type LifecycleCallbacks struct {
	PreTerminatePastClusters []string `yaml:"pre_terminate_past_clusters"`
}

type RegionConfig struct {
	Region string `yaml:"region"`
	VPC string `yaml:"vpc"`
	SecurityGroups []string `yaml:"security_groups"`
	HealthcheckLB string `yaml:"healthcheck_load_balancer"`
	HealthcheckTargetGroup string `yaml:"healthcheck_target_group"`
	TargetGroups []string `yaml:"target_groups"`
}

type Capacity struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
	Desired int `yaml:"desired"`
}

func NewBuilder() Builder {
	// Parsing Argument
	config := _argument_parsing()
	awsConfig := _parsingManifestFile(config.Manifest)

	// Get New Builder
	builder := Builder{
		Config: config,
		AwsConfig: awsConfig,
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
Beginning deploy
============================================================
name       : %s
env        : %s
ami        : %s
region     : %s
timeout    : %d
============================================================
Stacks
============================================================`
	summary := fmt.Sprintf(formatting, b.AwsConfig.Name, b.Config.Env, b.Config.Ami, b.Config.Region, b.Config.Timeout)
	fmt.Println(summary)

	for _, environment := range b.AwsConfig.Environments {
		_print_environment(environment)
	}
}

func _print_environment(environment Environment)  {
	formatting := `[ %s ]
Account                 : %s
Instance type           : %s
SSH key                 : %s
IAM Instance Profile    : %s
Ansible tags            : %s 
Extra vars              : %s 
Capacity                : %+v 
Block_devices           : %+v
============================================================
	`
	summary := fmt.Sprintf(formatting, environment.Stack, environment.Account, environment.InstanceType, environment.SshKey, environment.IamInstanceProfile, environment.AnsibleTags, environment.ExtraVars, environment.Capacity, environment.BlockDevices)
	fmt.Println(summary)
}

// Parsing Manifest File
func  _parsingManifestFile(manifest string) AWSConfig {
    awsConfig := AWSConfig{}
	yamlFile, err := ioutil.ReadFile(manifest)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		error_logging("Please check the yaml file")
	}

	err = yaml.Unmarshal(yamlFile, &awsConfig)
	if err != nil {
		_fatal_error(err)
	}

	return awsConfig
}

// Parsing Config from command
func _argument_parsing() Config {
	manifest := flag.String("manifest", "", "The manifest configuration file to use.")
	ami := flag.String("ami", "", "The AMI to use for the servers.")
	env := flag.String("env", "", "The environment that is being deployed into.")
	stack := flag.String("stack", "", "An ordered, comma-delimited list of stacks that should be deployed.")
	assume_role := flag.String("assume_role", "", "The Role ARN to assume into")
	timeout := flag.Int("timeout", 60, "Time in minutes to wait for deploy to finish before timing out")
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
		Confirm: *confirm,
	}

	return config
}



