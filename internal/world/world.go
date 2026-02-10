package world

import (
	"log"
	"test3d/internal/assets"
	"test3d/internal/audio"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/physics"
	_ "test3d/internal/scripts"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var ScenePath = "assets/scenes/main.json"

const FloorSize = 60.0

type World struct {
	Scene        *engine.Scene
	PhysicsWorld *physics.PhysicsWorld
	Renderer     *Renderer
	Light        *engine.GameObject
}

func New() *World {
	w := &World{
		Scene:        engine.NewScene("Main"),
		PhysicsWorld: physics.NewPhysicsWorld(),
		Renderer:     NewRenderer(),
	}
	w.Scene.World = w
	return w
}

func (w *World) Initialize() {
	assets.Init()
	audio.Init()
	w.Renderer.Initialize(FloorSize)

	// Load scene objects from JSON
	if err := w.LoadScene(ScenePath); err != nil {
		log.Fatalf("failed to load scene: %v", err)
	}

	// Start all GameObjects
	w.Scene.Start()
}

// ResetScene reloads the scene from disk, removing all dynamically spawned
// objects and restoring scene objects to their saved state.
func (w *World) ResetScene() {
	// Unload all models
	for _, g := range w.Scene.GameObjects {
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Unload()
		}
	}

	// Clear scene and physics
	w.Scene.GameObjects = w.Scene.GameObjects[:0]
	w.PhysicsWorld.Objects = w.PhysicsWorld.Objects[:0]
	w.PhysicsWorld.Statics = w.PhysicsWorld.Statics[:0]
	w.PhysicsWorld.Kinematics = w.PhysicsWorld.Kinematics[:0]

	// Reload scene from disk (includes Player now)
	if err := w.LoadScene(ScenePath); err != nil {
		log.Printf("failed to reload scene: %v", err)
		return
	}
	w.Scene.Start()
}

func (w *World) Update(deltaTime float32) {
	w.PhysicsWorld.Update(deltaTime)
	w.Scene.Update(deltaTime)
	audio.Update()
}

// SpawnObject adds a GameObject to both the scene and physics world.
func (w *World) SpawnObject(g *engine.GameObject) {
	w.Scene.AddGameObject(g)
	w.PhysicsWorld.AddObject(g)
}

// GetCollidableObjects returns all GameObjects that have BoxColliders
func (w *World) GetCollidableObjects() []*engine.GameObject {
	var result []*engine.GameObject
	for _, g := range w.Scene.GameObjects {
		if collider := engine.GetComponent[*components.BoxCollider](g); collider != nil {
			result = append(result, g)
		}
	}
	return result
}

// Destroy removes a GameObject and unloads its resources (for runtime/game use).
func (w *World) Destroy(g *engine.GameObject) {
	w.Scene.RemoveGameObject(g)
	w.PhysicsWorld.RemoveObject(g)

	// Unload model if it has a ModelRenderer
	if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
		renderer.Unload()
	}
}

// EditorDestroy removes a GameObject but keeps resources loaded (for undo support).
func (w *World) EditorDestroy(g *engine.GameObject) {
	w.Scene.RemoveGameObject(g)
	w.PhysicsWorld.RemoveObject(g)
}

// Raycast performs a physics raycast and returns the result
func (w *World) Raycast(origin, direction rl.Vector3, maxDistance float32) (engine.RaycastResult, bool) {
	hit, ok := w.PhysicsWorld.Raycast(origin, direction, maxDistance)
	if !ok {
		return engine.RaycastResult{}, false
	}
	return engine.RaycastResult{
		GameObject: hit.GameObject,
		Point:      hit.Point,
		Normal:     hit.Normal,
		Distance:   hit.Distance,
	}, true
}

// EditorRaycast performs raycast that also hits objects without colliders (using model bounds)
func (w *World) EditorRaycast(origin, direction rl.Vector3, maxDistance float32) (engine.RaycastResult, bool) {
	hit, ok := w.PhysicsWorld.EditorRaycast(origin, direction, maxDistance, w.Scene.GameObjects)
	if !ok {
		return engine.RaycastResult{}, false
	}
	return engine.RaycastResult{
		GameObject: hit.GameObject,
		Point:      hit.Point,
		Normal:     hit.Normal,
		Distance:   hit.Distance,
	}, true
}

func (w *World) Unload() {
	w.Renderer.Unload(w.Scene.GameObjects)
	assets.Unload()
	audio.Close()
}

func (w *World) GetShader() rl.Shader {
	return w.Renderer.Shader
}

// FindMainCamera returns the first Camera component with IsMain=true, or the first Camera found
func (w *World) FindMainCamera() *components.Camera {
	var firstCamera *components.Camera
	for _, g := range w.Scene.GameObjects {
		if cam := engine.GetComponent[*components.Camera](g); cam != nil {
			if cam.IsMain {
				return cam
			}
			if firstCamera == nil {
				firstCamera = cam
			}
		}
	}
	return firstCamera
}
