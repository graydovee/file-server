package main

import (
	"github.com/graydovee/fileManager/pkg"
	"github.com/graydovee/fileManager/pkg/config"
	"github.com/spf13/cobra"
	"os"
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
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
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
	f := rootCmd.Flags()
	f.StringVarP(&cfg.Address, "address", "a", config.GetDefault().Address, "server listen address")
	f.BoolVarP(&cfg.EnableTls, "tls", "t", config.GetDefault().EnableTls, "enable https")
	f.StringVar(&cfg.InternalHost, "internal-host", config.GetDefault().InternalHost, "internal host")

	f.StringVar(&cfg.Resource.StaticDir, "resource-static", config.GetDefault().Resource.StaticDir, "static file directory")
	f.StringVar(&cfg.Resource.TemplateDir, "template-dir", config.GetDefault().Resource.TemplateDir, "template file directory")

	f.StringVar(&cfg.Store.Type, "store-type", config.GetDefault().Store.Type, "store type")

	f.StringVar(&cfg.Store.Local.UploadDir, "upload-dir", config.GetDefault().Store.Local.UploadDir, "file upload directory")

	f.StringVar(&cfg.Store.S3.Endpoint, "s3-endpoint", config.GetDefault().Store.S3.Endpoint, "s3 endpoint")
	f.StringVar(&cfg.Store.S3.AccessKeyID, "s3-access-key-id", config.GetDefault().Store.S3.AccessKeyID, "s3 access key id")
	f.StringVar(&cfg.Store.S3.SecretAccessKey, "s3-secret-access-key", config.GetDefault().Store.S3.SecretAccessKey, "s3 secret access key")
	f.StringVar(&cfg.Store.S3.Bucket, "s3-bucket", config.GetDefault().Store.S3.Bucket, "s3 bucket")
	f.BoolVar(&cfg.Store.S3.DisablePathStyle, "s3-disable-path-style", config.GetDefault().Store.S3.DisablePathStyle, "s3 disable path style")
	f.BoolVar(&cfg.Store.S3.DisableSSL, "s3-disable-ssl", config.GetDefault().Store.S3.DisableSSL, "s3 disable ssl")
}
