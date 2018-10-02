package cmd

import (
	"fmt"

	"github.com/image-server/image-server/core"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print image server version",
	Long:  `Print image server version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("images version [%s]\ngit hash [%s]\nbuild stamp [%s]\n", core.VERSION, core.GitHash, core.BuildStamp)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
