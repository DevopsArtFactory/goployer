package application

import (
	Logger "github.com/sirupsen/logrus"
)


type BlueGreen struct {
	Deployer
}

func _NewBlueGrean(logger *Logger.Logger, mode string, prefix string, client AWSClient) BlueGreen {
	return BlueGreen{
		Deployer{
			Logger: logger,
			Mode: mode,
			Prefix: prefix,
			AWSClient: client,
		},
	}
}
