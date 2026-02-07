//go:build !game

package game

import (
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const maxUndoStack = 50

// UndoActionType represents the type of action that can be undone
type UndoActionType int

const (
	UndoTransform UndoActionType = iota
	UndoDelete
)

// UndoState captures state for undo operations
type UndoState struct {
	Type     UndoActionType
	Object   *engine.GameObject
	Position rl.Vector3
	Rotation rl.Vector3
	Scale    rl.Vector3

	// For delete undo - we store enough info to recreate
	DeletedName   string
	DeletedParent *engine.GameObject
}

// pushUndo saves the current transform state of the selected object
func (e *Editor) pushUndo() {
	if e.Selected == nil {
		return
	}
	state := UndoState{
		Type:     UndoTransform,
		Object:   e.Selected,
		Position: e.Selected.Transform.Position,
		Rotation: e.Selected.Transform.Rotation,
		Scale:    e.Selected.Transform.Scale,
	}
	e.addUndoState(state)
}

// pushDeleteUndo saves the deleted object so it can be restored
func (e *Editor) pushDeleteUndo(obj *engine.GameObject) {
	state := UndoState{
		Type:          UndoDelete,
		Object:        obj,
		DeletedName:   obj.Name,
		DeletedParent: obj.Parent,
		Position:      obj.Transform.Position,
		Rotation:      obj.Transform.Rotation,
		Scale:         obj.Transform.Scale,
	}
	e.addUndoState(state)
}

func (e *Editor) addUndoState(state UndoState) {
	// Cap stack size
	if len(e.undoStack) >= maxUndoStack {
		e.undoStack = e.undoStack[1:]
	}
	e.undoStack = append(e.undoStack, state)
}

// undo restores the last saved state
func (e *Editor) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	// Pop last state
	state := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]

	switch state.Type {
	case UndoTransform:
		// Restore transform
		if state.Object != nil {
			state.Object.Transform.Position = state.Position
			state.Object.Transform.Rotation = state.Rotation
			state.Object.Transform.Scale = state.Scale
			e.Selected = state.Object
		}

	case UndoDelete:
		// Restore deleted object
		if state.Object != nil {
			// Re-add to scene and physics
			e.world.Scene.AddGameObject(state.Object)
			e.world.PhysicsWorld.AddObject(state.Object)

			// Restore parent relationship if any
			if state.DeletedParent != nil {
				state.Object.Parent = state.DeletedParent
			}

			// Re-apply shader to model renderer if present
			if mr := engine.GetComponent[*components.ModelRenderer](state.Object); mr != nil {
				mr.SetShader(e.world.Renderer.Shader)
			}

			e.Selected = state.Object
			e.setMsg("Restored %s", state.DeletedName)
		}
	}
}
