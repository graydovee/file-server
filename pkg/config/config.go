package config

import (
	flag "github.com/spf13/pflag"
	"path/filepath"
)

type Config struct {
	Address   string
	EnableTls bool

	StaticDir string
	UploadDir string
}

var (
	defaultUploadDir = "./uploads"
	defaultStaticDir = "./static"
)

func (c *Config) InitFlags(f *flag.FlagSet) {
	f.StringVarP(&c.StaticDir, "static-dir", "s", defaultStaticDir, "static file directory")
	f.StringVarP(&c.UploadDir, "upload-dir", "u", defaultUploadDir, "file upload directory")
	f.StringVarP(&c.Address, "address", "a", ":8080", "server listen address")
	f.BoolVarP(&c.EnableTls, "tls", "t", false, "enable https")
}

func (c *Config) Build() error {
	absUploadDir, err := filepath.Abs(c.UploadDir)
	c.UploadDir = absUploadDir
	if err != nil {
		return err
	}
	return nil
}
