package physics

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// OBB represents an Oriented Bounding Box
type OBB struct {
	Center   rl.Vector3    // World-space center
	HalfSize rl.Vector3    // Half-extents along local axes
	Axes     [3]rl.Vector3 // Local X, Y, Z axes (rotated)
}

// NewOBB creates an OBB from center, size, and euler rotation (degrees)
func NewOBB(center, size, rotation rl.Vector3) OBB {
	// Convert to radians
	rx := float64(rotation.X) * math.Pi / 180
	ry := float64(rotation.Y) * math.Pi / 180
	rz := float64(rotation.Z) * math.Pi / 180

	// Build rotation matrix (same order as your engine: X, Y, Z)
	rotX := rl.MatrixRotateX(float32(rx))
	rotY := rl.MatrixRotateY(float32(ry))
	rotZ := rl.MatrixRotateZ(float32(rz))
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

	// Extract rotated axes
	axes := [3]rl.Vector3{
		rl.Vector3Normalize(rl.Vector3{X: rotMatrix.M0, Y: rotMatrix.M1, Z: rotMatrix.M2}),
		rl.Vector3Normalize(rl.Vector3{X: rotMatrix.M4, Y: rotMatrix.M5, Z: rotMatrix.M6}),
		rl.Vector3Normalize(rl.Vector3{X: rotMatrix.M8, Y: rotMatrix.M9, Z: rotMatrix.M10}),
	}

	return OBB{
		Center:   center,
		HalfSize: rl.Vector3{X: size.X / 2, Y: size.Y / 2, Z: size.Z / 2},
		Axes:     axes,
	}
}

// NewAABBasOBB creates an axis-aligned OBB (no rotation)
func NewAABBasOBB(center, size rl.Vector3) OBB {
	return OBB{
		Center:   center,
		HalfSize: rl.Vector3{X: size.X / 2, Y: size.Y / 2, Z: size.Z / 2},
		Axes: [3]rl.Vector3{
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
			{X: 0, Y: 0, Z: 1},
		},
	}
}

// IntersectsOBB tests if two OBBs intersect using the Separating Axis Theorem
func (a OBB) IntersectsOBB(b OBB) bool {
	// Vector from A's center to B's center
	t := rl.Vector3Subtract(b.Center, a.Center)

	// We need to test 15 axes:
	// - 3 face normals from A
	// - 3 face normals from B
	// - 9 cross products of edges (A's edges x B's edges)

	// Test A's face normals
	for i := 0; i < 3; i++ {
		if !overlapOnAxis(a, b, a.Axes[i], t) {
			return false
		}
	}

	// Test B's face normals
	for i := 0; i < 3; i++ {
		if !overlapOnAxis(a, b, b.Axes[i], t) {
			return false
		}
	}

	// Test cross products of edges
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			axis := rl.Vector3CrossProduct(a.Axes[i], b.Axes[j])
			// Skip near-zero axes (parallel edges)
			if rl.Vector3Length(axis) > 0.0001 {
				axis = rl.Vector3Normalize(axis)
				if !overlapOnAxis(a, b, axis, t) {
					return false
				}
			}
		}
	}

	return true
}

// overlapOnAxis checks if two OBBs overlap when projected onto a given axis
func overlapOnAxis(a, b OBB, axis, t rl.Vector3) bool {
	// Project the half-sizes of both boxes onto the axis
	aProjection := a.HalfSize.X*absf(rl.Vector3DotProduct(a.Axes[0], axis)) +
		a.HalfSize.Y*absf(rl.Vector3DotProduct(a.Axes[1], axis)) +
		a.HalfSize.Z*absf(rl.Vector3DotProduct(a.Axes[2], axis))

	bProjection := b.HalfSize.X*absf(rl.Vector3DotProduct(b.Axes[0], axis)) +
		b.HalfSize.Y*absf(rl.Vector3DotProduct(b.Axes[1], axis)) +
		b.HalfSize.Z*absf(rl.Vector3DotProduct(b.Axes[2], axis))

	// Project the distance between centers onto the axis
	distance := absf(rl.Vector3DotProduct(t, axis))

	// If the distance is greater than the sum of projections, there's a separating axis
	return distance <= aProjection+bProjection
}

// ResolveOBB returns the minimum translation vector to push 'a' out of 'b'
// Returns zero vector if no overlap
func (a OBB) ResolveOBB(b OBB) rl.Vector3 {
	if !a.IntersectsOBB(b) {
		return rl.Vector3Zero()
	}

	t := rl.Vector3Subtract(b.Center, a.Center)
	minPenetration := float32(math.MaxFloat32)
	var mtv rl.Vector3

	// Test all 15 axes and find the one with minimum penetration
	testAxis := func(axis rl.Vector3) {
		if rl.Vector3Length(axis) < 0.0001 {
			return
		}
		axis = rl.Vector3Normalize(axis)

		aProj := a.HalfSize.X*absf(rl.Vector3DotProduct(a.Axes[0], axis)) +
			a.HalfSize.Y*absf(rl.Vector3DotProduct(a.Axes[1], axis)) +
			a.HalfSize.Z*absf(rl.Vector3DotProduct(a.Axes[2], axis))

		bProj := b.HalfSize.X*absf(rl.Vector3DotProduct(b.Axes[0], axis)) +
			b.HalfSize.Y*absf(rl.Vector3DotProduct(b.Axes[1], axis)) +
			b.HalfSize.Z*absf(rl.Vector3DotProduct(b.Axes[2], axis))

		dist := rl.Vector3DotProduct(t, axis)
		penetration := aProj + bProj - absf(dist)

		if penetration < minPenetration {
			minPenetration = penetration
			// Push in the direction away from B
			if dist < 0 {
				mtv = rl.Vector3Scale(axis, penetration)
			} else {
				mtv = rl.Vector3Scale(axis, -penetration)
			}
		}
	}

	// Test A's face normals
	for i := 0; i < 3; i++ {
		testAxis(a.Axes[i])
	}

	// Test B's face normals
	for i := 0; i < 3; i++ {
		testAxis(b.Axes[i])
	}

	// Test cross products of edges
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			testAxis(rl.Vector3CrossProduct(a.Axes[i], b.Axes[j]))
		}
	}

	return mtv
}

// IntersectsSphere tests if an OBB intersects with a sphere
func (o OBB) IntersectsSphere(center rl.Vector3, radius float32) bool {
	// Transform sphere center to OBB's local space
	local := rl.Vector3Subtract(center, o.Center)
	localX := rl.Vector3DotProduct(local, o.Axes[0])
	localY := rl.Vector3DotProduct(local, o.Axes[1])
	localZ := rl.Vector3DotProduct(local, o.Axes[2])

	// Clamp to box extents
	closestX := clampf(localX, -o.HalfSize.X, o.HalfSize.X)
	closestY := clampf(localY, -o.HalfSize.Y, o.HalfSize.Y)
	closestZ := clampf(localZ, -o.HalfSize.Z, o.HalfSize.Z)

	// Distance from sphere center to closest point on box
	dx := localX - closestX
	dy := localY - closestY
	dz := localZ - closestZ
	distSq := dx*dx + dy*dy + dz*dz

	return distSq <= radius*radius
}

// ClosestPointOnOBB returns the closest point on the OBB surface to the given point
func ClosestPointOnOBB(o OBB, point rl.Vector3) rl.Vector3 {
	// Transform point to OBB's local space
	local := rl.Vector3Subtract(point, o.Center)
	localX := rl.Vector3DotProduct(local, o.Axes[0])
	localY := rl.Vector3DotProduct(local, o.Axes[1])
	localZ := rl.Vector3DotProduct(local, o.Axes[2])

	// Clamp to box extents
	closestX := clampf(localX, -o.HalfSize.X, o.HalfSize.X)
	closestY := clampf(localY, -o.HalfSize.Y, o.HalfSize.Y)
	closestZ := clampf(localZ, -o.HalfSize.Z, o.HalfSize.Z)

	// Transform back to world space
	result := o.Center
	result = rl.Vector3Add(result, rl.Vector3Scale(o.Axes[0], closestX))
	result = rl.Vector3Add(result, rl.Vector3Scale(o.Axes[1], closestY))
	result = rl.Vector3Add(result, rl.Vector3Scale(o.Axes[2], closestZ))

	return result
}

func absf(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clampf(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// NewOBBFromBox creates an OBB from center, size, rotation, and scale
// This is a convenience function to avoid import cycles
func NewOBBFromBox(center, size, rotation, scale rl.Vector3) OBB {
	// Apply scale to size
	scaledSize := rl.Vector3{
		X: size.X * scale.X,
		Y: size.Y * scale.Y,
		Z: size.Z * scale.Z,
	}
	return NewOBB(center, scaledSize, rotation)
}
