package world

import (
	"fmt"
	"math"
	"math/rand"
	"test3d/internal/assets"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/physics"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const FloorSize = 60.0

type World struct {
	Scene        *engine.Scene
	PhysicsWorld *physics.PhysicsWorld
	Renderer     *Renderer
	Light        *engine.GameObject
}

func New() *World {
	return &World{
		Scene:        engine.NewScene("Main"),
		PhysicsWorld: physics.NewPhysicsWorld(),
		Renderer:     NewRenderer(),
	}
}

func (w *World) Initialize() {
	assets.Init()
	w.Renderer.Initialize(FloorSize)

	// Create floor as a regular game object
	floor := engine.NewGameObject("Floor")
	floor.Transform.Position = rl.Vector3Zero()

	// Floor visual (plane mesh at Y=0)
	floorMesh := rl.GenMeshPlane(FloorSize, FloorSize, 1, 1)
	floorModel := rl.LoadModelFromMesh(floorMesh)
	floorRenderer := components.NewModelRenderer(floorModel, rl.LightGray)
	floorRenderer.SetShader(w.Renderer.Shader)
	floor.AddComponent(floorRenderer)

	// Floor collider (offset down so top face is at Y=0)
	floorCollider := components.NewBoxCollider(rl.Vector3{X: FloorSize, Y: 1.0, Z: FloorSize})
	floorCollider.Offset = rl.Vector3{X: 0, Y: -0.5, Z: 0}
	floor.AddComponent(floorCollider)

	w.Scene.AddGameObject(floor)
	w.PhysicsWorld.AddObject(floor) // No rigidbody = static

	// Create cube GameObjects
	w.createCubes()

	// Create duck
	w.createDuck()

	// Create player
	w.createPlayer()

	// Create light
	w.createLight()

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
	player.AddComponent(NewPlayerCollision(w))

	// Shooter (sphere spawning on mouse click)
	player.AddComponent(components.NewShooter(w, w.Renderer.Shader))

	w.Scene.AddGameObject(player)
	w.PhysicsWorld.AddObject(player)
}

func (w *World) createCubes() {
	numCubes := 15
	colors := []rl.Color{
		rl.Red, rl.Blue, rl.Green, rl.Purple, rl.Orange,
		rl.Yellow, rl.Pink, rl.SkyBlue, rl.Lime, rl.Magenta,
	}

	for i := range numCubes {
		angle := float32(i) * (2 * math.Pi / float32(numCubes))
		radius := float32(8 + rand.Float64()*5)

		pos := rl.Vector3{
			X: float32(math.Cos(float64(angle))) * radius,
			Y: float32(5 + rand.Float64()*10), // Start higher so they fall
			Z: float32(math.Sin(float64(angle))) * radius,
		}

		size := rl.Vector3{X: 1.5, Y: 1.5, Z: 1.5}
		color := colors[i%len(colors)]

		// Create GameObject
		cube := engine.NewGameObject(fmt.Sprintf("Cube_%d", i))
		cube.Transform.Position = pos

		// Create model and renderer
		mesh := rl.GenMeshCube(size.X, size.Y, size.Z)
		model := rl.LoadModelFromMesh(mesh)
		renderer := components.NewModelRenderer(model, color)
		renderer.SetShader(w.Renderer.Shader)
		cube.AddComponent(renderer)

		// Add box collider
		boxCol := components.NewBoxCollider(size)
		cube.AddComponent(boxCol)

		// Add rigidbody for physics
		rb := components.NewRigidbody()
		rb.Bounciness = 0.4 + rand.Float32()*0.4 // Random bounce 0.4-0.8
		rb.Friction = 0.05 + rand.Float32()*0.1
		// Give some initial spin
		cube.AddComponent(rb)

		w.Scene.AddGameObject(cube)
		w.PhysicsWorld.AddObject(cube)
	}
}

func (w *World) createDuck() {
	duck := engine.NewGameObject("Duck")
	duck.Transform.Position = rl.Vector3{X: 0, Y: 5, Z: 0}
	duck.Transform.Scale = rl.Vector3{X: 10, Y: 10, Z: 10}

	renderer := components.NewModelRendererFromFile("assets/models/rubber_duck_toy_1k.gltf/rubber_duck_toy_1k.gltf", rl.White)
	renderer.SetShader(w.Renderer.Shader)
	duck.AddComponent(renderer)

	// Add collider (approximate duck size when scaled)
	collider := components.NewBoxCollider(rl.Vector3{X: 0.8, Y: 0.6, Z: 0.8})
	duck.AddComponent(collider)

	// Add rigidbody for physics
	rb := components.NewRigidbody()
	rb.Bounciness = 0.6
	duck.AddComponent(rb)

	w.Scene.AddGameObject(duck)
	w.PhysicsWorld.AddObject(duck)
}

func (w *World) createLight() {
	light := engine.NewGameObject("DirectionalLight")

	lightComp := components.NewDirectionalLight()
	light.AddComponent(lightComp)

	w.Light = light
	w.Scene.AddGameObject(light)

	// Set the light on the renderer
	w.Renderer.SetLight(lightComp)
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

func (w *World) Unload() {
	w.Renderer.Unload(w.Scene.GameObjects)
	assets.Unload()
}
