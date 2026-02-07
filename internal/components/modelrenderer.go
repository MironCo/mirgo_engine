package components

import (
	"test3d/internal/assets"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type ModelRenderer struct {
	engine.BaseComponent
	Model    rl.Model
	Color    rl.Color
	shader   rl.Shader
	fromFile bool // true if loaded via asset manager
}

func NewModelRenderer(model rl.Model, color rl.Color) *ModelRenderer {
	return &ModelRenderer{
		Model:    model,
		Color:    color,
		fromFile: false,
	}
}

func NewModelRendererFromFile(path string, color rl.Color) *ModelRenderer {
	return &ModelRenderer{
		Model:    assets.LoadModel(path),
		Color:    color,
		fromFile: true,
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

	// Build scale matrix
	scale := g.Transform.Scale
	scaleMatrix := rl.MatrixScale(scale.X, scale.Y, scale.Z)

	// Build rotation matrix from Euler angles
	rot := g.Transform.Rotation
	rotX := rl.MatrixRotateX(rot.X * rl.Deg2rad)
	rotY := rl.MatrixRotateY(rot.Y * rl.Deg2rad)
	rotZ := rl.MatrixRotateZ(rot.Z * rl.Deg2rad)
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

	// Build translation matrix
	pos := g.Transform.Position
	transMatrix := rl.MatrixTranslate(pos.X, pos.Y, pos.Z)

	// Combine: scale -> rotate -> translate
	m.Model.Transform = rl.MatrixMultiply(rl.MatrixMultiply(scaleMatrix, rotMatrix), transMatrix)

	rl.DrawModel(m.Model, rl.Vector3Zero(), 1.0, rl.White)
}

func (m *ModelRenderer) Unload() {
	// Only unload if not from asset manager (asset manager handles its own cleanup)
	if !m.fromFile {
		rl.UnloadModel(m.Model)
	}
}
