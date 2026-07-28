package main

import (
	"embed"
	"log"

	"github.com/zaidejjo/zgit/pkg/core"
)

// Version is injected at build time via ldflags (e.g. -X main.Version=0.1.0).
// If empty, "dev" is used as fallback.
var Version = "dev"

//go:embed frontend/dist
var assets embed.FS

func main() {
	// Create the core engine
	engine, err := core.New(core.DefaultEngineOptions())
	if err != nil {
		log.Fatalf("failed to create engine: %v", err)
	}

	// Open the current repo
	if err := engine.OpenRepo(""); err != nil {
		log.Printf("warning: no git repo at cwd: %v", err)
	}

	// Create and run the Wails app
	app := NewApp(engine)
	if err := app.Run(assets); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
