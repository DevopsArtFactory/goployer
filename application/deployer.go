package application

import (
	"fmt"
	Logger "github.com/sirupsen/logrus"
)

type Deployer struct {
	Mode string
	Prefix string
	Logger 	*Logger.Logger
	AWSClient AWSClient
}

func _get_current_version(prev_versions []int) int {
	if len(prev_versions) == 0 {
		return 0
	}
	return (prev_versions[len(prev_versions)-1] + 1) % 1000
}

func (d Deployer) Deploy(builder Builder) {
	d.Logger.Info("Deploy Mode is " + d.Mode)

	// Get All Autoscaling Groups
	asgGroups := d.AWSClient.EC2Service.GetAllMatchingAutoscalingGroups(d.Prefix)

	//Get All Previous Autoscaling Groups and versions
	prev_asgs := []string{}
	prev_versions := []int{}
	for _, asgGroup := range asgGroups {
		prev_asgs = append(prev_asgs, *asgGroup.AutoScalingGroupName)
		prev_versions = append(prev_versions, _parse_version(*asgGroup.AutoScalingGroupName))
	}
	d.Logger.Info("Previous Versions : ", prev_asgs)

	// Get Current Version
	cur_version := _get_current_version(prev_versions)
	d.Logger.Info("Current Version :", cur_version)

	//Get AMI
	ami := builder.Config.Ami

	// Generate new name for autoscaling group and launch configuration
	new_asg_name := _generate_asg_name(d.Prefix, cur_version)
	launch_configuration_name := _generate_lc_name(new_asg_name)

	userdata := (builder.LocalProvider).provide()

	//Environment Check
	hasStack, targetEnv, regionEnv := builder.GetTargetEnvironment()
	if ! hasStack {
		error_logging(fmt.Sprintf("Cannot find the stack information : %s", builder.Config.Stack))
	}

	securityGroups := d.AWSClient.EC2Service.GetSecurityGroupList(regionEnv.VPC, regionEnv.SecurityGroups)
	blockDevices := d.AWSClient.EC2Service.MakeBlockDevices(targetEnv.BlockDevices)
	ebsOptimized := targetEnv.EbsOptimized

	ret := d.AWSClient.EC2Service.CreateNewLaunchConfiguration(
		launch_configuration_name,
		ami,
		targetEnv.InstanceType,
		targetEnv.SshKey,
		targetEnv.IamInstanceProfile,
		userdata,
		ebsOptimized,
		securityGroups,
		blockDevices,
	)

	if ! ret {
		error_logging("Unknown error happened creating new launch configuration.")
	}

	health_elb := regionEnv.HealthcheckLB
	loadbalancers := regionEnv.LoadBalancers
	if ! IsStringInArray(health_elb, loadbalancers) {
		loadbalancers = append(loadbalancers, health_elb)
	}

	healthcheck_target_groups := regionEnv.HealthcheckTargetGroup
	target_groups := regionEnv.TargetGroups
	if ! IsStringInArray(healthcheck_target_groups, target_groups) {
		target_groups = append(target_groups, healthcheck_target_groups)
	}

	use_public_subnets := regionEnv.UsePublicSubnets
	healthcheck_type := DEFAULT_HEALTHCHECK_TYPE
	healthcheck_grace_period := int64(DEFAULT_HEALTHCHECK_GRACE_PERIOD)
	termination_policies := []*string{}
	availability_zones := d.AWSClient.EC2Service.GetAvailabilityZones(regionEnv.VPC, regionEnv.AvailabilityZones)
	target_group_arns := d.AWSClient.ELBService.GetTargetGroupARNs(target_groups)
	tags  := d.AWSClient.EC2Service.GenerateTags(builder.AwsConfig.Tags, new_asg_name, builder.AwsConfig.Name, builder.Config.Stack)
	subnets := d.AWSClient.EC2Service.GetSubnets(regionEnv.VPC, use_public_subnets)

	d.AWSClient.EC2Service.CreateAutoScalingGroup(
		new_asg_name,
		launch_configuration_name,
		healthcheck_type,
		healthcheck_grace_period,
		targetEnv.Capacity,
		_make_string_array_to_aws_strings(loadbalancers),
		target_group_arns,
		termination_policies,
		_make_string_array_to_aws_strings(availability_zones),
		tags,
		subnets,
	)
}