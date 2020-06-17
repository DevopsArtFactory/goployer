package deployer

import (
	"errors"
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	Logger "github.com/sirupsen/logrus"
)

// Deployer per stack
type Deployer struct {
	Mode          	string
	AsgNames      	map[string]string
	PrevAsgs      	map[string][]string
	Logger        	*Logger.Logger
	Stack         	builder.Stack
	AwsConfig     	builder.AWSConfig
	AWSClients    	[]aws.AWSClient
	LocalProvider 	builder.UserdataProvider
	Slack			tool.Slack
}

// getCurrentVersion returns current version for current deployment step
func getCurrentVersion(prev_versions []int) int {
	if len(prev_versions) == 0 {
		return 0
	}
	return (prev_versions[len(prev_versions)-1] + 1) % 100
}

// Polling for healthcheck
func (d Deployer) polling(region builder.RegionConfig, asg *autoscaling.Group, client aws.AWSClient) bool {
	if *asg.AutoScalingGroupName == "" {
		tool.ErrorLogging(fmt.Sprintf("No autoscaling found for %s", d.AsgNames[region.Region]))
	}

	healthcheckTargetGroup 	 := region.HealthcheckTargetGroup
	healthcheckTargetGroupArn := (client.ELBService.GetTargetGroupARNs([]string{healthcheckTargetGroup}))[0]

	threshold := d.Stack.Capacity.Desired
	targetHosts := client.ELBService.GetHostInTarget(asg, healthcheckTargetGroupArn)

	healthHostCount := int64(0)

	for _, host := range targetHosts {
		Logger.Info(fmt.Sprintf("%+v", host))
		if host.Healthy {
			healthHostCount += 1
		}
	}

	if healthHostCount >= threshold {
		// Success
		Logger.Info(fmt.Sprintf("Healthy Count for %s : %d/%d", d.AsgNames[region.Region], healthHostCount, threshold))
		d.Slack.SendSimpleMessage(fmt.Sprintf("All instances are healthy in %s  :  %d/%d", d.AsgNames[region.Region], healthHostCount, threshold), d.Stack.Env)
		return true
	}

	Logger.Info(fmt.Sprintf("Healthy count does not meet the requirement(%s) : %d/%d", d.AsgNames[region.Region], healthHostCount, threshold))
	d.Slack.SendSimpleMessage(fmt.Sprintf("Waiting for healthy instances %s  :  %d/%d", d.AsgNames[region.Region], healthHostCount, threshold), d.Stack.Env)

	return false
}

// CheckTerminating checks if all of instances are terminated well
func (d Deployer) CheckTerminating(client aws.AWSClient, target string) bool {
	asgInfo := client.EC2Service.GetMatchingAutoscalingGroup(target)
	if asgInfo == nil {
		Logger.Info("Already deleted autoscaling group : ", target)
		return true
	}

	d.Logger.Info(fmt.Sprintf("Waiting for instance termination in asg %s", target))
	if len(asgInfo.Instances) > 0 {
		d.Logger.Info(fmt.Sprintf("%d instance found : %s", len(asgInfo.Instances), target))
		d.Slack.SendSimpleMessage(fmt.Sprintf("Still %d instance found : %s", len(asgInfo.Instances), target), d.Stack.Env)

		return false
	}
	d.Slack.SendSimpleMessage(fmt.Sprintf(":+1: All instances are deleted : %s", target), d.Stack.Env)

	d.Logger.Info(fmt.Sprintf("Start deleting autoscaling group : %s\n", target))
	ok := client.EC2Service.DeleteAutoscalingSet(target)
	if !ok {
		return false
	}
	d.Logger.Info(fmt.Sprintf("Autoscaling group is deleted : %s\n", target))

	d.Logger.Info(fmt.Sprintf("Start deleting launch templates in %s\n", target))
	err := client.EC2Service.DeleteLaunchTemplates(target)
	if err != nil {
		d.Logger.Errorln(err.Error())
		return false
	}
	d.Logger.Info(fmt.Sprintf("Launch templates are deleted in %s\n", target))

	return true
}

// ResizingAutoScalingGroupToZero set autoscaling group instance count to 0
func (d Deployer) ResizingAutoScalingGroupToZero(client aws.AWSClient, stack, asg string) error {
	d.Logger.Info(fmt.Sprintf("Modifying the size of autoscaling group to 0 : %s(%s)", asg, stack))
	d.Slack.SendSimpleMessage(fmt.Sprintf("Modifying the size of autoscaling group to 0 : %s/%s", asg, stack), d.Stack.Env)
	err := client.EC2Service.UpdateAutoScalingGroup(asg, 0, 0, 0)
	if err != nil {
		d.Logger.Errorln(err.Error())
		return err
	}

	return nil
}

func selectClientFromList(awsClients []aws.AWSClient, region string) (aws.AWSClient, error) {
	for _, c := range awsClients {
		if c.Region == region {
			return c, nil
		}
	}
	return aws.AWSClient{}, errors.New("No AWS Client is selected")
}
