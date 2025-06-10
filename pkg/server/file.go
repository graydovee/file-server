package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/labstack/echo/v4"
)

const downloadPath = "download"

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

func (f *FilerServer) Setup(e *echo.Echo) error {
	e.GET("/", f.handleHelpPage)
	e.POST("/upload", f.uploadFileHandlerByForm)
	e.PUT("/upload", f.uploadFileHandlerByStream)
	e.GET(fmt.Sprintf("/%s", downloadPath), f.downloadFileHandler)
	e.GET(fmt.Sprintf("/%s/", downloadPath), f.downloadFileHandler)
	e.GET(fmt.Sprintf("/%s/*", downloadPath), f.downloadFileHandler)
	e.DELETE("/delete/*", f.deleteFileHandler)

	return nil
}

func (f *FilerServer) handleHelpPage(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"UploadAddress":   getUploadAddress(c.Request().Host, f.cfg.EnableTls),
		"DownloadAddress": getDownloadUrl(c.Request().Host, "[year]/[month]/[file_name]", f.cfg.EnableTls),
	})
}

func (f *FilerServer) uploadFileHandlerByForm(c echo.Context) error {
	c.Logger().Printf("File uploaded by form")

	contentType := c.Request().Header.Get("Content-Type")

	if !strings.HasPrefix(contentType, "multipart/form-data") {
		c.Logger().Errorf("Unsupported content type: %s", contentType)
		return c.String(http.StatusBadRequest, "Unsupported content type")
	}

	// Handle form file upload
	formFile, header, err := c.Request().FormFile("file")
	if err != nil {
		c.Logger().Errorf("Error retrieving the file: %s", err.Error())
		return c.String(http.StatusInternalServerError, "Error retrieving the file")
	}
	defer formFile.Close()

	return f.saveFile(formFile, header.Filename, c)
}

func (f *FilerServer) uploadFileHandlerByStream(c echo.Context) error {
	c.Logger().Printf("File uploaded by stream")

	// Handle direct file upload
	file := c.Request().Body
	fileName := c.Request().Header.Get("X-Filename")
	if fileName == "" {
		c.Logger().Errorf("Error: X-Filename header is missing")
		return c.String(http.StatusBadRequest, "X-Filename header is missing")
	}

	return f.saveFile(file, fileName, c)
}

func (f *FilerServer) saveFile(file io.ReadCloser, filename string, c echo.Context) error {
	// Generate unique filename using timestamp and original filename
	newFileName := strings.ReplaceAll(GetTimeStamp(), "-", "") + "-" + filename

	// Create directory structure based on current year and month
	now := time.Now()
	yearMonthPath := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	filePath := fmt.Sprintf("%s/%s", yearMonthPath, newFileName)

	// Upload to Store
	err := f.store.UploadFile(context.Background(), file, filePath)
	if err != nil {
		c.Logger().Errorf("Error uploading the file %s to store: %s", filename, err.Error())
		return c.String(http.StatusInternalServerError, "Error uploading the file to the store")
	}

	c.Logger().Printf("File %s uploaded successfully to store: %s", filename, filePath)

	downloadUrl := getDownloadUrl(c.Request().Host, EscapeUrlPath(filePath), f.cfg.EnableTls)

	// External download command
	respData := fmt.Sprintf(`
File uploaded successfully.

Download command:
	wget %s -O %s
`, downloadUrl, escapeFileName(filename))

	internalDownloadUrl := getDownloadUrl(getInternalHost(f.cfg.Address, f.cfg.InternalHost), EscapeUrlPath(filePath), false)
	if internalDownloadUrl != downloadUrl {
		// Internal download command
		respData += fmt.Sprintf(`
Internal download command:
	wget %s -O %s
`, internalDownloadUrl, escapeFileName(filename))
	}

	return c.String(http.StatusOK, respData)
}

func (f *FilerServer) downloadFileHandler(c echo.Context) error {
	file := strings.TrimPrefix(c.Param("*"), "/")

	c.Logger().Printf("Download file: %s", file)

	meta, err := f.store.FileMeta(context.Background(), strings.TrimSuffix(file, "/"))
	if err != nil {
		c.Logger().Errorf("Error checking the file %s: %s", file, err.Error())
		return c.String(http.StatusInternalServerError, "Error checking the file")
	}

	if meta == nil {
		// file list page
		fileMetas, err := f.store.List(context.Background(), file)
		if err != nil {
			c.Logger().Errorf("Error listing the file %s: %s", file, err.Error())
			return c.String(http.StatusInternalServerError, "Error listing the file")
		}

		return c.Render(http.StatusOK, "list.html", map[string]interface{}{
			"DownloadEndpoint": getDownloadUrl(c.Request().Host, file, f.cfg.EnableTls),
			"DeleteEndpoint":   getDeleteUrl(c.Request().Host, file, f.cfg.EnableTls),
			"BasePath":         downloadPath,
			"Files":            fileMetas,
		})
	}

	// download file

	// Set headers
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(file)))
	c.Response().Header().Set("Content-Type", "application/octet-stream")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	c.Response().WriteHeader(http.StatusOK)

	// Stream the file
	err = f.store.DownloadFile(context.Background(), c.Response().Writer, file)
	if err != nil {
		c.Logger().Errorf("Error downloading the file %s: %s", file, err.Error())
		return err // You may choose to handle this differently
	}

	return nil
}

func (f *FilerServer) deleteFileHandler(c echo.Context) error {
	file := strings.TrimPrefix(c.Param("*"), "/")

	c.Logger().Printf("Delete file: %s", file)

	err := f.store.DeleteFile(context.Background(), file)
	if err != nil {
		c.Logger().Errorf("Error deleting the file %s: %s", file, err.Error())
		return c.String(http.StatusInternalServerError, "Error deleting the file")
	}

	c.Logger().Printf("File %s deleted successfully", file)

	return c.String(http.StatusOK, "File deleted successfully")
}

func getUploadAddress(host string, enableTls bool) string {
	return getUrl(host, "upload", enableTls)
}

func getDownloadUrl(host, filePath string, enableTls bool) string {
	return getUrl(host, filepath.Join(downloadPath, strings.TrimPrefix(filePath, "/")), enableTls)
}

func getDeleteUrl(host, filePath string, enableTls bool) string {
	return getUrl(host, filepath.Join("delete", strings.TrimPrefix(filePath, "/")), enableTls)
}

func getUrl(host, filePath string, enableTls bool) string {
	var schema string
	if enableTls {
		schema = "https"
	} else {
		schema = "http"
	}

	filePath = strings.Trim(filePath, "/")

	endPoint := fmt.Sprintf("%s://%s", schema, host)
	if filePath == "" {
		return endPoint
	}
	return fmt.Sprintf("%s/%s", endPoint, filePath)
}

func EscapeUrlPath(filePath string) string {
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
