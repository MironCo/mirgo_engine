package camera

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type FPSCamera struct {
	Position  rl.Vector3
	Yaw       float32
	Pitch     float32
	MoveSpeed float32
	LookSpeed float32
}

func New(pos rl.Vector3) *FPSCamera {
	return &FPSCamera{
		Position:  pos,
		Yaw:       -135.0,
		Pitch:     -30.0,
		MoveSpeed: 0.2,
		LookSpeed: 0.1,
	}
}

func (c *FPSCamera) Update() {
	// Mouse look
	mouseDelta := rl.GetMouseDelta()
	c.Yaw += mouseDelta.X * c.LookSpeed
	c.Pitch -= mouseDelta.Y * c.LookSpeed

	// Clamp pitch
	if c.Pitch > 89 {
		c.Pitch = 89
	}
	if c.Pitch < -89 {
		c.Pitch = -89
	}

	// Calculate movement vectors
	forward, right := c.getDirections()

	// WASD movement
	if rl.IsKeyDown(rl.KeyW) {
		c.Position.X += forward.X * c.MoveSpeed
		c.Position.Z += forward.Z * c.MoveSpeed
	}
	if rl.IsKeyDown(rl.KeyS) {
		c.Position.X -= forward.X * c.MoveSpeed
		c.Position.Z -= forward.Z * c.MoveSpeed
	}
	if rl.IsKeyDown(rl.KeyA) {
		c.Position.X += right.X * c.MoveSpeed
		c.Position.Z += right.Z * c.MoveSpeed
	}
	if rl.IsKeyDown(rl.KeyD) {
		c.Position.X -= right.X * c.MoveSpeed
		c.Position.Z -= right.Z * c.MoveSpeed
	}
	if rl.IsKeyDown(rl.KeySpace) {
		c.Position.Y += c.MoveSpeed
	}
	if rl.IsKeyDown(rl.KeyLeftShift) {
		c.Position.Y -= c.MoveSpeed
	}
}

func (c *FPSCamera) getDirections() (forward, right rl.Vector3) {
	yawRad := float64(c.Yaw) * math.Pi / 180
	forward = rl.Vector3{
		X: float32(math.Cos(yawRad)),
		Y: 0,
		Z: float32(math.Sin(yawRad)),
	}
	right = rl.Vector3{
		X: float32(math.Sin(yawRad)),
		Y: 0,
		Z: float32(-math.Cos(yawRad)),
	}
	return
}

func (c *FPSCamera) GetRaylibCamera() rl.Camera3D {
	yawRad := float64(c.Yaw) * math.Pi / 180
	pitchRad := float64(c.Pitch) * math.Pi / 180

	target := rl.Vector3{
		X: c.Position.X + float32(math.Cos(yawRad)*math.Cos(pitchRad)),
		Y: c.Position.Y + float32(math.Sin(pitchRad)),
		Z: c.Position.Z + float32(math.Sin(yawRad)*math.Cos(pitchRad)),
	}

	return rl.Camera3D{
		Position:   c.Position,
		Target:     target,
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}
}
