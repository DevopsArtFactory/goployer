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
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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

// GetAwsConfig generates new aws config
func GetAwsConfig(ctx context.Context, region string) (aws.Config, error) {
	profile := viper.GetString("profile")

	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	return config.LoadDefaultConfig(ctx, opts...)
}

// BootstrapServices creates AWS client list
func BootstrapServices(region string, assumeRole string) Client {
	ctx := context.Background()
	cfg, err := GetAwsConfig(ctx, region)
	if err != nil {
		panic(err)
	}

	if assumeRole != "" {
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = stscreds.NewAssumeRoleProvider(stsClient, assumeRole)
	}

	return Client{
		Region:            region,
		EC2Service:        NewEC2Client(cfg),
		ELBV2Service:      NewELBV2Client(cfg),
		ELBService:        NewELBClient(cfg),
		CloudWatchService: NewCloudWatchClient(cfg),
		SSMService:        NewSSMClient(cfg),
	}
}

func BootstrapMetricService(region string, assumeRole string) MetricClient {
	ctx := context.Background()
	cfg, err := GetAwsConfig(ctx, region)
	if err != nil {
		panic(err)
	}

	if assumeRole != "" {
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = stscreds.NewAssumeRoleProvider(stsClient, assumeRole)
	}

	return MetricClient{
		Region:            region,
		DynamoDBService:   NewDynamoDBClient(cfg),
		CloudWatchService: NewCloudWatchClient(cfg),
	}
}

func BootstrapManifestService(region string, assumeRole string) ManifestClient {
	ctx := context.Background()
	cfg, err := GetAwsConfig(ctx, region)
	if err != nil {
		panic(err)
	}

	if assumeRole != "" {
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = stscreds.NewAssumeRoleProvider(stsClient, assumeRole)
	}

	return ManifestClient{
		Region:    region,
		S3Service: NewS3Client(cfg),
	}
}
