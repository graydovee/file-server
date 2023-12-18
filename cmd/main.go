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

	defaultUploadDir = "./uploads"
	defaultStaticDir = "./static"
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
	f.StringVarP(&cfg.StaticDir, "static-dir", "s", defaultStaticDir, "static file directory")
	f.StringVarP(&cfg.UploadDir, "upload-dir", "u", defaultUploadDir, "file upload directory")
	f.StringVarP(&cfg.Address, "address", "a", ":8080", "server listen address")
	f.BoolVarP(&cfg.EnableTls, "tls", "t", false, "enable https")
}
