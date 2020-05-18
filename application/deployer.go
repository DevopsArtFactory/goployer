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
	Logger 	 		*Logger.Logger
	Stack	 		Stack
	AwsConfig		AWSConfig
	AWSClients 		[]AWSClient
	LocalProvider 	UserdataProvider
}

func _get_current_version(prev_versions []int) int {
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

//Check timeout
func _check_timeout(start int64, timeout int64) bool {
	now := time.Now().Unix()
	timeoutSec := timeout * 60

	//Over timeout
	if (now - start) > timeoutSec {
		Logger.Error("Timeout has been exceeded : %s minutes", timeout)
		os.Exit(1)
	}

	return false
}

func _select_client_from_list(awsClients []AWSClient, region string) (AWSClient, error) {
	for _, c := range awsClients {
		if c.Region == region {
			return c, nil
		}
	}
	return AWSClient{}, errors.New("No AWS Client is selected")
}