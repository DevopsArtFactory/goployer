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

func (d Deployer) Deploy(builder Builder)  {
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
}