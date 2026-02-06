package world

import (
	"fmt"
	"math"
	"math/rand"
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
}

func New() *World {
	return &World{
		Scene:        engine.NewScene("Main"),
		PhysicsWorld: physics.NewPhysicsWorld(),
		Renderer:     NewRenderer(),
	}
}

func (w *World) Initialize() {
	w.Renderer.Initialize(FloorSize)

	// Create floor collider for physics (thin box at Y=0)
	floor := engine.NewGameObject("Floor")
	floor.Transform.Position = rl.Vector3{X: 0, Y: -0.5, Z: 0} // Center of box is below surface
	floorCollider := components.NewBoxCollider(rl.Vector3{X: FloorSize, Y: 1.0, Z: FloorSize})
	floor.AddComponent(floorCollider)
	w.Scene.AddGameObject(floor)
	w.PhysicsWorld.AddObject(floor) // No rigidbody = static

	// Create cube GameObjects
	w.createCubes()

	// Start all GameObjects
	w.Scene.Start()
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
		rb.AngularVelocity = rl.Vector3{
			X: (rand.Float32() - 0.5) * 200,
			Y: (rand.Float32() - 0.5) * 200,
			Z: (rand.Float32() - 0.5) * 200,
		}
		cube.AddComponent(rb)

		w.Scene.AddGameObject(cube)
		w.PhysicsWorld.AddObject(cube)
	}
}

func (w *World) Update(deltaTime float32) {
	w.PhysicsWorld.Update(deltaTime)
	w.Scene.Update(deltaTime)
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
}
