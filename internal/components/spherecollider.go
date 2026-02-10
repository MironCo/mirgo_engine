package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("SphereCollider", func() engine.Serializable {
		return NewSphereCollider(0.5)
	})
}

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

// TypeName implements engine.Serializable
func (s *SphereCollider) TypeName() string {
	return "SphereCollider"
}

// Serialize implements engine.Serializable
func (s *SphereCollider) Serialize() map[string]any {
	return map[string]any{
		"type":   "SphereCollider",
		"radius": s.Radius,
	}
}

// Deserialize implements engine.Serializable
func (s *SphereCollider) Deserialize(data map[string]any) {
	if r, ok := data["radius"].(float64); ok {
		s.Radius = float32(r)
	}
}
