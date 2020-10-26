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
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	eaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/olekukonko/tablewriter"
	Logger "github.com/sirupsen/logrus"
	vegeta "github.com/tsenart/vegeta/lib"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/collector"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/slack"
	"github.com/DevopsArtFactory/goployer/pkg/templates"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

// Deployer per stack
type Deployer struct {
	Mode              string
	AsgNames          map[string]string
	PrevAsgs          map[string][]string
	PrevInstances     map[string][]string
	PrevVersions      map[string][]int
	PrevInstanceCount map[string]schemas.Capacity
	SecurityGroup     map[string]*string
	LatestAsg         map[string]string
	Logger            *Logger.Logger
	Stack             schemas.Stack
	AwsConfig         schemas.AWSConfig
	APITestTemplate   *schemas.APITestTemplate
	AWSClients        []aws.Client
	LocalProvider     builder.UserdataProvider
	Slack             slack.Slack
	Collector         collector.Collector
	StepStatus        map[int64]bool
	CanaryFlag        map[string]bool
}

type APIAttacker struct {
	Name     string
	Attacker *vegeta.Attacker
	Rate     vegeta.Rate
	Duration time.Duration
	Targets  []vegeta.Target
}

// Polling is polling healthy information from instance/target group
func (d *Deployer) Polling(region schemas.RegionConfig, asg *autoscaling.Group, client aws.Client, forceManifestCapacity, isUpdate, downsizingUpdate bool) (bool, error) {
	if *asg.AutoScalingGroupName == "" {
		return false, fmt.Errorf("no autoscaling found for %s", d.AsgNames[region.Region])
	}

	var threshold int64
	if !forceManifestCapacity && d.PrevInstanceCount[region.Region].Desired > d.Stack.Capacity.Desired && d.Mode != constants.CanaryDeployment {
		threshold = d.PrevInstanceCount[region.Region].Desired
	} else {
		threshold = d.Stack.Capacity.Desired
	}

	if region.HealthcheckTargetGroup == "" && region.HealthcheckLB == "" {
		d.Logger.Info("health check skipped because of neither target group nor classic load balancer specified")
		return true, nil
	}

	var targetHosts []aws.HealthcheckHost
	var err error
	validHostCount := int64(0)

	d.Logger.Debugf("[Checking healthy host count] Autoscaling Group: %s", *asg.AutoScalingGroupName)
	if len(region.HealthcheckTargetGroup) > 0 {
		var healthCheckTargetGroupArn *string
		if tool.IsTargetGroupArn(region.HealthcheckTargetGroup, region.Region) {
			healthCheckTargetGroupArn = &region.HealthcheckTargetGroup
		} else {
			tgs := []string{region.HealthcheckTargetGroup}
			tgARNs, err := client.ELBV2Service.GetTargetGroupARNs(tgs)
			if err != nil {
				return false, err
			}
			healthCheckTargetGroupArn = tgARNs[0]
		}
		d.Logger.Debugf("[Checking healthy host count] Target Group : %s", *healthCheckTargetGroupArn)

		targetHosts, err = client.ELBV2Service.GetHostInTarget(asg, healthCheckTargetGroupArn, isUpdate, downsizingUpdate)
		if err != nil {
			return false, err
		}
	} else if len(region.HealthcheckLB) > 0 {
		d.Logger.Debugf("[Checking healthy host count] Load Balancer : %s", region.HealthcheckLB)
		targetHosts, err = client.ELBService.GetHealthyHostInELB(asg, region.HealthcheckLB)
		if err != nil {
			return false, err
		}
	}

	validHostCount = d.GetValidHostCount(targetHosts)

	if isUpdate {
		if validHostCount == threshold {
			d.Logger.Infof("[Update completed] current / desired : %d/%d", validHostCount, threshold)
			return true, nil
		}
		d.Logger.Infof("Desired count does not meet the requirement: %d/%d", validHostCount, threshold)
	} else {
		if validHostCount >= threshold {
			d.Logger.Infof("Healthy Count for %s : %d/%d", d.AsgNames[region.Region], validHostCount, threshold)
			d.Slack.SendSimpleMessage(fmt.Sprintf("All instances are healthy in %s  :  %d/%d", d.AsgNames[region.Region], validHostCount, threshold))
			return true, nil
		}

		d.Logger.Infof("Healthy count does not meet the requirement(%s) : %d/%d", d.AsgNames[region.Region], validHostCount, threshold)
		d.Slack.SendSimpleMessage(fmt.Sprintf("Waiting for healthy instances %s  :  %d/%d", d.AsgNames[region.Region], validHostCount, threshold))
	}
	return false, nil
}

// CheckTerminating checks if all of instances are terminated well
func (d *Deployer) CheckTerminating(client aws.Client, target string, disableMetrics bool) bool {
	asgInfo, err := client.EC2Service.GetMatchingAutoscalingGroup(target)
	if err != nil {
		d.Logger.Errorf(err.Error())
		return true
	}

	if asgInfo == nil {
		d.Logger.Infof("Already deleted autoscaling group : %s", target)
		return true
	}

	d.Logger.Infof("Waiting for instance termination in asg %s", target)
	if len(asgInfo.Instances) > 0 {
		d.Logger.Infof("%d instance found : %s", len(asgInfo.Instances), target)
		d.Slack.SendSimpleMessage(fmt.Sprintf("Still %d instance found : %s", len(asgInfo.Instances), target))

		return false
	}
	d.Slack.SendSimpleMessage(fmt.Sprintf(":+1: All instances are deleted : %s", target))

	if err := d.CleanAutoscalingSet(client, target); err != nil {
		d.Logger.Errorf(err.Error())
		return false
	}

	if !disableMetrics {
		d.Logger.Debugf("update status of autoscaling group to teminated : %s", target)
		if err := d.Collector.UpdateStatus(target, "terminated", nil); err != nil {
			d.Logger.Errorf(err.Error())
			return false
		}
		d.Logger.Debugf("update status of %s is finished", target)
	}

	d.Logger.Debugf("Start deleting launch templates in %s", target)
	if err := client.EC2Service.DeleteLaunchTemplates(target); err != nil {
		d.Logger.Errorln(err.Error())
		return false
	}
	d.Logger.Debugf("Launch templates are deleted in %s\n", target)

	return true
}

// CleanAutoscalingSet cleans autoscaling group itself
func (d *Deployer) CleanAutoscalingSet(client aws.Client, target string) error {
	d.Logger.Debugf("Start deleting autoscaling group : %s", target)
	if err := client.EC2Service.DeleteAutoscalingSet(target); err != nil {
		return err
	}
	d.Logger.Debugf("Autoscaling group is deleted : %s", target)

	return nil
}

// ResizingAutoScalingGroupToZero set autoscaling group instance count to 0
func (d *Deployer) ResizingAutoScalingGroupToZero(client aws.Client, asg string) error {
	d.Logger.Info(fmt.Sprintf("Modifying the size of autoscaling group to 0 : %s(%s)", asg, d.Stack.Stack))
	d.Slack.SendSimpleMessage(fmt.Sprintf("Modifying the size of autoscaling group to 0 : %s/%s", asg, d.Stack.Stack))

	retry := int64(3)
	var err error
	for {
		retry, err = client.EC2Service.UpdateAutoScalingGroupSize(asg, 0, 0, 0, retry)
		if err != nil {
			if retry > 0 {
				d.Logger.Debugf("error occurred and remained retry count is %d", retry)
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

// RunLifecycleCallbacks runs commands before terminating.
func (d *Deployer) RunLifecycleCallbacks(client aws.Client, region string) bool {
	target := d.PrevInstances[region]
	if len(target) == 0 {
		d.Logger.Debugf("no target instance exists\n")
		return false
	}

	commands := d.Stack.LifecycleCallbacks.PreTerminatePastClusters

	d.Logger.Debugf("run lifecycle callbacks before termination : %s", target)
	return client.SSMService.SendCommand(
		eaws.StringSlice(target),
		eaws.StringSlice(commands),
	)
}

// selectClientFromList get aws client.
func selectClientFromList(awsClients []aws.Client, region string) (aws.Client, error) {
	for _, c := range awsClients {
		if c.Region == region {
			return c, nil
		}
	}
	return aws.Client{}, errors.New("no AWS Client is selected")
}

// GatherMetrics gathers metrics of autoscaling group
func (d *Deployer) GatherMetrics(client aws.Client, asg string) error {
	targetGroups, err := client.EC2Service.GetTargetGroups(asg)
	if err != nil {
		return err
	}

	if len(targetGroups) == 0 {
		d.Logger.Warnf("this autoscaling group does not belong to any target group ")
		return nil
	}

	lbs, err := client.ELBV2Service.GetLoadBalancerFromTG(targetGroups)
	if err != nil {
		return err
	}

	d.Logger.Debugf("start retrieving additional metrics")
	metricData, err := d.Collector.GetAdditionalMetric(asg, targetGroups, lbs, d.Logger)
	if err != nil {
		return err
	}

	d.Logger.Debugf("start updating additional metrics to DynamoDB")
	if err := d.Collector.UpdateStatistics(asg, metricData); err != nil {
		return err
	}
	d.Logger.Debugf("finish updating additional metrics to DynamoDB")

	return nil
}

// GetValidHostCount return the number of health host
func (d *Deployer) GetValidHostCount(targetHosts []aws.HealthcheckHost) int64 {
	ret := 0
	var data [][]string
	for _, host := range targetHosts {
		data = append(data, []string{host.InstanceID, host.LifecycleState, host.TargetStatus, host.HealthStatus, fmt.Sprintf("%t", host.Valid)})
		if host.Valid {
			ret++
		}
	}

	if len(data) > 0 {
		printCurrentHostStatus(data)
	}

	return int64(ret)
}

// GenerateAPIAttacker create API Attacker
func (d *Deployer) GenerateAPIAttacker(template schemas.APITestTemplate) (*APIAttacker, error) {
	attacker := APIAttacker{
		Name:     template.Name,
		Rate:     vegeta.Rate{Freq: template.RequestPerSecond, Per: time.Second},
		Duration: template.Duration,
		Attacker: vegeta.NewAttacker(),
	}

	var targets []vegeta.Target
	for _, api := range template.APIs {
		tempT := vegeta.Target{
			Method: strings.ToUpper(api.Method),
			URL:    api.URL,
		}

		if len(api.Body) > 0 {
			b, err := tool.CreateBodyStruct(api.Body)
			if err != nil {
				return nil, err
			}

			tempT.Body = b
		}

		if len(api.Header) > 0 {
			h, err := tool.CreateHeaderStruct(api.Header)
			if err != nil {
				return nil, err
			}

			tempT.Header = h
		} else {
			tempT.Header = tool.SetCommonHeader()
		}

		targets = append(targets, tempT)
	}
	attacker.Targets = targets

	return &attacker, nil
}

// Run calls apis to check
func (a APIAttacker) Run() ([]schemas.MetricResult, error) {
	var result []schemas.MetricResult
	wg := sync.WaitGroup{}
	for _, tgt := range a.Targets {
		wg.Add(1)
		go func(tgt vegeta.Target) {
			defer wg.Done()
			metrics := vegeta.Metrics{}
			tgtr := vegeta.NewStaticTargeter(tgt)
			for res := range a.Attacker.Attack(tgtr, a.Rate, a.Duration, a.Name) {
				metrics.Add(res)
			}
			metrics.Close()

			result = append(result, schemas.MetricResult{
				URL:    tgt.URL,
				Method: tgt.Method,
				Data:   metrics,
			})
		}(tgt)
	}

	wg.Wait()

	return result, nil
}

// Print shows results
func (a APIAttacker) Print(metrics []schemas.MetricResult) (string, error) {
	var data = struct {
		Metrics []schemas.MetricResult
		Name    string
	}{
		Metrics: metrics,
		Name:    a.Name,
	}

	funcMap := template.FuncMap{
		"decorate": tool.DecorateAttr,
		"round":    tool.RoundTime,
		"roundNum": tool.RoundNum,
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("API Test Result").Funcs(funcMap).Parse(templates.APITestResultTemplate))

	err := t.Execute(w, data)
	if err != nil {
		return constants.EmptyString, err
	}

	str := buf.String()
	fmt.Println(str)

	return str, nil
}

// printCurrentHostStatus shows current instance status
func printCurrentHostStatus(data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Instance ID", "Lifecycle State", "Target Status", "Health Status", "Valid"})
	table.SetCenterSeparator("|")
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)

	table.AppendBulk(data)
	table.Render()
}

// GetStackName returns name of stack
func (d *Deployer) GetStackName() string {
	return d.Stack.Stack
}

// SkipDeployStep skips deployment processes
func (d *Deployer) SkipDeployStep() {
	d.StepStatus[constants.StepDeploy] = true
	d.StepStatus[constants.StepAdditionalWork] = true
}

// CheckPrevious checks if there is any previous version of autoscaling group
func (d *Deployer) CheckPrevious(config schemas.Config) error {
	// Make Frigga
	frigga := tool.Frigga{}
	for _, region := range d.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		var prevAsgs []string
		var prevInstanceIds []string
		var prevVersions []int
		canaryFlag := false

		//Setup frigga with prefix
		frigga.Prefix = tool.BuildPrefixName(d.AwsConfig.Name, d.Stack.Env, region.Region)

		//select client
		client, err := selectClientFromList(d.AWSClients, region.Region)
		if err != nil {
			return err
		}

		// Get All Autoscaling Groups
		asgGroups, err := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(frigga.Prefix)
		if err != nil {
			return err
		}

		//Get All Previous Autoscaling Groups and versions
		var prevInstanceCount schemas.Capacity
		var latestAsg *autoscaling.Group
		for _, asgGroup := range asgGroups {
			prevVersions = append(prevVersions, tool.ParseAutoScalingVersion(*asgGroup.AutoScalingGroupName))
			for _, instance := range asgGroup.Instances {
				prevInstanceIds = append(prevInstanceIds, *instance.InstanceId)
			}

			prevInstanceCount.Desired = *asgGroup.DesiredCapacity
			prevInstanceCount.Max = *asgGroup.MaxSize
			prevInstanceCount.Min = *asgGroup.MinSize

			if d.Mode == constants.CanaryDeployment {
				isCanaryAuto := false
				for _, tag := range asgGroup.Tags {
					if *tag.Key == constants.DeploymentTagKey && *tag.Value == constants.CanaryDeployment {
						canaryFlag = true
						isCanaryAuto = true
						break
					}
				}

				if isCanaryAuto || config.CompleteCanary {
					prevAsgs = append(prevAsgs, *asgGroup.AutoScalingGroupName)
				}
			} else {
				prevAsgs = append(prevAsgs, *asgGroup.AutoScalingGroupName)
			}

			if latestAsg == nil || asgGroup.CreatedTime.Sub(*latestAsg.CreatedTime) > 0 {
				latestAsg = asgGroup
			}
		}

		if latestAsg == nil && d.Mode == constants.CanaryDeployment {
			d.StepStatus[constants.StepCheckPrevious] = true
		}
		d.Logger.Infof("Previous Versions : %s", strings.Join(prevAsgs, " | "))

		d.PrevAsgs[region.Region] = prevAsgs
		d.PrevInstances[region.Region] = prevInstanceIds
		d.PrevVersions[region.Region] = prevVersions
		d.PrevInstanceCount[region.Region] = prevInstanceCount
		d.CanaryFlag[region.Region] = canaryFlag
		if latestAsg != nil {
			d.LatestAsg[region.Region] = *latestAsg.AutoScalingGroupName
			d.Logger.Infof("Latest autoscaling group version : %s", *latestAsg.AutoScalingGroupName)
		}

		d.Logger.Debugf("Previous deployment with canary: %t", canaryFlag)
	}

	d.StepStatus[constants.StepCheckPrevious] = true
	return nil
}

// Deploy is a basic deployment process for any deployment method
func (d *Deployer) Deploy(config schemas.Config, region schemas.RegionConfig) error {
	var terminationPolicies []*string
	var lifecycleHooksSpecificationList []*autoscaling.LifecycleHookSpecification

	// Make Frigga
	frigga := tool.Frigga{}

	//select client
	client, err := selectClientFromList(d.AWSClients, region.Region)
	if err != nil {
		return err
	}

	//Setup frigga with prefix
	frigga.Prefix = tool.BuildPrefixName(d.AwsConfig.Name, d.Stack.Env, region.Region)

	// Get Current Version
	curVersion := getCurrentVersion(d.PrevVersions[region.Region])
	d.Logger.Infof("Current Version: %d", curVersion)

	//Get AMI
	var ami string
	if len(config.Ami) > 0 {
		ami = config.Ami
	} else {
		ami = region.AmiID
	}

	// Generate new name for autoscaling group and launch configuration
	newAsgName := tool.GenerateAsgName(frigga.Prefix, curVersion)
	d.Logger.Debugf("New autoscaling group name: %s", newAsgName)

	launchTemplateName := tool.GenerateLcName(newAsgName)
	d.Logger.Debugf("New launch template name: %s", launchTemplateName)

	userdata, err := d.LocalProvider.Provide()
	if err != nil {
		return err
	}

	//Stack check
	securityGroups, err := client.EC2Service.GetSecurityGroupList(region.VPC, region.SecurityGroups)
	if err != nil {
		return err
	}

	if d.SecurityGroup[region.Region] != nil {
		securityGroups = append(securityGroups, d.SecurityGroup[region.Region])
		d.Logger.Debugf("additional security group applied to %s: %s", newAsgName, *d.SecurityGroup[region.Region])
	}

	blockDevices := client.EC2Service.MakeLaunchTemplateBlockDeviceMappings(d.Stack.BlockDevices)
	ebsOptimized := d.Stack.EbsOptimized

	// Instance Type Override
	instanceType := region.InstanceType
	if len(config.OverrideInstanceType) > 0 {
		instanceType = config.OverrideInstanceType

		if d.Stack.MixedInstancesPolicy.Enabled {
			d.Logger.Warnf("--override-instance-type won't be applied because mixed_instances_policy is enabled")
		}

		d.Logger.Debugf("Instance type is overridden with %s", config.OverrideInstanceType)
	}

	// LaunchTemplate
	err = client.EC2Service.CreateNewLaunchTemplate(
		launchTemplateName,
		ami,
		instanceType,
		region.SSHKey,
		d.Stack.IamInstanceProfile,
		userdata,
		ebsOptimized,
		d.Stack.MixedInstancesPolicy.Enabled,
		securityGroups,
		blockDevices,
		d.Stack.InstanceMarketOptions,
		region.DetailedMonitoringEnabled,
	)

	if err != nil {
		return err
	}

	healthElb := region.HealthcheckLB
	loadBalancers := region.LoadBalancers
	if healthElb != "" && !tool.IsStringInArray(healthElb, loadBalancers) {
		loadBalancers = append(loadBalancers, healthElb)
	}

	targetGroups := d.GetTargetGroupNames(region)

	healthCheckType := constants.DefaultHealthcheckType
	healthCheckGracePeriod := int64(constants.DefaultHealthcheckGracePeriod)
	tags := d.GenerateTags(newAsgName, d.Stack.Stack, config.ExtraTags, config.AnsibleExtraVars, region.Region)

	availabilityZones, err := client.EC2Service.GetAvailabilityZones(region.VPC, region.AvailabilityZones)
	if err != nil {
		return err
	}
	subnets, err := client.EC2Service.GetSubnets(region.VPC, region.UsePublicSubnets, availabilityZones)
	if err != nil {
		return err
	}

	targetGroupARNs, err := client.ELBV2Service.GetTargetGroupARNs(targetGroups)
	if err != nil {
		return err
	}

	appliedCapacity, err := d.DecideCapacity(config.ForceManifestCapacity, config.CompleteCanary, region.Region)
	if err != nil {
		return err
	}

	d.Logger.Infof("Applied instance capacity - Min: %d, Desired: %d, Max: %d", appliedCapacity.Min, appliedCapacity.Desired, appliedCapacity.Max)

	if d.Stack.LifecycleHooks != nil {
		lifecycleHooksSpecificationList = client.EC2Service.GenerateLifecycleHooks(*d.Stack.LifecycleHooks)
	}

	if len(region.TerminationPolicies) > 0 {
		terminationPolicies = eaws.StringSlice(region.TerminationPolicies)
	}

	err = client.EC2Service.CreateAutoScalingGroup(
		newAsgName,
		launchTemplateName,
		healthCheckType,
		healthCheckGracePeriod,
		appliedCapacity,
		loadBalancers,
		availabilityZones,
		targetGroupARNs,
		terminationPolicies,
		tags,
		subnets,
		d.Stack.MixedInstancesPolicy,
		lifecycleHooksSpecificationList,
	)

	if err != nil {
		return err
	}

	if d.Collector.MetricConfig.Enabled {
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

		if err := d.Collector.StampDeployment(d.Stack, config, tags, newAsgName, "creating", additionalFields); err != nil {
			d.Logger.Error(err.Error())
		}
	}

	d.AsgNames[region.Region] = newAsgName
	d.Stack.Capacity.Desired = appliedCapacity.Desired

	return nil
}

// DecideCapacity returns Applied Capacity for deployment
func (d *Deployer) DecideCapacity(forceManifestCapacity, completeCanary bool, region string) (schemas.Capacity, error) {
	if d.Mode == constants.CanaryDeployment && !completeCanary {
		return schemas.Capacity{
			Min:     1,
			Max:     1,
			Desired: 1,
		}, nil
	}

	var appliedCapacity schemas.Capacity

	if !forceManifestCapacity && d.PrevInstanceCount[region].Desired > d.Stack.Capacity.Desired {
		appliedCapacity = d.PrevInstanceCount[region]
	} else {
		appliedCapacity = d.Stack.Capacity
	}

	return appliedCapacity, nil
}

// GenerateTags creates tag list for autoscaling group
func (d *Deployer) GenerateTags(asgName, stack string, extraTags, ansibleExtraVars, region string) []*autoscaling.Tag {
	var ret []*autoscaling.Tag
	var keyList []string
	for _, tagKV := range d.AwsConfig.Tags {
		arr := strings.Split(tagKV, "=")
		k := arr[0]
		v := arr[1]

		keyList = append(keyList, k)
		ret = append(ret, &autoscaling.Tag{
			Key:   eaws.String(k),
			Value: eaws.String(v),
		})
	}

	// Add Name
	ret = append(ret, &autoscaling.Tag{
		Key:   eaws.String("Name"),
		Value: eaws.String(asgName),
	})

	// Add stack name
	ret = append(ret, &autoscaling.Tag{
		Key:   eaws.String("stack"),
		Value: eaws.String(fmt.Sprintf("%s_%s", stack, strings.ReplaceAll(region, "-", ""))),
	})

	// Add pkg name
	ret = append(ret, &autoscaling.Tag{
		Key:   eaws.String("app"),
		Value: eaws.String(d.AwsConfig.Name),
	})

	// Add ansibleTags
	// This will be deprecated
	if len(d.Stack.AnsibleTags) > 0 {
		ret = append(ret, &autoscaling.Tag{
			Key:   eaws.String("ansible-tags"),
			Value: eaws.String(d.Stack.AnsibleTags),
		})
	}

	for _, t := range d.Stack.Tags {
		arr := strings.Split(t, "=")
		k := arr[0]
		v := arr[1]

		if !tool.IsStringInArray(k, keyList) {
			ret = append(ret, &autoscaling.Tag{
				Key:   eaws.String(k),
				Value: eaws.String(v),
			})
		} else {
			for _, t := range ret {
				if *t.Key == k {
					*t.Value = v
					break
				}
			}
		}
	}

	//Add extraTags
	if len(extraTags) > 0 {
		if strings.Contains(extraTags, ",") {
			ts := strings.Split(extraTags, ",")
			for _, s := range ts {
				if !strings.Contains(strings.TrimSpace(s), "=") {
					Logger.Warnln("extra-tags usage : --extra-tags=key1=value1,key2=value2...")
					continue
				}

				kv := strings.Split(strings.TrimSpace(s), "=")
				ret = append(ret, &autoscaling.Tag{
					Key:   eaws.String(kv[0]),
					Value: eaws.String(kv[1]),
				})
			}
		}
	}

	// Add ansibleExtraVars
	if len(ansibleExtraVars) > 0 {
		ret = append(ret, &autoscaling.Tag{
			Key:   eaws.String("ansible-extra-vars"),
			Value: eaws.String(ansibleExtraVars),
		})
	}

	// Canary Tags
	if d.Mode == constants.CanaryDeployment {
		ret = append(ret, &autoscaling.Tag{
			Key:   eaws.String(constants.DeploymentTagKey),
			Value: eaws.String(constants.CanaryDeployment),
		})
	}

	return ret
}

// GetTargetGroupNames retrieves slice of target group name string
func (d *Deployer) GetTargetGroupNames(region schemas.RegionConfig) []string {
	healthCheckTargetGroup := region.HealthcheckTargetGroup

	targetGroups := region.TargetGroups
	if healthCheckTargetGroup != "" && !tool.IsStringInArray(healthCheckTargetGroup, targetGroups) {
		targetGroups = append(targetGroups, healthCheckTargetGroup)
	}

	return targetGroups
}

// DescribeTargetGroups retrieves target group details
func (d *Deployer) DescribeTargetGroup(targetGroup string, region string) (*elbv2.TargetGroup, error) {
	client, err := selectClientFromList(d.AWSClients, region)
	if err != nil {
		return nil, err
	}

	ret, err := client.ELBV2Service.DescribeTargetGroups(eaws.StringSlice([]string{targetGroup}))
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, fmt.Errorf("target group details not found: %s", targetGroup)
	}
	d.Logger.Debugf("Successfully retrieved target group details: %s", *ret[0].TargetGroupName)

	return ret[0], nil
}

// HealthChecking does health check
func (d *Deployer) HealthChecking(config schemas.Config) (bool, error) {
	var finished []string

	isUpdate := len(config.TargetAutoscalingGroup) > 0
	stackName := d.GetStackName()
	if !d.StepStatus[constants.StepDeploy] && !isUpdate {
		return true, nil
	}
	d.Logger.Debugf("Health checking for stack starts : %s", stackName)

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(d.Stack.Regions)
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, d.Stack.Regions) {
			validCount = 0
		}
	}

	for _, region := range d.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debugf("This region is skipped by user: %s", region.Region)
			continue
		}
		d.Logger.Debugf("Health checking for region starts: %s", region.Region)

		//select client
		client, err := selectClientFromList(d.AWSClients, region.Region)
		if err != nil {
			return false, err
		}

		var targetAsgName string
		if len(config.TargetAutoscalingGroup) > 0 {
			targetAsgName = config.TargetAutoscalingGroup
		} else {
			targetAsgName = d.AsgNames[region.Region]
		}
		d.Logger.Debugf("Target autoscaling group for health check: %s / %s", region.Region, targetAsgName)

		asg, err := client.EC2Service.GetMatchingAutoscalingGroup(targetAsgName)
		if err != nil {
			return false, err
		}
		d.Logger.Debugf("Health check target autoscaling group: %s / %s", region.Region, *asg.AutoScalingGroupName)

		isHealthy, err := d.Polling(region, asg, client, config.ForceManifestCapacity, isUpdate, config.DownSizingUpdate)
		if err != nil {
			return false, err
		}

		if isHealthy {
			if d.Collector.MetricConfig.Enabled {
				if err := d.Collector.UpdateStatus(*asg.AutoScalingGroupName, "deployed", nil); err != nil {
					d.Logger.Errorf("Update status Error, %s : %s", err.Error(), *asg.AutoScalingGroupName)
				}
			}
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return true, nil
	}

	return false, nil
}

// DoCommonAdditionalWork does the common work regardless of replacement type
func (d *Deployer) DoCommonAdditionalWork(config schemas.Config) error {
	//Apply AutoScaling Policies
	for _, region := range d.Stack.Regions {
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		d.Logger.Info("Attaching autoscaling policies : " + region.Region)

		//select client
		client, err := selectClientFromList(d.AWSClients, region.Region)
		if err != nil {
			return err
		}

		if len(d.Stack.Autoscaling) == 0 {
			d.Logger.Debug("no scaling policy exists")
		} else {
			//putting autoscaling group policies
			policyArns := map[string]string{}
			for _, policy := range d.Stack.Autoscaling {
				policyArn, err := client.EC2Service.CreateScalingPolicy(policy, d.AsgNames[region.Region])
				if err != nil {
					return err
				}
				d.Logger.Debugf("policy arn created: %s", *policyArn)
				policyArns[policy.Name] = *policyArn
			}

			if err := client.EC2Service.EnableMetrics(d.AsgNames[region.Region]); err != nil {
				return err
			}

			if err := client.CloudWatchService.CreateScalingAlarms(d.AsgNames[region.Region], d.Stack.Alarms, policyArns); err != nil {
				return err
			}
		}

		if len(region.ScheduledActions) > 0 {
			d.Logger.Debugf("create scheduled actions")
			selectedActions := []schemas.ScheduledAction{}
			for _, sa := range d.AwsConfig.ScheduledActions {
				if tool.IsStringInArray(sa.Name, region.ScheduledActions) {
					selectedActions = append(selectedActions, sa)
				}
			}

			d.Logger.Debugf("selected actions [ %s ]", strings.Join(region.ScheduledActions, ","))
			if err := client.EC2Service.CreateScheduledActions(d.AsgNames[region.Region], selectedActions); err != nil {
				return err
			}
			d.Logger.Debugf("finished adding scheduled actions")
		}
	}

	return nil
}

// TriggerLifecycleCallbacks runs lifecycle callbacks before cleaning.
func (d *Deployer) TriggerLifecycleCallbacks(config schemas.Config) error {
	skipped := false
	if d.Stack.LifecycleCallbacks == nil || len(d.Stack.LifecycleCallbacks.PreTerminatePastClusters) == 0 {
		d.Logger.Debugf("no lifecycle callbacks in %s\n", d.Stack.Stack)
		skipped = true
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, d.Stack.Regions) {
			d.Logger.Debugf("region [ %s ] is not in the stack [ %s ].", config.Region, d.Stack.Stack)
			skipped = true
		}
	}

	if !skipped {
		for _, region := range d.Stack.Regions {
			if config.Region != "" && config.Region != region.Region {
				d.Logger.Debug("This region is skipped by user : " + region.Region)
				continue
			}

			//select client
			client, err := selectClientFromList(d.AWSClients, region.Region)
			if err != nil {
				return err
			}

			if len(d.PrevInstances[region.Region]) > 0 {
				d.Logger.Debugf("Run lifecycle callbacks: %s", d.PrevInstances[region.Region])
				d.RunLifecycleCallbacks(client, region.Region)
			} else {
				d.Logger.Debugf("No previous versions to be deleted : %s\n", region.Region)
				d.Slack.SendSimpleMessage(fmt.Sprintf("No previous versions to be deleted : %s\n", region.Region))
			}
		}
	}
	d.StepStatus[constants.StepTriggerLifecycleCallback] = true
	return nil
}

//CleanPreviousAutoScalingGroup cleans previous version of autoscaling group
func (d Deployer) CleanPreviousAutoScalingGroup(config schemas.Config) error {
	for _, region := range d.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debugf("This region is skipped by user: %s", region.Region)
			continue
		}

		d.Logger.Infof("[%s]The number of previous versions to delete is %d", region.Region, len(d.PrevAsgs[region.Region]))

		//select client
		client, err := selectClientFromList(d.AWSClients, region.Region)
		if err != nil {
			return err
		}

		// First make autoscaling group size to 0
		if len(d.PrevAsgs[region.Region]) > 0 {
			for _, asg := range d.PrevAsgs[region.Region] {
				if _, ok := d.LatestAsg[region.Region]; ok && asg == d.LatestAsg[region.Region] && d.Mode == constants.CanaryDeployment {
					continue
				}

				d.Logger.Debugf("[Resizing to 0] target autoscaling group : %s", asg)
				if err := d.ResizingAutoScalingGroupToZero(client, asg); err != nil {
					d.Logger.Errorf(err.Error())
				}
			}
		} else {
			d.Logger.Infof("No previous versions to be deleted : %s", region.Region)
			d.Slack.SendSimpleMessage(fmt.Sprintf("No previous versions to be deleted : %s\n", region.Region))
		}
	}

	return nil
}

// CleanChecking checks if instances in previous autoscaling groups are terminated or not
func (d *Deployer) CleanChecking(config schemas.Config) (bool, error) {
	var finished []string

	stackName := d.GetStackName()
	if !d.StepStatus[constants.StepCleanPreviousVersion] {
		return true, nil
	}
	d.Logger.Info(fmt.Sprintf("Termination Checking for %s starts...", stackName))

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(d.Stack.Regions)
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, d.Stack.Regions) {
			validCount = 0
		}
	}

	for _, region := range d.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debugf("This region is skipped by user : %s", region.Region)
			continue
		}

		d.Logger.Infof("Checking Termination stack for region starts : %s", region.Region)

		//select client
		client, err := selectClientFromList(d.AWSClients, region.Region)
		if err != nil {
			return false, err
		}

		targets := d.PrevAsgs[region.Region]
		if len(targets) == 0 {
			d.Logger.Infof("No target to delete : %s", region.Region)
			finished = append(finished, region.Region)
			continue
		}

		okCount := 0
		for _, target := range targets {
			ok := d.CheckTerminating(client, target, config.DisableMetrics)
			if ok {
				d.Logger.Infof("finished : %s", target)
				okCount++
			}
		}

		if okCount == len(targets) {
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return true, nil
	}

	return false, nil
}

// StartGatheringMetrics starts to gather the whole metrics from deployer
func (d *Deployer) StartGatheringMetrics(config schemas.Config) error {
	for _, region := range d.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		d.Logger.Infof("[%s]The number of previous autoscaling groups for gathering metrics is %d", region.Region, len(d.PrevAsgs[region.Region]))

		//select client
		client, err := selectClientFromList(d.AWSClients, region.Region)
		if err != nil {
			return err
		}

		if len(d.PrevAsgs[region.Region]) > 0 {
			var errorList []error
			for _, asg := range d.PrevAsgs[region.Region] {
				d.Logger.Debugf("Start gathering metrics about autoscaling group : %s", asg)
				err := d.GatherMetrics(client, asg)
				if err != nil {
					errorList = append(errorList, err)
				}
				d.Logger.Debugf("Finish gathering metrics about autoscaling group %s.", asg)
			}

			if len(errorList) > 0 {
				for _, e := range errorList {
					d.Logger.Errorf(e.Error())
				}
				return errors.New("error occurred on gathering metrics")
			}
		} else {
			d.Logger.Debugf("No previous versions to gather metrics : %s\n", region.Region)
		}
	}

	return nil
}

// RunAPITest tries to run API Test
func (d *Deployer) RunAPITest(config schemas.Config) error {
	if !d.StepStatus[constants.StepCleanChecking] {
		return nil
	}

	if !d.Stack.APITestEnabled {
		d.Logger.Infof("API test is disabled for this stack: %s", d.Stack.Stack)
		return nil
	}

	d.Logger.Debugf("Create API attacker")
	attacker, err := d.GenerateAPIAttacker(*d.APITestTemplate)
	if err != nil {
		return err
	}

	d.Logger.Debugf("Run API attacker")
	result, err := attacker.Run()
	if err != nil {
		return err
	}

	d.Logger.Debugf("Print API test result")
	_, err = attacker.Print(result)
	if err != nil {
		return err
	}

	if err := d.Slack.SendAPITestResultMessage(result); err != nil {
		return err
	}
	d.Logger.Debugf("API test is done")

	d.StepStatus[constants.StepRunAPI] = true
	return nil
}

// DescribeAutoScalingGroup describes autoscaling group
func (d *Deployer) DescribeAutoScalingGroup(asg string, region schemas.RegionConfig) (*autoscaling.Group, error) {
	client, err := selectClientFromList(d.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	ret, err := client.EC2Service.GetMatchingAutoscalingGroup(asg)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// DescribeInstances describes instances
func (d *Deployer) DescribeInstances(instanceIds []*string, region schemas.RegionConfig) ([]*ec2.Instance, error) {
	client, err := selectClientFromList(d.AWSClients, region.Region)
	if err != nil {
		return nil, err
	}

	ret, err := client.EC2Service.DescribeInstances(instanceIds)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// getCurrentVersion returns current version for current deployment step
func getCurrentVersion(prevVersions []int) int {
	if len(prevVersions) == 0 {
		return 0
	}
	return (prevVersions[len(prevVersions)-1] + 1) % 1000
}
