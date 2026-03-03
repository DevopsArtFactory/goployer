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
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	astypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type ELBClient struct {
	Client *elb.Client
}

func NewELBClient(cfg aws.Config) ELBClient {
	return ELBClient{
		Client: elb.NewFromConfig(cfg),
	}
}

// GetHealthyHostInELB returns instances in ELB
func (e ELBClient) GetHealthyHostInELB(group *astypes.AutoScalingGroup, elbName string) ([]HealthcheckHost, error) {
	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	result, err := e.Client.DescribeInstanceHealth(context.Background(), input)
	if err != nil {
		return nil, err
	}

	ret := []HealthcheckHost{}
	targetInstances := []string{}
	for _, instance := range group.Instances {
		targetInstances = append(targetInstances, *instance.InstanceId)
	}

	for _, instance := range result.InstanceStates {
		valid := *instance.State == constants.InServiceStatus
		if tool.IsStringInArray(*instance.InstanceId, targetInstances) {
			ret = append(ret, HealthcheckHost{
				InstanceID:     *instance.InstanceId,
				LifecycleState: *instance.State,
				Valid:          valid,
			})
		}
	}

	return ret, nil
}
