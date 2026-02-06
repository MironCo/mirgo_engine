package components

import (
	"test3d/internal/engine"
	"test3d/internal/physics"

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

func (b *BoxCollider) GetAABB() physics.AABB {
	g := b.GetGameObject()
	center := rl.Vector3Add(g.Transform.Position, b.Offset)
	return physics.NewAABBFromCenter(center, b.Size)
}
