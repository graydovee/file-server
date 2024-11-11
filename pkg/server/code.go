package server

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type CodeServer struct {
	store store.Store
}

func NewCodeServer(_ *config.Config, st store.Store) *CodeServer {
	return &CodeServer{
		store: st,
	}
}

func (s *CodeServer) Setup(e *gin.Engine) error {
	e.GET("/code", s.handleUploadPage)
	e.GET("/code/:lang/:hash", s.handleCodeShow)
	e.POST("/code", s.handleUpload)
	return nil
}

var extMap = map[string]string{
	"c":          ".c",
	"cpp":        ".cpp",
	"bash":       ".sh",
	"go":         ".go",
	"python":     ".py",
	"java":       ".java",
	"javascript": ".js",
	"rust":       ".rs",
	"php":        ".php",
	"html":       ".html",
	"yaml":       ".yaml",
	"json":       ".json",
	"xml":        ".xml",
}

func (s *CodeServer) handleUploadPage(c *gin.Context) {
	c.HTML(http.StatusOK, "code.html", gin.H{
		"ExtMap": extMap,
	})
}

func (s *CodeServer) handleUpload(c *gin.Context) {
	code := c.PostForm("code")
	language := c.PostForm("language")

	if code == "" || language == "" {
		c.String(http.StatusBadRequest, "code or language is empty")
		return
	}
	ext, ok := extMap[language]
	if !ok {
		c.String(http.StatusBadRequest, "language not supported")
		return
	}

	dirname := filepath.Join("code", language)
	if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
		c.String(http.StatusInternalServerError, "create directory failed")
		return
	}

	filename := shortHash(code)
	filePath := filepath.Join(dirname, filename+ext)

	buffer := bytes.NewBuffer([]byte(code))
	if err := s.store.UploadFile(context.Background(), buffer, filePath); err != nil {
		log.Printf("failed to save code %s: %v\n", filePath, err)
		return
	}
}

func (s *CodeServer) handleCodeShow(c *gin.Context) {
	lang := c.Param("lang")
	hash := c.Param("hash")

	ext, ok := extMap[lang]
	if !ok {
		c.String(http.StatusBadRequest, "language not supported")
		return
	}

	buffer := bytes.NewBuffer(nil)
	filePath := filepath.Join("code", lang, hash+ext)

	if err := s.store.DownloadFile(context.Background(), buffer, filePath); err != nil {
		c.String(http.StatusNotFound, "code not found")
		return
	}
	c.HTML(http.StatusOK, "codeshow.html", gin.H{
		"Code":     buffer.String(),
		"Language": lang,
	})
}

func shortHash(code string) string {
	hash := md5.Sum([]byte(code))
	hexString := hex.EncodeToString(hash[:])
	return hexString[:8]
}
