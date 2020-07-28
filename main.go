package main

import (
	"github.com/DevopsArtFactory/goployer/pkg/runner"
	Logger "github.com/sirupsen/logrus"
	"os"
)

func main() {
	//Create new builder
	builderSt, err := runner.SetupBuilder()
	if err != nil {
		Logger.Errorf(err.Error())
		os.Exit(1)
	}

	//Start runner
	if err := runner.Start(builderSt); err != nil {
		Logger.Errorf(err.Error())
		os.Exit(1)
	}
}
