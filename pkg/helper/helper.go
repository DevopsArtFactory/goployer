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

package helper

import (
	"github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/collector"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/slack"
)

// DeployerHelper is a struct for passing parameters when creating new deployer
type DeployerHelper struct {
	Logger           *logrus.Logger
	Stack            schemas.Stack
	AwsConfig        schemas.AWSConfig
	APITestTemplates *schemas.APITestTemplate
	Region           string
	Slack            slack.Slack
	Collector        collector.Collector
}

// InitStartStatus set start status for deployment
func InitStartStatus() map[int64]bool {
	return map[int64]bool{
		constants.StepCheckPrevious:            false,
		constants.StepDeploy:                   false,
		constants.StepAdditionalWork:           false,
		constants.StepTriggerLifecycleCallback: false,
		constants.StepCleanPreviousVersion:     false,
		constants.StepCleanChecking:            false,
		constants.StepGatherMetrics:            false,
		constants.StepRunAPI:                   false,
	}
}
