//go:build game

package game

import (
	"test3d/internal/engine"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Editor struct {
	Active   bool
	Selected *engine.GameObject
}

type EditorPrefs struct {
	WindowWidth  int
	WindowHeight int
	WindowX      int
	WindowY      int
	ScenePath    string
}

func NewEditor(_ *world.World) *Editor { return &Editor{} }
func (e *Editor) Enter(_ rl.Camera3D)  { rl.DisableCursor() }
func (e *Editor) Pause(_ rl.Camera3D)  {}
func (e *Editor) Exit()                {}
func (e *Editor) Update(_ float32)     {}
func (e *Editor) RestoreState()        {}
func (e *Editor) GetRaylibCamera() rl.Camera3D {
	return rl.Camera3D{}
}
func (e *Editor) Draw3D()                   {}
func (e *Editor) DrawUI()                   {}
func (e *Editor) SavePrefs()                {}
func (e *Editor) ApplyPrefs(_ *EditorPrefs) {}
func LoadEditorPrefs() *EditorPrefs         { return nil }
