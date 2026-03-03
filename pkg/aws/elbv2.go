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
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type ELBV2Client struct {
	Client *elbv2.Client
}

type HealthcheckHost struct {
	InstanceID     string
	LifecycleState string
	TargetStatus   string
	HealthStatus   string
	Valid          bool
}

func NewELBV2Client(cfg aws.Config) ELBV2Client {
	return ELBV2Client{
		Client: elbv2.NewFromConfig(cfg),
	}
}

// GetTargetGroupARNs returns arn list of target groups
func (e ELBV2Client) GetTargetGroupARNs(targetGroups []string) ([]string, error) {
	if len(targetGroups) == 0 {
		return nil, nil
	}

	tgWithDetails, err := e.DescribeTargetGroups(targetGroups)
	if err != nil {
		return nil, err
	}

	if len(tgWithDetails) == 0 {
		return nil, nil
	}

	var tgs []string
	for _, group := range tgWithDetails {
		tgs = append(tgs, *group.TargetGroupArn)
	}

	return tgs, nil
}

// GetHostInTarget gets host instance
func (e ELBV2Client) GetHostInTarget(group *astypes.AutoScalingGroup, targetGroupArn *string, isUpdate, downSizingUpdate bool) ([]HealthcheckHost, error) {
	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: targetGroupArn,
	}

	result, err := e.Client.DescribeTargetHealth(context.Background(), input)
	if err != nil {
		return nil, err
	}

	ret := []HealthcheckHost{}
	for _, instance := range group.Instances {
		targetState := constants.InitialStatus
		for _, hd := range result.TargetHealthDescriptions {
			if *hd.Target.Id == *instance.InstanceId {
				targetState = string(hd.TargetHealth.State)
				break
			}
		}

		var valid bool
		if isUpdate && downSizingUpdate {
			valid = string(instance.LifecycleState) == constants.InServiceStatus || targetState == "healthy" || *instance.HealthStatus == "Healthy"
		} else {
			valid = string(instance.LifecycleState) == constants.InServiceStatus && targetState == "healthy" && *instance.HealthStatus == "Healthy"
		}

		ret = append(ret, HealthcheckHost{
			InstanceID:     *instance.InstanceId,
			LifecycleState: string(instance.LifecycleState),
			TargetStatus:   targetState,
			HealthStatus:   *instance.HealthStatus,
			Valid:          valid,
		})
	}
	return ret, nil
}

// GetLoadBalancerFromTG returns list of loadbalancer from target groups
func (e ELBV2Client) GetLoadBalancerFromTG(targetGroups []string) ([]string, error) {
	input := &elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: targetGroups,
	}

	result, err := e.Client.DescribeTargetGroups(context.Background(), input)
	if err != nil {
		return nil, err
	}

	var lbs []string
	for _, group := range result.TargetGroups {
		for _, lb := range group.LoadBalancerArns {
			if !tool.IsStringInArray(lb, lbs) {
				lbs = append(lbs, lb)
			}
		}
	}

	return lbs, nil
}

// CreateTargetGroup creates a new target group
func (e ELBV2Client) CreateTargetGroup(tg *elbv2types.TargetGroup, tgName string) (*elbv2types.TargetGroup, error) {
	input := &elbv2.CreateTargetGroupInput{
		Name:     aws.String(tgName),
		Port:     tg.Port,
		Protocol: tg.Protocol,
		VpcId:    tg.VpcId,
	}

	result, err := e.Client.CreateTargetGroup(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return &result.TargetGroups[0], nil
}

// DescribeTargetGroups returns arn list of target groups with detailed information
func (e ELBV2Client) DescribeTargetGroups(targetGroups []string) ([]elbv2types.TargetGroup, error) {
	input := &elbv2.DescribeTargetGroupsInput{
		Names: targetGroups,
	}

	result, err := e.Client.DescribeTargetGroups(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return result.TargetGroups, nil
}

// DeleteTargetGroup deletes a target group
func (e ELBV2Client) DeleteTargetGroup(targetGroup *string) error {
	_, err := e.Client.DeleteTargetGroup(context.Background(), &elbv2.DeleteTargetGroupInput{
		TargetGroupArn: targetGroup,
	})
	return err
}

// DeleteLoadBalancer deletes a load balancer
func (e ELBV2Client) DeleteLoadBalancer(lb string) error {
	_, err := e.Client.DeleteLoadBalancer(context.Background(), &elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(lb),
	})
	return err
}

// DescribeLoadBalancers retrieves all load balancers
func (e ELBV2Client) DescribeLoadBalancers() ([]elbv2types.LoadBalancer, error) {
	result, err := e.Client.DescribeLoadBalancers(context.Background(), &elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, err
	}
	return result.LoadBalancers, nil
}

// GetMatchingLoadBalancer retrieves matching load balancer
func (e ELBV2Client) GetMatchingLoadBalancer(lb string) (*elbv2types.LoadBalancer, error) {
	input := &elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{lb},
	}

	result, err := e.Client.DescribeLoadBalancers(context.Background(), input)
	if err != nil {
		// Check if load balancer not found
		return nil, nil
	}

	if len(result.LoadBalancers) == 0 {
		return nil, nil
	}

	return &result.LoadBalancers[0], nil
}

// CreateLoadBalancer creates a new load balancer
func (e ELBV2Client) CreateLoadBalancer(app string, subnets []string, groupID *string) (*elbv2types.LoadBalancer, error) {
	input := &elbv2.CreateLoadBalancerInput{
		Name: aws.String(app),
		Tags: []elbv2types.Tag{
			{
				Key:   aws.String(constants.DeploymentTagKey),
				Value: aws.String(constants.CanaryDeployment),
			},
		},
		Subnets: subnets,
	}

	if groupID != nil {
		input.SecurityGroups = []string{*groupID}
	}

	result, err := e.Client.CreateLoadBalancer(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return &result.LoadBalancers[0], nil
}

// CreateNewListener creates a new listener and attach target group to load balancer
func (e ELBV2Client) CreateNewListener(loadBalancerArn string, targetGroupArn string) error {
	input := &elbv2.CreateListenerInput{
		DefaultActions: []elbv2types.Action{
			{
				TargetGroupArn: aws.String(targetGroupArn),
				Type:           elbv2types.ActionTypeEnumForward,
			},
		},
		LoadBalancerArn: aws.String(loadBalancerArn),
		Port:            aws.Int32(80),
		Protocol:        elbv2types.ProtocolEnumHttp,
	}

	_, err := e.Client.CreateListener(context.Background(), input)
	return err
}

// DescribeListeners describes all listeners in the load balancer
func (e ELBV2Client) DescribeListeners(loadBalancerArn string) ([]elbv2types.Listener, error) {
	result, err := e.Client.DescribeListeners(context.Background(), &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
	})
	if err != nil {
		return nil, err
	}
	return result.Listeners, nil
}

// ModifyListener modifies the existing listener and change target to newly created target group
func (e ELBV2Client) ModifyListener(listenerArn *string, targetGroupArn string) error {
	input := &elbv2.ModifyListenerInput{
		DefaultActions: []elbv2types.Action{
			{
				TargetGroupArn: aws.String(targetGroupArn),
				Type:           elbv2types.ActionTypeEnumForward,
			},
		},
		ListenerArn: listenerArn,
	}

	_, err := e.Client.ModifyListener(context.Background(), input)
	return err
}
