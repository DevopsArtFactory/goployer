package application

import (
	"github.com/aws/aws-sdk-go/aws/session"
)

type AWSClient struct {
	Region string
	EC2Service EC2Client
	ELBService ELBV2Client
	CloudWatchService CloudWatchClient
	SSMService SSMClient
}

func _get_aws_session() *session.Session {
	mySession := session.Must(session.NewSession())
	return mySession
}
