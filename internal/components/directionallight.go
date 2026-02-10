package components

import (
	"math"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("DirectionalLight", func() engine.Serializable {
		return NewDirectionalLight()
	})
}

type DirectionalLight struct {
	engine.BaseComponent
	Direction      rl.Vector3
	Color          rl.Color
	Intensity      float32
	AmbientColor   rl.Color
	ShadowDistance float32
}

func NewDirectionalLight() *DirectionalLight {
	return &DirectionalLight{
		Direction:      rl.Vector3Normalize(rl.Vector3{X: 0.35, Y: -1.0, Z: -0.35}),
		Color:          rl.White,
		Intensity:      1.0,
		AmbientColor:   rl.NewColor(25, 25, 25, 255),
		ShadowDistance: 50.0,
	}
}

// TypeName implements engine.Serializable
func (l *DirectionalLight) TypeName() string {
	return "DirectionalLight"
}

// Serialize implements engine.Serializable
func (l *DirectionalLight) Serialize() map[string]any {
	return map[string]any{
		"type":      "DirectionalLight",
		"direction": [3]float32{l.Direction.X, l.Direction.Y, l.Direction.Z},
		"intensity": l.Intensity,
	}
}

// Deserialize implements engine.Serializable
func (l *DirectionalLight) Deserialize(data map[string]any) {
	if dir, ok := data["direction"].([]any); ok && len(dir) == 3 {
		l.Direction.X = float32(dir[0].(float64))
		l.Direction.Y = float32(dir[1].(float64))
		l.Direction.Z = float32(dir[2].(float64))
	}
	if i, ok := data["intensity"].(float64); ok {
		l.Intensity = float32(i)
	}
}

func (l *DirectionalLight) GetLightCamera(orthoSize float32) rl.Camera3D {
	return rl.Camera3D{
		Position:   rl.Vector3Scale(l.Direction, -l.ShadowDistance),
		Target:     rl.Vector3Zero(),
		Up:         l.lightCameraUp(),
		Fovy:       orthoSize,
		Projection: rl.CameraOrthographic,
	}
}

func (l *DirectionalLight) MoveLightDir(dx, dy, dz float32) {
	l.Direction.X += dx
	l.Direction.Y += dy
	l.Direction.Z += dz
	l.Direction = rl.Vector3Normalize(l.Direction)
}

func (l *DirectionalLight) GetColorFloat() []float32 {
	return []float32{
		float32(l.Color.R) / 255.0 * l.Intensity,
		float32(l.Color.G) / 255.0 * l.Intensity,
		float32(l.Color.B) / 255.0 * l.Intensity,
		1.0,
	}
}

func (l *DirectionalLight) GetAmbientFloat() []float32 {
	return []float32{
		float32(l.AmbientColor.R) / 255.0,
		float32(l.AmbientColor.G) / 255.0,
		float32(l.AmbientColor.B) / 255.0,
		1.0,
	}
}

func (l *DirectionalLight) lightCameraUp() rl.Vector3 {
	if math.Abs(float64(l.Direction.Y)) > 0.9 {
		return rl.Vector3{X: 0, Y: 0, Z: 1}
	}
	return rl.Vector3{X: 0, Y: 1, Z: 0}
}
