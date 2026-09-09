package server

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/labstack/echo/v4"
)

// CodeServer 提供代码片段的分享与展示 API
type CodeServer struct {
	store store.Store
}

func NewCodeServer(_ *config.Config, st store.Store) *CodeServer {
	return &CodeServer{store: st}
}

func (s *CodeServer) Setup(e *echo.Echo) error {
	g := e.Group("/api/code")
	g.POST("", s.handleUpload)
	g.GET("/:lang/:hash", s.handleCodeShow)
	return nil
}

// extMap maps programming languages to their respective file extensions
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

type codeUploadRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

func (s *CodeServer) handleUpload(c echo.Context) error {
	var req codeUploadRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Errorf("Error parsing code upload request: %v", err)
		return jsonError(c, http.StatusBadRequest, "Invalid request")
	}

	if req.Code == "" || req.Language == "" {
		return jsonError(c, http.StatusBadRequest, "Code or language is empty")
	}

	ext, ok := extMap[req.Language]
	if !ok {
		return jsonError(c, http.StatusBadRequest, "Language not supported")
	}

	// Generate filename using a short hash of the code
	filename := GetTimeStamp() + "-" + shortHash(req.Code)
	filePath := filepath.Join("code", req.Language, filename+ext)

	// Upload the file to the store
	buffer := bytes.NewBufferString(req.Code)
	if err := s.store.UploadFile(c.Request().Context(), buffer, filePath); err != nil {
		c.Logger().Errorf("Failed to save code %s: %v", filePath, err)
		return jsonError(c, http.StatusInternalServerError, "Failed to save code")
	}

	displayURL := fmt.Sprintf("/code/%s/%s", req.Language, filename)
	return c.JSON(http.StatusOK, map[string]string{"url": displayURL})
}

func (s *CodeServer) handleCodeShow(c echo.Context) error {
	lang := c.Param("lang")
	hash := c.Param("hash")

	ext, ok := extMap[lang]
	if !ok {
		return jsonError(c, http.StatusBadRequest, "Language not supported")
	}

	filePath := filepath.Join("code", lang, hash+ext)

	// Check if the file exists
	meta, err := s.store.FileMeta(c.Request().Context(), filePath)
	if err != nil {
		c.Logger().Errorf("Error checking file existence %s: %v", filePath, err)
		return jsonError(c, http.StatusInternalServerError, "Error checking file")
	}

	if meta == nil {
		return jsonError(c, http.StatusNotFound, "Code not found")
	}

	// Download the file content
	buffer := bytes.NewBuffer(nil)
	if err := s.store.DownloadFile(c.Request().Context(), buffer, filePath); err != nil {
		c.Logger().Errorf("Failed to download code %s: %v", filePath, err)
		return jsonError(c, http.StatusInternalServerError, "Failed to download code")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"language":    lang,
		"code":        buffer.String(),
		"downloadUrl": "/download/" + EscapeUrlPath(filePath),
	})
}

// shortHash generates a short MD5 hash from the given code string
func shortHash(code string) string {
	hash := md5.Sum([]byte(code))
	hexString := hex.EncodeToString(hash[:])
	return hexString[:8]
}
