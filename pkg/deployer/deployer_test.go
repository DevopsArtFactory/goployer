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
	"testing"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

func TestDeployer_DecideCapacity(t *testing.T) {
	deployer := Deployer{
		Mode: constants.BlueGreenDeployment,
	}

	intended := schemas.Capacity{
		Min:     1,
		Max:     1,
		Desired: 1,
	}

	deployer.Stack = schemas.Stack{Capacity: intended}

	prev := schemas.Capacity{
		Min:     3,
		Max:     3,
		Desired: 3,
	}

	deployer.PrevInstanceCount = map[string]schemas.Capacity{
		constants.DefaultRegion: prev,
	}

	forceManifestCapacity := false
	completeCanary := false
	if out, _ := deployer.DecideCapacity(forceManifestCapacity, completeCanary, constants.DefaultRegion, len(deployer.PrevAsgs[constants.DefaultRegion]), deployer.Stack.RollingUpdateInstanceCount); out != prev {
		t.Error("BlueGreen capacity setting error")
	}

	forceManifestCapacity = true
	if out, _ := deployer.DecideCapacity(forceManifestCapacity, completeCanary, constants.DefaultRegion, len(deployer.PrevAsgs[constants.DefaultRegion]), deployer.Stack.RollingUpdateInstanceCount); out != intended {
		t.Error("BlueGreen capacity setting error with forceManifestCapacity")
	}

	canaryIntended := schemas.Capacity{
		Min:     1,
		Max:     1,
		Desired: 1,
	}
	deployer.Mode = constants.CanaryDeployment
	if out, _ := deployer.DecideCapacity(forceManifestCapacity, completeCanary, constants.DefaultRegion, len(deployer.PrevAsgs[constants.DefaultRegion]), deployer.Stack.RollingUpdateInstanceCount); out != canaryIntended {
		t.Error("Canary capacity setting error")
	}

	completeCanary = true
	forceManifestCapacity = false
	deployer.Mode = constants.CanaryDeployment
	if out, _ := deployer.DecideCapacity(forceManifestCapacity, completeCanary, constants.DefaultRegion, len(deployer.PrevAsgs[constants.DefaultRegion]), deployer.Stack.RollingUpdateInstanceCount); out != prev {
		t.Error("Canary capacity setting error with completeCanary")
	}

	forceManifestCapacity = true
	if out, _ := deployer.DecideCapacity(forceManifestCapacity, completeCanary, constants.DefaultRegion, len(deployer.PrevAsgs[constants.DefaultRegion]), deployer.Stack.RollingUpdateInstanceCount); out != intended {
		t.Error("Canary capacity setting error with forceManifestCapacity")
	}
}

func TestDeployer_ValidateOption(t *testing.T) {
	overRideSpotInstanceType := "t2.small|t3.small"
	instanceTypeList := []string{"c6g", "t4g", "m6g", "a1", "r6g"}
	validErr := checkSpotInstanceOption(overRideSpotInstanceType, instanceTypeList)
	if validErr != nil {
		t.Errorf("Invalid Override Spot Types Option: %s", validErr)
	}
}
