package aws

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	Logger "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strings"
)

type EC2Client struct {
	Client   *ec2.EC2
	AsClient *autoscaling.AutoScaling
}

func NewEC2Client(session *session.Session, region string, creds *credentials.Credentials) EC2Client {
	return EC2Client{
		Client:   getEC2ClientFn(session, region, creds),
		AsClient: getAsgClientFn(session, region, creds),
	}
}

func getEC2ClientFn(session *session.Session, region string, creds *credentials.Credentials) *ec2.EC2 {
	if creds == nil {
		return ec2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ec2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func getAsgClientFn(session *session.Session, region string, creds *credentials.Credentials) *autoscaling.AutoScaling {
	if creds == nil {
		return autoscaling.New(session, &aws.Config{Region: aws.String(region)})
	}
	return autoscaling.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func (e EC2Client) GetMatchingAutoscalingGroup(name string) *autoscaling.Group {

	asgGroups := []*autoscaling.Group{}
	asgGroups = getAutoScalingGroups(e.AsClient, asgGroups, nil)

	ret := []*autoscaling.Group{}
	for _, asgGroup := range asgGroups {
		if *asgGroup.AutoScalingGroupName == name {
			ret = append(ret, asgGroup)
		}
	}

	if len(ret) > 0 {
		return ret[0]
	}

	return nil
}

// Delete All Launch Configurations belongs to the autoscaling group
func (e EC2Client) DeleteLaunchConfigurations(asg_name string) error {
	lcs := getAllLaunchConfigurations(e.AsClient, []*autoscaling.LaunchConfiguration{}, nil)

	for _, lc := range lcs {
		if strings.HasPrefix(*lc.LaunchConfigurationName, asg_name) {
			err := deleteLaunchConfiguration(e.AsClient, *lc.LaunchConfigurationName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete all launch template belongs to the autoscaling group
func (e EC2Client) DeleteLaunchTemplates(asg_name string) error {
	lts := getAllLaunchTemplates(e.Client, []*ec2.LaunchTemplate{}, nil)

	for _, lt := range lts {
		if strings.HasPrefix(*lt.LaunchTemplateName, asg_name) {
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
func (e EC2Client) DeleteAutoscalingSet(asg_name string) bool {
	input := &autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asg_name),
	}

	_, err := e.AsClient.DeleteAutoScalingGroup(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeScalingActivityInProgressFault:
				Logger.Errorln(autoscaling.ErrCodeScalingActivityInProgressFault, aerr.Error())
			case autoscaling.ErrCodeResourceInUseFault:
				Logger.Errorln(autoscaling.ErrCodeResourceInUseFault, aerr.Error())
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

	return true
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
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			Logger.Errorln(err.Error())
		}
		return nil
	}

	lts = append(lts, ret.LaunchTemplates...)

	if ret.NextToken != nil {
		return getAllLaunchTemplates(client, lts, ret.NextToken)
	}

	return lts
}

// Delete Single Launch Configuration
func deleteLaunchConfiguration(client *autoscaling.AutoScaling, lc_name string) error {
	input := &autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(lc_name),
	}

	_, err := client.DeleteLaunchConfiguration(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeResourceInUseFault:
				Logger.Errorln(autoscaling.ErrCodeResourceInUseFault, aerr.Error())
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
		return err
	}

	return nil
}

// Delete Single Launch Template
func deleteLaunchTemplate(client *ec2.EC2, lt_name string) error {
	input := &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: aws.String(lt_name),
	}

	_, err := client.DeleteLaunchTemplate(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			Logger.Errorln(err.Error())
		}
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
func (e EC2Client) CreateNewLaunchTemplate(name, ami, instanceType, keyName, iamProfileName, userdata string, ebsOptimized, mixedInstancePolicyEnabled bool, securityGroups []*string, blockDevices []*ec2.LaunchTemplateBlockDeviceMappingRequest, instanceMarketOptions *builder.InstanceMarketOptions) bool {
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
		},
		LaunchTemplateName: aws.String(name),
	}

	if len(blockDevices) > 0 {
		input.LaunchTemplateData.BlockDeviceMappings = blockDevices
	}

	if instanceMarketOptions != nil && !mixedInstancePolicyEnabled {
		input.LaunchTemplateData.InstanceMarketOptions = &ec2.LaunchTemplateInstanceMarketOptionsRequest{
			MarketType: aws.String(instanceMarketOptions.MarketType),
			SpotOptions: &ec2.LaunchTemplateSpotMarketOptionsRequest{
				BlockDurationMinutes:         aws.Int64(instanceMarketOptions.SpotOptions.BlockDurationMinutes),
				InstanceInterruptionBehavior: aws.String(instanceMarketOptions.SpotOptions.InstanceInterruptionBehavior),
				SpotInstanceType:             aws.String(instanceMarketOptions.SpotOptions.SpotInstanceType),
			},
		}

		if len(instanceMarketOptions.SpotOptions.MaxPrice) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice = aws.String(instanceMarketOptions.SpotOptions.MaxPrice)
		}
	}

	_, err := e.Client.CreateLaunchTemplate(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
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

	Logger.Info("Successfully create new launch template : ", name)

	return true
}

// Get All Security Group Information New Launch Configuration
func (e EC2Client) GetSecurityGroupList(vpc string, sgList []string) []*string {
	if len(sgList) == 0 {
		tool.ErrorLogging("Need to specify at least one security group")
	}

	vpcId := e.GetVPCId(vpc)

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
						aws.String(vpcId),
					},
				},
			},
		}

		result, err := e.Client.DescribeSecurityGroups(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					Logger.Errorln(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				Logger.Errorln(err.Error())
			}

			os.Exit(1)
		}

		//If it matches 0 or more than 1, it is wrong
		if len(result.SecurityGroups) != 1 {
			matched := []string{}
			for _, s := range result.SecurityGroups {
				matched = append(matched, *s.GroupName)
			}
			tool.ErrorLogging(fmt.Sprintf("Expected only one security group on name lookup for \"%s\" got \"%s\"", sg, strings.Join(matched, ",")))
		}

		retList = append(retList, aws.String(*result.SecurityGroups[0].GroupId))
	}

	return retList
}

// MakeBlockDevices returns list of block device mapping for launch configuration
func (e EC2Client) MakeBlockDevices(blocks []builder.BlockDevice) []*autoscaling.BlockDeviceMapping {
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
func (e EC2Client) MakeLaunchTemplateBlockDeviceMappings(blocks []builder.BlockDevice) []*ec2.LaunchTemplateBlockDeviceMappingRequest {
	ret := []*ec2.LaunchTemplateBlockDeviceMappingRequest{}

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

		ret = append(ret, &ec2.LaunchTemplateBlockDeviceMappingRequest{
			DeviceName: aws.String(block.DeviceName),
			Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeSize: aws.Int64(bSize),
				VolumeType: aws.String(bType),
			},
			NoDevice:    nil,
			VirtualName: nil,
		})
	}

	return ret
}

func (e EC2Client) GetVPCId(vpc string) string {
	ret, err := regexp.MatchString("vpc-[0-9A-Fa-f]{17}", vpc)
	if err != nil {
		fmt.Errorf("Error occurs when checking regex %v", err.Error())
		os.Exit(1)
	}

	if ret {
		return vpc
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
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		os.Exit(1)
	}

	// More than 1 vpc..
	if len(result.Vpcs) > 1 {
		tool.ErrorLogging(fmt.Sprintf("Expected only one VPC on name lookup for %v", vpc))
	}

	// No VPC found
	if len(result.Vpcs) < 1 {
		tool.ErrorLogging(fmt.Sprintf("Unable to find VPC on name lookup for %v", vpc))
	}

	return *result.Vpcs[0].VpcId
}

func (e EC2Client) CreateAutoScalingGroup(name, launch_template_name, healthcheck_type string,
	healthcheck_grace_period int64,
	capacity builder.Capacity,
	loadbalancers, target_group_arns, termination_policies, availability_zones []*string,
	tags []*(autoscaling.Tag),
	subnets []string,
	mixedInstancePolicy builder.MixedInstancesPolicy,
	hooks []*autoscaling.LifecycleHookSpecification) bool {

	lt := autoscaling.LaunchTemplateSpecification{
		LaunchTemplateName: aws.String(launch_template_name),
	}

	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:   aws.String(name),
		MaxSize:                aws.Int64(capacity.Max),
		MinSize:                aws.Int64(capacity.Min),
		DesiredCapacity:        aws.Int64(capacity.Desired),
		AvailabilityZones:      availability_zones,
		HealthCheckType:        aws.String(healthcheck_type),
		HealthCheckGracePeriod: aws.Int64(healthcheck_grace_period),
		TerminationPolicies:    termination_policies,
		Tags:                   tags,
		VPCZoneIdentifier:      aws.String(strings.Join(subnets, ",")),
	}

	if len(loadbalancers) > 0 {
		input.LoadBalancerNames = loadbalancers
	}

	if len(target_group_arns) > 0 {
		input.TargetGroupARNs = target_group_arns
	}

	if mixedInstancePolicy.Enabled {
		input.MixedInstancesPolicy = &autoscaling.MixedInstancesPolicy{
			InstancesDistribution: &autoscaling.InstancesDistribution{
				OnDemandPercentageAboveBaseCapacity: aws.Int64(mixedInstancePolicy.OnDemandPercentage),
				SpotAllocationStrategy:              aws.String(mixedInstancePolicy.SpotAllocationStrategy),
				SpotInstancePools:                   aws.Int64(mixedInstancePolicy.SpotInstancePools),
				SpotMaxPrice:                        aws.String(mixedInstancePolicy.SpotMaxPrice),
			},
			LaunchTemplate: &autoscaling.LaunchTemplate{
				LaunchTemplateSpecification: &lt,
			},
		}

		if len(mixedInstancePolicy.Override) != 0 {
			overrides := []*autoscaling.LaunchTemplateOverrides{}
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
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeAlreadyExistsFault:
				Logger.Errorln(autoscaling.ErrCodeAlreadyExistsFault, aerr.Error())
			case autoscaling.ErrCodeLimitExceededFault:
				Logger.Errorln(autoscaling.ErrCodeLimitExceededFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				Logger.Errorln(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			case autoscaling.ErrCodeServiceLinkedRoleFailure:
				Logger.Errorln(autoscaling.ErrCodeServiceLinkedRoleFailure, aerr.Error())
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

	Logger.Info("Successfully create new autoscaling group : ", name)

	return true
}

// GenerateTags creates tag list for autoscaling group
func (e EC2Client) GenerateTags(tagList []string, asg_name, app, stack, ansibleTags, extraTags, ansibleExtraVars, region string) []*autoscaling.Tag {
	ret := []*autoscaling.Tag{}

	for _, tagKV := range tagList {
		arr := strings.Split(tagKV, "=")
		k := arr[0]
		v := arr[1]

		ret = append(ret, &autoscaling.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	//Add Name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(asg_name),
	})

	//Add pkg name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("app"),
		Value: aws.String(app),
	})

	//Add stack name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("stack"),
		Value: aws.String(fmt.Sprintf("%s_%s", stack, strings.ReplaceAll(region, "-", ""))),
	})

	//Add ansibleTags
	if len(ansibleTags) > 0 {
		ret = append(ret, &autoscaling.Tag{
			Key:   aws.String("ansible-tags"),
			Value: aws.String(ansibleTags),
		})
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

func (e EC2Client) GetAvailabilityZones(vpc string, azs []string) []string {
	ret := []string{}
	vpcId := e.GetVPCId(vpc)

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	result, err := e.Client.DescribeSubnets(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		os.Exit(1)
	}

	for _, subnet := range result.Subnets {
		if tool.IsStringInArray(*subnet.AvailabilityZone, ret) || (len(azs) > 0 && !tool.IsStringInArray(*subnet.AvailabilityZone, azs)) {
			continue
		}
		ret = append(ret, *subnet.AvailabilityZone)
	}

	return ret
}

func (e EC2Client) GetSubnets(vpc string, use_public_subnets bool, azs []string) []string {
	vpcId := e.GetVPCId(vpc)

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	result, err := e.Client.DescribeSubnets(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		os.Exit(1)
	}

	ret := []string{}
	subnetType := "private"
	if use_public_subnets {
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

	return ret
}

// Update Autoscaling Group size
func (e EC2Client) UpdateAutoScalingGroup(asg string, min, max, desired, retry int64) (error, int64) {
	input := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asg),
		MaxSize:              aws.Int64(max),
		MinSize:              aws.Int64(min),
		DesiredCapacity:      aws.Int64(desired),
	}

	Logger.Info(input)

	_, err := e.AsClient.UpdateAutoScalingGroup(input)
	if err != nil {
		return err, retry - 1
	}

	return nil, 0
}

//CreateScalingPolicy creates scaling policy
func (e EC2Client) CreateScalingPolicy(policy builder.ScalePolicy, asg_name string) (*string, error) {
	input := &autoscaling.PutScalingPolicyInput{
		AdjustmentType:       aws.String(policy.AdjustmentType),
		AutoScalingGroupName: aws.String(asg_name),
		PolicyName:           aws.String(policy.Name),
		ScalingAdjustment:    aws.Int64(policy.ScalingAdjustment),
		Cooldown:             aws.Int64(policy.Cooldown),
	}

	result, err := e.AsClient.PutScalingPolicy(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeLimitExceededFault:
				Logger.Errorln(autoscaling.ErrCodeLimitExceededFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				Logger.Errorln(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			case autoscaling.ErrCodeServiceLinkedRoleFailure:
				Logger.Errorln(autoscaling.ErrCodeServiceLinkedRoleFailure, aerr.Error())
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		return nil, err
	}

	return result.PolicyARN, nil
}

// EnableMetrics enables metric monitoring of autoscaling group
func (e EC2Client) EnableMetrics(asg_name string) error {
	input := &autoscaling.EnableMetricsCollectionInput{
		AutoScalingGroupName: aws.String(asg_name),
		Granularity:          aws.String("1Minute"),
	}

	_, err := e.AsClient.EnableMetricsCollection(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
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
		return err
	}

	Logger.Info(fmt.Sprintf("Metrics monitoring of autoscaling group is enabled : %s", asg_name))

	return nil
}

// Generate Lifecycle Hooks
func (e EC2Client) GenerateLifecycleHooks(hooks builder.LifecycleHooks) []*autoscaling.LifecycleHookSpecification {
	ret := []*autoscaling.LifecycleHookSpecification{}

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

func createSingleLifecycleHookSpecification(l builder.LifecycleHookSpecification, transition string) autoscaling.LifecycleHookSpecification {
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
