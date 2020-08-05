package cmd

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

// Create new deploy command
func NewServerCommand() *cobra.Command {
	return NewCmd("server").
		WithDescription("Run goployer as server").
		RunWithNoArgs(funcServer)
}

// funcDeploy run deployment
func funcServer(ctx context.Context, _ io.Writer, mode string) error {
	return runWithoutExecutor(ctx, func() error {
		// ready to develop
		return nil
	})
}
