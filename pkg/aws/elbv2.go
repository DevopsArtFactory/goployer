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
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	// Return nil if there is no target group.
	if len(targetGroups) == 0 {
		return nil, nil
	}

	tgWithDetails, err := e.DescribeTargetGroups(aws.StringSlice(targetGroups))
	if err != nil {
		return nil, err
	}

	if len(tgWithDetails) == 0 {
		return nil, nil
	}

	var tgs []*string
	for _, group := range tgWithDetails {
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

// CreateTargetGroup creates a new target group
func (e ELBV2Client) CreateTargetGroup(tg *elbv2.TargetGroup, tgName string) (*elbv2.TargetGroup, error) {
	input := &elbv2.CreateTargetGroupInput{
		Name:     aws.String(tgName),
		Port:     tg.Port,
		Protocol: tg.Protocol,
		VpcId:    tg.VpcId,
	}

	result, err := e.Client.CreateTargetGroup(input)
	if err != nil {
		return nil, err
	}

	return result.TargetGroups[0], nil
}

// DescribeTargetGroups returns arn list of target groups with detailed information
func (e ELBV2Client) DescribeTargetGroups(targetGroups []*string) ([]*elbv2.TargetGroup, error) {
	input := &elbv2.DescribeTargetGroupsInput{
		Names: targetGroups,
	}

	result, err := e.Client.DescribeTargetGroups(input)
	if err != nil {
		return nil, err
	}

	return result.TargetGroups, nil
}

// DeleteTargetGroup deletes a target group
func (e ELBV2Client) DeleteTargetGroup(targetGroup *string) error {
	input := &elbv2.DeleteTargetGroupInput{
		TargetGroupArn: targetGroup,
	}

	_, err := e.Client.DeleteTargetGroup(input)
	if err != nil {
		return err
	}

	return nil
}

// DeleteLoadBalancer deletes a load balancer
func (e ELBV2Client) DeleteLoadBalancer(lb string) error {
	input := &elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(lb),
	}

	_, err := e.Client.DeleteLoadBalancer(input)
	if err != nil {
		return err
	}

	return nil
}

// DescribeLoadBalancers retrieves all load balancers
func (e ELBV2Client) DescribeLoadBalancers() ([]*elbv2.LoadBalancer, error) {
	input := &elbv2.DescribeLoadBalancersInput{}

	result, err := e.Client.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	return result.LoadBalancers, nil
}

// DescribeLoadBalancers retrieves matching load balancer
func (e ELBV2Client) GetMatchingLoadBalancer(lb string) (*elbv2.LoadBalancer, error) {
	input := &elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{
			aws.String(lb),
		},
	}

	result, err := e.Client.DescribeLoadBalancers(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == elbv2.ErrCodeLoadBalancerNotFoundException {
				return nil, nil
			}
		}
		return nil, err
	}

	if len(result.LoadBalancers) == 0 {
		return nil, nil
	}

	return result.LoadBalancers[0], nil
}

// CreateLoadBalancer retrieves all load balancers
func (e ELBV2Client) CreateLoadBalancer(app string, subnets []string, groupID *string) (*elbv2.LoadBalancer, error) {
	input := &elbv2.CreateLoadBalancerInput{
		Name: aws.String(app),
		Tags: []*elbv2.Tag{
			{
				Key:   aws.String(constants.DeploymentTagKey),
				Value: aws.String(constants.CanaryDeployment),
			},
		},
		Subnets: aws.StringSlice(subnets),
	}

	if groupID != nil {
		input.SecurityGroups = []*string{groupID}
	}

	result, err := e.Client.CreateLoadBalancer(input)
	if err != nil {
		return nil, err
	}

	return result.LoadBalancers[0], nil
}

// CreateNewListener creates a new listener and attach target group to load balancer
func (e ELBV2Client) CreateNewListener(loadBalancerArn string, targetGroupArn string) error {
	input := &elbv2.CreateListenerInput{
		DefaultActions: []*elbv2.Action{
			{
				TargetGroupArn: aws.String(targetGroupArn),
				Type:           aws.String("forward"),
			},
		},
		LoadBalancerArn: aws.String(loadBalancerArn),
		Port:            aws.Int64(80),
		Protocol:        aws.String("HTTP"),
	}

	_, err := e.Client.CreateListener(input)
	if err != nil {
		return err
	}

	return nil
}

// DescribeListeners describes all listeners in the load balancer
func (e ELBV2Client) DescribeListeners(loadBalancerArn string) ([]*elbv2.Listener, error) {
	input := &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
	}

	result, err := e.Client.DescribeListeners(input)
	if err != nil {
		return nil, err
	}

	return result.Listeners, nil
}

// ModifyListener modifies the existing listener and change target to newly created target group
func (e ELBV2Client) ModifyListener(listenerArn *string, targetGroupArn string) error {
	input := &elbv2.ModifyListenerInput{
		DefaultActions: []*elbv2.Action{
			{
				TargetGroupArn: aws.String(targetGroupArn),
				Type:           aws.String("forward"),
			},
		},
		ListenerArn: listenerArn,
	}

	_, err := e.Client.ModifyListener(input)
	if err != nil {
		return err
	}

	return nil
}
