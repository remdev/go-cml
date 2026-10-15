package main

import (
	"os"

	"github.com/remdev/go-cml/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
