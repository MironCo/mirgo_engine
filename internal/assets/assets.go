package assets

import (
	"encoding/json"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Material defines surface properties for rendering
type Material struct {
	Name       string
	Color      rl.Color
	Metallic   float32
	Roughness  float32
	Emissive   float32
	Albedo     rl.Texture2D // diffuse/albedo texture (if ID > 0, use texture instead of color)
	AlbedoPath string       // path to albedo texture (for saving)
}

// materialDef is the JSON format for material files
type materialDef struct {
	Name      string  `json:"name"`
	Color     string  `json:"color"`
	Metallic  float32 `json:"metallic"`
	Roughness float32 `json:"roughness"`
	Emissive  float32 `json:"emissive"`
	Albedo    string  `json:"albedo,omitempty"` // path to albedo texture
}

var manager *Manager

type Manager struct {
	models    map[string]rl.Model
	textures  map[string]rl.Texture2D
	materials map[string]*Material
	meshes    map[string]rl.Mesh // unused, kept for compatibility
}

// Color name mapping for materials
var colorByName = map[string]rl.Color{
	"Red":       rl.Red,
	"Blue":      rl.Blue,
	"Green":     rl.Green,
	"Purple":    rl.Purple,
	"Orange":    rl.Orange,
	"Yellow":    rl.Yellow,
	"Gold":      rl.Gold,
	"White":     rl.White,
	"Gray":      rl.Gray,
	"LightGray": rl.LightGray,
	"DarkGray":  rl.DarkGray,
	"Black":     rl.Black,
	"Pink":      rl.Pink,
	"Maroon":    rl.Maroon,
	"Brown":     rl.Brown,
	"Beige":     rl.Beige,
	"SkyBlue":   rl.SkyBlue,
	"DarkBlue":  rl.DarkBlue,
	"Lime":      rl.Lime,
	"DarkGreen": rl.DarkGreen,
}

// LookupColor returns a raylib color from a name string
func LookupColor(name string) rl.Color {
	if c, ok := colorByName[name]; ok {
		return c
	}
	return rl.White
}

// LookupColorName returns the name of a color, or "White" if not found
func LookupColorName(c rl.Color) string {
	for name, col := range colorByName {
		if col.R == c.R && col.G == c.G && col.B == c.B && col.A == c.A {
			return name
		}
	}
	return "White"
}

func Init() {
	manager = &Manager{
		models:    make(map[string]rl.Model),
		textures:  make(map[string]rl.Texture2D),
		materials: make(map[string]*Material),
		meshes:    make(map[string]rl.Mesh),
	}
}

// GetSphereModel returns a cached sphere model. All spheres share this model.
// Scale is applied via transform, not mesh generation.
// The returned model should NOT be unloaded by the caller.
func GetSphereModel() rl.Model {
	if manager == nil {
		Init()
	}
	key := "sphere_16_16"
	if model, exists := manager.models[key]; exists {
		return model
	}
	mesh := rl.GenMeshSphere(1.0, 16, 16) // unit sphere, scale via transform
	model := rl.LoadModelFromMesh(mesh)
	manager.models[key] = model
	return model
}

// GetCubeModel returns a cached unit cube model.
// The returned model should NOT be unloaded by the caller.
func GetCubeModel() rl.Model {
	if manager == nil {
		Init()
	}
	key := "cube_1"
	if model, exists := manager.models[key]; exists {
		return model
	}
	mesh := rl.GenMeshCube(1.0, 1.0, 1.0)
	model := rl.LoadModelFromMesh(mesh)
	manager.models[key] = model
	return model
}

func LoadModel(path string) rl.Model {
	if manager == nil {
		Init()
	}

	if model, exists := manager.models[path]; exists {
		return model
	}

	model := rl.LoadModel(path)
	manager.models[path] = model
	return model
}

func LoadTexture(path string) rl.Texture2D {
	if manager == nil {
		Init()
	}

	if texture, exists := manager.textures[path]; exists {
		return texture
	}

	texture := rl.LoadTexture(path)
	manager.textures[path] = texture
	return texture
}

// LoadMaterial loads a material from a JSON file, caching it for reuse
func LoadMaterial(path string) *Material {
	if manager == nil {
		Init()
	}

	if material, exists := manager.materials[path]; exists {
		return material
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Return default material on error
		return &Material{
			Name:      "default",
			Color:     rl.White,
			Roughness: 0.5,
		}
	}

	var def materialDef
	if err := json.Unmarshal(data, &def); err != nil {
		return &Material{
			Name:      "default",
			Color:     rl.White,
			Roughness: 0.5,
		}
	}

	material := &Material{
		Name:      def.Name,
		Color:     LookupColor(def.Color),
		Metallic:  def.Metallic,
		Roughness: def.Roughness,
		Emissive:  def.Emissive,
	}

	// Load albedo texture if specified
	if def.Albedo != "" {
		material.Albedo = LoadTexture(def.Albedo)
		material.AlbedoPath = def.Albedo
	}

	manager.materials[path] = material
	return material
}

// SaveMaterial saves a material back to its JSON file
func SaveMaterial(path string, mat *Material) error {
	def := materialDef{
		Name:      mat.Name,
		Color:     LookupColorName(mat.Color),
		Metallic:  mat.Metallic,
		Roughness: mat.Roughness,
		Emissive:  mat.Emissive,
		Albedo:    mat.AlbedoPath,
	}

	data, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Unload() {
	if manager == nil {
		return
	}

	for _, model := range manager.models {
		rl.UnloadModel(model)
	}

	for _, texture := range manager.textures {
		rl.UnloadTexture(texture)
	}

	manager.models = make(map[string]rl.Model)
	manager.textures = make(map[string]rl.Texture2D)
	manager.materials = make(map[string]*Material)
	manager.meshes = make(map[string]rl.Mesh)
}
