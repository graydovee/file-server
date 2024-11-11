package store

import (
	"context"
	"io"
)

type Store interface {
	UploadFile(ctx context.Context, reader io.Reader, filePath string) error
	DeleteFile(ctx context.Context, filePath string) error
	FileExists(ctx context.Context, file string) (bool, error)
	List(ctx context.Context, dir string) ([]string, []string, error)
	DownloadFile(ctx context.Context, writer io.Writer, key string) error
}
