package pkg

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/graydovee/fileManager/pkg/config"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNewHttpServer(t *testing.T) {
	if err := os.Chdir("../"); err != nil {
		t.Fatal(err)
	}

	flag.Parse()

	gin.SetMode(gin.TestMode)
	server, err := NewHttpServer(config.GetDefault())
	if err != nil {
		t.Fatal(err)
	}

	// 创建测试服务器
	ts := httptest.NewServer(server.engine)
	defer ts.Close()

	// 创建 HTTP 客户端
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 2 {
				return http.ErrUseLastResponse
			}
			t.Log("Redirect to:", req.URL)
			return nil
		},
	}

	// 测试 GET "/"
	resp, err := client.Get(ts.URL + "/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	filContent := []byte("This is a test file content")

	// 测试 Post "/upload"
	// 模拟文件上传
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "testfile.txt")
	assert.NoError(t, err)

	_, err = part.Write(filContent)
	assert.NoError(t, err)
	writer.Close()

	req, err := http.NewRequest("POST", ts.URL+"/upload", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	re := regexp.MustCompile(`(?i)http[s]?://[^\s]+`)
	url := re.FindString(string(data))
	downloadFilePath := url[strings.Index(url, "/download")+len("/download"):]

	// 测试 GET "/download/*file"
	resp, err = client.Get(ts.URL + filepath.Join("/download", downloadFilePath))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证下载的文件内容
	content, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, filContent, content)
	resp.Body.Close()

	// 测试 DELETE "/delete/*file"
	req, err = http.NewRequest("DELETE", ts.URL+filepath.Join("/delete", downloadFilePath), nil)
	assert.NoError(t, err)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证文件已被删除
	resp, err = client.Get(ts.URL + filepath.Join("/download", downloadFilePath))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

}
