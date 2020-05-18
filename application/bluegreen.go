package application

import (
	Logger "github.com/sirupsen/logrus"
)


type BlueGreen struct {
	Deployer
}

func _NewBlueGrean(mode string, logger *Logger.Logger, awsConfig AWSConfig, stack Stack) BlueGreen {
	awsClients := []AWSClient{}
	for _, region := range stack.Regions {
		awsClients = append(awsClients, _bootstrap_services(region.Region, stack.AssumeRole))
	}
	return BlueGreen{
		Deployer{
			Mode:  mode,
			Logger: logger,
			AwsConfig: awsConfig,
			AWSClients: awsClients,
			AsgNames: map[string]string{},
			Stack: stack,
		},
	}
}

// Deploy function
func (b BlueGreen) Deploy(config Config) {
	b.Logger.Info("Deploy Mode is " + b.Mode)

	//Get LocalFileProvider
	b.LocalProvider = _set_userdata_provider(b.Stack.Userdata, b.AwsConfig.Userdata)

	// Make Frigga
	frigga := Frigga{}
	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Info("This region is skipped by user : " + region.Region)
			continue
		}

		//Setup frigga with prefix
		frigga.Prefix = _build_prefix_name(b.AwsConfig.Name, b.Stack.Env, region.Region)

		//select client
		client, err := _select_client_from_list(b.AWSClients, region.Region)
		if err != nil {
			error_logging(err.Error())
		}

		// Get All Autoscaling Groups
		asgGroups := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(frigga.Prefix)

		//Get All Previous Autoscaling Groups and versions
		prev_asgs := []string{}
		prev_versions := []int{}
		for _, asgGroup := range asgGroups {
			prev_asgs = append(prev_asgs, *asgGroup.AutoScalingGroupName)
			prev_versions = append(prev_versions, _parse_version(*asgGroup.AutoScalingGroupName))
		}
		b.Logger.Info("Previous Versions : ", prev_asgs)

		// Get Current Version
		cur_version := _get_current_version(prev_versions)
		b.Logger.Info("Current Version :", cur_version)

		//Get AMI
		ami := config.Ami

		// Generate new name for autoscaling group and launch configuration
		new_asg_name := _generate_asg_name(frigga.Prefix, cur_version)
		launch_configuration_name := _generate_lc_name(new_asg_name)

		userdata := (b.LocalProvider).provide()

		//Stack check
		securityGroups := client.EC2Service.GetSecurityGroupList(region.VPC, region.SecurityGroups)
		blockDevices := client.EC2Service.MakeBlockDevices(b.Stack.BlockDevices)
		ebsOptimized := b.Stack.EbsOptimized

		ret := client.EC2Service.CreateNewLaunchConfiguration(
			launch_configuration_name,
			ami,
			b.Stack.InstanceType,
			b.Stack.SshKey,
			b.Stack.IamInstanceProfile,
			userdata,
			ebsOptimized,
			securityGroups,
			blockDevices,
		)

		if ! ret {
			error_logging("Unknown error happened creating new launch configuration.")
		}

		health_elb := region.HealthcheckLB
		loadbalancers := region.LoadBalancers
		if ! IsStringInArray(health_elb, loadbalancers) {
			loadbalancers = append(loadbalancers, health_elb)
		}

		healthcheck_target_groups := region.HealthcheckTargetGroup
		target_groups := region.TargetGroups
		if ! IsStringInArray(healthcheck_target_groups, target_groups) {
			target_groups = append(target_groups, healthcheck_target_groups)
		}

		use_public_subnets := region.UsePublicSubnets
		healthcheck_type := DEFAULT_HEALTHCHECK_TYPE
		healthcheck_grace_period := int64(DEFAULT_HEALTHCHECK_GRACE_PERIOD)
		termination_policies := []*string{}
		availability_zones := client.EC2Service.GetAvailabilityZones(region.VPC, region.AvailabilityZones)
		target_group_arns := client.ELBService.GetTargetGroupARNs(target_groups)
		tags  := client.EC2Service.GenerateTags(b.AwsConfig.Tags, new_asg_name, b.AwsConfig.Name, config.Stack)
		subnets := client.EC2Service.GetSubnets(region.VPC, use_public_subnets)

		client.EC2Service.CreateAutoScalingGroup(
			new_asg_name,
			launch_configuration_name,
			healthcheck_type,
			healthcheck_grace_period,
			b.Stack.Capacity,
			_make_string_array_to_aws_strings(loadbalancers),
			target_group_arns,
			termination_policies,
			_make_string_array_to_aws_strings(availability_zones),
			tags,
			subnets,
		)

		b.AsgNames[region.Region] = new_asg_name
	}
}

// Healthchecking
func (b BlueGreen) Healthchecking(config Config) bool {
	finished := []string{}

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(b.Stack.Regions)
	}

	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Info("This region is skipped by user : " + region.Region)
			continue
		}

		if IsStringInArray(region.Region, finished) {
			continue
		}

		Logger.Info("Healthchecking for region starts... : " + region.Region )

		//select client
		client, err := _select_client_from_list(b.AWSClients, region.Region)
		if err != nil {
			error_logging(err.Error())
		}

		asg := client.EC2Service.GetAllMatchingAutoscalingGroups(b.AsgNames[region.Region])

		isHealthy := b.Deployer.polling(region, asg, client)

		if isHealthy {
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return true
	}

	return false
}

//Stack Name Getter
func (b BlueGreen) GetStackName() string {
	return b.Stack.Stack
}