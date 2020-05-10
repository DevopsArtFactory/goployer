package application

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	Logger "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/ssm"
	"os"
)

type Runner struct {
	Logger 	*Logger.Logger
	Builder   Builder
	AWSClient AWSClient
	Slacker Slack
}

type AWSClient struct {
	Region string
	EC2Service *ec2.EC2
	ELBService *elbv2.ELBV2
	CloudWatchService *cloudwatch.CloudWatch
	SSMService *ssm.SSM
}

func NewRunner(builder Builder) Runner {
	return Runner{
		Logger:  Logger.New(),
		Builder: builder,
		AWSClient: _bootstrap_services(builder.Config.Region, builder.Config.AssumeRole),
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
		EC2Service: _get_ec2_client_fn(aws_session, region, creds),
		ELBService: _get_elb_client_fn(aws_session, region, creds),
		CloudWatchService: _get_cloudwatch_client_fn(aws_session, region, creds),
		SSMService: _get_ssm_client_fn(aws_session, region, creds),
	}

	return client
}

func _get_aws_session() *session.Session {
	mySession := session.Must(session.NewSession())
	return mySession
}

func _get_ec2_client_fn(session *session.Session, region string, creds *credentials.Credentials) *ec2.EC2 {
	if creds == nil {
		return ec2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ec2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func _get_elb_client_fn(session *session.Session, region string, creds *credentials.Credentials) *elbv2.ELBV2 {
	if creds == nil {
		return elbv2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return elbv2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func _get_cloudwatch_client_fn(session *session.Session, region string, creds *credentials.Credentials) *cloudwatch.CloudWatch {
	if creds == nil {
		return cloudwatch.New(session, &aws.Config{Region: aws.String(region)})
	}
	return cloudwatch.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func _get_ssm_client_fn(session *session.Session, region string, creds *credentials.Credentials) *ssm.SSM {
	if creds == nil {
		return ssm.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ssm.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}
