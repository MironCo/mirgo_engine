//go:build !game

package game

import (
	"fmt"
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

// drawInspector draws the selected object's inspector on the right.
func (e *Editor) drawInspector() {
	if e.Selected == nil {
		return
	}

	panelW := e.inspectorWidth
	panelX := int32(rl.GetScreenWidth()) - panelW
	panelY := int32(36)
	panelH := int32(rl.GetScreenHeight()) - panelY

	// Fixed button area at bottom
	btnH := int32(26)
	btnAreaH := btnH + 20 // button height + padding
	scrollableH := panelH - btnAreaH

	// Panel background with subtle border
	rl.DrawRectangle(panelX, panelY, panelW, panelH, colorBgPanel)
	// Resize handle - slightly thicker border on left edge
	rl.DrawRectangle(panelX, panelY, 2, panelH, colorBorder)

	// Check for scroll input when mouse is in inspector (only in scrollable area)
	mousePos := rl.GetMousePosition()
	mouseInPanel := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH)
	mouseInScrollArea := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+scrollableH)

	if mouseInScrollArea && !rl.IsMouseButtonDown(rl.MouseRightButton) && !e.showAddComponentMenu {
		scroll := rl.GetMouseWheelMove()
		e.inspectorScroll -= int32(scroll * 20)
		if e.inspectorScroll < 0 {
			e.inspectorScroll = 0
		}
	}

	// Begin scissor mode for scrolling (only for scrollable content area)
	rl.BeginScissorMode(panelX, panelY, panelW, scrollableH)

	y := panelY + 8 - e.inspectorScroll

	// Name (editable)
	y = e.drawNameField(panelX, y, panelW, mousePos)

	// Tags (editable)
	y = e.drawTagsField(panelX, y, panelW, mousePos)

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

	// Clamp scroll to content (before ending scissor mode)
	totalHeight := y + e.inspectorScroll - panelY + 50
	maxScroll := totalHeight - scrollableH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.inspectorScroll > maxScroll {
		e.inspectorScroll = maxScroll
	}

	rl.EndScissorMode()

	// Fixed Add Component button at bottom (outside scissor mode)
	btnW := panelW - 40
	btnX := panelX + 20
	btnY := panelY + scrollableH + 10 // Fixed position at bottom of panel

	// Draw background for button area to cover any scrolled content
	rl.DrawRectangle(panelX, panelY+scrollableH, panelW, btnAreaH, colorBgPanel)
	// Separator line above button
	rl.DrawLine(panelX+12, panelY+scrollableH+2, panelX+panelW-12, panelY+scrollableH+2, rl.NewColor(40, 40, 55, 255))

	btnHovered := mouseInPanel && mousePos.X >= float32(btnX) && mousePos.X <= float32(btnX+btnW) &&
		mousePos.Y >= float32(btnY) && mousePos.Y <= float32(btnY+btnH)

	btnColor := colorBgElement
	txtColor := colorTextSecondary
	if btnHovered {
		btnColor = colorAccent
		txtColor = colorTextPrimary
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(btnX), Y: float32(btnY), Width: float32(btnW), Height: float32(btnH)}, 0.3, 6, btnColor)
	textW := rl.MeasureText("+ Add Component", 16)
	drawTextEx(editorFont, "+ Add Component", btnX+(btnW-textW)/2, btnY+5, 16, txtColor)

	clickedAddButton := false
	if btnHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.showAddComponentMenu = !e.showAddComponentMenu
		if e.showAddComponentMenu {
			e.addComponentScroll = 0 // Reset scroll when opening menu
		}
		clickedAddButton = true
	}

	// Draw add component dropdown menu (shows upward from button)
	if e.showAddComponentMenu {
		e.drawAddComponentMenu(btnX, btnY, btnW, clickedAddButton)
	}
}

// drawNameField draws the editable name field and returns the new Y position.
func (e *Editor) drawNameField(panelX, y, panelW int32, mousePos rl.Vector2) int32 {
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

	return y + nameFieldH + 4
}

// drawTagsField draws the editable tags field and returns the new Y position.
func (e *Editor) drawTagsField(panelX, y, panelW int32, mousePos rl.Vector2) int32 {
	drawTextEx(editorFont, "Tags", panelX+12, y, 14, colorTextMuted)
	y += 18

	tagsFieldW := panelW - 20
	tagsFieldH := int32(22)
	tagsFieldX := panelX + 10
	tagsFieldY := y

	tagsHovered := mousePos.X >= float32(tagsFieldX) && mousePos.X <= float32(tagsFieldX+tagsFieldW) &&
		mousePos.Y >= float32(tagsFieldY) && mousePos.Y <= float32(tagsFieldY+tagsFieldH)

	// Background for tags field
	tagsBgColor := colorBgElement
	if e.editingTags {
		tagsBgColor = colorBgActive
	} else if tagsHovered {
		tagsBgColor = colorBgHover
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(tagsFieldX), Y: float32(tagsFieldY), Width: float32(tagsFieldW), Height: float32(tagsFieldH)}, 0.2, 6, tagsBgColor)
	if e.editingTags {
		rl.DrawRectangleRoundedLinesEx(rl.Rectangle{X: float32(tagsFieldX), Y: float32(tagsFieldY), Width: float32(tagsFieldW), Height: float32(tagsFieldH)}, 0.2, 6, 1, colorAccent)
	}

	if e.editingTags {
		// Draw editing text with cursor
		drawTextEx(editorFont, e.tagsEditBuffer+"_", tagsFieldX+6, tagsFieldY+4, 14, colorTextPrimary)

		// Handle typing
		for {
			key := rl.GetCharPressed()
			if key == 0 {
				break
			}
			e.tagsEditBuffer += string(rune(key))
		}

		// Backspace
		if rl.IsKeyPressed(rl.KeyBackspace) && len(e.tagsEditBuffer) > 0 {
			e.tagsEditBuffer = e.tagsEditBuffer[:len(e.tagsEditBuffer)-1]
		}

		// Enter to confirm
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			e.applyTagsFromBuffer()
		}

		// Escape to cancel
		if rl.IsKeyPressed(rl.KeyEscape) {
			e.editingTags = false
			e.tagsEditBuffer = ""
		}

		// Click outside to confirm
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !tagsHovered {
			e.applyTagsFromBuffer()
		}
	} else {
		// Display tags
		tagStr := strings.Join(e.Selected.Tags, ", ")
		if tagStr == "" {
			tagStr = "(none)"
			drawTextEx(editorFont, tagStr, tagsFieldX+6, tagsFieldY+4, 14, colorTextMuted)
		} else {
			drawTextEx(editorFont, tagStr, tagsFieldX+6, tagsFieldY+4, 14, colorTextSecondary)
		}

		// Click to edit
		if tagsHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.editingTags = true
			e.tagsEditBuffer = strings.Join(e.Selected.Tags, ", ")
		}
	}

	return y + tagsFieldH + 6
}

// applyTagsFromBuffer parses the tags edit buffer and applies it to the selected object.
func (e *Editor) applyTagsFromBuffer() {
	if e.Selected == nil {
		e.editingTags = false
		e.tagsEditBuffer = ""
		return
	}

	// Parse comma-separated tags
	parts := strings.Split(e.tagsEditBuffer, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	e.Selected.Tags = tags
	e.editingTags = false
	e.tagsEditBuffer = ""
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

	case *components.MeshCollider:
		// Show read-only info about the mesh collider
		if comp.IsBuilt() {
			info := fmt.Sprintf("%d triangles", comp.TriangleCount())
			drawTextEx(editorFont, info, indent, y+4, 15, colorTextMuted)
		} else {
			drawTextEx(editorFont, "Not built", indent, y+4, 15, rl.Red)
		}
		y += fieldH + 6

	case *components.CharacterController:
		// Height
		drawTextEx(editorFont, "Height", indent, y+4, 15, colorTextMuted)
		comp.Height = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("cc%d.height", compIdx), comp.Height)
		y += fieldH + 2

		// Radius
		drawTextEx(editorFont, "Radius", indent, y+4, 15, colorTextMuted)
		comp.Radius = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("cc%d.radius", compIdx), comp.Radius)
		y += fieldH + 2

		// Step Height
		drawTextEx(editorFont, "Step", indent, y+4, 15, colorTextMuted)
		comp.StepHeight = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("cc%d.step", compIdx), comp.StepHeight)
		y += fieldH + 2

		// Gravity
		drawTextEx(editorFont, "Gravity", indent, y+4, 15, colorTextMuted)
		comp.Gravity = e.drawFloatField(indent+labelW, y, fieldW, fieldH, fmt.Sprintf("cc%d.grav", compIdx), comp.Gravity)
		y += fieldH + 4

		// UseGravity checkbox
		gravityBounds := rl.Rectangle{X: float32(indent), Y: float32(y), Width: float32(fieldH), Height: float32(fieldH)}
		comp.UseGravity = gui.CheckBox(gravityBounds, "Use Gravity", comp.UseGravity)
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
// The menu appears ABOVE the button (y is the button's top position).
func (e *Editor) drawAddComponentMenu(x, btnY, w int32, justOpened bool) {
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

	// Menu appears above the button
	menuY := btnY - menuH - 4

	mousePos := rl.GetMousePosition()
	mouseInMenu := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
		mousePos.Y >= float32(menuY) && mousePos.Y <= float32(menuY+menuH)

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
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(x), Y: float32(menuY), Width: float32(w), Height: float32(menuH)}, 0.1, 4, colorBgPanel)
	rl.DrawRectangleRoundedLinesEx(rl.Rectangle{X: float32(x), Y: float32(menuY), Width: float32(w), Height: float32(menuH)}, 0.1, 4, 1, colorBorder)

	// Begin scissor mode to clip items outside menu area
	rl.BeginScissorMode(x, menuY, w, menuH)

	itemIndex := int32(0)

	// Built-in components
	for _, compType := range editorComponentTypes {
		itemY := menuY + itemIndex*itemH - e.addComponentScroll

		// Skip if completely outside visible area
		if itemY+itemH < menuY || itemY > menuY+menuH {
			itemIndex++
			continue
		}

		hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
			mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH) &&
			mousePos.Y >= float32(menuY) && mousePos.Y < float32(menuY+menuH)

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
		sepY := menuY + itemIndex*itemH - e.addComponentScroll
		if sepY+itemH >= menuY && sepY <= menuY+menuH {
			rl.DrawRectangle(x+10, sepY+itemH/2, w-20, 1, colorBorder)
			drawTextEx(editorFont, "Scripts", x+12, sepY+5, 14, colorTextMuted)
		}
		itemIndex++

		// Script items
		for _, scriptName := range scripts {
			itemY := menuY + itemIndex*itemH - e.addComponentScroll

			// Skip if completely outside visible area
			if itemY+itemH < menuY || itemY > menuY+menuH {
				itemIndex++
				continue
			}

			hovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+w) &&
				mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH) &&
				mousePos.Y >= float32(menuY) && mousePos.Y < float32(menuY+menuH)

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
		scrollThumbY := menuY + 4 + int32(float32(scrollTrackH-scrollThumbH)*float32(e.addComponentScroll)/float32(maxScroll))

		// Draw scroll track
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(scrollBarX), Y: float32(menuY + 4), Width: float32(scrollBarW), Height: float32(scrollTrackH)}, 0.5, 4, colorBgDark)
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
			if newComp != nil {
				e.Selected.AddComponent(newComp)
			}

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

		// Auto-attach PlayerCollision for any PlayerController script
		if _, ok := newComp.(engine.PlayerController); ok {
			e.Selected.AddComponent(&world.PlayerCollision{})
		}

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
