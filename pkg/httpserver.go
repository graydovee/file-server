package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/server"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
)

type HttpServer struct {
	engine *echo.Echo

	cfg *config.Config
}

type SubServer interface {
	Setup(s *gin.Engine) error
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
	s.engine.Use(middleware.Logger())
	s.engine.Use(middleware.Recover())

	var fileStore store.Store
	switch s.cfg.Store.Type {
	case config.StoreTypeLocal:
		fileStore = store.NewLocalStore(&s.cfg.Store.Local)
	case config.StoreTypeS3:
		st, err := store.NewS3Store(&s.cfg.Store.S3)
		if err != nil {
			log.Fatalf("Error creating S3 store: %s\n", err.Error())
		}
		fileStore = st
	default:
		log.Fatalf("Unsupported store type: %s\n", s.cfg.Store.Type)
	}

	if err := server.NewFileServer(s.cfg, fileStore).Setup(s.engine); err != nil {
		return err
	}
	if err := server.NewCodeServer(s.cfg, fileStore).Setup(s.engine); err != nil {
		return err
	}

	templates, err := template.ParseGlob(filepath.Join(s.cfg.Resource.TemplateDir, "*"))
	if err != nil {
		return err
	}
	s.engine.Renderer = &Template{
		templates: templates,
	}
	s.engine.Static("/assert", s.cfg.Resource.StaticDir) // Update the path to your static files
	return nil
}

func (s *HttpServer) Run() error {
	log.Printf("Server started at %s\n", s.cfg.Address)
	if err := http.ListenAndServe(s.cfg.Address, s.engine); err != nil {
		return err
	}
	return nil
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
