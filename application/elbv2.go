package application

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"os"
)

type ELBV2Client struct {
	Client *elbv2.ELBV2
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
