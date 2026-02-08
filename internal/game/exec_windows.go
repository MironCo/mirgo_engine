//go:build windows && !game

package game

import (
	"os"
	"os/exec"
)

// execNewBinary spawns a new process and exits (Windows)
func execNewBinary(path string, args []string) error {
	cmd := exec.Command(path, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	os.Exit(0)
	return nil // unreachable
}
