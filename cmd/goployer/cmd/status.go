package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
	"github.com/spf13/cobra"
)

// Create new deploy command
func NewStatusCommand() *cobra.Command {
	return NewCmd("status").
		WithDescription("Get status of deployment").
		SetFlags().
		RunWithArgs(funcStatus)
}

// funcStatus shows deployment status
func funcStatus(ctx context.Context, _ io.Writer, args []string, mode string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: goployer status <application name> --region=<region ID>")
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
