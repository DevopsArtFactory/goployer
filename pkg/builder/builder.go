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
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/olekukonko/tablewriter"
	Logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v2"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/templates"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Builder struct { // Do not add comments for this struct
	// Config from command
	Config schemas.Config

	// AWS related Configuration
	AwsConfig schemas.AWSConfig

	// Configuration for metrics
	MetricConfig schemas.MetricConfig

	// Stack configuration
	Stacks []schemas.Stack

	// API Test configuration
	APITestTemplates []*schemas.APITestTemplate
}

var armTypeList = []string{"a1", "m6g", "m6gd", "t4g", "c6g", "c6gd", "c6gn", "r6g", "r6gd", "x2gd"}

type UserdataProvider interface {
	Provide() (string, error)
}

type LocalProvider struct {
	Path string
}

type S3Provider struct {
	Path string
}

const delimiterRegex = "[,/|!@$%^&*_=`~]+"

// Provide provides userdata from local file
func (l LocalProvider) Provide() (string, error) {
	if l.Path == "" {
		return constants.EmptyString, errors.New("please specify userdata script path")
	}
	if !tool.CheckFileExists(l.Path) {
		return constants.EmptyString, fmt.Errorf("file does not exist in %s", l.Path)
	}

	userdata, err := ioutil.ReadFile(l.Path)
	if err != nil {
		return constants.EmptyString, errors.New("error reading userdata file")
	}

	return base64.StdEncoding.EncodeToString(userdata), nil
}

// Provide provides userdata from s3
// Need to develop
func (s S3Provider) Provide() (string, error) {
	return constants.EmptyString, nil
}

// NewBuilder create new builder
func NewBuilder(config *schemas.Config) (Builder, error) {
	builder := Builder{}

	// parsing argument
	if config == nil {
		c, err := argumentParsing()
		if err != nil {
			return builder, err
		}
		config = &c
	}

	// set config
	builder.Config = *config

	return builder, nil
}

// SetManifestConfig set manifest configuration from local file
func (b Builder) SetManifestConfig() Builder {
	awsConfig, stacks, apiTestTemplates := ParsingManifestFile(b.Config.Manifest)
	b.AwsConfig = awsConfig

	if len(apiTestTemplates) > 0 {
		b.APITestTemplates = apiTestTemplates
	}

	return b.SetStacks(stacks)
}

// SetManifestConfigWithS3 set manifest configuration with s3
func (b Builder) SetManifestConfigWithS3(fileBytes []byte) Builder {
	awsConfig, stacks, apiTestTemplates := buildStructFromYaml(fileBytes)
	b.AwsConfig = awsConfig

	if len(apiTestTemplates) > 0 {
		b.APITestTemplates = apiTestTemplates
	}

	return b.SetStacks(stacks)
}

// SetStacks set stack information
func (b Builder) SetStacks(stacks []schemas.Stack) Builder {
	if len(b.Config.AssumeRole) > 0 {
		for i := range stacks {
			stacks[i].AssumeRole = b.Config.AssumeRole
		}
	}

	for i, stack := range stacks {
		if b.Config.PollingInterval > 0 {
			stacks[i].PollingInterval = b.Config.PollingInterval
		}

		stacks[i].ReplacementType = strings.ToLower(stack.ReplacementType)
		if stacks[i].ReplacementType == constants.RollingUpdateDeployment && stacks[i].RollingUpdateInstanceCount == 0 {
			stacks[i].RollingUpdateInstanceCount = 1
		}
	}

	b.Stacks = stacks

	return b
}

// CheckValidation validates all configurations
func (b Builder) CheckValidation() error {
	targetAmi := b.Config.Ami
	targetRegion := b.Config.Region

	// check configurations
	if len(b.AwsConfig.Tags) > 0 && HasProhibited(b.AwsConfig.Tags) {
		return fmt.Errorf("you cannot use prohibited tags : %s", strings.Join(constants.ProhibitedTags, ","))
	}

	if len(b.Config.Stack) > 0 {
		hasStack := false
		for _, stack := range b.Stacks {
			if stack.Stack == b.Config.Stack {
				hasStack = true
				break
			}
		}
		if !hasStack {
			return fmt.Errorf("stack does not exist: %s", b.Config.Stack)
		}
	}

	if len(b.AwsConfig.ScheduledActions) > 0 {
		for _, sa := range b.AwsConfig.ScheduledActions {
			if len(sa.Name) == 0 {
				return errors.New("you have to set name of scheduled action")
			}

			if len(sa.Recurrence) == 0 {
				return fmt.Errorf("recurrence is required field: %s", sa.Name)
			}

			if sa.Capacity == nil {
				return fmt.Errorf("capacity is required field: %s", sa.Name)
			}
		}
	}

	if len(b.Config.ExtraTags) > 0 && HasProhibited(strings.Split(b.Config.ExtraTags, ",")) {
		return fmt.Errorf("you cannot use prohibited tags : %s", strings.Join(constants.ProhibitedTags, ","))
	}

	// global AMI check
	if len(targetRegion) == 0 && len(targetAmi) != 0 && strings.HasPrefix(targetAmi, "ami-") {
		return fmt.Errorf("ami id cannot be used in different regions : %s", targetAmi)
	}

	// check release notes
	if len(b.Config.ReleaseNotes) > 0 && len(b.Config.ReleaseNotesBase64) > 0 {
		return errors.New("you cannot specify the release-notes and release-notes-base64 at the same time")
	}

	// check polling interval
	if b.Config.PollingInterval < constants.MinPollingInterval {
		return fmt.Errorf("polling interval cannot be smaller than %.0f sec", constants.MinPollingInterval.Seconds())
	}

	if b.Config.PollingInterval >= b.Config.Timeout {
		return fmt.Errorf("polling interval should be lower than %.0f min", b.Config.Timeout.Minutes())
	}

	// Check Configuration about metrics
	if !b.Config.DisableMetrics {
		if len(b.MetricConfig.Region) == 0 {
			return errors.New("you do not specify the region for metrics")
		}

		if len(b.MetricConfig.Storage.Name) == 0 {
			return errors.New("you do not specify the name of storage for metrics")
		}

		if !tool.CheckFileExists(constants.MetricYamlPath) {
			return fmt.Errorf("no %s file exists", constants.MetricYamlPath)
		}
	}

	overRideSpotInstanceType := b.Config.OverrideSpotType
	if len(overRideSpotInstanceType) > 0 {
		var delimiterCount = strings.Count(overRideSpotInstanceType, "|")
		spotInstanceTypes := regexp.MustCompile(delimiterRegex).Split(overRideSpotInstanceType, -1)
		spotInstanceTypeCount := len(spotInstanceTypes)
		if delimiterCount != spotInstanceTypeCount-1 {
			return errors.New("you must using delimiter '|'")
		}
		var armTypeCount = 0
		for _, spotInstanceType := range spotInstanceTypes {
			instanceTypeCategory := strings.Split(spotInstanceType, ".")
			if Contains(armTypeList, instanceTypeCategory[0]) {
				armTypeCount++
			}
		}
		if !(spotInstanceTypeCount == armTypeCount || armTypeCount == 0) {
			return errors.New("you can only use same type of spot instance type(arm64 and intel_x86 type)")
		}
	}
	// duplicated value check
	stackMap := map[string]int{}
	for _, stack := range b.Stacks {
		if stackMap[stack.Stack] >= 1 {
			return fmt.Errorf("duplicated stack key between stacks : %s", stack.Stack)
		}
		stackMap[stack.Stack]++
	}

	stackMap = map[string]int{}
	for _, stack := range b.Stacks {
		if stackMap[stack.Env] >= 1 {
			return fmt.Errorf("duplicated env between stacks : %s", stack.Env)
		}
		stackMap[stack.Env]++
	}

	// check validations in API test templates
	if b.APITestTemplates != nil && len(b.APITestTemplates) > 0 {
		for _, att := range b.APITestTemplates {
			if len(att.Name) == 0 {
				return errors.New("name of API test is required")
			}

			if att.Duration < constants.MinAPITestDuration {
				return fmt.Errorf("duration for api test cannot be smaller than %.0f seconds", constants.MinAPITestDuration.Seconds())
			}

			if att.RequestPerSecond == 0 {
				return errors.New("request per second should be specified")
			}

			for _, api := range att.APIs {
				if !tool.IsStringInArray(strings.ToUpper(api.Method), constants.AllowedRequestMethod) {
					return fmt.Errorf("api is not allowed: %s", api.Method)
				}

				if strings.ToUpper(api.Method) == "GET" && len(api.Body) > 0 {
					return errors.New("api with GET request cannot have body")
				}
			}
		}
	}

	// check validations in each stack
	for _, stack := range b.Stacks {
		if len(b.Config.Stack) > 0 && stack.Stack != b.Config.Stack {
			continue
		}

		if len(stack.Tags) > 0 && HasProhibited(stack.Tags) {
			return fmt.Errorf("you cannot use prohibited tags : %s", strings.Join(constants.ProhibitedTags, ","))
		}

		// Check AMI
		// Check Autoscaling and Alarm setting
		if len(stack.Autoscaling) != 0 && len(stack.Alarms) != 0 {
			policies := []string{}
			for _, scaling := range stack.Autoscaling {
				if len(scaling.Name) == 0 {
					return errors.New("autoscaling policy doesn't have a name")
				}
				policies = append(policies, scaling.Name)
			}
			for _, alarm := range stack.Alarms {
				if len(alarm.Name) == 0 {
					return errors.New("cloudwatch alarm doesn't have a name")
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
				return errors.New("block_duration_minutes should be one of [ 60, 120, 180, 240, 300, 360 ]")
			}

			if stack.InstanceMarketOptions.SpotOptions.SpotInstanceType == "persistent" && stack.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior == "terminate" {
				return errors.New("persistent type is not allowed with terminate behavior")
			}
		}

		// Check block device setting
		if len(stack.BlockDevices) > 0 {
			dNames := []string{}
			for _, block := range stack.BlockDevices {
				if len(block.DeviceName) == 0 {
					return errors.New("name of device is required")
				}

				if !tool.IsStringInArray(block.VolumeType, constants.AvailableBlockTypes) {
					return fmt.Errorf("not available volume type : %s", block.VolumeType)
				}

				if tool.IsStringInArray(block.VolumeType, []string{"gp2", "gp3"}) && block.VolumeSize < 1 {
					return errors.New("volume size of gp2 or gp3 type should be larger than 1GiB")
				}

				if tool.IsStringInArray(block.VolumeType, constants.IopsRequiredBlockType) && block.VolumeSize < 4 {
					return errors.New("volume size of io1 and io2 type should be larger than 4GiB")
				}

				if tool.IsStringInArray(block.VolumeType, constants.IopsRequiredBlockType) && block.Iops < 100 {
					return errors.New("iops of io1 and io2 type should be larger than 100")
				}

				if block.VolumeType == "st1" && block.VolumeSize < 500 {
					return errors.New("volume size of st1 type should be larger than 500GiB")
				}

				if tool.IsStringInArray(block.DeviceName, dNames) {
					return fmt.Errorf("device names are duplicated : %s", block.DeviceName)
				}
				dNames = append(dNames, block.DeviceName)
			}
		}

		if stack.LifecycleHooks != nil {
			if len(stack.LifecycleHooks.LaunchTransition) > 0 {
				for _, l := range stack.LifecycleHooks.LaunchTransition {
					if len(l.NotificationTargetARN) > 0 && len(l.RoleARN) == 0 {
						return fmt.Errorf("role_arn is needed if notification_target_arn is not empty : %s", l.LifecycleHookName)
					}

					if len(l.RoleARN) > 0 && len(l.NotificationTargetARN) == 0 {
						return fmt.Errorf("notification_target_arn is needed if role_arn is not empty: %s", l.LifecycleHookName)
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
						return fmt.Errorf("notification_target_arn is needed if role_arn is not empty: %s", l.LifecycleHookName)
					}

					if l.HeartbeatTimeout == 0 {
						Logger.Warnf("you didn't specify the heartbeat timeout. you might have to wait too long time.")
					}
				}
			}
		}

		if stack.ReplacementType == constants.BlueGreenDeployment {
			if stack.TerminationDelayRate > 100 {
				return fmt.Errorf("termination_delay_rate cannot exceed 100. It should be 0<=x<=100")
			}

			if stack.TerminationDelayRate < 0 {
				return fmt.Errorf("termination_delay_rate cannot be negative. It should be 0<=x<=100")
			}
		}

		for _, region := range stack.Regions {
			// Check ami id
			if len(targetAmi) == 0 && len(region.AmiID) == 0 {
				return errors.New("you have to specify at least one ami id")
			}

			// Check instance type
			if len(region.InstanceType) == 0 {
				return errors.New("you have to specify the instance type")
			}

			// Check target group
			if len(region.TargetGroups) > 0 && region.HealthcheckTargetGroup == "" {
				return errors.New("you have to choose one target group as healthcheck_target_group")
			}

			// Check load balancer
			if len(region.LoadBalancers) > 0 && region.HealthcheckLB == "" {
				return errors.New("you have to choose one load balancer as healthcheck_load_balancer")
			}

			// Check load balancer
			if region.HealthcheckLB != "" && len(region.TargetGroups) > 0 {
				return errors.New("you cannot use healthcheck_load_balancer with target_groups")
			}

			// Check load balancer and target group
			if region.HealthcheckLB != "" && region.HealthcheckTargetGroup != "" {
				return errors.New("you cannot use healthcheck_target_group and healthcheck_load_balancer at the same time")
			}

			// Check userdata
			if stack.Userdata.Type == "local" && len(stack.Userdata.Path) > 0 && !tool.CheckFileExists(stack.Userdata.Path) {
				return errors.New("script file does not exists")
			}

			// Check scheduled actions
			if len(region.ScheduledActions) > 0 {
				for _, sa := range region.ScheduledActions {
					if !ContainsActions(sa, b.AwsConfig.ScheduledActions) {
						return fmt.Errorf("scheduled action is not defined: %s", sa)
					}
				}

				for _, sa := range b.AwsConfig.ScheduledActions {
					if tool.IsStringInArray(sa.Name, region.ScheduledActions) {
						if isValid, err := ValidCronExpression(sa.Recurrence); !isValid {
							return err
						}
					}
				}
			}
		}

		if stack.MixedInstancesPolicy.Enabled {
			if len(stack.MixedInstancesPolicy.SpotAllocationStrategy) == 0 {
				stack.MixedInstancesPolicy.SpotAllocationStrategy = constants.DefaultSpotAllocationStrategy
			}

			if stack.MixedInstancesPolicy.SpotAllocationStrategy != "lowest-price" && stack.MixedInstancesPolicy.SpotInstancePools > 0 {
				return errors.New("you can only set spot_instance_pools with lowest-price spot_allocation_strategy")
			}

			if len(stack.MixedInstancesPolicy.Override) == 0 {
				return errors.New("you have to set at least one instance type to use in override")
			}
		}

		if stack.APITestEnabled {
			if len(stack.APITestTemplate) == 0 {
				return fmt.Errorf("you have to specify the name of template for api test: %s", stack.Stack)
			}

			isExist := false
			for _, att := range b.APITestTemplates {
				if att.Name == stack.APITestTemplate {
					isExist = true
					break
				}
			}

			if !isExist {
				return fmt.Errorf("template does not exist in the list: %s", stack.APITestTemplate)
			}
		}
	}

	return nil
}

func Contains(items []string, target string) bool {
	for _, element := range items {
		if target == element {
			return true
		}
	}
	return false
}

// MakeSummary prints all configurations in summary
func (b Builder) PrintSummary(out io.Writer, targetStack, targetRegion string) error {
	configStr := &strings.Builder{}
	table := tablewriter.NewWriter(configStr)
	table.SetHeader([]string{"Configuration", "Value"})
	table.SetCenterSeparator("|")
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)

	data := ExtractAppliedConfig(b.Config)

	table.AppendBulk(data)
	table.Render()

	// Print Stack Information
	var ts []schemas.Stack
	for _, stack := range b.Stacks {
		if stack.Stack == targetStack {
			ts = append(ts, stack)
			break
		}
	}

	if len(ts) == 0 {
		ts = b.Stacks
	}

	var deploymentData = struct {
		Stacks        []schemas.Stack
		Region        string
		ConfigSummary string
	}{
		Stacks:        ts,
		Region:        targetRegion,
		ConfigSummary: configStr.String(),
	}

	funcMap := template.FuncMap{
		"decorate":   tool.DecorateAttr,
		"joinString": tool.JoinString,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Stack Information").Funcs(funcMap).Parse(templates.DeploymentSummary))

	err := t.Execute(w, deploymentData)
	if err != nil {
		return err
	}

	return nil
}

// Parsing Manifest File
func ParsingManifestFile(manifest string) (schemas.AWSConfig, []schemas.Stack, []*schemas.APITestTemplate) {
	var yamlFile []byte
	var err error

	yamlFile, err = ioutil.ReadFile(manifest)
	if err != nil {
		Logger.Errorf("Error reading YAML file: %s\n", err)
		return schemas.AWSConfig{}, nil, nil
	}

	return buildStructFromYaml(yamlFile)
}

// buildStructFromYaml creates custom structure from manifest
func buildStructFromYaml(yamlFile []byte) (schemas.AWSConfig, []schemas.Stack, []*schemas.APITestTemplate) {
	yamlConfig := schemas.YamlConfig{}
	err := yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		tool.FatalError(err)
	}

	awsConfig := schemas.AWSConfig{
		Name:             yamlConfig.Name,
		Userdata:         yamlConfig.Userdata,
		Tags:             yamlConfig.Tags,
		ScheduledActions: yamlConfig.ScheduledActions,
	}

	Stacks := yamlConfig.Stacks

	return awsConfig, Stacks, yamlConfig.APITestTemplates
}

// argumentParsing parses arguments from command
func argumentParsing() (schemas.Config, error) {
	keys := viper.AllKeys()
	config := schemas.Config{}

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
				case reflect.Int:
					t.SetInt(viper.GetInt64(key))
				case reflect.Int64: // should use int64 not, int
					if tool.IsStringInArray(key, constants.TimeFields) {
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
func SetUserdataProvider(userdata schemas.Userdata, defaultUserdata schemas.Userdata) UserdataProvider {
	//Set default if no userdata exists in the stack
	if userdata.Type == "" {
		userdata.Type = defaultUserdata.Type
	}

	if userdata.Path == "" {
		userdata.Path = defaultUserdata.Path
	}

	if userdata.Type == "s3" {
		return S3Provider{Path: userdata.Path}
	}

	return LocalProvider{
		Path: userdata.Path,
	}
}

// PreConfigValidation validates manifest existence
func (b Builder) PreConfigValidation() error {
	// check manifest file
	if len(b.Config.Manifest) == 0 {
		return errors.New("you should specify manifest file")
	}

	if strings.HasPrefix(b.Config.Manifest, constants.S3Prefix) && len(b.Config.ManifestS3Region) == 0 {
		return errors.New("you have to specify region of s3 bucket: --manifest-s3-region")
	}

	if len(b.Config.Manifest) == 0 || (!strings.HasPrefix(b.Config.Manifest, constants.S3Prefix) && !tool.CheckFileExists(b.Config.Manifest)) {
		return errors.New(constants.NoManifestFileExists)
	}

	return nil
}

// RefineConfig refines the values for clear setting
func RefineConfig(config schemas.Config) (schemas.Config, error) {
	if config.Timeout < time.Minute {
		config.Timeout *= time.Minute
	}

	if config.PollingInterval < time.Second {
		config.PollingInterval *= time.Second
	}

	config.StartTimestamp = time.Now().Unix()

	if len(config.Region) == 0 {
		regionConfig, err := setDefaultRegion("default")
		if err != nil {
			return config, err
		}

		config.Region = regionConfig
	}

	return config, nil
}

// HasProhibited checks if there is any prohibited tags
func HasProhibited(tags []string) bool {
	for _, t := range tags {
		arr := strings.Split(t, "=")
		k := arr[0]

		if tool.IsStringInArray(k, constants.ProhibitedTags) {
			return true
		}
	}

	return false
}

// ContainsActions checks if scheduled action is specified
func ContainsActions(target string, sas []schemas.ScheduledAction) bool {
	for _, sa := range sas {
		if sa.Name == target {
			return true
		}
	}
	return false
}

// ValidCronExpression checks if the cron expression is valid or not
// It should be [Minute] [Hour] [Day_of_Month] [Month_of_Year] [Day_of_Week]
func ValidCronExpression(expression string) (bool, error) {
	elements := strings.Split(expression, " ")
	if len(elements) != 5 {
		return false, fmt.Errorf("cron expression should be in the format of [Minute] [Hour] [Day_of_Month] [Month_of_Year] [Day_of_Week]: %s", expression)
	}

	// minutes
	if len(elements[0]) > 0 && elements[0] != "*" {
		i, _ := strconv.Atoi(elements[0])
		if i < 0 || i > 59 {
			return false, fmt.Errorf("first element should be from 0 to 59 or *: %s", expression)
		}
	}

	// hours
	if len(elements[1]) > 0 && elements[1] != "*" {
		hours := strings.Split(elements[1], ",")
		for _, h := range hours {
			i, _ := strconv.Atoi(h)
			if i < 0 || i > 23 {
				return false, fmt.Errorf("second element should be from 0 to 23 or *: %s", expression)
			}
		}
	}

	// day of month
	if len(elements[2]) > 0 && elements[2] != "*" {
		days := strings.Split(elements[2], ",")
		for _, d := range days {
			i, _ := strconv.Atoi(d)
			if i < 1 || i > 31 {
				return false, fmt.Errorf("third element should be from 1 to 31 or *: %s", expression)
			}
		}
	}

	// month of year
	if len(elements[3]) > 0 && elements[3] != "*" {
		months := strings.Split(elements[3], ",")
		for _, m := range months {
			i, _ := strconv.Atoi(m)
			if i < 1 || i > 12 {
				return false, fmt.Errorf("fourth element should be from 1 to 12 or *: %s", expression)
			}
		}
	}

	// day of week
	if len(elements[4]) > 0 && elements[4] != "*" {
		dayOfWeeks := strings.Split(elements[4], ",")
		for _, dow := range dayOfWeeks {
			singleDay := strings.Split(dow, "-")
			if len(singleDay) > 2 {
				return false, fmt.Errorf("fifth element should be single day of week or combination of two, not more than two: %s", expression)
			}
			for _, s := range singleDay {
				if !tool.IsStringInArray(s, constants.DaysOfWeek) {
					return false, fmt.Errorf("fifth element format error: %s", expression)
				}
			}
		}
	}

	return true, nil
}

// setDefaultRegion gets default region with env or configuration file
func setDefaultRegion(profile string) (string, error) {
	if len(os.Getenv(constants.DefaultRegionVariable)) > 0 {
		return os.Getenv(constants.DefaultRegionVariable), nil
	}

	functions := []func() (*ini.File, error){
		ReadAWSCredentials,
		ReadAWSConfig,
	}

	for _, f := range functions {
		cfg, err := f()
		if err != nil {
			return constants.EmptyString, err
		}

		section, err := cfg.GetSection(profile)
		if err != nil {
			return constants.EmptyString, err
		}

		if _, err := section.GetKey("region"); err == nil && len(section.Key("region").String()) > 0 {
			return section.Key("region").String(), nil
		}
	}
	return constants.EmptyString, errors.New("no aws region configuration exists")
}

// ReadAWSCredentials parse an aws credentials
func ReadAWSCredentials() (*ini.File, error) {
	if !tool.CheckFileExists(constants.AWSCredentialsPath) {
		return ReadAWSConfig()
	}

	cfg, err := ini.Load(constants.AWSCredentialsPath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ReadAWSConfig parse an aws configuration
func ReadAWSConfig() (*ini.File, error) {
	if !tool.CheckFileExists(constants.AWSConfigPath) {
		return nil, fmt.Errorf("no aws configuration file exists in $HOME/%s", constants.AWSConfigPath)
	}

	cfg, err := ini.Load(constants.AWSConfigPath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ExtractAppliedConfig extracts configurations that are used in the deployment
func ExtractAppliedConfig(config schemas.Config) [][]string {
	keys := viper.AllKeys()

	var data [][]string
	val := reflect.ValueOf(&config).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		key := strings.ReplaceAll(typeField.Tag.Get("json"), "_", "-")
		if tool.IsStringInArray(key, keys) {
			t := val.FieldByName(typeField.Name)
			if t.CanSet() {
				switch t.Kind() {
				case reflect.String:
					if len(val.FieldByName(typeField.Name).String()) > 0 {
						data = append(data, []string{key, val.FieldByName(typeField.Name).String()})
					}
				case reflect.Int, reflect.Int64:
					if val.FieldByName(typeField.Name).Int() > 0 {
						if tool.IsStringInArray(key, []string{"polling-interval", "timeout"}) {
							data = append(data, []string{key, fmt.Sprintf("%.0fs", time.Duration(val.FieldByName(typeField.Name).Int()).Seconds())})
						} else {
							data = append(data, []string{key, fmt.Sprintf("%d", val.FieldByName(typeField.Name).Int())})
						}
					}
				case reflect.Bool:
					data = append(data, []string{key, fmt.Sprintf("%t", val.FieldByName(typeField.Name).Bool())})
				}
			}
		}
	}

	return data
}
