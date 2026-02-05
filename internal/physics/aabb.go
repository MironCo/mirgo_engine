package physics

import rl "github.com/gen2brain/raylib-go/raylib"

type AABB struct {
	Min rl.Vector3
	Max rl.Vector3
}

// NewAABBFromCenter creates an AABB from a center point and full size dimensions.
func NewAABBFromCenter(center, size rl.Vector3) AABB {
	half := rl.Vector3{X: size.X / 2, Y: size.Y / 2, Z: size.Z / 2}
	return AABB{
		Min: rl.Vector3Subtract(center, half),
		Max: rl.Vector3Add(center, half),
	}
}

func (a AABB) Intersects(b AABB) bool {
	return a.Min.X <= b.Max.X && a.Max.X >= b.Min.X &&
		a.Min.Y <= b.Max.Y && a.Max.Y >= b.Min.Y &&
		a.Min.Z <= b.Max.Z && a.Max.Z >= b.Min.Z
}

// Resolve returns the minimum translation vector to push 'a' out of 'b'.
// Returns zero vector if no overlap.
func (a AABB) Resolve(b AABB) rl.Vector3 {
	if !a.Intersects(b) {
		return rl.Vector3Zero()
	}

	// Penetration depth in each direction
	dx1 := b.Max.X - a.Min.X // push a in +X
	dx2 := a.Max.X - b.Min.X // push a in -X
	dy1 := b.Max.Y - a.Min.Y // push a in +Y
	dy2 := a.Max.Y - b.Min.Y // push a in -Y
	dz1 := b.Max.Z - a.Min.Z // push a in +Z
	dz2 := a.Max.Z - b.Min.Z // push a in -Z

	// Find the axis with minimum penetration — that's the push-out direction
	min := dx1
	result := rl.Vector3{X: dx1}

	if dx2 < min {
		min = dx2
		result = rl.Vector3{X: -dx2}
	}
	if dy1 < min {
		min = dy1
		result = rl.Vector3{Y: dy1}
	}
	if dy2 < min {
		min = dy2
		result = rl.Vector3{Y: -dy2}
	}
	if dz1 < min {
		min = dz1
		result = rl.Vector3{Z: dz1}
	}
	if dz2 < min {
		result = rl.Vector3{Z: -dz2}
	}

	return result
}
