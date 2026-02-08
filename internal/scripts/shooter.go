package scripts

import (
	"fmt"
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
	fps := engine.GetComponent[*components.FPSController](g)
	if fps == nil {
		return
	}

	origin := g.Transform.Position
	origin.Y += fps.EyeHeight
	direction := fps.GetLookDirection()

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
	fps := engine.GetComponent[*components.FPSController](g)
	if fps == nil {
		return
	}

	s.shotCounter++

	eyePos := g.Transform.Position
	eyePos.Y += fps.EyeHeight
	lookDir := fps.GetLookDirection()
	spawnPos := rl.Vector3Add(eyePos, rl.Vector3Scale(lookDir, 3))

	radius := float32(0.5)

	sphere := engine.NewGameObject(fmt.Sprintf("Shot_%d", s.shotCounter))
	sphere.Transform.Position = spawnPos

	mesh := rl.GenMeshSphere(radius, 16, 16)
	model := rl.LoadModelFromMesh(mesh)
	renderer := components.NewModelRenderer(model, rl.Orange)
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

func init() {
	engine.RegisterScript("Shooter", shooterFactory, shooterSerializer)
}

func shooterFactory(props map[string]any) engine.Component {
	cooldown := 0.15
	if v, ok := props["cooldown"].(float64); ok {
		cooldown = v
	}
	return &Shooter{Cooldown: cooldown}
}

func shooterSerializer(c engine.Component) map[string]any {
	s, ok := c.(*Shooter)
	if !ok {
		return nil
	}
	return map[string]any{
		"cooldown": s.Cooldown,
	}
}
