// Stress test comparing CPU vs GPU broad-phase collision detection
package main

import (
	"fmt"
	"math/rand"
	"time"

	"test3d/internal/compute"
)

func main() {
	// Initialize compute
	info, err := compute.Initialize()
	if err != nil {
		panic(fmt.Sprintf("Failed to init compute: %v", err))
	}
	fmt.Printf("GPU: %s | %s | %s\n\n", info.Backend, info.Vendor, info.Name)

	// Test various object counts
	testCounts := []int{100, 500, 1000, 2000, 5000, 10000, 20000}

	for _, count := range testCounts {
		testBroadPhase(count)
	}
}

func testBroadPhase(count int) {
	// Generate random spheres in a bounded space
	spheres := make([]compute.Sphere, count)
	rand.Seed(42) // Consistent results

	// Spawn in a cube, size scales with count to keep density reasonable
	spawnSize := float32(50.0) + float32(count)/100.0

	for i := range spheres {
		spheres[i] = compute.Sphere{
			X:      rand.Float32()*spawnSize - spawnSize/2,
			Y:      rand.Float32()*spawnSize - spawnSize/2,
			Z:      rand.Float32()*spawnSize - spawnSize/2,
			Radius: 0.5 + rand.Float32()*0.5, // 0.5 to 1.0 radius
		}
	}

	// GPU broad-phase
	maxPairs := uint32(count * 20) // Generous pair buffer
	bp, err := compute.NewBroadPhase(uint32(count), maxPairs)
	if err != nil {
		fmt.Printf("%5d objects: GPU ERROR: %v\n", count, err)
		return
	}
	defer bp.Release()

	// Warm up
	bp.DetectPairs(spheres)

	// Time GPU
	gpuStart := time.Now()
	const gpuIterations = 10
	var gpuPairs []compute.CollisionPair
	for i := 0; i < gpuIterations; i++ {
		gpuPairs, _ = bp.DetectPairs(spheres)
	}
	gpuTime := time.Since(gpuStart) / gpuIterations

	// Time CPU (naive O(n²))
	cpuStart := time.Now()
	const cpuIterations = 10
	var cpuPairCount int
	for iter := 0; iter < cpuIterations; iter++ {
		cpuPairCount = 0
		for i := 0; i < len(spheres); i++ {
			for j := i + 1; j < len(spheres); j++ {
				dx := spheres[i].X - spheres[j].X
				dy := spheres[i].Y - spheres[j].Y
				dz := spheres[i].Z - spheres[j].Z
				distSq := dx*dx + dy*dy + dz*dz
				radiusSum := spheres[i].Radius + spheres[j].Radius
				if distSq < radiusSum*radiusSum {
					cpuPairCount++
				}
			}
		}
	}
	cpuTime := time.Since(cpuStart) / cpuIterations

	// Calculate speedup
	speedup := float64(cpuTime) / float64(gpuTime)

	fmt.Printf("%5d objects: GPU %8v (%4d pairs) | CPU %10v (%4d pairs) | %.1fx speedup\n",
		count, gpuTime.Round(time.Microsecond), len(gpuPairs),
		cpuTime.Round(time.Microsecond), cpuPairCount, speedup)
}