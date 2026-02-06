package components

import (
	"math"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type FPSController struct {
	engine.BaseComponent
	Yaw          float32
	Pitch        float32
	MoveSpeed    float32
	LookSpeed    float32
	Velocity     rl.Vector3
	Gravity      float32
	JumpStrength float32
	Grounded     bool
	EyeHeight    float32
}

func NewFPSController() *FPSController {
	return &FPSController{
		Yaw:          -135.0,
		Pitch:        -30.0,
		MoveSpeed:    8.0,
		LookSpeed:    0.1,
		Gravity:      20.0,
		JumpStrength: 8.0,
		Grounded:     false,
		EyeHeight:    5.0,
	}
}

func (f *FPSController) Update(deltaTime float32) {
	g := f.GetGameObject()
	if g == nil {
		return
	}

	// Mouse look
	mouseDelta := rl.GetMouseDelta()
	f.Yaw += mouseDelta.X * f.LookSpeed
	f.Pitch -= mouseDelta.Y * f.LookSpeed

	// Clamp pitch
	if f.Pitch > 89 {
		f.Pitch = 89
	}
	if f.Pitch < -89 {
		f.Pitch = -89
	}

	// Calculate movement vectors (horizontal plane only)
	forward, right := f.getDirections()

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

	// Normalize diagonal movement
	moveLen := float32(math.Sqrt(float64(moveDir.X*moveDir.X + moveDir.Z*moveDir.Z)))
	if moveLen > 0 {
		moveDir.X /= moveLen
		moveDir.Z /= moveLen
	}

	// Apply horizontal velocity
	f.Velocity.X = moveDir.X * f.MoveSpeed
	f.Velocity.Z = moveDir.Z * f.MoveSpeed

	// Jump
	if rl.IsKeyPressed(rl.KeySpace) && f.Grounded {
		f.Velocity.Y = f.JumpStrength
		f.Grounded = false
	}

	// Apply gravity
	if !f.Grounded {
		f.Velocity.Y -= f.Gravity * deltaTime
	}

	// Update position
	g.Transform.Position.X += f.Velocity.X * deltaTime
	g.Transform.Position.Y += f.Velocity.Y * deltaTime
	g.Transform.Position.Z += f.Velocity.Z * deltaTime
}

func (f *FPSController) getDirections() (forward, right rl.Vector3) {
	yawRad := float64(f.Yaw) * math.Pi / 180
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

func (f *FPSController) GetLookDirection() rl.Vector3 {
	yawRad := float64(f.Yaw) * math.Pi / 180
	pitchRad := float64(f.Pitch) * math.Pi / 180
	return rl.Vector3{
		X: float32(math.Cos(yawRad) * math.Cos(pitchRad)),
		Y: float32(math.Sin(pitchRad)),
		Z: float32(math.Sin(yawRad) * math.Cos(pitchRad)),
	}
}
