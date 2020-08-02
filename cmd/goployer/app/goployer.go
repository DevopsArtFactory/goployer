package app

import (
	"context"
	"github.com/DevopsArtFactory/goployer/cmd/goployer/cmd"
	"io"
)

func Run(out, stderr io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	catchCtrlC(cancel)

	c := cmd.NewRootCommand(out, stderr)
	return c.ExecuteContext(ctx)
}
