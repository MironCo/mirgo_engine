//go:build !game

package game

import (
	"fmt"
	"math"
	"reflect"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/world"

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

type EditorCamera struct {
	Position  rl.Vector3
	Yaw       float32
	Pitch     float32
	MoveSpeed float32
}

type Editor struct {
	Active   bool
	camera   EditorCamera
	Selected *engine.GameObject
	world    *world.World

	// Gizmo state
	gizmoMode       GizmoMode
	dragging        bool
	dragAxisIdx     int
	dragAxis        rl.Vector3
	dragPlaneNormal rl.Vector3
	dragStart       float32
	dragInitPos     rl.Vector3
	dragInitRot     rl.Vector3
	dragInitScale   rl.Vector3
	hoveredAxis     int // -1 = none, 0=X, 1=Y, 2=Z

	// Hierarchy panel
	hierarchyScroll int32

	// Save feedback
	saveMsg     string
	saveMsgTime float64
}

func NewEditor(w *world.World) *Editor {
	return &Editor{
		world: w,
		camera: EditorCamera{
			MoveSpeed: 10.0,
		},
		hoveredAxis: -1,
	}
}

func (e *Editor) Enter(currentCam rl.Camera3D) {
	e.Active = true
	rl.EnableCursor()

	// Reload scene from disk to undo all play mode changes
	e.world.ResetScene()
	e.Selected = nil

	e.camera.Position = currentCam.Position

	dir := rl.Vector3Subtract(currentCam.Target, currentCam.Position)
	dir = rl.Vector3Normalize(dir)
	e.camera.Pitch = float32(math.Asin(float64(dir.Y))) * rl.Rad2deg
	e.camera.Yaw = float32(math.Atan2(float64(dir.Z), float64(dir.X))) * rl.Rad2deg
}

func (e *Editor) Exit() {
	e.Active = false
	e.Selected = nil
	e.dragging = false
	e.hoveredAxis = -1
	rl.DisableCursor()
}

func (e *Editor) Update(deltaTime float32) {
	// Ctrl+S: save scene
	if rl.IsKeyDown(rl.KeyLeftControl) && rl.IsKeyPressed(rl.KeyS) {
		if err := e.world.SaveScene(world.ScenePath); err != nil {
			e.saveMsg = fmt.Sprintf("Save failed: %v", err)
		} else {
			e.saveMsg = "Scene saved!"
		}
		e.saveMsgTime = rl.GetTime()
	}

	// Camera: right-click + drag to look, right-click + WASD to fly
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		mouseDelta := rl.GetMouseDelta()
		e.camera.Yaw += mouseDelta.X * 0.1
		e.camera.Pitch -= mouseDelta.Y * 0.1
		if e.camera.Pitch > 89 {
			e.camera.Pitch = 89
		}
		if e.camera.Pitch < -89 {
			e.camera.Pitch = -89
		}

		forward, right := e.getDirections()
		speed := e.camera.MoveSpeed * deltaTime

		if rl.IsKeyDown(rl.KeyW) {
			e.camera.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(forward, speed))
		}
		if rl.IsKeyDown(rl.KeyS) {
			e.camera.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(forward, -speed))
		}
		if rl.IsKeyDown(rl.KeyA) {
			e.camera.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(right, speed))
		}
		if rl.IsKeyDown(rl.KeyD) {
			e.camera.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(right, -speed))
		}
		if rl.IsKeyDown(rl.KeyE) {
			e.camera.Position.Y += speed
		}
		if rl.IsKeyDown(rl.KeyQ) {
			e.camera.Position.Y -= speed
		}
	}

	// Scroll wheel adjusts fly speed
	scroll := rl.GetMouseWheelMove()
	if scroll != 0 {
		e.camera.MoveSpeed += scroll * 2.0
		if e.camera.MoveSpeed < 1.0 {
			e.camera.MoveSpeed = 1.0
		}
		if e.camera.MoveSpeed > 100.0 {
			e.camera.MoveSpeed = 100.0
		}
	}

	// Gizmo mode hotkeys (only when not holding RMB for camera)
	if !rl.IsMouseButtonDown(rl.MouseRightButton) {
		if rl.IsKeyPressed(rl.KeyW) {
			e.gizmoMode = GizmoMove
		}
		if rl.IsKeyPressed(rl.KeyE) {
			e.gizmoMode = GizmoRotate
		}
		if rl.IsKeyPressed(rl.KeyR) {
			e.gizmoMode = GizmoScale
		}
	}

	cam := e.GetRaylibCamera()
	ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), cam)

	// Handle active drag
	if e.dragging {
		if !rl.IsMouseButtonDown(rl.MouseLeftButton) {
			e.dragging = false
		} else {
			e.updateDrag(ray)
		}
		return
	}

	// Update hovered axis for visual feedback
	e.hoveredAxis = -1
	if e.Selected != nil {
		e.hoveredAxis = e.pickGizmoAxis(ray)
	}

	// Left-click: skip 3D interaction if mouse is over a UI panel
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !e.mouseInPanel() {
		if e.Selected != nil {
			axisIdx := e.pickGizmoAxis(ray)
			if axisIdx >= 0 {
				e.startDrag(axisIdx, ray)
				return
			}
		}

		hit, ok := e.world.Raycast(ray.Position, ray.Direction, 1000)
		if ok {
			e.Selected = hit.GameObject
		} else {
			e.Selected = nil
		}
	}
}

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
	e.dragging = true
	e.dragAxisIdx = axisIdx
	e.dragAxis = gizmoAxes[axisIdx]
	e.dragInitPos = e.Selected.Transform.Position
	e.dragInitRot = e.Selected.Transform.Rotation
	e.dragInitScale = e.Selected.Transform.Scale

	// Build a drag plane using world position for correct 3D picking
	worldPos := e.Selected.WorldPosition()
	viewDir := rl.Vector3Normalize(rl.Vector3Subtract(worldPos, e.camera.Position))
	cross1 := rl.Vector3CrossProduct(viewDir, e.dragAxis)
	e.dragPlaneNormal = rl.Vector3Normalize(rl.Vector3CrossProduct(e.dragAxis, cross1))

	if pt, ok := rayPlaneIntersect(ray.Position, ray.Direction, worldPos, e.dragPlaneNormal); ok {
		e.dragStart = rl.Vector3DotProduct(rl.Vector3Subtract(pt, worldPos), e.dragAxis)
	}
}

func (e *Editor) updateDrag(ray rl.Ray) {
	if e.Selected == nil {
		e.dragging = false
		return
	}

	pt, ok := rayPlaneIntersect(ray.Position, ray.Direction, e.dragInitPos, e.dragPlaneNormal)
	if !ok {
		return
	}

	currentT := rl.Vector3DotProduct(rl.Vector3Subtract(pt, e.dragInitPos), e.dragAxis)
	delta := currentT - e.dragStart

	switch e.gizmoMode {
	case GizmoMove:
		e.Selected.Transform.Position = rl.Vector3Add(e.dragInitPos, rl.Vector3Scale(e.dragAxis, delta))

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

func (e *Editor) getDirections() (forward, right rl.Vector3) {
	yawRad := float64(e.camera.Yaw) * math.Pi / 180
	pitchRad := float64(e.camera.Pitch) * math.Pi / 180

	forward = rl.Vector3{
		X: float32(math.Cos(yawRad) * math.Cos(pitchRad)),
		Y: float32(math.Sin(pitchRad)),
		Z: float32(math.Sin(yawRad) * math.Cos(pitchRad)),
	}
	right = rl.Vector3{
		X: float32(math.Sin(yawRad)),
		Y: 0,
		Z: float32(-math.Cos(yawRad)),
	}
	return
}

func (e *Editor) GetRaylibCamera() rl.Camera3D {
	forward, _ := e.getDirections()
	target := rl.Vector3Add(e.camera.Position, forward)
	return rl.Camera3D{
		Position:   e.camera.Position,
		Target:     target,
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}
}

// Draw3D draws selection wireframes and gizmo. Call inside BeginMode3D/EndMode3D.
func (e *Editor) Draw3D() {
	if e.Selected == nil {
		return
	}

	// Selection wireframe
	if box := engine.GetComponent[*components.BoxCollider](e.Selected); box != nil {
		center := box.GetCenter()
		rl.DrawCubeWiresV(center, box.Size, rl.Yellow)
	} else if sphere := engine.GetComponent[*components.SphereCollider](e.Selected); sphere != nil {
		center := sphere.GetCenter()
		rl.DrawSphereWires(center, sphere.Radius, 8, 8, rl.Yellow)
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
}

// DrawUI draws the editor overlay: top bar, hierarchy panel (left), inspector panel (right).
func (e *Editor) DrawUI() {
	// Top bar
	rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), 32, rl.NewColor(20, 20, 20, 220))
	rl.DrawText("EDITOR", 10, 6, 20, rl.Yellow)
	// Gizmo mode indicator
	modeNames := [3]string{"[W] Move", "[E] Rotate", "[R] Scale"}
	for i, name := range modeNames {
		x := int32(80 + i*90)
		color := rl.Gray
		if GizmoMode(i) == e.gizmoMode {
			color = rl.Yellow
		}
		rl.DrawText(name, x, 8, 16, color)
	}
	rl.DrawText("| Ctrl+S: Save | F2: Play", 350, 8, 16, rl.LightGray)
	rl.DrawText(fmt.Sprintf("Speed: %.0f", e.camera.MoveSpeed), int32(rl.GetScreenWidth())-100, 8, 16, rl.LightGray)

	// Save message flash
	if e.saveMsg != "" && rl.GetTime()-e.saveMsgTime < 2.0 {
		color := rl.Green
		if e.saveMsg != "Scene saved!" {
			color = rl.Red
		}
		rl.DrawText(e.saveMsg, int32(rl.GetScreenWidth()/2)-50, 8, 20, color)
	}

	e.drawHierarchy()
	e.drawInspector()
}

// drawHierarchy draws the scene hierarchy panel on the left.
func (e *Editor) drawHierarchy() {
	panelX := int32(0)
	panelY := int32(32)
	panelW := int32(200)
	panelH := int32(rl.GetScreenHeight()) - panelY

	rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.NewColor(25, 25, 30, 230))
	rl.DrawLine(panelX+panelW, panelY, panelX+panelW, panelY+panelH, rl.NewColor(60, 60, 60, 255))

	rl.DrawText("Hierarchy", panelX+8, panelY+6, 16, rl.Gray)
	y := panelY + 28

	// Scroll with mouse wheel when hovering hierarchy
	mousePos := rl.GetMousePosition()
	mouseInPanel := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH)

	if mouseInPanel && !rl.IsMouseButtonDown(rl.MouseRightButton) {
		scroll := rl.GetMouseWheelMove()
		e.hierarchyScroll -= int32(scroll * 20)
		if e.hierarchyScroll < 0 {
			e.hierarchyScroll = 0
		}
	}

	itemH := int32(22)
	objects := e.world.Scene.GameObjects
	maxScroll := int32(len(objects))*itemH - panelH + 30
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.hierarchyScroll > maxScroll {
		e.hierarchyScroll = maxScroll
	}

	// Clip to panel area
	rl.BeginScissorMode(panelX, panelY+24, panelW, panelH-24)

	for i, g := range objects {
		itemY := y + int32(i)*itemH - e.hierarchyScroll

		// Skip if off screen
		if itemY+itemH < panelY+24 || itemY > panelY+panelH {
			continue
		}

		// Hover highlight
		hovered := mouseInPanel && mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH)
		selected := e.Selected == g

		if selected {
			rl.DrawRectangle(panelX, itemY, panelW, itemH, rl.NewColor(80, 80, 20, 180))
		} else if hovered {
			rl.DrawRectangle(panelX, itemY, panelW, itemH, rl.NewColor(50, 50, 50, 150))
		}

		// Click to select
		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.Selected = g
		}

		// Compute depth for indentation
		depth := int32(0)
		p := g.Parent
		for p != nil {
			depth++
			p = p.Parent
		}
		indent := int32(12) + depth*16

		textColor := rl.LightGray
		if selected {
			textColor = rl.Yellow
		}
		rl.DrawText(g.Name, panelX+indent, itemY+4, 14, textColor)
	}

	rl.EndScissorMode()
}

// drawInspector draws the selected object's inspector on the right.
func (e *Editor) drawInspector() {
	if e.Selected == nil {
		return
	}

	panelW := int32(300)
	panelX := int32(rl.GetScreenWidth()) - panelW
	panelY := int32(32)
	panelH := int32(rl.GetScreenHeight()) - panelY

	rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.NewColor(25, 25, 30, 230))
	rl.DrawLine(panelX, panelY, panelX, panelY+panelH, rl.NewColor(60, 60, 60, 255))

	y := panelY + 8

	// Name
	rl.DrawText(e.Selected.Name, panelX+10, y, 20, rl.Yellow)
	y += 28

	// Tags
	if tags := e.Selected.Tags; len(tags) > 0 {
		tagStr := ""
		for i, t := range tags {
			if i > 0 {
				tagStr += ", "
			}
			tagStr += t
		}
		rl.DrawText("Tags: "+tagStr, panelX+10, y, 14, rl.Gray)
		y += 20
	}

	// Separator
	rl.DrawLine(panelX+10, y+2, panelX+panelW-10, y+2, rl.NewColor(60, 60, 60, 255))
	y += 10

	// Transform
	rl.DrawText("Transform", panelX+10, y, 16, rl.Gray)
	y += 22

	pos := e.Selected.Transform.Position
	rl.DrawText(fmt.Sprintf("Pos   %.2f, %.2f, %.2f", pos.X, pos.Y, pos.Z), panelX+14, y, 14, rl.White)
	y += 18

	if e.Selected.Parent != nil {
		wPos := e.Selected.WorldPosition()
		rl.DrawText(fmt.Sprintf("World %.2f, %.2f, %.2f", wPos.X, wPos.Y, wPos.Z), panelX+14, y, 12, rl.Gray)
		y += 16
	}

	rot := e.Selected.Transform.Rotation
	rl.DrawText(fmt.Sprintf("Rot   %.2f, %.2f, %.2f", rot.X, rot.Y, rot.Z), panelX+14, y, 14, rl.White)
	y += 18

	scale := e.Selected.Transform.Scale
	rl.DrawText(fmt.Sprintf("Scale %.2f, %.2f, %.2f", scale.X, scale.Y, scale.Z), panelX+14, y, 14, rl.White)
	y += 24

	// Separator
	rl.DrawLine(panelX+10, y+2, panelX+panelW-10, y+2, rl.NewColor(60, 60, 60, 255))
	y += 10

	// Components
	rl.DrawText("Components", panelX+10, y, 16, rl.Gray)
	y += 22

	for _, c := range e.Selected.Components() {
		typeName := reflect.TypeOf(c).Elem().Name()
		rl.DrawText(typeName, panelX+14, y, 14, rl.LightGray)
		y += 18
	}
}

// mouseInPanel returns true if the mouse is over the hierarchy or inspector panel.
func (e *Editor) mouseInPanel() bool {
	m := rl.GetMousePosition()
	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())
	// Hierarchy: left 200px
	if m.X <= 200 && m.Y >= 32 && m.Y <= screenH {
		return true
	}
	// Inspector: right 300px
	if m.X >= screenW-300 && m.Y >= 32 && m.Y <= screenH {
		return true
	}
	// Top bar
	if m.Y <= 32 {
		return true
	}
	return false
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
