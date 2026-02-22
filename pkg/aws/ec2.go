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
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type EC2Client struct {
	Client    *ec2.EC2
	AsClient  *autoscaling.AutoScaling
	KMSClient *kms.KMS
}

func NewEC2Client(session client.ConfigProvider, region string, creds *credentials.Credentials) EC2Client {
	return EC2Client{
		Client:    getEC2ClientFn(session, region, creds),
		AsClient:  getAsgClientFn(session, region, creds),
		KMSClient: getKMSClientFn(session, region, creds),
	}
}

// getEC2ClientFn creates an EC2 client
func getEC2ClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *ec2.EC2 {
	if creds == nil {
		return ec2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ec2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

// getAsgClientFn creates an Autoscaling client
func getAsgClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *autoscaling.AutoScaling {
	if creds == nil {
		return autoscaling.New(session, &aws.Config{Region: aws.String(region)})
	}
	return autoscaling.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

// getEC2ClientFn creates an EC2 client
func getKMSClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *kms.KMS {
	if creds == nil {
		return kms.New(session, &aws.Config{Region: aws.String(region)})
	}
	return kms.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

// GetMatchingAutoscalingGroup returns only one matching autoscaling group information
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

// DeleteLaunchConfigurations deletes All Launch Configurations belongs to the autoscaling group
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

// DeleteLaunchTemplates deletes all launch template belongs to the autoscaling group
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

// DeleteAutoscalingSet Delete Autoscaling group Set
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

// GetAllMatchingAutoscalingGroupsWithPrefix Get All matching autoscaling groups with aws prefix
// By this function, you could get the latest version of deployment
func (e EC2Client) GetAllMatchingAutoscalingGroupsWithPrefix(prefix string) ([]*autoscaling.Group, error) {
	asgGroups, err := getAutoScalingGroups(e.AsClient, []*autoscaling.Group{}, nil)
	if err != nil {
		return nil, err
	}

	var ret []*autoscaling.Group
	for _, asgGroup := range asgGroups {
		if strings.HasPrefix(*asgGroup.AutoScalingGroupName, prefix) {
			ret = append(ret, asgGroup)
		}
	}

	return ret, nil
}

// Batch of retrieving list of autoscaling group
// By Token, if needed, you could get all autoscaling groups with paging.
func getAutoScalingGroups(client *autoscaling.AutoScaling, asgGroup []*(autoscaling.Group), nextToken *string) ([]*autoscaling.Group, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		NextToken: nextToken,
	}

	ret, err := client.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	}

	asgGroup = append(asgGroup, ret.AutoScalingGroups...)

	if ret.NextToken != nil {
		return getAutoScalingGroups(client, asgGroup, ret.NextToken)
	}

	return asgGroup, nil
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

// CreateNewLaunchConfiguration Create New Launch Configuration
func (e EC2Client) CreateNewLaunchConfiguration(name, ami, instanceType, keyName, iamProfileName, userdata string, ebsOptimized bool, securityGroups []*string, blockDevices []*autoscaling.BlockDeviceMapping) bool {
	input := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(name),
		ImageId:                 aws.String(ami),
		KeyName:                 aws.String(keyName),
		InstanceType:            aws.String(instanceType),
		UserData:                aws.String(userdata),
		SecurityGroups:          securityGroups,
		EbsOptimized:            aws.Bool(ebsOptimized),
		BlockDeviceMappings:     blockDevices,
	}

	if len(iamProfileName) > 0 {
		input.IamInstanceProfile = aws.String(iamProfileName)
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

// ValidateSecurityGroupsConfig validates the security group configuration
func (e EC2Client) ValidateSecurityGroupsConfig(securityGroups []*string, primaryENI *schemas.ENIConfig, secondaryENIs []*schemas.ENIConfig) error {
	// Check if both security groups and ENI are specified
	if len(securityGroups) > 0 && (primaryENI != nil || len(secondaryENIs) > 0) {
		return fmt.Errorf("cannot use both launch template security groups and ENI security groups at the same time")
	}

	// If ENI is specified, ensure it has security groups
	if primaryENI != nil && len(primaryENI.SecurityGroups) == 0 {
		return fmt.Errorf("security groups must be specified for primary ENI")
	}
	for _, eni := range secondaryENIs {
		if len(eni.SecurityGroups) == 0 {
			return fmt.Errorf("security groups must be specified for secondary ENI")
		}
	}

	// If ENI is not specified, ensure launch template has security groups
	if primaryENI == nil && len(secondaryENIs) == 0 && len(securityGroups) == 0 {
		return fmt.Errorf("security groups must be specified for launch template when ENI is not used")
	}

	// Validate security group IDs
	if len(securityGroups) > 0 {
		for _, sg := range securityGroups {
			if sg == nil || !strings.HasPrefix(*sg, "sg-") {
				return fmt.Errorf("invalid security group ID format: %v", sg)
			}
		}
	}

	return nil
}

// CreateNewLaunchTemplate creates a new launch template
func (e EC2Client) CreateNewLaunchTemplate(name, ami, instanceType, keyName, iamProfileName, userdata string, ebsOptimized, mixedInstancePolicyEnabled bool, securityGroups []*string, blockDevices []*ec2.LaunchTemplateBlockDeviceMappingRequest, instanceMarketOptions *schemas.InstanceMarketOptions, detailedMonitoringEnabled bool, primaryENI *schemas.ENIConfig, secondaryENIs []*schemas.ENIConfig, tags []string, httpPutResponseHopLimit int64) error {
	// Validate security group configuration
	if err := e.ValidateSecurityGroupsConfig(securityGroups, primaryENI, secondaryENIs); err != nil {
		return err
	}

	// Set default hop limit if not specified
	if httpPutResponseHopLimit <= 0 {
		httpPutResponseHopLimit = 1
	}

	launchTemplateData := &ec2.RequestLaunchTemplateData{
		ImageId:      aws.String(ami),
		InstanceType: aws.String(instanceType),
		IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Name: aws.String(iamProfileName),
		},
		UserData:     aws.String(userdata),
		EbsOptimized: aws.Bool(ebsOptimized),
		Monitoring:   &ec2.LaunchTemplatesMonitoringRequest{Enabled: aws.Bool(detailedMonitoringEnabled)},
		MetadataOptions: &ec2.LaunchTemplateInstanceMetadataOptionsRequest{
			HttpTokens:              aws.String("required"),
			HttpPutResponseHopLimit: aws.Int64(httpPutResponseHopLimit),
			HttpEndpoint:            aws.String("enabled"),
		},
	}

	// Only set SecurityGroupIds if it's not empty
	if len(securityGroups) > 0 {
		launchTemplateData.SecurityGroupIds = securityGroups
	}

	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: launchTemplateData,
		LaunchTemplateName: aws.String(name),
	}

	// Add resource tags if provided
	if len(tags) > 0 {
		var tagSpecs []*ec2.LaunchTemplateTagSpecificationRequest
		var ec2Tags []*ec2.Tag
		for _, tag := range tags {
			parts := strings.Split(tag, "=")
			if len(parts) == 2 {
				ec2Tags = append(ec2Tags, &ec2.Tag{
					Key:   aws.String(parts[0]),
					Value: aws.String(parts[1]),
				})
			}
		}
		tagSpecs = append(tagSpecs, &ec2.LaunchTemplateTagSpecificationRequest{
			ResourceType: aws.String("volume"),
			Tags:         ec2Tags,
		})
		input.LaunchTemplateData.SetTagSpecifications(tagSpecs)
	}

	if len(blockDevices) > 0 {
		input.LaunchTemplateData.SetBlockDeviceMappings(blockDevices)
	}

	if len(keyName) > 0 {
		input.LaunchTemplateData.SetKeyName(keyName)
	}

	// Configure network interfaces
	var networkInterfaces []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest

	// Configure primary ENI if specified
	if primaryENI != nil {
		primaryInterface := &ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
			DeviceIndex:         aws.Int64(int64(primaryENI.DeviceIndex)),
			SubnetId:            aws.String(primaryENI.SubnetID),
			DeleteOnTermination: aws.Bool(primaryENI.DeleteOnTermination),
		}

		if len(primaryENI.SecurityGroups) > 0 {
			primaryInterface.Groups = aws.StringSlice(primaryENI.SecurityGroups)
		}

		if primaryENI.PrivateIPAddress != "" {
			primaryInterface.PrivateIpAddress = aws.String(primaryENI.PrivateIPAddress)
		}

		networkInterfaces = append(networkInterfaces, primaryInterface)
	}

	// Configure secondary ENIs if specified
	for _, secondaryENI := range secondaryENIs {
		secondaryInterface := &ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
			DeviceIndex:         aws.Int64(int64(secondaryENI.DeviceIndex)),
			SubnetId:            aws.String(secondaryENI.SubnetID),
			DeleteOnTermination: aws.Bool(secondaryENI.DeleteOnTermination),
		}

		if len(secondaryENI.SecurityGroups) > 0 {
			secondaryInterface.Groups = aws.StringSlice(secondaryENI.SecurityGroups)
		}

		if secondaryENI.PrivateIPAddress != "" {
			secondaryInterface.PrivateIpAddress = aws.String(secondaryENI.PrivateIPAddress)
		}

		networkInterfaces = append(networkInterfaces, secondaryInterface)
	}

	if len(networkInterfaces) > 0 {
		// Validate that if network interfaces are specified, security groups should not be set in launch template
		if len(securityGroups) > 0 {
			return fmt.Errorf("cannot specify both security groups and network interfaces (ENIs) in launch template")
		}
		input.LaunchTemplateData.NetworkInterfaces = networkInterfaces
	} else if len(securityGroups) == 0 {
		// Validate that if no network interfaces are specified, security groups must be set
		return fmt.Errorf("either security groups or network interfaces (ENIs) must be specified in launch template")
	}

	if instanceMarketOptions != nil && !mixedInstancePolicyEnabled {
		input.LaunchTemplateData.InstanceMarketOptions = &ec2.LaunchTemplateInstanceMarketOptionsRequest{
			MarketType:  aws.String(instanceMarketOptions.MarketType),
			SpotOptions: &ec2.LaunchTemplateSpotMarketOptionsRequest{},
		}

		if instanceMarketOptions.SpotOptions.BlockDurationMinutes > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SetBlockDurationMinutes(instanceMarketOptions.SpotOptions.BlockDurationMinutes)
		}

		if len(instanceMarketOptions.SpotOptions.InstanceInterruptionBehavior) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SetInstanceInterruptionBehavior(instanceMarketOptions.SpotOptions.InstanceInterruptionBehavior)
		}

		if len(instanceMarketOptions.SpotOptions.SpotInstanceType) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SetSpotInstanceType(instanceMarketOptions.SpotOptions.SpotInstanceType)
		}

		if len(instanceMarketOptions.SpotOptions.MaxPrice) > 0 {
			input.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SetMaxPrice(instanceMarketOptions.SpotOptions.MaxPrice)
		}
	}

	_, err := e.Client.CreateLaunchTemplate(input)
	if err != nil {
		return err
	}

	Logger.Info("Successfully create new launch template : ", name)

	return nil
}

// GetSecurityGroupList Get All Security Group Information New Launch Configuration
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

		// if it matches 0 or more than 1, it is wrong
		if len(result.SecurityGroups) != 1 {
			var matched []string
			for _, s := range result.SecurityGroups {
				matched = append(matched, *s.GroupName)
			}
			return nil, fmt.Errorf("expected only one security group on name lookup for \"%s\" got \"%s\"", sg, strings.Join(matched, ","))
		}

		retList = append(retList, result.SecurityGroups[0].GroupId)
	}

	return retList, nil
}

// MakeBlockDevices returns list of block device mapping for launch configuration
func (e EC2Client) MakeBlockDevices(blocks []schemas.BlockDevice) []*autoscaling.BlockDeviceMapping {
	var ret []*autoscaling.BlockDeviceMapping

	for _, block := range blocks {
		enabledEBSEncrypted := block.Encrypted
		Logger.Infof("Encrypt ebs %t", enabledEBSEncrypted)
		var ebsDevice *autoscaling.Ebs

		if enabledEBSEncrypted {
			ebsDevice = &autoscaling.Ebs{
				VolumeSize:          aws.Int64(block.VolumeSize),
				VolumeType:          aws.String(block.VolumeType),
				Encrypted:           aws.Bool(enabledEBSEncrypted),
				DeleteOnTermination: aws.Bool(block.DeleteOnTermination),
			}
		} else {
			ebsDevice = &autoscaling.Ebs{
				VolumeSize:          aws.Int64(block.VolumeSize),
				VolumeType:          aws.String(block.VolumeType),
				DeleteOnTermination: aws.Bool(block.DeleteOnTermination),
			}
		}
		Logger.Infof("ebs %s", ebsDevice)

		if len(block.SnapshotID) > 0 {
			if !isValidSnapshotID(block.SnapshotID) {
				Logger.Error(fmt.Sprintf("Invalid snapshot ID format: %s", block.SnapshotID))
				continue
			}
			ebsDevice.SnapshotId = aws.String(block.SnapshotID)
		}

		tmp := &autoscaling.BlockDeviceMapping{
			DeviceName: aws.String(block.DeviceName),
			Ebs:        ebsDevice,
		}
		Logger.Infof("tmp %s", tmp)

		if block.VolumeType == "io1" || block.VolumeType == "io2" {
			tmp.Ebs.Iops = aws.Int64(block.Iops)
		}

		ret = append(ret, tmp)
	}

	return ret
}

// MakeLaunchTemplateBlockDeviceMappings returns list of block device mappings for launch template
func (e EC2Client) MakeLaunchTemplateBlockDeviceMappings(blocks []schemas.BlockDevice) []*ec2.LaunchTemplateBlockDeviceMappingRequest {
	var ret []*ec2.LaunchTemplateBlockDeviceMappingRequest

	for _, block := range blocks {
		enabledEBSEncrypted := block.Encrypted

		var LaunchTemplateEbsBlockDevice *ec2.LaunchTemplateEbsBlockDeviceRequest

		if enabledEBSEncrypted {
			var keyId string
			var err error

			// Priority: KmsKeyId > KmsAlias
			if len(block.KmsKeyId) > 0 {
				keyId = block.KmsKeyId
				Logger.Infof("Using provided KMS Key ID: %s", keyId)
			} else {
				keyId, err = e.getKmsKeyIdByAlias(block.KmsAlias)
				if err != nil {
					Logger.Fatal(fmt.Sprintf("Error: %s", err.Error()))
				}
			}

			LaunchTemplateEbsBlockDevice = &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeSize:          aws.Int64(block.VolumeSize),
				VolumeType:          aws.String(block.VolumeType),
				Encrypted:           aws.Bool(enabledEBSEncrypted),
				KmsKeyId:            aws.String(keyId),
				DeleteOnTermination: aws.Bool(block.DeleteOnTermination),
			}
		} else {
			LaunchTemplateEbsBlockDevice = &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeSize:          aws.Int64(block.VolumeSize),
				VolumeType:          aws.String(block.VolumeType),
				DeleteOnTermination: aws.Bool(block.DeleteOnTermination),
			}
		}

		if len(block.SnapshotID) > 0 {
			if !isValidSnapshotID(block.SnapshotID) {
				Logger.Error(fmt.Sprintf("Invalid snapshot ID format: %s", block.SnapshotID))
				continue
			}
			LaunchTemplateEbsBlockDevice.SnapshotId = aws.String(block.SnapshotID)
		}

		tmp := ec2.LaunchTemplateBlockDeviceMappingRequest{
			DeviceName: aws.String(block.DeviceName),
			Ebs:        LaunchTemplateEbsBlockDevice,
		}

		if block.VolumeType == "io1" || block.VolumeType == "io2" {
			tmp.Ebs.Iops = aws.Int64(block.Iops)
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
	tags []*autoscaling.Tag,
	subnets []string,
	mixedInstancePolicy schemas.MixedInstancesPolicy,
	hooks []*autoscaling.LifecycleHookSpecification) error {
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

		if mixedInstancePolicy.OnDemandPercentage >= 0 {
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
		return err
	}

	Logger.Info("Successfully create new autoscaling group : ", name)
	return nil
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

// UpdateAutoScalingGroupSize Update Autoscaling Group size
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

// CreateScalingPolicy creates scaling policy
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

// GenerateLifecycleHooks generate lifecycle hooks
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

// GetTargetGroups returns list of target group ARN of autoscaling group
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

// UpdateAutoScalingGroup  updates auto scaling group information
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

	var scheduledUpdateGroupActions []*autoscaling.ScheduledUpdateGroupActionRequest
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

// AttachAsgToTargetGroups attaches autoscaling group to target groups of ELB
func (e EC2Client) AttachAsgToTargetGroups(asg string, targetGroups []*string) error {
	input := &autoscaling.AttachLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: aws.String(asg),
		TargetGroupARNs:      targetGroups,
	}

	_, err := e.AsClient.AttachLoadBalancerTargetGroups(input)
	if err != nil {
		return err
	}

	return nil
}

// DetachAsgFromTargetGroups detaches autoscaling group from target groups of ELB
func (e EC2Client) DetachAsgFromTargetGroups(asg string, targetGroups []*string) error {
	input := &autoscaling.DetachLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: aws.String(asg),
		TargetGroupARNs:      targetGroups,
	}

	_, err := e.AsClient.DetachLoadBalancerTargetGroups(input)
	if err != nil {
		return err
	}

	return nil
}

// CreateSecurityGroup creates new security group
func (e EC2Client) CreateSecurityGroup(sgName string, vpcID *string) (*string, error) {
	input := &ec2.CreateSecurityGroupInput{
		Description: aws.String("canary deployment"),
		GroupName:   aws.String(sgName),
		VpcId:       vpcID,
	}

	ret, err := e.Client.CreateSecurityGroup(input)
	if err != nil {
		return nil, err
	}

	return ret.GroupId, nil
}

// GetSecurityGroup retrieves group id of existing security group
func (e EC2Client) GetSecurityGroup(sgName string) (*string, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("group-name"),
				Values: []*string{
					aws.String(sgName),
				},
			},
		},
	}

	result, err := e.Client.DescribeSecurityGroups(input)
	if err != nil {
		return nil, err
	}

	if len(result.SecurityGroups) == 0 {
		return nil, fmt.Errorf("checked duplicated but cannot find the security group: %s", sgName)
	}

	return result.SecurityGroups[0].GroupId, nil
}

// UpdateInboundRules updates inbound rules for security group  with IP
func (e EC2Client) UpdateInboundRules(sgID, protocol, cidr, description string, fromPort, toPort int64) error {
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(fromPort),
				ToPort:     aws.Int64(toPort),
				IpProtocol: aws.String(protocol),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String(cidr),
						Description: aws.String(description),
					},
				},
			},
		},
	}

	_, err := e.Client.AuthorizeSecurityGroupIngress(input)
	if err != nil {
		return err
	}

	return nil
}

// UpdateInboundRulesWithGroup updates inbound rules for security group  with other security group
func (e EC2Client) UpdateInboundRulesWithGroup(sgID, protocol, description string, fromSg *string, fromPort, toPort int64) error {
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(fromPort),
				ToPort:     aws.Int64(toPort),
				IpProtocol: aws.String(protocol),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: aws.String(description),
						GroupId:     fromSg,
					},
				},
			},
		},
	}

	_, err := e.Client.AuthorizeSecurityGroupIngress(input)
	if err != nil {
		return err
	}

	return nil
}

// UpdateOutboundRules updates inbound rules  for security group  with IP
func (e EC2Client) UpdateOutboundRules(sgID, protocol, cidr, description string, fromPort, toPort int64) error {
	input := &ec2.AuthorizeSecurityGroupEgressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(fromPort),
				ToPort:     aws.Int64(toPort),
				IpProtocol: aws.String(protocol),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String(cidr),
						Description: aws.String(description),
					},
				},
			},
		},
	}

	if protocol == "-1" {
		input.IpPermissions[0].FromPort = nil
		input.IpPermissions[0].ToPort = nil
	}

	_, err := e.Client.AuthorizeSecurityGroupEgress(input)
	if err != nil {
		return err
	}

	return nil
}

// DeleteSecurityGroup deletes security group
func (e EC2Client) DeleteSecurityGroup(sg string) error {
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(sg),
	}

	_, err := e.Client.DeleteSecurityGroup(input)
	if err != nil {
		return err
	}
	return nil
}

// RevokeInboundRulesWithGroup revokes inbound rules for security group  with other security group
func (e EC2Client) RevokeInboundRulesWithGroup(sgID, protocol string, fromSg *string, fromPort, toPort int64) error {
	input := &ec2.RevokeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(fromPort),
				ToPort:     aws.Int64(toPort),
				IpProtocol: aws.String(protocol),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						GroupId: fromSg,
					},
				},
			},
		},
	}

	_, err := e.Client.RevokeSecurityGroupIngress(input)
	if err != nil {
		return err
	}

	return nil
}

// DeleteCanaryTag deletes canary tag from auto scaling group
func (e EC2Client) DeleteCanaryTag(asg string) error {
	input := &autoscaling.DeleteTagsInput{
		Tags: []*autoscaling.Tag{
			{
				ResourceId:   aws.String(asg),
				ResourceType: aws.String("auto-scaling-group"),
				Key:          aws.String(constants.DeploymentTagKey),
				Value:        aws.String(constants.CanaryDeployment),
			},
		},
	}

	_, err := e.AsClient.DeleteTags(input)
	if err != nil {
		return err
	}

	return nil
}

// DescribeInstances return detailed information of instances
func (e EC2Client) DescribeInstances(instanceIds []*string) ([]*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}

	result, err := e.Client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) == 0 {
		return nil, nil
	}

	return result.Reservations[0].Instances, nil
}

// ModifyNetworkInterfaces modifies network interface attributes
func (e EC2Client) ModifyNetworkInterfaces(eni *string, groups []*string) error {
	input := &ec2.ModifyNetworkInterfaceAttributeInput{
		Groups:             groups,
		NetworkInterfaceId: eni,
	}

	_, err := e.Client.ModifyNetworkInterfaceAttribute(input)
	if err != nil {
		return err
	}

	return nil
}

// CreateNewLaunchTemplateVersion creates new version of launch template
func (e EC2Client) CreateNewLaunchTemplateVersion(lt *ec2.LaunchTemplateVersion, sgs []*string) (*ec2.LaunchTemplateVersion, error) {
	input := &ec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			SecurityGroupIds: sgs,
		},
		LaunchTemplateId:   lt.LaunchTemplateId,
		SourceVersion:      aws.String("1"),
		VersionDescription: aws.String("Canary Completion"),
	}

	result, err := e.Client.CreateLaunchTemplateVersion(input)
	if err != nil {
		return nil, err
	}

	return result.LaunchTemplateVersion, nil
}

// UpdateAutoScalingLaunchTemplate updates autoscaling launch template
func (e EC2Client) UpdateAutoScalingLaunchTemplate(asg string, lt *ec2.LaunchTemplateVersion) error {
	input := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asg),
		LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
			LaunchTemplateId: lt.LaunchTemplateId,
			Version:          aws.String(strconv.FormatInt(*lt.VersionNumber, 10)),
		},
	}

	_, err := e.AsClient.UpdateAutoScalingGroup(input)
	if err != nil {
		return err
	}

	return nil
}

// DetachLoadBalancerTargetGroup detaches target group from autoscaling group
func (e EC2Client) DetachLoadBalancerTargetGroup(asg string, tgARNs []*string) error {
	input := &autoscaling.DetachLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: aws.String(asg),
		TargetGroupARNs:      tgARNs,
	}

	_, err := e.AsClient.DetachLoadBalancerTargetGroups(input)
	if err != nil {
		return err
	}

	return nil
}

// StartInstanceRefresh starts instance refresh
func (e EC2Client) StartInstanceRefresh(name *string, instanceWarmup, minHealthyPercentage int64) (*string, error) {
	input := &autoscaling.StartInstanceRefreshInput{
		AutoScalingGroupName: name,
		Preferences: &autoscaling.RefreshPreferences{
			InstanceWarmup:       aws.Int64(instanceWarmup),
			MinHealthyPercentage: aws.Int64(minHealthyPercentage),
		},
	}

	result, err := e.AsClient.StartInstanceRefresh(input)
	if err != nil {
		return nil, err
	}

	return result.InstanceRefreshId, nil
}

// DescribeInstanceRefreshes describes instance refresh information
func (e EC2Client) DescribeInstanceRefreshes(name, id *string) (*autoscaling.InstanceRefresh, error) {
	input := &autoscaling.DescribeInstanceRefreshesInput{
		AutoScalingGroupName: name,
	}

	if id != nil {
		input.InstanceRefreshIds = []*string{id}
	}

	result, err := e.AsClient.DescribeInstanceRefreshes(input)
	if err != nil {
		return nil, err
	}

	var targets []*autoscaling.InstanceRefresh
	for _, ir := range result.InstanceRefreshes {
		if ir.EndTime == nil || (id != nil && *ir.InstanceRefreshId == *id) {
			targets = append(targets, ir)
		}
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no instance refresh exists: %s", *name)
	}

	return targets[0], nil
}

// aws ec2 describe-instance-types --filters Name=processor-info.supported-architecture,Values=arm64 --query "InstanceTypes[*].InstanceType"
func (e EC2Client) DescribeInstanceTypes() ([]string, error) {
	var instanceTypeList []string
	params := &ec2.DescribeInstanceTypesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("processor-info.supported-architecture"),
				Values: []*string{aws.String("arm64")},
			},
		},
	}
	result, err := e.Client.DescribeInstanceTypes(params)
	if err == nil {
		for i := 0; i < len(result.InstanceTypes); i++ {
			instanceType := strings.Split(*result.InstanceTypes[i].InstanceType, ".")[0]
			if !tool.IsStringInArray(instanceType, instanceTypeList) {
				instanceTypeList = append(instanceTypeList, instanceType)
			}
		}
	} else {
		return nil, errors.New("you cannot get instanceType from aws, please check your configuration")
	}
	return instanceTypeList, nil
}

// $ aws ec2 describe-images --filters Name=image-id,Values=ami-01288945bd24ed49a --query "Images[*].Architecture"
func (e EC2Client) DescribeAMIArchitecture(amiID string) (string, error) {
	var amiArchitecture string
	params := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("image-id"),
				Values: []*string{aws.String(amiID)},
			},
		},
	}
	result, err := e.Client.DescribeImages(params)
	if err == nil {
		amiArchitecture = *result.Images[0].Architecture
	} else {
		return "", errors.New("you cannot get ami-architecture from aws, please check your configuration")
	}
	return amiArchitecture, nil
}

func (e EC2Client) getKmsKeyIdByAlias(alias string) (string, error) {
	if len(alias) == 0 {
		Logger.Info("Volume Encrypt default KMS Key(aws/ebs)")
		alias = "alias/aws/ebs"
	} else if !strings.HasPrefix(alias, "alias") {
		var sb strings.Builder
		sb.WriteString("alias/")
		sb.WriteString(alias)
		alias = sb.String()
	}

	result, err := e.KMSClient.ListAliases(&kms.ListAliasesInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list aliases, %v", err)
	}

	for _, aliasEntry := range result.Aliases {
		if aliasEntry.AliasName != nil && *aliasEntry.AliasName == alias {
			if aliasEntry.TargetKeyId != nil {
				return *aliasEntry.TargetKeyId, nil
			}
		}
	}
	return "", fmt.Errorf("alias %s not found", alias)
}

func isValidSnapshotID(snapshotID string) bool {
	// AWS 스냅샷 ID 형식: snap-xxxxxxxx 또는 snap-xxxxxxxxxxxxxxxxx
	// x는 16진수(0-9, a-f)
	pattern := `^snap-[0-9a-f]{8}([0-9a-f]{9})?$`
	matched, err := regexp.MatchString(pattern, snapshotID)
	return err == nil && matched
}
