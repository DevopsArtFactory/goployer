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
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/helper"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Canary struct {
	PrevTargetGroups            map[string][]string
	TargetGroups                map[string][]*string
	PrevHealthCheckTargetGroups map[string]string
	LoadBalancer                map[string]string
	LBSecurityGroup             map[string]*string
	*Deployer
}

// NewCanary creates new canary deployment deployer
func NewCanary(h *helper.DeployerHelper) *Canary {
	var awsClients []aws.Client
	for _, region := range h.Stack.Regions {
		if len(h.Region) > 0 && h.Region != region.Region {
			h.Logger.Debugf("skip creating aws clients in %s region", region.Region)
			continue
		}
		awsClients = append(awsClients, aws.BootstrapServices(region.Region, h.Stack.AssumeRole))
	}

	d := InitDeploymentConfiguration(h, awsClients)

	return &Canary{
		PrevHealthCheckTargetGroups: map[string]string{},
		PrevTargetGroups:            map[string][]string{},
		TargetGroups:                map[string][]*string{},
		LoadBalancer:                map[string]string{},
		LBSecurityGroup:             map[string]*string{},
		Deployer:                    &d,
	}
}

// GetDeployer returns canary deployer
func (c *Canary) GetDeployer() *Deployer {
	return c.Deployer
}

// CheckPreviousResources checks if there is any previous version of autoscaling group
func (c *Canary) CheckPreviousResources(config schemas.Config) error {
	err := c.CheckPrevious(config)
	if err != nil {
		return err
	}

	return nil
}

// Deploy runs deployments with canary approach
func (c *Canary) Deploy(config schemas.Config) error {
	if !c.StepStatus[constants.StepCheckPrevious] {
		return nil
	}
	c.Logger.Infof("Deploy Mode is %s", c.Mode)

	// Get LocalFileProvider
	c.LocalProvider = builder.SetUserdataProvider(c.Stack.Userdata, c.AwsConfig.Userdata)
	for i, region := range c.Stack.Regions {
		// Region check
		// If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			c.Logger.Debugf("This region is skipped by user : %s", region.Region)
			continue
		}

		if err := c.ValidateCanaryDeployment(config, region.Region); err != nil {
			return err
		}

		latestASG := c.LatestAsg[region.Region]
		targetGroups, err := c.GetAsgTargetGroups(latestASG, region.Region)
		if err != nil {
			return err
		}

		canaryVersion := CheckCanaryVersion(targetGroups, region.Region)
		c.Logger.Debugf("Current canary version: %d", canaryVersion)

		selectedTargetGroup := c.SelectTargetGroupForCopy(region, canaryVersion)
		c.Logger.Debugf("Selected target group to copy: %s", selectedTargetGroup)

		tgDetail, err := c.DescribeTargetGroup(selectedTargetGroup, region.Region)
		if err != nil {
			return err
		}

		// Check canary load balancer
		lbSg, canaryLoadBalancer, err := c.GetLoadBalancerAndSecurityGroupForCanary(region, tgDetail, config.CompleteCanary)
		if err != nil {
			return err
		}

		// Create canary security group
		err = c.GetEC2CanarySecurityGroup(tgDetail, region, lbSg, config.CompleteCanary)
		if err != nil {
			return err
		}

		switch config.CompleteCanary {
		case true:
			if err := c.CompleteCanaryDeployment(config, region, latestASG); err != nil {
				return err
			}
		case false:
			changedRegionConfig, err := c.RunCanaryDeployment(config, region, tgDetail, canaryLoadBalancer, canaryVersion)
			if err != nil {
				return err
			}
			c.Stack.Regions[i] = changedRegionConfig
		}
	}

	c.StepStatus[constants.StepDeploy] = true
	return nil
}

// HealthChecking does health checking for canary deployment
func (c *Canary) HealthChecking(config schemas.Config) error {
	healthy := false

	for !healthy {
		c.Logger.Debugf("Start Timestamp: %d, timeout: %s", config.StartTimestamp, config.Timeout)
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err := c.Deployer.HealthChecking(config)
		if err != nil {
			return errors.New("error happened while health checking")
		}

		if isDone {
			healthy = true
		} else {
			time.Sleep(config.PollingInterval)
		}
	}

	return nil
}

// FinishAdditionalWork processes additional work for the new deployment
func (c *Canary) FinishAdditionalWork(config schemas.Config) error {
	if !c.StepStatus[constants.StepDeploy] {
		return nil
	}

	if config.CompleteCanary {
		c.StepStatus[constants.StepAdditionalWork] = true
		return nil
	}

	skipped := len(config.Region) > 0 && !CheckRegionExist(config.Region, c.Stack.Regions)

	if !skipped {
		// attach to the previous target group
		if len(c.PrevTargetGroups) > 0 {
			if err := c.AttachToOriginalTargetGroups(config); err != nil {
				return err
			}

			if err := c.HealthChecking(config); err != nil {
				return err
			}
		}

		if err := c.DoCommonAdditionalWork(config); err != nil {
			return err
		}
	}

	c.Logger.Debug("Finish additional works.")
	c.StepStatus[constants.StepAdditionalWork] = true
	return nil
}

// TriggerLifecycleCallbacks runs lifecycle callbacks before cleaning.
func (c *Canary) TriggerLifecycleCallbacks(config schemas.Config) error {
	if !c.StepStatus[constants.StepAdditionalWork] {
		return nil
	}
	if config.CompleteCanary {
		c.StepStatus[constants.StepTriggerLifecycleCallback] = true
		return nil
	}
	return c.Deployer.TriggerLifecycleCallbacks(config)
}

// CleanPreviousVersion cleans previous version of autoscaling group or canary target group
func (c *Canary) CleanPreviousVersion(config schemas.Config) error {
	if !c.StepStatus[constants.StepTriggerLifecycleCallback] {
		return nil
	}
	c.Logger.Debug("Delete Mode is " + c.Mode)

	skipped := false
	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, c.Stack.Regions) {
			skipped = true
		}
	}

	if len(c.PrevAsgs) == 0 && !config.CompleteCanary {
		c.Logger.Debug("canary is being used and there is no resources to delete")
		skipped = true
	}

	if !skipped {
		c.Logger.Debugf("Start to clean resources from previous canary deployment")
		for _, region := range c.Stack.Regions {
			if err := c.CleanPreviousCanaryResources(region, config.CompleteCanary); err != nil {
				return err
			}
		}
		// TODO: Need to uncomment if goployer supports gradual canary deployment
		////Apply AutoScaling Policies
		/*for _, region := range c.Stack.Regions {
			if err := c.ReduceOriginalAutoscalingGroupCount(region); err != nil {
				return err
			}
		}*/
	}
	c.StepStatus[constants.StepCleanPreviousVersion] = true
	return nil
}

// GatherMetrics gathers the whole metrics from deployer
func (c *Canary) GatherMetrics(config schemas.Config) error {
	if !c.StepStatus[constants.StepCleanChecking] {
		return nil
	}
	if config.DisableMetrics {
		return nil
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, c.Stack.Regions) {
			return nil
		}
	}

	if !config.CompleteCanary {
		c.Logger.Debug("Skip gathering metrics because canary is now applied")
		return nil
	}

	if err := c.StartGatheringMetrics(config); err != nil {
		return err
	}

	c.StepStatus[constants.StepGatherMetrics] = true
	return nil
}

// RunAPITest tries to run API Test
func (c *Canary) RunAPITest(config schemas.Config) error {
	if !c.StepStatus[constants.StepGatherMetrics] {
		return nil
	}

	if !config.CompleteCanary {
		c.Logger.Debug("Skip API test because canary is now applied")
		return nil
	}

	err := c.RunAPITest(config)
	if err != nil {
		return err
	}

	c.StepStatus[constants.StepRunAPI] = true
	return nil
}

// ValidateCanaryDeployment validates if configuration is right for canary deployment
func (c *Canary) ValidateCanaryDeployment(config schemas.Config, region string) error {
	if c.DeploymentFlag[region] != constants.CanaryDeployment && config.CompleteCanary {
		return errors.New("you cannot complete canary deployment before start canary before")
	}

	return nil
}

// CopyTargetGroups creates copy existing target group for canary
func (c *Canary) CopyTargetGroups(tg *elbv2.TargetGroup, canaryTgName, region string) (*elbv2.TargetGroup, error) {
	client, err := selectClientFromList(c.AWSClients, region)
	if err != nil {
		return nil, err
	}

	newTargetGroup, err := client.ELBV2Service.CreateTargetGroup(tg, canaryTgName)
	if err != nil {
		return nil, err
	}

	return newTargetGroup, nil
}

// GenerateCanaryTargetGroupName generates name of canary target group for canary
func (c *Canary) GenerateCanaryTargetGroupName(canaryVersion int) string {
	return fmt.Sprintf("%s-%s-canary-v%03d", c.AwsConfig.Name, c.Stack.Env, canaryVersion+1)
}

// GenerateCanaryLoadBalancerName generates name of canary load balancer for canary
func (c *Canary) GenerateCanaryLoadBalancerName(region string) string {
	return fmt.Sprintf("%s-%s-%s-%s", c.AwsConfig.Name, c.Stack.Env, strings.ReplaceAll(region, "-", ""), constants.CanaryMark)
}

// GenerateCanarySecurityGroupName generates name of canary load balancer for canary
func (c *Canary) GenerateCanarySecurityGroupName(region string) string {
	return fmt.Sprintf("%s-%s-%s-%s", c.AwsConfig.Name, c.Stack.Env, strings.ReplaceAll(region, "-", ""), constants.CanaryMark)
}

// GenerateCanaryLBSecurityGroupName generates name of canary load balancer for canary
func (c *Canary) GenerateCanaryLBSecurityGroupName(region string) string {
	return fmt.Sprintf("%s-%s-%s-lb-%s", c.AwsConfig.Name, c.Stack.Env, strings.ReplaceAll(region, "-", ""), constants.CanaryMark)
}

// GetAsgTargetGroups retrieves target group list of autoscaling group
func (c *Canary) GetAsgTargetGroups(asg, region string) ([]*string, error) {
	client, err := selectClientFromList(c.AWSClients, region)
	if err != nil {
		return nil, err
	}

	asgList, err := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(asg)
	if err != nil {
		return nil, err
	}

	var tgARNs []*string
	for _, asg := range asgList {
		for _, tg := range asg.TargetGroupARNs {
			if !tool.IsStringInPointerArray(*tg, tgARNs) {
				tgARNs = append(tgARNs, tg)
			}
		}
	}

	if len(tgARNs) > 0 {
		c.Logger.Debugf("Found target groups for canary deployment %s: %d", asg, len(tgARNs))
	}

	c.TargetGroups[region] = tgARNs

	return tgARNs, nil
}

// SelectTargetGroupForCopy select target group for copy
func (c *Canary) SelectTargetGroupForCopy(region schemas.RegionConfig, canaryVersion int) string {
	// no canary version
	if canaryVersion == 0 {
		if len(region.HealthcheckTargetGroup) > 0 {
			return region.HealthcheckTargetGroup
		}

		return constants.EmptyString
	}

	return c.GenerateCanaryTargetGroupName(canaryVersion - 1)
}

// AttachToOriginalTargetGroups attaches the new autoscaling group to original target groups
func (c *Canary) AttachToOriginalTargetGroups(config schemas.Config) error {
	// Apply AutoScaling Policies
	for _, region := range c.Stack.Regions {
		// If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			c.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		client, err := selectClientFromList(c.AWSClients, region.Region)
		if err != nil {
			return err
		}

		c.Logger.Debugf("Get target group ARN of original target groups: %s", c.PrevTargetGroups[region.Region])
		targetGroupARNs, err := client.ELBV2Service.GetTargetGroupARNs(c.PrevTargetGroups[region.Region])
		if err != nil {
			return err
		}

		if targetGroupARNs == nil {
			return fmt.Errorf("there is no target group specified")
		}

		c.Logger.Debugf("Attach autoscaling group to original target groups: %s", c.AsgNames[region.Region])
		if err := client.EC2Service.AttachAsgToTargetGroups(c.AsgNames[region.Region], targetGroupARNs); err != nil {
			return err
		}
	}

	c.Logger.Debug("Finish attaching autoscaling group to original target groups")
	return nil
}

// ChangeTargetGroupInfo changes existing target group to the new one for canary deployment
func (c *Canary) ChangeTargetGroupInfo(newTgName string, region schemas.RegionConfig) schemas.RegionConfig {
	if len(region.HealthcheckTargetGroup) > 0 {
		c.PrevHealthCheckTargetGroups[region.Region] = region.HealthcheckTargetGroup
	}
	region.HealthcheckTargetGroup = newTgName

	if len(region.TargetGroups) > 0 {
		c.PrevTargetGroups[region.Region] = region.TargetGroups
	}
	region.TargetGroups = []string{newTgName}
	return region
}

// CleanChecking checks Termination status
func (c *Canary) CleanChecking(config schemas.Config) error {
	if !c.StepStatus[constants.StepCleanPreviousVersion] {
		return nil
	}
	done := false
	isDone := false
	var err error

	for !done {
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err = c.Deployer.CleanChecking(config)
		if err != nil {
			return errors.New("error happened while health checking")
		}

		if isDone {
			done = true
		} else {
			c.Logger.Info("All stacks are not ready to be terminated... Please waiting...")
			time.Sleep(config.PollingInterval)
		}
	}

	c.StepStatus[constants.StepCleanChecking] = true
	return nil
}

// FindCanaryLoadBalancer finds if there is canary-related load balancer
func (c *Canary) FindCanaryLoadBalancer(region schemas.RegionConfig) (*elbv2.LoadBalancer, error) {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	loadBalancers, err := client.ELBV2Service.DescribeLoadBalancers()
	if err != nil {
		return nil, err
	}

	for _, lb := range loadBalancers {
		if c.CheckValidCanaryLB(c.AwsConfig.Name, *lb.LoadBalancerName) {
			return lb, nil
		}
	}

	return nil, nil
}

// CheckValidCanaryLB checks if load balancer is canary-related or not
func (c *Canary) CheckValidCanaryLB(app, lb string) bool {
	return strings.HasPrefix(lb, app) && strings.Contains(lb, constants.CanaryMark)
}

// CheckCanaryVersion checks latest version of canary target group
func CheckCanaryVersion(tgs []*string, region string) int {
	latestVersion := 0
	for _, tg := range tgs {
		if tool.IsCanaryTargetGroupArn(*tg, region) {
			name := tool.ParseTargetGroupName(*tg)
			v := tool.ParseTargetGroupVersion(name)
			if v > 0 && v > latestVersion {
				latestVersion = v
			}
		}
	}

	return latestVersion
}

// CreateCanaryLoadBalancer creates a new load balancer for canary
func (c *Canary) CreateCanaryLoadBalancer(region schemas.RegionConfig, groupID *string) (*elbv2.LoadBalancer, error) {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	newLBName := c.GenerateCanaryLoadBalancerName(region.Region)

	availabilityZones, err := client.EC2Service.GetAvailabilityZones(region.VPC, region.AvailabilityZones)
	if err != nil {
		return nil, err
	}

	subnets := region.SubnetIDs
	if len(subnets) == 0 {
		c.Logger.Info("Not Subnet ID Specific")
		subnets, err = client.EC2Service.GetSubnets(region.VPC, region.UsePublicSubnets, availabilityZones)
		if err != nil {
			return nil, err
		}
	} else {
		subnetIds := strings.Join(subnets, " ")
		c.Logger.Infof("Subnet ID are Specific : %s", subnetIds)
	}

	lb, err := client.ELBV2Service.CreateLoadBalancer(newLBName, subnets, groupID)
	if err != nil {
		return nil, err
	}

	return lb, nil
}

// AttachCanaryTargetGroup attaches target group to load balancer
func (c *Canary) AttachCanaryTargetGroup(lbArn, tgArn string, region schemas.RegionConfig) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	existingListeners, err := client.ELBV2Service.DescribeListeners(lbArn)
	if err != nil {
		return err
	}

	if len(existingListeners) == 0 {
		return client.ELBV2Service.CreateNewListener(lbArn, tgArn)
	}

	return client.ELBV2Service.ModifyListener(existingListeners[0].ListenerArn, tgArn)
}

// GetEC2CanarySecurityGroup creates a new security group for canary
func (c *Canary) GetEC2CanarySecurityGroup(tg *elbv2.TargetGroup, region schemas.RegionConfig, lbSg *string, completeCanary bool) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	newSGName := c.GenerateCanarySecurityGroupName(region.Region)

	if completeCanary {
		groupID, err := client.EC2Service.GetSecurityGroup(newSGName)
		if err != nil {
			return err
		}
		c.SecurityGroup[region.Region] = groupID
		c.Logger.Debugf("Found existing security group id: %s", *groupID)

		return nil
	}

	duplicated := false
	groupID, err := client.EC2Service.CreateSecurityGroup(newSGName, tg.VpcId)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidGroup.Duplicate" {
			c.Logger.Debugf("Security group is already created: %s", newSGName)
			duplicated = true
		}

		if !duplicated {
			return err
		}
	}

	if duplicated {
		groupID, err = client.EC2Service.GetSecurityGroup(newSGName)
		if err != nil {
			return err
		}
		c.Logger.Debugf("Found existing security group id: %s", *groupID)
	} else if err := client.EC2Service.UpdateOutboundRules(*groupID, "-1", "0.0.0.0/0", "outbound to internet", -1, -1); err != nil {
		c.Logger.Warn(err.Error())
	}

	// inbound
	if err := client.EC2Service.UpdateInboundRulesWithGroup(*groupID, "tcp", "Allow access from canary load balancer", lbSg, *tg.Port, *tg.Port); err != nil {
		c.Logger.Warn(err.Error())
	}

	c.SecurityGroup[region.Region] = groupID
	c.Logger.Debugf("Security group for this canary deployment: %s", *groupID)

	return nil
}

// GetCanaryLoadBalancerSecurityGroup retrieves existing load balancer security group for canary
func (c *Canary) GetCanaryLoadBalancerSecurityGroup(region schemas.RegionConfig) (*string, error) {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	newLBName := c.GenerateCanaryLBSecurityGroupName(region.Region)

	groupID, err := client.EC2Service.GetSecurityGroup(newLBName)
	if err != nil {
		c.Logger.Warn(err.Error())
		return nil, nil
	}

	c.Logger.Debugf("Found existing lb security group id: %s", *groupID)

	return groupID, nil
}

// CreateCanaryLBSecurityGroup creates a new security group for canary
func (c *Canary) CreateCanaryLBSecurityGroup(tg *elbv2.TargetGroup, region schemas.RegionConfig) (*string, error) {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	lbSGName := c.GenerateCanaryLBSecurityGroupName(region.Region)

	duplicated := false
	groupID, err := client.EC2Service.CreateSecurityGroup(lbSGName, tg.VpcId)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidGroup.Duplicate" {
			c.Logger.Debugf("Security group is already created: %s", lbSGName)
			duplicated = true
		}
		if !duplicated {
			return nil, err
		}
	}

	if duplicated {
		groupID, err = client.EC2Service.GetSecurityGroup(lbSGName)
		if err != nil {
			return nil, err
		}

		c.Logger.Debugf("Found existing security group id: %s", *groupID)
	}

	// inbound
	if err := client.EC2Service.UpdateInboundRules(*groupID, "tcp", "0.0.0.0/0", "inbound from internet", 80, 80); err != nil {
		c.Logger.Warn(err.Error())
	}

	// outbound
	if err := client.EC2Service.UpdateOutboundRules(*groupID, "-1", "0.0.0.0/0", "outbound to internet", -1, -1); err != nil {
		c.Logger.Warn(err.Error())
	}

	return groupID, nil
}

// ReduceOriginalAutoscalingGroupCount set existing autoscaling group count to -1
func (c *Canary) ReduceOriginalAutoscalingGroupCount(region schemas.RegionConfig) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	changedCapacity := c.PrevInstanceCount[region.Region]
	if changedCapacity.Desired <= 1 {
		c.Logger.Debugf("Autoscaling group has only %d instances so that goployer cannot terminate one instance: %s", changedCapacity.Desired, c.LatestAsg[region.Region])
		return nil
	}

	c.Logger.Debugf("Reduce size of autoscaling group by one instance: %s / %s", c.LatestAsg[region.Region], region.Region)
	c.Slack.SendSimpleMessage(fmt.Sprintf("Reducing the size of autoscaling group by 1 : %s / %s", c.LatestAsg[region.Region], region.Region))
	changedCapacity.Desired--
	if changedCapacity.Desired < changedCapacity.Min {
		changedCapacity.Min--
	}

	c.Logger.Debugf("[%s]Previous capacity count - Min: %d, Desired: %d, Max: %d", c.LatestAsg[region.Region], c.PrevInstanceCount[region.Region].Min, c.PrevInstanceCount[region.Region].Desired, c.PrevInstanceCount[region.Region].Max)
	c.Logger.Debugf("[%s]Changed capacity count - Min: %d, Desired: %d, Max: %d", c.LatestAsg[region.Region], changedCapacity.Min, changedCapacity.Desired, changedCapacity.Max)

	retry := int64(3)
	for {
		retry, err = client.EC2Service.UpdateAutoScalingGroupSize(c.LatestAsg[region.Region], changedCapacity.Min, changedCapacity.Desired, changedCapacity.Max, retry)
		if err != nil {
			if retry > 0 {
				c.Logger.Debugf("error occurred and remained retry count is %d", retry)
				time.Sleep(time.Duration(1+2*(3-retry)) * time.Second)
			} else {
				return err
			}
		}

		if err == nil {
			break
		}
	}

	return nil
}

// CleanPreviousCanaryResources cleans previous canary resources
func (c *Canary) CleanPreviousCanaryResources(region schemas.RegionConfig, completeCanary bool) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	prefix := tool.BuildPrefixName(c.AwsConfig.Name, c.Stack.Env, region.Region)

	asgList, err := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(prefix)
	if err != nil {
		return err
	}

	for _, asg := range asgList {
		if (completeCanary && *asg.AutoScalingGroupName == c.LatestAsg[region.Region]) || !tool.IsStringInArray(*asg.AutoScalingGroupName, c.PrevAsgs[region.Region]) {
			continue
		}

		c.Logger.Debugf("[Resizing] target autoscaling group : %s", *asg.AutoScalingGroupName)
		if err := c.ResizingAutoScalingGroupCount(client, *asg.AutoScalingGroupName, 0); err != nil {
			c.Logger.Error(err.Error())
		}
		c.Logger.Debugf("Resizing autoscaling group finished: %s", *asg.AutoScalingGroupName)

		for _, tg := range asg.TargetGroupARNs {
			if tool.IsCanaryTargetGroupArn(*tg, region.Region) {
				c.Logger.Debugf("Try to delete target group: %s", *tg)
				if err := client.ELBV2Service.DeleteTargetGroup(tg); err != nil {
					return err
				}
				c.Logger.Debugf("Deleted target group: %s", *tg)
			}
		}
	}

	c.Logger.Debugf("Start to delete load balancer and security group for canary")
	if completeCanary {
		if err := c.DeleteLoadBalancer(region); err != nil {
			return err
		}

		if err := c.LoadBalancerDeletionChecking(region); err != nil {
			return err
		}

		if err := c.DeleteEC2IngressRules(region); err != nil {
			return err
		}

		if err := c.DeleteEC2SecurityGroup(region); err != nil {
			return err
		}

		if err := c.DeleteLBSecurityGroup(region); err != nil {
			return err
		}
	}

	return nil
}

// DeleteLoadBalancer deletes load balancer
func (c *Canary) DeleteLoadBalancer(region schemas.RegionConfig) error {
	if len(c.LoadBalancer[region.Region]) == 0 {
		c.Logger.Debugf("No load balancer to delete")
		return nil
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	err = client.ELBV2Service.DeleteLoadBalancer(c.LoadBalancer[region.Region])
	if err != nil {
		return err
	}

	c.Logger.Debugf("Delete load balancer: %s", c.LoadBalancer[region.Region])

	return nil
}

// DeleteLBSecurityGroup deletes load balancer security group
func (c *Canary) DeleteLBSecurityGroup(region schemas.RegionConfig) error {
	if c.LBSecurityGroup[region.Region] == nil {
		c.Logger.Debugf("No lb security group to delete")
		return nil
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	c.Logger.Debug("Wait 30 seconds until load balancer is successfully terminated")
	time.Sleep(30 * time.Second)

	retry := int64(4)
	for {
		err = client.EC2Service.DeleteSecurityGroup(*c.LBSecurityGroup[region.Region])
		if err != nil {
			if retry > 0 {
				retry--
				c.Logger.Debugf("error occurred on lb deletion and remained retry count is %d", retry)
				time.Sleep(time.Duration(1+5*(3-retry)) * time.Second)
			} else {
				return err
			}
		}

		if err == nil {
			break
		}
	}
	c.Logger.Debugf("Delete load balancer security group: %s", *c.LBSecurityGroup[region.Region])

	return nil
}

// DeleteEC2SecurityGroup deletes EC2 security group for canary
func (c *Canary) DeleteEC2SecurityGroup(region schemas.RegionConfig) error {
	if c.SecurityGroup[region.Region] == nil {
		c.Logger.Debugf("No EC2 security group to delete")
		return nil
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	err = client.EC2Service.DeleteSecurityGroup(*c.SecurityGroup[region.Region])
	if err != nil {
		return err
	}

	c.Logger.Debugf("Delete canary EC2 security group: %s", *c.SecurityGroup[region.Region])

	return nil
}

// DeleteEC2IngressRules deletes ingress rules for EC2
func (c *Canary) DeleteEC2IngressRules(region schemas.RegionConfig) error {
	if c.SecurityGroup[region.Region] == nil {
		c.Logger.Debugf("No EC2 security group to delete")
		return nil
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	// inbound
	sgDetails, err := client.EC2Service.GetSecurityGroupDetails([]*string{c.SecurityGroup[region.Region]})
	if err != nil {
		return err
	}

	if len(sgDetails) != 1 {
		return fmt.Errorf("delete ec2 ingress error because more than one or no security group detected: %d", len(sgDetails))
	}

	sgID := sgDetails[0].GroupId
	for _, in := range sgDetails[0].IpPermissions {
		if len(in.UserIdGroupPairs) > 0 {
			for _, uip := range in.UserIdGroupPairs {
				if err := client.EC2Service.RevokeInboundRulesWithGroup(*sgID, *in.IpProtocol, uip.GroupId, *in.FromPort, *in.ToPort); err != nil {
					c.Logger.Warn(err.Error())
				}
			}
		}
	}

	c.Logger.Debugf("Detach lb security group from EC2 security group: %s", *c.SecurityGroup[region.Region])

	return nil
}

// RemoveCanaryTag deletes Canary tag from auto scaling group
func (c *Canary) RemoveCanaryTag(asg string, region schemas.RegionConfig) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	err = client.EC2Service.DeleteCanaryTag(asg)
	if err != nil {
		return err
	}

	c.Logger.Debugf("Remove canary tag from autoscaling group")

	return nil
}

// DetachCanaryTargetGroup detaches canary target group from auto scaling group
func (c *Canary) DetachCanaryTargetGroup(asg string, region schemas.RegionConfig, tgs []*string) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	var targets []*string
	for _, tg := range tgs {
		if tool.IsCanaryTargetGroupArn(*tg, region.Region) {
			targets = append(targets, tg)
		}
	}
	err = client.EC2Service.DetachLoadBalancerTargetGroup(asg, targets)
	if err != nil {
		return err
	}

	c.Logger.Debugf("Remove canary target group from autoscaling group")

	return nil
}

// DetachSecurityGroup deletes lb security groups from instances
func (c *Canary) DetachSecurityGroup(nis []*ec2.InstanceNetworkInterface, region schemas.RegionConfig, excludeSg string) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	for _, ni := range nis {
		var sgs []*string
		for _, group := range ni.Groups {
			if *group.GroupId != excludeSg {
				sgs = append(sgs, group.GroupId)
			}
		}
		if len(sgs) > 0 {
			err = client.EC2Service.ModifyNetworkInterfaces(ni.NetworkInterfaceId, sgs)
			if err != nil {
				return err
			}

			c.Logger.Debugf("Remove security group from eni: %s", *ni.NetworkInterfaceId)
		}
	}

	return nil
}

// ChangeLaunchTemplateVersion changes launch template to the new version
func (c *Canary) ChangeLaunchTemplateVersion(asg string, lt *autoscaling.LaunchTemplateSpecification, region schemas.RegionConfig, excludeSg string) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	ltDetail, err := client.EC2Service.GetMatchingLaunchTemplate(*lt.LaunchTemplateId)
	if err != nil {
		return err
	}
	c.Logger.Debugf("Retrieved previous version of launch template of launch template: %s", *lt.LaunchTemplateId)

	var sgs []*string
	for _, sg := range ltDetail.LaunchTemplateData.SecurityGroupIds {
		if *sg != excludeSg {
			sgs = append(sgs, sg)
		}
	}

	ret, err := client.EC2Service.CreateNewLaunchTemplateVersion(ltDetail, sgs)
	if err != nil {
		return err
	}
	c.Logger.Debugf("Created new version of launch template: %s - version%d", *ret.LaunchTemplateId, *ret.VersionNumber)

	if err := client.EC2Service.UpdateAutoScalingLaunchTemplate(asg, ret); err != nil {
		return err
	}

	return nil
}

// RunCanaryDeployment runs canary deployment
func (c *Canary) RunCanaryDeployment(config schemas.Config, region schemas.RegionConfig, tgDetail *elbv2.TargetGroup, canaryLoadBalancer *elbv2.LoadBalancer, canaryVersion int) (schemas.RegionConfig, error) {
	newTgName := c.GenerateCanaryTargetGroupName(canaryVersion)
	c.Logger.Debugf("New target group will be created for canary deployment: %s", newTgName)

	tg, err := c.CopyTargetGroups(tgDetail, newTgName, region.Region)
	if err != nil {
		return region, err
	}
	c.Logger.Debugf("New target group is created: %s", *tg.TargetGroupName)

	if err := c.AttachCanaryTargetGroup(*canaryLoadBalancer.LoadBalancerArn, *tg.TargetGroupArn, region); err != nil {
		return region, err
	}
	c.Logger.Debugf("Attached target group to load balancer: %s", *canaryLoadBalancer.LoadBalancerName)

	c.Logger.Debugf("Change target group information with new target group: %s", newTgName)
	region = c.ChangeTargetGroupInfo(newTgName, region)
	c.Logger.Debugf("Changed information: %s / %s", region.HealthcheckTargetGroup, region.TargetGroups)

	if err := c.Deployer.Deploy(config, region); err != nil {
		return region, err
	}

	return region, nil
}

// CompleteCanaryDeployment completes canary deployment
func (c *Canary) CompleteCanaryDeployment(config schemas.Config, region schemas.RegionConfig, latestASG string) error {
	asgDetail, err := c.DescribeAutoScalingGroup(latestASG, region.Region)
	if err != nil {
		return err
	}

	if asgDetail == nil {
		return fmt.Errorf("no autoscaling group information retrieved. Please check autoscaling group resource: %s", latestASG)
	}

	instanceIds := extractInstanceIds(asgDetail)
	instancesDetail, err := c.DescribeInstances(instanceIds, region)
	if err != nil {
		return err
	}

	nis := getNetworkInterfaces(instancesDetail)

	if err := c.DetachSecurityGroup(nis, region, *c.SecurityGroup[region.Region]); err != nil {
		return err
	}

	if err := c.RemoveCanaryTag(latestASG, region); err != nil {
		return err
	}

	if err := c.DetachCanaryTargetGroup(latestASG, region, asgDetail.TargetGroupARNs); err != nil {
		return err
	}

	if err := c.ChangeLaunchTemplateVersion(latestASG, asgDetail.LaunchTemplate, region, *c.SecurityGroup[region.Region]); err != nil {
		return err
	}

	appliedCapacity, err := c.DecideCapacity(config.ForceManifestCapacity, config.CompleteCanary, region.Region, len(c.PrevAsgs[region.Region]), c.Stack.RollingUpdateInstanceCount)
	if err != nil {
		return err
	}

	c.Logger.Debugf("Resizing latest autoscaling group: min - %d, desired - %d, max - %d", appliedCapacity.Min, appliedCapacity.Desired, appliedCapacity.Max)
	if err := c.ResizingAutoScalingGroup(latestASG, region.Region, appliedCapacity); err != nil {
		return err
	}

	// settings for health checking
	c.Stack.Capacity.Desired = appliedCapacity.Desired
	c.AsgNames[region.Region] = latestASG

	return nil
}

// GetLoadBalancerAndSecurityGroupForCanary gets load balancer and security group for canary deployment
func (c *Canary) GetLoadBalancerAndSecurityGroupForCanary(region schemas.RegionConfig, tgDetail *elbv2.TargetGroup, completeCanary bool) (*string, *elbv2.LoadBalancer, error) {
	canaryLoadBalancer, err := c.FindCanaryLoadBalancer(region)
	if err != nil {
		return nil, nil, err
	}

	var lbSg *string
	if len(region.HealthcheckLB) > 0 || len(region.TargetGroups) > 0 {
		if canaryLoadBalancer == nil {
			if !completeCanary {
				lbSg, err := c.CreateCanaryLBSecurityGroup(tgDetail, region)
				if err != nil {
					return nil, nil, err
				}

				canaryLoadBalancer, err = c.CreateCanaryLoadBalancer(region, lbSg)
				if err != nil {
					return nil, nil, err
				}
				c.Logger.Debugf("Created a new load balancer for canary: %s", *canaryLoadBalancer.LoadBalancerName)
			}
		} else {
			c.Logger.Debugf("Found existing load balancer for canary: %s", *canaryLoadBalancer.LoadBalancerName)
			lbSg, err = c.GetCanaryLoadBalancerSecurityGroup(region)
			if err != nil {
				return nil, nil, err
			}
		}

		if lbSg == nil && !completeCanary {
			lbSg, err = c.CreateCanaryLBSecurityGroup(tgDetail, region)
			if err != nil {
				return nil, nil, err
			}
			c.Logger.Debugf("New lb security group is created: %s", *lbSg)
		}
	}

	c.LBSecurityGroup[region.Region] = lbSg
	if canaryLoadBalancer != nil {
		c.LoadBalancer[region.Region] = *canaryLoadBalancer.LoadBalancerArn
	}

	return lbSg, canaryLoadBalancer, nil
}

// LoadBalancerDeletionChecking checks if load balancer is deleted well or not
func (c *Canary) LoadBalancerDeletionChecking(region schemas.RegionConfig) error {
	if len(c.LoadBalancer[region.Region]) == 0 {
		c.Logger.Debugf("No load balancer to delete")
		return nil
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	done := false
	for !done {
		lb, err := client.ELBV2Service.GetMatchingLoadBalancer(c.LoadBalancer[region.Region])
		if err != nil {
			return err
		}

		if lb == nil {
			c.Logger.Debugf("Canary load balancer is deleted: %s", c.LoadBalancer[region.Region])
			done = true
		} else {
			time.Sleep(10 * time.Second)
		}
	}

	return nil
}

// extractInstanceIds gathers pointer of instance's id and make slice with them
func extractInstanceIds(asgDetail *autoscaling.Group) []*string {
	var instanceIds []*string
	for _, ins := range asgDetail.Instances {
		instanceIds = append(instanceIds, ins.InstanceId)
	}

	return instanceIds
}

// getNetworkInterfaces gathers all network interfaces from EC2 instances
func getNetworkInterfaces(instances []*ec2.Instance) []*ec2.InstanceNetworkInterface {
	var nis []*ec2.InstanceNetworkInterface
	for _, instance := range instances {
		if instance.NetworkInterfaces != nil {
			nis = append(nis, instance.NetworkInterfaces...)
		}
	}

	return nis
}
