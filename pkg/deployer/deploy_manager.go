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
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

type DeployManager interface {
	GetStackName() string
	Deploy(config schemas.Config) error
	CheckPrevious(config schemas.Config) error
	HealthChecking(config schemas.Config) map[string]bool
	FinishAdditionalWork(config schemas.Config) error
	CleanPreviousVersion(config schemas.Config) error
	TriggerLifecycleCallbacks(config schemas.Config) error
	TerminateChecking(config schemas.Config) map[string]bool
	GatherMetrics(config schemas.Config) error
	RunAPITest(config schemas.Config) error
	SkipDeployStep()
}
