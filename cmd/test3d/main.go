package main

import (
	"fmt"
	"os"
	"path/filepath"
	"test3d/internal/game"
)

func main() {
	// Change working directory to executable location
	// This ensures assets are found on all platforms
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		if err := os.Chdir(execDir); err != nil {
			fmt.Printf("Warning: could not change to executable directory: %v\n", err)
		}
	}

	restoreEditor := len(os.Args) > 1 && os.Args[1] == "--restore-editor"
	g := game.New()
	g.Run(restoreEditor)
}
