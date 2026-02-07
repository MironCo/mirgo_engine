package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type MeshType int

const (
	MeshCube MeshType = iota
	MeshSphere
	MeshPlane
)

type MeshRenderer struct {
	engine.BaseComponent
	MeshType MeshType
	Color    rl.Color
	Size     rl.Vector3
}

func NewMeshRenderer(meshType MeshType, color rl.Color, size rl.Vector3) *MeshRenderer {
	return &MeshRenderer{
		MeshType: meshType,
		Color:    color,
		Size:     size,
	}
}

func (m *MeshRenderer) Draw() {
	g := m.GetGameObject()
	if g == nil || !g.Active {
		return
	}

	pos := g.WorldPosition()

	switch m.MeshType {
	case MeshCube:
		rl.DrawCubeV(pos, m.Size, m.Color)
	case MeshSphere:
		rl.DrawSphere(pos, m.Size.X, m.Color)
	case MeshPlane:
		rl.DrawPlane(pos, rl.Vector2{X: m.Size.X, Y: m.Size.Z}, m.Color)
	}
}
