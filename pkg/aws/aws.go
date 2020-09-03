/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/viper"
)

type Client struct {
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

// GetAwsSession generates new aws session
func GetAwsSession() *session.Session {
	profile := viper.GetString("profile")

	mySession := session.Must(
		session.NewSession(&aws.Config{
			Credentials: credentials.NewCredentials(&credentials.SharedCredentialsProvider{
				Filename: defaults.SharedCredentialsFilename(),
				Profile:  profile,
			}),
		}),
	)
	return mySession
}

// BootstrapServices creates AWS client list
func BootstrapServices(region string, assumeRole string) Client {
	awsSession := GetAwsSession()

	var creds *credentials.Credentials
	if len(assumeRole) != 0 {
		creds = stscreds.NewCredentials(awsSession, assumeRole)
	}

	//Get all clients
	client := Client{
		Region:            region,
		EC2Service:        NewEC2Client(awsSession, region, creds),
		ELBV2Service:      NewELBV2Client(awsSession, region, creds),
		ELBService:        NewELBClient(awsSession, region, creds),
		CloudWatchService: NewCloudWatchClient(awsSession, region, creds),
		SSMService:        NewSSMClient(awsSession, region, creds),
	}

	return client
}

func BootstrapMetricService(region string, assumeRole string) MetricClient {
	awsSession := GetAwsSession()

	var creds *credentials.Credentials
	if len(assumeRole) != 0 {
		creds = stscreds.NewCredentials(awsSession, assumeRole)
	}

	//Get all clients
	client := MetricClient{
		Region:            region,
		DynamoDBService:   NewDynamoDBClient(awsSession, region, nil),
		CloudWatchService: NewCloudWatchClient(awsSession, region, creds),
	}

	return client
}

func BootstrapManifestService(region string, assumeRole string) ManifestClient {
	awsSession := GetAwsSession()

	var creds *credentials.Credentials
	if len(assumeRole) != 0 {
		creds = stscreds.NewCredentials(awsSession, assumeRole)
	}

	//Get all clients
	client := ManifestClient{
		Region:    region,
		S3Service: NewS3Client(awsSession, region, creds),
	}

	return client
}
