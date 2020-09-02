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

// Create new deploy command
func NewAddCommand() *cobra.Command {
	return NewCmd("add").
		WithDescription("Add goployer manifest file").
		SetFlags().
		RunWithArgs(funcAdd)
}

// funcAdd add single application manifest file
func funcAdd(ctx context.Context, _ io.Writer, args []string, _ string) error {
	return runWithoutExecutor(ctx, func() error {
		if err := runner.AddManifest(args); err != nil {
			return err
		}

		return nil
	})
}
