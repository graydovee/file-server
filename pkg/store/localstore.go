package store

import (
	"context"
	"fmt"
	"github.com/graydovee/fileManager/pkg/config"
	"io"
	"os"
	"path/filepath"
)

var _ Store = (*LocalStore)(nil)

type LocalStore struct {
	cfg *config.LocalStoreConfig
}

func NewLocalStore(cfg *config.LocalStoreConfig) *LocalStore {
	return &LocalStore{cfg: cfg}
}

func (l *LocalStore) UploadFile(ctx context.Context, reader io.Reader, filePath string) error {
	fullFilePath := l.getFullFilePath(filePath)

	// Create new file
	dir := filepath.Dir(fullFilePath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	newFile, err := os.Create(fullFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer newFile.Close()

	// Copy the uploaded file to the new file
	_, err = io.Copy(newFile, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func (l *LocalStore) DeleteFile(ctx context.Context, filePath string) error {
	err := os.Remove(l.getFullFilePath(filePath))
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (l *LocalStore) FileMeta(ctx context.Context, file string) (*FileMeta, error) {
	var meta FileMeta
	stat, err := os.Stat(l.getFullFilePath(file))
	meta.Name = filepath.Base(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check file: %w", err)
	}
	meta.Size = stat.Size()
	return &meta, nil
}

func (l *LocalStore) DownloadFile(ctx context.Context, writer io.Writer, key string) error {
	fullFilePath := l.getFullFilePath(key)
	file, err := os.Open(fullFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func (l *LocalStore) List(ctx context.Context, dir string) ([]*FileMeta, error) {
	l.getFullFilePath(dir)
	stats, err := os.ReadDir(l.getFullFilePath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []*FileMeta
	for _, stat := range stats {
		f := &FileMeta{
			Name:  stat.Name(),
			IsDir: stat.IsDir(),
		}
		if !stat.IsDir() {
			info, err := stat.Info()
			if err != nil {
				return nil, fmt.Errorf("failed to get file info: %w", err)
			}
			f.Size = info.Size()
		}
	}
	return files, nil
}

func (l *LocalStore) getFullFilePath(key string) string {
	return filepath.Join(l.cfg.UploadDir, key)
}
