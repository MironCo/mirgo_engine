//go:build !game

package game

import (
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ComponentType represents a type of component that can be added via the editor.
type ComponentType struct {
	Name    string
	Factory func(w *world.World, g *engine.GameObject) engine.Component
}

// editorComponentTypes lists all component types available in the Add Component menu.
var editorComponentTypes = []ComponentType{
	{"ModelRenderer", createModelRenderer},
	{"BoxCollider", createBoxCollider},
	{"SphereCollider", createSphereCollider},
	{"Rigidbody", createRigidbody},
	{"DirectionalLight", createDirectionalLight},
	{"PointLight", createPointLight},
	{"Camera", createCamera},
	{"FPSController", createFPSController},
}

func createModelRenderer(w *world.World, g *engine.GameObject) engine.Component {
	// Default: 1x1x1 white cube
	model := rl.LoadModelFromMesh(rl.GenMeshCube(1, 1, 1))
	renderer := components.NewModelRenderer(model, rl.White)
	renderer.MeshType = "cube"
	renderer.MeshSize = []float32{1, 1, 1}
	renderer.SetShader(w.Renderer.Shader)
	return renderer
}

func createBoxCollider(w *world.World, g *engine.GameObject) engine.Component {
	return components.NewBoxCollider(rl.Vector3{X: 1, Y: 1, Z: 1})
}

func createSphereCollider(w *world.World, g *engine.GameObject) engine.Component {
	return components.NewSphereCollider(0.5)
}

func createRigidbody(w *world.World, g *engine.GameObject) engine.Component {
	return components.NewRigidbody()
}

func createDirectionalLight(w *world.World, g *engine.GameObject) engine.Component {
	light := components.NewDirectionalLight()
	// Wire to renderer (only one directional light is supported)
	w.Light = g
	w.Renderer.SetLight(light)
	return light
}

func createPointLight(w *world.World, g *engine.GameObject) engine.Component {
	return components.NewPointLight()
}

func createCamera(w *world.World, g *engine.GameObject) engine.Component {
	return components.NewCamera()
}

func createFPSController(w *world.World, g *engine.GameObject) engine.Component {
	return components.NewFPSController()
}
