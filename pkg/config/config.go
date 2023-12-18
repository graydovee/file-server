package config

import (
	"path/filepath"
)

type Config struct {
	Address   string
	EnableTls bool

	UploadDir   string
	StaticDir   string
	TemplateDir string
}

func (c *Config) Build() error {
	absUploadDir, err := filepath.Abs(c.UploadDir)
	c.UploadDir = absUploadDir
	if err != nil {
		return err
	}
	return nil
}
