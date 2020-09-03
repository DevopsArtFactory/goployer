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

package main

import (
	"context"
	"errors"
	"os"

	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/cmd/goployer/app"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

func main() {
	if err := app.Run(os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, context.Canceled) {
			Logger.Debugln("ignore error since context is cancelled:", err)
		} else {
			tool.Red.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
