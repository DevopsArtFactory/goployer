package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
	"github.com/spf13/cobra"
)

// Create new update command
func NewUpdateCommand() *cobra.Command {
	return NewCmd("update").
		WithDescription("Update configuration of current deployment").
		SetFlags().
		RunWithArgs(funcUpdate)
}

// funcUpdate updates configurations of current deployment stack
func funcUpdate(ctx context.Context, _ io.Writer, args []string, mode string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: goployer update <application name> --region=<region ID> --min=val --max=val --desired=val")
	}
	return runWithoutExecutor(ctx, func() error {
		//Create new builder
		builderSt, err := runner.SetupBuilder(mode)
		if err != nil {
			return err
		}

		builderSt.Config.Application = args[0]
		builderSt.Config.LogLevel = "debug"

		//Start runner
		if err := runner.Start(builderSt, mode); err != nil {
			return err
		}

		return nil
	})
}
