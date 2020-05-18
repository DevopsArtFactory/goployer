package application

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elbv2"
	Logger "github.com/sirupsen/logrus"
	"os"
)

type ELBV2Client struct {
	Client *elbv2.ELBV2
}

type HealthcheckHost struct {
	InstanceId 		string
	LifecycleState 	string
	TargetStatus	string
	HealthStatus    string
	healthy			bool
}

func NewELBV2Client(session *session.Session, region string, creds *credentials.Credentials) ELBV2Client {
	return ELBV2Client{
		Client: _get_elb_client_fn(session, region, creds),
	}
}

func _get_elb_client_fn(session *session.Session, region string, creds *credentials.Credentials) *elbv2.ELBV2 {
	if creds == nil {
		return elbv2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return elbv2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}


func (e ELBV2Client) GetTargetGroupARNs(target_groups []string) []*string {
	if len(target_groups) == 0 {
		return nil
	}

	input := &elbv2.DescribeTargetGroupsInput{
		Names: _make_string_array_to_aws_strings(target_groups),
	}

	result, err := e.Client.DescribeTargetGroups(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				fmt.Println(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	ret := []*string{}
	for _, group := range result.TargetGroups {
		ret = append(ret, group.TargetGroupArn)
	}

	return ret
}

func (e ELBV2Client) GetHostInTarget(group *autoscaling.Group, target_group_arn *string) []HealthcheckHost {
	Logger.Info(fmt.Sprintf("[Checking healthy host count] Autoscaling Group: %s", *group.AutoScalingGroupName))


	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(*target_group_arn),
	}

	result, err := e.Client.DescribeTargetHealth(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeInvalidTargetException:
				fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeHealthUnavailableException:
				fmt.Println(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	ret := []HealthcheckHost{}
	for _, instance := range group.Instances {
		target_state := INITIAL_STATUS
		for _, hd := range result.TargetHealthDescriptions {
			if *hd.Target.Id == *instance.InstanceId {
				target_state = *hd.TargetHealth.State
				break
			}
		}

		ret = append(ret, HealthcheckHost{
			InstanceId:     *instance.InstanceId,
			LifecycleState: *instance.LifecycleState,
			TargetStatus:   target_state,
			HealthStatus:   *instance.HealthStatus,
			healthy:        *instance.LifecycleState == "InService" && target_state == "healthy" && *instance.HealthStatus == "Healthy",
		})
	}
	return ret
}