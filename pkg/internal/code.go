package internal

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/graydovee/fileManager/pkg/config"
	"net/http"
	"os"
	"path/filepath"
)

type CodeServer struct {
	cfg *config.Config
}

func NewCodeServer(cfg *config.Config) *CodeServer {
	return &CodeServer{
		cfg: cfg,
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

	dirname := filepath.Join(s.cfg.UploadDir, "code", language)
	if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
		c.String(http.StatusInternalServerError, "create directory failed")
		return
	}

	filename := shortHash(code)
	filePath := filepath.Join(dirname, filename+ext)
	if err := os.WriteFile(filePath, []byte(code), os.ModePerm); err != nil {
		c.String(http.StatusInternalServerError, "write file failed")
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/code/%s/%s", language, filename))
}

func (s *CodeServer) handleCodeShow(c *gin.Context) {
	lang := c.Param("lang")
	hash := c.Param("hash")

	ext, ok := extMap[lang]
	if !ok {
		c.String(http.StatusBadRequest, "language not supported")
		return
	}

	filePath := filepath.Join(s.cfg.UploadDir, "code", lang, hash+ext)
	file, err := os.ReadFile(filePath)
	if err != nil {
		c.String(http.StatusNotFound, "code not found")
		return
	}
	c.HTML(http.StatusOK, "codeshow.html", gin.H{
		"Code":     string(file),
		"Language": lang,
	})
}

func shortHash(code string) string {
	hash := md5.Sum([]byte(code))
	hexString := hex.EncodeToString(hash[:])
	return hexString[:8]
}
