//go:build !game

package game

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UI Edit Mode - 2D orthographic view for editing UI elements (like Unity's 2D mode)

// UIEditState holds state for the UI editing mode
type UIEditState struct {
	Active          bool
	SelectedElement *engine.GameObject
	Canvas          *engine.GameObject // The canvas being edited

	// View transform (pan and zoom)
	ViewOffset rl.Vector2 // Pan offset in screen pixels
	ViewZoom   float32    // Zoom level (1.0 = 100%)

	// Panning state
	Panning        bool
	PanStartMouse  rl.Vector2
	PanStartOffset rl.Vector2

	// Dragging state (moving elements)
	Dragging       bool
	DragStartMouse rl.Vector2
	DragStartPos   rl.Vector2 // AnchoredPosition when drag started

	// Resize state
	Resizing         bool
	ResizeHandle     int // 0-7 for corners and edges
	ResizeStartMouse rl.Vector2
	ResizeStartSize  rl.Vector2

	// Hover state
	HoveredElement *engine.GameObject
	HoveredHandle  int // -1 = none, 0-7 = resize handles
}

// Resize handle positions
const (
	HandleTopLeft = iota
	HandleTop
	HandleTopRight
	HandleRight
	HandleBottomRight
	HandleBottom
	HandleBottomLeft
	HandleLeft
)

// ToggleUIEditMode switches between 3D editor and UI edit mode
func (e *Editor) ToggleUIEditMode() {
	if e.uiEditState == nil {
		e.uiEditState = &UIEditState{
			HoveredHandle: -1,
			ViewZoom:      1.0,
		}
	}

	e.uiEditState.Active = !e.uiEditState.Active

	if e.uiEditState.Active {
		// Find the first canvas in the scene (recursively)
		e.uiEditState.Canvas = e.findCanvasRecursive(e.world.Scene.GameObjects)
		// Reset view to default
		e.uiEditState.ViewOffset = rl.Vector2{X: 0, Y: 0}
		e.uiEditState.ViewZoom = 1.0
	}
}

// findCanvasRecursive searches for a UICanvas component in the hierarchy
func (e *Editor) findCanvasRecursive(objects []*engine.GameObject) *engine.GameObject {
	for _, obj := range objects {
		if engine.GetComponent[*components.UICanvas](obj) != nil {
			return obj
		}
		// Search children
		if found := e.findCanvasRecursive(obj.Children); found != nil {
			return found
		}
	}
	return nil
}

// UpdateUIEditMode handles input in UI edit mode
func (e *Editor) UpdateUIEditMode() {
	if e.uiEditState == nil || !e.uiEditState.Active {
		return
	}

	mousePos := rl.GetMousePosition()

	// ESC to exit UI edit mode
	if rl.IsKeyPressed(rl.KeyEscape) {
		e.uiEditState.Active = false
		return
	}

	// Handle panning (middle mouse or right mouse drag)
	if e.uiEditState.Panning {
		// Stop panning when neither middle nor right mouse is held
		if !rl.IsMouseButtonDown(rl.MouseMiddleButton) && !rl.IsMouseButtonDown(rl.MouseRightButton) {
			e.uiEditState.Panning = false
		} else {
			delta := rl.Vector2{
				X: mousePos.X - e.uiEditState.PanStartMouse.X,
				Y: mousePos.Y - e.uiEditState.PanStartMouse.Y,
			}
			e.uiEditState.ViewOffset = rl.Vector2{
				X: e.uiEditState.PanStartOffset.X + delta.X,
				Y: e.uiEditState.PanStartOffset.Y + delta.Y,
			}
		}
		return
	}

	// Start panning with middle or right mouse
	if rl.IsMouseButtonPressed(rl.MouseMiddleButton) || rl.IsMouseButtonPressed(rl.MouseRightButton) {
		e.uiEditState.Panning = true
		e.uiEditState.PanStartMouse = mousePos
		e.uiEditState.PanStartOffset = e.uiEditState.ViewOffset
		return
	}

	// Zoom with scroll wheel (only when not in panels)
	if !e.mouseInPanel() {
		scroll := rl.GetMouseWheelMove()
		if scroll != 0 {
			// Zoom towards mouse position
			oldZoom := e.uiEditState.ViewZoom
			zoomSpeed := float32(0.1)
			e.uiEditState.ViewZoom += scroll * zoomSpeed * e.uiEditState.ViewZoom

			// Clamp zoom
			if e.uiEditState.ViewZoom < 0.1 {
				e.uiEditState.ViewZoom = 0.1
			}
			if e.uiEditState.ViewZoom > 5.0 {
				e.uiEditState.ViewZoom = 5.0
			}

			// Adjust offset to zoom towards mouse position
			zoomDelta := e.uiEditState.ViewZoom / oldZoom
			screenCenter := rl.Vector2{
				X: float32(rl.GetScreenWidth()) / 2,
				Y: float32(rl.GetScreenHeight()) / 2,
			}
			mouseFromCenter := rl.Vector2{
				X: mousePos.X - screenCenter.X,
				Y: mousePos.Y - screenCenter.Y,
			}
			e.uiEditState.ViewOffset.X = (e.uiEditState.ViewOffset.X-mouseFromCenter.X)*zoomDelta + mouseFromCenter.X
			e.uiEditState.ViewOffset.Y = (e.uiEditState.ViewOffset.Y-mouseFromCenter.Y)*zoomDelta + mouseFromCenter.Y
		}
	}

	// Skip element interaction if mouse is in editor panels
	if e.mouseInPanel() {
		return
	}

	// Convert mouse position to canvas space for element picking
	canvasMousePos := e.screenToCanvas(mousePos)

	// Handle dragging
	if e.uiEditState.Dragging {
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			e.uiEditState.Dragging = false
		} else {
			e.updateUIElementDrag(canvasMousePos)
		}
		return
	}

	// Handle resizing
	if e.uiEditState.Resizing {
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			e.uiEditState.Resizing = false
		} else {
			e.updateUIElementResize(canvasMousePos)
		}
		return
	}

	// Update hover state
	e.uiEditState.HoveredElement = nil
	e.uiEditState.HoveredHandle = -1

	// Check resize handles first (if element selected)
	if e.uiEditState.SelectedElement != nil {
		handle := e.pickResizeHandle(canvasMousePos)
		if handle >= 0 {
			e.uiEditState.HoveredHandle = handle
		}
	}

	// Then check element hover
	if e.uiEditState.HoveredHandle < 0 && e.uiEditState.Canvas != nil {
		e.uiEditState.HoveredElement = e.pickUIElement(canvasMousePos, e.uiEditState.Canvas)
	}

	// Left click
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		// Check resize handles first
		if e.uiEditState.SelectedElement != nil && e.uiEditState.HoveredHandle >= 0 {
			e.startUIElementResize(canvasMousePos, e.uiEditState.HoveredHandle)
			return
		}

		// Then check element selection/drag
		if e.uiEditState.HoveredElement != nil {
			if e.uiEditState.HoveredElement == e.uiEditState.SelectedElement {
				// Already selected - start drag
				e.startUIElementDrag(canvasMousePos)
			} else {
				// Select new element
				e.uiEditState.SelectedElement = e.uiEditState.HoveredElement
				e.Selected = e.uiEditState.HoveredElement // Sync with main editor selection
			}
		} else {
			// Clicked empty space - deselect
			e.uiEditState.SelectedElement = nil
		}
	}
}

// screenToCanvas converts screen coordinates to canvas coordinates (accounting for pan/zoom)
func (e *Editor) screenToCanvas(screenPos rl.Vector2) rl.Vector2 {
	screenCenter := rl.Vector2{
		X: float32(rl.GetScreenWidth()) / 2,
		Y: float32(rl.GetScreenHeight()) / 2,
	}
	// Remove offset and zoom
	return rl.Vector2{
		X: (screenPos.X - screenCenter.X - e.uiEditState.ViewOffset.X) / e.uiEditState.ViewZoom + screenCenter.X,
		Y: (screenPos.Y - screenCenter.Y - e.uiEditState.ViewOffset.Y) / e.uiEditState.ViewZoom + screenCenter.Y,
	}
}

// canvasToScreen converts canvas coordinates to screen coordinates (accounting for pan/zoom)
func (e *Editor) canvasToScreen(canvasPos rl.Vector2) rl.Vector2 {
	screenCenter := rl.Vector2{
		X: float32(rl.GetScreenWidth()) / 2,
		Y: float32(rl.GetScreenHeight()) / 2,
	}
	// Apply zoom and offset
	return rl.Vector2{
		X: (canvasPos.X-screenCenter.X)*e.uiEditState.ViewZoom + screenCenter.X + e.uiEditState.ViewOffset.X,
		Y: (canvasPos.Y-screenCenter.Y)*e.uiEditState.ViewZoom + screenCenter.Y + e.uiEditState.ViewOffset.Y,
	}
}

// canvasRectToScreen converts a canvas-space rectangle to screen-space
func (e *Editor) canvasRectToScreen(rect rl.Rectangle) rl.Rectangle {
	topLeft := e.canvasToScreen(rl.Vector2{X: rect.X, Y: rect.Y})
	return rl.Rectangle{
		X:      topLeft.X,
		Y:      topLeft.Y,
		Width:  rect.Width * e.uiEditState.ViewZoom,
		Height: rect.Height * e.uiEditState.ViewZoom,
	}
}

// pickUIElement finds the UI element under the mouse position
func (e *Editor) pickUIElement(mousePos rl.Vector2, parent *engine.GameObject) *engine.GameObject {
	if parent == nil || !parent.Active {
		return nil
	}

	screenRect := rl.Rectangle{
		X: 0, Y: 0,
		Width:  float32(rl.GetScreenWidth()),
		Height: float32(rl.GetScreenHeight()),
	}

	return e.pickUIElementRecursive(mousePos, parent, screenRect)
}

func (e *Editor) pickUIElementRecursive(mousePos rl.Vector2, obj *engine.GameObject, parentRect rl.Rectangle) *engine.GameObject {
	if obj == nil || !obj.Active {
		return nil
	}

	rt := engine.GetComponent[*components.RectTransform](obj)
	currentRect := parentRect

	if rt != nil {
		rt.CalculateRect(parentRect)
		currentRect = rt.GetScreenRect()
	}

	// Check children first (they're drawn on top)
	for i := len(obj.Children) - 1; i >= 0; i-- {
		child := obj.Children[i]
		hit := e.pickUIElementRecursive(mousePos, child, currentRect)
		if hit != nil {
			return hit
		}
	}

	// Then check this element (only if it has a RectTransform)
	if rt != nil && rl.CheckCollisionPointRec(mousePos, currentRect) {
		// Skip the canvas itself
		if engine.GetComponent[*components.UICanvas](obj) == nil {
			return obj
		}
	}

	return nil
}

// pickResizeHandle returns which resize handle is under the mouse (-1 if none)
func (e *Editor) pickResizeHandle(mousePos rl.Vector2) int {
	if e.uiEditState.SelectedElement == nil {
		return -1
	}

	rt := engine.GetComponent[*components.RectTransform](e.uiEditState.SelectedElement)
	if rt == nil {
		return -1
	}

	rect := rt.GetScreenRect()
	handleSize := float32(10)

	// Define handle positions
	handles := []rl.Vector2{
		{X: rect.X, Y: rect.Y},                              // TopLeft
		{X: rect.X + rect.Width/2, Y: rect.Y},               // Top
		{X: rect.X + rect.Width, Y: rect.Y},                 // TopRight
		{X: rect.X + rect.Width, Y: rect.Y + rect.Height/2}, // Right
		{X: rect.X + rect.Width, Y: rect.Y + rect.Height},   // BottomRight
		{X: rect.X + rect.Width/2, Y: rect.Y + rect.Height}, // Bottom
		{X: rect.X, Y: rect.Y + rect.Height},                // BottomLeft
		{X: rect.X, Y: rect.Y + rect.Height/2},              // Left
	}

	for i, pos := range handles {
		handleRect := rl.Rectangle{
			X:      pos.X - handleSize/2,
			Y:      pos.Y - handleSize/2,
			Width:  handleSize,
			Height: handleSize,
		}
		if rl.CheckCollisionPointRec(mousePos, handleRect) {
			return i
		}
	}

	return -1
}

// startUIElementDrag begins dragging the selected element
func (e *Editor) startUIElementDrag(mousePos rl.Vector2) {
	rt := engine.GetComponent[*components.RectTransform](e.uiEditState.SelectedElement)
	if rt == nil {
		return
	}

	e.uiEditState.Dragging = true
	e.uiEditState.DragStartMouse = mousePos
	e.uiEditState.DragStartPos = rt.AnchoredPosition
}

// updateUIElementDrag updates position while dragging
func (e *Editor) updateUIElementDrag(mousePos rl.Vector2) {
	rt := engine.GetComponent[*components.RectTransform](e.uiEditState.SelectedElement)
	if rt == nil {
		return
	}

	delta := rl.Vector2{
		X: mousePos.X - e.uiEditState.DragStartMouse.X,
		Y: mousePos.Y - e.uiEditState.DragStartMouse.Y,
	}

	rt.AnchoredPosition = rl.Vector2{
		X: e.uiEditState.DragStartPos.X + delta.X,
		Y: e.uiEditState.DragStartPos.Y + delta.Y,
	}
}

// startUIElementResize begins resizing the selected element
func (e *Editor) startUIElementResize(mousePos rl.Vector2, handle int) {
	rt := engine.GetComponent[*components.RectTransform](e.uiEditState.SelectedElement)
	if rt == nil {
		return
	}

	e.uiEditState.Resizing = true
	e.uiEditState.ResizeHandle = handle
	e.uiEditState.ResizeStartMouse = mousePos
	e.uiEditState.ResizeStartSize = rt.SizeDelta
	e.uiEditState.DragStartPos = rt.AnchoredPosition
}

// updateUIElementResize updates size while resizing
func (e *Editor) updateUIElementResize(mousePos rl.Vector2) {
	rt := engine.GetComponent[*components.RectTransform](e.uiEditState.SelectedElement)
	if rt == nil {
		return
	}

	deltaX := mousePos.X - e.uiEditState.ResizeStartMouse.X
	deltaY := mousePos.Y - e.uiEditState.ResizeStartMouse.Y

	newWidth := e.uiEditState.ResizeStartSize.X
	newHeight := e.uiEditState.ResizeStartSize.Y
	newPosX := e.uiEditState.DragStartPos.X
	newPosY := e.uiEditState.DragStartPos.Y

	switch e.uiEditState.ResizeHandle {
	case HandleTopLeft:
		newWidth -= deltaX
		newHeight -= deltaY
		newPosX += deltaX
		newPosY += deltaY
	case HandleTop:
		newHeight -= deltaY
		newPosY += deltaY
	case HandleTopRight:
		newWidth += deltaX
		newHeight -= deltaY
		newPosY += deltaY
	case HandleRight:
		newWidth += deltaX
	case HandleBottomRight:
		newWidth += deltaX
		newHeight += deltaY
	case HandleBottom:
		newHeight += deltaY
	case HandleBottomLeft:
		newWidth -= deltaX
		newHeight += deltaY
		newPosX += deltaX
	case HandleLeft:
		newWidth -= deltaX
		newPosX += deltaX
	}

	// Minimum size
	if newWidth < 10 {
		newWidth = 10
	}
	if newHeight < 10 {
		newHeight = 10
	}

	rt.SizeDelta = rl.Vector2{X: newWidth, Y: newHeight}
	rt.AnchoredPosition = rl.Vector2{X: newPosX, Y: newPosY}
}

// Draw3DForUIMode draws the 2D UI view in the 3D viewport
func (e *Editor) Draw3DForUIMode() {
	if e.uiEditState == nil || !e.uiEditState.Active {
		return
	}

	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())
	zoom := e.uiEditState.ViewZoom

	// Draw dark background
	rl.DrawRectangle(0, 0, int32(screenW), int32(screenH), rl.NewColor(30, 30, 40, 255))

	// Draw grid pattern (transformed by view)
	gridSize := float32(50) * zoom
	gridColor := rl.NewColor(45, 45, 55, 255)

	// Calculate grid offset based on pan
	startX := float32(int(e.uiEditState.ViewOffset.X)%int(gridSize)) - gridSize
	startY := float32(int(e.uiEditState.ViewOffset.Y)%int(gridSize)) - gridSize

	for x := startX; x < screenW+gridSize; x += gridSize {
		rl.DrawLineV(rl.Vector2{X: x, Y: 0}, rl.Vector2{X: x, Y: screenH}, gridColor)
	}
	for y := startY; y < screenH+gridSize; y += gridSize {
		rl.DrawLineV(rl.Vector2{X: 0, Y: y}, rl.Vector2{X: screenW, Y: y}, gridColor)
	}

	// Draw canvas boundary (the actual screen-sized canvas, transformed)
	canvasRect := rl.Rectangle{X: 0, Y: 0, Width: screenW, Height: screenH}
	screenCanvasRect := e.canvasRectToScreen(canvasRect)
	rl.DrawRectangleRec(screenCanvasRect, rl.NewColor(40, 40, 50, 255))
	rl.DrawRectangleLinesEx(screenCanvasRect, 2, rl.NewColor(80, 80, 100, 255))

	// Draw all UI elements (transformed)
	if e.uiEditState.Canvas != nil {
		e.drawUIElementsForEdit(e.uiEditState.Canvas, canvasRect)
	}

	// Draw selection highlight and resize handles (transformed)
	if e.uiEditState.SelectedElement != nil {
		rt := engine.GetComponent[*components.RectTransform](e.uiEditState.SelectedElement)
		if rt != nil {
			rect := rt.GetScreenRect()
			screenRect := e.canvasRectToScreen(rect)

			// Selection outline
			rl.DrawRectangleLinesEx(screenRect, 2, colorAccent)

			// Resize handles
			e.drawResizeHandles(screenRect)
		}
	}

	// Draw hover highlight (transformed)
	if e.uiEditState.HoveredElement != nil && e.uiEditState.HoveredElement != e.uiEditState.SelectedElement {
		rt := engine.GetComponent[*components.RectTransform](e.uiEditState.HoveredElement)
		if rt != nil {
			rect := rt.GetScreenRect()
			screenRect := e.canvasRectToScreen(rect)
			rl.DrawRectangleLinesEx(screenRect, 1, rl.NewColor(150, 150, 200, 180))
		}
	}
}

// DrawUIEditModeOverlay draws the UI edit mode indicator on top of editor UI
func (e *Editor) DrawUIEditModeOverlay() {
	if e.uiEditState == nil || !e.uiEditState.Active {
		return
	}

	// Draw "2D" indicator in the top bar area
	screenW := int32(rl.GetScreenWidth())

	// Mode badge
	badgeX := screenW/2 - 40
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(badgeX), Y: 6, Width: 80, Height: 24}, 0.5, 8, colorAccent)
	drawTextEx(editorFontBold, "2D UI", badgeX+18, 9, 16, rl.White)

	// Zoom level indicator
	zoomText := fmt.Sprintf("%.0f%%", e.uiEditState.ViewZoom*100)
	drawTextEx(editorFontMono, zoomText, badgeX+90, 9, 18, colorTextMuted)

	// Set cursor based on UI edit state
	if e.uiEditState.Panning {
		rl.SetMouseCursor(rl.MouseCursorResizeAll)
	} else if e.uiEditState.Resizing {
		e.setResizeCursor(e.uiEditState.ResizeHandle)
	} else if e.uiEditState.HoveredHandle >= 0 {
		e.setResizeCursor(e.uiEditState.HoveredHandle)
	} else if e.uiEditState.Dragging || e.uiEditState.HoveredElement != nil {
		rl.SetMouseCursor(rl.MouseCursorPointingHand)
	}
}

// drawUIElementsForEdit renders UI elements with outlines for editing
func (e *Editor) drawUIElementsForEdit(obj *engine.GameObject, parentRect rl.Rectangle) {
	if obj == nil || !obj.Active {
		return
	}

	rt := engine.GetComponent[*components.RectTransform](obj)
	currentRect := parentRect

	if rt != nil {
		rt.CalculateRect(parentRect)
		currentRect = rt.GetScreenRect()
	}

	// Transform to screen space for drawing
	screenRect := e.canvasRectToScreen(currentRect)

	// Draw element visuals (skip canvas itself for the outline)
	isCanvas := engine.GetComponent[*components.UICanvas](obj) != nil

	if !isCanvas && rt != nil {
		// Draw semi-transparent fill to show bounds
		rl.DrawRectangleRec(screenRect, rl.NewColor(60, 60, 80, 60))
		// Draw outline
		rl.DrawRectangleLinesEx(screenRect, 1, rl.NewColor(90, 90, 110, 180))
	}

	// Draw actual UI content (transformed)
	if panel := engine.GetComponent[*components.UIPanel](obj); panel != nil {
		panel.Draw(screenRect)
	}
	if bar := engine.GetComponent[*components.UIProgressBar](obj); bar != nil {
		bar.Draw(screenRect)
	}
	if img := engine.GetComponent[*components.UIImage](obj); img != nil {
		img.Draw(screenRect)
	}
	if btn := engine.GetComponent[*components.UIButton](obj); btn != nil {
		btn.Draw(screenRect)
	}
	if text := engine.GetComponent[*components.UIText](obj); text != nil {
		e.drawUITextScaled(text, screenRect)
	}

	// Draw children
	for _, child := range obj.Children {
		e.drawUIElementsForEdit(child, currentRect)
	}
}

// drawUITextScaled draws text scaled proportionally with zoom
func (e *Editor) drawUITextScaled(text *components.UIText, rect rl.Rectangle) {
	if text.Text == "" {
		return
	}

	// Render text at original size, then use matrix transform to scale it
	fontSize := text.FontSize
	zoom := e.uiEditState.ViewZoom

	// Measure text at original size
	textWidth := float32(rl.MeasureText(text.Text, fontSize))
	textHeight := float32(fontSize)

	// Calculate position in unscaled space
	var xUnscaled float32
	switch text.Alignment {
	case components.TextAlignLeft:
		xUnscaled = 0
	case components.TextAlignCenter:
		xUnscaled = (rect.Width/zoom - textWidth) / 2
	case components.TextAlignRight:
		xUnscaled = rect.Width/zoom - textWidth
	}

	// Vertically center in unscaled space
	yUnscaled := (rect.Height/zoom - textHeight) / 2

	// Push matrix, scale, draw, pop
	rl.PushMatrix()
	rl.Translatef(rect.X, rect.Y, 0)
	rl.Scalef(zoom, zoom, 1)
	rl.DrawText(text.Text, int32(xUnscaled), int32(yUnscaled), fontSize, text.Color)
	rl.PopMatrix()
}

// drawResizeHandles draws the 8 resize handles around a rect
func (e *Editor) drawResizeHandles(rect rl.Rectangle) {
	handleSize := float32(8)
	handleColor := colorAccent
	hoveredColor := rl.White

	handles := []rl.Vector2{
		{X: rect.X, Y: rect.Y},
		{X: rect.X + rect.Width/2, Y: rect.Y},
		{X: rect.X + rect.Width, Y: rect.Y},
		{X: rect.X + rect.Width, Y: rect.Y + rect.Height/2},
		{X: rect.X + rect.Width, Y: rect.Y + rect.Height},
		{X: rect.X + rect.Width/2, Y: rect.Y + rect.Height},
		{X: rect.X, Y: rect.Y + rect.Height},
		{X: rect.X, Y: rect.Y + rect.Height/2},
	}

	for i, pos := range handles {
		color := handleColor
		if i == e.uiEditState.HoveredHandle {
			color = hoveredColor
		}
		// Draw filled square with border
		rl.DrawRectangle(
			int32(pos.X-handleSize/2),
			int32(pos.Y-handleSize/2),
			int32(handleSize),
			int32(handleSize),
			color,
		)
		rl.DrawRectangleLines(
			int32(pos.X-handleSize/2),
			int32(pos.Y-handleSize/2),
			int32(handleSize),
			int32(handleSize),
			rl.NewColor(40, 40, 50, 255),
		)
	}
}

// setResizeCursor sets the appropriate cursor for a resize handle
func (e *Editor) setResizeCursor(handle int) {
	switch handle {
	case HandleTopLeft, HandleBottomRight:
		rl.SetMouseCursor(rl.MouseCursorResizeNWSE)
	case HandleTopRight, HandleBottomLeft:
		rl.SetMouseCursor(rl.MouseCursorResizeNESW)
	case HandleTop, HandleBottom:
		rl.SetMouseCursor(rl.MouseCursorResizeNS)
	case HandleLeft, HandleRight:
		rl.SetMouseCursor(rl.MouseCursorResizeEW)
	default:
		rl.SetMouseCursor(rl.MouseCursorDefault)
	}
}

// IsUIEditModeActive returns true if the editor is in UI edit mode
func (e *Editor) IsUIEditModeActive() bool {
	return e.uiEditState != nil && e.uiEditState.Active
}
