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

package collector

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Collector struct {
	MetricConfig      schemas.MetricConfig
	MetricClient      aws.MetricClient
	TargetMetricsList []TargetMetrics
}

// Detailed metric structure
type TargetMetrics struct {
	// Metric name
	Name string

	// Enable
	Enable bool

	// Mapping function for metric
	MappingFunction func(*HelperStruct, *Logger.Logger, aws.MetricClient, string) (map[string]interface{}, error)
}

type HelperStruct struct {
	BaseTimeDuration float64
	StartDate        time.Time
	AutoScalingGroup string
	CurrentTime      time.Time
	TargetGroups     []*string
	LoadBalancers    []*string
	Storage          string
}

// NewCollector creates new collector
func NewCollector(mc schemas.MetricConfig, assumeRole string) Collector {
	return Collector{
		MetricConfig:      mc,
		MetricClient:      aws.BootstrapMetricService(mc.Region, assumeRole),
		TargetMetricsList: SetTargetMetrics(),
	}
}

// SetTargetMetrics sets what collector will gather for measurement
func SetTargetMetrics() []TargetMetrics {
	tm := []TargetMetrics{
		{
			Name:            "uptime",
			Enable:          true,
			MappingFunction: GatherUptime,
		},
		{
			Name:            "tg_request_count",
			Enable:          true,
			MappingFunction: GetTargetGroupRequestCount,
		},
		{
			Name:            "lb_request_count",
			Enable:          true,
			MappingFunction: GetLoadBalancerRequestCount,
		},
	}

	return tm
}

// CheckStorage is for checking metrics storage
func (c Collector) CheckStorage(logger *Logger.Logger) error {
	if len(c.MetricConfig.Storage.Type) == 0 {
		logger.Warnf("you did not specify the storage type so that default storage type will be applied : %s", constants.DefaultMetricStorageType)
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

// StampDeployment records deployment information to storage
func (c Collector) StampDeployment(stack schemas.Stack, config builder.Config, tags []*autoscaling.Tag, asg string, status string, additionalFields map[string]string) error {
	tagsMap := map[string]string{}

	for _, tag := range tags {
		tagsMap[*tag.Key] = *tag.Value
	}

	tagJSON, err := json.Marshal(tagsMap)
	if err != nil {
		return err
	}
	tagString := string(tagJSON)

	stackJSON, err := json.Marshal(stack)
	if err != nil {
		return err
	}
	stackString := string(stackJSON)

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	configString := string(configJSON)

	if err := c.MetricClient.DynamoDBService.MakeRecord(stackString, configString, tagString, asg, c.MetricConfig.Storage.Name, status, c.MetricConfig.Metrics.BaseTimezone, additionalFields); err != nil {
		return err
	}

	return err
}

// UpdateStatus updates status of deployment on the table
func (c Collector) UpdateStatus(asg string, status string, updateFields map[string]interface{}) error {
	Logger.Debugf("deployment statuses of previous autoscaling groups are started")
	if err := c.MetricClient.DynamoDBService.UpdateRecord("deployment_status", asg, c.MetricConfig.Storage.Name, status, c.MetricConfig.Metrics.BaseTimezone, updateFields); err != nil {
		return err
	}
	Logger.Debugf("deployment statuses of previous autoscaling groups are updated")

	return nil
}

// UpdateStatistics update value of metric table
func (c Collector) UpdateStatistics(asg string, updateFields map[string]interface{}) error {
	if err := c.MetricClient.DynamoDBService.UpdateStatistics(asg, c.MetricConfig.Storage.Name, c.MetricConfig.Metrics.BaseTimezone, updateFields); err != nil {
		return err
	}
	return nil
}

// GetAdditionalMetric retrieves additional metrics to store
func (c Collector) GetAdditionalMetric(asg string, tgs []*string, lbs []*string, logger *Logger.Logger) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	hs := HelperStruct{
		BaseTimeDuration: constants.ZeroFloat64,
		StartDate:        time.Now(),
		AutoScalingGroup: asg,
		CurrentTime:      tool.GetBaseTimeWithTimezone(c.MetricConfig.Metrics.BaseTimezone),
		TargetGroups:     tgs,
		LoadBalancers:    lbs,
		Storage:          c.MetricConfig.Storage.Name,
	}

	logger.Debugf("current time in timezone %s : %s", c.MetricConfig.Metrics.BaseTimezone, hs.CurrentTime)

	for _, f := range c.TargetMetricsList {
		if f.Enable {
			temp, err := f.MappingFunction(&hs, logger, c.MetricClient, f.Name)
			if err != nil {
				return nil, err
			}

			for k, v := range temp {
				ret[k] = v
			}
		}
	}

	return ret, nil
}

// GatherUptime gathers uptime information about autoscaling group
func GatherUptime(hs *HelperStruct, _ *Logger.Logger, client aws.MetricClient, name string) (map[string]interface{}, error) {
	item, err := client.DynamoDBService.GetSingleItem(hs.AutoScalingGroup, hs.Storage)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, fmt.Errorf("there is no dynamodb record for %s", hs.AutoScalingGroup)
	}

	ret := map[string]interface{}{}
	for k, v := range item {
		if k == "deployed_date" {
			d, _ := time.Parse(time.RFC3339, *v.S)
			diff := hs.CurrentTime.Sub(d)
			ret["uptime_second"] = fmt.Sprintf("%f", diff.Seconds())
			ret["uptime_minute"] = fmt.Sprintf("%f", diff.Minutes())
			ret["uptime_hour"] = fmt.Sprintf("%f", diff.Hours())
			ret[name] = fmt.Sprintf("%f", diff.Hours())

			hs.BaseTimeDuration = diff.Seconds()
			hs.StartDate = d
			break
		}
	}

	return ret, nil
}

// GetTargetGroupRequestCount retrieves target group request count
func GetTargetGroupRequestCount(hs *HelperStruct, logger *Logger.Logger, client aws.MetricClient, name string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	if len(hs.TargetGroups) > 0 && (hs.StartDate.Sub(constants.MinTimestamp) > 0) && constants.DefaultEnableStats {
		// if baseTimeDuration is over a month which is the maximum duration of cloudwatch
		// fix the time to one month
		if hs.BaseTimeDuration > constants.MonthToSec {
			hs.StartDate = hs.CurrentTime.Add(-2592000 * time.Second)
		}

		//hs.StartDate = tool.GetBaseStartTime(hs.StartDate)
		if hs.CurrentTime.Sub(hs.StartDate) < 0 {
			logger.Debugf("too short to gather metrics: current: %s,terminated: %s", hs.CurrentTime, hs.StartDate)
		} else {
			logger.Debugf("StartDate : %s", hs.StartDate)

			targetRequest, err := client.CloudWatchService.GetTargetGroupRequestStatistics(hs.TargetGroups, hs.StartDate, hs.CurrentTime, logger)
			if err != nil {
				return ret, err
			}

			ret[name] = targetRequest
		}
	}

	return ret, nil
}

// GetLoadBalancerRequestCount retrieves load balancer request count
func GetLoadBalancerRequestCount(hs *HelperStruct, logger *Logger.Logger, client aws.MetricClient, name string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	if len(hs.TargetGroups) > 0 && (hs.StartDate.Sub(constants.MinTimestamp) > 0) && constants.DefaultEnableStats {
		if hs.BaseTimeDuration > constants.MonthToSec {
			hs.StartDate = hs.CurrentTime.Add(-2592000 * time.Second)
		}

		//hs.StartDate = tool.GetBaseStartTime(hs.StartDate)
		if hs.CurrentTime.Sub(hs.StartDate) < 0 {
			logger.Debugf("too short to gather metrics: current: %s,terminated: %s", hs.CurrentTime, hs.StartDate)
		} else {
			logger.Debugf("StartDate : %s", hs.StartDate)

			statistics, err := client.CloudWatchService.GetLoadBalancerRequestStatistics(hs.LoadBalancers, hs.StartDate, hs.CurrentTime, logger)
			if err != nil {
				return ret, err
			}

			ret[name] = statistics
		}
	}

	return ret, nil
}
