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

func (l *LocalStore) FileExists(ctx context.Context, file string) (bool, error) {
	stat, err := os.Stat(l.getFullFilePath(file))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file: %w", err)
	}
	return !stat.IsDir(), nil
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

func (l *LocalStore) List(ctx context.Context, dir string) ([]string, []string, error) {
	l.getFullFilePath(dir)
	stats, err := os.ReadDir(l.getFullFilePath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []string
	var files []string
	for _, stat := range stats {
		if stat.IsDir() {
			dirs = append(dirs, stat.Name())
		} else {
			files = append(files, stat.Name())
		}
	}
	return dirs, files, nil
}

func (l *LocalStore) getFullFilePath(key string) string {
	return filepath.Join(l.cfg.UploadDir, key)
}
