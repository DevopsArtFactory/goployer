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

package cmd

import (
	"context"
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
)

// Create new refresh command
func NewRefreshCommand() *cobra.Command {
	return NewCmd("refresh").
		WithDescription("Refresh autoscaling group instances").
		SetFlags().
		RunWithArgs(funcRefresh)
}

// funcRefresh refreshes autoscaling group instances with new template without creating new autoscaling group
func funcRefresh(ctx context.Context, _ io.Writer, args []string, mode string) error {
	if len(args) != 1 {
		return errors.New("usage: goployer refresh <application name> [ --region <region-id> ]")
	}

	return runWithoutExecutor(ctx, func() error {
		//Create new builder
		builderSt, err := runner.SetupBuilder(mode)
		if err != nil {
			return err
		}

		builderSt.Config.Application = args[0]

		//Start runner
		if err := runner.Start(builderSt, mode); err != nil {
			return err
		}

		return nil
	})
}
