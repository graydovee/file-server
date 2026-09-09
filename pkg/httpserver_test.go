package pkg

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/graydovee/fileManager/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer 构建基于本地存储 + 临时前端产物的测试服务
func newTestServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()

	distDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html>filemanager-spa</html>"), 0644))

	cfg := &config.Config{
		Address: ":0",
		Web:     config.WebConfig{DistDir: distDir},
		Store: config.StoreConfig{
			Type:  config.StoreTypeLocal,
			Local: config.LocalStoreConfig{UploadDir: t.TempDir()},
		},
	}
	require.NoError(t, cfg.Build())

	server, err := NewHttpServer(cfg)
	require.NoError(t, err)

	ts := httptest.NewServer(server.engine)
	t.Cleanup(ts.Close)
	return ts, cfg
}

// seedFile 直接往本地存储写入文件，绕过上传接口
func seedFile(t *testing.T, cfg *config.Config, key, content string) {
	t.Helper()
	ls := store.NewLocalStore(&cfg.Store.Local)
	require.NoError(t, ls.UploadFile(t.Context(), strings.NewReader(content), key))
}

func TestHealthWithoutDist(t *testing.T) {
	cfg := &config.Config{
		Store: config.StoreConfig{
			Type:  config.StoreTypeLocal,
			Local: config.LocalStoreConfig{UploadDir: t.TempDir()},
		},
	}
	require.NoError(t, cfg.Build())
	server, err := NewHttpServer(cfg)
	require.NoError(t, err)
	ts := httptest.NewServer(server.engine)
	defer ts.Close()

	// 前端未构建时根路径仍需 200（k8s 探针）
	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUploadAndDownloadRoundtrip(t *testing.T) {
	ts, _ := newTestServer(t)

	content := "This is a test file content"

	// PUT 流式上传
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/upload", strings.NewReader(content))
	require.NoError(t, err)
	req.Header.Set("X-Filename", "hello world.txt")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	re := regexp.MustCompile(`(?i)http[s]?://[^\s]+`)
	downloadUrl := re.FindString(string(body))
	require.NotEmpty(t, string(body))
	downloadPath := downloadUrl[strings.Index(downloadUrl, "/download")+len("/download"):]

	// 下载内容一致
	resp, err = http.Get(ts.URL + "/download" + downloadPath)
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, content, string(got))
	assert.True(t, strings.HasSuffix(downloadPath, "hello%20world.txt"))
}

func TestFileInfoAndDelete(t *testing.T) {
	ts, cfg := newTestServer(t)
	seedFile(t, cfg, "2025/01/a.txt", "aaa")

	// 文件元信息存在 → 下载
	resp, err := http.Get(ts.URL + "/download/2025/01/a.txt")
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "aaa", string(got))

	// 删除
	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/delete/2025/01/a.txt", nil)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 文件不存在 → SPA 回退（前端文件浏览页）
	resp, err = http.Get(ts.URL + "/download/2025/01/a.txt")
	require.NoError(t, err)
	got, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "<html>filemanager-spa</html>", string(got))
}

func TestApiFilesListPagination(t *testing.T) {
	ts, cfg := newTestServer(t)
	seedFile(t, cfg, "2025/01/a.txt", "a")
	seedFile(t, cfg, "2025/01/b.txt", "b")
	seedFile(t, cfg, "2025/01/c.txt", "c")

	var page struct {
		Path       string `json:"path"`
		Entries    []struct {
			Name  string `json:"name"`
			Size  int64  `json:"size"`
			IsDir bool   `json:"isDir"`
			Href  string `json:"href"`
		} `json:"entries"`
		NextCursor string `json:"nextCursor"`
	}

	// 第一页 size=2
	resp, err := http.Get(ts.URL + "/api/files?path=2025/01&size=2")
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	resp.Body.Close()
	assert.Equal(t, "2025/01", page.Path)
	require.Len(t, page.Entries, 2)
	assert.Equal(t, "a.txt", page.Entries[0].Name)
	assert.Equal(t, "b.txt", page.Entries[1].Name)
	assert.NotEmpty(t, page.NextCursor)
	assert.Contains(t, page.Entries[0].Href, "/download/2025/01/a.txt")

	// 第二页
	resp, err = http.Get(ts.URL + "/api/files?path=2025/01&size=2&cursor=" + page.NextCursor)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	resp.Body.Close()
	require.Len(t, page.Entries, 1)
	assert.Equal(t, "c.txt", page.Entries[0].Name)
	assert.Empty(t, page.NextCursor)
}

func TestApiFilesSearch(t *testing.T) {
	ts, cfg := newTestServer(t)
	seedFile(t, cfg, "2025/01/report-final.txt", "1")
	seedFile(t, cfg, "2025/02/report-draft.txt", "2")
	seedFile(t, cfg, "2025/02/photo.txt", "3")

	var result struct {
		Results []struct {
			Path string `json:"path"`
			Size int64  `json:"size"`
		} `json:"results"`
		Truncated bool `json:"truncated"`
	}

	// 大小写不敏感 + 多关键词 AND
	resp, err := http.Get(ts.URL + "/api/files/search?q=REPORT+draft")
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	resp.Body.Close()
	require.Len(t, result.Results, 1)
	assert.Equal(t, "2025/02/report-draft.txt", result.Results[0].Path)
	assert.False(t, result.Truncated)

	// limit 截断
	resp, err = http.Get(ts.URL + "/api/files/search?q=txt&limit=2")
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	resp.Body.Close()
	assert.Len(t, result.Results, 2)
	assert.True(t, result.Truncated)

	// 空关键词
	resp, err = http.Get(ts.URL + "/api/files/search?q=")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestDownloadDirectoryFallsBackToSpa(t *testing.T) {
	ts, cfg := newTestServer(t)
	seedFile(t, cfg, "2025/01/a.txt", "a")

	resp, err := http.Get(ts.URL + "/download/2025/01")
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "<html>filemanager-spa</html>", string(got))
}

func TestApiInfo(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/api/info")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var info map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))
	assert.Equal(t, false, info["presignEnabled"])
	assert.Contains(t, info["uploadAddress"], "/upload")
}

func TestApiCodeUploadAndShow(t *testing.T) {
	ts, _ := newTestServer(t)

	// JSON 上传
	payload := `{"code":"package main","language":"go"}`
	resp, err := http.Post(ts.URL+"/api/code", "application/json", strings.NewReader(payload))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var up struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(body, &up))
	require.Regexp(t, `^/code/go/\d{17}-[0-9a-f]{8}$`, up.URL)

	// 展示（注意展示路由是 /code/...，API 是 /api/code/...）
	resp, err = http.Get(ts.URL + "/api" + up.URL)
	require.NoError(t, err)
	var show struct {
		Language    string `json:"language"`
		Code        string `json:"code"`
		DownloadURL string `json:"downloadUrl"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&show))
	resp.Body.Close()
	assert.Equal(t, "go", show.Language)
	assert.Equal(t, "package main", show.Code)
	assert.True(t, strings.HasPrefix(show.DownloadURL, "/download/code/go/"))

	// downloadUrl 可直接下载到原内容
	resp, err = http.Get(ts.URL + show.DownloadURL)
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "package main", string(got))

	// 不存在的片段
	resp, err = http.Get(ts.URL + "/api/code/go/00000000-00000000")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// 不支持的语言
	resp, err = http.Post(ts.URL+"/api/code", "application/json", strings.NewReader(`{"code":"x","language":"cobol"}`))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestArchiveZip(t *testing.T) {
	ts, cfg := newTestServer(t)
	seedFile(t, cfg, "2025/01/a.txt", "aaa")
	seedFile(t, cfg, "2025/01/sub/b.txt", "bbb")

	resp, err := http.Get(ts.URL + "/archive/2025/01")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/zip", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "01.zip")

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	contents := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		c, err := io.ReadAll(rc)
		require.NoError(t, err)
		rc.Close()
		contents[f.Name] = string(c)
	}
	assert.Equal(t, map[string]string{
		"2025/01/a.txt":     "aaa",
		"2025/01/sub/b.txt": "bbb",
	}, contents)
}

func TestScriptDownload(t *testing.T) {
	ts, cfg := newTestServer(t)
	seedFile(t, cfg, "2025/01/a'b & c.txt", "weird")
	seedFile(t, cfg, "2025/01/plain.txt", "plain")

	// bash 脚本
	resp, err := http.Get(ts.URL + "/script/download?path=2025/01&os=bash")
	require.NoError(t, err)
	script, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/x-shellscript; charset=utf-8", resp.Header.Get("Content-Type"))
	s := string(script)
	assert.Contains(t, s, "#!/usr/bin/env bash")
	// 目标路径保留目录结构
	assert.Contains(t, s, "2025/01/plain.txt")
	// 特殊字符被单引号转义（' → '\''）
	assert.Contains(t, s, `2025/01/a'\''b & c.txt`)
	assert.Contains(t, s, "curl -fL")

	// powershell 脚本
	resp, err = http.Get(ts.URL + "/script/download?path=2025/01&os=powershell")
	require.NoError(t, err)
	script, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Contains(t, string(script), "Invoke-WebRequest")

	// download=1 返回附件头
	resp, err = http.Get(ts.URL + "/script/download?path=2025/01&os=bash&download=1")
	require.NoError(t, err)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "download.sh")

	// 非法 os
	resp, err = http.Get(ts.URL + "/script/download?os=cmd")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestFormUpload(t *testing.T) {
	ts, _ := newTestServer(t)

	content := "form upload content"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "form.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(respBody), "File uploaded successfully")

	// 非 multipart 拒绝
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/upload", strings.NewReader("x"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSpaFallbackForUnknownRoutes(t *testing.T) {
	ts, _ := newTestServer(t)

	// 前端路由 → index.html
	resp, err := http.Get(ts.URL + "/code/go/abcd1234-ef567890")
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "<html>filemanager-spa</html>", string(got))

	// API 路径不回落 → JSON 404
	resp, err = http.Get(ts.URL + "/api/not-exist")
	require.NoError(t, err)
	got, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(string(got)), "{"))

	// 缺失的静态资源不回落 → 404
	resp, err = http.Get(ts.URL + "/assets/missing.js")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
