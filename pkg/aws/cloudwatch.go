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
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type CloudWatchClient struct {
	Client *cloudwatch.CloudWatch
}

func NewCloudWatchClient(session client.ConfigProvider, region string, creds *credentials.Credentials) CloudWatchClient {
	return CloudWatchClient{
		Client: getCloudwatchClientFn(session, region, creds),
	}
}

func getCloudwatchClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *cloudwatch.CloudWatch {
	if creds == nil {
		return cloudwatch.New(session, &aws.Config{Region: aws.String(region)})
	}
	return cloudwatch.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

//CreateScalingAlarms creates scaling alarms
func (c CloudWatchClient) CreateScalingAlarms(asgName string, alarms []schemas.AlarmConfigs, policyArns map[string]string) error {
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
		if err := c.CreateCloudWatchAlarm(asgName, alarm); err != nil {
			return err
		}
	}

	return nil
}

// Create cloudwatch alarms for autoscaling group
func (c CloudWatchClient) CreateCloudWatchAlarm(asgName string, alarm schemas.AlarmConfigs) error {
	input := &cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String(alarm.Name),
		AlarmActions:       aws.StringSlice(alarm.AlarmActions),
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
				Value: aws.String(asgName),
			},
		},
	}

	_, err := c.Client.PutMetricAlarm(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case cloudwatch.ErrCodeLimitExceededFault:
				Logger.Errorln(cloudwatch.ErrCodeLimitExceededFault, aerr.Error())
			default:
				Logger.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Errorln(err.Error())
		}
		return err
	}

	Logger.Info(fmt.Sprintf("New metric alarm is created : %s / asg : %s", alarm.Name, asgName))

	return nil
}

// GetRequestStatics returns statistics for terminating autoscaling group
func (c CloudWatchClient) GetRequestStatistics(tgs []*string, startTime, terminatedDate time.Time, logger *Logger.Logger) (map[string]map[string]float64, error) {
	ret := map[string]map[string]float64{}

	resetStartTime := startTime
	if len(tgs) > 0 {
		for _, tg := range tgs {
			tgName := (*tg)[strings.LastIndex(*tg, ":")+1:]
			logger.Debugf("GetRequestStatistics: %s", tgName)

			appliedPeriod := constants.HourToSec
			vSum := float64(0)

			startTime = resetStartTime
			isFinished := false
			for !isFinished {
				id := fmt.Sprintf("m%s", tool.GetTimePrefix(startTime))
				logger.Debugf("Metric id : %s", id)

				endTime := tool.GetBaseTime(startTime.Add(time.Duration(constants.DayToSec) * time.Second)).Add(-1 * time.Second)
				if CheckMetricTimeValidation(terminatedDate, endTime) {
					logger.Debugf("Terminated Date is earlier than End Date: %s/%s", terminatedDate, endTime)
					endTime = terminatedDate
				}
				logger.Debugf("Start Time : %s, End Time : %s, Applied period: %d", startTime, endTime, appliedPeriod)

				if !CheckMetricTimeValidation(startTime, endTime) {
					logger.Debugf("Finish gathering metrics")
					break
				}

				v, s, err := c.GetOneDayStatistics(tgName, startTime, endTime, appliedPeriod, id)
				if err != nil {
					return nil, err
				}

				if v != nil {
					ret[tgName] = v
					vSum += s
				}

				startTime = endTime.Add(1 * time.Second)
				logger.Debugf("Next Start Time : %s", startTime)

				if !CheckMetricTimeValidation(startTime, terminatedDate) {
					logger.Debugf("Finish gathering metrics")
					isFinished = true
				}
			}

			if ret[tgName] == nil {
				ret[tgName] = map[string]float64{}
			}
			ret[tgName]["total"] = vSum
		}
	}

	return ret, nil
}

// GetOneDayStatistics returns all stats of one day
func (c CloudWatchClient) GetOneDayStatistics(tg string, startTime, endTime time.Time, period int64, id string) (map[string]float64, float64, error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
	}

	var mdq []*cloudwatch.MetricDataQuery
	mdq = append(mdq, &cloudwatch.MetricDataQuery{
		Id:         aws.String(id),
		ReturnData: aws.Bool(false),
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("TargetGroup"),
						Value: aws.String(tg),
					},
				},
				MetricName: aws.String("RequestCountPerTarget"),
				Namespace:  aws.String("AWS/ApplicationELB"),
			},
			Period: aws.Int64(period),
			Stat:   aws.String("Sum"),
		},
	})

	mdq = append(mdq, &cloudwatch.MetricDataQuery{
		Expression: aws.String(id),
		Id:         aws.String(fmt.Sprintf("%s%s", "t", id)),
		Label:      aws.String("RequestSum"),
	})

	input.MetricDataQueries = mdq

	result, err := c.Client.GetMetricData(input)
	if err != nil {
		return nil, 0, err
	}

	// If no result exists, then set it to zero
	if len(result.MetricDataResults) == 0 {
		return nil, 0, nil
	}

	Logger.Infoln(result.MetricDataResults)

	ret := map[string]float64{}
	sum := float64(0)
	for i, t := range result.MetricDataResults[0].Timestamps {
		val := *result.MetricDataResults[0].Values[i]
		ret[t.Format(time.RFC3339)] = val
		sum += val
	}

	return ret, sum, nil
}

func CheckMetricTimeValidation(startTime time.Time, endTime time.Time) bool {
	return endTime.Sub(startTime) > 0
}
