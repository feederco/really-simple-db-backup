package main

import (
	"os"

	"github.com/feederco/really-simple-db-backup/cmd"
)

func main() {
	cmd.Begin(os.Args)
}
