//go:build !game

package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"test3d/internal/assets"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
	gui "github.com/gen2brain/raylib-go/raygui"
)

type EditorCamera struct {
	Position  rl.Vector3
	Yaw       float32
	Pitch     float32
	MoveSpeed float32
}

type Editor struct {
	Active   bool
	Paused   bool // True if entered via pause (preserves scene state)
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
	showAssetBrowser     bool
	assetBrowserScroll   int32
	assetFiles           []AssetEntry
	currentAssetPath     string           // Current directory being viewed
	selectedMaterialPath string           // Selected material for editing
	selectedMaterial     *assets.Material // Loaded material being edited
	lastClickTime        float64          // For double-click detection
	lastClickedAsset     string           // Path of last clicked asset

	// Script hot-reload
	scriptModTimes  map[string]int64 // path -> mod time (unix nano)
	scriptsChanged  bool
	lastScriptCheck float64

	// Drag-and-drop state
	draggingAsset       bool         // True if dragging an asset from the browser
	draggedAsset        *AssetEntry  // The asset being dragged
	draggingHierarchy   bool         // True if dragging an object in hierarchy
	draggedObject       *engine.GameObject // The object being dragged for reparenting
	hierarchyDropTarget *engine.GameObject // Target for hierarchy drop (parent candidate)
	hierarchyDropIndex  int          // Index where to drop (-1 = as child, >= 0 = at position)

	// Name editing state
	editingName    bool   // True if editing the object name
	nameEditBuffer string // Current text in name edit field
}

// AssetEntry represents a file or folder in the asset browser
type AssetEntry struct {
	Name     string // Display name
	Path     string // Full path to the file/folder
	IsFolder bool   // True if this is a folder
	Type     string // "folder", "model", "material", "texture", "scene", etc.
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
	e.Paused = false
	rl.EnableCursor()

	// Reload scene from disk to undo all play mode changes
	e.world.ResetScene()
	e.Selected = nil

	// Initialize script hot-reload watcher
	e.scanScriptModTimes()

	e.camera.Position = currentCam.Position

	dir := rl.Vector3Subtract(currentCam.Target, currentCam.Position)
	dir = rl.Vector3Normalize(dir)
	e.camera.Pitch = float32(math.Asin(float64(dir.Y))) * rl.Rad2deg
	e.camera.Yaw = float32(math.Atan2(float64(dir.Z), float64(dir.X))) * rl.Rad2deg

	// Initialize raygui dark style
	initRayguiStyle()
}

// Pause enters editor mode without resetting the scene (preserves physics state)
func (e *Editor) Pause(currentCam rl.Camera3D) {
	e.Active = true
	e.Paused = true
	rl.EnableCursor()

	// Don't reset scene - preserve current state
	e.Selected = nil

	// Initialize script hot-reload watcher
	e.scanScriptModTimes()

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
	// Save scene before entering play mode
	if err := e.world.SaveScene(world.ScenePath); err != nil {
		log.Printf("Warning: Failed to save scene before play mode: %v", err)
	}

	e.Active = false
	e.Paused = false
	e.Selected = nil
	e.dragging = false
	e.hoveredAxis = -1
	rl.DisableCursor()
}

func (e *Editor) Update(deltaTime float32) {
	// Check for script file changes
	e.checkScriptChanges()

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

	// Ctrl+B: build game
	if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyLeftSuper)) && rl.IsKeyPressed(rl.KeyB) {
		e.buildGame()
	}

	// Ctrl+R: rebuild and relaunch (for script hot-reload)
	if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyLeftSuper)) && rl.IsKeyPressed(rl.KeyR) {
		e.rebuildAndRelaunch()
	}

	// Tab: toggle asset browser
	if rl.IsKeyPressed(rl.KeyTab) {
		e.showAssetBrowser = !e.showAssetBrowser
		if e.showAssetBrowser {
			if e.currentAssetPath == "" {
				e.currentAssetPath = "assets"
			}
			e.scanAssets()
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

		hit, ok := e.world.EditorRaycast(ray.Position, ray.Direction, 1000)
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

// DrawUI draws the editor overlay: top bar, hierarchy panel (left), inspector panel (right).
func (e *Editor) DrawUI() {
	// Top bar
	rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), 32, rl.NewColor(20, 20, 20, 220))
	if e.Paused {
		rl.DrawText("PAUSED", 10, 6, 20, rl.Orange)
	} else {
		rl.DrawText("EDITOR", 10, 6, 20, rl.Yellow)
	}
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
	helpText := "| Ctrl+S: Save | Ctrl+B: Build | Ctrl+Z: Undo"
	if e.Paused {
		helpText = "| P: Resume | Ctrl+S: Save"
	}
	rl.DrawText(helpText, 350, 8, 16, rl.LightGray)
	rl.DrawText(fmt.Sprintf("Speed: %.0f", e.camera.MoveSpeed), int32(rl.GetScreenWidth())-100, 8, 16, rl.LightGray)

	// Scripts changed banner
	if e.scriptsChanged {
		bannerText := "Scripts changed - Press Ctrl+R to rebuild"
		textWidth := rl.MeasureText(bannerText, 16)
		bannerX := (int32(rl.GetScreenWidth()) - textWidth) / 2
		rl.DrawRectangle(bannerX-10, 36, textWidth+20, 24, rl.NewColor(80, 60, 0, 230))
		rl.DrawText(bannerText, bannerX, 40, 16, rl.Yellow)
	}

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
			if e.currentAssetPath == "" {
				e.currentAssetPath = "assets"
			}
			e.scanAssets()
		}
	}

	if e.showAssetBrowser {
		e.drawAssetBrowser()
	}

	// Draw material drag indicator
	if e.draggingAsset && e.draggedAsset != nil {
		mousePos := rl.GetMousePosition()
		rl.DrawRectangle(int32(mousePos.X)+12, int32(mousePos.Y)-10, 80, 20, rl.NewColor(40, 60, 80, 220))
		name := e.draggedAsset.Name
		if len(name) > 12 {
			name = name[:11] + "…"
		}
		rl.DrawText(name, int32(mousePos.X)+16, int32(mousePos.Y)-6, 12, rl.SkyBlue)

		// Handle drop on mouse release
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			e.handleMaterialDrop()
		}
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

	// Reset drop target each frame
	e.hierarchyDropTarget = nil

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
		isDragTarget := e.draggingHierarchy && hovered && e.draggedObject != g && !e.isDescendantOf(g, e.draggedObject)

		if isDragTarget {
			// Highlight as drop target
			rl.DrawRectangle(panelX, itemY, panelW, itemH, rl.NewColor(50, 80, 120, 200))
			e.hierarchyDropTarget = g
		} else if selected {
			rl.DrawRectangle(panelX, itemY, panelW, itemH, rl.NewColor(80, 80, 20, 180))
		} else if hovered {
			rl.DrawRectangle(panelX, itemY, panelW, itemH, rl.NewColor(50, 50, 50, 150))
		}

		// Start drag on mouse down (if not already dragging)
		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !clickedNewButton && !e.draggingHierarchy {
			e.Selected = g
			e.draggingHierarchy = true
			e.draggedObject = g
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
		if e.draggingHierarchy && e.draggedObject == g {
			textColor = rl.SkyBlue // Indicate dragged item
		}
		rl.DrawText(g.Name, panelX+indent, itemY+4, 14, textColor)
	}

	rl.EndScissorMode()

	// "Unparent" drop zone at bottom of hierarchy (drawn outside scissor mode)
	if e.draggingHierarchy && e.draggedObject != nil && e.draggedObject.Parent != nil {
		unparentY := panelY + panelH - itemH - 4
		unparentHovered := mouseInPanel && mousePos.Y >= float32(unparentY) && mousePos.Y <= float32(panelY+panelH)

		bgColor := rl.NewColor(60, 40, 40, 200)
		if unparentHovered {
			bgColor = rl.NewColor(100, 60, 60, 220)
			e.hierarchyDropTarget = nil // nil means unparent
			e.hierarchyDropIndex = -2   // special value for unparent
		}
		rl.DrawRectangle(panelX, unparentY, panelW, itemH, bgColor)
		rl.DrawText("-- Unparent --", panelX+50, unparentY+4, 14, rl.LightGray)
	}

	// Handle drop on mouse release
	if e.draggingHierarchy && rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		if e.draggedObject != nil {
			if e.hierarchyDropIndex == -2 {
				// Unparent
				e.reparentObject(e.draggedObject, nil)
			} else if e.hierarchyDropTarget != nil && e.hierarchyDropTarget != e.draggedObject {
				// Reparent to drop target
				e.reparentObject(e.draggedObject, e.hierarchyDropTarget)
			}
		}
		e.draggingHierarchy = false
		e.draggedObject = nil
		e.hierarchyDropTarget = nil
		e.hierarchyDropIndex = 0
	}

	// Draw drag indicator if dragging
	if e.draggingHierarchy && e.draggedObject != nil {
		rl.DrawText(e.draggedObject.Name, int32(mousePos.X)+10, int32(mousePos.Y)-8, 12, rl.SkyBlue)
	}
}

// isDescendantOf checks if 'potential' is a descendant of 'ancestor'
func (e *Editor) isDescendantOf(potential, ancestor *engine.GameObject) bool {
	p := potential.Parent
	for p != nil {
		if p == ancestor {
			return true
		}
		p = p.Parent
	}
	return false
}

// reparentObject changes the parent of an object, preserving world position
func (e *Editor) reparentObject(child, newParent *engine.GameObject) {
	if child == nil || child == newParent {
		return
	}

	// Don't allow parenting to a descendant
	if newParent != nil && e.isDescendantOf(newParent, child) {
		return
	}

	// Store world position before reparenting
	worldPos := child.WorldPosition()

	// Remove from old parent
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}

	// Add to new parent
	if newParent != nil {
		newParent.AddChild(child)
		// Convert world position to new local position
		parentWorldPos := newParent.WorldPosition()
		child.Transform.Position = rl.Vector3Subtract(worldPos, parentWorldPos)
	} else {
		child.Parent = nil
		child.Transform.Position = worldPos
	}

	e.saveMsg = fmt.Sprintf("Reparented %s", child.Name)
	e.saveMsgTime = rl.GetTime()
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

	// Name (editable)
	nameFieldW := panelW - 20
	nameFieldH := int32(24)
	nameFieldX := panelX + 10
	nameFieldY := y

	nameHovered := mousePos.X >= float32(nameFieldX) && mousePos.X <= float32(nameFieldX+nameFieldW) &&
		mousePos.Y >= float32(nameFieldY) && mousePos.Y <= float32(nameFieldY+nameFieldH)

	// Background for name field
	nameBgColor := rl.NewColor(40, 40, 45, 255)
	if e.editingName {
		nameBgColor = rl.NewColor(50, 50, 60, 255)
	} else if nameHovered {
		nameBgColor = rl.NewColor(45, 45, 50, 255)
	}
	rl.DrawRectangle(nameFieldX, nameFieldY, nameFieldW, nameFieldH, nameBgColor)
	rl.DrawRectangleLines(nameFieldX, nameFieldY, nameFieldW, nameFieldH, rl.NewColor(70, 70, 80, 255))

	if e.editingName {
		// Draw editing text with cursor
		rl.DrawText(e.nameEditBuffer+"_", nameFieldX+6, nameFieldY+4, 16, rl.White)

		// Handle typing
		for {
			key := rl.GetCharPressed()
			if key == 0 {
				break
			}
			e.nameEditBuffer += string(rune(key))
		}

		// Backspace
		if rl.IsKeyPressed(rl.KeyBackspace) && len(e.nameEditBuffer) > 0 {
			e.nameEditBuffer = e.nameEditBuffer[:len(e.nameEditBuffer)-1]
		}

		// Enter to confirm
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			if e.nameEditBuffer != "" {
				e.Selected.Name = e.nameEditBuffer
			}
			e.editingName = false
			e.nameEditBuffer = ""
		}

		// Escape to cancel
		if rl.IsKeyPressed(rl.KeyEscape) {
			e.editingName = false
			e.nameEditBuffer = ""
		}

		// Click outside to confirm
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !nameHovered {
			if e.nameEditBuffer != "" {
				e.Selected.Name = e.nameEditBuffer
			}
			e.editingName = false
			e.nameEditBuffer = ""
		}
	} else {
		// Display name
		rl.DrawText(e.Selected.Name, nameFieldX+6, nameFieldY+4, 16, rl.Yellow)

		// Click to edit
		if nameHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.editingName = true
			e.nameEditBuffer = e.Selected.Name
		}
	}
	y += nameFieldH + 4

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

// drawTextureField draws an editable text field for texture paths
func (e *Editor) drawTextureField(x, y, w, h int32, id string, value string) string {
	mousePos := rl.GetMousePosition()
	hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
		mousePos.Y >= float32(y) && mousePos.Y <= float32(y+h)

	editMode := e.activeInputID == id

	// Background color
	bgColor := rl.NewColor(45, 45, 50, 255)
	if editMode {
		bgColor = rl.NewColor(60, 60, 70, 255)
	} else if hovered {
		bgColor = rl.NewColor(55, 55, 60, 255)
	}
	rl.DrawRectangle(x, y, w, h, bgColor)
	rl.DrawRectangleLines(x, y, w, h, rl.NewColor(80, 80, 90, 255))

	// Click to edit
	if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !editMode {
		e.activeInputID = id
		e.inputTextValue = value
	}

	// Text display/editing
	if editMode {
		// Draw text input with cursor
		displayText := e.inputTextValue
		if len(displayText) > 14 {
			displayText = "…" + displayText[len(displayText)-13:]
		}
		rl.DrawText(displayText+"_", x+4, y+3, 11, rl.White)

		// Handle typing
		for {
			key := rl.GetCharPressed()
			if key == 0 {
				break
			}
			e.inputTextValue += string(rune(key))
		}

		// Backspace
		if rl.IsKeyPressed(rl.KeyBackspace) && len(e.inputTextValue) > 0 {
			e.inputTextValue = e.inputTextValue[:len(e.inputTextValue)-1]
		}

		// Enter or click outside to confirm
		clickedOutside := rl.IsMouseButtonPressed(rl.MouseLeftButton) && !hovered
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) || clickedOutside || rl.IsKeyPressed(rl.KeyTab) {
			value = e.inputTextValue
			e.activeInputID = ""
			e.inputTextValue = ""
		}

		// Escape to cancel
		if rl.IsKeyPressed(rl.KeyEscape) {
			e.activeInputID = ""
			e.inputTextValue = ""
		}
	} else {
		// Display current value (truncated)
		displayText := value
		if displayText == "" {
			displayText = "(none)"
		} else {
			displayText = filepath.Base(displayText)
		}
		if len(displayText) > 15 {
			displayText = displayText[:14] + "…"
		}
		textColor := rl.LightGray
		if value == "" {
			textColor = rl.Gray
		}
		rl.DrawText(displayText, x+4, y+3, 11, textColor)
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

		// Material asset reference
		if comp.MaterialPath != "" {
			rl.DrawText(fmt.Sprintf("Material: %s", filepath.Base(comp.MaterialPath)), indent, y, 12, rl.SkyBlue)
			y += 16
			// Editable material properties (saves to material file)
			if comp.Material != nil {
				id := fmt.Sprintf("mat%d", compIdx)
				oldMet := comp.Material.Metallic
				oldRough := comp.Material.Roughness
				oldEmit := comp.Material.Emissive

				rl.DrawText("Metallic", indent, y+2, 12, propColor)
				comp.Material.Metallic = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".met", comp.Material.Metallic)
				y += fieldH + 2

				rl.DrawText("Roughness", indent, y+2, 12, propColor)
				comp.Material.Roughness = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".rough", comp.Material.Roughness)
				y += fieldH + 2

				rl.DrawText("Emissive", indent, y+2, 12, propColor)
				comp.Material.Emissive = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".emit", comp.Material.Emissive)
				y += fieldH + 4

				// Save material if any value changed
				if comp.Material.Metallic != oldMet || comp.Material.Roughness != oldRough || comp.Material.Emissive != oldEmit {
					assets.SaveMaterial(comp.MaterialPath, comp.Material)
				}
			}
		} else if comp.FilePath != "" {
			// GLTF model using built-in materials
			rl.DrawText("Material: Built-in", indent, y, 12, rl.Gray)
			y += 16
		} else {
			// Generated mesh - inline material properties (editable)
			// Color dropdown would go here - for now just display
			rl.DrawText(fmt.Sprintf("Color: %s", colorName(comp.Color)), indent, y, 12, propColor)
			y += 18

			id := fmt.Sprintf("mr%d", compIdx)
			rl.DrawText("Metallic", indent, y+2, 12, propColor)
			comp.Metallic = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".met", comp.Metallic)
			y += fieldH + 2

			rl.DrawText("Roughness", indent, y+2, 12, propColor)
			comp.Roughness = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".rough", comp.Roughness)
			y += fieldH + 2

			rl.DrawText("Emissive", indent, y+2, 12, propColor)
			comp.Emissive = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".emit", comp.Emissive)
			y += fieldH + 4
		}

		// Flip Normals button for GLTF models
		if comp.FilePath != "" {
			btnW := int32(90)
			btnH := int32(18)
			btnX := indent
			btnY := y
			mousePos := rl.GetMousePosition()
			btnHovered := mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
				mousePos.Y >= float32(btnY) && mousePos.Y <= float32(btnY+btnH)
			btnColor := rl.NewColor(60, 60, 70, 200)
			if btnHovered {
				btnColor = rl.NewColor(80, 80, 90, 220)
			}
			rl.DrawRectangle(btnX, btnY, btnW, btnH, btnColor)
			rl.DrawText("Flip Normals", btnX+6, btnY+3, 11, rl.LightGray)

			if btnHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				cmd := exec.Command("./mirgo-utils", "flipnormals", comp.FilePath)
				if err := cmd.Run(); err != nil {
					fmt.Printf("Failed to flip normals: %v\n", err)
				} else {
					fmt.Printf("Flipped normals for %s\n", comp.FilePath)
				}
			}
			y += btnH + 4
		}

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

	case *components.PointLight:
		id := fmt.Sprintf("pointlight%d", compIdx)

		// Color picker (simplified - show RGB sliders)
		rl.DrawText("Color", indent, y+2, 12, propColor)
		colorPreview := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		rl.DrawRectangleRec(colorPreview, comp.Color)
		rl.DrawRectangleLinesEx(colorPreview, 1, rl.Gray)
		// R/G/B fields
		comp.Color.R = uint8(e.drawFloatField(indent+labelW+fieldH+4, y, fieldW-10, fieldH, id+".r", float32(comp.Color.R)))
		comp.Color.G = uint8(e.drawFloatField(indent+labelW+fieldH+4+fieldW-8, y, fieldW-10, fieldH, id+".g", float32(comp.Color.G)))
		comp.Color.B = uint8(e.drawFloatField(indent+labelW+fieldH+4+2*(fieldW-8), y, fieldW-10, fieldH, id+".b", float32(comp.Color.B)))
		y += fieldH + 4

		// Intensity slider
		rl.DrawText("Intensity", indent, y+2, 12, propColor)
		intensityBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldW * 2), Height: float32(fieldH)}
		comp.Intensity = gui.Slider(intensityBounds, "", fmt.Sprintf("%.1f", comp.Intensity), comp.Intensity, 0, 5)
		y += fieldH + 4

		// Radius slider
		rl.DrawText("Radius", indent, y+2, 12, propColor)
		radiusBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldW * 2), Height: float32(fieldH)}
		comp.Radius = gui.Slider(radiusBounds, "", fmt.Sprintf("%.1f", comp.Radius), comp.Radius, 1, 50)
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
	// Asset browser: bottom 150px (when visible)
	if e.showAssetBrowser && m.Y >= screenH-150 && m.X > 200 && m.X < screenW-300 {
		return true
	}
	return false
}

// drawAssetBrowser draws the asset browser panel at the bottom of the screen
func (e *Editor) drawAssetBrowser() {
	panelH := int32(150)
	panelY := int32(rl.GetScreenHeight()) - panelH
	panelX := int32(200) // Start after hierarchy
	panelW := int32(rl.GetScreenWidth()) - 200 - 300 // Between hierarchy and inspector

	// Reserve space for material editor on the right when a material is selected
	contentW := panelW
	if e.selectedMaterial != nil {
		contentW = panelW - 180
	}

	// Background
	rl.DrawRectangle(panelX, panelY, panelW, panelH, rl.NewColor(25, 25, 30, 240))
	rl.DrawLine(panelX, panelY, panelX+panelW, panelY, rl.NewColor(60, 60, 60, 255))

	mousePos := rl.GetMousePosition()

	// Header with back button and path
	headerY := panelY + 4

	// Back button (only show if not at root)
	backBtnX := panelX + 8
	backBtnW := int32(24)
	backBtnH := int32(18)
	canGoBack := e.currentAssetPath != "assets" && e.currentAssetPath != ""

	if canGoBack {
		backHovered := mousePos.X >= float32(backBtnX) && mousePos.X <= float32(backBtnX+backBtnW) &&
			mousePos.Y >= float32(headerY) && mousePos.Y <= float32(headerY+backBtnH)

		backColor := rl.NewColor(50, 50, 60, 200)
		if backHovered {
			backColor = rl.NewColor(70, 70, 80, 220)
		}
		rl.DrawRectangle(backBtnX, headerY, backBtnW, backBtnH, backColor)
		rl.DrawText("<", backBtnX+8, headerY+2, 14, rl.LightGray)

		if backHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			// Go up one directory
			e.currentAssetPath = filepath.Dir(e.currentAssetPath)
			e.assetBrowserScroll = 0
			e.selectedMaterial = nil
			e.selectedMaterialPath = ""
			e.scanAssets()
		}
	}

	// Current path
	pathX := backBtnX + backBtnW + 8
	if !canGoBack {
		pathX = panelX + 10
	}
	rl.DrawText(e.currentAssetPath+"/", pathX, headerY+2, 14, rl.Gray)

	// Refresh button
	refreshBtnX := panelX + contentW - 70
	refreshBtnY := headerY
	refreshBtnW := int32(60)
	refreshBtnH := int32(18)

	refreshHovered := mousePos.X >= float32(refreshBtnX) && mousePos.X <= float32(refreshBtnX+refreshBtnW) &&
		mousePos.Y >= float32(refreshBtnY) && mousePos.Y <= float32(refreshBtnY+refreshBtnH)

	refreshColor := rl.NewColor(50, 50, 60, 200)
	if refreshHovered {
		refreshColor = rl.NewColor(70, 70, 80, 220)
	}
	rl.DrawRectangle(refreshBtnX, refreshBtnY, refreshBtnW, refreshBtnH, refreshColor)
	rl.DrawText("Refresh", refreshBtnX+6, refreshBtnY+2, 12, rl.LightGray)

	if refreshHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.scanAssets()
	}

	// Asset grid
	itemW := int32(70)
	itemH := int32(70)
	startX := panelX + 10
	startY := panelY + 28
	cols := (contentW - 20) / (itemW + 6)
	if cols < 1 {
		cols = 1
	}

	// Scroll handling
	mouseInPanel := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+contentW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH)

	if mouseInPanel && !rl.IsMouseButtonDown(rl.MouseRightButton) {
		scroll := rl.GetMouseWheelMove()
		e.assetBrowserScroll -= int32(scroll * 30)
		if e.assetBrowserScroll < 0 {
			e.assetBrowserScroll = 0
		}
	}

	// Clip content
	rl.BeginScissorMode(panelX, panelY+24, contentW, panelH-24)

	for i, asset := range e.assetFiles {
		col := int32(i) % cols
		row := int32(i) / cols

		x := startX + col*(itemW+6)
		y := startY + row*(itemH+6) - e.assetBrowserScroll

		// Skip if off screen
		if y+itemH < panelY+24 || y > panelY+panelH {
			continue
		}

		// Item background
		itemHovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+itemW) &&
			mousePos.Y >= float32(y) && mousePos.Y <= float32(y+itemH)

		isSelected := asset.Path == e.selectedMaterialPath

		bgColor := rl.NewColor(40, 40, 45, 200)
		if isSelected {
			bgColor = rl.NewColor(60, 80, 100, 220)
		} else if itemHovered {
			bgColor = rl.NewColor(55, 60, 55, 220)
		}
		rl.DrawRectangle(x, y, itemW, itemH, bgColor)

		// Draw icon based on type
		iconSize := int32(32)
		iconX := x + (itemW-iconSize)/2
		iconY := y + 6

		switch asset.Type {
		case "folder":
			// Folder icon (yellow rectangle with tab)
			rl.DrawRectangle(iconX, iconY+6, iconSize, iconSize-6, rl.NewColor(180, 150, 50, 220))
			rl.DrawRectangle(iconX, iconY, iconSize/2, 6, rl.NewColor(180, 150, 50, 220))
		case "material":
			// Material icon (circle/orb)
			rl.DrawCircle(iconX+iconSize/2, iconY+iconSize/2, float32(iconSize)/2-2, rl.NewColor(100, 150, 200, 220))
			rl.DrawCircleLines(iconX+iconSize/2, iconY+iconSize/2, float32(iconSize)/2-2, rl.NewColor(150, 180, 220, 255))
		case "model":
			// Model icon (cube)
			rl.DrawRectangle(iconX+4, iconY+4, iconSize-8, iconSize-8, rl.NewColor(100, 180, 100, 220))
			rl.DrawText("3D", iconX+8, iconY+10, 12, rl.White)
		case "texture":
			// Texture icon (checkerboard)
			half := iconSize / 2
			rl.DrawRectangle(iconX, iconY, half, half, rl.NewColor(200, 200, 200, 220))
			rl.DrawRectangle(iconX+half, iconY+half, half, half, rl.NewColor(200, 200, 200, 220))
			rl.DrawRectangle(iconX+half, iconY, half, half, rl.NewColor(100, 100, 100, 220))
			rl.DrawRectangle(iconX, iconY+half, half, half, rl.NewColor(100, 100, 100, 220))
		default:
			// Generic file icon
			rl.DrawRectangle(iconX+4, iconY, iconSize-8, iconSize, rl.NewColor(120, 120, 130, 220))
			rl.DrawTriangle(
				rl.NewVector2(float32(iconX+iconSize-4), float32(iconY)),
				rl.NewVector2(float32(iconX+iconSize-4), float32(iconY+8)),
				rl.NewVector2(float32(iconX+iconSize-12), float32(iconY)),
				rl.NewColor(80, 80, 90, 220),
			)
		}

		// Name (truncated)
		name := asset.Name
		if len(name) > 9 {
			name = name[:8] + "…"
		}
		textW := rl.MeasureText(name, 10)
		rl.DrawText(name, x+(itemW-textW)/2, y+itemH-14, 10, rl.LightGray)

		// Handle clicks
		if itemHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !e.draggingAsset {
			now := rl.GetTime()
			isDoubleClick := (now-e.lastClickTime < 0.3) && (e.lastClickedAsset == asset.Path)

			if asset.IsFolder {
				if isDoubleClick {
					// Double-click folder: navigate into it
					e.currentAssetPath = asset.Path
					e.assetBrowserScroll = 0
					e.selectedMaterial = nil
					e.selectedMaterialPath = ""
					e.scanAssets()
				}
			} else if asset.Type == "material" {
				// Start dragging material
				e.draggingAsset = true
				assetCopy := asset // Make a copy to avoid referencing loop variable
				e.draggedAsset = &assetCopy
				e.selectedMaterialPath = asset.Path
				e.selectedMaterial = assets.LoadMaterial(asset.Path)
			} else if asset.Type == "model" {
				// Click model: spawn into scene
				e.spawnModelFromAsset(asset)
			}

			e.lastClickTime = now
			e.lastClickedAsset = asset.Path
		}
	}

	rl.EndScissorMode()

	// Clamp scroll
	rows := (int32(len(e.assetFiles)) + cols - 1) / cols
	maxScroll := rows*(itemH+6) - (panelH - 28)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.assetBrowserScroll > maxScroll {
		e.assetBrowserScroll = maxScroll
	}

	// Empty state
	if len(e.assetFiles) == 0 {
		rl.DrawText("Empty folder", panelX+20, panelY+60, 14, rl.Gray)
	}

	// Draw material editor panel on the right
	if e.selectedMaterial != nil {
		e.drawMaterialEditor(panelX+contentW, panelY, panelW-contentW, panelH)
	}
}

// drawMaterialEditor draws the material properties editor in the asset browser
func (e *Editor) drawMaterialEditor(x, y, w, h int32) {
	// Background
	rl.DrawRectangle(x, y, w, h, rl.NewColor(30, 30, 35, 250))
	rl.DrawLine(x, y, x, y+h, rl.NewColor(60, 60, 60, 255))

	// Header
	name := filepath.Base(e.selectedMaterialPath)
	rl.DrawText(name, x+8, y+6, 12, rl.SkyBlue)

	// Close button
	closeBtnX := x + w - 20
	closeBtnY := y + 4
	closeBtnSize := int32(14)
	mousePos := rl.GetMousePosition()
	closeHovered := mousePos.X >= float32(closeBtnX) && mousePos.X <= float32(closeBtnX+closeBtnSize) &&
		mousePos.Y >= float32(closeBtnY) && mousePos.Y <= float32(closeBtnY+closeBtnSize)

	closeColor := rl.NewColor(80, 50, 50, 200)
	if closeHovered {
		closeColor = rl.NewColor(120, 60, 60, 220)
	}
	rl.DrawRectangle(closeBtnX, closeBtnY, closeBtnSize, closeBtnSize, closeColor)
	rl.DrawText("x", closeBtnX+3, closeBtnY, 12, rl.White)

	if closeHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.selectedMaterial = nil
		e.selectedMaterialPath = ""
		return
	}

	// Properties
	propY := y + 24
	labelW := int32(60)
	fieldW := w - labelW - 16
	fieldH := int32(16)
	indent := x + 8

	mat := e.selectedMaterial
	oldMet := mat.Metallic
	oldRough := mat.Roughness
	oldEmit := mat.Emissive

	// Material name (read-only for now)
	rl.DrawText("Name:", indent, propY+2, 11, rl.Gray)
	rl.DrawText(mat.Name, indent+labelW, propY+2, 11, rl.LightGray)
	propY += fieldH + 4

	// Color (read-only display)
	rl.DrawText("Color:", indent, propY+2, 11, rl.Gray)
	colorName := assets.LookupColorName(mat.Color)
	rl.DrawRectangle(indent+labelW, propY, fieldH, fieldH, mat.Color)
	rl.DrawText(colorName, indent+labelW+fieldH+4, propY+2, 11, rl.LightGray)
	propY += fieldH + 4

	// Metallic
	rl.DrawText("Metallic:", indent, propY+2, 11, rl.Gray)
	mat.Metallic = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.met", mat.Metallic)
	propY += fieldH + 4

	// Roughness
	rl.DrawText("Rough:", indent, propY+2, 11, rl.Gray)
	mat.Roughness = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.rough", mat.Roughness)
	propY += fieldH + 4

	// Emissive
	rl.DrawText("Emissive:", indent, propY+2, 11, rl.Gray)
	mat.Emissive = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.emit", mat.Emissive)
	propY += fieldH + 4

	// Albedo texture path (editable)
	rl.DrawText("Albedo:", indent, propY+2, 11, rl.Gray)
	oldAlbedo := mat.AlbedoPath
	mat.AlbedoPath = e.drawTextureField(indent+labelW, propY, fieldW, fieldH, "mated.albedo", mat.AlbedoPath)
	propY += fieldH + 4

	// Load texture if path changed
	albedoChanged := mat.AlbedoPath != oldAlbedo
	if albedoChanged && mat.AlbedoPath != "" {
		mat.Albedo = assets.LoadTexture(mat.AlbedoPath)
	} else if albedoChanged && mat.AlbedoPath == "" {
		mat.Albedo = rl.Texture2D{} // Clear texture
	}

	// Auto-save if changed
	if mat.Metallic != oldMet || mat.Roughness != oldRough || mat.Emissive != oldEmit || albedoChanged {
		assets.SaveMaterial(e.selectedMaterialPath, mat)
	}
}

// handleMaterialDrop handles dropping a material onto an object
func (e *Editor) handleMaterialDrop() {
	if e.draggedAsset == nil || e.draggedAsset.Type != "material" {
		e.draggingAsset = false
		e.draggedAsset = nil
		return
	}

	mousePos := rl.GetMousePosition()
	materialPath := e.draggedAsset.Path

	// Check if dropped on hierarchy item
	panelX := int32(0)
	panelY := int32(32)
	panelW := int32(200)
	panelH := int32(rl.GetScreenHeight()) - panelY
	itemH := int32(22)

	if mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH) {
		// Find which object was dropped on
		y := panelY + 28
		for i, g := range e.world.Scene.GameObjects {
			itemY := y + int32(i)*itemH - e.hierarchyScroll
			if mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH) {
				e.applyMaterialToObject(g, materialPath)
				break
			}
		}
	} else if !e.mouseInPanel() {
		// Dropped in 3D scene area - raycast to find object
		cam := e.GetRaylibCamera()
		ray := rl.GetScreenToWorldRay(mousePos, cam)
		hit, ok := e.world.EditorRaycast(ray.Position, ray.Direction, 1000)
		if ok && hit.GameObject != nil {
			e.applyMaterialToObject(hit.GameObject, materialPath)
		}
	}

	e.draggingAsset = false
	e.draggedAsset = nil
}

// applyMaterialToObject applies a material to an object's ModelRenderer
func (e *Editor) applyMaterialToObject(obj *engine.GameObject, materialPath string) {
	mr := engine.GetComponent[*components.ModelRenderer](obj)
	if mr == nil {
		e.saveMsg = fmt.Sprintf("%s has no ModelRenderer", obj.Name)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Load and apply the material
	mat := assets.LoadMaterial(materialPath)
	mr.Material = mat
	mr.MaterialPath = materialPath

	e.saveMsg = fmt.Sprintf("Applied material to %s", obj.Name)
	e.saveMsgTime = rl.GetTime()
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

// scanAssets scans the current asset directory and populates the asset list
func (e *Editor) scanAssets() {
	e.assetFiles = nil

	entries, err := os.ReadDir(e.currentAssetPath)
	if err != nil {
		return
	}

	// Sort: folders first, then files
	for _, entry := range entries {
		if entry.IsDir() {
			e.assetFiles = append(e.assetFiles, AssetEntry{
				Name:     entry.Name(),
				Path:     filepath.Join(e.currentAssetPath, entry.Name()),
				IsFolder: true,
				Type:     "folder",
			})
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		fullPath := filepath.Join(e.currentAssetPath, name)

		var assetType string
		switch ext {
		case ".json":
			// Check if in materials folder
			if strings.Contains(e.currentAssetPath, "materials") {
				assetType = "material"
			} else if strings.Contains(e.currentAssetPath, "scenes") {
				assetType = "scene"
			} else {
				assetType = "json"
			}
		case ".gltf", ".glb":
			assetType = "model"
		case ".png", ".jpg", ".jpeg":
			assetType = "texture"
		default:
			assetType = "file"
		}

		e.assetFiles = append(e.assetFiles, AssetEntry{
			Name:     name,
			Path:     fullPath,
			IsFolder: false,
			Type:     assetType,
		})
	}
}

// buildGame runs the Rust utility to build and package the game
func (e *Editor) buildGame() {
	e.saveMsg = "Building game..."
	e.saveMsgTime = rl.GetTime()

	cmd := exec.Command("./mirgo-utils", "build")
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.saveMsg = fmt.Sprintf("Build failed: %v", err)
		fmt.Printf("Build error: %v\nOutput: %s\n", err, string(output))
	} else {
		e.saveMsg = "Build complete! See build/"
		fmt.Printf("Build output:\n%s\n", string(output))
	}
	e.saveMsgTime = rl.GetTime()
}

// scanScriptModTimes records the modification times of all script files
func (e *Editor) scanScriptModTimes() {
	e.scriptModTimes = make(map[string]int64)
	e.scriptsChanged = false
	scriptsDir := "internal/scripts"

	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(scriptsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		e.scriptModTimes[path] = info.ModTime().UnixNano()
	}
	e.lastScriptCheck = rl.GetTime()
}

// checkScriptChanges checks if any script files have been modified
func (e *Editor) checkScriptChanges() {
	// Only check every 0.5 seconds
	if rl.GetTime()-e.lastScriptCheck < 0.5 {
		return
	}
	e.lastScriptCheck = rl.GetTime()

	scriptsDir := "internal/scripts"
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(scriptsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime().UnixNano()

		// Check if file is new or modified
		if oldTime, exists := e.scriptModTimes[path]; !exists || modTime != oldTime {
			e.scriptsChanged = true
			return
		}
	}
}

// EditorRestoreState holds editor state for hot-reload restoration
type EditorRestoreState struct {
	CameraPosition  rl.Vector3 `json:"cameraPosition"`
	CameraYaw       float32    `json:"cameraYaw"`
	CameraPitch     float32    `json:"cameraPitch"`
	CameraMoveSpeed float32    `json:"cameraMoveSpeed"`
	GizmoMode       int        `json:"gizmoMode"`
	SelectedUID     uint64     `json:"selectedUID,omitempty"`
}

const editorRestoreFile = ".editor_restore.json"

// RestoreState restores editor camera state from the restore file after hot-reload
func (e *Editor) RestoreState() {
	data, err := os.ReadFile(editorRestoreFile)
	if err != nil {
		return // No restore file, that's fine
	}

	var state EditorRestoreState
	if err := json.Unmarshal(data, &state); err != nil {
		fmt.Printf("Failed to parse restore state: %v\n", err)
		os.Remove(editorRestoreFile)
		return
	}

	// Apply restored state
	e.camera.Position = state.CameraPosition
	e.camera.Yaw = state.CameraYaw
	e.camera.Pitch = state.CameraPitch
	if state.CameraMoveSpeed > 0 {
		e.camera.MoveSpeed = state.CameraMoveSpeed
	}
	e.gizmoMode = GizmoMode(state.GizmoMode)

	// Restore selected object by UID
	if state.SelectedUID > 0 {
		e.Selected = e.world.Scene.FindByUID(state.SelectedUID)
	}

	// Clean up the restore file
	os.Remove(editorRestoreFile)

	fmt.Println("Scripts reloaded successfully")
}

// rebuildAndRelaunch saves state, rebuilds the binary, and relaunches
func (e *Editor) rebuildAndRelaunch() {
	e.saveMsg = "Compiling..."
	e.saveMsgTime = rl.GetTime()

	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		e.saveMsg = fmt.Sprintf("Failed to get executable: %v", err)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Build to a temp file first to check if it compiles
	tempExec := execPath + ".new"
	fmt.Println("Compiling...")

	// Run gen-scripts first to ensure scripts are up to date
	genCmd := exec.Command("go", "run", "./cmd/gen-scripts")
	genOutput, genErr := genCmd.CombinedOutput()
	if genErr != nil {
		e.saveMsg = "Script generation failed!"
		e.saveMsgTime = rl.GetTime()
		fmt.Printf("Script generation error:\n%s\n", string(genOutput))
		return
	}

	cmd := exec.Command("go", "build", "-o", tempExec, "./cmd/test3d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Build failed - show error, keep window open
		e.saveMsg = "Build failed!"
		e.saveMsgTime = rl.GetTime()
		fmt.Printf("Build error:\n%s\n", string(output))
		os.Remove(tempExec)
		return
	}

	// Build succeeded - now save state and relaunch
	e.saveMsg = "Reloading..."
	e.saveMsgTime = rl.GetTime()

	// Save the scene
	if err := e.world.SaveScene(world.ScenePath); err != nil {
		e.saveMsg = fmt.Sprintf("Save failed: %v", err)
		e.saveMsgTime = rl.GetTime()
		os.Remove(tempExec)
		return
	}

	// Save editor state for restoration
	state := EditorRestoreState{
		CameraPosition:  e.camera.Position,
		CameraYaw:       e.camera.Yaw,
		CameraPitch:     e.camera.Pitch,
		CameraMoveSpeed: e.camera.MoveSpeed,
		GizmoMode:       int(e.gizmoMode),
	}
	if e.Selected != nil {
		state.SelectedUID = e.Selected.UID
	}
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(editorRestoreFile, stateJSON, 0644); err != nil {
		e.saveMsg = fmt.Sprintf("Failed to save state: %v", err)
		e.saveMsgTime = rl.GetTime()
		os.Remove(tempExec)
		return
	}

	// Replace old binary with new one
	if err := os.Rename(tempExec, execPath); err != nil {
		e.saveMsg = fmt.Sprintf("Failed to replace binary: %v", err)
		e.saveMsgTime = rl.GetTime()
		os.Remove(tempExec)
		os.Remove(editorRestoreFile)
		return
	}

	fmt.Println("Relaunching...")

	// Close the window before relaunching
	rl.CloseWindow()

	// Replace current process with new binary
	err = execNewBinary(execPath, []string{execPath, "--restore-editor"})
	if err != nil {
		fmt.Printf("Failed to exec: %v\n", err)
		os.Exit(1)
	}
}
