package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Camera struct {
	engine.BaseComponent
	FOV        float32
	Near       float32
	Far        float32
	Projection rl.CameraProjection
}

func NewCamera() *Camera {
	return &Camera{
		FOV:        45.0,
		Near:       0.1,
		Far:        1000.0,
		Projection: rl.CameraPerspective,
	}
}

func (c *Camera) GetRaylibCamera() rl.Camera3D {
	g := c.GetGameObject()
	if g == nil {
		return rl.Camera3D{}
	}

	// Try to get FPSController for look direction
	fps := engine.GetComponent[*FPSController](g)

	var target rl.Vector3
	if fps != nil {
		lookDir := fps.GetLookDirection()
		target = rl.Vector3Add(g.Transform.Position, lookDir)
	} else {
		// Default: look forward along Z
		target = rl.Vector3Add(g.Transform.Position, rl.Vector3{X: 0, Y: 0, Z: 1})
	}

	return rl.Camera3D{
		Position:   g.Transform.Position,
		Target:     target,
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       c.FOV,
		Projection: c.Projection,
	}
}
