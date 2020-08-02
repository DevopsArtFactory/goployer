package cmd

import (
	"context"
	"io"

	"github.com/DevopsArtFactory/goployer/pkg/runner"
	"github.com/spf13/cobra"
)

// Create new deploy command
func NewDeployCommand() *cobra.Command {
	return NewCmd("deploy").
		WithDescription("Deploy a new application").
		SetFlags().
		RunWithNoArgs(funcDeploy)
}

// funcDeploy run deployment
func funcDeploy(ctx context.Context, _ io.Writer) error {
	return runWithoutExecutor(ctx, func() error {
		//Create new builder
		builderSt, err := runner.SetupBuilder()
		if err != nil {
			return err
		}

		//Start runner
		if err := runner.Start(builderSt); err != nil {
			return err
		}

		return nil
	})
}
