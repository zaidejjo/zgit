// zgit — A modern, fast Git & GitHub client.
package main

import (
	"fmt"
	"os"

	"github.com/zaidejjo/zgit/internal/cli"
)

// Version is injected at build time via ldflags (e.g. -X main.Version=0.1.0).
// If empty, "dev" is used as fallback.
var Version = "dev"

func main() {
	// Handle --version at top level before cobra processes
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("zgit %s\n", Version)
			return
		}
	}
	cli.Execute()
}
