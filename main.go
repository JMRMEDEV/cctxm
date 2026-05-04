package main

import (
	"os"

	"github.com/jmrmedev/cctxm/cmd/root"
)

var version = "dev"

func main() {
	root.Cmd.Version = version
	if err := root.Cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
