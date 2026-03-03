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
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type SSMClient struct {
	Client *ssm.Client
}

func NewSSMClient(cfg aws.Config) SSMClient {
	return SSMClient{
		Client: ssm.NewFromConfig(cfg),
	}
}

// SendCommand sends SSM command
func (s SSMClient) SendCommand(target []string, commands []string) bool {
	input := &ssm.SendCommandInput{
		DocumentName:   aws.String("AWS-RunShellScript"),
		TimeoutSeconds: aws.Int32(3600),
		InstanceIds:    target,
		Comment:        aws.String("goployer lifecycle callbacks"),
		Parameters: map[string][]string{
			"commands": commands,
		},
	}

	if _, err := s.Client.SendCommand(context.Background(), input); err != nil {
		return false
	}

	return true
}
