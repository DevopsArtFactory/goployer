package builder

import "time"

type Builder struct { // Do not add comments for this struct
	// Config from command
	Config Config

	// AWS related Configuration
	AwsConfig AWSConfig

	// Configuration for metrics
	MetricConfig MetricConfig

	// Stack configuration
	Stacks []Stack
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
	StartTimestamp        int64
	Confirm               bool
}

//Yaml configuration from manifest file
type YamlConfig struct {
	// Application Name
	Name string `yaml:"name"`

	// Configuration about userdata file
	Userdata Userdata `yaml:"userdata"`

	// Autoscaling tag list. This is attached to EC2 instance
	Tags []string `yaml:"tags"`

	// List of stack configuration
	Stacks []Stack `yaml:"stacks"`
}

type AWSConfig struct {
	Name     string
	Userdata Userdata
	Tags     []string
}

// Userdata configuration
type Userdata struct {
	// Type of storage that contains userdata
	Type string `yaml:"type"`

	// Path of userdata file
	Path string `yaml:"path"`
}

// Stack configuration
type Stack struct {
	// Name of stack
	Stack string `yaml:"stack"`

	// Name of AWS Account
	Account string `yaml:"account"`

	// Environment of stack
	Env string `yaml:"env"`

	// Type of Replacement for deployment
	ReplacementType string `yaml:"replacement_type"`

	// Userdata configuration for stack deployment
	Userdata Userdata `yaml:"userdata"`

	// AWS IAM instance profile.
	IamInstanceProfile string `yaml:"iam_instance_profile"`

	// Tags about ansible
	AnsibleTags string `yaml:"ansible_tags"`

	// IAM Role ARN for assume role
	AssumeRole string `yaml:"assume_role"`

	// Polling interval when health checking
	PollingInterval time.Duration `yaml:"polling_interval"`

	// Whether using EBS Optimized option or not
	EbsOptimized bool `yaml:"ebs_optimized"`

	// Instance market options like spot
	InstanceMarketOptions InstanceMarketOptions `yaml:"instance_market_options"`

	// MixedInstancePolicy of autoscaling group
	MixedInstancesPolicy MixedInstancesPolicy `yaml:"mixed_instances_policy,omitempty"`

	// EBS Block Devices for EC2 Instance
	BlockDevices []BlockDevice `yaml:"block_devices"`

	// Autoscaling Capacity
	Capacity Capacity `yaml:"capacity"`

	// Autoscaling Policy according to the metrics
	Autoscaling []ScalePolicy `yaml:"autoscaling"`

	// CloudWatch alarm for autoscaling action
	Alarms []AlarmConfigs `yaml:"alarms"`

	// List of commands which will be run before terminating instances
	LifecycleCallbacks LifecycleCallbacks `yaml:"lifecycle_callbacks"`

	// Lifecycle hooks of autoscaling group
	LifecycleHooks LifecycleHooks `yaml:"lifecycle_hooks"`

	// List of region configurations
	Regions []RegionConfig `yaml:"regions"`
}

// Instance Market Options Configuration
type InstanceMarketOptions struct {
	// Type of market for EC2 instance
	MarketType string `yaml:"market_type"`

	// Options for spot instance
	SpotOptions SpotOptions `yaml:"spot_options"`
}

// MixedInstancesPolicy of autoscaling group
type MixedInstancesPolicy struct {
	// Whether or not to use mixedInstancesPolicy
	Enabled bool `yaml:"enabled"`

	// List of EC2 instance types for spot instance
	Override []string `yaml:"override_instance_types"`

	// Percentage of On Demand instance
	OnDemandPercentage int64 `yaml:"on_demand_percentage"`

	// Allocation strategy for spot instances
	SpotAllocationStrategy string `yaml:"spot_allocation_strategy"`

	// The number of pools of instance type for spot instances
	SpotInstancePools int64 `yaml:"spot_instance_pools"`

	// Maximum number of spot
	SpotMaxPrice string `yaml:"spot_max_price,omitempty"`
}

// Spot configurations
type SpotOptions struct {
	// BlockDurationMinutes menas How long you want to use spot instance for sure
	BlockDurationMinutes int64 `yaml:"block_duration_minutes"`

	// Behavior when spot instance is interrupted
	InstanceInterruptionBehavior string `yaml:"instance_interruption_behavior"`

	// Maximum price of spot instance
	MaxPrice string `yaml:"max_price"`

	// Spot instance type
	SpotInstanceType string `yaml:"spot_instance_type"`
}

// EBS Block device configuration
type BlockDevice struct {
	// Name of block device
	DeviceName string `yaml:"device_name"`

	// Size of volume
	VolumeSize int64 `yaml:"volume_size"`

	// Type of volume (gp2, io1, st1, sc1)
	VolumeType string `yaml:"volume_type"`
}

// Lifecycle Callback configuration
type LifecycleCallbacks struct {
	// List of command before terminating previous autoscaling group
	PreTerminatePastClusters []string `yaml:"pre_terminate_past_cluster"`
}

// Policy of scaling policy
type ScalePolicy struct {
	// Name of scaling policy
	Name string `yaml:"name"`

	// Type of adjustment for autoscaling
	// https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-scaling-simple-step.html
	AdjustmentType string `yaml:"adjustment_type"`

	// Amount of adjustment for scaling
	ScalingAdjustment int64 `yaml:"scaling_adjustment"`

	// Cooldown time between scaling actions
	Cooldown int64 `yaml:"cooldown"`
}

// Configuration of CloudWatch alarm used with scaling policy
type AlarmConfigs struct {
	// Name of alarm
	Name string

	// Namespace of metrics
	Namespace string

	// Metrics type for scaling
	Metric string

	// Type of statistics for metrics
	Statistic string

	// Comparsion operator for triggering alarm
	Comparison string

	// Threshold of alarm trigger
	Threshold float64

	// Period for metrics
	Period int64

	// The number of periods for evaluation
	EvaluationPeriods int64 `yaml:"evaluation_periods"`

	// List of actions when alarm is triggered
	// Element of this list should be defined with scaling_policy
	AlarmActions []string `yaml:"alarm_actions"`
}

// Region configuration
type RegionConfig struct {
	// Region name
	Region string `yaml:"region"`

	// Whether or not to use public subnets
	UsePublicSubnets bool `yaml:"use_public_subnets"`

	// Type of EC2 instance
	InstanceType string `yaml:"instance_type"`

	// Key name of SSH access
	SshKey string `yaml:"ssh_key"`

	// Amazon AMI ID
	AmiId string `yaml:"ami_id"`

	// Name of VPC
	VPC string `yaml:"vpc"`

	// List of security group name
	SecurityGroups []string `yaml:"security_groups"`

	// Load balancer name for healthcheck
	HealthcheckLB string `yaml:"healthcheck_load_balancer"`

	// Target group name for healthcheck
	HealthcheckTargetGroup string `yaml:"healthcheck_target_group"`

	// Target group list of load balancer
	TargetGroups []string `yaml:"target_groups"`

	// List of  load balancers
	LoadBalancers []string `yaml:"loadbalancers"`

	// Availability zones for autoscaling group
	AvailabilityZones []string `yaml:"availability_zones"`
}

// Instance capacity of autoscaling group
type Capacity struct {
	// Minimum number of instances
	Min int64 `yaml:"min"`

	// Maximum number of instances
	Max int64 `yaml:"max"`

	// Desired number of instances
	Desired int64 `yaml:"desired"`
}

// Lifecycle Hooks
type LifecycleHooks struct {
	// Launch Transition configuration - triggered before starting instance
	LaunchTransition []LifecycleHookSpecification `yaml:"launch_transition"`

	// Terminate Transition configuration - triggered before terminating instance
	TerminateTransition []LifecycleHookSpecification `yaml:"terminate_transition"`
}

// Lifecycle Hook Specification
type LifecycleHookSpecification struct {
	// Name of lifecycle hook
	LifecycleHookName string `yaml:"lifecycle_hook_name"`

	// Default result of lifecycle hook
	DefaultResult string `yaml:"default_result"`

	// Heartbeat timeout of lifecycle hook
	HeartbeatTimeout int64 `yaml:"heartbeat_timeout"`

	// Notification Metadata of lifecycle hook
	NotificationMetadata string `yaml:"notification_metadata"`

	// Notification Target ARN like AWS Simple Notification Service
	NotificationTargetARN string `yaml:"notification_target_arn"`

	// IAM Role ARN for notification
	RoleARN string `yaml:"role_arn"`
}
