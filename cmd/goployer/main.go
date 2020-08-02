package main

import (
	"context"
	"errors"
	"github.com/DevopsArtFactory/goployer/cmd/goployer/app"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
	"os"
)

func main() {
	if err := app.Run(os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, context.Canceled) {
			Logger.Debugln("ignore error since context is cancelled:", err)
		} else {
			tool.Red.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
