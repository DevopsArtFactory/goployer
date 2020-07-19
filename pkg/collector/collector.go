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

	if err := c.MetricClient.DynamoDBService.MakeRecord(stackString, configString, tagString, asg, c.MetricConfig.Storage.Name, status, additionalFields); err != nil {
		return err
	}

	return err
}

func (c Collector) UpdateStatus(asg string, status string, updateFields map[string]string) error {
	if err := c.MetricClient.DynamoDBService.UpdateRecord("deployment_status", asg, c.MetricConfig.Storage.Name, status, updateFields); err != nil {
		return err
	}
	Logger.Debugf("deployment metric is updated")

	return nil
}

func (c Collector) GetAdditionalMetric(asg string) (map[string]string, error) {
	item, err := c.MetricClient.DynamoDBService.GetSingleItem(asg, c.MetricConfig.Storage.Name)
	if err != nil {
		return nil, err
	}

	ret := map[string]string{}
	for k, v := range item {
		if k == "deployed_date_kst" {
			curr := tool.GetKstTimestamp()
			d, _ := time.Parse(time.RFC3339, *v.S)
			diff := curr.Sub(d)
			ret["uptime_second"] = fmt.Sprintf("%f", diff.Seconds())
			ret["uptime_minute"] = fmt.Sprintf("%f", diff.Minutes())
			ret["uptime_hour"] = fmt.Sprintf("%f", diff.Hours())
		}
	}

	return ret, nil
}
