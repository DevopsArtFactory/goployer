package aws

import (
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	Logger "github.com/sirupsen/logrus"
)

type CloudWatchClient struct {
	Client *cloudwatch.CloudWatch
}

func NewCloudWatchClient(session *session.Session, region string, creds *credentials.Credentials) CloudWatchClient {
	return CloudWatchClient{
		Client: getCloudwatchClientFn(session, region, creds),
	}
}

func getCloudwatchClientFn(session *session.Session, region string, creds *credentials.Credentials) *cloudwatch.CloudWatch {
	if creds == nil {
		return cloudwatch.New(session, &aws.Config{Region: aws.String(region)})
	}
	return cloudwatch.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

//CreateScalingAlarms creates scaling alarms
func (c CloudWatchClient) CreateScalingAlarms(asg_name string, alarms []builder.AlarmConfigs, policyArns map[string]string) error {
	if len(alarms) == 0 {
		return nil
	}

	//Create cloudwatch alarms
	for _, alarm := range alarms {
		arns := []string{}
		for _, action := range alarm.AlarmActions {
			arns = append(arns, policyArns[action])
		}
		alarm.AlarmActions = arns
		if err := c.CreateCloudWatchAlarm(asg_name, alarm); err != nil {
			return err
		}
	}

	return nil
}

// Create cloudwatch alarms for autoscaling group
func (c CloudWatchClient) CreateCloudWatchAlarm(asg_name string, alarm builder.AlarmConfigs) error {
	input := &cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String(alarm.Name),
		AlarmActions:       MakeStringArrayToAwsStrings(alarm.AlarmActions),
		MetricName:         aws.String(alarm.Metric),
		Namespace:          aws.String(alarm.Namespace),
		Statistic:          aws.String(alarm.Statistic),
		ComparisonOperator: aws.String(alarm.Comparison),
		Threshold:          aws.Float64(alarm.Threshold),
		Period:             aws.Int64(alarm.Period),
		EvaluationPeriods:  aws.Int64(alarm.EvaluationPeriods),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("AutoScalingGroupName"),
				Value: aws.String(asg_name),
			},
		},
	}

	_, err := c.Client.PutMetricAlarm(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case cloudwatch.ErrCodeLimitExceededFault:
				fmt.Println(cloudwatch.ErrCodeLimitExceededFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	Logger.Info(fmt.Sprintf("New metric alarm is created : %s / asg : %s", alarm.Name, asg_name))

	return nil
}
