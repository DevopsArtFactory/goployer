package application

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
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


func _bootstrap_services(region string, assume_role string) AWSClient {
	aws_session := _get_aws_session()

	var creds *credentials.Credentials
	if len(assume_role) != 0  {
		creds = stscreds.NewCredentials(aws_session, assume_role)
	}

	//Get all clients
	client := AWSClient{
		Region: region,
		EC2Service: NewEC2Client(aws_session, region, creds),
		ELBService: NewELBV2Client(aws_session, region, creds),
		CloudWatchService: NewCloudWatchClient(aws_session, region, creds),
		SSMService: NewSSMClient(aws_session, region, creds),
	}

	return client
}

