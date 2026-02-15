package world

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Frustum represents the 6 planes of a view frustum for culling
type Frustum struct {
	planes [6]Plane // left, right, bottom, top, near, far
}

// Plane represents a plane in 3D space (ax + by + cz + d = 0)
type Plane struct {
	normal   rl.Vector3
	distance float32
}

// ExtractFrustum extracts frustum planes from a view-projection matrix
// Uses the Gribb/Hartmann method for plane extraction
func ExtractFrustum(camera rl.Camera3D) Frustum {
	// Get the current view and projection matrices from raylib
	view := rl.GetCameraMatrix(camera)

	// Build projection matrix based on camera settings
	aspect := float32(rl.GetScreenWidth()) / float32(rl.GetScreenHeight())
	var proj rl.Matrix
	if camera.Projection == rl.CameraPerspective {
		proj = rl.MatrixPerspective(camera.Fovy*rl.Deg2rad, aspect, 0.1, 1000.0)
	} else {
		halfH := camera.Fovy / 2.0
		halfW := halfH * aspect
		proj = rl.MatrixOrtho(-halfW, halfW, -halfH, halfH, 0.1, 1000.0)
	}

	// Combine view and projection: VP = P * V
	vp := rl.MatrixMultiply(view, proj)

	var f Frustum

	// Left plane: row4 + row1
	f.planes[0] = normalizePlane(Plane{
		normal: rl.Vector3{
			X: vp.M3 + vp.M0,
			Y: vp.M7 + vp.M4,
			Z: vp.M11 + vp.M8,
		},
		distance: vp.M15 + vp.M12,
	})

	// Right plane: row4 - row1
	f.planes[1] = normalizePlane(Plane{
		normal: rl.Vector3{
			X: vp.M3 - vp.M0,
			Y: vp.M7 - vp.M4,
			Z: vp.M11 - vp.M8,
		},
		distance: vp.M15 - vp.M12,
	})

	// Bottom plane: row4 + row2
	f.planes[2] = normalizePlane(Plane{
		normal: rl.Vector3{
			X: vp.M3 + vp.M1,
			Y: vp.M7 + vp.M5,
			Z: vp.M11 + vp.M9,
		},
		distance: vp.M15 + vp.M13,
	})

	// Top plane: row4 - row2
	f.planes[3] = normalizePlane(Plane{
		normal: rl.Vector3{
			X: vp.M3 - vp.M1,
			Y: vp.M7 - vp.M5,
			Z: vp.M11 - vp.M9,
		},
		distance: vp.M15 - vp.M13,
	})

	// Near plane: row4 + row3
	f.planes[4] = normalizePlane(Plane{
		normal: rl.Vector3{
			X: vp.M3 + vp.M2,
			Y: vp.M7 + vp.M6,
			Z: vp.M11 + vp.M10,
		},
		distance: vp.M15 + vp.M14,
	})

	// Far plane: row4 - row3
	f.planes[5] = normalizePlane(Plane{
		normal: rl.Vector3{
			X: vp.M3 - vp.M2,
			Y: vp.M7 - vp.M6,
			Z: vp.M11 - vp.M10,
		},
		distance: vp.M15 - vp.M14,
	})

	return f
}

// normalizePlane normalizes a plane equation
func normalizePlane(p Plane) Plane {
	length := rl.Vector3Length(p.normal)
	if length == 0 {
		return p
	}
	return Plane{
		normal:   rl.Vector3Scale(p.normal, 1.0/length),
		distance: p.distance / length,
	}
}

// ContainsSphere tests if a sphere is inside or intersects the frustum
// Returns true if the sphere should be rendered
func (f *Frustum) ContainsSphere(center rl.Vector3, radius float32) bool {
	for i := 0; i < 6; i++ {
		// Distance from center to plane
		dist := rl.Vector3DotProduct(f.planes[i].normal, center) + f.planes[i].distance
		// If sphere is completely behind any plane, it's outside
		if dist < -radius {
			return false
		}
	}
	return true
}

// ContainsPoint tests if a point is inside the frustum
func (f *Frustum) ContainsPoint(point rl.Vector3) bool {
	for i := 0; i < 6; i++ {
		dist := rl.Vector3DotProduct(f.planes[i].normal, point) + f.planes[i].distance
		if dist < 0 {
			return false
		}
	}
	return true
}
