package assets

import (
	"encoding/json"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Material defines surface properties for rendering
type Material struct {
	Name      string
	Color     rl.Color
	Metallic  float32
	Roughness float32
	Emissive  float32
}

// materialDef is the JSON format for material files
type materialDef struct {
	Name      string  `json:"name"`
	Color     string  `json:"color"`
	Metallic  float32 `json:"metallic"`
	Roughness float32 `json:"roughness"`
	Emissive  float32 `json:"emissive"`
}

var manager *Manager

type Manager struct {
	models    map[string]rl.Model
	textures  map[string]rl.Texture2D
	materials map[string]*Material
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

func Init() {
	manager = &Manager{
		models:    make(map[string]rl.Model),
		textures:  make(map[string]rl.Texture2D),
		materials: make(map[string]*Material),
	}
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

	manager.materials[path] = material
	return material
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
}
