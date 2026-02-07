package components

import (
	"test3d/internal/assets"
	"test3d/internal/engine"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type ModelRenderer struct {
	engine.BaseComponent
	Model    rl.Model
	Color    rl.Color
	shader   rl.Shader
	FilePath string    // non-empty for file-loaded models
	MeshType string    // "cube", "plane", "sphere" for generated meshes
	MeshSize []float32 // mesh generation parameters

	// Material properties
	Metallic  float32 // 0 = diffuse, 1 = metallic
	Roughness float32 // 0 = smooth/shiny, 1 = rough/matte
	Emissive  float32 // emission intensity (0 = none)
}

func NewModelRenderer(model rl.Model, color rl.Color) *ModelRenderer {
	return &ModelRenderer{
		Model:     model,
		Color:     color,
		Roughness: 0.5,
	}
}

func NewModelRendererFromFile(path string, color rl.Color) *ModelRenderer {
	return &ModelRenderer{
		Model:     assets.LoadModel(path),
		Color:     color,
		FilePath:  path,
		Roughness: 0.5,
	}
}

func (m *ModelRenderer) SetShader(shader rl.Shader) {
	m.shader = shader

	// Apply shader to ALL materials (GLTF models can have multiple)
	materials := unsafe.Slice(m.Model.Materials, m.Model.MaterialCount)
	for i := range materials {
		materials[i].Shader = shader
	}

	// Only override color for generated meshes, not file-loaded models
	// File-loaded models (GLTF) keep their original textures
	if m.FilePath == "" {
		m.Model.Materials.Maps.Color = m.Color
	}
}

func (m *ModelRenderer) Draw() {
	g := m.GetGameObject()
	if g == nil || !g.Active {
		return
	}

	// Build scale matrix (world space)
	scale := g.WorldScale()
	scaleMatrix := rl.MatrixScale(scale.X, scale.Y, scale.Z)

	// Build rotation matrix from world Euler angles
	rot := g.WorldRotation()
	rotX := rl.MatrixRotateX(rot.X * rl.Deg2rad)
	rotY := rl.MatrixRotateY(rot.Y * rl.Deg2rad)
	rotZ := rl.MatrixRotateZ(rot.Z * rl.Deg2rad)
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

	// Build translation matrix (world space)
	pos := g.WorldPosition()
	transMatrix := rl.MatrixTranslate(pos.X, pos.Y, pos.Z)

	// Combine: scale -> rotate -> translate
	m.Model.Transform = rl.MatrixMultiply(rl.MatrixMultiply(scaleMatrix, rotMatrix), transMatrix)

	// Set material uniforms
	if m.shader.ID > 0 {
		metallicLoc := rl.GetShaderLocation(m.shader, "metallic")
		roughnessLoc := rl.GetShaderLocation(m.shader, "roughness")
		emissiveLoc := rl.GetShaderLocation(m.shader, "emissive")

		rl.SetShaderValue(m.shader, metallicLoc, []float32{m.Metallic}, rl.ShaderUniformFloat)
		rl.SetShaderValue(m.shader, roughnessLoc, []float32{m.Roughness}, rl.ShaderUniformFloat)
		rl.SetShaderValue(m.shader, emissiveLoc, []float32{m.Emissive}, rl.ShaderUniformFloat)
	}

	rl.DrawModel(m.Model, rl.Vector3Zero(), 1.0, rl.White)
}

func (m *ModelRenderer) Unload() {
	// Only unload if not from asset manager (asset manager handles its own cleanup)
	if m.FilePath == "" {
		rl.UnloadModel(m.Model)
	}
}
