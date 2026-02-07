package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type BoxCollider struct {
	engine.BaseComponent
	Size   rl.Vector3
	Offset rl.Vector3
}

func NewBoxCollider(size rl.Vector3) *BoxCollider {
	return &BoxCollider{
		Size:   size,
		Offset: rl.Vector3{},
	}
}

// GetCenter returns the world-space center of this collider
func (b *BoxCollider) GetCenter() rl.Vector3 {
	g := b.GetGameObject()
	return rl.Vector3Add(g.WorldPosition(), b.Offset)
}
