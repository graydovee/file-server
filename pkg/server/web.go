package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// webHandler 提供前端构建产物的静态资源服务与 SPA 回退：
// 未匹配路由的 GET 请求优先尝试 dist 内同名文件（favicon 等），
// 否则回落 index.html，由前端路由接管（如 /download/2025/09、/code/go/xxx）
type webHandler struct {
	distDir string
	// defaultErrorHandler 保存 echo 原生错误处理，供非 404 场景兜底
	defaultErrorHandler echo.HTTPErrorHandler
}

func NewWebHandler(distDir string) *webHandler {
	return &webHandler{distDir: distDir}
}

// Register 必须在所有业务路由之后调用
func (w *webHandler) Register(e *echo.Echo) {
	e.GET("/assets/*", func(c echo.Context) error {
		if w.serveDistFile(c, path.Join("assets", c.Param("*"))) {
			return nil
		}
		return echo.NewHTTPError(http.StatusNotFound)
	})
	w.defaultErrorHandler = e.DefaultHTTPErrorHandler
	e.HTTPErrorHandler = w.errorHandler
}

func (w *webHandler) errorHandler(err error, c echo.Context) {
	if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusNotFound {
		if c.Request().Method == http.MethodGet && !w.isReservedPath(c.Request().URL.Path) {
			if w.serveDistFile(c, c.Request().URL.Path) {
				return
			}
			if w.serveIndex(c) {
				return
			}
			// 前端未构建时保证根路径探针可用
			if c.Request().URL.Path == "/" {
				c.String(http.StatusOK, "fileserver is running (web ui not built)\n")
				return
			}
		}
	}
	echo := w.defaultErrorHandler
	if echo == nil {
		// Register 未被调用时的保险兜底
		c.Logger().Errorf("unhandled error: %v", err)
		return
	}
	echo(err, c)
}

// isReservedPath API 与静态资源类路径不回落前端页面，保持 JSON 404
func (w *webHandler) isReservedPath(p string) bool {
	return strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/script/") || strings.HasPrefix(p, "/assets/")
}

func (w *webHandler) serveDistFile(c echo.Context, p string) bool {
	p = strings.TrimPrefix(path.Clean("/"+p), "/")
	if p == "" || p == "." {
		return false
	}
	fp := filepath.Join(w.distDir, filepath.FromSlash(p))
	fi, err := os.Stat(fp)
	if err != nil || fi.IsDir() {
		return false
	}
	c.File(fp)
	return true
}

func (w *webHandler) serveIndex(c echo.Context) bool {
	indexPath := filepath.Join(w.distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}
	c.File(indexPath)
	return true
}
