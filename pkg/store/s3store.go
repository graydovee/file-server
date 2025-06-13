package store

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	fconfig "github.com/graydovee/fileManager/pkg/config"
)

var _ Store = (*S3Store)(nil)

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
	state, err := s.FileMeta(ctx, filePath)
	if err != nil {
		return err
	}

	if state == nil {
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

func (s *S3Store) FileMeta(ctx context.Context, file string) (*FileMeta, error) {
	if file == "" {
		// if file is empty, we consider it in root directory
		return nil, nil
	}

	head, err := s.getHead(ctx, file)

	if err != nil {
		var notFoundErr *s3types.NotFound
		if errors.As(err, &notFoundErr) {
			return nil, nil
		}
		return nil, err
	}

	meta := &FileMeta{
		Name: filepath.Base(file),
		Size: *head.ContentLength,
	}

	return meta, nil
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

// List lists all the directories and files in the given directory.
func (s *S3Store) List(ctx context.Context, dir string) ([]*FileMeta, error) {
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	dir = strings.TrimPrefix(dir, "/")

	var metas []*FileMeta
	var continuationToken *string

	for {
		objects, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.cfg.Bucket),
			Prefix:            aws.String(dir),
			Delimiter:         aws.String("/"),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %v", err)
		}

		for _, obj := range objects.CommonPrefixes {
			metas = append(metas, &FileMeta{
				Name:  strings.TrimPrefix(*obj.Prefix, dir),
				IsDir: true,
			})
		}

		for _, obj := range objects.Contents {
			metas = append(metas, &FileMeta{
				Name: strings.TrimPrefix(*obj.Key, dir),
				Size: *obj.Size,
			})
		}

		if objects.IsTruncated == nil || !*objects.IsTruncated {
			break
		}
		continuationToken = objects.NextContinuationToken
	}

	return metas, nil
}

func (s *S3Store) getHead(ctx context.Context, file string) (*s3.HeadObjectOutput, error) {
	return s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &file,
	})
}
