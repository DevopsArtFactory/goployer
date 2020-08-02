package cmd

import (
	"context"
	"io"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
	"github.com/spf13/cobra"
)

// Create new deploy command
func NewDeleteCommand() *cobra.Command {
	return NewCmd("delete").
		WithDescription("Delete previous applications").
		SetFlags().
		RunWithNoArgs(funcDelete)
}

// funcDelete delete stacks
func funcDelete(ctx context.Context, _ io.Writer) error {
	return runWithoutExecutor(ctx, func() error {
		//Create new builder
		builderSt, err := runner.SetupBuilder()
		if err != nil {
			return err
		}

		//Start runner
		if err := runner.Start(builderSt, "delete"); err != nil {
			return err
		}

		return nil
	})
}
