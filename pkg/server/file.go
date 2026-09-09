package server

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/labstack/echo/v4"
)

const (
	downloadPath     = "download"
	defaultPageSize  = 100
	maxPageSize      = 1000
	defaultSearchLim = 200
	maxSearchLimit   = 1000
	maxArchiveFiles  = 10000
	maxScriptFiles   = 10000
)

type FileServer struct {
	cfg       *config.Config
	store     store.Store
	presigner store.Presigner
}

func NewFileServer(cfg *config.Config, st store.Store) *FileServer {
	f := &FileServer{
		cfg:   cfg,
		store: st,
	}
	if p, ok := st.(store.Presigner); ok {
		f.presigner = p
	}
	return f
}

func (f *FileServer) presignEnabled() bool {
	return f.cfg.Store.S3.PresignDownload && f.presigner != nil
}

func (f *FileServer) presignURL(ctx context.Context, key string) (string, error) {
	return f.presigner.PresignDownloadURL(ctx, key, f.cfg.Store.S3.PresignExpiry)
}

func (f *FileServer) Setup(e *echo.Echo) error {
	e.POST("/upload", f.uploadFileHandlerByForm)
	e.PUT("/upload", f.uploadFileHandlerByStream)
	e.GET("/download", f.downloadHandler)
	e.GET("/download/*", f.downloadHandler)
	e.DELETE("/delete/*", f.deleteFileHandler)
	e.GET("/archive/*", f.archiveHandler)
	e.GET("/script/download", f.scriptHandler)
	e.GET("/api/info", f.infoHandler)
	e.GET("/api/files", f.listHandler)
	e.GET("/api/files/search", f.searchHandler)
	return nil
}

func (f *FileServer) uploadFileHandlerByForm(c echo.Context) error {
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

func (f *FileServer) uploadFileHandlerByStream(c echo.Context) error {
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

func (f *FileServer) saveFile(file io.ReadCloser, filename string, c echo.Context) error {
	// Generate unique filename using timestamp and original filename
	newFileName := strings.ReplaceAll(GetTimeStamp(), "-", "") + "-" + filename

	// Create directory structure based on current year and month
	now := time.Now()
	yearMonthPath := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	filePath := fmt.Sprintf("%s/%s", yearMonthPath, newFileName)

	// Upload to Store
	err := f.store.UploadFile(c.Request().Context(), file, filePath)
	if err != nil {
		c.Logger().Errorf("Error uploading the file %s to store: %s", filename, err.Error())
		return c.String(http.StatusInternalServerError, "Error uploading the file to the store")
	}

	c.Logger().Printf("File %s uploaded successfully to store: %s", filename, filePath)

	// presign 开启时直接返回预签名直链，服务器不再代理下载
	if f.presignEnabled() {
		downloadUrl, err := f.presignURL(c.Request().Context(), filePath)
		if err == nil {
			return c.String(http.StatusOK, fmt.Sprintf(`
File uploaded successfully.

Download command (presigned url, valid for %s):
	wget %s -O %s
`, f.cfg.Store.S3.PresignExpiry, downloadUrl, escapeFileName(filename)))
		}
		c.Logger().Errorf("Error presigning the file %s: %s, fallback to proxy url", filePath, err.Error())
	}

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

func (f *FileServer) downloadHandler(c echo.Context) error {
	file := strings.Trim(c.Param("*"), "/")

	c.Logger().Printf("Download file: %s", file)

	meta, err := f.store.FileMeta(c.Request().Context(), file)
	if err != nil {
		c.Logger().Errorf("Error checking the file %s: %s", file, err.Error())
		return jsonError(c, http.StatusInternalServerError, "Error checking the file")
	}

	// 目录或不存在：交给前端文件浏览页（SPA 回退统一处理）
	if meta == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// presign 开启时 302 跳转直链，数据不再经服务器转发
	if f.presignEnabled() {
		presigned, err := f.presignURL(c.Request().Context(), file)
		if err != nil {
			c.Logger().Errorf("Error presigning the file %s: %s, fallback to proxy", file, err.Error())
		} else {
			return c.Redirect(http.StatusFound, presigned)
		}
	}

	// Set headers
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(file)))
	c.Response().Header().Set("Content-Type", "application/octet-stream")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	c.Response().WriteHeader(http.StatusOK)

	// Stream the file
	err = f.store.DownloadFile(c.Request().Context(), c.Response().Writer, file)
	if err != nil {
		c.Logger().Errorf("Error downloading the file %s: %s", file, err.Error())
		return err
	}

	return nil
}

func (f *FileServer) deleteFileHandler(c echo.Context) error {
	file := strings.TrimPrefix(c.Param("*"), "/")

	c.Logger().Printf("Delete file: %s", file)

	err := f.store.DeleteFile(c.Request().Context(), file)
	if err != nil {
		c.Logger().Errorf("Error deleting the file %s: %s", file, err.Error())
		return c.String(http.StatusInternalServerError, "Error deleting the file")
	}

	c.Logger().Printf("File %s deleted successfully", file)

	return c.String(http.StatusOK, "File deleted successfully")
}

// ---- JSON API ----

type fileEntry struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
	Href  string `json:"href,omitempty"`
}

func (f *FileServer) infoHandler(c echo.Context) error {
	host := c.Request().Host
	return c.JSON(http.StatusOK, map[string]any{
		"uploadAddress":   getUploadAddress(host, f.cfg.EnableTls),
		"downloadAddress": getDownloadUrl(host, "[year]/[month]/[file_name]", f.cfg.EnableTls),
		"presignEnabled":  f.presignEnabled(),
	})
}

func (f *FileServer) listHandler(c echo.Context) error {
	dir := strings.Trim(c.QueryParam("path"), "/")
	cursor := c.QueryParam("cursor")
	size := queryInt(c, "size", defaultPageSize, 1, maxPageSize)

	entries, next, err := f.store.ListPage(c.Request().Context(), dir, cursor, size)
	if err != nil {
		c.Logger().Errorf("Error listing %s: %s", dir, err.Error())
		return jsonError(c, http.StatusInternalServerError, "Error listing the directory")
	}

	items := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		item := fileEntry{Name: e.Name, Size: e.Size, IsDir: e.IsDir}
		if !e.IsDir {
			item.Href = f.fileHref(c, joinKey(dir, e.Name))
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"path":       dir,
		"entries":    items,
		"nextCursor": next,
	})
}

type searchResult struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Href string `json:"href,omitempty"`
}

func (f *FileServer) searchHandler(c echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	if q == "" {
		return jsonError(c, http.StatusBadRequest, "Query is empty")
	}
	dir := strings.Trim(c.QueryParam("path"), "/")
	limit := queryInt(c, "limit", defaultSearchLim, 1, maxSearchLimit)

	terms := strings.Fields(strings.ToLower(q))

	var results []searchResult
	truncated := false
	err := f.store.Walk(c.Request().Context(), dir, func(m *store.FileMeta) error {
		lower := strings.ToLower(m.Name)
		for _, t := range terms {
			if !strings.Contains(lower, t) {
				return nil
			}
		}
		results = append(results, searchResult{Path: m.Name, Size: m.Size, Href: f.fileHref(c, m.Name)})
		if len(results) >= limit {
			truncated = true
			return store.ErrWalkStop
		}
		return nil
	})
	if err != nil {
		c.Logger().Errorf("Error searching %s under %s: %s", q, dir, err.Error())
		return jsonError(c, http.StatusInternalServerError, "Error searching files")
	}

	if results == nil {
		results = []searchResult{}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"path":      dir,
		"results":   results,
		"truncated": truncated,
	})
}

// ---- 批量下载 ----

// archiveHandler 流式打包目录为 zip，全程不整体缓冲
func (f *FileServer) archiveHandler(c echo.Context) error {
	dir := strings.Trim(c.Param("*"), "/")

	c.Logger().Printf("Archive directory: %s", dir)

	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", archiveName(dir)))
	c.Response().WriteHeader(http.StatusOK)

	zw := zip.NewWriter(c.Response().Writer)
	count := 0
	err := f.store.Walk(c.Request().Context(), dir, func(m *store.FileMeta) error {
		if count >= maxArchiveFiles {
			c.Logger().Errorf("Archive aborted: file count exceeds %d under %s", maxArchiveFiles, dir)
			return store.ErrWalkStop
		}
		hdr := &zip.FileHeader{
			Name:     m.Name,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if err := f.store.DownloadFile(c.Request().Context(), w, m.Name); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		c.Logger().Errorf("Error archiving %s: %s", dir, err.Error())
	}
	if closeErr := zw.Close(); closeErr != nil {
		c.Logger().Errorf("Error closing archive %s: %s", dir, closeErr.Error())
	}
	// 响应已开始输出，无法再更改状态码
	return nil
}

func (f *FileServer) scriptHandler(c echo.Context) error {
	dir := strings.Trim(c.QueryParam("path"), "/")
	osKind := c.QueryParam("os")
	if osKind == "" {
		osKind = "bash"
	}
	if osKind != "bash" && osKind != "powershell" {
		return jsonError(c, http.StatusBadRequest, "Unsupported os, expect bash or powershell")
	}

	files, truncated, err := f.collectScriptFiles(c, dir)
	if err != nil {
		c.Logger().Errorf("Error collecting files under %s: %s", dir, err.Error())
		return jsonError(c, http.StatusInternalServerError, "Error collecting the files")
	}

	var script string
	if osKind == "bash" {
		script = buildBashScript(dir, files, truncated, f.scriptNote())
	} else {
		script = buildPowerShellScript(dir, files, truncated, f.scriptNote())
	}

	if osKind == "bash" {
		c.Response().Header().Set(echo.HeaderContentType, "text/x-shellscript; charset=utf-8")
	} else {
		c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	}
	if c.QueryParam("download") == "1" {
		filename := "download.sh"
		if osKind == "powershell" {
			filename = "download.ps1"
		}
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))
	}
	return c.String(http.StatusOK, script)
}

type scriptFile struct {
	url    string
	target string
}

func (f *FileServer) collectScriptFiles(c echo.Context, dir string) ([]scriptFile, bool, error) {
	prefix := dirPrefix(dir)
	var files []scriptFile
	truncated := false
	err := f.store.Walk(c.Request().Context(), dir, func(m *store.FileMeta) error {
		if len(files) >= maxScriptFiles {
			truncated = true
			return store.ErrWalkStop
		}
		rel := strings.TrimPrefix(m.Name, prefix)
		files = append(files, scriptFile{
			url:    f.fileHref(c, m.Name),
			target: filepath.ToSlash(filepath.Join(prefix, rel)),
		})
		return nil
	})
	return files, truncated, err
}

// fileHref 返回文件的下载地址：presign 开启时为预签名直链，否则为代理地址
func (f *FileServer) fileHref(c echo.Context, key string) string {
	if f.presignEnabled() {
		href, err := f.presignURL(c.Request().Context(), key)
		if err == nil {
			return href
		}
		c.Logger().Errorf("Error presigning the file %s: %s, fallback to proxy url", key, err.Error())
	}
	return getDownloadUrl(c.Request().Host, EscapeUrlPath(key), f.cfg.EnableTls)
}

func (f *FileServer) scriptNote() string {
	if f.presignEnabled() {
		return fmt.Sprintf("链接为预签名直链（不经过本服务器），有效期 %s，请尽快执行", f.cfg.Store.S3.PresignExpiry)
	}
	return "文件由服务器转发下载"
}

// ---- helpers ----

func getUploadAddress(host string, enableTls bool) string {
	return getUrl(host, "upload", enableTls)
}

func getDownloadUrl(host, filePath string, enableTls bool) string {
	return getUrl(host, filepath.Join(downloadPath, strings.TrimPrefix(filePath, "/")), enableTls)
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

func queryInt(c echo.Context, name string, def, min, max int) int {
	v := c.QueryParam(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func joinKey(dir, name string) string {
	dir = strings.Trim(dir, "/")
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func dirPrefix(dir string) string {
	dir = strings.TrimPrefix(filepath.ToSlash(dir), "/")
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

func archiveName(dir string) string {
	base := filepath.Base(strings.Trim(dir, "/"))
	if base == "" || base == "." || base == "/" {
		base = "files"
	}
	return base + ".zip"
}
