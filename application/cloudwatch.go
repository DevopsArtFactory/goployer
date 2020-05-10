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
		Client: _get_cloudwatch_client_fn(session, region, creds),
	}
}

func _get_cloudwatch_client_fn(session *session.Session, region string, creds *credentials.Credentials) *cloudwatch.CloudWatch {
	if creds == nil {
		return cloudwatch.New(session, &aws.Config{Region: aws.String(region)})
	}
	return cloudwatch.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

