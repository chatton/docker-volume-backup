package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//func init() {
//	rootCmd.PersistentFlags().String("mode", "file", "specify either file or s3")
//}

var rootCmd = &cobra.Command{
	Use:   "docker-volume-backup",
	Short: "cli with docker volume backup utility commands",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
