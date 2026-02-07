//go:build !game

package game

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const maxUndoStack = 50

// UndoState captures the transform state of an object for undo
type UndoState struct {
	Object   *engine.GameObject
	Position rl.Vector3
	Rotation rl.Vector3
	Scale    rl.Vector3
}

// pushUndo saves the current transform state of the selected object
func (e *Editor) pushUndo() {
	if e.Selected == nil {
		return
	}
	state := UndoState{
		Object:   e.Selected,
		Position: e.Selected.Transform.Position,
		Rotation: e.Selected.Transform.Rotation,
		Scale:    e.Selected.Transform.Scale,
	}
	// Cap stack size
	if len(e.undoStack) >= maxUndoStack {
		e.undoStack = e.undoStack[1:]
	}
	e.undoStack = append(e.undoStack, state)
}

// undo restores the last saved transform state
func (e *Editor) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	// Pop last state
	state := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]

	// Restore transform
	if state.Object != nil {
		state.Object.Transform.Position = state.Position
		state.Object.Transform.Rotation = state.Rotation
		state.Object.Transform.Scale = state.Scale
		e.Selected = state.Object
	}
}
