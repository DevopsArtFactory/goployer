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
	"os"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"github.com/DevopsArtFactory/goployer/pkg/aws"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

const templ = `{{decorate "bold" "Name"}}:	{{ .Summary.Name }}
{{decorate "bold" "Created Time"}}:	{{ .Summary.CreatedTime }}

{{decorate "capacity" ""}}{{decorate "underline bold" "Capacity"}}
MINIMUM 	DESIRED 	MAXIMUM
{{ .Summary.Capacity.Min }}	{{ .Summary.Capacity.Desired }}	{{ .Summary.Capacity.Max }}

{{decorate "instance_statistics" ""}}{{decorate "underline bold" "Instance Statistics"}}

{{- if eq (len .Summary.InstanceType) 0 }}
 No instance exists
{{- else }}
{{- range $k, $v := .Summary.InstanceType }}
 {{decorate "bullet" $k }}: {{ $v }}
{{- end }}
{{- end }}

{{decorate "tags" ""}}{{decorate "underline bold" "Tags"}}

{{- if eq (len .Summary.Tags) 0 }}
 No tag
{{- else }}
{{- range $result := .Summary.Tags }}
 {{decorate "bullet" $result }}
{{- end }}
{{- end }}

`

type Inspector struct {
	AWSClient     aws.Client
	StatusSummary StatusSummary
	UpdateFields  UpdateFields
}

type StatusSummary struct {
	Name         string
	Capacity     schemas.Capacity
	CreatedTime  time.Time
	InstanceType map[string]int64
	Tags         []string
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

// Print prints the current status of deployment
func (i Inspector) Print() error {
	var data = struct {
		Summary StatusSummary
	}{
		Summary: i.StatusSummary,
	}

	funcMap := template.FuncMap{
		"decorate": tool.DecorateAttr,
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe status of deployment").Funcs(funcMap).Parse(templ))

	err := t.Execute(w, data)
	if err != nil {
		return err
	}
	return w.Flush()
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
