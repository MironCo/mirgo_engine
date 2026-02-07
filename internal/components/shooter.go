package components

import (
	"fmt"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Shooter struct {
	engine.BaseComponent
	World       engine.WorldAccess
	Shader      rl.Shader
	Cooldown    float64
	lastShotTime float64
	shotCounter  int
}

func NewShooter(world engine.WorldAccess, shader rl.Shader) *Shooter {
	return &Shooter{
		World:    world,
		Shader:   shader,
		Cooldown: 0.15,
	}
}

func (s *Shooter) Update(deltaTime float32) {
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.GetTime()-s.lastShotTime >= s.Cooldown {
		s.Shoot()
		s.lastShotTime = rl.GetTime()
	}
}

func (s *Shooter) Shoot() {
	g := s.GetGameObject()
	fps := engine.GetComponent[*FPSController](g)
	if fps == nil {
		return
	}

	s.shotCounter++

	lookDir := fps.GetLookDirection()
	spawnPos := rl.Vector3Add(g.Transform.Position, rl.Vector3Scale(lookDir, 3))

	radius := float32(0.5)

	sphere := engine.NewGameObject(fmt.Sprintf("Shot_%d", s.shotCounter))
	sphere.Transform.Position = spawnPos

	mesh := rl.GenMeshSphere(radius, 16, 16)
	model := rl.LoadModelFromMesh(mesh)
	renderer := NewModelRenderer(model, rl.Orange)
	renderer.SetShader(s.Shader)
	sphere.AddComponent(renderer)

	sphere.AddComponent(NewSphereCollider(radius))

	rb := NewRigidbody()
	rb.Bounciness = 0.6
	rb.Friction = 0.1
	rb.Velocity = rl.Vector3Scale(lookDir, 30)
	sphere.AddComponent(rb)

	sphere.Start()
	s.World.SpawnObject(sphere)
}
