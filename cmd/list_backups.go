package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

var rxp *regexp.Regexp

func init() {
	rxp = regexp.MustCompile("(.*)-\\d+-\\d+-\\d{4}.*.tar.gz$")
	listBackupsCommand.Flags().String("host-path", "", "backup host path")
	if err := listBackupsCommand.MarkFlagRequired("host-path"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(listBackupsCommand)
}

// createVolumeFromArchive creates a docker volume and pre-populates it with
// data from a specified archive.
var listBackupsCommand = &cobra.Command{
	Use:   "list-backups",
	Short: "list existing backups",
	Long:  "List backups that exist in the specified host directory",
	Run: func(cmd *cobra.Command, args []string) {
		hostDir, err := cmd.Flags().GetString("host-path")
		if err != nil {
			panic(err)
		}

		if err := cmdListBackups(hostDir); err != nil {
			panic(err)
		}
	},
}

func cmdListBackups(hostDir string) error {
	var list []string
	err := filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		list = append(list, path)
		return nil
	})

	if err != nil {
		return err
	}

	for _, fn := range list {
		fileName := path.Base(fn)
		if !rxp.Match([]byte(fileName)) {
			continue
		}
		match := rxp.FindAllStringSubmatch(fileName, -1)
		dockerVolumeName := match[0][1]
		fmt.Println(dockerVolumeName)
	}
	return nil
}
