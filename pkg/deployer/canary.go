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

	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/helper"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Canary struct {
	PrevTargetGroups            map[string][]string
	TargetGroups                map[string][]string
	OriginalTargetGroupArn      map[string]string
	CanaryTargetGroupArn        map[string]string
	PrevHealthCheckTargetGroups map[string]string
	*Deployer
}

const (
	defaultCanaryWeight = int64(10)
	maxCanaryWeight     = int64(90)
)

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
		TargetGroups:                map[string][]string{},
		OriginalTargetGroupArn:      map[string]string{},
		CanaryTargetGroupArn:        map[string]string{},
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

		switch config.CompleteCanary {
		case true:
			if err := c.CompleteCanaryDeployment(config, region, latestASG, tgDetail); err != nil {
				return err
			}
		case false:
			changedRegionConfig, err := c.RunCanaryDeployment(config, region, tgDetail, canaryVersion)
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
			err := fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
			if rollbackErr := c.RollbackCanaryDeployment(config); rollbackErr != nil {
				return fmt.Errorf("%w; rollback failed: %s", err, rollbackErr)
			}
			return err
		}

		isDone, err := c.Deployer.HealthChecking(config)
		if err != nil {
			if rollbackErr := c.RollbackCanaryDeployment(config); rollbackErr != nil {
				return fmt.Errorf("error happened while health checking: %w; rollback failed: %s", err, rollbackErr)
			}
			return fmt.Errorf("error happened while health checking: %w", err)
		}

		if isDone {
			healthy = true
		} else {
			time.Sleep(config.PollingInterval)
		}
	}

	return nil
}

// ShouldRollbackCanary returns whether a failed health check owns canary resources to clean up.
func (c *Canary) ShouldRollbackCanary(config schemas.Config) bool {
	return c.StepStatus[constants.StepDeploy] && !config.CompleteCanary
}

// RollbackCanaryDeployment restores listener traffic and removes failed canary resources.
func (c *Canary) RollbackCanaryDeployment(config schemas.Config) error {
	if !c.ShouldRollbackCanary(config) {
		return nil
	}

	var retErr error
	recordErr := func(err error) {
		if err != nil && retErr == nil {
			retErr = err
		}
	}
	rollbackStartTimestamp := time.Now().Unix()

	for _, region := range c.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			c.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		if c.OriginalTargetGroupArn[region.Region] != "" {
			if err := c.ModifyListenerToStableTargetGroup(region, 100, 0); err != nil {
				recordErr(err)
			}
		}

		canaryASG := c.CanaryAutoScalingGroupName(region.Region)
		if err := c.DetachLatestCanaryResources(schemas.Config{Region: region.Region}); err != nil {
			recordErr(err)
		}

		if canaryASG != "" {
			client, err := selectClientFromList(c.AWSClients, region.Region)
			if err != nil {
				recordErr(err)
			} else if err := c.ResizingAutoScalingGroupCount(client, canaryASG, 0); err != nil {
				recordErr(err)
			} else {
				for {
					isTimeout, _ := tool.CheckTimeout(rollbackStartTimestamp, config.Timeout)
					if isTimeout {
						recordErr(fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes()))
						break
					}

					done, err := c.CheckAutoscalingInstanceCount(client, canaryASG, 0)
					if err != nil {
						recordErr(err)
						break
					}
					if done {
						if ok := c.ClearResources(client, canaryASG, config.DisableMetrics); !ok {
							recordErr(fmt.Errorf("error happened while cleaning rollback resources: %s", canaryASG))
						}
						break
					}

					c.Logger.Info("Rollback autoscaling group is not empty yet... Please waiting...")
					time.Sleep(config.PollingInterval)
				}
			}
		}
	}

	if retErr == nil {
		c.StepStatus[constants.StepDeploy] = false
	}
	return retErr
}

// CanaryAutoScalingGroupName returns the ASG owned by the active canary deployment.
func (c *Canary) CanaryAutoScalingGroupName(region string) string {
	if c.AsgNames != nil && c.AsgNames[region] != "" {
		return c.AsgNames[region]
	}
	return c.LatestAsg[region]
}

// FinishAdditionalWork processes additional work for the new deployment
func (c *Canary) FinishAdditionalWork(config schemas.Config) error {
	if !c.StepStatus[constants.StepDeploy] {
		return nil
	}

	if config.CompleteCanary {
		if err := c.ApplyCompleteCanaryWeights(config); err != nil {
			return err
		}
		if err := c.DetachLatestCanaryResources(config); err != nil {
			return err
		}
		c.StepStatus[constants.StepAdditionalWork] = true
		return nil
	}

	skipped := len(config.Region) > 0 && !CheckRegionExist(config.Region, c.Stack.Regions)

	if !skipped {
		if err := c.ApplyCanaryWeights(config); err != nil {
			return err
		}
		c.WaitCanaryBakeTime(config)

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

	err := c.Deployer.RunAPITest(config)
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
	for _, regionConfig := range c.Stack.Regions {
		if regionConfig.Region != region {
			continue
		}
		if regionConfig.Canary.ListenerARN == "" && (regionConfig.Canary.LoadBalancer == "" || regionConfig.Canary.ListenerPort == 0) {
			return errors.New("canary.listener_arn or canary.load_balancer with canary.listener_port is required for canary deployment")
		}
		if regionConfig.Canary.Weight < 0 || regionConfig.Canary.Weight > maxCanaryWeight {
			return errors.New("canary.weight must be between 0 and 90")
		}
	}

	return nil
}

// CopyTargetGroups creates copy existing target group for canary
func (c *Canary) CopyTargetGroups(tg *elbv2types.TargetGroup, canaryTgName, region string) (*elbv2types.TargetGroup, error) {
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

// GetAsgTargetGroups retrieves target group list of autoscaling group
func (c *Canary) GetAsgTargetGroups(asg, region string) ([]string, error) {
	client, err := selectClientFromList(c.AWSClients, region)
	if err != nil {
		return nil, err
	}

	asgList, err := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(asg)
	if err != nil {
		return nil, err
	}

	var tgARNs []string
	for _, asg := range asgList {
		for _, tg := range asg.TargetGroupARNs {
			if !tool.IsStringInArray(tg, tgARNs) {
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

// TargetGroupARNsForRegion returns original target groups from manifest for a region.
func (c *Canary) TargetGroupARNsForRegion(region schemas.RegionConfig) ([]string, error) {
	targetGroups := c.GetTargetGroupNames(region)
	if len(targetGroups) == 0 {
		return nil, fmt.Errorf("there is no target group specified")
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	targetGroupARNs, err := client.ELBV2Service.GetTargetGroupARNs(targetGroups)
	if err != nil {
		return nil, err
	}
	if targetGroupARNs == nil {
		return nil, fmt.Errorf("there is no target group specified")
	}

	return targetGroupARNs, nil
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

// CanaryWeights returns stable/canary weights for listener default actions.
func CanaryWeights(region schemas.RegionConfig, completeCanary bool) (int32, int32) {
	if completeCanary {
		return 100, 0
	}

	weight := region.Canary.Weight
	if weight == 0 {
		weight = defaultCanaryWeight
	}

	return int32(100 - weight), int32(weight)
}

// ResolveCanaryListenerARN resolves the listener from either ARN or load balancer name plus port.
func (c *Canary) ResolveCanaryListenerARN(region schemas.RegionConfig, client aws.Client) (string, error) {
	if region.Canary.ListenerARN != "" {
		return region.Canary.ListenerARN, nil
	}

	lb, err := client.ELBV2Service.GetLoadBalancerByName(region.Canary.LoadBalancer)
	if err != nil {
		return "", err
	}
	if lb == nil {
		return "", fmt.Errorf("canary load balancer not found: %s", region.Canary.LoadBalancer)
	}

	listeners, err := client.ELBV2Service.DescribeListeners(*lb.LoadBalancerArn)
	if err != nil {
		return "", err
	}
	for _, listener := range listeners {
		if listener.Port == nil || *listener.Port != region.Canary.ListenerPort {
			continue
		}
		if region.Canary.ListenerProtocol != "" && !strings.EqualFold(string(listener.Protocol), region.Canary.ListenerProtocol) {
			continue
		}
		if listener.ListenerArn == nil {
			break
		}
		return *listener.ListenerArn, nil
	}

	return "", fmt.Errorf("canary listener not found: %s:%d", region.Canary.LoadBalancer, region.Canary.ListenerPort)
}

// ApplyCanaryWeights sends the configured percentage of listener traffic to canary target groups.
func (c *Canary) ApplyCanaryWeights(config schemas.Config) error {
	for _, region := range c.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			c.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		stableWeight, canaryWeight := CanaryWeights(region, false)
		if err := c.ModifyWeightedListener(region, stableWeight, canaryWeight); err != nil {
			return err
		}
	}

	return nil
}

// WaitCanaryBakeTime keeps the canary weight active before the deploy command exits.
func (c *Canary) WaitCanaryBakeTime(config schemas.Config) {
	for _, region := range c.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			continue
		}
		if region.Canary.BakeTime <= 0 {
			continue
		}

		c.Logger.Infof("Waiting canary bake time in %s: %s", region.Region, region.Canary.BakeTime)
		time.Sleep(region.Canary.BakeTime)
	}
}

// ApplyCompleteCanaryWeights restores all listener traffic to the original target group.
func (c *Canary) ApplyCompleteCanaryWeights(config schemas.Config) error {
	for _, region := range c.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			c.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		stableWeight, canaryWeight := CanaryWeights(region, true)
		if err := c.ModifyListenerToStableTargetGroup(region, stableWeight, canaryWeight); err != nil {
			return err
		}
	}

	return nil
}

// ModifyWeightedListener updates the existing ALB listener's default action.
func (c *Canary) ModifyWeightedListener(region schemas.RegionConfig, stableWeight, canaryWeight int32) error {
	stableTargetGroupArn := c.OriginalTargetGroupArn[region.Region]
	canaryTargetGroupArn := c.CanaryTargetGroupArn[region.Region]
	if stableTargetGroupArn == "" || canaryTargetGroupArn == "" {
		return fmt.Errorf("canary target group information is missing for %s", region.Region)
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	listenerArn, err := c.ResolveCanaryListenerARN(region, client)
	if err != nil {
		return err
	}

	c.Logger.Infof("Changing canary listener weights in %s: stable=%d, canary=%d", region.Region, stableWeight, canaryWeight)
	return client.ELBV2Service.ModifyListenerWeightedForward(listenerArn, stableTargetGroupArn, canaryTargetGroupArn, stableWeight, canaryWeight)
}

// ModifyListenerToStableTargetGroup restores the listener default action to the original target group.
func (c *Canary) ModifyListenerToStableTargetGroup(region schemas.RegionConfig, stableWeight, canaryWeight int32) error {
	stableTargetGroupArn := c.OriginalTargetGroupArn[region.Region]
	if stableTargetGroupArn == "" {
		return fmt.Errorf("stable target group information is missing for %s", region.Region)
	}

	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	listenerArn, err := c.ResolveCanaryListenerARN(region, client)
	if err != nil {
		return err
	}

	c.Logger.Infof("Restoring canary listener weights in %s: stable=%d, canary=%d", region.Region, stableWeight, canaryWeight)
	return client.ELBV2Service.ModifyListener(&listenerArn, stableTargetGroupArn)
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

// CheckCanaryVersion checks latest version of canary target group
func CheckCanaryVersion(tgs []string, region string) int {
	latestVersion := 0
	for _, tg := range tgs {
		if tool.IsCanaryTargetGroupArn(tg, region) {
			name := tool.ParseTargetGroupName(tg)
			v := tool.ParseTargetGroupVersion(name)
			if v > 0 && v > latestVersion {
				latestVersion = v
			}
		}
	}

	return latestVersion
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
			if tool.IsCanaryTargetGroupArn(tg, region.Region) {
				c.Logger.Debugf("Try to delete target group: %s", tg)
				if err := client.ELBV2Service.DeleteTargetGroup(&tg); err != nil {
					return err
				}
				c.Logger.Debugf("Deleted target group: %s", tg)
			}
		}
	}

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
func (c *Canary) DetachCanaryTargetGroup(asg string, region schemas.RegionConfig, tgs []string) error {
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}

	var targets []string
	for _, tg := range tgs {
		if tool.IsCanaryTargetGroupArn(tg, region.Region) {
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

// DetachLatestCanaryResources detaches and deletes the canary target group after listener restore.
func (c *Canary) DetachLatestCanaryResources(config schemas.Config) error {
	for _, region := range c.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			c.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		canaryTargetGroupArn := c.CanaryTargetGroupArn[region.Region]
		if canaryTargetGroupArn == "" {
			continue
		}

		canaryASG := c.CanaryAutoScalingGroupName(region.Region)
		if canaryASG != "" {
			if err := c.DetachCanaryTargetGroup(canaryASG, region, []string{canaryTargetGroupArn}); err != nil {
				return err
			}
		}

		client, err := selectClientFromList(c.AWSClients, region.Region)
		if err != nil {
			return err
		}
		if err := client.ELBV2Service.DeleteTargetGroup(&canaryTargetGroupArn); err != nil {
			return err
		}
		c.Logger.Debugf("Deleted canary target group: %s", canaryTargetGroupArn)

		if canaryASG != "" {
			if err := c.RemoveCanaryTag(canaryASG, region); err != nil {
				return err
			}
		}
	}

	return nil
}

// RunCanaryDeployment runs canary deployment
func (c *Canary) RunCanaryDeployment(config schemas.Config, region schemas.RegionConfig, tgDetail *elbv2types.TargetGroup, canaryVersion int) (schemas.RegionConfig, error) {
	newTgName := c.GenerateCanaryTargetGroupName(canaryVersion)
	c.Logger.Debugf("New target group will be created for canary deployment: %s", newTgName)

	tg, err := c.CopyTargetGroups(tgDetail, newTgName, region.Region)
	if err != nil {
		return region, err
	}
	c.Logger.Debugf("New target group is created: %s", *tg.TargetGroupName)
	c.OriginalTargetGroupArn[region.Region] = *tgDetail.TargetGroupArn
	c.CanaryTargetGroupArn[region.Region] = *tg.TargetGroupArn

	c.Logger.Debugf("Change target group information with new target group: %s", newTgName)
	region = c.ChangeTargetGroupInfo(newTgName, region)
	c.Logger.Debugf("Changed information: %s / %s", region.HealthcheckTargetGroup, region.TargetGroups)

	if err := c.Deployer.Deploy(config, region); err != nil {
		return region, err
	}

	if err := c.ModifyWeightedListener(region, 100, 0); err != nil {
		return region, err
	}

	return region, nil
}

// CompleteCanaryDeployment completes canary deployment
func (c *Canary) CompleteCanaryDeployment(config schemas.Config, region schemas.RegionConfig, latestASG string, canaryTGDetail *elbv2types.TargetGroup) error {
	asgDetail, err := c.DescribeAutoScalingGroup(latestASG, region.Region)
	if err != nil {
		return err
	}

	if asgDetail == nil {
		return fmt.Errorf("no autoscaling group information retrieved. Please check autoscaling group resource: %s", latestASG)
	}

	originalTGDetail, err := c.DescribeTargetGroup(region.HealthcheckTargetGroup, region.Region)
	if err != nil {
		return err
	}
	c.OriginalTargetGroupArn[region.Region] = *originalTGDetail.TargetGroupArn
	c.CanaryTargetGroupArn[region.Region] = *canaryTGDetail.TargetGroupArn

	targetGroupARNs, err := c.TargetGroupARNsForRegion(region)
	if err != nil {
		return err
	}
	client, err := selectClientFromList(c.AWSClients, region.Region)
	if err != nil {
		return err
	}
	if err := client.EC2Service.AttachAsgToTargetGroups(latestASG, targetGroupARNs); err != nil {
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
	c.AppliedCapacity = &appliedCapacity
	c.AsgNames[region.Region] = latestASG
	c.PrevAsgs[region.Region] = withoutString(c.PrevAsgs[region.Region], latestASG)

	return nil
}

func withoutString(values []string, target string) []string {
	ret := values[:0]
	for _, value := range values {
		if value != target {
			ret = append(ret, value)
		}
	}
	return ret
}
