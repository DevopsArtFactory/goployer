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

package inspector

import (
	"errors"
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/fatih/color"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

type Inspector struct {
	AWSClient     aws.Client
	StatusSummary StatusSummary
	UpdateFields  UpdateFields
}

type StatusSummary struct {
	Name           string
	Capacity       schemas.Capacity
	UpdateCapacity schemas.Capacity
	CreatedTime    time.Time
	UpdateTime     time.Time
	InstanceType   map[string]int64
	Tags           []string
}

type UpdateFields struct {
	AutoscalingName string
	Capacity        schemas.Capacity
}

// New creates new Inspector
func New(region string) Inspector {
	return Inspector{
		AWSClient: aws.BootstrapServices(region, constants.EmptyString),
	}
}

// SelectStack selects a stack
func (i Inspector) SelectStack(application string) (string, error) {
	asgOptions, err := i.GetStacks(application)
	if err != nil {
		return "", err
	}

	var target string
	if len(asgOptions) == 1 {
		target = asgOptions[0]
	} else {
		prompt := &survey.Select{
			Message: "Choose autoscaling group:",
			Options: asgOptions,
		}
		survey.AskOne(prompt, &target)
	}

	if target == constants.EmptyString {
		return constants.EmptyString, errors.New("you have to choose at least one autoscaling group")
	}

	return target, nil
}

func (i Inspector) GetStacks(application string) ([]string, error) {
	asgGroups := i.AWSClient.EC2Service.GetAllMatchingAutoscalingGroupsWithPrefix(application)
	options := []string{}
	for _, a := range asgGroups {
		options = append(options, *a.AutoScalingGroupName)
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no autoscaling group exists: %s", application)
	}

	return options, nil
}

func (i Inspector) GetStackInformation(asgName string) (*autoscaling.Group, error) {
	asg, err := i.AWSClient.EC2Service.GetMatchingAutoscalingGroup(asgName)
	if err != nil {
		return nil, err
	}

	return asg, nil
}

func (i Inspector) SetStatusSummary(asg *autoscaling.Group) StatusSummary {
	summary := StatusSummary{}
	summary.Name = *asg.AutoScalingGroupName
	summary.Capacity = schemas.Capacity{
		Min:     *asg.MinSize,
		Max:     *asg.MaxSize,
		Desired: *asg.DesiredCapacity,
	}
	summary.CreatedTime = *asg.CreatedTime

	instanceTypeInfo := map[string]int64{}
	for _, i := range asg.Instances {
		c, ok := instanceTypeInfo[*i.InstanceType]
		if !ok {
			instanceTypeInfo[*i.InstanceType] = 1
		} else {
			instanceTypeInfo[*i.InstanceType] = c + 1
		}
	}
	summary.InstanceType = instanceTypeInfo

	tags := []string{}
	for _, t := range asg.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", *t.Key, *t.Value))
	}
	summary.Tags = tags
	return summary
}

func (i Inspector) SetUpdateSummary(asg *autoscaling.Group) StatusSummary {
	summary := StatusSummary{}
	summary.Name = *asg.AutoScalingGroupName
	summary.Capacity = schemas.Capacity{
		Min:     *asg.MinSize,
		Max:     *asg.MaxSize,
		Desired: *asg.DesiredCapacity,
	}
	summary.CreatedTime = *asg.CreatedTime

	instanceTypeInfo := map[string]int64{}
	for _, i := range asg.Instances {
		c, ok := instanceTypeInfo[*i.InstanceType]
		if !ok {
			instanceTypeInfo[*i.InstanceType] = 1
		} else {
			instanceTypeInfo[*i.InstanceType] = c + 1
		}
	}
	summary.InstanceType = instanceTypeInfo

	tags := []string{}
	for _, t := range asg.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", *t.Key, *t.Value))
	}
	summary.Tags = tags
	return summary
}

func (i Inspector) Print() error {
	color.Blue("Name: %s", i.StatusSummary.Name)
	color.Black("Min/Desired/Max: %d/%d/%d", i.StatusSummary.Capacity.Min, i.StatusSummary.Capacity.Desired, i.StatusSummary.Capacity.Max)
	color.Black("Created: %s", i.StatusSummary.CreatedTime)

	if len(i.StatusSummary.InstanceType) > 0 {
		color.Black("Instance Statistics: ")
		for k, v := range i.StatusSummary.InstanceType {
			color.Black(" - type:%s, count:%d", k, v)
		}
	}

	if len(i.StatusSummary.Tags) > 0 {
		color.Black("Tags: ")
		for _, t := range i.StatusSummary.Tags {
			color.Black(" - %s", t)
		}
	}

	fmt.Println()

	return nil
}

func (i Inspector) Update() error {
	if err := i.AWSClient.EC2Service.UpdateAutoScalingGroup(i.UpdateFields.AutoscalingName, i.UpdateFields.Capacity); err != nil {
		return err
	}
	return nil
}

func (i Inspector) GenerateStack(region string, group *autoscaling.Group) schemas.Stack {
	s := schemas.Stack{
		Stack:    "update-stack",
		Capacity: i.UpdateFields.Capacity,
		Regions: []schemas.RegionConfig{
			{
				Region: region,
			},
		},
	}

	if len(group.TargetGroupARNs) > 0 {
		s.Regions[0].HealthcheckTargetGroup = *(group.TargetGroupARNs[0])
	}

	if len(group.LoadBalancerNames) > 0 {
		s.Regions[0].HealthcheckLB = *(group.LoadBalancerNames[0])
	}

	return s
}
