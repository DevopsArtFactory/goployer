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
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
)

func TestCheckCanaryVersion(t *testing.T) {
	region := constants.DefaultRegion
	regionShard := strings.ReplaceAll(region, "-", "")
	testData := []struct {
		Input    []string
		Expected int
	}{
		{
			Input: []string{
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s/xxxxxx", region, regionShard),
			},
			Expected: 0,
		},
		{
			Input: []string{
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s/xxxxxx", region, regionShard),
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s-canary-v001/xxxxxx", region, regionShard),
			},
			Expected: 1,
		},
		{
			Input: []string{
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s/xxxxxx", region, regionShard),
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s-canary-v001/xxxxxx", region, regionShard),
				fmt.Sprintf("arn:aws:elasticloadbalancing:%s:12345678910:targetgroup/test-dev_%s-canary-v002/xxxxxx", region, regionShard),
			},
			Expected: 2,
		},
	}

	for _, td := range testData {
		if output := CheckCanaryVersion(aws.StringSlice(td.Input), region); output != td.Expected {
			t.Errorf("expected: %d, output: %d", td.Expected, output)
		}
	}
}
