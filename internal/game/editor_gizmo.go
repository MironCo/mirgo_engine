//go:build !game

package game

import (
	"math"

	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GizmoMode int

const (
	GizmoMove   GizmoMode = 0
	GizmoRotate GizmoMode = 1
	GizmoScale  GizmoMode = 2
)

const (
	gizmoLength    float32 = 2.0
	gizmoTipSize   float32 = 0.2
	gizmoHitDist   float32 = 0.3
	gizmoThickness float32 = 0.06
)

var gizmoAxes = [3]rl.Vector3{
	{X: 1, Y: 0, Z: 0}, // X - red
	{X: 0, Y: 1, Z: 0}, // Y - green
	{X: 0, Y: 0, Z: 1}, // Z - blue
}

var gizmoColors = [3]rl.Color{rl.Red, rl.Green, rl.Blue}

// pickGizmoAxis returns the index of the gizmo axis closest to the mouse ray, or -1.
func (e *Editor) pickGizmoAxis(ray rl.Ray) int {
	if e.Selected == nil {
		return -1
	}

	center := e.Selected.WorldPosition()
	bestDist := float32(999.0)
	bestAxis := -1

	if e.gizmoMode == GizmoRotate {
		// For rotation gizmo, check distance to each ring
		radius := gizmoLength * 0.8
		ringHitDist := float32(0.4) // More forgiving hit distance for rings

		for i := range gizmoAxes {
			// Get the plane normal for this ring
			var planeNormal rl.Vector3
			switch i {
			case 0: // X axis - ring in YZ plane
				planeNormal = rl.Vector3{X: 1, Y: 0, Z: 0}
			case 1: // Y axis - ring in XZ plane
				planeNormal = rl.Vector3{X: 0, Y: 1, Z: 0}
			case 2: // Z axis - ring in XY plane
				planeNormal = rl.Vector3{X: 0, Y: 0, Z: 1}
			}

			// Intersect ray with the ring's plane
			if pt, ok := rayPlaneIntersect(ray.Position, ray.Direction, center, planeNormal); ok {
				// Check if intersection point is near the ring
				distFromCenter := rl.Vector3Length(rl.Vector3Subtract(pt, center))
				distFromRing := float32(math.Abs(float64(distFromCenter - radius)))

				if distFromRing < ringHitDist && distFromRing < bestDist {
					bestDist = distFromRing
					bestAxis = i
				}
			}
		}
	} else {
		// For move/scale gizmos, use line-ray intersection
		for i, axis := range gizmoAxes {
			_, t2, dist := closestPointBetweenRays(ray.Position, ray.Direction, center, axis)
			if t2 > 0 && t2 < gizmoLength && dist < gizmoHitDist {
				if dist < bestDist {
					bestDist = dist
					bestAxis = i
				}
			}
		}
	}
	return bestAxis
}

func (e *Editor) startDrag(axisIdx int, ray rl.Ray) {
	// Save undo state before modifying
	e.pushUndo()

	e.dragging = true
	e.dragAxisIdx = axisIdx
	e.dragAxis = gizmoAxes[axisIdx]
	e.dragInitPos = e.Selected.Transform.Position
	e.dragInitWorldPos = e.Selected.WorldPosition()
	e.dragInitRot = e.Selected.Transform.Rotation
	e.dragInitScale = e.Selected.Transform.Scale

	// Build a drag plane using world position for correct 3D picking
	viewDir := rl.Vector3Normalize(rl.Vector3Subtract(e.dragInitWorldPos, e.camera.Position))
	cross1 := rl.Vector3CrossProduct(viewDir, e.dragAxis)
	e.dragPlaneNormal = rl.Vector3Normalize(rl.Vector3CrossProduct(e.dragAxis, cross1))

	if pt, ok := rayPlaneIntersect(ray.Position, ray.Direction, e.dragInitWorldPos, e.dragPlaneNormal); ok {
		e.dragStart = rl.Vector3DotProduct(rl.Vector3Subtract(pt, e.dragInitWorldPos), e.dragAxis)
	}
}

func (e *Editor) updateDrag(ray rl.Ray) {
	if e.Selected == nil {
		e.dragging = false
		return
	}

	// Use the stored initial world position for drag plane intersection
	pt, ok := rayPlaneIntersect(ray.Position, ray.Direction, e.dragInitWorldPos, e.dragPlaneNormal)
	if !ok {
		return
	}

	currentT := rl.Vector3DotProduct(rl.Vector3Subtract(pt, e.dragInitWorldPos), e.dragAxis)
	delta := currentT - e.dragStart

	switch e.gizmoMode {
	case GizmoMove:
		// Calculate world-space delta
		worldDelta := rl.Vector3Scale(e.dragAxis, delta)

		// Convert to local space if object has a parent
		if e.Selected.Parent != nil {
			// Get inverse parent rotation
			parentRot := e.Selected.Parent.WorldRotation()
			rx := float64(-parentRot.X) * math.Pi / 180
			ry := float64(-parentRot.Y) * math.Pi / 180
			rz := float64(-parentRot.Z) * math.Pi / 180
			// Inverse rotation order: Z, Y, X (reverse of forward)
			rotZ := rl.MatrixRotateZ(float32(rz))
			rotY := rl.MatrixRotateY(float32(ry))
			rotX := rl.MatrixRotateX(float32(rx))
			invRotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotZ, rotY), rotX)

			// Rotate delta into parent's local space
			localDelta := rl.Vector3Transform(worldDelta, invRotMatrix)

			// Account for parent scale
			parentScale := e.Selected.Parent.WorldScale()
			localDelta.X /= parentScale.X
			localDelta.Y /= parentScale.Y
			localDelta.Z /= parentScale.Z

			e.Selected.Transform.Position = rl.Vector3Add(e.dragInitPos, localDelta)
		} else {
			e.Selected.Transform.Position = rl.Vector3Add(e.dragInitPos, worldDelta)
		}

	case GizmoRotate:
		// Map drag distance to degrees (1 unit = 45 degrees)
		degrees := delta * 45.0
		rot := e.dragInitRot
		switch e.dragAxisIdx {
		case 0:
			rot.X += degrees
		case 1:
			rot.Y += degrees
		case 2:
			rot.Z += degrees
		}
		e.Selected.Transform.Rotation = rot

	case GizmoScale:
		// Map drag distance to scale factor (drag right = bigger)
		factor := float32(1.0) + delta*0.5
		if factor < 0.1 {
			factor = 0.1
		}
		s := e.dragInitScale
		switch e.dragAxisIdx {
		case 0:
			s.X = e.dragInitScale.X * factor
		case 1:
			s.Y = e.dragInitScale.Y * factor
		case 2:
			s.Z = e.dragInitScale.Z * factor
		}
		e.Selected.Transform.Scale = s
	}
}

// Draw3D draws selection wireframes and gizmo. Call inside BeginMode3D/EndMode3D.
func (e *Editor) Draw3D() {
	// Draw point light gizmos for all point lights (not just selected)
	for _, g := range e.world.Scene.GameObjects {
		if pl := engine.GetComponent[*components.PointLight](g); pl != nil {
			pos := pl.GetPosition()
			// Draw small sphere at light position
			rl.DrawSphere(pos, 0.15, pl.Color)
			// Draw radius wireframe
			rl.DrawSphereWires(pos, pl.Radius, 8, 8, rl.Fade(pl.Color, 0.3))
		}

		// Debug: draw bounding boxes for objects without colliders
		if engine.GetComponent[*components.BoxCollider](g) == nil &&
			engine.GetComponent[*components.SphereCollider](g) == nil &&
			engine.GetComponent[*components.MeshCollider](g) == nil {
			if mr := engine.GetComponent[*components.ModelRenderer](g); mr != nil {
				bounds := rl.GetModelBoundingBox(mr.Model)
				pos := g.WorldPosition()
				scale := g.WorldScale()

				// Calculate size from bounds (always positive)
				size := rl.Vector3{
					X: (bounds.Max.X - bounds.Min.X) * absF(scale.X),
					Y: (bounds.Max.Y - bounds.Min.Y) * absF(scale.Y),
					Z: (bounds.Max.Z - bounds.Min.Z) * absF(scale.Z),
				}

				// Bounding box is centered at object position
				rl.DrawCubeWiresV(pos, size, rl.Magenta)
			}
		}
	}

	if e.Selected == nil {
		return
	}

	// Disable depth testing so gizmos always draw on top
	rl.DrawRenderBatchActive() // Force flush of previous draw calls
	rl.DisableDepthTest()

	// Selection wireframe
	if box := engine.GetComponent[*components.BoxCollider](e.Selected); box != nil {
		center := box.GetCenter()
		rot := e.Selected.WorldRotation()
		drawRotatedBoxWires(center, box.GetWorldSize(), rot, rl.Yellow)
	} else if sphere := engine.GetComponent[*components.SphereCollider](e.Selected); sphere != nil {
		center := sphere.GetCenter()
		rl.DrawSphereWires(center, sphere.Radius, 8, 8, rl.Yellow)
	} else if mesh := engine.GetComponent[*components.MeshCollider](e.Selected); mesh != nil && mesh.IsBuilt() {
		// Draw BVH root bounds for mesh collider
		bounds := mesh.GetBounds()
		center := rl.Vector3{
			X: (bounds.Min.X + bounds.Max.X) / 2,
			Y: (bounds.Min.Y + bounds.Max.Y) / 2,
			Z: (bounds.Min.Z + bounds.Max.Z) / 2,
		}
		size := rl.Vector3{
			X: bounds.Max.X - bounds.Min.X,
			Y: bounds.Max.Y - bounds.Min.Y,
			Z: bounds.Max.Z - bounds.Min.Z,
		}
		rl.DrawCubeWiresV(center, size, rl.Yellow)
	} else if pl := engine.GetComponent[*components.PointLight](e.Selected); pl != nil {
		// Highlight selected point light
		pos := pl.GetPosition()
		rl.DrawSphereWires(pos, pl.Radius, 12, 12, rl.Yellow)
	} else if mr := engine.GetComponent[*components.ModelRenderer](e.Selected); mr != nil {
		// Draw model bounding box for objects without colliders
		bounds := rl.GetModelBoundingBox(mr.Model)
		pos := e.Selected.WorldPosition()
		scale := e.Selected.WorldScale()

		// Calculate world-space bounding box size
		size := rl.Vector3{
			X: (bounds.Max.X - bounds.Min.X) * scale.X,
			Y: (bounds.Max.Y - bounds.Min.Y) * scale.Y,
			Z: (bounds.Max.Z - bounds.Min.Z) * scale.Z,
		}
		// Calculate center offset
		localCenter := rl.Vector3{
			X: (bounds.Min.X + bounds.Max.X) / 2 * scale.X,
			Y: (bounds.Min.Y + bounds.Max.Y) / 2 * scale.Y,
			Z: (bounds.Min.Z + bounds.Max.Z) / 2 * scale.Z,
		}
		worldCenter := rl.Vector3Add(pos, localCenter)
		rl.DrawCubeWiresV(worldCenter, size, rl.Yellow)
	}

	// Transform gizmo
	center := e.Selected.WorldPosition()

	for i, axis := range gizmoAxes {
		color := gizmoColors[i]
		if e.dragging && e.dragAxisIdx == i {
			color = rl.Yellow
		} else if !e.dragging && e.hoveredAxis == i {
			color = rl.Yellow
		}

		end := rl.Vector3Add(center, rl.Vector3Scale(axis, gizmoLength))

		switch e.gizmoMode {
		case GizmoMove:
			rl.DrawCylinderEx(center, end, gizmoThickness, gizmoThickness, 8, color)
			tip := rl.Vector3{X: gizmoTipSize, Y: gizmoTipSize, Z: gizmoTipSize}
			rl.DrawCubeV(end, tip, color)
		case GizmoRotate:
			// Draw arc segments as thick cylinders to suggest rotation
			segments := 16
			radius := gizmoLength * 0.8
			for s := range segments {
				t0 := float64(s) / float64(segments) * math.Pi * 2
				t1 := float64(s+1) / float64(segments) * math.Pi * 2
				var p0, p1 rl.Vector3
				switch i {
				case 0: // X - rotate in YZ plane
					p0 = rl.Vector3{X: center.X, Y: center.Y + radius*float32(math.Cos(t0)), Z: center.Z + radius*float32(math.Sin(t0))}
					p1 = rl.Vector3{X: center.X, Y: center.Y + radius*float32(math.Cos(t1)), Z: center.Z + radius*float32(math.Sin(t1))}
				case 1: // Y - rotate in XZ plane
					p0 = rl.Vector3{X: center.X + radius*float32(math.Cos(t0)), Y: center.Y, Z: center.Z + radius*float32(math.Sin(t0))}
					p1 = rl.Vector3{X: center.X + radius*float32(math.Cos(t1)), Y: center.Y, Z: center.Z + radius*float32(math.Sin(t1))}
				case 2: // Z - rotate in XY plane
					p0 = rl.Vector3{X: center.X + radius*float32(math.Cos(t0)), Y: center.Y + radius*float32(math.Sin(t0)), Z: center.Z}
					p1 = rl.Vector3{X: center.X + radius*float32(math.Cos(t1)), Y: center.Y + radius*float32(math.Sin(t1)), Z: center.Z}
				}
				rl.DrawCylinderEx(p0, p1, gizmoThickness*0.7, gizmoThickness*0.7, 6, color)
			}
		case GizmoScale:
			rl.DrawCylinderEx(center, end, gizmoThickness, gizmoThickness, 8, color)
			// Cube at the end instead of small tip
			cubeSize := rl.Vector3{X: 0.25, Y: 0.25, Z: 0.25}
			rl.DrawCubeV(end, cubeSize, color)
			rl.DrawCubeWiresV(end, cubeSize, color)
		}
	}

	// Re-enable depth testing
	rl.DrawRenderBatchActive() // Force flush of gizmo draw calls
	rl.EnableDepthTest()
}

// --- math helpers ---

// closestPointBetweenRays finds the closest approach between two rays.
// Returns (t1, t2, distance) where t1/t2 are parameters along each ray.
func closestPointBetweenRays(a, u, b, v rl.Vector3) (t1, t2, dist float32) {
	w := rl.Vector3Subtract(a, b)
	uu := rl.Vector3DotProduct(u, u)
	uv := rl.Vector3DotProduct(u, v)
	vv := rl.Vector3DotProduct(v, v)
	uw := rl.Vector3DotProduct(u, w)
	vw := rl.Vector3DotProduct(v, w)

	denom := uu*vv - uv*uv
	if denom < 1e-6 {
		return 0, 0, 999
	}

	t1 = (uv*vw - vv*uw) / denom
	t2 = (uu*vw - uv*uw) / denom

	p1 := rl.Vector3Add(a, rl.Vector3Scale(u, t1))
	p2 := rl.Vector3Add(b, rl.Vector3Scale(v, t2))
	dist = rl.Vector3Length(rl.Vector3Subtract(p1, p2))
	return
}

// rayPlaneIntersect returns where a ray hits a plane (defined by point + normal).
func rayPlaneIntersect(rayOrigin, rayDir, planePoint, planeNormal rl.Vector3) (rl.Vector3, bool) {
	denom := rl.Vector3DotProduct(rayDir, planeNormal)
	if math.Abs(float64(denom)) < 1e-6 {
		return rl.Vector3{}, false
	}
	t := rl.Vector3DotProduct(rl.Vector3Subtract(planePoint, rayOrigin), planeNormal) / denom
	if t < 0 {
		return rl.Vector3{}, false
	}
	return rl.Vector3Add(rayOrigin, rl.Vector3Scale(rayDir, t)), true
}

// drawRotatedBoxWires draws a wireframe box with rotation applied
func drawRotatedBoxWires(center, size, rotation rl.Vector3, color rl.Color) {
	// Build rotation matrix
	rx := float64(rotation.X) * math.Pi / 180
	ry := float64(rotation.Y) * math.Pi / 180
	rz := float64(rotation.Z) * math.Pi / 180
	rotX := rl.MatrixRotateX(float32(rx))
	rotY := rl.MatrixRotateY(float32(ry))
	rotZ := rl.MatrixRotateZ(float32(rz))
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

	// Half extents
	hx, hy, hz := size.X/2, size.Y/2, size.Z/2

	// 8 corners in local space
	corners := [8]rl.Vector3{
		{X: -hx, Y: -hy, Z: -hz},
		{X: hx, Y: -hy, Z: -hz},
		{X: hx, Y: hy, Z: -hz},
		{X: -hx, Y: hy, Z: -hz},
		{X: -hx, Y: -hy, Z: hz},
		{X: hx, Y: -hy, Z: hz},
		{X: hx, Y: hy, Z: hz},
		{X: -hx, Y: hy, Z: hz},
	}

	// Transform corners to world space
	for i := range corners {
		corners[i] = rl.Vector3Transform(corners[i], rotMatrix)
		corners[i] = rl.Vector3Add(corners[i], center)
	}

	// Draw 12 edges
	// Bottom face
	rl.DrawLine3D(corners[0], corners[1], color)
	rl.DrawLine3D(corners[1], corners[2], color)
	rl.DrawLine3D(corners[2], corners[3], color)
	rl.DrawLine3D(corners[3], corners[0], color)
	// Top face
	rl.DrawLine3D(corners[4], corners[5], color)
	rl.DrawLine3D(corners[5], corners[6], color)
	rl.DrawLine3D(corners[6], corners[7], color)
	rl.DrawLine3D(corners[7], corners[4], color)
	// Vertical edges
	rl.DrawLine3D(corners[0], corners[4], color)
	rl.DrawLine3D(corners[1], corners[5], color)
	rl.DrawLine3D(corners[2], corners[6], color)
	rl.DrawLine3D(corners[3], corners[7], color)
}

func absF(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
