package application

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	Logger "github.com/sirupsen/logrus"
	"os"
	"time"
)

// Deployer per stack
type Deployer struct {
	Mode 	 		string
	AsgNames		map[string]string
	PrevAsgs		map[string][]string
	Logger 	 		*Logger.Logger
	Stack	 		Stack
	AwsConfig		AWSConfig
	AWSClients 		[]AWSClient
	LocalProvider 	UserdataProvider
}

// getCurrentVersion returns current version for current deployment step
func getCurrentVersion(prev_versions []int) int {
	if len(prev_versions) == 0 {
		return 0
	}
	return (prev_versions[len(prev_versions)-1] + 1) % 1000
}

// Polling for healthcheck
func (d Deployer) polling(region RegionConfig, asg *autoscaling.Group, client AWSClient) bool {
	if *asg.AutoScalingGroupName == "" {
		error_logging(fmt.Sprintf("No autoscaling found for %s", d.AsgNames[region.Region]))
	}

	healthcheck_target_group 	 := region.HealthcheckTargetGroup
	healthcheck_target_group_arn := (client.ELBService.GetTargetGroupARNs([]string{healthcheck_target_group}))[0]

	threshold := d.Stack.Capacity.Desired
	targetHosts := client.ELBService.GetHostInTarget(asg, healthcheck_target_group_arn)

	healthHostCount := int64(0)

	for _, host := range targetHosts {
		Logger.Info(fmt.Sprintf("%+v", host))
		if host.healthy {
			healthHostCount += 1
		}
	}

	if healthHostCount >= threshold {
		// Success
		Logger.Info(fmt.Sprintf("Healthy Count for %s : %d/%d", d.AsgNames[region.Region], healthHostCount, threshold))
		return true
	}

	Logger.Info(fmt.Sprintf("Healthy count does not meet the requirement(%s) : %d/%d", d.AsgNames[region.Region], healthHostCount, threshold))

	return false
}

func (d Deployer) CheckTerminating(client AWSClient, region, target string) bool {
	asgInfo := client.EC2Service.GetMatchingAutoscalingGroup(target)
	if asgInfo == nil {
		return false
	}

	d.Logger.Info(fmt.Sprintf("Waiting for instance termination in asg %s", target))
	if len(asgInfo.Instances) > 0 {
		d.Logger.Info(fmt.Sprintf("%d instance found", len(asgInfo.Instances)))
		return false
	}

	d.Logger.Info(fmt.Sprintf("Start deleting autoscaling group : %s\n", target))
	ok := client.EC2Service.DeleteAutoscalingSet(target)
	if !ok {
		return false
	}

	d.Logger.Info(fmt.Sprintf("Start deleting launch configurations in %s\n", target))
	err := client.EC2Service.DeleteLaunchConfigurations(target)
	if err != nil {
		d.Logger.Errorln(err.Error())
		return false
	}

	return true
}

func (d Deployer) ResizingAutoScalingGroupToZero(client AWSClient, stack, asg string) error {
	d.Logger.Info(fmt.Sprintf("Modifying the size of autoscaling group to 0 : %s(%s)", asg, stack))
	err := client.EC2Service.UpdateAutoScalingGroup(asg, 0, 0, 0)
	if err != nil {
		d.Logger.Errorln(err.Error())
		return err
	}

	return nil
}

//Check timeout
func checkTimeout(start int64, timeout int64) bool {
	now := time.Now().Unix()
	timeoutSec := timeout * 60

	//Over timeout
	if (now - start) > timeoutSec {
		Logger.Error("Timeout has been exceeded : %s minutes", timeout)
		os.Exit(1)
	}

	return false
}

func selectClientFromList(awsClients []AWSClient, region string) (AWSClient, error) {
	for _, c := range awsClients {
		if c.Region == region {
			return c, nil
		}
	}
	return AWSClient{}, errors.New("No AWS Client is selected")
}
