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
	"github.com/DevopsArtFactory/goployer/pkg/builder"
)

type DeployManager interface {
	GetStackName() string
	Deploy(config builder.Config) error
	CheckPrevious(config builder.Config) error
	HealthChecking(config builder.Config) map[string]bool
	FinishAdditionalWork(config builder.Config) error
	CleanPreviousVersion(config builder.Config) error
	TriggerLifecycleCallbacks(config builder.Config) error
	TerminateChecking(config builder.Config) map[string]bool
	GatherMetrics(config builder.Config) error
	SkipDeployStep()
}
