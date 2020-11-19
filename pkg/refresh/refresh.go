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

package refresh

import (
	"errors"
	"html/template"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/templates"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Refresher struct {
	AWSClient   aws.Client
	TargetGroup *autoscaling.Group
	RefreshID   *string
	Info        *autoscaling.InstanceRefresh
}

// New creates new Refresher
func New(region string) Refresher {
	return Refresher{
		AWSClient: aws.BootstrapServices(region, constants.EmptyString),
	}
}

// SetTarget sets target autoscaling group
func (r *Refresher) SetTarget(group *autoscaling.Group) {
	r.TargetGroup = group
}

// StartRefresh starts instance refresh
func (r *Refresher) StartRefresh(instanceWarmup, minHealthyPercentage int64) error {
	logrus.Debugf("Start to trigger instance refresh")
	logrus.Debugf("Instance Warmup time: %ds", instanceWarmup)
	logrus.Debugf("Minimum healthy instance percentage: %d%", minHealthyPercentage)
	id, err := r.AWSClient.EC2Service.StartInstanceRefresh(r.TargetGroup.AutoScalingGroupName, instanceWarmup, minHealthyPercentage)
	if err != nil {
		return err
	}

	r.RefreshID = id
	logrus.Debugf("Instance Refresh is initiated: %s", *id)

	return nil
}

// DescribeRefreshStatus retrieves status information of instance refresh
func (r *Refresher) DescribeRefreshStatus() error {
	info, err := r.AWSClient.EC2Service.DescribeInstanceRefreshes(r.TargetGroup.AutoScalingGroupName, r.RefreshID)
	if err != nil {
		return err
	}

	r.Info = info
	r.RefreshID = info.InstanceRefreshId
	logrus.Debugf("Current status: %s / %s", *info.InstanceRefreshId, *info.Status)

	return nil
}

// StatusCheck starts status check
func (r *Refresher) StatusCheck(pollingInterval, timeout time.Duration) error {
	logrus.Debug("Start to check instance refresh status")
	startTime := time.Now()
	logrus.Debugf("Current time: %s / Timeout: %s", startTime, timeout)
	logrus.Debugf("Polling interval: %s", pollingInterval)
	for {
		isTimeout, _ := tool.CheckTimeout(startTime.Unix(), timeout)
		if isTimeout {
			return errors.New("timeout limit exceeded")
		}

		if err := r.DescribeRefreshStatus(); err != nil {
			return err
		}

		if tool.IsStringInArray(*r.Info.Status, []string{"Successful", "Cancelled", "Failed"}) {
			logrus.Debugf("Instance refresh is finished because the status is %s", *r.Info.Status)
			break
		}

		time.Sleep(pollingInterval)
	}

	return nil
}

// PrintResult prints result of refresh work
func (r *Refresher) PrintResult() error {
	var data = struct {
		Target  autoscaling.Group
		Summary autoscaling.InstanceRefresh
	}{
		Target:  *r.TargetGroup,
		Summary: *r.Info,
	}

	funcMap := template.FuncMap{
		"decorate": tool.DecorateAttr,
	}

	t := template.Must(template.New("Instance Refresh Result").Funcs(funcMap).Parse(templates.InstanceRefreshStatusTemplate))

	return tool.PrintTemplate(data, t)
}
