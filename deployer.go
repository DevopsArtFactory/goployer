package main

import (
	"github.com/DevopsArtFactory/deployer/application"
	Logger "github.com/sirupsen/logrus"
	"os"
)

func main()  {
	//Create new builder
	if err := application.Start(); err != nil{
		Logger.Error(err.Error())
		os.Exit(1)
	}
}
