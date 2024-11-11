package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Address string

	EnableTls bool

	InternalHost string

	Resource ResourceConfig

	Store StoreConfig
}

type StoreConfig struct {
	Type  string
	S3    S3StoreConfig
	Local LocalStoreConfig
}

type ResourceConfig struct {
	StaticDir   string
	TemplateDir string
}

type LocalStoreConfig struct {
	UploadDir string
}

type S3StoreConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string

	DisablePathStyle bool
	DisableSSL       bool
}

const (
	StoreTypeLocal = "local"
	StoreTypeS3    = "s3"
)

func (l *LocalStoreConfig) Build() error {
	absUploadDir, err := filepath.Abs(l.UploadDir)
	l.UploadDir = absUploadDir
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) Build() error {
	if c.Store.Type == StoreTypeLocal {
		if err := c.Store.Local.Build(); err != nil {
			return err
		}
	}
	return nil
}

const (
	defaultUploadDir   = "./uploads"
	defaultStaticDir   = "./assert"
	defaultTemplateDir = "./template"
)

var defaultConfigLoader sync.Once
var defaultConfig Config

func GetDefault(envFileName ...string) *Config {
	defaultConfigLoader.Do(func() {
		if err := godotenv.Load(envFileName...); err != nil {
			log.Panicln("Error loading .env file")
		}
		defaultConfig = Config{
			Address:   GetEnvOrDefault("SERVER_LISTEN_ADDRESS", ":8080"),
			EnableTls: EnvExist("SERVER_ENABLE_TLS"),
			Resource: ResourceConfig{
				StaticDir:   GetEnvOrDefault("RESOURCE_STATIC_DIR", defaultStaticDir),
				TemplateDir: GetEnvOrDefault("RESOURCE_TEMPLATE_DIR", defaultTemplateDir),
			},
			Store: StoreConfig{
				Type: os.Getenv("STORE_TYPE"),
				Local: LocalStoreConfig{
					UploadDir: GetEnvOrDefault("STORE_LOCAL_UPLOAD_DIR", defaultUploadDir),
				},
				S3: S3StoreConfig{
					Endpoint:         GetEnvOrDefault("STORE_S3_ENDPOINT"),
					AccessKeyID:      GetEnvOrDefault("STORE_S3_ACCESS_KEY_ID"),
					SecretAccessKey:  GetEnvOrDefault("STORE_S3_SECRET_ACCESS_KEY"),
					Bucket:           GetEnvOrDefault("STORE_S3_BUCKET"),
					DisablePathStyle: EnvExist("STORE_S3_DISABLE_PATH_STYLE"),
					DisableSSL:       EnvExist("STORE_S3_DISABLE_SSL"),
				},
			},
		}
	})
	return &defaultConfig
}

func EnvExist(envKey string) bool {
	return os.Getenv(envKey) != ""
}

func GetEnvOrDefault(envKey string, defaultValue ...string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (c *Config) RegisterFlags(f *pflag.FlagSet) {
}
