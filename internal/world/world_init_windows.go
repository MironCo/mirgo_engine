//go:build windows

package world

import "log"

func (w *World) initializeCompute() {
	// TODO: Disabled on Windows to get a build, maybe it will be implemented?
	log.Println("Compute: disabled on Windows, at least for now")
}
