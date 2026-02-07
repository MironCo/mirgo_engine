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

const (
	gizmoLength  float32 = 2.0
	gizmoTipSize float32 = 0.12
	gizmoHitDist float32 = 0.3
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
	dragging        bool
	dragAxisIdx     int
	dragAxis        rl.Vector3
	dragPlaneNormal rl.Vector3
	dragStart       float32
	dragInitPos     rl.Vector3
	hoveredAxis     int // -1 = none, 0=X, 1=Y, 2=Z
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

	// Left-click: try gizmo first, then object selection
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
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

	center := e.Selected.Transform.Position
	bestDist := float32(999.0)
	bestAxis := -1

	for i, axis := range gizmoAxes {
		_, t2, dist := closestPointBetweenRays(ray.Position, ray.Direction, center, axis)
		if t2 > 0 && t2 < gizmoLength && dist < gizmoHitDist {
			if dist < bestDist {
				bestDist = dist
				bestAxis = i
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

	// Build a drag plane that contains the axis and faces the camera
	viewDir := rl.Vector3Normalize(rl.Vector3Subtract(e.dragInitPos, e.camera.Position))
	cross1 := rl.Vector3CrossProduct(viewDir, e.dragAxis)
	e.dragPlaneNormal = rl.Vector3Normalize(rl.Vector3CrossProduct(e.dragAxis, cross1))

	if pt, ok := rayPlaneIntersect(ray.Position, ray.Direction, e.dragInitPos, e.dragPlaneNormal); ok {
		e.dragStart = rl.Vector3DotProduct(rl.Vector3Subtract(pt, e.dragInitPos), e.dragAxis)
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
	e.Selected.Transform.Position = rl.Vector3Add(e.dragInitPos, rl.Vector3Scale(e.dragAxis, delta))
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
	center := e.Selected.Transform.Position
	tip := rl.Vector3{X: gizmoTipSize, Y: gizmoTipSize, Z: gizmoTipSize}

	for i, axis := range gizmoAxes {
		end := rl.Vector3Add(center, rl.Vector3Scale(axis, gizmoLength))

		color := gizmoColors[i]
		if e.dragging && e.dragAxisIdx == i {
			color = rl.Yellow
		} else if !e.dragging && e.hoveredAxis == i {
			color = rl.Yellow
		}

		rl.DrawLine3D(center, end, color)
		rl.DrawCubeV(end, tip, color)
	}
}

// DrawUI draws the editor overlay (mode indicator + inspector panel).
func (e *Editor) DrawUI() {
	rl.DrawText("EDITOR MODE", 10, 10, 24, rl.Yellow)
	rl.DrawText("RMB: Look/Fly | LMB: Select/Drag | Scroll: Speed | F2: Exit", 10, 40, 16, rl.LightGray)
	rl.DrawText(fmt.Sprintf("Speed: %.0f", e.camera.MoveSpeed), 10, 60, 16, rl.LightGray)

	if e.Selected == nil {
		return
	}

	// Inspector panel (bottom-left, dynamic height)
	comps := e.Selected.Components()
	panelX := int32(10)
	panelH := int32(138 + len(comps)*18)
	panelW := int32(320)
	panelY := int32(rl.GetScreenHeight()) - panelH - 10

	rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.NewColor(20, 20, 20, 200))
	rl.DrawRectangleLines(panelX, panelY, panelW, panelH, rl.Yellow)

	y := panelY + 8
	rl.DrawText(e.Selected.Name, panelX+10, y, 20, rl.Yellow)
	y += 28

	pos := e.Selected.Transform.Position
	rl.DrawText(fmt.Sprintf("Pos:   %.2f, %.2f, %.2f", pos.X, pos.Y, pos.Z), panelX+10, y, 16, rl.White)
	y += 22

	rot := e.Selected.Transform.Rotation
	rl.DrawText(fmt.Sprintf("Rot:   %.2f, %.2f, %.2f", rot.X, rot.Y, rot.Z), panelX+10, y, 16, rl.White)
	y += 22

	scale := e.Selected.Transform.Scale
	rl.DrawText(fmt.Sprintf("Scale: %.2f, %.2f, %.2f", scale.X, scale.Y, scale.Z), panelX+10, y, 16, rl.White)
	y += 28

	rl.DrawText("Components:", panelX+10, y, 16, rl.Gray)
	y += 22

	for _, c := range comps {
		typeName := reflect.TypeOf(c).Elem().Name()
		rl.DrawText("  "+typeName, panelX+10, y, 14, rl.LightGray)
		y += 18
	}
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
