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
}

func NewEditor(w *world.World) *Editor {
	return &Editor{
		world: w,
		camera: EditorCamera{
			MoveSpeed: 10.0,
		},
	}
}

func (e *Editor) Enter(currentCam rl.Camera3D) {
	e.Active = true
	rl.EnableCursor()

	// Initialize editor camera from current game camera
	e.camera.Position = currentCam.Position

	// Derive yaw/pitch from camera look direction
	dir := rl.Vector3Subtract(currentCam.Target, currentCam.Position)
	dir = rl.Vector3Normalize(dir)
	e.camera.Pitch = float32(math.Asin(float64(dir.Y))) * rl.Rad2deg
	e.camera.Yaw = float32(math.Atan2(float64(dir.Z), float64(dir.X))) * rl.Rad2deg
}

func (e *Editor) Exit() {
	e.Active = false
	e.Selected = nil
	rl.DisableCursor()
}

func (e *Editor) Update(deltaTime float32) {
	// Right-click: rotate camera + fly with WASD/QE
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

	// Left-click: select object via raycast
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		cam := e.GetRaylibCamera()
		ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), cam)
		hit, ok := e.world.Raycast(ray.Position, ray.Direction, 1000)
		if ok {
			e.Selected = hit.GameObject
		} else {
			e.Selected = nil
		}
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

// Draw3D draws selection wireframes. Call inside BeginMode3D/EndMode3D.
func (e *Editor) Draw3D() {
	if e.Selected == nil {
		return
	}

	if box := engine.GetComponent[*components.BoxCollider](e.Selected); box != nil {
		center := box.GetCenter()
		rl.DrawCubeWiresV(center, box.Size, rl.Yellow)
	} else if sphere := engine.GetComponent[*components.SphereCollider](e.Selected); sphere != nil {
		center := sphere.GetCenter()
		rl.DrawSphereWires(center, sphere.Radius, 8, 8, rl.Yellow)
	}
}

// DrawUI draws the editor overlay (mode indicator + inspector panel).
func (e *Editor) DrawUI() {
	rl.DrawText("EDITOR MODE", 10, 10, 24, rl.Yellow)
	rl.DrawText("RMB+Drag: Look | RMB+WASD: Fly | Q/E: Down/Up | Scroll: Speed | LMB: Select", 10, 40, 16, rl.LightGray)
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
