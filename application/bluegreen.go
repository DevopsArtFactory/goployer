package application

import (
	"fmt"
	Logger "github.com/sirupsen/logrus"
	"strings"
)


type BlueGreen struct {
	Deployer
}

func NewBlueGrean(mode string, logger *Logger.Logger, awsConfig AWSConfig, stack Stack) BlueGreen {
	awsClients := []AWSClient{}
	for _, region := range stack.Regions {
		awsClients = append(awsClients, bootstrapServices(region.Region, stack.AssumeRole))
	}
	return BlueGreen{
		Deployer{
			Mode:  mode,
			Logger: logger,
			AwsConfig: awsConfig,
			AWSClients: awsClients,
			AsgNames: map[string]string{},
			PrevAsgs: map[string][]string{},
			Stack: stack,
		},
	}
}

// Deploy function
func (b BlueGreen) Deploy(config Config) {
	b.Logger.Info("Deploy Mode is " + b.Mode)

	//Get LocalFileProvider
	b.LocalProvider = setUserdataProvider(b.Stack.Userdata, b.AwsConfig.Userdata)

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
		frigga.Prefix = buildPrefixName(b.AwsConfig.Name, b.Stack.Env, region.Region)

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
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
			prev_versions = append(prev_versions, parseVersion(*asgGroup.AutoScalingGroupName))
		}
		b.Logger.Info("Previous Versions : ", strings.Join(prev_asgs, " | "))

		// Get Current Version
		cur_version := getCurrentVersion(prev_versions)
		b.Logger.Info("Current Version :", cur_version)

		//Get AMI
		var ami string
		if len(config.Ami) > 0 {
			ami = config.Ami
		} else {
			ami = region.AmiId
		}

		// Generate new name for autoscaling group and launch configuration
		new_asg_name := generateAsgName(frigga.Prefix, cur_version)
		launch_template_name := generateLcName(new_asg_name)

		userdata := (b.LocalProvider).provide()

		//Stack check
		securityGroups := client.EC2Service.GetSecurityGroupList(region.VPC, region.SecurityGroups)
		blockDevices := client.EC2Service.MakeLaunchTemplateBlockDeviceMappings(b.Stack.BlockDevices)
		ebsOptimized := b.Stack.EbsOptimized

		// Launch Configuration
		//ret := client.EC2Service.CreateNewLaunchConfiguration(
		//	launch_configuration_name,
		//	ami,
		//	b.Stack.InstanceType,
		//	b.Stack.SshKey,
		//	b.Stack.IamInstanceProfile,
		//	userdata,
		//	ebsOptimized,
		//	securityGroups,
		//	blockDevices,
		//)

		// LaunchTemplate
		ret := client.EC2Service.CreateNewLaunchTemplate(
			launch_template_name,
			ami,
			region.InstanceType,
			region.SshKey,
			b.Stack.IamInstanceProfile,
			userdata,
			ebsOptimized,
			securityGroups,
			blockDevices,
			b.Stack.InstanceMarketOptions,
		)

		if ! ret {
			error_logging("Unknown error happened creating new launch template.")
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
		subnets := client.EC2Service.GetSubnets(region.VPC, use_public_subnets, availability_zones)

		ret = client.EC2Service.CreateAutoScalingGroup(
			new_asg_name,
			launch_template_name,
			healthcheck_type,
			healthcheck_grace_period,
			b.Stack.Capacity,
			makeStringArrayToAwsStrings(loadbalancers),
			target_group_arns,
			termination_policies,
			makeStringArrayToAwsStrings(availability_zones),
			tags,
			subnets,
		)

		if ! ret {
			error_logging("Unknown error happened creating new autoscaling group.")
		}

		b.AsgNames[region.Region] = new_asg_name
		b.PrevAsgs[region.Region] = prev_asgs
	}
}

// Healthchecking
func (b BlueGreen) HealthChecking(config Config) map[string]bool {
	stack_name := b.GetStackName()
	Logger.Info(fmt.Sprintf("Healthchecking for stack %s starts : ", stack_name ))
	finished := []string{}

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(b.Stack.Regions)
	}

	if len(config.Region) > 0 {
		if ! checkRegionExist(config.Region, b.Stack.Regions) {
			validCount = 0
		}
	}

	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Info("This region is skipped by user : " + region.Region)
			continue
		}

		b.Logger.Info("Healthchecking for region starts : " + region.Region )

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			error_logging(err.Error())
		}

		asg := client.EC2Service.GetMatchingAutoscalingGroup(b.AsgNames[region.Region])

		isHealthy := b.Deployer.polling(region, asg, client)

		if isHealthy {
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return map[string]bool{stack_name: true}
	}

	return map[string]bool{stack_name: false}
}

//Stack Name Getter
func (b BlueGreen) GetStackName() string {
	return b.Stack.Stack
}

//BlueGreen finish final work
func (b BlueGreen) FinishAdditionalWork(config Config) error {
	if len(b.Stack.Autoscaling) == 0 {
		b.Logger.Info("No scaling policy exists")
		return nil
	}

	if len(config.Region) > 0 && !checkRegionExist(config.Region, b.Stack.Regions) {
		return nil
	}

	for _, region := range b.Stack.Regions {
		//Apply Autosacling Policies
		b.Logger.Info("Attaching autoscaling policies")

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			error_logging(err.Error())
		}

		//putting autoscaling group policies
		policies := []string{}
		policyArns := map[string]string{}
		for _, policy := range b.Stack.Autoscaling {
			policyArn, err := client.EC2Service.CreateScalingPolicy(policy, b.AsgNames[region.Region])
			if err != nil {
				error_logging(err.Error())
				return err
			}
			policyArns[policy.Name] = *policyArn
			policies = append(policies, policy.Name)
		}

		if err := client.EC2Service.EnableMetrics(b.AsgNames[region.Region]); err != nil {
			return err
		}

		if err := client.CloudWatchService.CreateScalingAlarms(b.AsgNames[region.Region], b.Stack.Alarms, policyArns); err != nil {
			return nil
		}


		//Apply lifecycle callback options
		b.Logger.Info("Attaching lifecycle callbacks.")
	}

	Logger.Info("Finish addtional works.")
	return nil
}

//Clean Previous Version
func (b BlueGreen) CleanPreviousVersion(config Config) error {
	b.Logger.Info("Delete Mode is " + b.Mode)

	if len(config.Region) > 0 {
		if ! checkRegionExist(config.Region, b.Stack.Regions) {
			return nil
		}
	}

	for _, region := range b.Stack.Regions {
		b.Logger.Info(fmt.Sprintf("The number of previous versions to delete is %d.\n", len(b.PrevAsgs[region.Region])))

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			error_logging(err.Error())
		}

		if len(b.PrevAsgs[region.Region]) > 0 {
			for _, asg := range b.PrevAsgs[region.Region] {
				// First make autoscaling group size to 0
				err := b.ResizingAutoScalingGroupToZero(client, b.Stack.Stack, asg)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Clean Teramination Checking
func (b BlueGreen) TerminateChecking(config Config) map[string]bool {
	stack_name := b.GetStackName()
	Logger.Info(fmt.Sprintf("Termination Checking for %s starts...", stack_name ))

	//Valid Count
	validCount := 1
	if config.Region == "" {
		validCount = len(b.Stack.Regions)
	}

	if len(config.Region) > 0 {
		if ! checkRegionExist(config.Region, b.Stack.Regions) {
			validCount = 0
		}
	}

	finished := []string{}
	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Info("This region is skipped by user : " + region.Region)
			continue
		}

		b.Logger.Info("Checking Termination stack for region starts : " + region.Region )

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			error_logging(err.Error())
		}

		targets := b.PrevAsgs[region.Region]
		if len(targets) == 0 {
			Logger.Info("No target to delete : ", region.Region)
			finished = append(finished, region.Region)
			continue
		}

		ok_count := 0
		for _, target := range targets {
			ok := b.Deployer.CheckTerminating(client, target)
			if ok {
				Logger.Info("finished : ", target)
				ok_count++
			}
		}

		if ok_count == len(targets) {
			finished = append(finished, region.Region)
		}
	}

	if len(finished) == validCount {
		return map[string]bool{stack_name: true}
	}

	return map[string]bool{stack_name: false}
}


//checkRegionExist checks if target region is really in regions described in manifest file
func checkRegionExist(target string, regions []RegionConfig) bool {
	regionExists := false
	for _, region := range regions {
		if region.Region == target {
			regionExists = true
			break
		}
	}

	return regionExists
}
