//go:build darwin

package world

import (
	"log"
	"test3d/internal/compute"
)

func (w *World) initializeCompute() {
	// Initialize GPU compute (Metal on Mac works fine)
	if info, err := compute.Initialize(); err != nil {
		log.Printf("Compute shaders unavailable: %v", err)
	} else {
		log.Printf("Compute: %s | %s | %s | %s", info.Backend, info.Vendor, info.Name, info.DeviceType)
		w.PhysicsWorld.InitGPU()
	}
}
