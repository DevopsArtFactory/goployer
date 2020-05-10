package application

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type SSMClient struct {
	Client *ssm.SSM
}

func NewSSMClient(session *session.Session, region string, creds *credentials.Credentials) SSMClient {
	return SSMClient{
		Client: _get_ssm_client_fn(session, region, creds),
	}
}

func _get_ssm_client_fn(session *session.Session, region string, creds *credentials.Credentials) *ssm.SSM {
	if creds == nil {
		return ssm.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ssm.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

