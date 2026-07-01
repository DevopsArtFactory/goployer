/*
copyright 2026 the Goployer authors

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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

func TestBuildWeightedForwardAction(t *testing.T) {
	action := BuildWeightedForwardAction("stable-tg", "canary-tg", 90, 10)

	if action.Type != elbv2types.ActionTypeEnumForward {
		t.Fatalf("expected forward action, got %s", action.Type)
	}
	if action.ForwardConfig == nil {
		t.Fatal("expected forward config")
	}
	if len(action.ForwardConfig.TargetGroups) != 2 {
		t.Fatalf("expected two target groups, got %d", len(action.ForwardConfig.TargetGroups))
	}

	stable := action.ForwardConfig.TargetGroups[0]
	if *stable.TargetGroupArn != "stable-tg" || *stable.Weight != 90 {
		t.Fatalf("unexpected stable target group: %#v", stable)
	}

	canary := action.ForwardConfig.TargetGroups[1]
	if *canary.TargetGroupArn != "canary-tg" || *canary.Weight != 10 {
		t.Fatalf("unexpected canary target group: %#v", canary)
	}
}

func TestBuildCreateTargetGroupInputCopiesSettings(t *testing.T) {
	tg := &elbv2types.TargetGroup{
		HealthCheckEnabled:         aws.Bool(true),
		HealthCheckIntervalSeconds: aws.Int32(15),
		HealthCheckPath:            aws.String("/healthz"),
		HealthCheckPort:            aws.String("8080"),
		HealthCheckProtocol:        elbv2types.ProtocolEnumHttp,
		HealthCheckTimeoutSeconds:  aws.Int32(5),
		HealthyThresholdCount:      aws.Int32(3),
		IpAddressType:              elbv2types.TargetGroupIpAddressTypeEnumIpv4,
		Matcher: &elbv2types.Matcher{
			HttpCode: aws.String("200-299"),
		},
		Port:                    aws.Int32(8080),
		Protocol:                elbv2types.ProtocolEnumHttp,
		ProtocolVersion:         aws.String("HTTP2"),
		TargetControlPort:       aws.Int32(9000),
		TargetType:              elbv2types.TargetTypeEnumIp,
		UnhealthyThresholdCount: aws.Int32(4),
		VpcId:                   aws.String("vpc-123456"),
	}

	input := BuildCreateTargetGroupInput(tg, "demo-canary")

	if *input.Name != "demo-canary" {
		t.Fatalf("expected target group name demo-canary, got %s", *input.Name)
	}
	if input.HealthCheckEnabled != tg.HealthCheckEnabled {
		t.Fatal("expected health check enabled to be copied")
	}
	if input.HealthCheckIntervalSeconds != tg.HealthCheckIntervalSeconds {
		t.Fatal("expected health check interval to be copied")
	}
	if input.HealthCheckPath != tg.HealthCheckPath {
		t.Fatal("expected health check path to be copied")
	}
	if input.HealthCheckPort != tg.HealthCheckPort {
		t.Fatal("expected health check port to be copied")
	}
	if input.HealthCheckProtocol != tg.HealthCheckProtocol {
		t.Fatal("expected health check protocol to be copied")
	}
	if input.HealthCheckTimeoutSeconds != tg.HealthCheckTimeoutSeconds {
		t.Fatal("expected health check timeout to be copied")
	}
	if input.HealthyThresholdCount != tg.HealthyThresholdCount {
		t.Fatal("expected healthy threshold to be copied")
	}
	if input.IpAddressType != tg.IpAddressType {
		t.Fatal("expected IP address type to be copied")
	}
	if input.Matcher != tg.Matcher {
		t.Fatal("expected matcher to be copied")
	}
	if input.Port != tg.Port {
		t.Fatal("expected port to be copied")
	}
	if input.Protocol != tg.Protocol {
		t.Fatal("expected protocol to be copied")
	}
	if input.ProtocolVersion != tg.ProtocolVersion {
		t.Fatal("expected protocol version to be copied")
	}
	if input.TargetControlPort != tg.TargetControlPort {
		t.Fatal("expected target control port to be copied")
	}
	if input.TargetType != tg.TargetType {
		t.Fatal("expected target type to be copied")
	}
	if input.UnhealthyThresholdCount != tg.UnhealthyThresholdCount {
		t.Fatal("expected unhealthy threshold to be copied")
	}
	if input.VpcId != tg.VpcId {
		t.Fatal("expected VPC ID to be copied")
	}
}
