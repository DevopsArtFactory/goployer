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

package deployer

import (
	"errors"
	"fmt"
	"time"

	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/helper"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type DeployOnly struct {
	*Deployer
}

// NewDeployOnly creates new DeployOnly deployment deployer
func NewDeployOnly(h *helper.DeployerHelper) *DeployOnly {
	awsClients := []aws.Client{}
	for _, region := range h.Stack.Regions {
		if len(h.Region) > 0 && h.Region != region.Region {
			Logger.Debugf("skip creating aws clients in %s region", region.Region)
			continue
		}
		awsClients = append(awsClients, aws.BootstrapServices(region.Region, h.Stack.AssumeRole))
	}

	d := InitDeploymentConfiguration(h, awsClients)

	return &DeployOnly{
		Deployer: &d,
	}
}

// GetDeployer returns Deployer struct
func (d *DeployOnly) GetDeployer() *Deployer {
	return d.Deployer
}

// CheckPreviousResources checks if there is any previous version of autoscaling group
func (d *DeployOnly) CheckPreviousResources(config schemas.Config) error {
	err := d.Deployer.CheckPrevious(config)
	if err != nil {
		return err
	}

	return nil
}

// Deploy function
func (d *DeployOnly) Deploy(config schemas.Config) error {
	if !d.StepStatus[constants.StepCheckPrevious] {
		return nil
	}

	d.Logger.Info("Deploy Mode is " + d.Mode)

	//Get LocalFileProvider
	d.LocalProvider = builder.SetUserdataProvider(d.Stack.Userdata, d.AwsConfig.Userdata)

	for _, region := range d.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			d.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		err := d.Deployer.Deploy(config, region)
		if err != nil {
			return err
		}
	}

	d.StepStatus[constants.StepDeploy] = true
	return nil
}

// HealthChecking does health checking for d.ue-green deployment
func (d *DeployOnly) HealthChecking(_ schemas.Config) error {
	// No health checking is needed
	Logger.Info("Skip health check because this is DeployOnly mode")

	// sleep 30 seconds for a new instance to be ready
	time.Sleep(30 * time.Second)

	return nil
}

// FinishAdditionalWork processes additional work for the new deployment
func (d *DeployOnly) FinishAdditionalWork(config schemas.Config) error {
	if !d.StepStatus[constants.StepDeploy] {
		return nil
	}

	skipped := false
	if len(config.Region) > 0 && !CheckRegionExist(config.Region, d.Stack.Regions) {
		skipped = true
	}

	if !skipped {
		if err := d.DoCommonAdditionalWork(config); err != nil {
			return err
		}
	}

	d.Logger.Debug("Finish additional works.")
	d.StepStatus[constants.StepAdditionalWork] = true
	return nil
}

// TriggerLifecycleCallbacks.cks runs lifecycle callbacks d.fore cleaning.
func (d *DeployOnly) TriggerLifecycleCallbacks(config schemas.Config) error {
	if !d.StepStatus[constants.StepAdditionalWork] {
		return nil
	}
	return d.Deployer.TriggerLifecycleCallbacks(config)
}

//CleanPreviousVersion cleans previous version of autoscaling group
func (d *DeployOnly) CleanPreviousVersion(config schemas.Config) error {
	if !d.StepStatus[constants.StepTriggerLifecycleCallback] {
		return nil
	}
	d.Logger.Debug("Delete Mode is " + d.Mode)

	skipped := false
	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, d.Stack.Regions) {
			skipped = true
		}
	}

	if !skipped {
		if err := d.Deployer.CleanPreviousAutoScalingGroup(config); err != nil {
			return err
		}
	}
	d.StepStatus[constants.StepCleanPreviousVersion] = true
	return nil
}

// CleanChecking checks Termination status
func (d *DeployOnly) CleanChecking(config schemas.Config) error {
	if !d.StepStatus[constants.StepCleanPreviousVersion] {
		return nil
	}
	done := false

	for !done {
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err := d.Deployer.CleanChecking(config)
		if err != nil {
			return errors.New("error happened while health checking")
		}

		if isDone {
			done = true
		} else {
			d.Logger.Info("All stacks are not ready to be terminated... Please waiting...")
			time.Sleep(config.PollingInterval)
		}
	}

	d.StepStatus[constants.StepCleanChecking] = true
	return nil
}

// GatherMetrics gathers the whole metrics from deployer
func (d *DeployOnly) GatherMetrics(config schemas.Config) error {
	if config.DisableMetrics {
		return nil
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, d.Stack.Regions) {
			return nil
		}
	}

	if err := d.Deployer.StartGatheringMetrics(config); err != nil {
		return err
	}

	return nil
}

// RunAPITest tries to run API Test
func (d *DeployOnly) RunAPITest(config schemas.Config) error {
	return d.Deployer.RunAPITest(config)
}
