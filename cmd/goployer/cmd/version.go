package cmd

import (
	"context"
	"github.com/DevopsArtFactory/goployer/pkg/version"
	"github.com/spf13/cobra"
	"io"
)

//Create Command for get pod
func NewVersionCommand() *cobra.Command {
	return NewCmd("version").
		WithDescription("check goployer release version").
		RunWithNoArgs(execVersion)
}

// Function for search execution
func execVersion(_ context.Context, _ io.Writer, _ string) error {
	return version.New().Print()
}
