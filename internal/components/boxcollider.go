package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("BoxCollider", func() engine.Serializable {
		return NewBoxCollider(rl.Vector3{X: 1, Y: 1, Z: 1})
	})
}

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
	scale := g.WorldScale()
	// Scale the offset by the object's scale
	scaledOffset := rl.Vector3{
		X: b.Offset.X * scale.X,
		Y: b.Offset.Y * scale.Y,
		Z: b.Offset.Z * scale.Z,
	}
	return rl.Vector3Add(g.WorldPosition(), scaledOffset)
}

// GetWorldSize returns the collider size scaled by the object's transform
func (b *BoxCollider) GetWorldSize() rl.Vector3 {
	g := b.GetGameObject()
	scale := g.WorldScale()
	return rl.Vector3{
		X: b.Size.X * scale.X,
		Y: b.Size.Y * scale.Y,
		Z: b.Size.Z * scale.Z,
	}
}

// TypeName implements engine.Serializable
func (b *BoxCollider) TypeName() string {
	return "BoxCollider"
}

// Serialize implements engine.Serializable
func (b *BoxCollider) Serialize() map[string]any {
	return map[string]any{
		"type":   "BoxCollider",
		"size":   [3]float32{b.Size.X, b.Size.Y, b.Size.Z},
		"offset": [3]float32{b.Offset.X, b.Offset.Y, b.Offset.Z},
	}
}

// Deserialize implements engine.Serializable
func (b *BoxCollider) Deserialize(data map[string]any) {
	if size, ok := data["size"].([]any); ok && len(size) == 3 {
		b.Size.X = float32(size[0].(float64))
		b.Size.Y = float32(size[1].(float64))
		b.Size.Z = float32(size[2].(float64))
	}
	if offset, ok := data["offset"].([]any); ok && len(offset) == 3 {
		b.Offset.X = float32(offset[0].(float64))
		b.Offset.Y = float32(offset[1].(float64))
		b.Offset.Z = float32(offset[2].(float64))
	}
}
