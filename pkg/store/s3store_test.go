package store

import (
	"bytes"
	"context"
	"flag"
	"github.com/graydovee/fileManager/pkg/config"
	"testing"
)

func TestListObj(t *testing.T) {
	cfg := config.GetDefault("../../.env")

	flag.Parse()

	store, err := NewS3Store(&cfg.Store.S3)
	if err != nil {
		t.Fatal(err)
	}
	//store := NewLocalStore(&cfg.Store.Local)

	dirs, files, err := store.List(context.Background(), "/codex/xx")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(dirs)
	t.Log(files)
}

func TestS3Store(t *testing.T) {
	cfg := config.GetDefault("../../.env")

	flag.Parse()

	store, err := NewS3Store(&cfg.Store.S3)
	if err != nil {
		t.Fatal(err)
	}

	buffer := bytes.NewBuffer([]byte("test"))

	err = store.UploadFile(context.Background(), buffer, "test/test1.txt")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("test/test1.txt")

	exists, err := store.FileExists(context.Background(), "test/test1.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("file not exists")
	}

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
