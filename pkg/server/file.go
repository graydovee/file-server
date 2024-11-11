package server

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type FilerServer struct {
	cfg   *config.Config
	store store.Store
}

func NewFileServer(cfg *config.Config, st store.Store) *FilerServer {
	return &FilerServer{
		cfg:   cfg,
		store: st,
	}
}

func (f *FilerServer) Setup(s *gin.Engine) error {
	s.GET("/", f.handleHelpPage)
	s.POST("/upload", f.uploadFileHandlerByForm)
	s.PUT("/upload", f.uploadFileHandlerByStream)
	s.GET("/download/*file", f.downloadFileHandler)
	s.DELETE("/delete/*file", f.deleteFileHandler)
	return nil
}

func (f *FilerServer) handleHelpPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"UploadAddress":   getUploadAddress(c.Request.Host, f.cfg.EnableTls),
		"DownloadAddress": getDownloadUrl(c.Request.Host, "[year]/[month]/[file_name]", f.cfg.EnableTls),
	})
}

func (f *FilerServer) uploadFileHandlerByForm(c *gin.Context) {
	log.Printf("File uploaded by form")

	contentType := c.Request.Header.Get("Content-Type")

	if !strings.HasPrefix(contentType, "multipart/form-data") {
		log.Printf("Unsupported content type: %s\n", contentType)
		c.String(http.StatusBadRequest, "Unsupported content type")
		return
	}

	// Handle form file upload
	formFile, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Error retrieving the file: %s\n", err.Error())
		c.String(http.StatusInternalServerError, "Error retrieving the file")
		return
	}
	defer formFile.Close()

	f.saveFile(formFile, header.Filename, c)
}

func (f *FilerServer) uploadFileHandlerByStream(c *gin.Context) {
	log.Printf("File uploaded by stream")

	// Handle direct file upload
	file := c.Request.Body
	fileName := c.Request.Header.Get("X-Filename")
	if fileName == "" {
		log.Printf("Error: X-Filename header is missing")
		c.String(http.StatusBadRequest, "X-Filename header is missing")
		return
	}

	f.saveFile(file, fileName, c)
}

const layout = "20060102150405000"

func (f *FilerServer) saveFile(file io.ReadCloser, filename string, c *gin.Context) {
	// Generate UUID for filename and remove dashes
	newFileName := strings.ReplaceAll(time.Now().Format(layout), "-", "") + "-" + filename

	// Create directory structure based on current year and month
	now := time.Now()
	yearMonthPath := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	filePath := fmt.Sprintf("%s/%s", yearMonthPath, newFileName)

	// Upload to Store
	err := f.store.UploadFile(context.Background(), file, filePath)
	if err != nil {
		log.Printf("Error uploading the file %s to S3: %s\n", filename, err.Error())
		c.String(http.StatusInternalServerError, "Error uploading the file to S3")
		return
	}

	log.Printf("File %s uploaded successfully to S3: %s\n", filename, filePath)

	downloadUrl := getDownloadUrl(c.Request.Host, filePath, f.cfg.EnableTls)

	// external download command
	respData := fmt.Sprintf(`
File uploaded successfully.

Download command:
	wget %s -O %s
`, downloadUrl, escapeFileName(filename))

	internalDownloadUrl := getDownloadUrl(getInternalHost(f.cfg.Address, f.cfg.InternalHost), filePath, false)
	if internalDownloadUrl != downloadUrl {
		// internal download command
		respData += fmt.Sprintf(`
Internal download command:
	wget %s -O %s
`, internalDownloadUrl, escapeFileName(filename))
	}

	c.String(http.StatusOK, respData)
}

func (f *FilerServer) downloadFileHandler(c *gin.Context) {

	file := strings.TrimPrefix(c.Param("file"), "/")

	log.Printf("Download file: %s\n", file)

	exists, err := f.store.FileExists(context.Background(), file)
	if err != nil {
		log.Printf("Error checking the file %s: %s\n", file, err.Error())
		c.String(http.StatusInternalServerError, "Error checking the file")
		return
	}

	if !exists {
		log.Printf("File %s not found\n", file)
		c.String(http.StatusNotFound, "File not found")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(file)))
	c.Header("Content-Type", "application/octet-stream")

	c.Status(http.StatusOK)
	if err = f.store.DownloadFile(context.Background(), c.Writer, file); err != nil {
		log.Printf("Error downloading the file %s: %s\n", file, err.Error())
		return
	}
}

func (f *FilerServer) deleteFileHandler(c *gin.Context) {
	file := strings.TrimPrefix(c.Param("file"), "/")

	log.Printf("Delete file: %s\n", file)

	err := f.store.DeleteFile(context.Background(), file)
	if err != nil {
		log.Printf("Error deleting the file %s: %s\n", file, err.Error())
		c.String(http.StatusInternalServerError, "Error deleting the file")
		return
	}

	log.Printf("File %s deleted successfully\n", file)

	c.String(http.StatusOK, "File deleted successfully")
}

func getUploadAddress(host string, enableTls bool) string {
	var schema string
	if enableTls {
		schema = "https"
	} else {
		schema = "http"
	}
	return fmt.Sprintf("%s://%s/upload", schema, host)
}

func getDownloadUrl(host, filePath string, enableTls bool) string {
	var schema string
	if enableTls {
		schema = "https"
	} else {
		schema = "http"
	}
	return fmt.Sprintf("%s://%s/download/%s", schema, host, encodeUrlPath(filePath))
}

func encodeUrlPath(filePath string) string {
	if len(filePath) == 0 {
		return ""
	}
	filePath = filepath.ToSlash(filePath)

	parts := strings.Split(filePath, "/")

	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}

	return strings.Join(parts, "/")
}

func escapeFileName(fileName string) string {
	specialChars := []string{"(", ")", " ", "&", "$", "#", "@"}

	for _, char := range specialChars {
		fileName = strings.ReplaceAll(fileName, char, "\\"+char)
	}

	return fileName
}

func getInternalHost(listenAddr, overrideHost string) string {
	sp := strings.Split(listenAddr, ":")
	if len(sp) != 2 {
		return overrideHost
	}
	port := sp[1]
	host := sp[0]
	if overrideHost != "" {
		host = overrideHost
	}
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%s", host, port)
}
