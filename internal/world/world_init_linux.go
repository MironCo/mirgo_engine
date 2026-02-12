//go:build linux

package world

import "log"

func (w *World) initializeCompute() {
	// Disabled on Linux due to EGL/WebGPU conflicts with NVIDIA on X11
	log.Println("Compute: disabled on Linux (EGL conflict workaround)")
}
