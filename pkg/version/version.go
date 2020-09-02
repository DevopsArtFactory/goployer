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

package version

import (
	"fmt"
	"runtime"
)

var version, gitCommit, gitTreeState, buildDate string
var platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

type Controller struct{}

type Info struct {
	Version      string
	BuildDate    string
	GitCommit    string
	GitTreeState string
	Platform     string
}

func Get() Info {
	return Info{
		Version:      version,
		BuildDate:    buildDate,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		Platform:     platform,
	}
}

func (v Controller) Print(info Info) error {
	_, err := fmt.Println(info.Version)
	return err
}
