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

package aws

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type EC2Client struct {
	Client   *ec2.EC2
	AsClient *autoscaling.AutoScaling
}

func NewEC2Client(session client.ConfigProvider, region string, creds *credentials.Credentials) EC2Client {
	return EC2Client{
		Client:   getEC2ClientFn(session, region, creds),
		AsClient: getAsgClientFn(session, region, creds),
	}
}

func getEC2ClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *ec2.EC2 {
	if creds == nil {
		return ec2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ec2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func getAsgClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *autoscaling.AutoScaling {
	if creds == nil {
		return autoscaling.New(session, &aws.Config{Region: aws.String(region)})
	}
	return autoscaling.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func (e EC2Client) GetMatchingAutoscalingGroup(name string) (*autoscaling.Group, error) {
	asgGroup, err := getSingleAutoScalingGroup(e.AsClient, name)
	if err != nil {
		return nil, err
	}

	return asgGroup, nil
}

// GetMatchingLaunchTemplate returns information of launch template with matched ID
func (e EC2Client) GetMatchingLaunchTemplate(ltID string) (*ec2.LaunchTemplateVersion, error) {
	input := &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: aws.String(ltID),
	}

	ret, err := e.Client.DescribeLaunchTemplateVersions(input)
	if err != nil {
		return nil, err
	}

	return ret.LaunchTemplateVersions[0], nil
}

// GetSecurityGroupDetails returns detailed information for security group
func (e EC2Client) GetSecurityGroupDetails(sgIds []*string) ([]*ec2.SecurityGroup, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: sgIds,
	}

	result, err := e.Client.DescribeSecurityGroups(input)
	if err != nil {
		return nil, err
	}

	return result.SecurityGroups, nil
}

// Delete All Launch Configurations belongs to the autoscaling group
func (e EC2Client) DeleteLaunchConfigurations(asgName string) error {
	lcs := getAllLaunchConfigurations(e.AsClient, []*autoscaling.LaunchConfiguration{}, nil)

	for _, lc := range lcs {
		if strings.HasPrefix(*lc.LaunchConfigurationName, asgName) {
			err := deleteLaunchConfiguration(e.AsClient, *lc.LaunchConfigurationName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete all launch template belongs to the autoscaling group
func (e EC2Client) DeleteLaunchTemplates(asgName string) error {
	lts := getAllLaunchTemplates(e.Client, []*ec2.LaunchTemplate{}, nil)

	for _, lt := range lts {
		if strings.HasPrefix(*lt.LaunchTemplateName, asgName) {
			err := deleteLaunchTemplate(e.Client, *lt.LaunchTemplateName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete Autoscaling group Set
// 1. Autoscaling Group
// 2. Luanch Configurations in asg
func (e EC2Client) DeleteAutoscalingSet(asgName string) error {
	input := &autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
	}

	_, err := e.AsClient.DeleteAutoScalingGroup(input)
	if err != nil {
		return err
	}

	return nil
}

// Get All matching autoscaling groups with aws prefix
// By this function, you could get the latest version of deployment
func (e EC2Client) GetAllMatchingAutoscalingGroupsWithPrefix(prefix string) []*autoscaling.Group {
	asgGroups := []*autoscaling.Group{}
	asgGroups = getAutoScalingGroups(e.AsClient, asgGroups, nil)

	ret := []*autoscaling.Group{}
	for _, asgGroup := range asgGroups {
		if strings.HasPrefix(*asgGroup.AutoScalingGroupName, prefix) {
			ret = append(ret, asgGroup)
		}
	}

	return ret
}

// Batch of retrieving list of autoscaling group
// By Token, if needed, you could get all autoscaling groups with paging.
func getAutoScalingGroups(client *autoscaling.AutoScaling, asgGroup []*(autoscaling.Group), nextToken *string) []*autoscaling.Group {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		NextToken: nextToken,
	}
	ret, err := client.DescribeAutoScalingGroups(input)
	if err != nil {
		tool.FatalError(err)
	}

	asgGroup = append(asgGroup, ret.AutoScalingGroups...)

	if ret.NextToken != nil {
		return getAutoScalingGroups(client, asgGroup, ret.NextToken)
	}

	return asgGroup
}

// Batch of retrieving all launch configurations
func getAllLaunchConfigurations(client *autoscaling.AutoScaling, lcs []*autoscaling.LaunchConfiguration, nextToken *string) []*autoscaling.LaunchConfiguration {
	input := &autoscaling.DescribeLaunchConfigurationsInput{
		NextToken: nextToken,
	}

	ret, err := client.DescribeLaunchConfigurations(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeInvalidNextToken:
				Logger.Errorln(autoscaling.ErrCodeInvalidNextToken, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				Logger.Errorln(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		return nil
	}

	lcs = append(lcs, ret.LaunchConfigurations...)

	if ret.NextToken != nil {
		return getAllLaunchConfigurations(client, lcs, ret.NextToken)
	}

	return lcs
}

// Batch of retrieving all launch templates
func getAllLaunchTemplates(client *ec2.EC2, lts []*ec2.LaunchTemplate, nextToken *string) []*ec2.LaunchTemplate {
	input := &ec2.DescribeLaunchTemplatesInput{
		NextToken: nextToken,
	}

	ret, err := client.DescribeLaunchTemplates(input)
	if err != nil {
		return nil
	}

	lts = append(lts, ret.LaunchTemplates...)

	if ret.NextToken != nil {
		return getAllLaunchTemplates(client, lts, ret.NextToken)
	}

	return lts
}

// Delete Single Launch Configuration
func deleteLaunchConfiguration(client *autoscaling.AutoScaling, lcName string) error {
	input := &autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(lcName),
	}

	_, err := client.DeleteLaunchConfiguration(input)
	if err != nil {
		return err
	}

	return nil
}

// Delete Single Launch Template
func deleteLaunchTemplate(client *ec2.EC2, ltName string) error {
	input := &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: aws.String(ltName),
	}

	_, err := client.DeleteLaunchTemplate(input)
	if err != nil {
		return err
	}

	return nil
}

// Create New Launch Configuration
func (e EC2Client) CreateNewLaunchConfiguration(name, ami, instanceType, keyName, iamProfileName, userdata string, ebsOptimized bool, securityGroups []*string, blockDevices []*autoscaling.BlockDeviceMapping) bool {
	input := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(name),
		ImageId:                 aws.String(ami),
		KeyName:                 aws.String(keyName),
		InstanceType:            aws.String(instanceType),
		IamInstanceProfile:      aws.String(iamProfileName),
		UserData:                aws.String(userdata),
		SecurityGroups:          securityGroups,
		EbsOptimized:            aws.Bool(ebsOptimized),
		BlockDeviceMappings:     blockDevices,
	}

	_, err := e.AsClient.CreateLaunchConfiguration(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeAlreadyExistsFault:
				Logger.Errorln(autoscaling.ErrCodeAlreadyExistsFault, aerr.Error())
			case autoscaling.ErrCodeLimitExceededFault:
				Logger.Errorln(autoscaling.ErrCodeLimitExceededFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				Logger.Errorln(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		return false
	}

	Logger.Info("Successfully create new launch configurations : ", name)

	return true
}

// Create New Launch Template
func (e EC2Client) CreateNewLaunchTemplate(name, ami, instanceType, keyName, iamProfileName, userdata string, ebsOptimized, mixedInstancePolicyEnabled bool, securityGroups []*string, blockDevices []*ec2.LaunchTemplateBlockDeviceMappingRequest, instanceMarketOptions *schemas.InstanceMarketOptions, detailedMonitoringEnabled bool) error {
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:      aws.String(ami),
			InstanceType: aws.String(instanceType),
			KeyName:      aws.String(keyName),
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
				Name: aws.String(iamProfileName),
			},
			UserData:         aws.String(userdata),
			SecurityGroupIds: securityGroups,
			EbsOptimized:     aws.Bool(ebsOptimized),
			Monitoring:       &ec2.LaunchTemplatesMonitoringRequest{Enabled: aws.Bool(detailedMonitoringEnabled)},
		},
		LaunchTemplateName: aws.String(name),
	}

	if len(blockDevices) > 0 {
		input.LaunchTemplateData.BlockDeviceMappings = blockDevices
	}

	if instanceMarketOptions != nil && !mixedInstancePolicyEnabled {
		input.LaunchTemplateData.InstanceMarketOptions = &ec2.LaunchTemplateInstanceMarketOptionsRequest{
			MarketType:  aws.String(instanceMarketOptions.MarketType),
			SpotOptions: &ec2.LaunchTemplateSpotMarketOptionsRequest{},
		}

		if instanceMarketOptions.SpotOptions.BlockDurationMinutes > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes = aws.Int64(instanceMarketOptions.SpotOptions.BlockDurationMinutes)
		}

		if len(instanceMarketOptions.SpotOptions.InstanceInterruptionBehavior) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior = aws.String(instanceMarketOptions.SpotOptions.InstanceInterruptionBehavior)
		}

		if len(instanceMarketOptions.SpotOptions.SpotInstanceType) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType = aws.String(instanceMarketOptions.SpotOptions.SpotInstanceType)
		}

		if len(instanceMarketOptions.SpotOptions.MaxPrice) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice = aws.String(instanceMarketOptions.SpotOptions.MaxPrice)
		}
	}

	_, err := e.Client.CreateLaunchTemplate(input)
	if err != nil {
		return err
	}

	Logger.Info("Successfully create new launch template : ", name)

	return nil
}

// Get All Security Group Information New Launch Configuration
func (e EC2Client) GetSecurityGroupList(vpc string, sgList []string) ([]*string, error) {
	if len(sgList) == 0 {
		return nil, errors.New("need to specify at least one security group")
	}

	vpcID, err := e.GetVPCId(vpc)
	if err != nil {
		return nil, err
	}

	var retList []*string
	for _, sg := range sgList {
		if strings.HasPrefix(sg, "sg-") {
			retList = append(retList, aws.String(sg))
			continue
		}

		input := &ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("group-name"),
					Values: []*string{
						aws.String(sg),
					},
				},
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						aws.String(vpcID),
					},
				},
			},
		}

		result, err := e.Client.DescribeSecurityGroups(input)
		if err != nil {
			return nil, err
		}

		//If it matches 0 or more than 1, it is wrong
		if len(result.SecurityGroups) != 1 {
			matched := []string{}
			for _, s := range result.SecurityGroups {
				matched = append(matched, *s.GroupName)
			}
			return nil, fmt.Errorf("expected only one security group on name lookup for \"%s\" got \"%s\"", sg, strings.Join(matched, ","))
		}

		retList = append(retList, aws.String(*result.SecurityGroups[0].GroupId))
	}

	return retList, nil
}

// MakeBlockDevices returns list of block device mapping for launch configuration
func (e EC2Client) MakeBlockDevices(blocks []schemas.BlockDevice) []*autoscaling.BlockDeviceMapping {
	ret := []*autoscaling.BlockDeviceMapping{}

	for _, block := range blocks {
		bType := block.VolumeType
		if bType == "" {
			Logger.Info("Volume type not defined for device mapping: defaulting to \"gp2\"")
			bType = "gp2"
		}

		bSize := block.VolumeSize
		if bSize == 0 {
			Logger.Info("Volume size not defined for device mapping: defaulting to 16GB")
			bSize = 16
		}

		ret = append(ret, &autoscaling.BlockDeviceMapping{
			DeviceName: aws.String(block.DeviceName),
			Ebs: &autoscaling.Ebs{
				VolumeSize: aws.Int64(bSize),
				VolumeType: aws.String(bType),
			},
			NoDevice:    nil,
			VirtualName: nil,
		})
	}

	return ret
}

//MakeLaunchTemplateBlockDeviceMappings returns list of block device mappings for launch template
func (e EC2Client) MakeLaunchTemplateBlockDeviceMappings(blocks []schemas.BlockDevice) []*ec2.LaunchTemplateBlockDeviceMappingRequest {
	ret := []*ec2.LaunchTemplateBlockDeviceMappingRequest{}

	for _, block := range blocks {
		bType := block.VolumeType
		if bType == "" {
			Logger.Info("Default value is applied because volume type not defined : gp2")
			bType = "gp2"
		}

		bSize := block.VolumeSize
		if bSize == 0 {
			Logger.Info("Volume size not defined for device mapping: defaulting to 16GB")
			bSize = 16
		}

		tmp := ec2.LaunchTemplateBlockDeviceMappingRequest{
			DeviceName: aws.String(block.DeviceName),
			Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeSize: aws.Int64(bSize),
				VolumeType: aws.String(bType),
			},
			NoDevice:    nil,
			VirtualName: nil,
		}

		if tool.IsStringInArray(bType, constants.IopsRequiredBlockType) {
			tmp.Ebs.Iops = aws.Int64(block.Iops)
			Logger.Debugf("iops applied: %d", block.Iops)
		}

		ret = append(ret, &tmp)
	}

	return ret
}

func (e EC2Client) GetVPCId(vpc string) (string, error) {
	ret, err := regexp.MatchString("vpc-[0-9A-Fa-f]{17}", vpc)
	if err != nil {
		return constants.EmptyString, fmt.Errorf("error occurs when checking regex %v", err.Error())
	}

	if ret {
		return vpc, nil
	}

	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(vpc),
				},
			},
		},
	}

	result, err := e.Client.DescribeVpcs(input)
	if err != nil {
		return constants.EmptyString, err
	}

	// More than 1 vpc..
	if len(result.Vpcs) > 1 {
		return constants.EmptyString, fmt.Errorf("expected only one VPC on name lookup for %v", vpc)
	}

	// No VPC found
	if len(result.Vpcs) < 1 {
		return constants.EmptyString, fmt.Errorf("unable to find VPC on name lookup for %v", vpc)
	}

	return *result.Vpcs[0].VpcId, nil
}

// CreateAutoScalingGroup creates new autoscaling group
func (e EC2Client) CreateAutoScalingGroup(name, launchTemplateName, healthcheckType string,
	healthcheckGracePeriod int64,
	capacity schemas.Capacity,
	loadbalancers, availabilityZones []string,
	targetGroupArns, terminationPolicies []*string,
	tags []*(autoscaling.Tag),
	subnets []string,
	mixedInstancePolicy schemas.MixedInstancesPolicy,
	hooks []*autoscaling.LifecycleHookSpecification) (bool, error) {
	lt := autoscaling.LaunchTemplateSpecification{
		LaunchTemplateName: aws.String(launchTemplateName),
	}

	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:   aws.String(name),
		MaxSize:                aws.Int64(capacity.Max),
		MinSize:                aws.Int64(capacity.Min),
		DesiredCapacity:        aws.Int64(capacity.Desired),
		AvailabilityZones:      aws.StringSlice(availabilityZones),
		HealthCheckType:        aws.String(healthcheckType),
		HealthCheckGracePeriod: aws.Int64(healthcheckGracePeriod),
		TerminationPolicies:    terminationPolicies,
		Tags:                   tags,
		VPCZoneIdentifier:      aws.String(strings.Join(subnets, ",")),
	}

	if len(loadbalancers) > 0 {
		input.LoadBalancerNames = aws.StringSlice(loadbalancers)
	}

	if len(targetGroupArns) > 0 {
		input.TargetGroupARNs = targetGroupArns
	}

	if mixedInstancePolicy.Enabled {
		input.MixedInstancesPolicy = &autoscaling.MixedInstancesPolicy{
			InstancesDistribution: &autoscaling.InstancesDistribution{
				OnDemandBaseCapacity:   aws.Int64(mixedInstancePolicy.OnDemandBaseCapacity),
				SpotAllocationStrategy: aws.String(mixedInstancePolicy.SpotAllocationStrategy),
				SpotInstancePools:      aws.Int64(mixedInstancePolicy.SpotInstancePools),
				SpotMaxPrice:           aws.String(mixedInstancePolicy.SpotMaxPrice),
			},
			LaunchTemplate: &autoscaling.LaunchTemplate{
				LaunchTemplateSpecification: &lt,
			},
		}

		if mixedInstancePolicy.OnDemandPercentage > 0 {
			input.MixedInstancesPolicy.InstancesDistribution.OnDemandPercentageAboveBaseCapacity = aws.Int64(mixedInstancePolicy.OnDemandPercentage)
		}

		if len(mixedInstancePolicy.Override) != 0 {
			var overrides []*autoscaling.LaunchTemplateOverrides
			for _, o := range mixedInstancePolicy.Override {
				overrides = append(overrides, &autoscaling.LaunchTemplateOverrides{
					InstanceType: aws.String(o),
				})
			}

			input.MixedInstancesPolicy.LaunchTemplate.Overrides = overrides
		}
	} else {
		input.LaunchTemplate = &lt
	}

	if len(hooks) > 0 {
		input.LifecycleHookSpecificationList = hooks
	}

	_, err := e.AsClient.CreateAutoScalingGroup(input)
	if err != nil {
		return false, err
	}

	Logger.Info("Successfully create new autoscaling group : ", name)
	return true, nil
}

// GenerateTags creates tag list for autoscaling group
func (e EC2Client) GenerateTags(tagList []string, asgName, app, stack, ansibleTags string, stackTags []string, extraTags, ansibleExtraVars, region string) []*autoscaling.Tag {
	ret := []*autoscaling.Tag{}
	keyList := []string{}

	for _, tagKV := range tagList {
		arr := strings.Split(tagKV, "=")
		k := arr[0]
		v := arr[1]

		keyList = append(keyList, k)
		ret = append(ret, &autoscaling.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// Add Name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(asgName),
	})

	// Add stack name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("stack"),
		Value: aws.String(fmt.Sprintf("%s_%s", stack, strings.ReplaceAll(region, "-", ""))),
	})

	// Add pkg name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("app"),
		Value: aws.String(app),
	})

	// Add ansibleTags
	// This will be deprecated
	if len(ansibleTags) > 0 {
		ret = append(ret, &autoscaling.Tag{
			Key:   aws.String("ansible-tags"),
			Value: aws.String(ansibleTags),
		})
	}

	for _, t := range stackTags {
		arr := strings.Split(t, "=")
		k := arr[0]
		v := arr[1]

		if !tool.IsStringInArray(k, keyList) {
			ret = append(ret, &autoscaling.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
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
					Key:   aws.String(kv[0]),
					Value: aws.String(kv[1]),
				})
			}
		}
	}

	// Add ansibleExtraVars
	if len(ansibleExtraVars) > 0 {
		ret = append(ret, &autoscaling.Tag{
			Key:   aws.String("ansible-extra-vars"),
			Value: aws.String(ansibleExtraVars),
		})
	}

	return ret
}

// GetAvailabilityZones get all available availability zones
func (e EC2Client) GetAvailabilityZones(vpc string, azs []string) ([]string, error) {
	var ret []string
	vpcID, err := e.GetVPCId(vpc)
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcID),
				},
			},
		},
	}

	result, err := e.Client.DescribeSubnets(input)
	if err != nil {
		return nil, err
	}

	for _, subnet := range result.Subnets {
		if tool.IsStringInArray(*subnet.AvailabilityZone, ret) || (len(azs) > 0 && !tool.IsStringInArray(*subnet.AvailabilityZone, azs)) {
			continue
		}
		ret = append(ret, *subnet.AvailabilityZone)
	}

	return ret, nil
}

// GetSubnets retrieves all subnets available
func (e EC2Client) GetSubnets(vpc string, usePublicSubnets bool, azs []string) ([]string, error) {
	vpcID, err := e.GetVPCId(vpc)
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcID),
				},
			},
		},
	}

	result, err := e.Client.DescribeSubnets(input)
	if err != nil {
		return nil, err
	}

	ret := []string{}
	subnetType := "private"
	if usePublicSubnets {
		subnetType = "public"
	}
	for _, subnet := range result.Subnets {
		if !tool.IsStringInArray(*subnet.AvailabilityZone, azs) {
			continue
		}

		for _, tag := range subnet.Tags {
			if *tag.Key == "Name" && strings.HasPrefix(*tag.Value, subnetType) {
				ret = append(ret, *subnet.SubnetId)
			}
		}
	}

	return ret, nil
}

// Update Autoscaling Group size
func (e EC2Client) UpdateAutoScalingGroupSize(asg string, min, max, desired, retry int64) (int64, error) {
	input := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asg),
		MaxSize:              aws.Int64(max),
		MinSize:              aws.Int64(min),
		DesiredCapacity:      aws.Int64(desired),
	}

	_, err := e.AsClient.UpdateAutoScalingGroup(input)
	if err != nil {
		return retry - 1, err
	}

	return 0, nil
}

//CreateScalingPolicy creates scaling policy
func (e EC2Client) CreateScalingPolicy(policy schemas.ScalePolicy, asgName string) (*string, error) {
	input := &autoscaling.PutScalingPolicyInput{
		AdjustmentType:       aws.String(policy.AdjustmentType),
		AutoScalingGroupName: aws.String(asgName),
		PolicyName:           aws.String(policy.Name),
		ScalingAdjustment:    aws.Int64(policy.ScalingAdjustment),
		Cooldown:             aws.Int64(policy.Cooldown),
	}

	result, err := e.AsClient.PutScalingPolicy(input)
	if err != nil {
		return nil, err
	}

	return result.PolicyARN, nil
}

// EnableMetrics enables metric monitoring of autoscaling group
func (e EC2Client) EnableMetrics(asgName string) error {
	input := &autoscaling.EnableMetricsCollectionInput{
		AutoScalingGroupName: aws.String(asgName),
		Granularity:          aws.String("1Minute"),
	}

	_, err := e.AsClient.EnableMetricsCollection(input)
	if err != nil {
		return err
	}

	Logger.Info(fmt.Sprintf("Metrics monitoring of autoscaling group is enabled : %s", asgName))

	return nil
}

// Generate Lifecycle Hooks
func (e EC2Client) GenerateLifecycleHooks(hooks schemas.LifecycleHooks) []*autoscaling.LifecycleHookSpecification {
	var ret []*autoscaling.LifecycleHookSpecification

	if len(hooks.LaunchTransition) > 0 {
		for _, l := range hooks.LaunchTransition {
			lhs := createSingleLifecycleHookSpecification(l, "autoscaling:EC2_INSTANCE_LAUNCHING")
			ret = append(ret, &lhs)
		}
	}

	if len(hooks.TerminateTransition) > 0 {
		for _, l := range hooks.TerminateTransition {
			lhs := createSingleLifecycleHookSpecification(l, "autoscaling:EC2_INSTANCE_TERMINATING")
			ret = append(ret, &lhs)
		}
	}

	return ret
}

// createSingleLifecycleHookSpecification create a lifecycle hook specification
func createSingleLifecycleHookSpecification(l schemas.LifecycleHookSpecification, transition string) autoscaling.LifecycleHookSpecification {
	lhs := autoscaling.LifecycleHookSpecification{
		LifecycleHookName:   aws.String(l.LifecycleHookName),
		LifecycleTransition: aws.String(transition),
	}

	if len(l.DefaultResult) > 0 {
		lhs.DefaultResult = aws.String(l.DefaultResult)
	}

	if l.HeartbeatTimeout > 0 {
		lhs.HeartbeatTimeout = aws.Int64(l.HeartbeatTimeout)
	}

	if len(l.NotificationMetadata) > 0 {
		lhs.NotificationMetadata = aws.String(l.NotificationMetadata)
	}

	if len(l.NotificationTargetARN) > 0 {
		lhs.NotificationTargetARN = aws.String(l.NotificationTargetARN)
	}

	if len(l.RoleARN) > 0 {
		lhs.RoleARN = aws.String(l.RoleARN)
	}

	return lhs
}

// GetTargetGroup returns list of target group ARN of autoscaling group
func (e EC2Client) GetTargetGroups(asgName string) ([]*string, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(asgName),
		},
	}

	result, err := e.AsClient.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	}

	var ret []*string
	for _, a := range result.AutoScalingGroups {
		ret = a.TargetGroupARNs
	}

	return ret, nil
}

// getSingleAutoScalingGroup return detailed information of autoscaling group
func getSingleAutoScalingGroup(client *autoscaling.AutoScaling, asgName string) (*autoscaling.Group, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice([]string{asgName}),
	}
	ret, err := client.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	}

	if len(ret.AutoScalingGroups) == 0 {
		return nil, fmt.Errorf("no autoscaling group exists with name: %s", asgName)
	}

	return ret.AutoScalingGroups[0], nil
}

// Update Autoscaling Group information
func (e EC2Client) UpdateAutoScalingGroup(asg string, capacity schemas.Capacity) error {
	input := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asg),
		MaxSize:              aws.Int64(capacity.Max),
		MinSize:              aws.Int64(capacity.Min),
		DesiredCapacity:      aws.Int64(capacity.Desired),
	}

	_, err := e.AsClient.UpdateAutoScalingGroup(input)
	if err != nil {
		return err
	}

	return nil
}

// CreateScheduledActions creates scheduled actions
func (e EC2Client) CreateScheduledActions(asg string, actions []schemas.ScheduledAction) error {
	input := &autoscaling.BatchPutScheduledUpdateGroupActionInput{
		AutoScalingGroupName: aws.String(asg),
	}

	scheduledUpdateGroupActions := []*autoscaling.ScheduledUpdateGroupActionRequest{}
	for _, a := range actions {
		newSa := autoscaling.ScheduledUpdateGroupActionRequest{
			ScheduledActionName: aws.String(a.Name),
			Recurrence:          aws.String(a.Recurrence),
			MinSize:             aws.Int64(a.Capacity.Min),
			DesiredCapacity:     aws.Int64(a.Capacity.Desired),
			MaxSize:             aws.Int64(a.Capacity.Max),
		}

		scheduledUpdateGroupActions = append(scheduledUpdateGroupActions, &newSa)
	}

	input.ScheduledUpdateGroupActions = scheduledUpdateGroupActions

	_, err := e.AsClient.BatchPutScheduledUpdateGroupAction(input)
	if err != nil {
		return err
	}

	return nil
}
