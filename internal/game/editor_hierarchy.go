//go:build !game

package game

import (
	"fmt"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

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
			now := rl.GetTime()
			isDoubleClick := (now-e.lastHierarchyClick < 0.3) && (e.lastClickedObject == g)

			e.Selected = g
			e.draggingHierarchy = true
			e.draggedObject = g

			// If in UI edit mode, also sync the UI edit state selection
			if e.IsUIEditModeActive() {
				e.uiEditState.SelectedElement = g
			}

			if isDoubleClick {
				// Double-click: focus camera on object
				e.focusOnObject(g)
			}

			e.lastHierarchyClick = now
			e.lastClickedObject = g
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
