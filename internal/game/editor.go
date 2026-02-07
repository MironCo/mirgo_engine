//go:build !game

package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
	gui "github.com/gen2brain/raylib-go/raygui"
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
	gizmoMode        GizmoMode
	dragging         bool
	dragAxisIdx      int
	dragAxis         rl.Vector3
	dragPlaneNormal  rl.Vector3
	dragStart        float32
	dragInitPos      rl.Vector3 // Local position
	dragInitWorldPos rl.Vector3 // World position (for drag plane math)
	dragInitRot      rl.Vector3
	dragInitScale    rl.Vector3
	hoveredAxis      int // -1 = none, 0=X, 1=Y, 2=Z

	// Hierarchy panel
	hierarchyScroll int32

	// Inspector panel
	inspectorScroll      int32
	showAddComponentMenu bool

	// Float field editing state
	activeInputID     string  // e.g., "pos.x", "rot.y", "mass"
	inputTextValue    string  // current text being edited
	fieldDragging     bool    // true if drag-scrubbing a field
	fieldDragID       string  // which field is being dragged
	fieldDragStartX   float32 // mouse X when drag started
	fieldDragStartVal float32 // value when drag started
	fieldHoveredAny   bool    // true if any float field is hovered this frame

	// Save feedback
	saveMsg     string
	saveMsgTime float64

	// Undo stack
	undoStack []UndoState

	// Asset browser
	showAssetBrowser bool
	assetBrowserScroll int32
	assetFiles       []AssetEntry
}

// AssetEntry represents a model file in the asset browser
type AssetEntry struct {
	Name     string // Display name
	Path     string // Full path to the model file
	IsFolder bool   // True if this is a folder
}

func NewEditor(w *world.World) *Editor {
	return &Editor{
		world: w,
		camera: EditorCamera{
			MoveSpeed: 10.0,
		},
		hoveredAxis: -1,
		undoStack:   make([]UndoState, 0, maxUndoStack),
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

	// Initialize raygui dark style
	initRayguiStyle()
}

// editorFont holds the custom font for the editor UI (zero value = use default)
var editorFont rl.Font
var editorFontLoaded bool

// initRayguiStyle sets up a dark theme for raygui widgets
func initRayguiStyle() {
	// Load a nicer font if available (only once)
	if !editorFontLoaded {
		editorFontLoaded = true
		if _, err := os.Stat("assets/fonts/editor.ttf"); err == nil {
			editorFont = rl.LoadFontEx("assets/fonts/editor.ttf", 16, nil)
			gui.SetFont(editorFont)
		}
	}

	// Background colors
	gui.SetStyle(gui.DEFAULT, gui.BACKGROUND_COLOR, gui.NewColorPropertyValue(rl.NewColor(30, 30, 35, 255)))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_NORMAL, gui.NewColorPropertyValue(rl.NewColor(45, 45, 50, 255)))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_FOCUSED, gui.NewColorPropertyValue(rl.NewColor(60, 60, 70, 255)))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_PRESSED, gui.NewColorPropertyValue(rl.NewColor(70, 80, 90, 255)))

	// Text colors
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_NORMAL, gui.NewColorPropertyValue(rl.NewColor(200, 200, 200, 255)))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_FOCUSED, gui.NewColorPropertyValue(rl.White))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_PRESSED, gui.NewColorPropertyValue(rl.Yellow))

	// Border colors
	gui.SetStyle(gui.DEFAULT, gui.BORDER_COLOR_NORMAL, gui.NewColorPropertyValue(rl.NewColor(80, 80, 90, 255)))
	gui.SetStyle(gui.DEFAULT, gui.BORDER_COLOR_FOCUSED, gui.NewColorPropertyValue(rl.NewColor(100, 100, 120, 255)))

	// Line color (for separators)
	gui.SetStyle(gui.DEFAULT, gui.LINE_COLOR, gui.NewColorPropertyValue(rl.NewColor(60, 60, 60, 255)))

	// Text size
	gui.SetStyle(gui.DEFAULT, gui.TEXT_SIZE, 14)
}

func (e *Editor) Exit() {
	e.Active = false
	e.Selected = nil
	e.dragging = false
	e.hoveredAxis = -1
	rl.DisableCursor()
}

func (e *Editor) Update(deltaTime float32) {
	// Handle file drops (GLTF models, etc.)
	e.handleFileDrop()

	// Ctrl+Z or Cmd+Z: undo
	if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyLeftSuper)) && rl.IsKeyPressed(rl.KeyZ) {
		e.undo()
	}

	// Ctrl+S: save scene
	if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyLeftSuper)) && rl.IsKeyPressed(rl.KeyS) {
		if err := e.world.SaveScene(world.ScenePath); err != nil {
			e.saveMsg = fmt.Sprintf("Save failed: %v", err)
		} else {
			e.saveMsg = "Scene saved!"
		}
		e.saveMsgTime = rl.GetTime()
	}

	// Tab: toggle asset browser
	if rl.IsKeyPressed(rl.KeyTab) {
		e.showAssetBrowser = !e.showAssetBrowser
		if e.showAssetBrowser {
			e.scanAssetModels()
		}
	}

	// Cmd+Delete (Mac) or Ctrl+Delete: delete selected object
	if e.Selected != nil && (rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyLeftControl)) && rl.IsKeyPressed(rl.KeyBackspace) {
		e.deleteSelectedObject()
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
		rot := e.Selected.WorldRotation()
		scale := e.Selected.WorldScale()
		scaledSize := rl.Vector3{
			X: box.Size.X * scale.X,
			Y: box.Size.Y * scale.Y,
			Z: box.Size.Z * scale.Z,
		}
		drawRotatedBoxWires(center, scaledSize, rot, rl.Yellow)
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
	rl.DrawText("| Ctrl+S: Save | Ctrl+Z: Undo | F2: Play", 350, 8, 16, rl.LightGray)
	rl.DrawText(fmt.Sprintf("Speed: %.0f", e.camera.MoveSpeed), int32(rl.GetScreenWidth())-100, 8, 16, rl.LightGray)

	// Save message flash
	if e.saveMsg != "" && rl.GetTime()-e.saveMsgTime < 2.0 {
		color := rl.Green
		if e.saveMsg != "Scene saved!" {
			color = rl.Red
		}
		rl.DrawText(e.saveMsg, int32(rl.GetScreenWidth()/2)-50, 8, 20, color)
	}

	// Reset field hover tracking for this frame
	e.fieldHoveredAny = false

	e.drawHierarchy()
	e.drawInspector()

	// Asset browser toggle button in top bar
	abBtnX := int32(rl.GetScreenWidth()) - 200
	abBtnW := int32(90)
	abBtnH := int32(22)
	abBtnY := int32(5)

	mousePos := rl.GetMousePosition()
	abHovered := mousePos.X >= float32(abBtnX) && mousePos.X <= float32(abBtnX+abBtnW) &&
		mousePos.Y >= float32(abBtnY) && mousePos.Y <= float32(abBtnY+abBtnH)

	abBtnColor := rl.NewColor(50, 50, 60, 200)
	if e.showAssetBrowser {
		abBtnColor = rl.NewColor(60, 80, 60, 220)
	} else if abHovered {
		abBtnColor = rl.NewColor(70, 70, 80, 220)
	}
	rl.DrawRectangle(abBtnX, abBtnY, abBtnW, abBtnH, abBtnColor)
	rl.DrawText("Assets [Tab]", abBtnX+6, abBtnY+4, 14, rl.LightGray)

	if abHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.showAssetBrowser = !e.showAssetBrowser
		if e.showAssetBrowser {
			e.scanAssetModels()
		}
	}

	if e.showAssetBrowser {
		e.drawAssetBrowser()
	}

	// Set cursor based on field hover state
	if e.fieldHoveredAny || e.fieldDragging {
		rl.SetMouseCursor(rl.MouseCursorResizeEW)
	} else {
		rl.SetMouseCursor(rl.MouseCursorDefault)
	}
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

	// "New Object" button
	btnX := panelX + panelW - 55
	btnY := panelY + 4
	btnW := int32(50)
	btnH := int32(18)

	mousePos := rl.GetMousePosition()
	btnHovered := mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
		mousePos.Y >= float32(btnY) && mousePos.Y <= float32(btnY+btnH)

	btnColor := rl.NewColor(50, 70, 50, 200)
	if btnHovered {
		btnColor = rl.NewColor(70, 100, 70, 220)
	}
	rl.DrawRectangle(btnX, btnY, btnW, btnH, btnColor)
	rl.DrawText("+ New", btnX+6, btnY+2, 14, rl.LightGray)

	clickedNewButton := false
	if btnHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.createNewGameObject()
		clickedNewButton = true
	}

	y := panelY + 28

	// Scroll with mouse wheel when hovering hierarchy
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

		// Click to select (but not if we just clicked the New button)
		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !clickedNewButton {
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

	// Check for scroll input when mouse is in inspector
	mousePos := rl.GetMousePosition()
	mouseInPanel := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH)

	if mouseInPanel && !rl.IsMouseButtonDown(rl.MouseRightButton) && !e.showAddComponentMenu {
		scroll := rl.GetMouseWheelMove()
		e.inspectorScroll -= int32(scroll * 20)
		if e.inspectorScroll < 0 {
			e.inspectorScroll = 0
		}
	}

	// Begin scissor mode for scrolling
	rl.BeginScissorMode(panelX, panelY, panelW, panelH)

	y := panelY + 8 - e.inspectorScroll

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

	// Transform section
	y = e.drawTransformSection(panelX, y, panelW)

	// Separator
	rl.DrawLine(panelX+10, y+2, panelX+panelW-10, y+2, rl.NewColor(60, 60, 60, 255))
	y += 10

	// Components section header
	rl.DrawText("Components", panelX+10, y, 16, rl.Gray)
	y += 22

	// Draw each component with properties and remove button
	comps := e.Selected.Components()
	removeIdx := -1
	for i, c := range comps {
		newY, shouldRemove := e.drawComponentEntry(panelX, y, panelW, i, c, mouseInPanel)
		if shouldRemove {
			removeIdx = i
		}
		y = newY + 8 // spacing between components
	}

	// Handle removal (deferred to avoid modifying slice during iteration)
	if removeIdx >= 0 {
		e.removeComponentAtIndex(removeIdx)
	}

	// Add Component button
	y += 10
	btnW := panelW - 40
	btnH := int32(24)
	btnX := panelX + 20
	btnY := y

	hovered := mouseInPanel && mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
		mousePos.Y >= float32(btnY+e.inspectorScroll) && mousePos.Y <= float32(btnY+btnH+e.inspectorScroll)

	btnColor := rl.NewColor(50, 70, 50, 220)
	if hovered {
		btnColor = rl.NewColor(70, 100, 70, 220)
	}
	rl.DrawRectangle(btnX, btnY, btnW, btnH, btnColor)
	textW := rl.MeasureText("+ Add Component", 14)
	rl.DrawText("+ Add Component", btnX+(btnW-textW)/2, btnY+5, 14, rl.LightGray)

	clickedAddButton := false
	if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.showAddComponentMenu = !e.showAddComponentMenu
		clickedAddButton = true
	}

	rl.EndScissorMode()

	// Draw add component dropdown menu (outside scissor mode so it's not clipped)
	if e.showAddComponentMenu {
		e.drawAddComponentMenu(btnX, btnY+btnH-e.inspectorScroll, btnW, clickedAddButton)
	}

	// Clamp scroll to content
	totalHeight := y + e.inspectorScroll - panelY + 100
	maxScroll := totalHeight - panelH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.inspectorScroll > maxScroll {
		e.inspectorScroll = maxScroll
	}
}

// drawTransformSection draws the transform properties and returns the new Y position.
func (e *Editor) drawTransformSection(panelX, y, panelW int32) int32 {
	rl.DrawText("Transform", panelX+10, y, 16, rl.Gray)
	y += 22

	labelW := int32(35)
	fieldW := (panelW - 30 - labelW) / 3
	fieldH := int32(20)
	startX := panelX + 10 + labelW

	// Position
	rl.DrawText("Pos", panelX+12, y+2, 14, rl.LightGray)
	e.Selected.Transform.Position.X = e.drawFloatField(startX, y, fieldW, fieldH, "pos.x", e.Selected.Transform.Position.X)
	e.Selected.Transform.Position.Y = e.drawFloatField(startX+fieldW+2, y, fieldW, fieldH, "pos.y", e.Selected.Transform.Position.Y)
	e.Selected.Transform.Position.Z = e.drawFloatField(startX+2*(fieldW+2), y, fieldW, fieldH, "pos.z", e.Selected.Transform.Position.Z)
	y += fieldH + 4

	if e.Selected.Parent != nil {
		wPos := e.Selected.WorldPosition()
		rl.DrawText(fmt.Sprintf("World %.1f, %.1f, %.1f", wPos.X, wPos.Y, wPos.Z), panelX+14, y, 11, rl.Gray)
		y += 14
	}

	// Rotation
	rl.DrawText("Rot", panelX+12, y+2, 14, rl.LightGray)
	e.Selected.Transform.Rotation.X = e.drawFloatField(startX, y, fieldW, fieldH, "rot.x", e.Selected.Transform.Rotation.X)
	e.Selected.Transform.Rotation.Y = e.drawFloatField(startX+fieldW+2, y, fieldW, fieldH, "rot.y", e.Selected.Transform.Rotation.Y)
	e.Selected.Transform.Rotation.Z = e.drawFloatField(startX+2*(fieldW+2), y, fieldW, fieldH, "rot.z", e.Selected.Transform.Rotation.Z)
	y += fieldH + 4

	// Scale
	rl.DrawText("Scale", panelX+12, y+2, 14, rl.LightGray)
	e.Selected.Transform.Scale.X = e.drawFloatField(startX, y, fieldW, fieldH, "scale.x", e.Selected.Transform.Scale.X)
	e.Selected.Transform.Scale.Y = e.drawFloatField(startX+fieldW+2, y, fieldW, fieldH, "scale.y", e.Selected.Transform.Scale.Y)
	e.Selected.Transform.Scale.Z = e.drawFloatField(startX+2*(fieldW+2), y, fieldW, fieldH, "scale.z", e.Selected.Transform.Scale.Z)
	y += fieldH + 8

	return y
}

// drawFloatField draws an editable float input field with drag-to-scrub support.
func (e *Editor) drawFloatField(x, y, w, h int32, id string, value float32) float32 {
	mousePos := rl.GetMousePosition()
	hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
		mousePos.Y >= float32(y) && mousePos.Y <= float32(y+h)

	editMode := e.activeInputID == id
	isDragging := e.fieldDragging && e.fieldDragID == id

	// Track if any field is hovered (for cursor management)
	if hovered && !editMode {
		e.fieldHoveredAny = true
	}

	// Background color
	bgColor := rl.NewColor(45, 45, 50, 255)
	if editMode {
		bgColor = rl.NewColor(60, 60, 70, 255)
	} else if hovered || isDragging {
		bgColor = rl.NewColor(55, 55, 60, 255)
	}
	rl.DrawRectangle(x, y, w, h, bgColor)
	rl.DrawRectangleLines(x, y, w, h, rl.NewColor(80, 80, 90, 255))

	// Handle drag-to-scrub (when not in edit mode)
	if !editMode {
		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			// Start drag
			e.fieldDragging = true
			e.fieldDragID = id
			e.fieldDragStartX = mousePos.X
			e.fieldDragStartVal = value
		}

		if isDragging {
			if rl.IsMouseButtonDown(rl.MouseLeftButton) {
				// Update value based on drag distance
				deltaX := mousePos.X - e.fieldDragStartX
				// Sensitivity: 100 pixels = 1.0 change, hold shift for fine control
				sensitivity := float32(0.01)
				if rl.IsKeyDown(rl.KeyLeftShift) {
					sensitivity = 0.001
				}
				value = e.fieldDragStartVal + deltaX*sensitivity
			} else {
				// End drag
				dragDist := mousePos.X - e.fieldDragStartX
				if dragDist > -2 && dragDist < 2 {
					// Was a click, not a drag - enter edit mode
					e.activeInputID = id
					e.inputTextValue = strconv.FormatFloat(float64(value), 'f', 2, 32)
				}
				e.fieldDragging = false
				e.fieldDragID = ""
			}
		}
	}

	// Text display/editing
	if editMode {
		// Draw text input
		rl.DrawText(e.inputTextValue+"_", x+4, y+3, 14, rl.White)

		// Handle typing
		for {
			key := rl.GetCharPressed()
			if key == 0 {
				break
			}
			ch := rune(key)
			// Allow digits, minus, dot
			if (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' {
				e.inputTextValue += string(ch)
			}
		}

		// Backspace
		if rl.IsKeyPressed(rl.KeyBackspace) && len(e.inputTextValue) > 0 {
			e.inputTextValue = e.inputTextValue[:len(e.inputTextValue)-1]
		}

		// Enter or click outside to confirm
		clickedOutside := rl.IsMouseButtonPressed(rl.MouseLeftButton) && !hovered
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) || clickedOutside || rl.IsKeyPressed(rl.KeyTab) {
			if e.inputTextValue != "" {
				if parsed, err := strconv.ParseFloat(e.inputTextValue, 32); err == nil {
					value = float32(parsed)
				}
			}
			e.activeInputID = ""
			e.inputTextValue = ""
		}

		// Escape to cancel
		if rl.IsKeyPressed(rl.KeyEscape) {
			e.activeInputID = ""
			e.inputTextValue = ""
		}
	} else {
		// Display current value
		text := strconv.FormatFloat(float64(value), 'f', 2, 32)
		rl.DrawText(text, x+4, y+3, 14, rl.LightGray)
	}

	return value
}

// drawComponentEntry draws a single component with its properties and X button.
// Returns the new Y position and whether the component should be removed.
func (e *Editor) drawComponentEntry(panelX, y, panelW int32, index int, c engine.Component, mouseInPanel bool) (int32, bool) {
	typeName := reflect.TypeOf(c).Elem().Name()

	// Component header with X button
	headerH := int32(20)
	xBtnSize := int32(16)
	xBtnX := panelX + panelW - 30
	xBtnY := y + 2

	// Draw header background
	rl.DrawRectangle(panelX+10, y, panelW-20, headerH, rl.NewColor(40, 40, 45, 200))
	rl.DrawText(typeName, panelX+14, y+3, 14, rl.LightGray)

	// Draw X button
	mousePos := rl.GetMousePosition()
	// Adjust for scroll when checking hover
	adjustedY := float32(xBtnY + e.inspectorScroll)
	xHovered := mouseInPanel &&
		mousePos.X >= float32(xBtnX) && mousePos.X <= float32(xBtnX+xBtnSize) &&
		mousePos.Y >= adjustedY-float32(e.inspectorScroll) && mousePos.Y <= adjustedY+float32(xBtnSize)-float32(e.inspectorScroll)

	xBtnColor := rl.NewColor(100, 50, 50, 200)
	if xHovered {
		xBtnColor = rl.NewColor(150, 60, 60, 220)
	}
	rl.DrawRectangle(xBtnX, xBtnY, xBtnSize, xBtnSize, xBtnColor)
	rl.DrawText("X", xBtnX+4, xBtnY+2, 12, rl.White)

	shouldRemove := xHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton)
	y += headerH + 2

	// Draw component-specific properties
	y = e.drawComponentProperties(panelX, y, c, index)

	return y, shouldRemove
}

// drawComponentProperties draws editable properties for each component type.
func (e *Editor) drawComponentProperties(panelX, y int32, c engine.Component, compIdx int) int32 {
	propColor := rl.NewColor(180, 180, 180, 255)
	indent := panelX + 14
	labelW := int32(70)
	fieldW := int32(70)
	fieldH := int32(18)

	switch comp := c.(type) {
	case *components.ModelRenderer:
		if comp.FilePath != "" {
			rl.DrawText(fmt.Sprintf("Model: %s", filepath.Base(comp.FilePath)), indent, y, 12, propColor)
			y += 16
		} else {
			rl.DrawText(fmt.Sprintf("Mesh: %s", comp.MeshType), indent, y, 12, propColor)
			y += 16
		}
		// Color dropdown would go here - for now just display
		rl.DrawText(fmt.Sprintf("Color: %s", colorName(comp.Color)), indent, y, 12, propColor)
		y += 18

	case *components.BoxCollider:
		// Size
		rl.DrawText("Size", indent, y+2, 12, propColor)
		id := fmt.Sprintf("box%d.size", compIdx)
		comp.Size.X = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".x", comp.Size.X)
		comp.Size.Y = e.drawFloatField(indent+labelW+fieldW+2, y, fieldW, fieldH, id+".y", comp.Size.Y)
		comp.Size.Z = e.drawFloatField(indent+labelW+2*(fieldW+2), y, fieldW, fieldH, id+".z", comp.Size.Z)
		y += fieldH + 4

		// Offset
		rl.DrawText("Offset", indent, y+2, 12, propColor)
		id = fmt.Sprintf("box%d.off", compIdx)
		comp.Offset.X = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".x", comp.Offset.X)
		comp.Offset.Y = e.drawFloatField(indent+labelW+fieldW+2, y, fieldW, fieldH, id+".y", comp.Offset.Y)
		comp.Offset.Z = e.drawFloatField(indent+labelW+2*(fieldW+2), y, fieldW, fieldH, id+".z", comp.Offset.Z)
		y += fieldH + 6

	case *components.SphereCollider:
		rl.DrawText("Radius", indent, y+2, 12, propColor)
		id := fmt.Sprintf("sphere%d.rad", compIdx)
		comp.Radius = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id, comp.Radius)
		y += fieldH + 6

	case *components.Rigidbody:
		// Mass
		rl.DrawText("Mass", indent, y+2, 12, propColor)
		comp.Mass = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("rb%d.mass", compIdx), comp.Mass)
		y += fieldH + 2

		// Bounciness
		rl.DrawText("Bounce", indent, y+2, 12, propColor)
		comp.Bounciness = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("rb%d.bounce", compIdx), comp.Bounciness)
		y += fieldH + 2

		// Friction
		rl.DrawText("Friction", indent, y+2, 12, propColor)
		comp.Friction = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("rb%d.friction", compIdx), comp.Friction)
		y += fieldH + 4

		// Checkboxes for booleans
		gravityBounds := rl.Rectangle{X: float32(indent), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		comp.UseGravity = gui.CheckBox(gravityBounds, "Gravity", comp.UseGravity)

		kinematicBounds := rl.Rectangle{X: float32(indent + 100), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		comp.IsKinematic = gui.CheckBox(kinematicBounds, "Kinematic", comp.IsKinematic)
		y += fieldH + 6

	case *components.DirectionalLight:
		// Direction
		rl.DrawText("Dir", indent, y+2, 12, propColor)
		id := fmt.Sprintf("light%d.dir", compIdx)
		comp.Direction.X = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".x", comp.Direction.X)
		comp.Direction.Y = e.drawFloatField(indent+labelW+fieldW+2, y, fieldW, fieldH, id+".y", comp.Direction.Y)
		comp.Direction.Z = e.drawFloatField(indent+labelW+2*(fieldW+2), y, fieldW, fieldH, id+".z", comp.Direction.Z)
		y += fieldH + 4

		// Intensity slider
		rl.DrawText("Intensity", indent, y+2, 12, propColor)
		sliderBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldW * 2), Height: float32(fieldH)}
		comp.Intensity = gui.Slider(sliderBounds, "", fmt.Sprintf("%.1f", comp.Intensity), comp.Intensity, 0, 2)
		y += fieldH + 6

	default:
		// For scripts and unknown components, try to get script name
		if name, props, ok := engine.SerializeScript(c); ok {
			rl.DrawText(fmt.Sprintf("Script: %s", name), indent, y, 12, propColor)
			y += 14
			for k, v := range props {
				rl.DrawText(fmt.Sprintf("  %s: %v", k, v), indent, y, 11, rl.Gray)
				y += 12
			}
			y += 4
		} else {
			y += 16
		}
	}

	return y
}

// drawAddComponentMenu draws the dropdown menu for adding components.
// justOpened prevents the menu from closing on the same frame it was opened.
func (e *Editor) drawAddComponentMenu(x, y, w int32, justOpened bool) {
	itemH := int32(22)
	menuH := int32(len(editorComponentTypes)) * itemH

	// Draw menu background
	rl.DrawRectangle(x, y, w, menuH, rl.NewColor(35, 35, 40, 250))
	rl.DrawRectangleLines(x, y, w, menuH, rl.NewColor(80, 80, 80, 255))

	mousePos := rl.GetMousePosition()
	mouseInMenu := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
		mousePos.Y >= float32(y) && mousePos.Y <= float32(y+menuH)

	for i, compType := range editorComponentTypes {
		itemY := y + int32(i)*itemH

		hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
			mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH)

		if hovered {
			rl.DrawRectangle(x, itemY, w, itemH, rl.NewColor(60, 80, 60, 200))
		}

		rl.DrawText(compType.Name, x+10, itemY+4, 14, rl.LightGray)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.addComponent(compType.Name)
			e.showAddComponentMenu = false
		}
	}

	// Close menu if clicking outside (but not on the frame we just opened it)
	if !justOpened && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !mouseInMenu {
		e.showAddComponentMenu = false
	}
}

// addComponent adds a new component of the given type to the selected object.
func (e *Editor) addComponent(typeName string) {
	if e.Selected == nil {
		return
	}

	for _, compType := range editorComponentTypes {
		if compType.Name == typeName {
			newComp := compType.Factory(e.world, e.Selected)
			e.Selected.AddComponent(newComp)

			// Re-register with physics world to update categorization
			e.updatePhysicsRegistration(e.Selected)

			e.saveMsg = fmt.Sprintf("Added %s", typeName)
			e.saveMsgTime = rl.GetTime()
			return
		}
	}
}

// removeComponentAtIndex removes the component at the given index from the selected object.
func (e *Editor) removeComponentAtIndex(index int) {
	if e.Selected == nil {
		return
	}

	comps := e.Selected.Components()
	if index < 0 || index >= len(comps) {
		return
	}

	comp := comps[index]
	typeName := reflect.TypeOf(comp).Elem().Name()

	// Cleanup for specific component types
	switch c := comp.(type) {
	case *components.ModelRenderer:
		c.Unload()
	case *components.DirectionalLight:
		// Clear light reference if this is the active light
		if e.world.Light == e.Selected {
			e.world.Light = nil
		}
	}

	e.Selected.RemoveComponentByIndex(index)

	// Update physics world registration
	e.updatePhysicsRegistration(e.Selected)

	e.saveMsg = fmt.Sprintf("Removed %s", typeName)
	e.saveMsgTime = rl.GetTime()
}

// updatePhysicsRegistration removes and re-adds an object to the physics world
// to update its categorization (static/kinematic/dynamic).
func (e *Editor) updatePhysicsRegistration(g *engine.GameObject) {
	e.world.PhysicsWorld.RemoveObject(g)
	e.world.PhysicsWorld.AddObject(g)
}

// colorName returns a human-readable name for common colors.
func colorName(c rl.Color) string {
	switch c {
	case rl.Red:
		return "Red"
	case rl.Blue:
		return "Blue"
	case rl.Green:
		return "Green"
	case rl.Purple:
		return "Purple"
	case rl.Orange:
		return "Orange"
	case rl.Yellow:
		return "Yellow"
	case rl.Pink:
		return "Pink"
	case rl.SkyBlue:
		return "SkyBlue"
	case rl.Lime:
		return "Lime"
	case rl.Magenta:
		return "Magenta"
	case rl.White:
		return "White"
	case rl.LightGray:
		return "LightGray"
	case rl.Gray:
		return "Gray"
	case rl.DarkGray:
		return "DarkGray"
	case rl.Black:
		return "Black"
	case rl.Brown:
		return "Brown"
	case rl.Beige:
		return "Beige"
	case rl.Maroon:
		return "Maroon"
	case rl.Gold:
		return "Gold"
	default:
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
}

// deleteSelectedObject removes the currently selected object from the scene.
func (e *Editor) deleteSelectedObject() {
	if e.Selected == nil {
		return
	}

	// Don't allow deleting the player
	if e.Selected.Name == "Player" {
		e.saveMsg = "Cannot delete Player"
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Push undo state before deleting (keeps the object reference alive)
	e.pushDeleteUndo(e.Selected)

	name := e.Selected.Name

	// Remove from scene and physics, but keep model loaded (for undo)
	e.world.EditorDestroy(e.Selected)

	e.Selected = nil

	e.saveMsg = fmt.Sprintf("Deleted %s", name)
	e.saveMsgTime = rl.GetTime()
}

// createNewGameObject creates a new empty GameObject and adds it to the scene.
func (e *Editor) createNewGameObject() {
	// Generate unique name
	baseName := "GameObject"
	name := baseName
	count := 1
	for e.world.Scene.FindByName(name) != nil {
		name = fmt.Sprintf("%s (%d)", baseName, count)
		count++
	}

	obj := engine.NewGameObject(name)

	// Position in front of camera
	forward, _ := e.getDirections()
	obj.Transform.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(forward, 5))

	e.world.Scene.AddGameObject(obj)
	e.world.PhysicsWorld.AddObject(obj)

	e.Selected = obj
	e.saveMsg = fmt.Sprintf("Created %s", name)
	e.saveMsgTime = rl.GetTime()
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
	// Asset browser: bottom 120px (when visible)
	if e.showAssetBrowser && m.Y >= screenH-120 && m.X > 200 && m.X < screenW-300 {
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

// drawAssetBrowser draws the asset browser panel at the bottom of the screen
func (e *Editor) drawAssetBrowser() {
	panelH := int32(120)
	panelY := int32(rl.GetScreenHeight()) - panelH
	panelX := int32(200) // Start after hierarchy
	panelW := int32(rl.GetScreenWidth()) - 200 - 300 // Between hierarchy and inspector

	// Background
	rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.NewColor(25, 25, 30, 240))
	rl.DrawLine(panelX, panelY, panelX+panelW, panelY, rl.NewColor(60, 60, 60, 255))

	// Header
	rl.DrawText("Models", panelX+10, panelY+6, 14, rl.Gray)

	// Refresh button
	refreshBtnX := panelX + panelW - 70
	refreshBtnY := panelY + 4
	refreshBtnW := int32(60)
	refreshBtnH := int32(18)

	mousePos := rl.GetMousePosition()
	refreshHovered := mousePos.X >= float32(refreshBtnX) && mousePos.X <= float32(refreshBtnX+refreshBtnW) &&
		mousePos.Y >= float32(refreshBtnY) && mousePos.Y <= float32(refreshBtnY+refreshBtnH)

	refreshColor := rl.NewColor(50, 50, 60, 200)
	if refreshHovered {
		refreshColor = rl.NewColor(70, 70, 80, 220)
	}
	rl.DrawRectangle(refreshBtnX, refreshBtnY, refreshBtnW, refreshBtnH, refreshColor)
	rl.DrawText("Refresh", refreshBtnX+6, refreshBtnY+2, 12, rl.LightGray)

	if refreshHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.scanAssetModels()
	}

	// Asset grid
	itemW := int32(80)
	itemH := int32(70)
	startX := panelX + 10
	startY := panelY + 28
	cols := (panelW - 20) / (itemW + 8)

	// Scroll handling
	mouseInPanel := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH)

	if mouseInPanel && !rl.IsMouseButtonDown(rl.MouseRightButton) {
		scroll := rl.GetMouseWheelMove()
		e.assetBrowserScroll -= int32(scroll * 30)
		if e.assetBrowserScroll < 0 {
			e.assetBrowserScroll = 0
		}
	}

	// Clip content
	rl.BeginScissorMode(panelX, panelY+24, panelW, panelH-24)

	for i, asset := range e.assetFiles {
		col := int32(i) % cols
		row := int32(i) / cols

		x := startX + col*(itemW+8)
		y := startY + row*(itemH+8) - e.assetBrowserScroll

		// Skip if off screen
		if y+itemH < panelY+24 || y > panelY+panelH {
			continue
		}

		// Item background
		itemHovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+itemW) &&
			mousePos.Y >= float32(y) && mousePos.Y <= float32(y+itemH)

		bgColor := rl.NewColor(40, 40, 45, 200)
		if itemHovered {
			bgColor = rl.NewColor(60, 70, 60, 220)
		}
		rl.DrawRectangle(x, y, itemW, itemH, bgColor)

		// Icon placeholder (cube icon)
		iconSize := int32(32)
		iconX := x + (itemW-iconSize)/2
		iconY := y + 6
		rl.DrawRectangle(iconX, iconY, iconSize, iconSize, rl.NewColor(80, 80, 90, 200))
		rl.DrawText("3D", iconX+8, iconY+8, 14, rl.LightGray)

		// Name (truncated)
		name := asset.Name
		if len(name) > 10 {
			name = name[:9] + "…"
		}
		textW := rl.MeasureText(name, 11)
		rl.DrawText(name, x+(itemW-textW)/2, y+itemH-18, 11, rl.LightGray)

		// Click to spawn
		if itemHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.spawnModelFromAsset(asset)
		}
	}

	rl.EndScissorMode()

	// Clamp scroll
	rows := (int32(len(e.assetFiles)) + cols - 1) / cols
	maxScroll := rows*(itemH+8) - (panelH - 28)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.assetBrowserScroll > maxScroll {
		e.assetBrowserScroll = maxScroll
	}

	// Empty state
	if len(e.assetFiles) == 0 {
		rl.DrawText("No models found. Drop .gltf files to import.", panelX+20, panelY+50, 14, rl.Gray)
	}
}

// spawnModelFromAsset creates a new GameObject with the given model
func (e *Editor) spawnModelFromAsset(asset AssetEntry) {
	obj := engine.NewGameObject(asset.Name)

	// Position in front of camera
	forward, _ := e.getDirections()
	obj.Transform.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(forward, 5))
	obj.Transform.Scale = rl.NewVector3(1, 1, 1)

	// Add ModelRenderer
	modelRenderer := components.NewModelRendererFromFile(asset.Path, rl.White)
	obj.AddComponent(modelRenderer)

	// Add to scene
	e.world.Scene.AddGameObject(obj)
	e.world.PhysicsWorld.AddObject(obj)
	e.Selected = obj

	e.saveMsg = fmt.Sprintf("Spawned %s", asset.Name)
	e.saveMsgTime = rl.GetTime()
}

// scanAssetModels scans the assets/models folder and populates the asset list
func (e *Editor) scanAssetModels() {
	e.assetFiles = nil
	modelsDir := "assets/models"

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Look for .gltf or .glb inside the folder
			subPath := filepath.Join(modelsDir, entry.Name())
			subEntries, err := os.ReadDir(subPath)
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				if sub.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(sub.Name()))
				if ext == ".gltf" || ext == ".glb" {
					e.assetFiles = append(e.assetFiles, AssetEntry{
						Name: entry.Name(),
						Path: filepath.Join(subPath, sub.Name()),
					})
					break // Only add one model per folder
				}
			}
		} else {
			// Check for loose model files
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".gltf" || ext == ".glb" {
				name := strings.TrimSuffix(entry.Name(), ext)
				e.assetFiles = append(e.assetFiles, AssetEntry{
					Name: name,
					Path: filepath.Join(modelsDir, entry.Name()),
				})
			}
		}
	}
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
