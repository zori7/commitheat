// Command commitheat shows git statistics for a local repository in the terminal.
package main

import (
	"fmt"
	"os"

	"commitheat/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "commitheat:", err)
		os.Exit(1)
	}
}
