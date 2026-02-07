package assets

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var manager *Manager

type Manager struct {
	models   map[string]rl.Model
	textures map[string]rl.Texture2D
}

func Init() {
	manager = &Manager{
		models:   make(map[string]rl.Model),
		textures: make(map[string]rl.Texture2D),
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
}
