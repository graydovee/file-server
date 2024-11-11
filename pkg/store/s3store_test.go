package store

import (
	"bytes"
	"context"
	"flag"
	"github.com/graydovee/fileManager/pkg/config"
	"os"
	"testing"
	"time"
)

func TestS3Store(t *testing.T) {
	cfg := &config.S3StoreConfig{}

	flag.StringVar(&cfg.Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "s3 endpoint")
	flag.StringVar(&cfg.AccessKeyID, "s3-access-key-id", os.Getenv("S3_ACCESS_KEY_ID"), "s3 access key id")
	flag.StringVar(&cfg.SecretAccessKey, "s3-secret-access-key", os.Getenv("S3_SECRET_ACCESS_KEY"), "s3 secret access key")
	flag.StringVar(&cfg.Bucket, "s3-bucket", os.Getenv("S3_BUCKET"), "s3 bucket")
	flag.BoolVar(&cfg.DisablePathStyle, "s3-disable-path-style", false, "s3 disable path style")
	flag.BoolVar(&cfg.DisableSSL, "s3-disable-ssl", false, "s3 disable ssl")

	flag.Parse()

	store, err := NewS3Store(cfg)
	if err != nil {
		t.Fatal(err)
	}

	buffer := bytes.NewBuffer([]byte("test"))

	err := store.UploadFile(context.Background(), buffer, "test/test1.txt")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(file)

	exists, err := store.FileExists(context.Background(), "test/test1.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("file not exists")
	}

	objectURL, err := store.GeneratePrivateObjectDownloadURL("test/test1.txt", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(objectURL)

	err = store.DeleteFile(context.Background(), "test/test1.txt")
	if err != nil {
		t.Fatal(err)
	}
	exists, err = store.FileExists(context.Background(), "test/test1.txt")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("file exists")
	}
}
