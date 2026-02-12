package main

import (
	"os"
	"path/filepath"
	"strings"
	"test3d/internal/game"
)

func main() {
	// Change working directory to executable location for deployed builds.
	// Skip this for "go run" which puts the binary in a temp directory.
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		// Detect "go run" by checking if executable is in a temp/go-build directory
		if !strings.Contains(execDir, "go-build") {
			os.Chdir(execDir)
		}
	}

	restoreEditor := len(os.Args) > 1 && os.Args[1] == "--restore-editor"
	g := game.New()
	g.Run(restoreEditor)
}
