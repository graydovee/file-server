package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/graydovee/fileManager/pkg/config"
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
	stat, err := os.Stat(l.getFullFilePath(file))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check file: %w", err)
	}
	if stat.IsDir() {
		// 目录视为不存在，与接口约定一致
		return nil, nil
	}
	return &FileMeta{Name: filepath.Base(file), Size: stat.Size()}, nil
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

func (l *LocalStore) ListPage(ctx context.Context, dir, cursor string, size int) ([]*FileMeta, string, error) {
	stats, err := os.ReadDir(l.getFullFilePath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("failed to read directory: %w", err)
	}

	offset := 0
	if cursor != "" {
		offset, err = strconv.Atoi(cursor)
		if err != nil || offset < 0 {
			return nil, "", fmt.Errorf("invalid cursor: %s", cursor)
		}
	}

	var entries []*FileMeta
	next := ""
	for i := offset; i < len(stats); i++ {
		if len(entries) >= size {
			next = strconv.Itoa(i)
			break
		}
		stat := stats[i]
		f := &FileMeta{
			Name:  stat.Name(),
			IsDir: stat.IsDir(),
		}
		if !stat.IsDir() {
			info, err := stat.Info()
			if err != nil {
				return nil, "", fmt.Errorf("failed to get file info: %w", err)
			}
			f.Size = info.Size()
		}
		entries = append(entries, f)
	}
	return entries, next, nil
}

func (l *LocalStore) Walk(ctx context.Context, prefix string, fn func(meta *FileMeta) error) error {
	root := l.getFullFilePath(prefix)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 前缀目录不存在视为空
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(filepath.Join(filepath.ToSlash(prefix), rel))
		return fn(&FileMeta{Name: key, Size: info.Size()})
	})
	if errors.Is(err, ErrWalkStop) {
		return nil
	}
	return err
}

func (l *LocalStore) getFullFilePath(key string) string {
	return filepath.Join(l.cfg.UploadDir, key)
}
