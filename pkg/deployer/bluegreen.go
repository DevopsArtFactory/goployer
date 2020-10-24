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

type BlueGreen struct {
	*Deployer
}

// NewBlueGreen creates new BlueGreen deployment deployer
func NewBlueGreen(h *helper.DeployerHelper) *BlueGreen {
	awsClients := []aws.Client{}
	for _, region := range h.Stack.Regions {
		if len(h.Region) > 0 && h.Region != region.Region {
			Logger.Debugf("skip creating aws clients in %s region", region.Region)
			continue
		}
		awsClients = append(awsClients, aws.BootstrapServices(region.Region, h.Stack.AssumeRole))
	}

	d := Deployer{
		Mode:              h.Stack.ReplacementType,
		Logger:            h.Logger,
		AwsConfig:         h.AwsConfig,
		AWSClients:        awsClients,
		APITestTemplate:   h.APITestTemplates,
		AsgNames:          map[string]string{},
		PrevAsgs:          map[string][]string{},
		PrevInstances:     map[string][]string{},
		PrevInstanceCount: map[string]schemas.Capacity{},
		PrevVersions:      map[string][]int{},
		SecurityGroup:     map[string]*string{},
		CanaryFlag:        map[string]bool{},
		LatestAsg:         map[string]string{},
		Stack:             h.Stack,
		Slack:             h.Slack,
		Collector:         h.Collector,
		StepStatus:        helper.InitStartStatus(),
	}
	return &BlueGreen{
		Deployer: &d,
	}
}

// GetDeployer returns Deployer struct
func (b *BlueGreen) GetDeployer() *Deployer {
	return b.Deployer
}

// CheckPreviousResources checks if there is any previous version of autoscaling group
func (b *BlueGreen) CheckPreviousResources(config schemas.Config) error {
	err := b.Deployer.CheckPrevious(config)
	if err != nil {
		return err
	}

	return nil
}

// Deploy function
func (b *BlueGreen) Deploy(config schemas.Config) error {
	if !b.StepStatus[constants.StepCheckPrevious] {
		return nil
	}

	b.Logger.Info("Deploy Mode is " + b.Mode)

	//Get LocalFileProvider
	b.LocalProvider = builder.SetUserdataProvider(b.Stack.Userdata, b.AwsConfig.Userdata)

	for _, region := range b.Stack.Regions {
		//Region check
		//If region id is passed from command line, then deployer will deploy in that region only.
		if config.Region != "" && config.Region != region.Region {
			b.Logger.Debug("This region is skipped by user : " + region.Region)
			continue
		}

		err := b.Deployer.Deploy(config, region)
		if err != nil {
			return err
		}
	}

	b.StepStatus[constants.StepDeploy] = true
	return nil
}

// HealthChecking does health checking for blue-green deployment
func (b *BlueGreen) HealthChecking(config schemas.Config) error {
	healthy := false

	for !healthy {
		b.Logger.Debugf("Start Timestamp: %d, timeout: %s", config.StartTimestamp, config.Timeout)
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err := b.Deployer.HealthChecking(config)
		if err != nil {
			return errors.New("error happened while health checking")
		}

		if isDone {
			healthy = true
		} else {
			time.Sleep(config.PollingInterval)
		}
	}

	return nil
}

// FinishAdditionalWork processes additional work for the new deployment
func (b *BlueGreen) FinishAdditionalWork(config schemas.Config) error {
	if !b.StepStatus[constants.StepDeploy] {
		return nil
	}

	skipped := false
	if len(config.Region) > 0 && !CheckRegionExist(config.Region, b.Stack.Regions) {
		skipped = true
	}

	if !skipped {
		if err := b.DoCommonAdditionalWork(config); err != nil {
			return err
		}
	}

	b.Logger.Debug("Finish additional works.")
	b.StepStatus[constants.StepAdditionalWork] = true
	return nil
}

// TriggerLifecycleCallbacks runs lifecycle callbacks before cleaning.
func (b *BlueGreen) TriggerLifecycleCallbacks(config schemas.Config) error {
	if !b.StepStatus[constants.StepAdditionalWork] {
		return nil
	}
	return b.Deployer.TriggerLifecycleCallbacks(config)
}

//CleanPreviousVersion cleans previous version of autoscaling group
func (b *BlueGreen) CleanPreviousVersion(config schemas.Config) error {
	if !b.StepStatus[constants.StepTriggerLifecycleCallback] {
		return nil
	}
	b.Logger.Debug("Delete Mode is " + b.Mode)

	skipped := false
	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			skipped = true
		}
	}

	if !skipped {
		if err := b.Deployer.CleanPreviousAutoScalingGroup(config); err != nil {
			return err
		}
	}
	b.StepStatus[constants.StepCleanPreviousVersion] = true
	return nil
}

// CleanChecking checks Termination status
func (b *BlueGreen) CleanChecking(config schemas.Config) error {
	if !b.StepStatus[constants.StepCleanPreviousVersion] {
		return nil
	}
	done := false

	for !done {
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err := b.Deployer.CleanChecking(config)
		if err != nil {
			return errors.New("error happened while health checking")
		}

		if isDone {
			done = true
		} else {
			b.Logger.Info("All stacks are not ready to be terminated... Please waiting...")
			time.Sleep(config.PollingInterval)
		}
	}

	b.StepStatus[constants.StepCleanChecking] = true
	return nil
}

// CheckRegionExist checks if target region is really in regions described in manifest file
func CheckRegionExist(target string, regions []schemas.RegionConfig) bool {
	regionExists := false
	for _, region := range regions {
		if region.Region == target {
			regionExists = true
			break
		}
	}

	return regionExists
}

// GatherMetrics gathers the whole metrics from deployer
func (b *BlueGreen) GatherMetrics(config schemas.Config) error {
	if config.DisableMetrics {
		return nil
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, b.Stack.Regions) {
			return nil
		}
	}

	if err := b.Deployer.StartGatheringMetrics(config); err != nil {
		return err
	}

	return nil
}

// RunAPITest tries to run API Test
func (b *BlueGreen) RunAPITest(config schemas.Config) error {
	return b.Deployer.RunAPITest(config)
}
