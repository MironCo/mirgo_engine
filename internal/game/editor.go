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

	gui "github.com/gen2brain/raylib-go/raygui"
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

	// Panel sizing
	hierarchyWidth  int32 // Width of hierarchy panel (default 210)
	inspectorWidth  int32 // Width of inspector panel (default 310)
	resizingPanel   int   // 0=none, 1=hierarchy, 2=inspector
	resizeStartX    float32
	resizeStartW    int32
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

// Editor fonts - Outfit for UI, JetBrains Mono for values
var editorFont rl.Font      // Outfit Regular - main UI font
var editorFontBold rl.Font  // Outfit Bold - headers
var editorFontMono rl.Font  // JetBrains Mono - numeric values
var editorFontsLoaded bool

// Theme colors - Indigo/purple dark theme matching the website
var (
	// Base backgrounds (dark with slight blue tint)
	colorBgDark    = rl.NewColor(10, 10, 15, 255)   // Darkest - nav bg
	colorBgPanel   = rl.NewColor(18, 18, 24, 245)   // Panel backgrounds
	colorBgElement = rl.NewColor(28, 28, 38, 255)   // Input fields, buttons
	colorBgHover   = rl.NewColor(38, 38, 52, 255)   // Hover state
	colorBgActive  = rl.NewColor(48, 48, 65, 255)   // Active/pressed state

	// Accent colors - indigo/purple gradient
	colorAccent       = rl.NewColor(108, 99, 255, 255)  // Primary indigo #6c63ff
	colorAccentLight  = rl.NewColor(167, 139, 250, 255) // Light purple #a78bfa
	colorAccentHover  = rl.NewColor(130, 120, 255, 255) // Hover indigo
	colorAccentActive = rl.NewColor(90, 80, 220, 255)   // Pressed indigo

	// Text colors
	colorTextPrimary   = rl.NewColor(255, 255, 255, 255) // White
	colorTextSecondary = rl.NewColor(200, 200, 208, 255) // Light gray #c8c8d0
	colorTextMuted     = rl.NewColor(119, 119, 119, 255) // Muted #777

	// Borders
	colorBorder      = rl.NewColor(255, 255, 255, 13)  // rgba(255,255,255,0.05)
	colorBorderHover = rl.NewColor(108, 99, 255, 100)  // Indigo border on hover

	// Selection highlight (indigo tinted)
	colorSelection = rl.NewColor(108, 99, 255, 60) // Indigo with transparency
)

// initRayguiStyle sets up the modern indigo dark theme
func initRayguiStyle() {
	// Load fonts at high resolution for smooth scaling
	if !editorFontsLoaded {
		editorFontsLoaded = true

		// Load Outfit Regular for main UI (high res for smooth scaling)
		editorFont = rl.LoadFontEx("assets/fonts/Outfit-Regular.ttf", 48, nil)
		if editorFont.Texture.ID > 0 {
			rl.SetTextureFilter(editorFont.Texture, rl.FilterBilinear)
			gui.SetFont(editorFont)
			log.Println("Loaded Outfit-Regular font")
		} else {
			log.Println("Failed to load Outfit-Regular font")
		}

		// Load Outfit Bold for headers
		editorFontBold = rl.LoadFontEx("assets/fonts/Outfit-Bold.ttf", 48, nil)
		if editorFontBold.Texture.ID > 0 {
			rl.SetTextureFilter(editorFontBold.Texture, rl.FilterBilinear)
			log.Println("Loaded Outfit-Bold font")
		} else {
			log.Println("Failed to load Outfit-Bold font")
		}

		// Load JetBrains Mono for numeric values
		editorFontMono = rl.LoadFontEx("assets/fonts/JetBrainsMono-Regular.ttf", 48, nil)
		if editorFontMono.Texture.ID > 0 {
			rl.SetTextureFilter(editorFontMono.Texture, rl.FilterBilinear)
			log.Println("Loaded JetBrainsMono font")
		} else {
			log.Println("Failed to load JetBrainsMono font")
		}
	}

	// Background colors - dark with blue tint
	gui.SetStyle(gui.DEFAULT, gui.BACKGROUND_COLOR, gui.NewColorPropertyValue(colorBgDark))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_NORMAL, gui.NewColorPropertyValue(colorBgElement))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_FOCUSED, gui.NewColorPropertyValue(colorBgHover))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_PRESSED, gui.NewColorPropertyValue(colorAccent))

	// Text colors
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_NORMAL, gui.NewColorPropertyValue(colorTextSecondary))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_FOCUSED, gui.NewColorPropertyValue(colorTextPrimary))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_PRESSED, gui.NewColorPropertyValue(colorTextPrimary))

	// Border colors - subtle with indigo on focus
	gui.SetStyle(gui.DEFAULT, gui.BORDER_COLOR_NORMAL, gui.NewColorPropertyValue(rl.NewColor(50, 50, 65, 255)))
	gui.SetStyle(gui.DEFAULT, gui.BORDER_COLOR_FOCUSED, gui.NewColorPropertyValue(colorAccent))

	// Line color (for separators)
	gui.SetStyle(gui.DEFAULT, gui.LINE_COLOR, gui.NewColorPropertyValue(rl.NewColor(40, 40, 55, 255)))

	// Text size
	gui.SetStyle(gui.DEFAULT, gui.TEXT_SIZE, 15)
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
	isEditingText := e.editingName || e.activeInputID != ""
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

// drawTextEx draws text using the specified font scaled to the requested size
func drawTextEx(font rl.Font, text string, x, y int32, size float32, color rl.Color) {
	if font.Texture.ID > 0 {
		rl.DrawTextEx(font, text, rl.Vector2{X: float32(x), Y: float32(y)}, size, 0, color)
	} else {
		rl.DrawText(text, x, y, int32(size), color)
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

// drawHierarchy draws the scene hierarchy panel on the left.
func (e *Editor) drawHierarchy() {
	panelX := int32(0)
	panelY := int32(36)
	panelW := e.hierarchyWidth
	panelH := int32(rl.GetScreenHeight()) - panelY

	// Panel background with subtle border
	rl.DrawRectangle(panelX, panelY, panelW, panelH, colorBgPanel)
	// Resize handle - slightly thicker border on right edge
	rl.DrawRectangle(panelX+panelW-2, panelY, 2, panelH, colorBorder)

	// Header
	drawTextEx(editorFontBold, "Hierarchy", panelX+12, panelY+8, 18, colorTextSecondary)

	// "New Object" button - rounded pill
	btnX := panelX + panelW - 62
	btnY := panelY + 6
	btnW := int32(54)
	btnH := int32(22)

	mousePos := rl.GetMousePosition()
	btnHovered := mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
		mousePos.Y >= float32(btnY) && mousePos.Y <= float32(btnY+btnH)

	btnColor := colorBgElement
	textColor := colorTextSecondary
	if btnHovered {
		btnColor = colorAccent
		textColor = colorTextPrimary
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(btnX), Y: float32(btnY), Width: float32(btnW), Height: float32(btnH)}, 0.5, 6, btnColor)
	drawTextEx(editorFont, "+ New", btnX+8, btnY+3, 16, textColor)

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
			// Highlight as drop target - indigo
			rl.DrawRectangle(panelX, itemY, panelW, itemH, rl.NewColor(108, 99, 255, 60))
			e.hierarchyDropTarget = g
		} else if selected {
			// Selected - indigo tint
			rl.DrawRectangle(panelX, itemY, panelW, itemH, colorSelection)
			rl.DrawRectangle(panelX, itemY, 3, itemH, colorAccent) // Left accent bar
		} else if hovered {
			rl.DrawRectangle(panelX, itemY, panelW, itemH, colorBgHover)
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

		txtColor := colorTextSecondary
		if selected {
			txtColor = colorAccentLight
		}
		if e.draggingHierarchy && e.draggedObject == g {
			txtColor = colorAccent // Indicate dragged item
		}
		drawTextEx(editorFont, g.Name, panelX+indent, itemY+3, 16, txtColor)
	}

	rl.EndScissorMode()

	// "Unparent" drop zone at bottom of hierarchy (drawn outside scissor mode)
	if e.draggingHierarchy && e.draggedObject != nil && e.draggedObject.Parent != nil {
		unparentY := panelY + panelH - itemH - 4
		unparentHovered := mouseInPanel && mousePos.Y >= float32(unparentY) && mousePos.Y <= float32(panelY+panelH)

		bgColor := rl.NewColor(80, 50, 50, 180)
		if unparentHovered {
			bgColor = rl.NewColor(180, 80, 80, 200)
			e.hierarchyDropTarget = nil // nil means unparent
			e.hierarchyDropIndex = -2   // special value for unparent
		}
		rl.DrawRectangle(panelX, unparentY, panelW, itemH, bgColor)
		drawTextEx(editorFont, "— Unparent —", panelX+55, unparentY+3, 16, colorTextSecondary)
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
		drawTextEx(editorFont, e.draggedObject.Name, int32(mousePos.X)+10, int32(mousePos.Y)-8, 14, colorAccentLight)
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

	panelW := e.inspectorWidth
	panelX := int32(rl.GetScreenWidth()) - panelW
	panelY := int32(36)
	panelH := int32(rl.GetScreenHeight()) - panelY

	// Panel background with subtle border
	rl.DrawRectangle(panelX, panelY, panelW, panelH, colorBgPanel)
	// Resize handle - slightly thicker border on left edge
	rl.DrawRectangle(panelX, panelY, 2, panelH, colorBorder)

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

	// Background for name field - rounded
	nameBgColor := colorBgElement
	if e.editingName {
		nameBgColor = colorBgActive
	} else if nameHovered {
		nameBgColor = colorBgHover
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(nameFieldX), Y: float32(nameFieldY), Width: float32(nameFieldW), Height: float32(nameFieldH)}, 0.2, 6, nameBgColor)
	if e.editingName {
		rl.DrawRectangleRoundedLinesEx(rl.Rectangle{X: float32(nameFieldX), Y: float32(nameFieldY), Width: float32(nameFieldW), Height: float32(nameFieldH)}, 0.2, 6, 1, colorAccent)
	}

	if e.editingName {
		// Draw editing text with cursor
		drawTextEx(editorFont, e.nameEditBuffer+"_", nameFieldX+8, nameFieldY+4, 18, colorTextPrimary)

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
		// Display name with accent color
		drawTextEx(editorFontBold, e.Selected.Name, nameFieldX+8, nameFieldY+4, 18, colorAccentLight)

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
		drawTextEx(editorFont, "Tags: "+tagStr, panelX+12, y, 16, colorTextMuted)
		y += 22
	}

	// Separator
	rl.DrawLine(panelX+12, y+2, panelX+panelW-12, y+2, rl.NewColor(40, 40, 55, 255))
	y += 10

	// Transform section
	y = e.drawTransformSection(panelX, y, panelW)

	// Separator
	rl.DrawLine(panelX+12, y+2, panelX+panelW-12, y+2, rl.NewColor(40, 40, 55, 255))
	y += 10

	// Components section header
	drawTextEx(editorFontBold, "Components", panelX+12, y, 18, colorTextSecondary)
	y += 26

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

	// Add Component button - rounded with accent on hover
	y += 10
	btnW := panelW - 40
	btnH := int32(26)
	btnX := panelX + 20
	btnY := y

	hovered := mouseInPanel && mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
		mousePos.Y >= float32(btnY+e.inspectorScroll) && mousePos.Y <= float32(btnY+btnH+e.inspectorScroll)

	btnColor := colorBgElement
	txtColor := colorTextSecondary
	if hovered {
		btnColor = colorAccent
		txtColor = colorTextPrimary
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(btnX), Y: float32(btnY), Width: float32(btnW), Height: float32(btnH)}, 0.3, 6, btnColor)
	textW := rl.MeasureText("+ Add Component", 16)
	drawTextEx(editorFont, "+ Add Component", btnX+(btnW-textW)/2, btnY+5, 16, txtColor)

	clickedAddButton := false
	if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.showAddComponentMenu = !e.showAddComponentMenu
		if e.showAddComponentMenu {
			e.addComponentScroll = 0 // Reset scroll when opening menu
		}
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
	drawTextEx(editorFontBold, "Transform", panelX+12, y, 18, colorTextSecondary)
	y += 28

	labelW := int32(45)
	fieldW := (panelW - 38 - labelW) / 3
	fieldH := int32(24)
	startX := panelX + 12 + labelW

	// Position
	drawTextEx(editorFont, "Pos", panelX+14, y+4, 16, colorTextMuted)
	e.Selected.Transform.Position.X = e.drawFloatField(startX, y, fieldW, fieldH, "pos.x", e.Selected.Transform.Position.X)
	e.Selected.Transform.Position.Y = e.drawFloatField(startX+fieldW+2, y, fieldW, fieldH, "pos.y", e.Selected.Transform.Position.Y)
	e.Selected.Transform.Position.Z = e.drawFloatField(startX+2*(fieldW+2), y, fieldW, fieldH, "pos.z", e.Selected.Transform.Position.Z)
	y += fieldH + 4

	if e.Selected.Parent != nil {
		wPos := e.Selected.WorldPosition()
		drawTextEx(editorFontMono, fmt.Sprintf("World %.1f, %.1f, %.1f", wPos.X, wPos.Y, wPos.Z), panelX+16, y, 14, colorTextMuted)
		y += 18
	}

	// Rotation
	drawTextEx(editorFont, "Rot", panelX+14, y+4, 16, colorTextMuted)
	e.Selected.Transform.Rotation.X = e.drawFloatField(startX, y, fieldW, fieldH, "rot.x", e.Selected.Transform.Rotation.X)
	e.Selected.Transform.Rotation.Y = e.drawFloatField(startX+fieldW+2, y, fieldW, fieldH, "rot.y", e.Selected.Transform.Rotation.Y)
	e.Selected.Transform.Rotation.Z = e.drawFloatField(startX+2*(fieldW+2), y, fieldW, fieldH, "rot.z", e.Selected.Transform.Rotation.Z)
	y += fieldH + 4

	// Scale
	drawTextEx(editorFont, "Scale", panelX+14, y+4, 16, colorTextMuted)
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

	// Background color - indigo themed
	bgColor := colorBgElement
	if editMode {
		bgColor = colorBgActive
	} else if hovered || isDragging {
		bgColor = colorBgHover
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(w), Height: float32(h)}, 0.2, 4, bgColor)
	if editMode {
		rl.DrawRectangleRoundedLinesEx(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(w), Height: float32(h)}, 0.2, 4, 1, colorAccent)
	}

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
		drawTextEx(editorFontMono, e.inputTextValue+"_", x+6, y+5, 15, colorTextPrimary)

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
		// Display current value - monospace for numbers
		text := strconv.FormatFloat(float64(value), 'f', 2, 32)
		drawTextEx(editorFontMono, text, x+6, y+5, 15, colorTextSecondary)
	}

	return value
}

// drawTextureField draws an editable text field for texture paths
func (e *Editor) drawTextureField(x, y, w, h int32, id string, value string) string {
	mousePos := rl.GetMousePosition()
	hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
		mousePos.Y >= float32(y) && mousePos.Y <= float32(y+h)

	editMode := e.activeInputID == id

	// Background color - indigo themed
	bgColor := colorBgElement
	if editMode {
		bgColor = colorBgActive
	} else if hovered {
		bgColor = colorBgHover
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(w), Height: float32(h)}, 0.2, 4, bgColor)
	if editMode {
		rl.DrawRectangleRoundedLinesEx(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(w), Height: float32(h)}, 0.2, 4, 1, colorAccent)
	}

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
		drawTextEx(editorFontMono, displayText+"_", x+6, y+4, 14, colorTextPrimary)

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
		txtColor := colorTextSecondary
		if value == "" {
			txtColor = colorTextMuted
		}
		drawTextEx(editorFontMono, displayText, x+6, y+4, 14, txtColor)
	}

	return value
}

// drawComponentEntry draws a single component with its properties and X button.
// Returns the new Y position and whether the component should be removed.
func (e *Editor) drawComponentEntry(panelX, y, panelW int32, index int, c engine.Component, mouseInPanel bool) (int32, bool) {
	typeName := reflect.TypeOf(c).Elem().Name()

	// Component header with X button
	headerH := int32(24)
	xBtnSize := int32(18)
	xBtnX := panelX + panelW - 32
	xBtnY := y + 3

	// Draw header background - rounded
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(panelX + 10), Y: float32(y), Width: float32(panelW - 20), Height: float32(headerH)}, 0.15, 4, colorBgElement)
	drawTextEx(editorFontBold, typeName, panelX+16, y+4, 16, colorTextSecondary)

	// Draw X button - rounded
	mousePos := rl.GetMousePosition()
	// Adjust for scroll when checking hover
	adjustedY := float32(xBtnY + e.inspectorScroll)
	xHovered := mouseInPanel &&
		mousePos.X >= float32(xBtnX) && mousePos.X <= float32(xBtnX+xBtnSize) &&
		mousePos.Y >= adjustedY-float32(e.inspectorScroll) && mousePos.Y <= adjustedY+float32(xBtnSize)-float32(e.inspectorScroll)

	xBtnColor := rl.NewColor(100, 50, 50, 200)
	if xHovered {
		xBtnColor = rl.NewColor(180, 60, 60, 230)
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(xBtnX), Y: float32(xBtnY), Width: float32(xBtnSize), Height: float32(xBtnSize)}, 0.3, 4, xBtnColor)
	drawTextEx(editorFontBold, "x", xBtnX+5, xBtnY+2, 14, colorTextPrimary)

	shouldRemove := xHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton)
	y += headerH + 4

	// Draw component-specific properties
	y = e.drawComponentProperties(panelX, y, c, index)

	return y, shouldRemove
}

// drawComponentProperties draws editable properties for each component type.
func (e *Editor) drawComponentProperties(panelX, y int32, c engine.Component, compIdx int) int32 {
	indent := panelX + 16
	labelW := int32(80)
	fieldW := int32(75)
	fieldH := int32(22)

	switch comp := c.(type) {
	case *components.ModelRenderer:
		if comp.FilePath != "" {
			drawTextEx(editorFont, fmt.Sprintf("Model: %s", filepath.Base(comp.FilePath)), indent, y, 15, colorTextMuted)
			y += 20
		} else {
			drawTextEx(editorFont, fmt.Sprintf("Mesh: %s", comp.MeshType), indent, y, 15, colorTextMuted)
			y += 20
		}

		// Material asset reference
		if comp.MaterialPath != "" {
			drawTextEx(editorFont, fmt.Sprintf("Material: %s", filepath.Base(comp.MaterialPath)), indent, y, 15, colorAccentLight)
			y += 20
			// Editable material properties (saves to material file)
			if comp.Material != nil {
				id := fmt.Sprintf("mat%d", compIdx)
				oldMet := comp.Material.Metallic
				oldRough := comp.Material.Roughness
				oldEmit := comp.Material.Emissive

				drawTextEx(editorFont, "Metallic", indent, y+4, 15, colorTextMuted)
				comp.Material.Metallic = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".met", comp.Material.Metallic)
				y += fieldH + 2

				drawTextEx(editorFont, "Roughness", indent, y+4, 15, colorTextMuted)
				comp.Material.Roughness = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".rough", comp.Material.Roughness)
				y += fieldH + 2

				drawTextEx(editorFont, "Emissive", indent, y+4, 15, colorTextMuted)
				comp.Material.Emissive = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".emit", comp.Material.Emissive)
				y += fieldH + 4

				// Save material if any value changed
				if comp.Material.Metallic != oldMet || comp.Material.Roughness != oldRough || comp.Material.Emissive != oldEmit {
					assets.SaveMaterial(comp.MaterialPath, comp.Material)
				}
			}
		} else if comp.FilePath != "" {
			// GLTF model using built-in materials
			drawTextEx(editorFont, "Material: Built-in", indent, y, 15, colorTextMuted)
			y += 20
		} else {
			// Generated mesh - inline material properties (editable)
			// Color dropdown would go here - for now just display
			drawTextEx(editorFont, fmt.Sprintf("Color: %s", colorName(comp.Color)), indent, y, 15, colorTextMuted)
			y += 22

			id := fmt.Sprintf("mr%d", compIdx)
			drawTextEx(editorFont, "Metallic", indent, y+4, 15, colorTextMuted)
			comp.Metallic = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".met", comp.Metallic)
			y += fieldH + 2

			drawTextEx(editorFont, "Roughness", indent, y+4, 15, colorTextMuted)
			comp.Roughness = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".rough", comp.Roughness)
			y += fieldH + 2

			drawTextEx(editorFont, "Emissive", indent, y+4, 15, colorTextMuted)
			comp.Emissive = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".emit", comp.Emissive)
			y += fieldH + 4
		}

		// Flip Normals button for GLTF models
		if comp.FilePath != "" {
			btnW := int32(100)
			btnH := int32(22)
			btnX := indent
			btnY := y
			mousePos := rl.GetMousePosition()
			btnHovered := mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
				mousePos.Y >= float32(btnY) && mousePos.Y <= float32(btnY+btnH)
			btnColor := colorBgElement
			if btnHovered {
				btnColor = colorBgHover
			}
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(btnX), Y: float32(btnY), Width: float32(btnW), Height: float32(btnH)}, 0.3, 4, btnColor)
			drawTextEx(editorFont, "Flip Normals", btnX+8, btnY+4, 14, colorTextSecondary)

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
		drawTextEx(editorFont, "Size", indent, y+4, 15, colorTextMuted)
		id := fmt.Sprintf("box%d.size", compIdx)
		comp.Size.X = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".x", comp.Size.X)
		comp.Size.Y = e.drawFloatField(indent+labelW+fieldW+2, y, fieldW, fieldH, id+".y", comp.Size.Y)
		comp.Size.Z = e.drawFloatField(indent+labelW+2*(fieldW+2), y, fieldW, fieldH, id+".z", comp.Size.Z)
		y += fieldH + 4

		// Offset
		drawTextEx(editorFont, "Offset", indent, y+4, 15, colorTextMuted)
		id = fmt.Sprintf("box%d.off", compIdx)
		comp.Offset.X = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".x", comp.Offset.X)
		comp.Offset.Y = e.drawFloatField(indent+labelW+fieldW+2, y, fieldW, fieldH, id+".y", comp.Offset.Y)
		comp.Offset.Z = e.drawFloatField(indent+labelW+2*(fieldW+2), y, fieldW, fieldH, id+".z", comp.Offset.Z)
		y += fieldH + 6

	case *components.SphereCollider:
		drawTextEx(editorFont, "Radius", indent, y+4, 15, colorTextMuted)
		id := fmt.Sprintf("sphere%d.rad", compIdx)
		comp.Radius = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id, comp.Radius)
		y += fieldH + 6

	case *components.Rigidbody:
		// Mass
		drawTextEx(editorFont, "Mass", indent, y+4, 15, colorTextMuted)
		comp.Mass = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("rb%d.mass", compIdx), comp.Mass)
		y += fieldH + 2

		// Bounciness
		drawTextEx(editorFont, "Bounce", indent, y+4, 15, colorTextMuted)
		comp.Bounciness = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("rb%d.bounce", compIdx), comp.Bounciness)
		y += fieldH + 2

		// Friction
		drawTextEx(editorFont, "Friction", indent, y+4, 15, colorTextMuted)
		comp.Friction = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("rb%d.friction", compIdx), comp.Friction)
		y += fieldH + 4

		// Checkboxes for booleans
		gravityBounds := rl.Rectangle{X: float32(indent), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		comp.UseGravity = gui.CheckBox(gravityBounds, "Gravity", comp.UseGravity)

		kinematicBounds := rl.Rectangle{X: float32(indent + 110), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		comp.IsKinematic = gui.CheckBox(kinematicBounds, "Kinematic", comp.IsKinematic)
		y += fieldH + 6

	case *components.DirectionalLight:
		// Direction
		drawTextEx(editorFont, "Dir", indent, y+4, 15, colorTextMuted)
		id := fmt.Sprintf("light%d.dir", compIdx)
		comp.Direction.X = e.drawFloatField(indent+labelW, y, fieldW, fieldH, id+".x", comp.Direction.X)
		comp.Direction.Y = e.drawFloatField(indent+labelW+fieldW+2, y, fieldW, fieldH, id+".y", comp.Direction.Y)
		comp.Direction.Z = e.drawFloatField(indent+labelW+2*(fieldW+2), y, fieldW, fieldH, id+".z", comp.Direction.Z)
		y += fieldH + 4

		// Intensity slider
		drawTextEx(editorFont, "Intensity", indent, y+4, 15, colorTextMuted)
		sliderBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldW * 2), Height: float32(fieldH)}
		comp.Intensity = gui.Slider(sliderBounds, "", fmt.Sprintf("%.1f", comp.Intensity), comp.Intensity, 0, 2)
		y += fieldH + 6

	case *components.PointLight:
		id := fmt.Sprintf("pointlight%d", compIdx)

		// Color picker (simplified - show RGB sliders)
		drawTextEx(editorFont, "Color", indent, y+4, 15, colorTextMuted)
		colorPreview := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		rl.DrawRectangleRec(colorPreview, comp.Color)
		rl.DrawRectangleLinesEx(colorPreview, 1, rl.Gray)
		// R/G/B fields
		comp.Color.R = uint8(e.drawFloatField(indent+labelW+fieldH+4, y, fieldW-10, fieldH, id+".r", float32(comp.Color.R)))
		comp.Color.G = uint8(e.drawFloatField(indent+labelW+fieldH+4+fieldW-8, y, fieldW-10, fieldH, id+".g", float32(comp.Color.G)))
		comp.Color.B = uint8(e.drawFloatField(indent+labelW+fieldH+4+2*(fieldW-8), y, fieldW-10, fieldH, id+".b", float32(comp.Color.B)))
		y += fieldH + 4

		// Intensity slider
		drawTextEx(editorFont, "Intensity", indent, y+4, 15, colorTextMuted)
		intensityBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldW * 2), Height: float32(fieldH)}
		comp.Intensity = gui.Slider(intensityBounds, "", fmt.Sprintf("%.1f", comp.Intensity), comp.Intensity, 0, 5)
		y += fieldH + 4

		// Radius slider
		drawTextEx(editorFont, "Radius", indent, y+4, 15, colorTextMuted)
		radiusBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldW * 2), Height: float32(fieldH)}
		comp.Radius = gui.Slider(radiusBounds, "", fmt.Sprintf("%.1f", comp.Radius), comp.Radius, 1, 50)
		y += fieldH + 6

	default:
		// For scripts and unknown components, try to get script name
		if name, props, ok := engine.SerializeScript(c); ok {
			drawTextEx(editorFont, fmt.Sprintf("Script: %s", name), indent, y, 15, colorAccentLight)
			y += 20

			// Sort property keys for consistent display
			keys := make([]string, 0, len(props))
			for k := range props {
				keys = append(keys, k)
			}
			for i := 0; i < len(keys)-1; i++ {
				for j := i + 1; j < len(keys); j++ {
					if keys[i] > keys[j] {
						keys[i], keys[j] = keys[j], keys[i]
					}
				}
			}

			// Draw editable fields for each property
			for _, k := range keys {
				v := props[k]
				fieldID := fmt.Sprintf("script%d.%s", compIdx, k)

				switch val := v.(type) {
				case float32:
					drawTextEx(editorFont, k, indent, y+4, 14, colorTextMuted)
					newVal := e.drawFloatField(indent+labelW, y, fieldW, fieldH, fieldID, val)
					if newVal != val {
						engine.ApplyScriptProperty(c, k, float64(newVal))
					}
					y += fieldH + 4

				case float64:
					drawTextEx(editorFont, k, indent, y+4, 14, colorTextMuted)
					newVal := e.drawFloatField(indent+labelW, y, fieldW, fieldH, fieldID, float32(val))
					if float64(newVal) != val {
						engine.ApplyScriptProperty(c, k, float64(newVal))
					}
					y += fieldH + 4

				case int:
					drawTextEx(editorFont, k, indent, y+4, 14, colorTextMuted)
					newVal := e.drawFloatField(indent+labelW, y, fieldW, fieldH, fieldID, float32(val))
					if int(newVal) != val {
						engine.ApplyScriptProperty(c, k, float64(newVal))
					}
					y += fieldH + 4

				case bool:
					drawTextEx(editorFont, k, indent, y+4, 14, colorTextMuted)
					checkBounds := rl.Rectangle{X: float32(indent + labelW), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
					newVal := gui.CheckBox(checkBounds, "", val)
					if newVal != val {
						engine.ApplyScriptProperty(c, k, newVal)
					}
					y += fieldH + 4

				case string:
					drawTextEx(editorFont, k, indent, y+4, 14, colorTextMuted)
					drawTextEx(editorFont, val, indent+labelW, y+4, 14, colorTextSecondary)
					y += fieldH + 4

				default:
					// Display non-editable types as text
					drawTextEx(editorFont, fmt.Sprintf("%s: %v", k, v), indent, y, 14, colorTextMuted)
					y += 16
				}
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
	itemH := int32(26)
	maxVisibleItems := int32(12) // Max items visible before scrolling

	// Get registered scripts
	scripts := engine.GetRegisteredScripts()

	// Total items: built-in components + separator + scripts
	totalItems := int32(len(editorComponentTypes))
	if len(scripts) > 0 {
		totalItems += 1 + int32(len(scripts)) // +1 for separator
	}

	contentH := totalItems * itemH
	menuH := contentH
	needsScroll := totalItems > maxVisibleItems
	if needsScroll {
		menuH = maxVisibleItems * itemH
	}

	mousePos := rl.GetMousePosition()
	mouseInMenu := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
		mousePos.Y >= float32(y) && mousePos.Y <= float32(y+menuH)

	// Handle scroll wheel when hovering menu
	if mouseInMenu {
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			e.addComponentScroll -= int32(wheel * 26 * 2) // Scroll 2 items per wheel tick
			// Clamp scroll
			maxScroll := contentH - menuH
			if maxScroll < 0 {
				maxScroll = 0
			}
			if e.addComponentScroll < 0 {
				e.addComponentScroll = 0
			}
			if e.addComponentScroll > maxScroll {
				e.addComponentScroll = maxScroll
			}
		}
	}

	// Draw menu background - rounded with border
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(w), Height: float32(menuH)}, 0.1, 4, colorBgPanel)
	rl.DrawRectangleRoundedLinesEx(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(w), Height: float32(menuH)}, 0.1, 4, 1, colorBorder)

	// Begin scissor mode to clip items outside menu area
	rl.BeginScissorMode(x, y, w, menuH)

	itemIndex := int32(0)

	// Built-in components
	for _, compType := range editorComponentTypes {
		itemY := y + itemIndex*itemH - e.addComponentScroll

		// Skip if completely outside visible area
		if itemY+itemH < y || itemY > y+menuH {
			itemIndex++
			continue
		}

		hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
			mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH) &&
			mousePos.Y >= float32(y) && mousePos.Y < float32(y+menuH)

		if hovered {
			rl.DrawRectangle(x+2, itemY, w-4, itemH, colorAccent)
		}

		txtColor := colorTextSecondary
		if hovered {
			txtColor = colorTextPrimary
		}
		drawTextEx(editorFont, compType.Name, x+12, itemY+5, 16, txtColor)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.addComponent(compType.Name)
			e.showAddComponentMenu = false
		}
		itemIndex++
	}

	// Scripts section
	if len(scripts) > 0 {
		// Separator with "Scripts" label
		sepY := y + itemIndex*itemH - e.addComponentScroll
		if sepY+itemH >= y && sepY <= y+menuH {
			rl.DrawRectangle(x+10, sepY+itemH/2, w-20, 1, colorBorder)
			drawTextEx(editorFont, "Scripts", x+12, sepY+5, 14, colorTextMuted)
		}
		itemIndex++

		// Script items
		for _, scriptName := range scripts {
			itemY := y + itemIndex*itemH - e.addComponentScroll

			// Skip if completely outside visible area
			if itemY+itemH < y || itemY > y+menuH {
				itemIndex++
				continue
			}

			hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
				mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH) &&
				mousePos.Y >= float32(y) && mousePos.Y < float32(y+menuH)

			if hovered {
				rl.DrawRectangle(x+2, itemY, w-4, itemH, colorAccent)
			}

			txtColor := colorAccentLight // Scripts in accent color to differentiate
			if hovered {
				txtColor = colorTextPrimary
			}
			drawTextEx(editorFont, scriptName, x+12, itemY+5, 16, txtColor)

			if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				e.addScript(scriptName)
				e.showAddComponentMenu = false
			}
			itemIndex++
		}
	}

	rl.EndScissorMode()

	// Draw scroll indicator if needed
	if needsScroll {
		scrollBarW := int32(4)
		scrollBarX := x + w - scrollBarW - 4
		scrollTrackH := menuH - 8
		maxScroll := contentH - menuH
		scrollThumbH := int32(float32(scrollTrackH) * float32(menuH) / float32(contentH))
		if scrollThumbH < 20 {
			scrollThumbH = 20
		}
		scrollThumbY := y + 4 + int32(float32(scrollTrackH-scrollThumbH)*float32(e.addComponentScroll)/float32(maxScroll))

		// Draw scroll track
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(scrollBarX), Y: float32(y + 4), Width: float32(scrollBarW), Height: float32(scrollTrackH)}, 0.5, 4, colorBgDark)
		// Draw scroll thumb
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(scrollBarX), Y: float32(scrollThumbY), Width: float32(scrollBarW), Height: float32(scrollThumbH)}, 0.5, 4, colorAccent)
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

func (e *Editor) addScript(scriptName string) {
	if e.Selected == nil {
		return
	}

	newComp := engine.CreateScript(scriptName, map[string]any{})
	if newComp != nil {
		e.Selected.AddComponent(newComp)

		// Re-register with physics world to update categorization
		e.updatePhysicsRegistration(e.Selected)

		e.saveMsg = fmt.Sprintf("Added %s", scriptName)
		e.saveMsgTime = rl.GetTime()
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

// drawAssetBrowser draws the asset browser panel at the bottom of the screen
func (e *Editor) drawAssetBrowser() {
	panelH := int32(150)
	panelY := int32(rl.GetScreenHeight()) - panelH
	panelX := e.hierarchyWidth                                            // Start after hierarchy
	panelW := int32(rl.GetScreenWidth()) - e.hierarchyWidth - e.inspectorWidth // Between hierarchy and inspector

	// Reserve space for material editor on the right when a material is selected
	contentW := panelW
	if e.selectedMaterial != nil {
		contentW = panelW - 180
	}

	// Background with border
	rl.DrawRectangle(panelX, panelY, panelW, panelH, colorBgPanel)
	rl.DrawRectangle(panelX, panelY, panelW, 1, colorBorder)

	mousePos := rl.GetMousePosition()

	// Header with back button and path
	headerY := panelY + 6

	// Back button (only show if not at root)
	backBtnX := panelX + 10
	backBtnW := int32(26)
	backBtnH := int32(20)
	canGoBack := e.currentAssetPath != "assets" && e.currentAssetPath != ""

	if canGoBack {
		backHovered := mousePos.X >= float32(backBtnX) && mousePos.X <= float32(backBtnX+backBtnW) &&
			mousePos.Y >= float32(headerY) && mousePos.Y <= float32(headerY+backBtnH)

		backColor := colorBgElement
		if backHovered {
			backColor = colorAccent
		}
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(backBtnX), Y: float32(headerY), Width: float32(backBtnW), Height: float32(backBtnH)}, 0.3, 4, backColor)
		drawTextEx(editorFontBold, "<", backBtnX+8, headerY+3, 16, colorTextPrimary)

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
	pathX := backBtnX + backBtnW + 10
	if !canGoBack {
		pathX = panelX + 12
	}
	drawTextEx(editorFont, e.currentAssetPath+"/", pathX, headerY+3, 15, colorTextMuted)

	// Refresh button
	refreshBtnX := panelX + contentW - 75
	refreshBtnY := headerY
	refreshBtnW := int32(65)
	refreshBtnH := int32(20)

	refreshHovered := mousePos.X >= float32(refreshBtnX) && mousePos.X <= float32(refreshBtnX+refreshBtnW) &&
		mousePos.Y >= float32(refreshBtnY) && mousePos.Y <= float32(refreshBtnY+refreshBtnH)

	refreshColor := colorBgElement
	if refreshHovered {
		refreshColor = colorAccent
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(refreshBtnX), Y: float32(refreshBtnY), Width: float32(refreshBtnW), Height: float32(refreshBtnH)}, 0.3, 4, refreshColor)
	drawTextEx(editorFont, "Refresh", refreshBtnX+10, refreshBtnY+3, 14, colorTextSecondary)

	if refreshHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.scanAssets()
	}

	// Asset grid - larger items for better icons
	itemW := int32(80)
	itemH := int32(85)
	startX := panelX + 10
	startY := panelY + 30
	cols := (contentW - 20) / (itemW + 8)
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

		x := startX + col*(itemW+8)
		y := startY + row*(itemH+8) - e.assetBrowserScroll

		// Skip if off screen
		if y+itemH < panelY+24 || y > panelY+panelH {
			continue
		}

		// Item background - rounded
		itemHovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+itemW) &&
			mousePos.Y >= float32(y) && mousePos.Y <= float32(y+itemH)

		isSelected := asset.Path == e.selectedMaterialPath

		bgColor := colorBgElement
		if isSelected {
			bgColor = colorAccent
		} else if itemHovered {
			bgColor = colorBgHover
		}
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(itemW), Height: float32(itemH)}, 0.15, 4, bgColor)

		// Draw icon based on type - larger, smoother icons
		iconSize := int32(42)
		iconX := x + (itemW-iconSize)/2
		iconY := y + 8

		switch asset.Type {
		case "folder":
			// Folder icon - rounded with tab
			folderColor := rl.NewColor(220, 180, 80, 255)
			folderDark := rl.NewColor(180, 140, 50, 255)
			// Tab
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY), Width: float32(iconSize/2 + 4), Height: 8}, 0.4, 4, folderColor)
			// Body
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY + 6), Width: float32(iconSize), Height: float32(iconSize - 10)}, 0.2, 4, folderColor)
			// Shadow line
			rl.DrawRectangle(iconX+2, iconY+10, iconSize-4, 2, folderDark)
		case "material":
			// Material icon - gradient sphere effect
			centerX := iconX + iconSize/2
			centerY := iconY + iconSize/2
			radius := float32(iconSize) / 2 - 2
			// Outer ring
			rl.DrawCircle(centerX, centerY, radius, colorAccent)
			// Inner highlight
			rl.DrawCircle(centerX-4, centerY-4, radius*0.6, colorAccentLight)
			// Shine dot
			rl.DrawCircle(centerX-6, centerY-6, 4, rl.NewColor(255, 255, 255, 180))
		case "model":
			// Model icon - 3D cube with depth
			cubeColor := rl.NewColor(120, 200, 140, 255)
			cubeDark := rl.NewColor(80, 160, 100, 255)
			cubeLight := rl.NewColor(160, 230, 180, 255)
			// Main face
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 6), Y: float32(iconY + 6), Width: float32(iconSize - 12), Height: float32(iconSize - 12)}, 0.15, 4, cubeColor)
			// Top edge highlight
			rl.DrawRectangle(iconX+6, iconY+6, iconSize-12, 4, cubeLight)
			// Right edge shadow
			rl.DrawRectangle(iconX+iconSize-10, iconY+10, 4, iconSize-16, cubeDark)
			// "3D" text
			drawTextEx(editorFontBold, "3D", iconX+14, iconY+14, 16, rl.White)
		case "texture":
			// Texture icon - rounded checkerboard
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY), Width: float32(iconSize), Height: float32(iconSize)}, 0.15, 4, rl.NewColor(60, 60, 70, 255))
			half := iconSize / 2
			// Checkerboard pattern inside
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 4), Y: float32(iconY + 4), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(220, 220, 220, 255))
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + half + 2), Y: float32(iconY + half + 2), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(220, 220, 220, 255))
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + half + 2), Y: float32(iconY + 4), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(120, 120, 130, 255))
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 4), Y: float32(iconY + half + 2), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(120, 120, 130, 255))
		default:
			// Generic file icon - document style
			docColor := rl.NewColor(140, 140, 160, 255)
			// Main body
			rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 6), Y: float32(iconY), Width: float32(iconSize - 12), Height: float32(iconSize)}, 0.15, 4, docColor)
			// Corner fold
			rl.DrawTriangle(
				rl.NewVector2(float32(iconX+iconSize-6), float32(iconY)),
				rl.NewVector2(float32(iconX+iconSize-6), float32(iconY+10)),
				rl.NewVector2(float32(iconX+iconSize-16), float32(iconY)),
				rl.NewColor(100, 100, 120, 255),
			)
			// Lines to represent text
			rl.DrawRectangle(iconX+12, iconY+16, iconSize-24, 3, rl.NewColor(100, 100, 120, 255))
			rl.DrawRectangle(iconX+12, iconY+22, iconSize-28, 3, rl.NewColor(100, 100, 120, 255))
			rl.DrawRectangle(iconX+12, iconY+28, iconSize-24, 3, rl.NewColor(100, 100, 120, 255))
		}

		// Name (truncated) - centered below icon
		name := asset.Name
		if len(name) > 10 {
			name = name[:9] + "…"
		}
		textW := rl.MeasureText(name, 13)
		drawTextEx(editorFont, name, x+(itemW-textW)/2, y+itemH-18, 13, colorTextSecondary)

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
	maxScroll := rows*(itemH+8) - (panelH - 30)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.assetBrowserScroll > maxScroll {
		e.assetBrowserScroll = maxScroll
	}

	// Empty state
	if len(e.assetFiles) == 0 {
		drawTextEx(editorFont, "Empty folder", panelX+20, panelY+60, 16, colorTextMuted)
	}

	// Draw material editor panel on the right
	if e.selectedMaterial != nil {
		e.drawMaterialEditor(panelX+contentW, panelY, panelW-contentW, panelH)
	}
}

// drawMaterialEditor draws the material properties editor in the asset browser
func (e *Editor) drawMaterialEditor(x, y, w, h int32) {
	// Background with border
	rl.DrawRectangle(x, y, w, h, colorBgPanel)
	rl.DrawRectangle(x, y, 1, h, colorBorder)

	// Header
	name := filepath.Base(e.selectedMaterialPath)
	drawTextEx(editorFontBold, name, x+10, y+6, 14, colorAccentLight)

	// Close button - rounded
	closeBtnX := x + w - 22
	closeBtnY := y + 5
	closeBtnSize := int32(16)
	mousePos := rl.GetMousePosition()
	closeHovered := mousePos.X >= float32(closeBtnX) && mousePos.X <= float32(closeBtnX+closeBtnSize) &&
		mousePos.Y >= float32(closeBtnY) && mousePos.Y <= float32(closeBtnY+closeBtnSize)

	closeColor := rl.NewColor(80, 50, 50, 200)
	if closeHovered {
		closeColor = rl.NewColor(180, 60, 60, 230)
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(closeBtnX), Y: float32(closeBtnY), Width: float32(closeBtnSize), Height: float32(closeBtnSize)}, 0.3, 4, closeColor)
	drawTextEx(editorFontBold, "x", closeBtnX+4, closeBtnY+1, 12, colorTextPrimary)

	if closeHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.selectedMaterial = nil
		e.selectedMaterialPath = ""
		return
	}

	// Properties
	propY := y + 26
	labelW := int32(65)
	fieldW := w - labelW - 18
	fieldH := int32(18)
	indent := x + 10

	mat := e.selectedMaterial
	oldMet := mat.Metallic
	oldRough := mat.Roughness
	oldEmit := mat.Emissive

	// Material name (read-only for now)
	drawTextEx(editorFont, "Name:", indent, propY+2, 13, colorTextMuted)
	drawTextEx(editorFont, mat.Name, indent+labelW, propY+2, 13, colorTextSecondary)
	propY += fieldH + 4

	// Color (read-only display)
	drawTextEx(editorFont, "Color:", indent, propY+2, 13, colorTextMuted)
	colorName := assets.LookupColorName(mat.Color)
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(indent + labelW), Y: float32(propY), Width: float32(fieldH), Height: float32(fieldH)}, 0.2, 4, mat.Color)
	drawTextEx(editorFont, colorName, indent+labelW+fieldH+6, propY+2, 13, colorTextSecondary)
	propY += fieldH + 4

	// Metallic
	drawTextEx(editorFont, "Metallic:", indent, propY+3, 13, colorTextMuted)
	mat.Metallic = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.met", mat.Metallic)
	propY += fieldH + 4

	// Roughness
	drawTextEx(editorFont, "Rough:", indent, propY+3, 13, colorTextMuted)
	mat.Roughness = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.rough", mat.Roughness)
	propY += fieldH + 4

	// Emissive
	drawTextEx(editorFont, "Emissive:", indent, propY+3, 13, colorTextMuted)
	mat.Emissive = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.emit", mat.Emissive)
	propY += fieldH + 4

	// Albedo texture path (editable)
	drawTextEx(editorFont, "Albedo:", indent, propY+3, 13, colorTextMuted)
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

		// Skip hidden/system files
		if strings.HasPrefix(name, ".") {
			continue
		}

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
