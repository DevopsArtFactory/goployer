package application

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"strings"
)

type EC2Client struct {
	Client *ec2.EC2
	AsClient *autoscaling.AutoScaling
}

func NewEC2Client(session *session.Session, region string, creds *credentials.Credentials) EC2Client {
	return EC2Client{
		Client: _get_ec2_client_fn(session, region, creds),
		AsClient: _get_asg_client_fn(session, region, creds),
	}
}

func _get_ec2_client_fn(session *session.Session, region string, creds *credentials.Credentials) *ec2.EC2 {
	if creds == nil {
		return ec2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ec2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}


func _get_asg_client_fn(session *session.Session, region string, creds *credentials.Credentials) *autoscaling.AutoScaling {
	if creds == nil {
		return autoscaling.New(session, &aws.Config{Region: aws.String(region)})
	}
	return autoscaling.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func (c EC2Client) GetAllMatchingAutoscalingGroups(prefix string) []*autoscaling.Group {
	asgGroups := []*autoscaling.Group{}
	asgGroups = _get_autoscaling_groups(c.AsClient, asgGroups, nil)

	ret := []*autoscaling.Group{}
	for _, asgGroup := range asgGroups {
		if strings.HasPrefix(*asgGroup.AutoScalingGroupName, prefix) {
			ret = append(ret, asgGroup)
		}
	}

	return ret
}

func _get_autoscaling_groups(client *autoscaling.AutoScaling, asgGroup []*(autoscaling.Group), nextToken *string) []*autoscaling.Group {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		NextToken: nextToken,
	}
	ret, err := client.DescribeAutoScalingGroups(input)
	if err != nil {
		_fatal_error(err)
	}

	asgGroup = append(asgGroup, ret.AutoScalingGroups...)

	if ret.NextToken != nil {
		return _get_autoscaling_groups(client, asgGroup, ret.NextToken)
	}

	return asgGroup
}
