package camera

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type FPSCamera struct {
	Position  rl.Vector3
	Velocity  rl.Vector3
	Yaw       float32
	Pitch     float32
	MoveSpeed float32
	LookSpeed float32

	// Physics
	Gravity      float32
	JumpStrength float32
	Grounded     bool
	EyeHeight    float32 // Height of camera above feet
}

func New(pos rl.Vector3) *FPSCamera {
	return &FPSCamera{
		Position:     pos,
		Velocity:     rl.Vector3{},
		Yaw:          -135.0,
		Pitch:        -30.0,
		MoveSpeed:    8.0,  // Units per second
		LookSpeed:    0.1,
		Gravity:      20.0, // Units per second squared
		JumpStrength: 8.0,  // Initial upward velocity
		Grounded:     false,
		EyeHeight:    5.0,
	}
}

func (c *FPSCamera) Update(deltaTime float32) {
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

	// Calculate movement vectors (horizontal plane only)
	forward, right := c.getDirections()

	// Build horizontal movement from input
	var moveDir rl.Vector3
	if rl.IsKeyDown(rl.KeyW) {
		moveDir.X += forward.X
		moveDir.Z += forward.Z
	}
	if rl.IsKeyDown(rl.KeyS) {
		moveDir.X -= forward.X
		moveDir.Z -= forward.Z
	}
	if rl.IsKeyDown(rl.KeyA) {
		moveDir.X += right.X
		moveDir.Z += right.Z
	}
	if rl.IsKeyDown(rl.KeyD) {
		moveDir.X -= right.X
		moveDir.Z -= right.Z
	}

	// Normalize diagonal movement so you don't go faster diagonally
	moveLen := float32(math.Sqrt(float64(moveDir.X*moveDir.X + moveDir.Z*moveDir.Z)))
	if moveLen > 0 {
		moveDir.X /= moveLen
		moveDir.Z /= moveLen
	}

	// Apply horizontal velocity
	c.Velocity.X = moveDir.X * c.MoveSpeed
	c.Velocity.Z = moveDir.Z * c.MoveSpeed

	// Jump
	if rl.IsKeyPressed(rl.KeySpace) && c.Grounded {
		c.Velocity.Y = c.JumpStrength
		c.Grounded = false
	}

	// Apply gravity
	if !c.Grounded {
		c.Velocity.Y -= c.Gravity * deltaTime
	}

	// Update position
	c.Position.X += c.Velocity.X * deltaTime
	c.Position.Y += c.Velocity.Y * deltaTime
	c.Position.Z += c.Velocity.Z * deltaTime
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
