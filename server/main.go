/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package main

import (
	"net/http"

	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/server"
)

func main() {
	Logger.Info("Booting up goployer server")
	s := server.New().
		SetDefaultSetting().
		SetRouter()

	Logger.Info("Server setting is done")

	addr := s.GetAddr()
	s.Logger.Info("Start goployer server")
	if err := http.ListenAndServe(addr, s.Router); err != nil {
		s.Logger.Error(err.Error())
	}
	s.Logger.Info("Shutting down goployer server")
}
