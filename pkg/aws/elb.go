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
	"github.com/aws/aws-sdk-go/service/elb"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type ELBClient struct {
	Client *elb.ELB
}

func NewELBClient(session client.ConfigProvider, region string, creds *credentials.Credentials) ELBClient {
	return ELBClient{
		Client: getELBClientFn(session, region, creds),
	}
}

func getELBClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *elb.ELB {
	if creds == nil {
		return elb.New(session, &aws.Config{Region: aws.String(region)})
	}
	return elb.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

// GetHostInELB returns instances in ELB
func (e ELBClient) GetHealthyHostInELB(group *autoscaling.Group, elbName string) ([]HealthcheckHost, error) {
	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	result, err := e.Client.DescribeInstanceHealth(input)
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
