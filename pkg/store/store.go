package store

import (
	"context"
	"errors"
	"io"
	"time"
)

type FileMeta struct {
	Name  string
	Size  int64
	IsDir bool
}

// ErrWalkStop 由 Walk 的回调返回，表示提前终止遍历且不视为错误
var ErrWalkStop = errors.New("walk stop")

type Store interface {
	UploadFile(ctx context.Context, reader io.Reader, filePath string) error

	DeleteFile(ctx context.Context, filePath string) error

	// FileMeta 只支持文件 stat，目录视为不存在（返回 nil）；文件不存在也返回 nil
	FileMeta(ctx context.Context, file string) (*FileMeta, error)

	// ListPage 单层分页列出目录下的文件与子目录。
	// cursor 为上一页返回的 nextCursor（首页传空），返回的 nextCursor 为空表示没有更多数据。
	ListPage(ctx context.Context, dir, cursor string, size int) (entries []*FileMeta, nextCursor string, err error)

	// Walk 递归遍历 prefix 下的所有文件（不含目录）。
	// 回调拿到的 FileMeta.Name 是相对存储根的完整路径；回调返回 ErrWalkStop 时提前终止且不视为错误。
	Walk(ctx context.Context, prefix string, fn func(meta *FileMeta) error) error

	DownloadFile(ctx context.Context, writer io.Writer, key string) error
}

// Presigner 由支持预签名直链下载的存储实现（S3/OSS）
type Presigner interface {
	// PresignDownloadURL 为单个文件生成带 attachment 行为的预签名下载直链
	PresignDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
