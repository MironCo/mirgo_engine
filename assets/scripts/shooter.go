package scripts

import (
	"fmt"
	"test3d/internal/assets"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Shooter struct {
	engine.BaseComponent
	Cooldown     float64
	lastShotTime float64
	shotCounter  int
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
	look := engine.FindComponent[engine.LookProvider](g)
	if look == nil {
		return
	}

	origin := g.Transform.Position
	origin.Y += look.GetEyeHeight()
	dx, dy, dz := look.GetLookDirection()
	direction := rl.Vector3{X: dx, Y: dy, Z: dz}

	hit, ok := s.GetGameObject().Scene.World.Raycast(origin, direction, 100.0)
	if !ok {
		return
	}

	if hit.GameObject.Name == "Floor" || hit.GameObject.Name == "Player" {
		return
	}

	s.GetGameObject().Scene.World.Destroy(hit.GameObject)
}

func (s *Shooter) Shoot() {
	g := s.GetGameObject()
	look := engine.FindComponent[engine.LookProvider](g)
	if look == nil {
		return
	}

	s.shotCounter++

	eyePos := g.Transform.Position
	eyePos.Y += look.GetEyeHeight()
	dx, dy, dz := look.GetLookDirection()
	lookDir := rl.Vector3{X: dx, Y: dy, Z: dz}
	spawnPos := rl.Vector3Add(eyePos, rl.Vector3Scale(lookDir, 3))

	radius := float32(0.5)

	sphere := engine.NewGameObject(fmt.Sprintf("Shot_%d", s.shotCounter))
	sphere.Transform.Position = spawnPos
	sphere.Transform.Scale = rl.Vector3{X: radius, Y: radius, Z: radius} // unit sphere has radius 1, scale to desired radius

	model := assets.GetSphereModel() // shared cached model
	renderer := components.NewModelRenderer(model, rl.Orange)
	renderer.MeshType = "sphere" // mark for instanced batching
	renderer.SetShader(s.GetGameObject().Scene.World.GetShader())
	sphere.AddComponent(renderer)

	sphere.AddComponent(components.NewSphereCollider(radius))

	rb := components.NewRigidbody()
	rb.Bounciness = 0.6
	rb.Friction = 0.1
	rb.Velocity = rl.Vector3Scale(lookDir, 30)
	sphere.AddComponent(rb)

	sphere.Start()
	s.GetGameObject().Scene.World.SpawnObject(sphere)
}
