package main

import (
	"os"

	"github.com/graydovee/fileManager/pkg"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var (
	cfg config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fileManager",
	Short: "file download and upload manager",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cfg.Build(); err != nil {
			panic(err)
		}
		server, err := pkg.NewHttpServer(&cfg)
		if err != nil {
			panic(err)
		}
		if err := server.Run(); err != nil {
			panic(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	def := config.GetDefault()
	f := rootCmd.Flags()
	f.StringVarP(&cfg.Address, "address", "a", def.Address, "server listen address")
	f.BoolVarP(&cfg.EnableTls, "tls", "t", def.EnableTls, "enable https for generated urls")
	f.StringVar(&cfg.InternalHost, "internal-host", def.InternalHost, "internal host")

	f.StringVar(&cfg.Web.DistDir, "web-dist", def.Web.DistDir, "web dist directory (built frontend)")

	f.StringVar(&cfg.Store.Type, "store-type", def.Store.Type, "store type")

	f.StringVar(&cfg.Store.Local.UploadDir, "upload-dir", def.Store.Local.UploadDir, "file upload directory")

	f.StringVar(&cfg.Store.S3.Endpoint, "s3-endpoint", def.Store.S3.Endpoint, "s3 endpoint")
	f.StringVar(&cfg.Store.S3.AccessKeyID, "s3-access-key-id", def.Store.S3.AccessKeyID, "s3 access key id")
	f.StringVar(&cfg.Store.S3.SecretAccessKey, "s3-secret-access-key", def.Store.S3.SecretAccessKey, "s3 secret access key")
	f.StringVar(&cfg.Store.S3.Bucket, "s3-bucket", def.Store.S3.Bucket, "s3 bucket")
	f.BoolVar(&cfg.Store.S3.DisablePathStyle, "s3-disable-path-style", def.Store.S3.DisablePathStyle, "s3 disable path style")
	f.BoolVar(&cfg.Store.S3.DisableSSL, "s3-disable-ssl", def.Store.S3.DisableSSL, "s3 disable ssl")

	f.BoolVar(&cfg.Store.S3.PresignDownload, "s3-presign-download", def.Store.S3.PresignDownload, "return presigned download url instead of proxying (s3 store only)")
	f.DurationVar(&cfg.Store.S3.PresignExpiry, "s3-presign-expiry", def.Store.S3.PresignExpiry, "presigned url validity duration")
	f.StringVar(&cfg.Store.S3.PresignEndpoint, "s3-presign-endpoint", def.Store.S3.PresignEndpoint, "optional public endpoint used to presign urls (defaults to s3 endpoint)")
}
