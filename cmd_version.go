package main

import (
	"fmt"

	"github.com/image-server/image-server/core"
)

var cmdVersion = &Command{
	Run:       runVersion,
	UsageLine: "version",
	Short:     "print images version",
	Long:      `Version prints the images version.`,
}

func runVersion(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
	}
	fmt.Printf("images version [%s]\ngit hash [%s]\nbuild stamp [%s]\n", core.VERSION, core.GitHash, core.BuildStamp)
}
