package server

import (
	"encoding/json"
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/runner"
	Logger "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

var (
	defaultLogLevel   = Logger.InfoLevel
	defaultServerAddr = "localhost"
	defaultServerPort = int64(9037)
)

type Server struct {
	ServerConfig ServerConfig
	Router       *http.ServeMux
	Logger       *Logger.Logger
}

type ServerConfig struct {
	Addr string
	Port int64
}

type RequestBody struct {
	Config builder.Config `json:"config"`
}

func New() Server {
	return Server{
		Router: http.NewServeMux(),
		Logger: Logger.New(),
		ServerConfig: ServerConfig{
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
	return
}

func (s Server) TriggerDeploy(w http.ResponseWriter, req *http.Request) {
	body, err := parameterParsing(req.Body)
	if err != nil {
		s.Logger.Errorf(err.Error())
		return
	}

	builder, err := runner.ServerSetup(body.Config)
	if err != nil {
		s.Logger.Errorf(err.Error())
		return
	}

	if err := runner.Start(builder, "server"); err != nil {
		s.Logger.Errorf(err.Error())
		return
	}
}

func (s Server) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.ServerConfig.Addr, s.ServerConfig.Port)
}

//parameterParsing returns RequestBody
func parameterParsing(body io.ReadCloser) (RequestBody, error) {
	decoder := json.NewDecoder(body)

	r := RequestBody{}
	err := decoder.Decode(&r)
	if err != nil {
		return r, err
	}

	if r.Config.Timeout <= 0 {
		r.Config.Timeout = builder.DEFAULT_DEPLOYMENT_TIMEOUT
	}

	if r.Config.PollingInterval <= 0 {
		r.Config.PollingInterval = builder.DEFAULT_POLLING_INTERVAL
	}

	r.Config = builder.RefineConfig(r.Config)

	return r, nil
}
