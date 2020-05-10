package application

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
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

