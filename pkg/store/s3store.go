package store

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	fconfig "github.com/graydovee/fileManager/pkg/config"
	"io"
	"net/http"
	"sync"
)

type S3Store struct {
	cfg *fconfig.S3StoreConfig

	s3Client   *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

func NewS3Store(cfg *fconfig.S3StoreConfig) (*S3Store, error) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.DisableSSL}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"),
		config.WithHTTPClient(&http.Client{Transport: customTransport}),
		config.WithBaseEndpoint(cfg.Endpoint),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)

	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = !cfg.DisablePathStyle
	})

	return &S3Store{
		cfg:        cfg,
		s3Client:   s3Client,
		uploader:   manager.NewUploader(s3Client),
		downloader: manager.NewDownloader(s3Client),
	}, nil
}

func (s *S3Store) UploadFile(ctx context.Context, reader io.Reader, filePath string) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &filePath,
		Body:   reader,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file %s: %v", filePath, err)
	}

	return nil
}

func (s *S3Store) DeleteFile(ctx context.Context, filePath string) error {
	exists, err := s.FileExists(ctx, filePath)
	if err != nil {
		return err
	}

	if !exists {
		// file is already deleted
		return nil
	}

	_, err = s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &filePath,
	})
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %v", filePath, err)
	}

	return nil
}

func (s *S3Store) FileExists(ctx context.Context, file string) (bool, error) {
	_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &file,
	})

	if err != nil {
		var notFoundErr *s3types.NotFound
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

type writerAtAdapter struct {
	w      io.Writer
	mu     sync.Mutex
	offset int64
}

func (a *writerAtAdapter) WriteAt(p []byte, off int64) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if off != a.offset {
		return 0, fmt.Errorf("non-sequential write at offset %d (expected %d)", off, a.offset)
	}

	n, err = a.w.Write(p)
	a.offset += int64(n)
	return
}

func (s *S3Store) DownloadFile(ctx context.Context, writer io.Writer, key string) error {
	wa := &writerAtAdapter{w: writer}
	_, err := s.downloader.Download(ctx, wa, &s3.GetObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}, func(downloader *manager.Downloader) {
		// net.Conn not support concurrent write
		// so we set concurrency to 1
		downloader.Concurrency = 1
	})

	if err != nil {
		return fmt.Errorf("failed to download file %s: %v", key, err)
	}

	return nil
}
