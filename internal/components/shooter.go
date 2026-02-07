package components

import (
	"fmt"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Shooter struct {
	engine.BaseComponent
	Shader       rl.Shader
	Cooldown     float64
	lastShotTime float64
	shotCounter  int
}

func NewShooter(shader rl.Shader) *Shooter {
	return &Shooter{
		Shader:   shader,
		Cooldown: 0.15,
	}
}

func (s *Shooter) Update(deltaTime float32) {
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.GetTime()-s.lastShotTime >= s.Cooldown {
		s.Shoot()
		s.lastShotTime = rl.GetTime()
	}

	if rl.IsMouseButtonPressed(rl.MouseRightButton) {
		s.DeleteTarget()
	}
}

func (s *Shooter) DeleteTarget() {
	g := s.GetGameObject()
	fps := engine.GetComponent[*FPSController](g)
	if fps == nil {
		return
	}

	// Raycast from eye level, not feet
	origin := g.Transform.Position
	origin.Y += fps.EyeHeight
	direction := fps.GetLookDirection()

	hit, ok := s.GetGameObject().Scene.World.Raycast(origin, direction, 100.0)
	if !ok {
		return
	}

	// Don't delete the floor or the player
	if hit.GameObject.Name == "Floor" || hit.GameObject.Name == "Player" {
		return
	}

	s.GetGameObject().Scene.World.Destroy(hit.GameObject)
}

func (s *Shooter) Shoot() {
	g := s.GetGameObject()
	fps := engine.GetComponent[*FPSController](g)
	if fps == nil {
		return
	}

	s.shotCounter++

	// Spawn from eye level
	eyePos := g.Transform.Position
	eyePos.Y += fps.EyeHeight
	lookDir := fps.GetLookDirection()
	spawnPos := rl.Vector3Add(eyePos, rl.Vector3Scale(lookDir, 3))

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
	s.GetGameObject().Scene.World.SpawnObject(sphere)
}
