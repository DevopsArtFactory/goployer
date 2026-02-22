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

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	Logger "github.com/sirupsen/logrus"

	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/runner"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
)

var (
	defaultLogLevel   = Logger.InfoLevel
	defaultServerAddr = "localhost"
	defaultServerPort = int64(9037)
)

type Server struct {
	ServerConfig Config
	Router       *http.ServeMux
	Logger       *Logger.Logger
}

type Config struct {
	Addr string
	Port int64
}

type RequestBody struct {
	Config schemas.Config `json:"config"`
}

func New() Server {
	return Server{
		Router: http.NewServeMux(),
		Logger: Logger.New(),
		ServerConfig: Config{
			Addr: defaultServerAddr,
			Port: defaultServerPort,
		},
	}
}

func (s Server) SetRouter() Server {
	s.Router.HandleFunc("/health", s.Healthcheck)
	s.Router.HandleFunc("/deploy", s.TriggerDeploy)
	return s
}

func (s Server) SetDefaultSetting() Server {
	s.Logger.Infof("Setup Default Settings")

	s.Logger.SetLevel(defaultLogLevel)
	s.Logger.Infof("Log Level : %s", defaultLogLevel)

	return s
}

func (s Server) Healthcheck(w http.ResponseWriter, req *http.Request) {
	s.Logger.Infof("%s %s healthy", req.RemoteAddr, req.Method)
}

func (s Server) TriggerDeploy(w http.ResponseWriter, req *http.Request) {
	body, err := parameterParsing(req.Body)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}

	builder, err := runner.ServerSetup(body.Config)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}

	if err := runner.Start(builder, "server"); err != nil {
		s.Logger.Error(err.Error())
		return
	}
}

func (s Server) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.ServerConfig.Addr, s.ServerConfig.Port)
}

// parameterParsing returns RequestBody
func parameterParsing(body io.Reader) (RequestBody, error) {
	decoder := json.NewDecoder(body)

	r := RequestBody{}
	err := decoder.Decode(&r)
	if err != nil {
		return r, err
	}

	if r.Config.Timeout <= 0 {
		r.Config.Timeout = constants.DefaultDeploymentTimeout
	}

	if r.Config.PollingInterval <= 0 {
		r.Config.PollingInterval = constants.DefaultPollingInterval
	}

	r.Config, err = builder.RefineConfig(r.Config)
	if err != nil {
		return r, err
	}

	return r, nil
}
