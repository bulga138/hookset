package main

import (
	"fmt"
	"os"

	"github.com/bulga138/hookset/cmd/hookset/commands"
	"github.com/bulga138/hookset/internal/git"
)

func main() {
	// Skip version check for `hookset version` and `hookset help`.
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" ||
		os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		commands.Execute()
		return
	}

	if err := git.CheckMinVersion(2, 54); err != nil {
		fmt.Fprintln(os.Stderr, "[hookset] error:", err)
		os.Exit(1)
	}

	commands.Execute()
}
