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

	// Material properties (inline, used when Material is nil)
	Metallic  float32 // 0 = diffuse, 1 = metallic
	Roughness float32 // 0 = smooth/shiny, 1 = rough/matte
	Emissive  float32 // emission intensity (0 = none)

	// Material asset reference (takes precedence over inline properties)
	Material     *assets.Material
	MaterialPath string // path to material JSON file for serialization
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

	// Set material uniforms - use Material if set, otherwise use inline properties
	var metallic, roughness, emissive float32
	var color rl.Color
	if m.Material != nil {
		metallic = m.Material.Metallic
		roughness = m.Material.Roughness
		emissive = m.Material.Emissive
		color = m.Material.Color
	} else {
		metallic = m.Metallic
		roughness = m.Roughness
		emissive = m.Emissive
		color = m.Color
	}

	// Apply material texture/color to ALL materials (GLTF models can have multiple)
	materials := unsafe.Slice(m.Model.Materials, m.Model.MaterialCount)
	if m.Material != nil && m.Material.Albedo.ID > 0 {
		// Material has texture - apply to all models (overrides GLTF textures)
		for i := range materials {
			materials[i].Maps.Texture = m.Material.Albedo
			materials[i].Maps.Color = rl.White // tint white so texture shows true color
		}
	} else if m.FilePath == "" {
		// Generated mesh with no material texture - use color
		for i := range materials {
			materials[i].Maps.Color = color
		}
	}
	// else: file-loaded model with no material texture - keep original GLTF texture

	if m.shader.ID > 0 {
		metallicLoc := rl.GetShaderLocation(m.shader, "metallic")
		roughnessLoc := rl.GetShaderLocation(m.shader, "roughness")
		emissiveLoc := rl.GetShaderLocation(m.shader, "emissive")

		rl.SetShaderValue(m.shader, metallicLoc, []float32{metallic}, rl.ShaderUniformFloat)
		rl.SetShaderValue(m.shader, roughnessLoc, []float32{roughness}, rl.ShaderUniformFloat)
		rl.SetShaderValue(m.shader, emissiveLoc, []float32{emissive}, rl.ShaderUniformFloat)
	}

	rl.DrawModel(m.Model, rl.Vector3Zero(), 1.0, rl.White)
}

func (m *ModelRenderer) Unload() {
	// Only unload if not from asset manager (asset manager handles its own cleanup)
	// Skip if FilePath is set (loaded from file) or MeshType is set (shared primitive)
	if m.FilePath == "" && m.MeshType == "" {
		rl.UnloadModel(m.Model)
	}
}
