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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elbv2"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type ELBV2Client struct {
	Client *elbv2.ELBV2
}

type HealthcheckHost struct {
	InstanceID     string
	LifecycleState string
	TargetStatus   string
	HealthStatus   string
	Valid          bool
}

func NewELBV2Client(session client.ConfigProvider, region string, creds *credentials.Credentials) ELBV2Client {
	return ELBV2Client{
		Client: getElbClientFn(session, region, creds),
	}
}

func getElbClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *elbv2.ELBV2 {
	if creds == nil {
		return elbv2.New(session, &aws.Config{Region: aws.String(region)})
	}
	return elbv2.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

// GetTargetGroupARNs returns arn list of target groups
func (e ELBV2Client) GetTargetGroupARNs(targetGroups []string) ([]*string, error) {
	if len(targetGroups) == 0 {
		return nil, nil
	}

	input := &elbv2.DescribeTargetGroupsInput{
		Names: aws.StringSlice(targetGroups),
	}

	result, err := e.Client.DescribeTargetGroups(input)
	if err != nil {
		return nil, err
	}

	tgs := []*string{}
	for _, group := range result.TargetGroups {
		tgs = append(tgs, group.TargetGroupArn)
	}

	return tgs, nil
}

// GetHostInTarget gets host instance
func (e ELBV2Client) GetHostInTarget(group *autoscaling.Group, targetGroupArn *string, isUpdate, downSizingUpdate bool) ([]HealthcheckHost, error) {
	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(*targetGroupArn),
	}

	result, err := e.Client.DescribeTargetHealth(input)
	if err != nil {
		return nil, err
	}

	ret := []HealthcheckHost{}
	for _, instance := range group.Instances {
		targetState := constants.InitialStatus
		for _, hd := range result.TargetHealthDescriptions {
			if *hd.Target.Id == *instance.InstanceId {
				targetState = *hd.TargetHealth.State
				break
			}
		}

		var valid bool
		if isUpdate && downSizingUpdate {
			valid = *instance.LifecycleState == constants.InServiceStatus || targetState == "healthy" || *instance.HealthStatus == "Healthy"
		} else {
			valid = *instance.LifecycleState == constants.InServiceStatus && targetState == "healthy" && *instance.HealthStatus == "Healthy"
		}

		ret = append(ret, HealthcheckHost{
			InstanceID:     *instance.InstanceId,
			LifecycleState: *instance.LifecycleState,
			TargetStatus:   targetState,
			HealthStatus:   *instance.HealthStatus,
			Valid:          valid,
		})
	}
	return ret, nil
}

// GetLoadBalancerFromTG returns list of loadbalancer from target groups
func (e ELBV2Client) GetLoadBalancerFromTG(targetGroups []*string) ([]*string, error) {
	input := &elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: targetGroups,
	}

	result, err := e.Client.DescribeTargetGroups(input)
	if err != nil {
		return nil, err
	}

	lbs := []string{}
	for _, group := range result.TargetGroups {
		for _, lb := range group.LoadBalancerArns {
			if !tool.IsStringInArray(*lb, lbs) {
				lbs = append(lbs, *lb)
			}
		}
	}

	return aws.StringSlice(lbs), nil
}
