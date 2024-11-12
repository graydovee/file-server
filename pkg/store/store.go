package store

import (
	"context"
	"io"
)

type FileMeta struct {
	Name  string
	Size  int64
	IsDir bool
}

type Store interface {
	UploadFile(ctx context.Context, reader io.Reader, filePath string) error

	DeleteFile(ctx context.Context, filePath string) error

	// FileMeta Only support file stat, if it is a directory, consider it as not exist
	// return nil if file not exist
	FileMeta(ctx context.Context, file string) (*FileMeta, error)

	// List all files or directories under the directory
	List(ctx context.Context, dir string) ([]*FileMeta, error)

	DownloadFile(ctx context.Context, writer io.Writer, key string) error
}
