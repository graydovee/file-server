package main

import (
	"github.com/graydovee/fileManager/pkg"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	Execute()
}

var (
	address   string
	uploadDir string
	tls       bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fileManager",
	Short: "file download and upload manager",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		server, err := pkg.NewFileServer(address, uploadDir, tls)
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

var defaultUploadDir = "./uploads"

func init() {
	rootCmd.Flags().StringVarP(&uploadDir, "upload-dir", "u", defaultUploadDir, "file upload directory")
	rootCmd.Flags().StringVarP(&address, "address", "a", ":8080", "server listen address")
	rootCmd.Flags().BoolVarP(&tls, "tls", "t", false, "enable https")
}
