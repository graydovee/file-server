package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/internal"
	"log"
	"net/http"
)

type HttpServer struct {
	engine *gin.Engine

	cfg *config.Config
}

type SubServer interface {
	Setup(s *gin.Engine) error
}

func NewHttpServer(cfg *config.Config) (*HttpServer, error) {
	return &HttpServer{
		cfg: cfg,
	}, nil
}

func (s *HttpServer) Run() error {
	s.engine = gin.Default()
	s.engine.LoadHTMLGlob("template/*")
	s.engine.Static("/assert", "./assert")

	if err := internal.NewFileServer(s.cfg).Setup(s.engine); err != nil {
		return err
	}
	if err := internal.NewCodeServer(s.cfg).Setup(s.engine); err != nil {
		return err
	}

	log.Printf("Server started at %s\n", s.cfg.Address)
	if err := http.ListenAndServe(s.cfg.Address, s.engine); err != nil {
		return err
	}
	return nil
}
