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
	rl.DrawModel(m.Model, g.Transform.Position, 1.0, rl.White)
}

func (m *ModelRenderer) Unload() {
	rl.UnloadModel(m.Model)
}
