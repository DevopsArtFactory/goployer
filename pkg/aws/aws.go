package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

var (
	DEFAULT_HEALTHCHECK_TYPE         = "EC2"
	DEFAULT_HEALTHCHECK_GRACE_PERIOD = 300
)

type AWSClient struct {
	Region            string
	EC2Service        EC2Client
	ELBV2Service      ELBV2Client
	ELBService        ELBClient
	CloudWatchService CloudWatchClient
	SSMService        SSMClient
}

type MetricClient struct {
	Region            string
	DynamoDBService   DynamoDBClient
	CloudWatchService CloudWatchClient
}

type ManifestClient struct {
	Region    string
	S3Service S3Client
}

func GetAwsSession() *session.Session {
	mySession := session.Must(session.NewSession())
	return mySession
}

func MakeStringArrayToAwsStrings(arr []string) []*string {
	if len(arr) == 0 {
		return nil
	}

	ret := []*string{}
	for _, s := range arr {
		ret = append(ret, aws.String(s))
	}

	return ret
}

func BootstrapServices(region string, assume_role string) AWSClient {
	aws_session := GetAwsSession()

	var creds *credentials.Credentials
	if len(assume_role) != 0 {
		creds = stscreds.NewCredentials(aws_session, assume_role)
	}

	//Get all clients
	client := AWSClient{
		Region:            region,
		EC2Service:        NewEC2Client(aws_session, region, creds),
		ELBV2Service:      NewELBV2Client(aws_session, region, creds),
		ELBService:        NewELBClient(aws_session, region, creds),
		CloudWatchService: NewCloudWatchClient(aws_session, region, creds),
		SSMService:        NewSSMClient(aws_session, region, creds),
	}

	return client
}

func BootstrapMetricService(region string, assume_role string) MetricClient {
	aws_session := GetAwsSession()

	var creds *credentials.Credentials
	if len(assume_role) != 0 {
		creds = stscreds.NewCredentials(aws_session, assume_role)
	}

	//Get all clients
	client := MetricClient{
		Region:            region,
		DynamoDBService:   NewDynamoDBClient(aws_session, region, nil),
		CloudWatchService: NewCloudWatchClient(aws_session, region, creds),
	}

	return client
}

func BootstrapManifestService(region string, assume_role string) ManifestClient {
	aws_session := GetAwsSession()

	var creds *credentials.Credentials
	if len(assume_role) != 0 {
		creds = stscreds.NewCredentials(aws_session, assume_role)
	}

	//Get all clients
	client := ManifestClient{
		Region:    region,
		S3Service: NewS3Client(aws_session, region, creds),
	}

	return client
}
