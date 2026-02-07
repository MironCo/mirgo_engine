package world

import (
	"log"
	"test3d/internal/assets"
	"test3d/internal/components"
	_ "test3d/internal/components/scripts"
	"test3d/internal/engine"
	"test3d/internal/physics"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ScenePath = "assets/scenes/main.json"

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
	w.Renderer.Initialize(FloorSize)

	// Load scene objects from JSON
	if err := w.LoadScene(ScenePath); err != nil {
		log.Fatalf("failed to load scene: %v", err)
	}

	// Create player (code-managed, not in scene file)
	w.createPlayer()

	// Start all GameObjects
	w.Scene.Start()
}

func (w *World) createPlayer() {
	player := engine.NewGameObject("Player")
	player.Transform.Position = rl.Vector3{X: 10, Y: 10, Z: 10}

	// FPS controller (movement + mouse look)
	fps := components.NewFPSController()
	player.AddComponent(fps)

	// Camera
	cam := components.NewCamera()
	player.AddComponent(cam)

	// Box collider for player body
	collider := components.NewBoxCollider(rl.Vector3{X: 0.6, Y: 1.8, Z: 0.6})
	player.AddComponent(collider)

	// Kinematic rigidbody so player can push things
	rb := components.NewRigidbody()
	rb.IsKinematic = true
	rb.UseGravity = false // FPSController handles gravity
	player.AddComponent(rb)

	// Player collision (ground check + AABB resolution)
	player.AddComponent(&PlayerCollision{})

	// Shooter (sphere spawning on mouse click)
	player.AddComponent(components.NewShooter(w.Renderer.Shader))

	w.Scene.AddGameObject(player)
	w.PhysicsWorld.AddObject(player)
}


// ResetScene reloads the scene from disk, removing all dynamically spawned
// objects and restoring scene objects to their saved state.
// The Player is preserved but reset to its spawn position.
func (w *World) ResetScene() {
	player := w.Scene.FindByName("Player")

	// Unload models for all non-player objects
	for _, g := range w.Scene.GameObjects {
		if g == player {
			continue
		}
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Unload()
		}
	}

	// Clear scene and physics, keeping only player
	w.Scene.GameObjects = w.Scene.GameObjects[:0]
	w.PhysicsWorld.Objects = w.PhysicsWorld.Objects[:0]
	w.PhysicsWorld.Statics = w.PhysicsWorld.Statics[:0]
	w.PhysicsWorld.Kinematics = w.PhysicsWorld.Kinematics[:0]

	if player != nil {
		w.Scene.AddGameObject(player)
		w.PhysicsWorld.AddObject(player)
		player.Transform.Position = rl.Vector3{X: 10, Y: 10, Z: 10}
		player.Transform.Rotation = rl.Vector3{}
		if rb := engine.GetComponent[*components.Rigidbody](player); rb != nil {
			rb.Velocity = rl.Vector3{}
		}
	}

	// Reload scene from disk
	if err := w.LoadScene(ScenePath); err != nil {
		log.Printf("failed to reload scene: %v", err)
		return
	}
	w.Scene.Start()
}

func (w *World) Update(deltaTime float32) {
	w.PhysicsWorld.Update(deltaTime)
	w.Scene.Update(deltaTime)
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

func (w *World) Destroy(g *engine.GameObject) {
	w.Scene.RemoveGameObject(g)
	w.PhysicsWorld.RemoveObject(g)

	// Unload model if it has a ModelRenderer
	if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
		renderer.Unload()
	}
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

func (w *World) Unload() {
	w.Renderer.Unload(w.Scene.GameObjects)
	assets.Unload()
}
