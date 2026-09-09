package pkg

import (
	"fmt"
	"log"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/server"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type HttpServer struct {
	engine *echo.Echo

	cfg *config.Config
}

type SubServer interface {
	Setup(e *echo.Echo) error
}

func NewHttpServer(cfg *config.Config) (*HttpServer, error) {
	s := &HttpServer{
		cfg: cfg,
	}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *HttpServer) init() error {
	s.engine = echo.New()
	s.engine.HideBanner = true
	s.engine.Use(middleware.Logger())
	s.engine.Use(middleware.Recover())

	var fileStore store.Store
	switch s.cfg.Store.Type {
	case config.StoreTypeLocal:
		fileStore = store.NewLocalStore(&s.cfg.Store.Local)
	case config.StoreTypeS3:
		st, err := store.NewS3Store(&s.cfg.Store.S3)
		if err != nil {
			return fmt.Errorf("error creating S3 store: %w", err)
		}
		fileStore = st
	default:
		return fmt.Errorf("unsupported store type: %s", s.cfg.Store.Type)
	}

	for _, sub := range []SubServer{
		server.NewFileServer(s.cfg, fileStore),
		server.NewCodeServer(s.cfg, fileStore),
	} {
		if err := sub.Setup(s.engine); err != nil {
			return err
		}
	}

	// 前端静态资源与 SPA 回退必须最后注册
	web := server.NewWebHandler(s.cfg.Web.DistDir)
	web.Register(s.engine)
	return nil
}

func (s *HttpServer) Run() error {
	log.Printf("Server started at %s\n", s.cfg.Address)
	return s.engine.Start(s.cfg.Address)
}
