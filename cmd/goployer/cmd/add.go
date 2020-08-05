package cmd

import (
	"context"
	"io"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
	"github.com/spf13/cobra"
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
