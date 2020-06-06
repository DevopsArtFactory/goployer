package application

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type CloudWatchClient struct {
	Client *cloudwatch.CloudWatch
}

func NewCloudWatchClient(session *session.Session, region string, creds *credentials.Credentials) CloudWatchClient {
	return CloudWatchClient{
		Client: getCloudwatchClientFn(session, region, creds),
	}
}

func getCloudwatchClientFn(session *session.Session, region string, creds *credentials.Credentials) *cloudwatch.CloudWatch {
	if creds == nil {
		return cloudwatch.New(session, &aws.Config{Region: aws.String(region)})
	}
	return cloudwatch.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

//CreateScalingAlarms creates scaling alarms
func (c CloudWatchClient) CreateScalingAlarms(asg_name string, alarms []AlarmConfigs, policyArns []*string, policies []string) error {
	if len(alarms) == 0 {
		return nil
	}

	return nil
}
