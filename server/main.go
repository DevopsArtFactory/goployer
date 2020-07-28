package main

import (
	"github.com/DevopsArtFactory/goployer/pkg/server"
	Logger "github.com/sirupsen/logrus"
	"net/http"
)

func main() {
	Logger.Infof("Booting up goployer server")
	s := server.New().
		SetDefaultSetting().
		SetRouter()

	Logger.Infof("Server setting is done")

	addr := s.GetAddr()
	s.Logger.Infof("Start goployer server")
	if err := http.ListenAndServe(addr, s.Router); err != nil {
		s.Logger.Errorf(err.Error())
	}
	s.Logger.Infof("Shutting down goployer server")
}
