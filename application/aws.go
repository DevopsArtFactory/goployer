package application

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var (
	DEFAULT_HEALTHCHECK_TYPE="EC2"
	DEFAULT_HEALTHCHECK_GRACE_PERIOD=300
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

func _make_string_array_to_aws_strings (arr []string) []*string {
	if len(arr) == 0 {
		return nil
	}

	ret := []*string{}
	for _, s := range arr {
		ret = append(ret, aws.String(s))
	}

	return ret
}


