package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Address string

	EnableTls bool

	InternalHost string

	Web WebConfig

	Store StoreConfig
}

type WebConfig struct {
	// DistDir 为前端构建产物目录，提供静态资源与 SPA 回退
	DistDir string
}

type StoreConfig struct {
	Type  string
	S3    S3StoreConfig
	Local LocalStoreConfig
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

	// PresignDownload 开启后下载返回预签名直链（仅 S3/OSS 存储生效）
	PresignDownload bool
	// PresignExpiry 预签名链接有效期
	PresignExpiry time.Duration
	// PresignEndpoint 可选：预签名使用的独立 endpoint（如公网入口），
	// 不填则使用 S3 Endpoint 本身
	PresignEndpoint string
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
	defaultWebDistDir  = "./web/dist"
	defaultPresignExpy = time.Hour
)

var defaultConfigLoader sync.Once
var defaultConfig Config

func GetDefault(envFileName ...string) *Config {
	defaultConfigLoader.Do(func() {
		if err := godotenv.Load(envFileName...); err != nil {
			// ignore error
		}
		defaultConfig = Config{
			Address:      GetEnvOrDefault("SERVER_LISTEN_ADDRESS", ":8080"),
			EnableTls:    EnvExist("SERVER_ENABLE_TLS"),
			InternalHost: GetEnvOrDefault("INTERNAL_HOST", "127.0.0.1"),
			Web: WebConfig{
				DistDir: GetEnvOrDefault("WEB_DIST_DIR", defaultWebDistDir),
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
					PresignDownload:  EnvExist("STORE_S3_PRESIGN_DOWNLOAD"),
					PresignExpiry:    parseDurationOrDefault(GetEnvOrDefault("STORE_S3_PRESIGN_EXPIRY", defaultPresignExpy.String()), defaultPresignExpy),
					PresignEndpoint:  GetEnvOrDefault("STORE_S3_PRESIGN_ENDPOINT"),
				},
			},
		}
	})
	return &defaultConfig
}

func parseDurationOrDefault(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return def
	}
	return d
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
