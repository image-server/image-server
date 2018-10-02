package main

import (
	_ "net/http/pprof"

	"github.com/image-server/image-server/cmd"
)

func main() {
	cmd.Execute()
}
