package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/feederco/really-simple-db-backup/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	versionFlag := flag.Bool("version", false, "Show current version with format: version\\ncommit\\ndate")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s\n%s\n%s\n", version, commit, date)
		return
	}

	cmd.Begin(os.Args)
}
