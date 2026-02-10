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
// Note: FPSController is now a script, added via the Scripts section.
var editorComponentTypes = []ComponentType{
	{"ModelRenderer", createModelRenderer},
	{"BoxCollider", createBoxCollider},
	{"SphereCollider", createSphereCollider},
	{"MeshCollider", createMeshCollider},
	{"Rigidbody", createRigidbody},
	{"DirectionalLight", createDirectionalLight},
	{"PointLight", createPointLight},
	{"Camera", createCamera},
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

func createMeshCollider(w *world.World, g *engine.GameObject) engine.Component {
	meshCol := components.NewMeshCollider()
	// If object already has a ModelRenderer, build the collider from it
	if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
		// Need to add component first so it has access to GameObject
		g.AddComponent(meshCol)
		meshCol.BuildFromModel(renderer.Model)
		// Return nil since we already added it
		return nil
	}
	return meshCol
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
