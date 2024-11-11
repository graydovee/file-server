package server

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
)

// CodeServer handles code upload and display functionalities
type CodeServer struct {
	store store.Store
}

// NewCodeServer creates a new instance of CodeServer
func NewCodeServer(_ *config.Config, st store.Store) *CodeServer {
	return &CodeServer{
		store: st,
	}
}

// Setup configures the routes and middleware for CodeServer
func (s *CodeServer) Setup(e *echo.Echo) error {
	group := e.Group("/code")

	// Routes
	group.GET("", s.handleUploadPage)
	group.GET("/:lang/:hash", s.handleCodeShow)
	group.POST("", s.handleUpload)

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

// handleUploadPage renders the code upload page with supported extensions
func (s *CodeServer) handleUploadPage(c echo.Context) error {
	return c.Render(http.StatusOK, "code.html", map[string]interface{}{
		"ExtMap": extMap,
	})
}

// handleUpload processes the code upload form
func (s *CodeServer) handleUpload(c echo.Context) error {
	// Parse the form data
	if err := c.Request().ParseForm(); err != nil {
		c.Logger().Errorf("Error parsing form: %v", err)
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	code := c.FormValue("code")
	language := c.FormValue("language")

	if code == "" || language == "" {
		return c.String(http.StatusBadRequest, "Code or language is empty")
	}

	ext, ok := extMap[language]
	if !ok {
		return c.String(http.StatusBadRequest, "Language not supported")
	}

	// Create directory structure
	dirname := filepath.Join("code", language)
	if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
		c.Logger().Errorf("Failed to create directory %s: %v", dirname, err)
		return c.String(http.StatusInternalServerError, "Create directory failed")
	}

	// Generate filename using a short hash of the code
	filename := GetTimeStamp() + "-" + shortHash(code)
	filePath := filepath.Join(dirname, filename+ext)

	// Upload the file to the store
	buffer := bytes.NewBuffer([]byte(code))
	if err := s.store.UploadFile(context.Background(), buffer, filePath); err != nil {
		c.Logger().Errorf("Failed to save code %s: %v", filePath, err)
		return c.String(http.StatusInternalServerError, "Failed to save code")
	}

	// Optionally, you can redirect to the code display page
	displayURL := fmt.Sprintf("/code/%s/%s", language, filename)
	return c.Redirect(http.StatusSeeOther, displayURL)
}

// handleCodeShow retrieves and displays the uploaded code
func (s *CodeServer) handleCodeShow(c echo.Context) error {
	lang := c.Param("lang")
	hash := c.Param("hash")

	ext, ok := extMap[lang]
	if !ok {
		return c.String(http.StatusBadRequest, "Language not supported")
	}

	filePath := filepath.Join("code", lang, hash+ext)

	// Check if the file exists
	exists, err := s.store.FileExists(context.Background(), filePath)
	if err != nil {
		c.Logger().Errorf("Error checking file existence %s: %v", filePath, err)
		return c.String(http.StatusInternalServerError, "Error checking file")
	}

	if !exists {
		return c.String(http.StatusNotFound, "Code not found")
	}

	// Download the file content
	buffer := bytes.NewBuffer(nil)
	if err := s.store.DownloadFile(context.Background(), buffer, filePath); err != nil {
		c.Logger().Errorf("Failed to download code %s: %v", filePath, err)
		return c.String(http.StatusInternalServerError, "Failed to download code")
	}

	data := map[string]interface{}{
		"Code":     buffer.String(),
		"Language": lang,
	}

	return c.Render(http.StatusOK, "codeshow.html", data)
}

// shortHash generates a short MD5 hash from the given code string
func shortHash(code string) string {
	hash := md5.Sum([]byte(code))
	hexString := hex.EncodeToString(hash[:])
	return hexString[:8]
}
