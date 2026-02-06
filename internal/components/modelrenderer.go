package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type ModelRenderer struct {
	engine.BaseComponent
	Model  rl.Model
	Color  rl.Color
	shader rl.Shader
}

func NewModelRenderer(model rl.Model, color rl.Color) *ModelRenderer {
	return &ModelRenderer{
		Model: model,
		Color: color,
	}
}

func (m *ModelRenderer) SetShader(shader rl.Shader) {
	m.shader = shader
	m.Model.Materials.Shader = shader
	m.Model.Materials.Maps.Color = m.Color
}

func (m *ModelRenderer) Draw() {
	g := m.GetGameObject()
	if g == nil || !g.Active {
		return
	}

	// Build rotation matrix from Euler angles
	rot := g.Transform.Rotation
	rotX := rl.MatrixRotateX(rot.X * rl.Deg2rad)
	rotY := rl.MatrixRotateY(rot.Y * rl.Deg2rad)
	rotZ := rl.MatrixRotateZ(rot.Z * rl.Deg2rad)
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

	// Build translation matrix
	pos := g.Transform.Position
	transMatrix := rl.MatrixTranslate(pos.X, pos.Y, pos.Z)

	// Combine: first rotate, then translate
	m.Model.Transform = rl.MatrixMultiply(rotMatrix, transMatrix)

	rl.DrawModel(m.Model, rl.Vector3Zero(), 1.0, rl.White)
}

func (m *ModelRenderer) Unload() {
	rl.UnloadModel(m.Model)
}
