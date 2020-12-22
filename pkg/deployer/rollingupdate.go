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

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/helper"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type RollingUpdate struct {
	PrevTargetGroups            map[string][]string
	TargetGroups                map[string][]*string
	PrevHealthCheckTargetGroups map[string]string
	LoadBalancer                map[string]string
	LBSecurityGroup             map[string]*string
	*Deployer
}

// NewRollingUpdate creates new rolling-update deployment deployer
func NewRollingUpdate(h *helper.DeployerHelper) *RollingUpdate {
	var awsClients []aws.Client
	for _, region := range h.Stack.Regions {
		if len(h.Region) > 0 && h.Region != region.Region {
			h.Logger.Debugf("skip creating aws clients in %s region", region.Region)
			continue
		}
		awsClients = append(awsClients, aws.BootstrapServices(region.Region, h.Stack.AssumeRole))
	}

	d := InitDeploymentConfiguration(h, awsClients)

	return &RollingUpdate{
		PrevHealthCheckTargetGroups: map[string]string{},
		PrevTargetGroups:            map[string][]string{},
		TargetGroups:                map[string][]*string{},
		LoadBalancer:                map[string]string{},
		LBSecurityGroup:             map[string]*string{},
		Deployer:                    &d,
	}
}

// GetDeployer returns canary deployer
func (r *RollingUpdate) GetDeployer() *Deployer {
	return r.Deployer
}

// CheckPreviousResources checks if there is any previous version of autoscaling group
func (r *RollingUpdate) CheckPreviousResources(config schemas.Config) error {
	err := r.Deployer.CheckPrevious(config)
	if err != nil {
		return err
	}

	return nil
}

// Deploy runs deployments with canary approach
func (r *RollingUpdate) Deploy(config schemas.Config) error {
	if !r.StepStatus[constants.StepCheckPrevious] {
		return nil
	}
	r.Logger.Infof("Deploy Mode is %s", r.Mode)

	//Get LocalFileProvider
	r.LocalProvider = builder.SetUserdataProvider(r.Stack.Userdata, r.AwsConfig.Userdata)
	for _, region := range r.Stack.Regions {
		if config.Region != "" && config.Region != region.Region {
			r.Logger.Debugf("This region is skipped by user : %s", region.Region)
			continue
		}

		err := r.Deployer.Deploy(config, region)
		if err != nil {
			return err
		}
	}

	r.StepStatus[constants.StepDeploy] = true
	return nil
}

// HealthChecking does health checking for canary deployment
func (r *RollingUpdate) HealthChecking(config schemas.Config) error {
	healthy := false

	for !healthy {
		r.Logger.Debugf("Start Timestamp: %d, timeout: %s", config.StartTimestamp, config.Timeout)
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err := r.Deployer.HealthChecking(config)
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
func (r *RollingUpdate) FinishAdditionalWork(config schemas.Config) error {
	if !r.StepStatus[constants.StepDeploy] {
		return nil
	}

	skipped := false
	if len(config.Region) > 0 && !CheckRegionExist(config.Region, r.Stack.Regions) {
		skipped = true
	}

	if !skipped {
		for _, region := range r.Stack.Regions {
			if config.Region != "" && config.Region != region.Region {
				r.Logger.Debugf("This region is skipped by user : %s", region.Region)
				continue
			}

			if err := r.CompleteRollingUpdate(config, region); err != nil {
				return err
			}
		}

		if err := r.DoCommonAdditionalWork(config); err != nil {
			return err
		}
	}

	r.Logger.Debugln("Finish additional works")
	r.StepStatus[constants.StepAdditionalWork] = true
	return nil
}

// TriggerLifecycleCallbacks runs lifecycle callbacks before cleaning.
func (r *RollingUpdate) TriggerLifecycleCallbacks(config schemas.Config) error {
	if !r.StepStatus[constants.StepAdditionalWork] {
		return nil
	}
	if config.CompleteCanary {
		r.StepStatus[constants.StepTriggerLifecycleCallback] = true
		return nil
	}
	return r.Deployer.TriggerLifecycleCallbacks(config)
}

// CleanPreviousVersion cleans previous version of autoscaling group or canary target group
func (r *RollingUpdate) CleanPreviousVersion(config schemas.Config) error {
	if !r.StepStatus[constants.StepTriggerLifecycleCallback] {
		return nil
	}
	r.Logger.Debugf("Skip CleanPreviousVersion because instance(s) is(are) already deleted: %s", r.Mode)

	r.StepStatus[constants.StepCleanPreviousVersion] = true
	return nil
}

// GatherMetrics gathers the whole metrics from deployer
func (r *RollingUpdate) GatherMetrics(config schemas.Config) error {
	if !r.StepStatus[constants.StepCleanChecking] {
		return nil
	}
	if config.DisableMetrics {
		return nil
	}

	if len(config.Region) > 0 {
		if !CheckRegionExist(config.Region, r.Stack.Regions) {
			return nil
		}
	}

	if !config.CompleteCanary {
		r.Logger.Debug("Skip gathering metrics because canary is now applied")
		return nil
	}

	if err := r.Deployer.StartGatheringMetrics(config); err != nil {
		return err
	}

	r.StepStatus[constants.StepGatherMetrics] = true
	return nil
}

// RunAPITest tries to run API Test
func (r *RollingUpdate) RunAPITest(config schemas.Config) error {
	if !r.StepStatus[constants.StepGatherMetrics] {
		return nil
	}

	if !config.CompleteCanary {
		r.Logger.Debug("Skip API test because canary is now applied")
		return nil
	}

	err := r.Deployer.RunAPITest(config)
	if err != nil {
		return err
	}

	r.StepStatus[constants.StepRunAPI] = true
	return nil
}

// CleanChecking checks Termination status
func (r *RollingUpdate) CleanChecking(config schemas.Config) error {
	if !r.StepStatus[constants.StepCleanPreviousVersion] {
		return nil
	}
	done := false

	for !done {
		isTimeout, _ := tool.CheckTimeout(config.StartTimestamp, config.Timeout)
		if isTimeout {
			return fmt.Errorf("timeout has been exceeded : %.0f minutes", config.Timeout.Minutes())
		}

		isDone, err := r.Deployer.CleanChecking(config)
		if err != nil {
			return errors.New("error happened while health checking")
		}

		if isDone {
			done = true
		} else {
			r.Logger.Info("All stacks are not ready to be terminated... Please waiting...")
			time.Sleep(config.PollingInterval)
		}
	}

	r.StepStatus[constants.StepCleanChecking] = true
	return nil
}

// CompleteRollingUpdate processes the whole process of rolling update
func (r *RollingUpdate) CompleteRollingUpdate(config schemas.Config, region schemas.RegionConfig) error {
	latestASG := r.LatestAsg[region.Region]
	asgDetail, err := r.Deployer.DescribeAutoScalingGroup(latestASG, region.Region)
	if err != nil {
		return err
	}

	if asgDetail == nil {
		return fmt.Errorf("no autoscaling group information retrieved. Please check autoscaling group resource: %s", latestASG)
	}

	appliedCapacity, err := r.Deployer.DecideCapacity(config.ForceManifestCapacity, false, region.Region)
	if err != nil {
		return err
	}

	targetCapacity := r.Deployer.CompareWithCurrentCapacity(config.ForceManifestCapacity, region.Region)

	previousFinished := false
	for !IsFinishedRollingUpdate(appliedCapacity, targetCapacity) || !previousFinished {
		if !previousFinished {
			previousFinished, err = r.Deployer.ReducePreviousAutoScalingGroupCapacity(region.Region, 1)
			if err != nil {
				return err
			}
		}

		if err := RetrieveNextCapacity(&appliedCapacity, targetCapacity); err != nil {
			return err
		}

		r.Logger.Debugf("Rolling update of autoscaling group: min - %d, desired - %d, max - %d", appliedCapacity.Min, appliedCapacity.Desired, appliedCapacity.Max)
		if err := r.Deployer.ResizingAutoScalingGroup(r.AsgNames[region.Region], region.Region, appliedCapacity); err != nil {
			return err
		}

		// settings for health checking
		r.AppliedCapacity = &appliedCapacity

		if err := r.HealthChecking(config); err != nil {
			return err
		}
	}
	return nil
}

// RetrieveNextCapacity add one capacity at a time
func RetrieveNextCapacity(capacity *schemas.Capacity, targetCapacity schemas.Capacity) error {
	if targetCapacity.Min > capacity.Min {
		capacity.Min++
	}

	if targetCapacity.Desired > capacity.Desired {
		capacity.Desired++
	}

	if targetCapacity.Max > capacity.Max {
		capacity.Max++
	}
	return nil
}

// IsFinishedRollingUpdate checks if rolling update is done or not
func IsFinishedRollingUpdate(current schemas.Capacity, target schemas.Capacity) bool {
	return current.Min == target.Min && current.Desired == target.Desired && current.Max == target.Max
}
