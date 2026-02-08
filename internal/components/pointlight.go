package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type PointLight struct {
	engine.BaseComponent
	Color     rl.Color
	Intensity float32
	Radius    float32 // falloff distance
}

func NewPointLight() *PointLight {
	return &PointLight{
		Color:     rl.White,
		Intensity: 1.0,
		Radius:    10.0,
	}
}

func (p *PointLight) GetPosition() rl.Vector3 {
	if g := p.GetGameObject(); g != nil {
		return g.WorldPosition()
	}
	return rl.Vector3Zero()
}

func (p *PointLight) GetColorFloat() []float32 {
	return []float32{
		float32(p.Color.R) / 255.0 * p.Intensity,
		float32(p.Color.G) / 255.0 * p.Intensity,
		float32(p.Color.B) / 255.0 * p.Intensity,
	}
}
