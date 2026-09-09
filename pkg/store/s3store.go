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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	fconfig "github.com/graydovee/fileManager/pkg/config"
)

var _ Store = (*S3Store)(nil)
var _ Presigner = (*S3Store)(nil)

type S3Store struct {
	cfg *fconfig.S3StoreConfig

	s3Client      *s3.Client
	presignClient *s3.PresignClient
	uploader      *manager.Uploader
	downloader    *manager.Downloader
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

	newClient := func(endpoint string) *s3.Client {
		return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if endpoint != "" {
				o.BaseEndpoint = aws.String(endpoint)
			}
			o.UsePathStyle = !cfg.DisablePathStyle
		})
	}

	s3Client := newClient("")

	// presign 链接的 host 取决于签发时的 endpoint，集群内网地址签出的链接外部不可达，
	// 因此允许单独为 presign 指定公网入口
	var presignClient *s3.PresignClient
	if cfg.PresignEndpoint != "" {
		presignClient = s3.NewPresignClient(newClient(cfg.PresignEndpoint))
	} else {
		presignClient = s3.NewPresignClient(s3Client)
	}

	return &S3Store{
		cfg:           cfg,
		s3Client:      s3Client,
		presignClient: presignClient,
		uploader:      manager.NewUploader(s3Client),
		downloader:    manager.NewDownloader(s3Client),
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
		Size: aws.ToInt64(head.ContentLength),
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

// PresignDownloadURL 生成带 attachment 行为的预签名下载直链（纯本地签名，无网络请求）
func (s *S3Store) PresignDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	key = strings.TrimPrefix(key, "/")
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.cfg.Bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(fmt.Sprintf("attachment; filename=%q", filepath.Base(key))),
	}, func(o *s3.PresignOptions) {
		o.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign %s: %v", key, err)
	}
	return req.URL, nil
}

// ListPage 单层分页列出目录下的文件与子目录，cursor 为 S3 ContinuationToken
func (s *S3Store) ListPage(ctx context.Context, dir, cursor string, size int) ([]*FileMeta, string, error) {
	prefix := dirPrefix(dir)

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.cfg.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(int32(size)),
	}
	if cursor != "" {
		input.ContinuationToken = aws.String(cursor)
	}

	objects, err := s.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list objects: %v", err)
	}

	var entries []*FileMeta
	for _, obj := range objects.CommonPrefixes {
		entries = append(entries, &FileMeta{
			Name:  strings.TrimSuffix(strings.TrimPrefix(aws.ToString(obj.Prefix), prefix), "/"),
			IsDir: true,
		})
	}
	for _, obj := range objects.Contents {
		entries = append(entries, &FileMeta{
			Name: strings.TrimPrefix(aws.ToString(obj.Key), prefix),
			Size: aws.ToInt64(obj.Size),
		})
	}

	next := ""
	if aws.ToBool(objects.IsTruncated) && objects.NextContinuationToken != nil {
		next = aws.ToString(objects.NextContinuationToken)
	}
	return entries, next, nil
}

// Walk 递归遍历 prefix 下的所有文件，回调返回 ErrWalkStop 提前终止
func (s *S3Store) Walk(ctx context.Context, prefix string, fn func(meta *FileMeta) error) error {
	prefix = strings.TrimPrefix(prefix, "/")
	paginator := s3.NewListObjectsV2Paginator(s.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.cfg.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %v", err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if key == "" || strings.HasSuffix(key, "/") {
				continue
			}
			if err := fn(&FileMeta{Name: key, Size: aws.ToInt64(obj.Size)}); err != nil {
				if errors.Is(err, ErrWalkStop) {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

func (s *S3Store) getHead(ctx context.Context, file string) (*s3.HeadObjectOutput, error) {
	return s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &file,
	})
}

func dirPrefix(dir string) string {
	dir = strings.TrimPrefix(filepath.ToSlash(dir), "/")
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}
