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
	// Ensure depth testing is enabled for component gizmos (colliders, lights, cameras)
	// so they render correctly in 3D space
	rl.EnableDepthTest()

	// Draw component gizmos for all objects (depth-tested)
	for _, g := range e.world.Scene.GameObjects {
		e.drawAlwaysOnGizmos(g)
	}

	// Flush the depth-tested gizmos before switching modes
	rl.DrawRenderBatchActive()

	// Draw transform gizmo for selected object (always on top)
	e.drawSelectionGizmo()
}

// drawAlwaysOnGizmos draws gizmos that are always visible (not just when selected)
func (e *Editor) drawAlwaysOnGizmos(g *engine.GameObject) {
	isSelected := g == e.Selected

	// Point lights - always show
	if pl := engine.GetComponent[*components.PointLight](g); pl != nil {
		pos := pl.GetPosition()
		rl.DrawSphere(pos, 0.15, pl.Color)
		rl.DrawSphereWires(pos, pl.Radius, 8, 8, rl.Fade(pl.Color, 0.3))
	}

	// Box colliders - always show (green, yellow if selected)
	if box := engine.GetComponent[*components.BoxCollider](g); box != nil {
		center := box.GetCenter()
		rot := g.WorldRotation()
		color := rl.Fade(rl.Green, 0.5)
		if isSelected {
			color = rl.Yellow
		}
		drawRotatedBoxWires(center, box.GetWorldSize(), rot, color)
	}

	// Sphere colliders - always show
	if sphere := engine.GetComponent[*components.SphereCollider](g); sphere != nil {
		center := sphere.GetCenter()
		color := rl.Fade(rl.Green, 0.5)
		if isSelected {
			color = rl.Yellow
		}
		rl.DrawSphereWires(center, sphere.Radius, 8, 8, color)
	}

	// Character controllers - always show (green wireframe)
	if cc := engine.GetComponent[*components.CharacterController](g); cc != nil {
		pos := g.WorldPosition()
		size := rl.Vector3{
			X: cc.Radius * 2,
			Y: cc.Height,
			Z: cc.Radius * 2,
		}
		color := rl.Fade(rl.Green, 0.6)
		if isSelected {
			color = rl.Green
		}
		rl.DrawCubeWiresV(pos, size, color)
	}

	// Cameras - always show frustum
	if cam := engine.GetComponent[*components.Camera](g); cam != nil {
		e.drawCameraGizmo(g, cam, isSelected)
	}

	// Recurse into children
	for _, child := range g.Children {
		e.drawAlwaysOnGizmos(child)
	}
}

// drawCameraGizmo draws the camera frustum wireframe
func (e *Editor) drawCameraGizmo(g *engine.GameObject, cam *components.Camera, isSelected bool) {
	pos := g.WorldPosition()
	rot := g.WorldRotation()

	// Get forward direction from rotation
	yawRad := float64(rot.Y) * 3.14159265 / 180.0
	pitchRad := float64(rot.X) * 3.14159265 / 180.0
	forward := rl.Vector3{
		X: float32(-math.Sin(yawRad) * math.Cos(pitchRad)),
		Y: float32(-math.Sin(pitchRad)),
		Z: float32(-math.Cos(yawRad) * math.Cos(pitchRad)),
	}
	right := rl.Vector3{
		X: float32(math.Cos(yawRad)),
		Y: 0,
		Z: float32(-math.Sin(yawRad)),
	}
	up := rl.Vector3CrossProduct(right, forward)

	// Draw frustum lines
	nearDist := float32(0.5)
	farDist := float32(2.0)
	fovRad := float64(cam.FOV) * 3.14159265 / 180.0
	aspect := float32(1.7)
	nearH := nearDist * float32(math.Tan(fovRad/2))
	nearW := nearH * aspect
	farH := farDist * float32(math.Tan(fovRad/2))
	farW := farH * aspect

	// Near plane corners
	nearCenter := rl.Vector3Add(pos, rl.Vector3Scale(forward, nearDist))
	nearTL := rl.Vector3Add(nearCenter, rl.Vector3Add(rl.Vector3Scale(up, nearH), rl.Vector3Scale(right, -nearW)))
	nearTR := rl.Vector3Add(nearCenter, rl.Vector3Add(rl.Vector3Scale(up, nearH), rl.Vector3Scale(right, nearW)))
	nearBL := rl.Vector3Add(nearCenter, rl.Vector3Add(rl.Vector3Scale(up, -nearH), rl.Vector3Scale(right, -nearW)))
	nearBR := rl.Vector3Add(nearCenter, rl.Vector3Add(rl.Vector3Scale(up, -nearH), rl.Vector3Scale(right, nearW)))

	// Far plane corners
	farCenter := rl.Vector3Add(pos, rl.Vector3Scale(forward, farDist))
	farTL := rl.Vector3Add(farCenter, rl.Vector3Add(rl.Vector3Scale(up, farH), rl.Vector3Scale(right, -farW)))
	farTR := rl.Vector3Add(farCenter, rl.Vector3Add(rl.Vector3Scale(up, farH), rl.Vector3Scale(right, farW)))
	farBL := rl.Vector3Add(farCenter, rl.Vector3Add(rl.Vector3Scale(up, -farH), rl.Vector3Scale(right, -farW)))
	farBR := rl.Vector3Add(farCenter, rl.Vector3Add(rl.Vector3Scale(up, -farH), rl.Vector3Scale(right, farW)))

	lineColor := rl.Fade(rl.White, 0.5)
	coneColor := rl.Fade(rl.Yellow, 0.5)
	if isSelected {
		lineColor = rl.White
		coneColor = rl.Yellow
	}

	// Draw near plane
	rl.DrawLine3D(nearTL, nearTR, lineColor)
	rl.DrawLine3D(nearTR, nearBR, lineColor)
	rl.DrawLine3D(nearBR, nearBL, lineColor)
	rl.DrawLine3D(nearBL, nearTL, lineColor)

	// Draw far plane
	rl.DrawLine3D(farTL, farTR, lineColor)
	rl.DrawLine3D(farTR, farBR, lineColor)
	rl.DrawLine3D(farBR, farBL, lineColor)
	rl.DrawLine3D(farBL, farTL, lineColor)

	// Draw edges connecting near to far
	rl.DrawLine3D(nearTL, farTL, lineColor)
	rl.DrawLine3D(nearTR, farTR, lineColor)
	rl.DrawLine3D(nearBL, farBL, lineColor)
	rl.DrawLine3D(nearBR, farBR, lineColor)

	// Draw camera icon (small pyramid at position)
	rl.DrawLine3D(pos, nearTL, coneColor)
	rl.DrawLine3D(pos, nearTR, coneColor)
	rl.DrawLine3D(pos, nearBL, coneColor)
	rl.DrawLine3D(pos, nearBR, coneColor)
}

func (e *Editor) drawSelectionGizmo() {
	if e.Selected == nil {
		return
	}

	// Disable depth testing so transform gizmo always draws on top
	rl.DrawRenderBatchActive()
	rl.DisableDepthTest()

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
