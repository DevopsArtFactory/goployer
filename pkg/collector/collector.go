package collector

import (
	"encoding/json"
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	Logger "github.com/sirupsen/logrus"
	"time"
)

var (
	MONTH       = float64(2592000)
	enableStats = true
)

type Collector struct {
	MetricConfig builder.MetricConfig
	MetricClient aws.MetricClient
}

func NewCollector(mc builder.MetricConfig, assumeRole string) Collector {
	return Collector{
		MetricConfig: mc,
		MetricClient: aws.BootstrapMetricService(mc.Region, assumeRole),
	}
}

func (c Collector) CheckStorage(logger *Logger.Logger) error {
	if len(c.MetricConfig.Storage.Type) <= 0 {
		logger.Warnf("you did not specify the storage type so that default storage type will be applied : %s", builder.DEFAULT_METRIC_STORAGE_TYPE)
	}
	if c.MetricConfig.Storage.Type == "dynamodb" {
		isExist, err := c.MetricClient.DynamoDBService.CheckTableExists(c.MetricConfig.Storage.Name)
		if err != nil {
			return err
		}

		if isExist {
			logger.Infof("you already had a table : %s", c.MetricConfig.Storage.Name)
		} else {
			logger.Infof("you don't have a table : %s", c.MetricConfig.Storage.Name)
			if err := c.MetricClient.DynamoDBService.CreateTable(c.MetricConfig.Storage.Name); err != nil {
				return err
			}

			logger.Infof("new table is created : %s", c.MetricConfig.Storage.Name)
		}
	}

	return nil
}

func (c Collector) StampDeployment(stack builder.Stack, config builder.Config, tags []*autoscaling.Tag, asg string, status string, additionalFields map[string]string) error {
	tagsMap := map[string]string{}

	for _, tag := range tags {
		tagsMap[*tag.Key] = *tag.Value
	}

	tagJson, err := json.Marshal(tagsMap)
	if err != nil {
		return err
	}
	tagString := string(tagJson)

	stackJson, err := json.Marshal(stack)
	if err != nil {
		return err
	}
	stackString := string(stackJson)

	configJson, err := json.Marshal(config)
	if err != nil {
		return err
	}
	configString := string(configJson)

	if err := c.MetricClient.DynamoDBService.MakeRecord(stackString, configString, tagString, asg, c.MetricConfig.Storage.Name, status, c.MetricConfig.Metrics.BaseTimezone, additionalFields); err != nil {
		return err
	}

	return err
}

func (c Collector) UpdateStatus(asg string, status string, updateFields map[string]interface{}) error {
	Logger.Debugf("deployment statuses of previous autoscaling groups are started")
	if err := c.MetricClient.DynamoDBService.UpdateRecord("deployment_status", asg, c.MetricConfig.Storage.Name, status, c.MetricConfig.Metrics.BaseTimezone, updateFields); err != nil {
		return err
	}
	Logger.Debugf("deployment statuses of previous autoscaling groups are updated")

	return nil
}

func (c Collector) UpdateStatistics(asg string, updateFields map[string]interface{}) error {
	if err := c.MetricClient.DynamoDBService.UpdateStatistics(asg, c.MetricConfig.Storage.Name, c.MetricConfig.Metrics.BaseTimezone, updateFields); err != nil {
		return err
	}
	Logger.Debugf("deployment metric is updated")

	return nil
}

func (c Collector) GetAdditionalMetric(asg string, tgs []*string, logger *Logger.Logger) (map[string]interface{}, error) {
	item, err := c.MetricClient.DynamoDBService.GetSingleItem(asg, c.MetricConfig.Storage.Name)
	if err != nil {
		return nil, err
	}

	var baseTimeDuration float64
	var startDate time.Time

	curr := tool.GetBaseTimeWithTimestamp(c.MetricConfig.Metrics.BaseTimezone)
	logger.Debugf("current time in timezone %s : %s", c.MetricConfig.Metrics.BaseTimezone, curr)

	ret := map[string]interface{}{}
	for k, v := range item {
		if k == "deployed_date" {
			d, _ := time.Parse(time.RFC3339, *v.S)
			diff := curr.Sub(d)
			ret["uptime_second"] = fmt.Sprintf("%f", diff.Seconds())
			ret["uptime_minute"] = fmt.Sprintf("%f", diff.Minutes())
			ret["uptime_hour"] = fmt.Sprintf("%f", diff.Hours())

			baseTimeDuration = diff.Seconds()
			startDate = d
		}
	}

	var period int64
	if len(tgs) > 0 && enableStats {
		// if baseTimeDuration is over a month which is the maximum duration of cloudwatch
		// fix the time to one month
		if baseTimeDuration > MONTH {
			period = int64(MONTH)
			startDate = curr.Add(-2592000 * time.Second)
		}

		startDate = tool.GetBaseStartTime(startDate)

		logger.Debugf("StartDate : %s\n", startDate)

		targetRequest, err := c.MetricClient.CloudWatchService.GetRequestStatistics(tgs, startDate, curr, period, logger)
		if err != nil {
			return ret, err
		}

		ret["stat"] = targetRequest
		ret["timezone"] = c.MetricConfig.Metrics.BaseTimezone
	}

	return ret, nil
}
