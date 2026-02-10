package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("PointLight", func() engine.Serializable {
		return NewPointLight()
	})
}

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

// TypeName implements engine.Serializable
func (p *PointLight) TypeName() string {
	return "PointLight"
}

// Serialize implements engine.Serializable
func (p *PointLight) Serialize() map[string]any {
	return map[string]any{
		"type":      "PointLight",
		"color":     [3]uint8{p.Color.R, p.Color.G, p.Color.B},
		"intensity": p.Intensity,
		"radius":    p.Radius,
	}
}

// Deserialize implements engine.Serializable
func (p *PointLight) Deserialize(data map[string]any) {
	if c, ok := data["color"].([]any); ok && len(c) == 3 {
		p.Color.R = uint8(c[0].(float64))
		p.Color.G = uint8(c[1].(float64))
		p.Color.B = uint8(c[2].(float64))
		p.Color.A = 255
	}
	if i, ok := data["intensity"].(float64); ok {
		p.Intensity = float32(i)
	}
	if r, ok := data["radius"].(float64); ok {
		p.Radius = float32(r)
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
