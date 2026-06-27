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
