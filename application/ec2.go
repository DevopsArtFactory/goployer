package application

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	Logger "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strings"
)

type EC2Client struct {
	Client *ec2.EC2
	AsClient *autoscaling.AutoScaling
}

func NewEC2Client(session *session.Session, region string, creds *credentials.Credentials) EC2Client {
	return EC2Client{
		Client: _get_ec2_client_fn(session, region, creds),
		AsClient: _get_asg_client_fn(session, region, creds),
	}
}

func _get_ec2_client_fn(session *session.Session, region string, creds *credentials.Credentials) *ec2.EC2 {
	if creds == nil {
		return ec2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ec2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}


func _get_asg_client_fn(session *session.Session, region string, creds *credentials.Credentials) *autoscaling.AutoScaling {
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
				fmt.Println(autoscaling.ErrCodeScalingActivityInProgressFault, aerr.Error())
			case autoscaling.ErrCodeResourceInUseFault:
				fmt.Println(autoscaling.ErrCodeResourceInUseFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
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
		_fatal_error(err)
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
				fmt.Println(autoscaling.ErrCodeInvalidNextToken, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil
	}

	lcs = append(lcs, ret.LaunchConfigurations...)

	if ret.NextToken != nil {
		return getAllLaunchConfigurations(client, lcs, ret.NextToken)
	}

	return lcs
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
				fmt.Println(autoscaling.ErrCodeResourceInUseFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	return nil
}


// Create New Launch Configuration
func (e EC2Client) CreateNewLaunchConfiguration(name, ami, instanceType, keyName, iamProfileName, userdata string, ebsOptimized bool, securityGroups []*string, blockDevices []*autoscaling.BlockDeviceMapping) bool {
	input := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(name),
		ImageId: aws.String(ami),
		KeyName: aws.String(keyName),
		InstanceType: aws.String(instanceType),
		IamInstanceProfile: aws.String(iamProfileName),
		UserData: aws.String(userdata),
		SecurityGroups: securityGroups,
		EbsOptimized: aws.Bool(ebsOptimized),
		BlockDeviceMappings: blockDevices,
	}

	_, err := e.AsClient.CreateLaunchConfiguration(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeAlreadyExistsFault:
				fmt.Println(autoscaling.ErrCodeAlreadyExistsFault, aerr.Error())
			case autoscaling.ErrCodeLimitExceededFault:
				fmt.Println(autoscaling.ErrCodeLimitExceededFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false
	}

	Logger.Info("Successfully create new launch configurations : ", name)

	return true
}

// Get All Security Group Information New Launch Configuration
func (e EC2Client) GetSecurityGroupList(vpc string, sgList []string) []*string {
	if len (sgList) == 0 {
		error_logging("Need to specify at least one security group")
	}

	vpcId := e.GetVPCId(vpc)

	var retList []*string
	for _, sg := range sgList {
		if strings.HasPrefix(sg,"sg-") {
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
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}

			os.Exit(1)
		}

		//If it matches 0 or more than 1, it is wrong
		if len(result.SecurityGroups) != 1 {
			matched := []string{}
			for _, s := range result.SecurityGroups {
				matched = append(matched, *s.GroupName)
			}
			error_logging(fmt.Sprintf("Expected only one security group on name lookup for \"%s\" got \"%s\"", sg, strings.Join(matched, ",")))
		}

		retList = append(retList, aws.String(*result.SecurityGroups[0].GroupId))
	}

	return retList
}

func (e EC2Client) MakeBlockDevices(blocks []BlockDevice) []*autoscaling.BlockDeviceMapping {
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
			DeviceName:  aws.String(block.DeviceName),
			Ebs:         &autoscaling.Ebs{
				VolumeSize:          aws.Int64(bSize),
				VolumeType:          aws.String(bType),
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
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	// More than 1 vpc..
	if len (result.Vpcs) > 1 {
		error_logging(fmt.Sprintf("Expected only one VPC on name lookup for %v", vpc))
	}

	// No VPC found
	if len(result.Vpcs) < 1 {
		error_logging(fmt.Sprintf("Unable to find VPC on name lookup for %v", vpc))
	}

	return *result.Vpcs[0].VpcId
}

func (e EC2Client) CreateAutoScalingGroup(name, launch_config_name, healthcheck_type string, healthcheck_grace_period int64, capacity Capacity,  loadbalancers, target_group_arns, termination_policies, availability_zones []*string, tags []*(autoscaling.Tag), subnets []string) bool {
	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:    aws.String(name),
		LaunchConfigurationName: aws.String(launch_config_name),
		MaxSize:                 aws.Int64(capacity.Max),
		MinSize:                 aws.Int64(capacity.Min),
		DesiredCapacity:		 aws.Int64(capacity.Desired),
		AvailabilityZones:  	 availability_zones,
		HealthCheckType:  		 aws.String(healthcheck_type),
		HealthCheckGracePeriod:  aws.Int64(healthcheck_grace_period),
		TerminationPolicies:	 termination_policies,
		Tags: 				     tags,
		VPCZoneIdentifier: 		 aws.String(strings.Join(subnets,",")),
	}

	if *loadbalancers[0] != "" {
		input.LoadBalancerNames = loadbalancers
	}

	if *target_group_arns[0] != "" {
		input.TargetGroupARNs = target_group_arns
	}

	_, err := e.AsClient.CreateAutoScalingGroup(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeAlreadyExistsFault:
				fmt.Println(autoscaling.ErrCodeAlreadyExistsFault, aerr.Error())
			case autoscaling.ErrCodeLimitExceededFault:
				fmt.Println(autoscaling.ErrCodeLimitExceededFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			case autoscaling.ErrCodeServiceLinkedRoleFailure:
				fmt.Println(autoscaling.ErrCodeServiceLinkedRoleFailure, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false
	}

	Logger.Info("Successfully create new autoscaling group : ", name)

	return true
}

func (e EC2Client) GenerateTags(tagList []string, asg_name, app, stack string) []*autoscaling.Tag {
	ret := []*autoscaling.Tag{}

	for _, tagKV := range tagList {
		arr := strings.Split(tagKV, "=")
		k := arr[0]
		v := arr[1]

		ret = append(ret, &autoscaling.Tag{
			Key: aws.String(k),
			Value: aws.String(v),
		})
	}

	//Add Name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(asg_name),
	})

	//Add application name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("app"),
		Value: aws.String(app),
	})

	//Add stack name
	ret = append(ret, &autoscaling.Tag{
		Key:   aws.String("stack"),
		Value: aws.String(stack),
	})

	return ret
}

func (e EC2Client) GetAvailabilityZones(vpc string, azs []string) []string {
	ret := []string{}
	if len(azs) > 0 {
		for _, az := range azs {
			ret = append(ret, az)
		}
		return ret
	}

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
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	for _, subnet := range result.Subnets {
		if IsStringInArray(*subnet.AvailabilityZone, ret) {
			continue
		}
		ret = append(ret, *subnet.AvailabilityZone)
	}

	return ret
}

func (e EC2Client) GetSubnets(vpc string, use_public_subnets bool) []string {
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
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	ret := []string{}
	subnetType := "private"
	if use_public_subnets {
		subnetType = "public"
	}
	for _, subnet := range result.Subnets {
		for _, tag := range subnet.Tags {
			if *tag.Key == "Name" && strings.HasPrefix(*tag.Value, subnetType) {
				ret = append(ret, *subnet.SubnetId)
			}
		}
	}

	return ret
}

// Update Autoscaling Group size
func (e EC2Client) UpdateAutoScalingGroup(asg string, min, max, desired int64) error {
	input := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asg),
		MaxSize:              aws.Int64(max),
		MinSize:              aws.Int64(min),
		DesiredCapacity: 	  aws.Int64(desired),
	}

	_, err := e.AsClient.UpdateAutoScalingGroup(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeScalingActivityInProgressFault:
				fmt.Println(autoscaling.ErrCodeScalingActivityInProgressFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			case autoscaling.ErrCodeServiceLinkedRoleFailure:
				fmt.Println(autoscaling.ErrCodeServiceLinkedRoleFailure, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	return nil
}
