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
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/sirupsen/logrus"
)

type SSMClient struct {
	Client *ssm.SSM
}

func NewSSMClient(session client.ConfigProvider, region string, creds *credentials.Credentials) SSMClient {
	return SSMClient{
		Client: getSsmClientFn(session, region, creds),
	}
}

func getSsmClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *ssm.SSM {
	if creds == nil {
		return ssm.New(session, &aws.Config{Region: aws.String(region)})
	}
	return ssm.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

//SSM Send command
func (s SSMClient) SendCommand(target []*string, commands []*string) bool {
	input := &ssm.SendCommandInput{
		DocumentName:   aws.String("AWS-RunShellScript"),
		TimeoutSeconds: aws.Int64(3600),
		InstanceIds:    target,
		Comment:        aws.String("goployer lifecycle callbacks"),
		Parameters: map[string][]*string{
			"commands": commands,
		},
	}

	_, err := s.Client.SendCommand(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ssm.ErrCodeDuplicateInstanceId:
				logrus.Errorln(ssm.ErrCodeDuplicateInstanceId, aerr.Error())
			case ssm.ErrCodeInternalServerError:
				logrus.Errorln(ssm.ErrCodeInternalServerError, aerr.Error())
			case ssm.ErrCodeInvalidInstanceId:
				logrus.Errorln(ssm.ErrCodeInvalidInstanceId, aerr.Error())
			case ssm.ErrCodeInvalidParameters:
				logrus.Errorln(ssm.ErrCodeInvalidParameters, aerr.Error())
			default:
				logrus.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			logrus.Errorln(err.Error())
		}
		return false
	}

	return true
}
