//go:build !windows && !game

package game

import (
	"os"
	"syscall"
)

// execNewBinary replaces the current process with a new binary (Unix)
func execNewBinary(path string, args []string) error {
	return syscall.Exec(path, args, os.Environ())
}
