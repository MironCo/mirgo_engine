package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type SphereCollider struct {
	engine.BaseComponent
	Radius float32
	Offset rl.Vector3
}

func NewSphereCollider(radius float32) *SphereCollider {
	return &SphereCollider{
		Radius: radius,
		Offset: rl.Vector3{},
	}
}

// GetCenter returns the world-space center of this collider
func (s *SphereCollider) GetCenter() rl.Vector3 {
	g := s.GetGameObject()
	return rl.Vector3Add(g.WorldPosition(), s.Offset)
}
