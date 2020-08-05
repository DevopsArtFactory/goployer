package aws

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/tool"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elb"
	Logger "github.com/sirupsen/logrus"
)

type ELBClient struct {
	Client *elb.ELB
}

type HealthcheckELBHost struct {
	InstanceId     string
	LifecycleState string
	TargetStatus   string
	HealthStatus   string
	Healthy        bool
}

func NewELBClient(session *session.Session, region string, creds *credentials.Credentials) ELBClient {
	return ELBClient{
		Client: getELBClientFn(session, region, creds),
	}
}

func getELBClientFn(session *session.Session, region string, creds *credentials.Credentials) *elb.ELB {
	if creds == nil {
		return elb.New(session, &aws.Config{Region: aws.String(region)})
	}
	return elb.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

// GetHostInELB returns instances in ELB
func (e ELBClient) GetHealthyHostInELB(group *autoscaling.Group, elbName string) ([]HealthcheckHost, error) {
	Logger.Debug(fmt.Sprintf("[Checking healthy host count] Autoscaling Group: %s", *group.AutoScalingGroupName))

	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	result, err := e.Client.DescribeInstanceHealth(input)
	if err != nil {
		return nil, err
	}

	ret := []HealthcheckHost{}
	targetInstances := []string{}
	for _, instance := range group.Instances {
		targetInstances = append(targetInstances, *instance.InstanceId)
	}
	for _, instance := range result.InstanceStates {
		if tool.IsStringInArray(*instance.InstanceId, targetInstances) {
			ret = append(ret, HealthcheckHost{
				InstanceId:     *instance.InstanceId,
				LifecycleState: *instance.State,
				Healthy:        *instance.State == "InService",
			})
		}
	}

	return ret, nil
}
