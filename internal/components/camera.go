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
	IsMain     bool // If true, this is the active game camera
}

func NewCamera() *Camera {
	return &Camera{
		FOV:        45.0,
		Near:       0.1,
		Far:        1000.0,
		Projection: rl.CameraPerspective,
		IsMain:     false,
	}
}

func (c *Camera) GetRaylibCamera() rl.Camera3D {
	g := c.GetGameObject()
	if g == nil {
		return rl.Camera3D{}
	}

	// Get eye position (feet + eye height)
	eyePos := g.WorldPosition()

	// Look for any LookProvider component (FPSController, etc.)
	lookProvider := engine.FindComponent[engine.LookProvider](g)

	if lookProvider != nil {
		eyePos.Y += lookProvider.GetEyeHeight()
	}

	var target rl.Vector3
	if lookProvider != nil {
		x, y, z := lookProvider.GetLookDirection()
		lookDir := rl.Vector3{X: x, Y: y, Z: z}
		target = rl.Vector3Add(eyePos, lookDir)
	} else {
		// Default: look forward along Z
		target = rl.Vector3Add(eyePos, rl.Vector3{X: 0, Y: 0, Z: 1})
	}

	return rl.Camera3D{
		Position:   eyePos,
		Target:     target,
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       c.FOV,
		Projection: c.Projection,
	}
}
