package server

import (
	"fmt"
	Logger "github.com/sirupsen/logrus"
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

}

func (s Server) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.ServerConfig.Addr, s.ServerConfig.Port)
}
