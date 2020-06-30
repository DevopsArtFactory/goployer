package main

import (
	"github.com/DevopsArtFactory/goployer/pkg/runner"
	Logger "github.com/sirupsen/logrus"
	"os"
)

func main() {
	//Create new builder
	if err := runner.Start(); err != nil {
		Logger.Error(err.Error())
		os.Exit(1)
	}
}
