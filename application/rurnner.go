package application

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	Logger "github.com/sirupsen/logrus"

	"os"
)

type Runner struct {
	Logger 	*Logger.Logger
	Builder   Builder
	Slacker Slack
}

func NewRunner(builder Builder) Runner {
	return Runner{
		Logger:  Logger.New(),
		Builder: builder,
		Slacker: NewSlackClient(),
	}
}

func (r Runner) WarmUp()  {
	//logger.SetFormatter(&Logger.JSONFormatter{})
	r.Logger.SetOutput(os.Stdout)
	r.Logger.SetLevel(Logger.InfoLevel)

	r.Logger.Info("Warm up before starting deployment")
}

// Run
func (r Runner) Run()  {

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	//Send Beginning Message

	//Run
	r.Logger.Info("Beginning deploy for application: "+r.Builder.AwsConfig.Name)
	deployer := _NewBlueGrean(
		r.Logger,
		r.Builder.AwsConfig.ReplacementType,
		r.Builder.Frigga.Prefix,
		_bootstrap_services(r.Builder.Config.Region, r.Builder.Config.AssumeRole))

	// Deploy with builder
	deployer.Deploy(r.Builder)
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

