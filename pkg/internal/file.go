package internal

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/graydovee/fileManager/pkg/config"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FilerServer struct {
	cfg *config.Config
}

func NewFileServer(cfg *config.Config) *FilerServer {
	return &FilerServer{
		cfg: cfg,
	}
}

func (f *FilerServer) Setup(s *gin.Engine) error {
	s.GET("/", f.handleHelpPage)
	s.POST("/upload", f.uploadFileHandlerByForm)
	s.PUT("/upload", f.uploadFileHandlerByStream)
	s.GET("/download/*file", f.downloadFileHandler)
	log.Printf("Upload directory: %s\n", f.cfg.UploadDir)
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

func (f *FilerServer) saveFile(file io.ReadCloser, filename string, c *gin.Context) {

	// Generate UUID for filename and remove dashes
	newFileName := strings.ReplaceAll(uuid.New().String(), "-", "") + "-" + filename

	// Create directory structure based on current year and month
	now := time.Now()
	yearMonthPath := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	yearMonthPathDir := filepath.Join(f.cfg.UploadDir, yearMonthPath)
	if err := os.MkdirAll(yearMonthPathDir, os.ModePerm); err != nil {
		log.Printf("Error creating directory %s: %s\n", yearMonthPathDir, err.Error())
		c.String(http.StatusInternalServerError, "Error creating directory")
		return
	}

	filePath := filepath.Join(yearMonthPath, newFileName)

	fullPath := filepath.Join(f.cfg.UploadDir, filePath)

	// Create new file
	newFile, err := os.Create(fullPath)
	if err != nil {
		log.Printf("Error creating the file %s: %s\n", filePath, err.Error())
		c.String(http.StatusInternalServerError, "Error creating the file")
		return
	}
	defer newFile.Close()

	// Copy the uploaded file to the new file
	_, err = io.Copy(newFile, file)
	if err != nil {
		log.Printf("Error saving the file %s: %s\n", filePath, err.Error())
		c.String(http.StatusInternalServerError, "Error saving the file")
		return
	}

	log.Printf("File %s uploaded successfully: %s\n", filename, filePath)

	downloadUrl := getDownloadUrl(c.Request.Host, filePath, f.cfg.EnableTls)
	respData := fmt.Sprintf(`
File uploaded successfully.

Download command:
	wget %s -O %s
`, downloadUrl, filename)
	internalDownloadUrl := getDownloadUrl(getInternalHost(f.cfg.Address, f.cfg.InternalHost), filePath, false)
	if internalDownloadUrl != downloadUrl {
		respData += fmt.Sprintf(`
Internal download command:
	wget %s -O %s
`, internalDownloadUrl, filename)
	}
	c.String(http.StatusOK, respData)
}

func (f *FilerServer) downloadFileHandler(c *gin.Context) {
	file := c.Param("file")

	fullFilePath := filepath.Join(f.cfg.UploadDir, file)
	log.Printf("Download file: %s\n", file)

	c.File(fullFilePath)
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
	return fmt.Sprintf("%s://%s/download/%s", schema, host, filePath)
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
