package store

import (
	"context"
	"strings"
	"testing"

	"github.com/graydovee/fileManager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLocalStore(t *testing.T) *LocalStore {
	t.Helper()
	cfg := &config.LocalStoreConfig{UploadDir: t.TempDir()}
	require.NoError(t, cfg.Build())
	return NewLocalStore(cfg)
}

func TestLocalStoreListPage(t *testing.T) {
	ls := newTestLocalStore(t)
	ctx := context.Background()

	files := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	for _, name := range files {
		require.NoError(t, ls.UploadFile(ctx, strings.NewReader(name), "2025/09/"+name))
	}
	// 子目录
	require.NoError(t, ls.UploadFile(ctx, strings.NewReader("x"), "2025/09/sub/inner.txt"))

	// 第一页
	entries, next, err := ls.ListPage(ctx, "2025/09", "", 2)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "a.txt", entries[0].Name)
	assert.False(t, entries[0].IsDir)
	assert.Equal(t, int64(5), entries[0].Size)
	assert.NotEmpty(t, next)

	// 翻页直到没有更多
	var names []string
	cursor := ""
	for {
		page, nextCursor, err := ls.ListPage(ctx, "2025/09", cursor, 2)
		require.NoError(t, err)
		for _, e := range page {
			names = append(names, e.Name)
		}
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	assert.ElementsMatch(t, append(files, "sub"), names)

	// 目录不存在 → 空结果
	entries, next, err = ls.ListPage(ctx, "not-exist", "", 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Empty(t, next)

	// 非法 cursor
	_, _, err = ls.ListPage(ctx, "2025/09", "abc", 10)
	assert.Error(t, err)
}

func TestLocalStoreWalk(t *testing.T) {
	ls := newTestLocalStore(t)
	ctx := context.Background()

	require.NoError(t, ls.UploadFile(ctx, strings.NewReader("1"), "2025/01/a.txt"))
	require.NoError(t, ls.UploadFile(ctx, strings.NewReader("22"), "2025/01/sub/b.txt"))
	require.NoError(t, ls.UploadFile(ctx, strings.NewReader("333"), "2025/02/c.txt"))

	var keys []string
	err := ls.Walk(ctx, "", func(m *FileMeta) error {
		assert.False(t, m.IsDir)
		keys = append(keys, m.Name)
		return nil
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"2025/01/a.txt", "2025/01/sub/b.txt", "2025/02/c.txt"}, keys)

	// ErrWalkStop 提前终止且不报错
	var count int
	err = ls.Walk(ctx, "", func(m *FileMeta) error {
		count++
		if count >= 2 {
			return ErrWalkStop
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// 前缀不存在 → 空
	err = ls.Walk(ctx, "no-such-dir", func(m *FileMeta) error {
		t.Fatal("should not be called")
		return nil
	})
	require.NoError(t, err)
}

func TestLocalStoreFileMeta(t *testing.T) {
	ls := newTestLocalStore(t)
	ctx := context.Background()

	require.NoError(t, ls.UploadFile(ctx, strings.NewReader("hello"), "dir/file.txt"))

	// 文件存在
	meta, err := ls.FileMeta(ctx, "dir/file.txt")
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "file.txt", meta.Name)
	assert.Equal(t, int64(5), meta.Size)

	// 目录视为不存在
	meta, err = ls.FileMeta(ctx, "dir")
	require.NoError(t, err)
	assert.Nil(t, meta)

	// 不存在
	meta, err = ls.FileMeta(ctx, "no/such.txt")
	require.NoError(t, err)
	assert.Nil(t, meta)
}

func TestLocalStoreDelete(t *testing.T) {
	ls := newTestLocalStore(t)
	ctx := context.Background()

	require.NoError(t, ls.UploadFile(ctx, strings.NewReader("x"), "del.txt"))
	require.NoError(t, ls.DeleteFile(ctx, "del.txt"))

	meta, err := ls.FileMeta(ctx, "del.txt")
	require.NoError(t, err)
	assert.Nil(t, meta)

	// 删除不存在文件报错
	assert.Error(t, ls.DeleteFile(ctx, "del.txt"))
}
