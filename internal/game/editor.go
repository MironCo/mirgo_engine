//go:build !game

package game

import (
	"fmt"
	"math"
	"test3d/internal/assets"
	"test3d/internal/audio"
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
	addComponentScroll   int32 // Scroll offset for add component menu

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
	lastHierarchyClick   float64          // For hierarchy double-click detection
	lastClickedObject    *engine.GameObject // Last clicked object in hierarchy

	// Script hot-reload
	scriptModTimes  map[string]int64 // path -> mod time (unix nano)
	scriptsChanged  bool
	lastScriptCheck float64

	// Drag-and-drop state
	draggingAsset       bool               // True if dragging an asset from the browser
	draggedAsset        *AssetEntry        // The asset being dragged
	draggingHierarchy   bool               // True if dragging an object in hierarchy
	draggedObject       *engine.GameObject // The object being dragged for reparenting
	hierarchyDropTarget *engine.GameObject // Target for hierarchy drop (parent candidate)
	hierarchyDropIndex  int                // Index where to drop (-1 = as child, >= 0 = at position)

	// Name editing state
	editingName    bool   // True if editing the object name
	nameEditBuffer string // Current text in name edit field

	// Tags editing state
	editingTags    bool   // True if editing tags
	tagsEditBuffer string // Current text in tags edit field

	// Panel sizing
	hierarchyWidth  int32 // Width of hierarchy panel (default 210)
	inspectorWidth  int32 // Width of inspector panel (default 310)
	resizingPanel   int   // 0=none, 1=hierarchy, 2=inspector
	resizeStartX    float32
	resizeStartW    int32

	// Camera zoom animation
	zoomingToTarget bool
	zoomTargetPos   rl.Vector3
	zoomStartPos    rl.Vector3
	zoomProgress    float32
}

func NewEditor(w *world.World) *Editor {
	return &Editor{
		world: w,
		camera: EditorCamera{
			MoveSpeed: 10.0,
		},
		hoveredAxis:    -1,
		undoStack:      make([]UndoState, 0, maxUndoStack),
		hierarchyWidth: 210,
		inspectorWidth: 310,
	}
}

func (e *Editor) Enter(currentCam rl.Camera3D) {
	e.Active = true
	e.Paused = false
	rl.EnableCursor()
	audio.SetPlayMode(false)

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
	audio.SetPlayMode(false)

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

func (e *Editor) Exit() {
	// Save scene before entering play mode
	if err := e.world.SaveScene(world.ScenePath); err != nil {
		fmt.Printf("Warning: Failed to save scene before play mode: %v\n", err)
	}

	e.Active = false
	e.Paused = false
	e.Selected = nil
	e.dragging = false
	e.hoveredAxis = -1
	rl.DisableCursor()
	audio.SetPlayMode(true)
}

func (e *Editor) Update(deltaTime float32) {
	// Update camera zoom animation
	e.updateCameraZoom(deltaTime)

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

	// Ctrl+D: duplicate selected object
	if e.Selected != nil && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyLeftSuper)) && rl.IsKeyPressed(rl.KeyD) {
		newObj := e.world.DuplicateObject(e.Selected)
		e.Selected = newObj
	}

	// Cmd+Delete (Mac) or Ctrl+Delete: delete selected object
	if e.Selected != nil && (rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyLeftControl)) && rl.IsKeyPressed(rl.KeyBackspace) {
		e.deleteSelectedObject()
	}

	// Camera: right-click + drag to look, right-click + WASD to fly
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		// Cancel any zoom animation when user takes manual control
		e.zoomingToTarget = false

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

	// Scroll wheel + Shift adjusts fly speed
	scroll := rl.GetMouseWheelMove()
	if scroll != 0 && (rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)) {
		e.camera.MoveSpeed += scroll * 2.0
		if e.camera.MoveSpeed < 1.0 {
			e.camera.MoveSpeed = 1.0
		}
		if e.camera.MoveSpeed > 100.0 {
			e.camera.MoveSpeed = 100.0
		}
	}

	// Gizmo mode hotkeys (only when not holding RMB for camera and not editing text)
	isEditingText := e.editingName || e.editingTags || e.activeInputID != ""
	if !rl.IsMouseButtonDown(rl.MouseRightButton) && !isEditingText {
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

// focusOnObject starts a smooth camera zoom to look at the given object.
func (e *Editor) focusOnObject(obj *engine.GameObject) {
	if obj == nil {
		return
	}

	// Get object's world position
	targetPos := obj.WorldPosition()

	// Calculate object size to determine camera distance
	objectRadius := float32(1.0) // Default if no renderer

	// Try to get bounds from ModelRenderer
	if mr := engine.GetComponent[*components.ModelRenderer](obj); mr != nil {
		bounds := rl.GetModelBoundingBox(mr.Model)
		scale := obj.WorldScale()

		// Calculate size from bounds
		sizeX := (bounds.Max.X - bounds.Min.X) * float32(math.Abs(float64(scale.X)))
		sizeY := (bounds.Max.Y - bounds.Min.Y) * float32(math.Abs(float64(scale.Y)))
		sizeZ := (bounds.Max.Z - bounds.Min.Z) * float32(math.Abs(float64(scale.Z)))

		// Use largest dimension as radius
		objectRadius = sizeX
		if sizeY > objectRadius {
			objectRadius = sizeY
		}
		if sizeZ > objectRadius {
			objectRadius = sizeZ
		}
		objectRadius /= 2
	}

	// Position camera at a nice distance (3x object radius, minimum 3 units)
	distance := objectRadius * 3
	if distance < 3 {
		distance = 3
	}

	// Calculate camera position looking at object from current yaw/pitch direction
	yawRad := float64(e.camera.Yaw) * math.Pi / 180
	pitchRad := float64(e.camera.Pitch) * math.Pi / 180

	// Camera offset from target (opposite of look direction)
	offsetX := float32(-math.Cos(yawRad)*math.Cos(pitchRad)) * distance
	offsetY := float32(-math.Sin(pitchRad)) * distance
	offsetZ := float32(-math.Sin(yawRad)*math.Cos(pitchRad)) * distance

	// Start zoom animation
	e.zoomingToTarget = true
	e.zoomStartPos = e.camera.Position
	e.zoomTargetPos = rl.Vector3{
		X: targetPos.X + offsetX,
		Y: targetPos.Y + offsetY,
		Z: targetPos.Z + offsetZ,
	}
	e.zoomProgress = 0
}

// updateCameraZoom updates the smooth camera zoom animation.
func (e *Editor) updateCameraZoom(deltaTime float32) {
	if !e.zoomingToTarget {
		return
	}

	// Zoom speed - complete in ~0.3 seconds
	speed := float32(4.0)
	e.zoomProgress += deltaTime * speed

	if e.zoomProgress >= 1.0 {
		e.zoomProgress = 1.0
		e.zoomingToTarget = false
		e.camera.Position = e.zoomTargetPos
		return
	}

	// Smooth easing (ease-out cubic)
	t := e.zoomProgress
	ease := 1 - (1-t)*(1-t)*(1-t)

	// Lerp position
	e.camera.Position = rl.Vector3{
		X: e.zoomStartPos.X + (e.zoomTargetPos.X-e.zoomStartPos.X)*ease,
		Y: e.zoomStartPos.Y + (e.zoomTargetPos.Y-e.zoomStartPos.Y)*ease,
		Z: e.zoomStartPos.Z + (e.zoomTargetPos.Z-e.zoomStartPos.Z)*ease,
	}
}

// DrawUI draws the editor overlay: top bar, hierarchy panel (left), inspector panel (right).
func (e *Editor) DrawUI() {
	// Top bar - dark with subtle border
	rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), 36, colorBgDark)
	rl.DrawRectangle(0, 35, int32(rl.GetScreenWidth()), 1, colorBorder)

	// Mode indicator with accent color
	if e.Paused {
		drawTextEx(editorFontBold, "PAUSED", 12, 7, 22, rl.Orange)
	} else {
		drawTextEx(editorFontBold, "EDITOR", 12, 7, 22, colorAccent)
	}

	// Gizmo mode indicator
	modeNames := [3]string{"[W] Move", "[E] Rotate", "[R] Scale"}
	for i, name := range modeNames {
		x := int32(115 + i*100)
		color := colorTextMuted
		if GizmoMode(i) == e.gizmoMode {
			color = colorAccentLight
		}
		drawTextEx(editorFont, name, x, 9, 18, color)
	}
	helpText := "Ctrl+S: Save  |  Ctrl+B: Build  |  Ctrl+Z: Undo"
	if e.Paused {
		helpText = "P: Resume  |  Ctrl+S: Save"
	}
	drawTextEx(editorFont, helpText, 430, 9, 18, colorTextMuted)
	drawTextEx(editorFontMono, fmt.Sprintf("Speed: %.0f", e.camera.MoveSpeed), int32(rl.GetScreenWidth())-130, 9, 18, colorTextMuted)

	// Scripts changed banner (below top bar) - indigo themed
	if e.scriptsChanged {
		bannerText := "Scripts changed - Press Ctrl+R to rebuild"
		textWidth := rl.MeasureText(bannerText, 14)
		bannerX := (int32(rl.GetScreenWidth()) - textWidth) / 2
		rl.DrawRectangle(bannerX-12, 42, textWidth+24, 26, rl.NewColor(108, 99, 255, 40))
		rl.DrawRectangleLines(bannerX-12, 42, textWidth+24, 26, colorAccent)
		drawTextEx(editorFont, bannerText, bannerX, 47, 14, colorAccentLight)
	}

	// Save/build message flash (below top bar)
	if e.saveMsg != "" && rl.GetTime()-e.saveMsgTime < 2.0 {
		color := rl.NewColor(100, 220, 100, 255) // Soft green
		if e.saveMsg != "Scene saved!" {
			color = rl.NewColor(255, 120, 120, 255) // Soft red
		}
		drawTextEx(editorFontBold, e.saveMsg, int32(rl.GetScreenWidth()/2)-50, 47, 16, color)
	}

	// Reset field hover tracking for this frame
	e.fieldHoveredAny = false

	e.drawHierarchy()
	e.drawInspector()

	// Asset browser toggle button in top bar - pill shaped (positioned left of speed)
	abBtnW := int32(95)
	abBtnX := int32(rl.GetScreenWidth()) - 240
	abBtnH := int32(24)
	abBtnY := int32(6)

	mousePos := rl.GetMousePosition()
	abHovered := mousePos.X >= float32(abBtnX) && mousePos.X <= float32(abBtnX+abBtnW) &&
		mousePos.Y >= float32(abBtnY) && mousePos.Y <= float32(abBtnY+abBtnH)

	abBtnColor := colorBgElement
	textColor := colorTextSecondary
	if e.showAssetBrowser {
		abBtnColor = colorAccent
		textColor = colorTextPrimary
	} else if abHovered {
		abBtnColor = colorBgHover
		textColor = colorTextPrimary
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(abBtnX), Y: float32(abBtnY), Width: float32(abBtnW), Height: float32(abBtnH)}, 0.5, 8, abBtnColor)
	drawTextEx(editorFont, "Assets [Tab]", abBtnX+10, abBtnY+4, 16, textColor)

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

	// Draw material drag indicator - indigo themed
	if e.draggingAsset && e.draggedAsset != nil {
		mousePos := rl.GetMousePosition()
		rl.DrawRectangleRounded(rl.Rectangle{X: mousePos.X + 12, Y: mousePos.Y - 10, Width: 85, Height: 22}, 0.3, 6, colorAccent)
		name := e.draggedAsset.Name
		if len(name) > 12 {
			name = name[:11] + "…"
		}
		drawTextEx(editorFont, name, int32(mousePos.X)+18, int32(mousePos.Y)-6, 14, colorTextPrimary)

		// Handle drop on mouse release
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			e.handleMaterialDrop()
		}
	}

	// Handle panel resize
	e.handlePanelResize()

	// Set cursor based on state
	if e.resizingPanel > 0 || e.isOverPanelEdge() {
		rl.SetMouseCursor(rl.MouseCursorResizeEW)
	} else if e.fieldHoveredAny || e.fieldDragging {
		rl.SetMouseCursor(rl.MouseCursorResizeEW)
	} else {
		rl.SetMouseCursor(rl.MouseCursorDefault)
	}
}

// isOverPanelEdge checks if mouse is over a resizable panel edge
func (e *Editor) isOverPanelEdge() bool {
	mousePos := rl.GetMousePosition()
	screenH := float32(rl.GetScreenHeight())
	screenW := float32(rl.GetScreenWidth())

	// Hierarchy right edge (4px hit zone)
	hierEdge := float32(e.hierarchyWidth)
	if mousePos.X >= hierEdge-2 && mousePos.X <= hierEdge+2 && mousePos.Y > 36 && mousePos.Y < screenH {
		return true
	}

	// Inspector left edge
	inspEdge := screenW - float32(e.inspectorWidth)
	if mousePos.X >= inspEdge-2 && mousePos.X <= inspEdge+2 && mousePos.Y > 36 && mousePos.Y < screenH {
		return true
	}

	return false
}

// handlePanelResize handles drag-to-resize for panels
func (e *Editor) handlePanelResize() {
	mousePos := rl.GetMousePosition()
	screenW := int32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())

	// Start resize on mouse down
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && e.resizingPanel == 0 {
		hierEdge := float32(e.hierarchyWidth)
		inspEdge := float32(screenW) - float32(e.inspectorWidth)

		if mousePos.X >= hierEdge-2 && mousePos.X <= hierEdge+2 && mousePos.Y > 36 && mousePos.Y < screenH {
			e.resizingPanel = 1
			e.resizeStartX = mousePos.X
			e.resizeStartW = e.hierarchyWidth
		} else if mousePos.X >= inspEdge-2 && mousePos.X <= inspEdge+2 && mousePos.Y > 36 && mousePos.Y < screenH {
			e.resizingPanel = 2
			e.resizeStartX = mousePos.X
			e.resizeStartW = e.inspectorWidth
		}
	}

	// Update while dragging
	if e.resizingPanel > 0 && rl.IsMouseButtonDown(rl.MouseLeftButton) {
		delta := int32(mousePos.X - e.resizeStartX)

		if e.resizingPanel == 1 {
			// Hierarchy - drag right edge
			newW := e.resizeStartW + delta
			if newW < 150 {
				newW = 150
			} else if newW > 400 {
				newW = 400
			}
			e.hierarchyWidth = newW
		} else if e.resizingPanel == 2 {
			// Inspector - drag left edge (inverted)
			newW := e.resizeStartW - delta
			if newW < 250 {
				newW = 250
			} else if newW > 500 {
				newW = 500
			}
			e.inspectorWidth = newW
		}
	}

	// End resize
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		e.resizingPanel = 0
	}
}

// mouseInPanel returns true if the mouse is over the hierarchy or inspector panel.
func (e *Editor) mouseInPanel() bool {
	m := rl.GetMousePosition()
	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())
	hierW := float32(e.hierarchyWidth)
	inspW := float32(e.inspectorWidth)

	// Hierarchy panel
	if m.X <= hierW && m.Y >= 36 && m.Y <= screenH {
		return true
	}
	// Inspector panel
	if m.X >= screenW-inspW && m.Y >= 36 && m.Y <= screenH {
		return true
	}
	// Top bar
	if m.Y <= 36 {
		return true
	}
	// Asset browser: bottom 150px (when visible)
	if e.showAssetBrowser && m.Y >= screenH-150 && m.X > hierW && m.X < screenW-inspW {
		return true
	}
	return false
}
