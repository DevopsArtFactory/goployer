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
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type BlueGreen struct {
	Deployer
}

// NewBlueGrean creates new BlueGreen deployment deployer
func NewBlueGrean(mode string, logger *Logger.Logger, awsConfig schemas.AWSConfig, stack schemas.Stack, regionSelected string) BlueGreen {
	awsClients := []aws.Client{}
	for _, region := range stack.Regions {
		if len(regionSelected) > 0 && regionSelected != region.Region {
			Logger.Debugf("skip creating aws clients in %s region", region.Region)
			continue
		}
		awsClients = append(awsClients, aws.BootstrapServices(region.Region, stack.AssumeRole))
	}
	return BlueGreen{
		Deployer{
			Mode:              mode,
			Logger:            logger,
			AwsConfig:         awsConfig,
			AWSClients:        awsClients,
			AsgNames:          map[string]string{},
			PrevAsgs:          map[string][]string{},
			PrevInstances:     map[string][]string{},
			PrevInstanceCount: map[string]schemas.Capacity{},
			PrevVersions:      map[string][]int{},
			Stack:             stack,
			StepStatus: map[int64]bool{
				constants.StepCheckPrevious:            false,
				constants.StepDeploy:                   false,
				constants.StepAdditionalWork:           false,
				constants.StepTriggerLifecycleCallback: false,
				constants.StepCleanPreviousVersion:     false,
			},
		},
	}
}

// Deploy function
func (b BlueGreen) Deploy(config builder.Config) error {
	if !b.StepStatus[constants.StepCheckPrevious] {
		return nil
	}

	b.Logger.Info("Deploy Mode is " + b.Mode)

	//Get LocalFileProvider
	b.LocalProvider = builder.SetUserdataProvider(b.Stack.Userdata, b.AwsConfig.Userdata)

	// Make Frigga
	frigga := tool.Frigga{}
	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			return err
		}

		//Setup frigga with prefix
		frigga.Prefix = tool.BuildPrefixName(b.AwsConfig.Name, b.Stack.Env, region.Region)

		// Get Current Version
		curVersion := getCurrentVersion(b.PrevVersions[region.Region])
		b.Logger.Info("Current Version :", curVersion)

		//Get AMI
		var ami string
		if len(config.Ami) > 0 {
			ami = config.Ami
		} else {
			ami = region.AmiID
		}

		// Generate new name for autoscaling group and launch configuration
		newAsgName := tool.GenerateAsgName(frigga.Prefix, curVersion)
		launchTemplateName := tool.GenerateLcName(newAsgName)

		userdata, err := (b.LocalProvider).Provide()
		if err != nil {
			return err
		}

		//Stack check
		securityGroups, err := client.EC2Service.GetSecurityGroupList(region.VPC, region.SecurityGroups)
		if err != nil {
			return err
		}
		blockDevices := client.EC2Service.MakeLaunchTemplateBlockDeviceMappings(b.Stack.BlockDevices)
		ebsOptimized := b.Stack.EbsOptimized

		// Instance Type Override
		instanceType := region.InstanceType
		if len(config.OverrideInstanceType) > 0 {
			instanceType = config.OverrideInstanceType

			if b.Stack.MixedInstancesPolicy.Enabled {
				Logger.Warnf("--override-instance-type won't be applied because mixed_instances_policy is enabled")
			}
		}

		// LaunchTemplate
		err = client.EC2Service.CreateNewLaunchTemplate(
			launchTemplateName,
			ami,
			instanceType,
			region.SSHKey,
			b.Stack.IamInstanceProfile,
			userdata,
			ebsOptimized,
			b.Stack.MixedInstancesPolicy.Enabled,
			securityGroups,
			blockDevices,
			b.Stack.InstanceMarketOptions,
			region.DetailedMonitoringEnabled,
		)

		if err != nil {
			return errors.New("unknown error happened creating new launch template")
		}

		healthElb := region.HealthcheckLB
		loadbalancers := region.LoadBalancers
		if healthElb != "" && !tool.IsStringInArray(healthElb, loadbalancers) {
			loadbalancers = append(loadbalancers, healthElb)
		}

		healthcheckTargetGroup := region.HealthcheckTargetGroup
		targetGroups := region.TargetGroups
		if healthcheckTargetGroup != "" && !tool.IsStringInArray(healthcheckTargetGroup, targetGroups) {
			targetGroups = append(targetGroups, healthcheckTargetGroup)
		}

		terminationPolicies := []*string{}
		usePublicSubnets := region.UsePublicSubnets
		healthcheckType := constants.DefaultHealthcheckType
		healthcheckGracePeriod := int64(constants.DefaultHealthcheckGracePeriod)
		availabilityZones, err := client.EC2Service.GetAvailabilityZones(region.VPC, region.AvailabilityZones)
		if err != nil {
			return err
		}
		tags := client.EC2Service.GenerateTags(b.AwsConfig.Tags, newAsgName, b.AwsConfig.Name, config.Stack, b.Stack.AnsibleTags, b.Stack.Tags, config.ExtraTags, config.AnsibleExtraVars, region.Region)
		subnets, err := client.EC2Service.GetSubnets(region.VPC, usePublicSubnets, availabilityZones)
		if err != nil {
			return err
		}
		targetGroupArns, err := client.ELBV2Service.GetTargetGroupARNs(targetGroups)
		if err != nil {
			return err
		}

		var appliedCapacity schemas.Capacity
		if !config.ForceManifestCapacity && b.PrevInstanceCount[region.Region].Desired > b.Stack.Capacity.Desired {
			appliedCapacity = b.PrevInstanceCount[region.Region]
			b.Logger.Infof("Current desired instance count is larger than the number of instances in manifest file")
		} else {
			appliedCapacity = b.Stack.Capacity
		}

		b.Logger.Infof("Applied instance capacity - Min: %d, Desired: %d, Max: %d", appliedCapacity.Min, appliedCapacity.Desired, appliedCapacity.Max)

		var lifecycleHooksSpecificationList []*autoscaling.LifecycleHookSpecification
		if b.Stack.LifecycleHooks != nil {
			lifecycleHooksSpecificationList = client.EC2Service.GenerateLifecycleHooks(*b.Stack.LifecycleHooks)
		}

		_, err = client.EC2Service.CreateAutoScalingGroup(
			newAsgName,
			launchTemplateName,
			healthcheckType,
			healthcheckGracePeriod,
			appliedCapacity,
			loadbalancers,
			availabilityZones,
			targetGroupArns,
			terminationPolicies,
			tags,
			subnets,
			b.Stack.MixedInstancesPolicy,
			lifecycleHooksSpecificationList,
		)

		if err != nil {
			return err
		}

		if b.Collector.MetricConfig.Enabled {
			additionalFields := map[string]string{}
			if len(config.ReleaseNotes) > 0 {
				additionalFields["release-notes"] = config.ReleaseNotes
			}

			if len(config.ReleaseNotesBase64) > 0 {
				additionalFields["release-notes-base64"] = config.ReleaseNotesBase64
			}

			if len(userdata) > 0 {
				additionalFields["userdata"] = userdata
			}

			b.Collector.StampDeployment(b.Stack, config, tags, newAsgName, "creating", additionalFields)
		}

		b.AsgNames[region.Region] = newAsgName
		b.Stack.Capacity.Desired = appliedCapacity.Desired
	}

	b.StepStatus[constants.StepDeploy] = true
	return nil
}

// Healthchecking
func (b BlueGreen) HealthChecking(config builder.Config) map[string]bool {
	isUpdate := len(config.TargetAutoscalingGroup) > 0
	stackName := b.GetStackName()
	if !b.StepStatus[constants.StepDeploy] && !isUpdate {
		return map[string]bool{stackName: true}
	}
	b.Logger.Debugf("Healthchecking for stack starts : %s", stackName)
	finished := []string{}

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(b.Stack.Regions)
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			validCount = 0
		}
	}

	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debugf("This region is skipped by user: %s", region.Region)
			continue
		}

		b.Logger.Debugf("Healthchecking for region starts: %s", region.Region)

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			return map[string]bool{stackName: false, "error": true}
		}

		var targetAsgName string
		if len(config.TargetAutoscalingGroup) > 0 {
			targetAsgName = config.TargetAutoscalingGroup
		} else {
			targetAsgName = b.AsgNames[region.Region]
		}
		asg, err := client.EC2Service.GetMatchingAutoscalingGroup(targetAsgName)
		if err != nil {
			return map[string]bool{stackName: false, "error": true}
		}

		isHealthy, err := b.Deployer.polling(region, asg, client, config.ForceManifestCapacity, isUpdate, config.DownSizingUpdate)
		if err != nil {
			return map[string]bool{stackName: false, "error": true}
		}

		if isHealthy {
			if b.Collector.MetricConfig.Enabled {
				if err := b.Collector.UpdateStatus(*asg.AutoScalingGroupName, "deployed", nil); err != nil {
					Logger.Errorf("Update status Error, %s : %s", err.Error(), *asg.AutoScalingGroupName)
				}
			}
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return map[string]bool{stackName: true, "error": false}
	}

	return map[string]bool{stackName: false, "error": false}
}

// GetStackName returns name of stack
func (b BlueGreen) GetStackName() string {
	return b.Stack.Stack
}

// FinishAdditionalWork processes final work
func (b BlueGreen) FinishAdditionalWork(config builder.Config) error {
	if !b.StepStatus[constants.StepDeploy] {
		return nil
	}

	skipped := false
	if len(config.Region) > 0 && !CheckRegionExist(config.Region, b.Stack.Regions) {
		skipped = true
	}

	if !skipped {
		//Apply AutoScaling Policies
		for _, region := range b.Stack.Regions {
			//If region id is passed from command line, then deployer will deploy in that region only.
			if config.Region != "" && config.Region != region.Region {
				b.Logger.Debug("This region is skipped by user : " + region.Region)
				continue
			}

			b.Logger.Info("Attaching autoscaling policies : " + region.Region)

			//select client
			client, err := selectClientFromList(b.AWSClients, region.Region)
			if err != nil {
				return err
			}

			if len(b.Stack.Autoscaling) == 0 {
				b.Logger.Debug("no scaling policy exists")
			} else {
				//putting autoscaling group policies
				policyArns := map[string]string{}
				for _, policy := range b.Stack.Autoscaling {
					policyArn, err := client.EC2Service.CreateScalingPolicy(policy, b.AsgNames[region.Region])
					if err != nil {
						return err
					}
					b.Logger.Debugf("policy arn created: %s", *policyArn)
					policyArns[policy.Name] = *policyArn
				}

				if err := client.EC2Service.EnableMetrics(b.AsgNames[region.Region]); err != nil {
					return err
				}

				if err := client.CloudWatchService.CreateScalingAlarms(b.AsgNames[region.Region], b.Stack.Alarms, policyArns); err != nil {
					return err
				}
			}

			if len(region.ScheduledActions) > 0 {
				b.Logger.Debugf("create scheduled actions")
				selectedActions := []schemas.ScheduledAction{}
				for _, sa := range b.AwsConfig.ScheduledActions {
					if tool.IsStringInArray(sa.Name, region.ScheduledActions) {
						selectedActions = append(selectedActions, sa)
					}
				}

				b.Logger.Debugf("selected actions [ %s ]", strings.Join(region.ScheduledActions, ","))
				if err := client.EC2Service.CreateScheduledActions(b.AsgNames[region.Region], selectedActions); err != nil {
					return err
				}
				b.Logger.Debugf("finished adding scheduled actions")
			}
		}
	}

	Logger.Debug("Finish additional works.")
	b.StepStatus[constants.StepAdditionalWork] = true
	return nil
}

// TriggerLifecycleCallbacks runs lifecycle callbacks before cleaning.
func (b BlueGreen) TriggerLifecycleCallbacks(config builder.Config) error {
	if !b.StepStatus[constants.StepAdditionalWork] {
		return nil
	}

	skipped := false
	if b.Stack.LifecycleCallbacks == nil || len(b.Stack.LifecycleCallbacks.PreTerminatePastClusters) == 0 {
		b.Logger.Debugf("no lifecycle callbacks in %s\n", b.Stack.Stack)
		skipped = true
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			b.Logger.Debugf("region [ %s ] is not in the stack [ %s ].", config.Region, b.Stack.Stack)
			skipped = true
		}
	}

	if !skipped {
		for _, region := range b.Stack.Regions {
			if config.Region != "" && config.Region != region.Region {
				b.Logger.Debug("This region is skipped by user : " + region.Region)
				continue
			}

			//select client
			client, err := selectClientFromList(b.AWSClients, region.Region)
			if err != nil {
				return err
			}

			if len(b.PrevInstances[region.Region]) > 0 {
				b.Deployer.RunLifecycleCallbacks(client, b.PrevInstances[region.Region])
			} else {
				b.Logger.Infof("No previous versions to be deleted : %s\n", region.Region)
				b.Slack.SendSimpleMessage(fmt.Sprintf("No previous versions to be deleted : %s\n", region.Region), b.Stack.Env)
			}
		}
	}
	b.StepStatus[constants.StepTriggerLifecycleCallback] = true
	return nil
}

//Clean Previous Version
func (b BlueGreen) CleanPreviousVersion(config builder.Config) error {
	if !b.StepStatus[constants.StepTriggerLifecycleCallback] {
		return nil
	}
	b.Logger.Debug("Delete Mode is " + b.Mode)

	skipped := false
	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			skipped = true
		}
	}

	if !skipped {
		for _, region := range b.Stack.Regions {
			if config.Region != "" && config.Region != region.Region {
				b.Logger.Debug("This region is skipped by user : " + region.Region)
				continue
			}

			b.Logger.Infof("[%s]The number of previous versions to delete is %d", region.Region, len(b.PrevAsgs[region.Region]))

			//select client
			client, err := selectClientFromList(b.AWSClients, region.Region)
			if err != nil {
				return err
			}

			// First make autoscaling group size to 0
			if len(b.PrevAsgs[region.Region]) > 0 {
				for _, asg := range b.PrevAsgs[region.Region] {
					b.Logger.Debugf("[Resizing to 0] target autoscaling group : %s", asg)
					if err := b.ResizingAutoScalingGroupToZero(client, b.Stack.Stack, asg); err != nil {
						b.Logger.Errorf(err.Error())
					}
				}
			} else {
				b.Logger.Infof("No previous versions to be deleted : %s\n", region.Region)
				b.Slack.SendSimpleMessage(fmt.Sprintf("No previous versions to be deleted : %s\n", region.Region), b.Stack.Env)
			}
		}
	}
	b.StepStatus[constants.StepCleanPreviousVersion] = true
	return nil
}

// TerminateChecking checks Termination status
func (b BlueGreen) TerminateChecking(config builder.Config) map[string]bool {
	stackName := b.GetStackName()
	if !b.StepStatus[constants.StepCleanPreviousVersion] {
		return map[string]bool{stackName: true}
	}
	b.Logger.Info(fmt.Sprintf("Termination Checking for %s starts...", stackName))

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(b.Stack.Regions)
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			validCount = 0
		}
	}

	finished := []string{}
	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		b.Logger.Info("Checking Termination stack for region starts : " + region.Region)

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			return map[string]bool{
				stackName: false,
			}
		}

		targets := b.PrevAsgs[region.Region]
		if len(targets) == 0 {
			Logger.Info("No target to delete : ", region.Region)
			finished = append(finished, region.Region)
			continue
		}

		okCount := 0
		for _, target := range targets {
			ok := b.Deployer.CheckTerminating(client, target, config.DisableMetrics)
			if ok {
				b.Logger.Info("finished : ", target)
				okCount++
			}
		}

		if okCount == len(targets) {
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return map[string]bool{stackName: true}
	}

	return map[string]bool{stackName: false}
}

// CheckRegionExist checks if target region is really in regions described in manifest file
func CheckRegionExist(target string, regions []schemas.RegionConfig) bool {
	regionExists := false
	for _, region := range regions {
		if region.Region == target {
			regionExists = true
			break
		}
	}

	return regionExists
}

// Gather the whole metrics from deployer
func (b BlueGreen) GatherMetrics(config builder.Config) error {
	if config.DisableMetrics {
		return nil
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			return nil
		}
	}

	for _, region := range b.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		b.Logger.Infof("[%s]The number of previous autoscaling groups for gathering metrics is %d", region.Region, len(b.PrevAsgs[region.Region]))

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			return err
		}

		if len(b.PrevAsgs[region.Region]) > 0 {
			var errorList []error
			for _, asg := range b.PrevAsgs[region.Region] {
				b.Logger.Debugf("Start gathering metrics about autoscaling group : %s", asg)
				err := b.Deployer.GatherMetrics(client, asg)
				if err != nil {
					errorList = append(errorList, err)
				}
				b.Logger.Debugf("Finish gathering metrics about autoscaling group %s.", asg)
			}

			if len(errorList) > 0 {
				for _, e := range errorList {
					b.Logger.Errorf(e.Error())
				}
				return errors.New("error occurred on gathering metrics")
			}
		} else {
			b.Logger.Debugf("No previous versions to gather metrics : %s\n", region.Region)
		}
	}

	return nil
}

// CheckPrevious checks if there is any previous version of autoscaling group
func (b BlueGreen) CheckPrevious(config builder.Config) error {
	// Make Frigga
	frigga := tool.Frigga{}
	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		//Setup frigga with prefix
		frigga.Prefix = tool.BuildPrefixName(b.AwsConfig.Name, b.Stack.Env, region.Region)

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			return err
		}

		// Get All Autoscaling Groups
		asgGroups := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(frigga.Prefix)

		//Get All Previous Autoscaling Groups and versions
		prevAsgs := []string{}
		prevInstanceIds := []string{}
		prevVersions := []int{}
		var prevInstanceCount schemas.Capacity
		for _, asgGroup := range asgGroups {
			prevAsgs = append(prevAsgs, *asgGroup.AutoScalingGroupName)
			prevVersions = append(prevVersions, tool.ParseVersion(*asgGroup.AutoScalingGroupName))
			for _, instance := range asgGroup.Instances {
				prevInstanceIds = append(prevInstanceIds, *instance.InstanceId)
			}

			prevInstanceCount.Desired = *asgGroup.DesiredCapacity
			prevInstanceCount.Max = *asgGroup.MaxSize
			prevInstanceCount.Min = *asgGroup.MinSize
		}
		b.Logger.Info("Previous Versions : ", strings.Join(prevAsgs, " | "))

		b.PrevAsgs[region.Region] = prevAsgs
		b.PrevInstances[region.Region] = prevInstanceIds
		b.PrevVersions[region.Region] = prevVersions
		b.PrevInstanceCount[region.Region] = prevInstanceCount
	}

	b.StepStatus[constants.StepCheckPrevious] = true
	return nil
}

func (b BlueGreen) SkipDeployStep() {
	b.StepStatus[constants.StepDeploy] = true
	b.StepStatus[constants.StepAdditionalWork] = true
}
