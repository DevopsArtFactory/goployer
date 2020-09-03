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
	"io"

	"github.com/spf13/cobra"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
)

// Create new delete command
func NewDeleteCommand() *cobra.Command {
	return NewCmd("delete").
		WithDescription("Delete previous applications").
		SetFlags().
		RunWithNoArgs(funcDelete)
}

// funcDelete delete stacks
func funcDelete(ctx context.Context, _ io.Writer, mode string) error {
	return runWithoutExecutor(ctx, func() error {
		//Create new builder
		builderSt, err := runner.SetupBuilder(mode)
		if err != nil {
			return err
		}

		//Start runner
		if err := runner.Start(builderSt, mode); err != nil {
			return err
		}

		return nil
	})
}
