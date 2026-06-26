//go:build game

package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)

		// On macOS .app bundle, assets are in ../Resources/
		if runtime.GOOS == "darwin" {
			resourcesDir := filepath.Join(exeDir, "..", "Resources")
			if _, err := os.Stat(filepath.Join(resourcesDir, "assets")); err == nil {
				os.Chdir(resourcesDir)
				return
			}
		}

		// Fallback: change to executable's directory
		os.Chdir(exeDir)
	}
}
