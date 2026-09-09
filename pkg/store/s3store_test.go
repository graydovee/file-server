package store

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newS3IntegrationStore 连接真实 S3 的集成测试入口，未配置时跳过
func newS3IntegrationStore(t *testing.T) *S3Store {
	t.Helper()
	cfg := config.GetDefault("../../.env")
	if cfg.Store.Type != config.StoreTypeS3 || cfg.Store.S3.Endpoint == "" {
		t.Skip("S3 未配置（STORE_TYPE != s3），跳过集成测试")
	}
	st, err := NewS3Store(&cfg.Store.S3)
	require.NoError(t, err)
	return st
}

func TestS3StoreRoundtrip(t *testing.T) {
	st := newS3IntegrationStore(t)
	ctx := context.Background()

	err := st.UploadFile(ctx, bytes.NewBufferString("test"), "test/test1.txt")
	require.NoError(t, err)

	meta, err := st.FileMeta(ctx, "test/test1.txt")
	require.NoError(t, err)
	require.NotNil(t, meta, "file should exist")
	assert.Equal(t, int64(4), meta.Size)

	// 目录视为不存在
	meta, err = st.FileMeta(ctx, "test")
	require.NoError(t, err)
	assert.Nil(t, meta)

	require.NoError(t, st.DeleteFile(ctx, "test/test1.txt"))
	meta, err = st.FileMeta(ctx, "test/test1.txt")
	require.NoError(t, err)
	assert.Nil(t, meta, "file should be deleted")
}

func TestS3StoreListPage(t *testing.T) {
	st := newS3IntegrationStore(t)
	ctx := context.Background()

	keys := []string{"test/list/b.txt", "test/list/a.txt", "test/list/sub/c.txt"}
	for _, k := range keys {
		require.NoError(t, st.UploadFile(ctx, bytes.NewBufferString(k), k))
	}
	t.Cleanup(func() {
		for _, k := range keys {
			_ = st.DeleteFile(context.Background(), k)
		}
	})

	entries, _, err := st.ListPage(ctx, "test/list", "", 2)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// 翻完整个目录
	var names []string
	cursor := ""
	for {
		page, nextCursor, err := st.ListPage(ctx, "test/list", cursor, 1)
		require.NoError(t, err)
		for _, e := range page {
			names = append(names, e.Name)
		}
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	assert.ElementsMatch(t, []string{"a.txt", "b.txt", "sub"}, names)
}

func TestS3PresignDownloadURL(t *testing.T) {
	// presign 是纯本地签名，不需要真实 S3，离线可测
	cfg := &config.S3StoreConfig{
		Endpoint:        "http://127.0.0.1:9000",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Bucket:          "test-bucket",
	}
	st, err := NewS3Store(cfg)
	require.NoError(t, err)

	rawURL, err := st.PresignDownloadURL(context.Background(), "2025/01/a b.txt", 10*time.Minute)
	require.NoError(t, err)

	u, err := url.Parse(rawURL)
	require.NoError(t, err)

	// 使用主 endpoint 签名
	assert.Equal(t, "127.0.0.1:9000", u.Host)
	// path style
	assert.True(t, strings.HasPrefix(u.Path, "/test-bucket/2025/01/"))
	// 签名参数与 attachment 行为
	q := u.Query()
	assert.NotEmpty(t, q.Get("X-Amz-Signature"))
	assert.NotEmpty(t, q.Get("X-Amz-Expires"))
	assert.Contains(t, q.Get("response-content-disposition"), "attachment")
}

func TestS3PresignDownloadURLWithCustomEndpoint(t *testing.T) {
	cfg := &config.S3StoreConfig{
		Endpoint:        "http://minio.internal.svc:9000",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Bucket:          "test-bucket",
		PresignEndpoint: "https://minio-public.example.com",
	}
	st, err := NewS3Store(cfg)
	require.NoError(t, err)

	rawURL, err := st.PresignDownloadURL(context.Background(), "2025/01/file.txt", time.Hour)
	require.NoError(t, err)

	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	// presign URL 指向公网入口，内部下载仍走主 endpoint
	assert.Equal(t, "minio-public.example.com", u.Host)
	assert.NotEmpty(t, u.Query().Get("X-Amz-Signature"))
}
