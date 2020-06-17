package deployer

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"strings"
)


type BlueGreen struct {
	Deployer
}

func NewBlueGrean(mode string, logger *Logger.Logger, awsConfig builder.AWSConfig, stack builder.Stack) BlueGreen {
	awsClients := []aws.AWSClient{}
	for _, region := range stack.Regions {
		awsClients = append(awsClients, aws.BootstrapServices(region.Region, stack.AssumeRole))
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
func (b BlueGreen) Deploy(config builder.Config) {
	b.Logger.Info("Deploy Mode is " + b.Mode)

	//Get LocalFileProvider
	b.LocalProvider = builder.SetUserdataProvider(b.Stack.Userdata, b.AwsConfig.Userdata)

	// Make Frigga
	frigga := tool.Frigga{}
	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		//Setup frigga with prefix
		frigga.Prefix = tool.BuildPrefixName(b.AwsConfig.Name, b.Stack.Env, region.Region)

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			tool.ErrorLogging(err.Error())
		}

		// Get All Autoscaling Groups
		asgGroups := client.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(frigga.Prefix)

		//Get All Previous Autoscaling Groups and versions
		prev_asgs := []string{}
		prev_versions := []int{}
		for _, asgGroup := range asgGroups {
			prev_asgs = append(prev_asgs, *asgGroup.AutoScalingGroupName)
			prev_versions = append(prev_versions, tool.ParseVersion(*asgGroup.AutoScalingGroupName))
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
		new_asg_name := tool.GenerateAsgName(frigga.Prefix, cur_version)
		launch_template_name := tool.GenerateLcName(new_asg_name)

		userdata := (b.LocalProvider).Provide()

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
			tool.ErrorLogging("Unknown error happened creating new launch template.")
		}

		health_elb := region.HealthcheckLB
		loadbalancers := region.LoadBalancers
		if ! tool.IsStringInArray(health_elb, loadbalancers) {
			loadbalancers = append(loadbalancers, health_elb)
		}

		healthcheckTargetGroups := region.HealthcheckTargetGroup
		target_groups := region.TargetGroups
		if ! tool.IsStringInArray(healthcheckTargetGroups, target_groups) {
			target_groups = append(target_groups, healthcheckTargetGroups)
		}

		use_public_subnets := region.UsePublicSubnets
		healthcheck_type := aws.DEFAULT_HEALTHCHECK_TYPE
		healthcheck_grace_period := int64(aws.DEFAULT_HEALTHCHECK_GRACE_PERIOD)
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
			aws.MakeStringArrayToAwsStrings(loadbalancers),
			target_group_arns,
			termination_policies,
			aws.MakeStringArrayToAwsStrings(availability_zones),
			tags,
			subnets,
		)

		if ! ret {
			tool.ErrorLogging("Unknown error happened creating new autoscaling group.")
		}

		b.AsgNames[region.Region] = new_asg_name
		b.PrevAsgs[region.Region] = prev_asgs
	}
}

// Healthchecking
func (b BlueGreen) HealthChecking(config builder.Config) map[string]bool {
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
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		b.Logger.Info("Healthchecking for region starts : " + region.Region )

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			tool.ErrorLogging(err.Error())
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
func (b BlueGreen) FinishAdditionalWork(config builder.Config) error {
	if len(b.Stack.Autoscaling) == 0 {
		b.Logger.Info("No scaling policy exists")
		return nil
	}

	if len(config.Region) > 0 && !checkRegionExist(config.Region, b.Stack.Regions) {
		return nil
	}

	//Apply Autosacling Policies
	b.Logger.Info("Attaching autoscaling policies")
	for _, region := range b.Stack.Regions {
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			tool.ErrorLogging(err.Error())
		}

		//putting autoscaling group policies
		policies := []string{}
		policyArns := map[string]string{}
		for _, policy := range b.Stack.Autoscaling {
			policyArn, err := client.EC2Service.CreateScalingPolicy(policy, b.AsgNames[region.Region])
			if err != nil {
				tool.ErrorLogging(err.Error())
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
func (b BlueGreen) CleanPreviousVersion(config builder.Config) error {
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
			tool.ErrorLogging(err.Error())
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
func (b BlueGreen) TerminateChecking(config builder.Config) map[string]bool {
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
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		b.Logger.Info("Checking Termination stack for region starts : " + region.Region )

		//select client
		client, err := selectClientFromList(b.AWSClients, region.Region)
		if err != nil {
			tool.ErrorLogging(err.Error())
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
func checkRegionExist(target string, regions []builder.RegionConfig) bool {
	regionExists := false
	for _, region := range regions {
		if region.Region == target {
			regionExists = true
			break
		}
	}

	return regionExists
}
