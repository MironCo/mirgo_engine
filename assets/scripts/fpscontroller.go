package scripts

import (
	"math"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// FPSController provides first-person camera controls with WASD movement and mouse look.
type FPSController struct {
	engine.BaseComponent
	Yaw          float32
	Pitch        float32
	MoveSpeed    float32
	LookSpeed    float32
	Gravity      float32
	JumpStrength float32
	EyeHeight    float32
	// Velocity components (not serialized - runtime state)
	velocityX float32
	velocityY float32
	velocityZ float32
	grounded  bool
}

func (f *FPSController) Start() {
	// Set defaults if not loaded from scene
	if f.MoveSpeed == 0 {
		f.MoveSpeed = 8.0
	}
	if f.LookSpeed == 0 {
		f.LookSpeed = 0.1
	}
	if f.Gravity == 0 {
		f.Gravity = 20.0
	}
	if f.JumpStrength == 0 {
		f.JumpStrength = 8.0
	}
	if f.EyeHeight == 0 {
		f.EyeHeight = 1.6
	}
	if f.Yaw == 0 && f.Pitch == 0 {
		f.Yaw = -135.0
		f.Pitch = -30.0
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
	var moveDirX, moveDirZ float32
	if rl.IsKeyDown(rl.KeyW) {
		moveDirX += forward.X
		moveDirZ += forward.Z
	}
	if rl.IsKeyDown(rl.KeyS) {
		moveDirX -= forward.X
		moveDirZ -= forward.Z
	}
	if rl.IsKeyDown(rl.KeyA) {
		moveDirX += right.X
		moveDirZ += right.Z
	}
	if rl.IsKeyDown(rl.KeyD) {
		moveDirX -= right.X
		moveDirZ -= right.Z
	}

	// Normalize diagonal movement
	moveLen := float32(math.Sqrt(float64(moveDirX*moveDirX + moveDirZ*moveDirZ)))
	if moveLen > 0 {
		moveDirX /= moveLen
		moveDirZ /= moveLen
	}

	// Apply horizontal velocity
	f.velocityX = moveDirX * f.MoveSpeed
	f.velocityZ = moveDirZ * f.MoveSpeed

	// Jump
	if rl.IsKeyPressed(rl.KeySpace) && f.grounded {
		f.velocityY = f.JumpStrength
		f.grounded = false
	}

	// Apply gravity
	if !f.grounded {
		f.velocityY -= f.Gravity * deltaTime
	}

	// Update position
	g.Transform.Position.X += f.velocityX * deltaTime
	g.Transform.Position.Y += f.velocityY * deltaTime
	g.Transform.Position.Z += f.velocityZ * deltaTime

	// Simple ground check - floor is at Y=0
	feetY := g.Transform.Position.Y - f.EyeHeight
	if feetY <= 0 {
		g.Transform.Position.Y = f.EyeHeight
		f.velocityY = 0
		f.grounded = true
	} else {
		f.grounded = false
	}
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

// GetLookDirection implements engine.LookProvider
func (f *FPSController) GetLookDirection() (x, y, z float32) {
	yawRad := float64(f.Yaw) * math.Pi / 180
	pitchRad := float64(f.Pitch) * math.Pi / 180
	return float32(math.Cos(yawRad) * math.Cos(pitchRad)),
		float32(math.Sin(pitchRad)),
		float32(math.Sin(yawRad) * math.Cos(pitchRad))
}

// GetEyeHeight implements engine.LookProvider
func (f *FPSController) GetEyeHeight() float32 {
	return f.EyeHeight
}

// Grounded returns whether the controller is on the ground
func (f *FPSController) Grounded() bool {
	return f.grounded
}

// SetGrounded sets whether the controller is on the ground
func (f *FPSController) SetGrounded(grounded bool) {
	f.grounded = grounded
}

// SetVelocityY sets the vertical velocity (used by physics)
func (f *FPSController) SetVelocityY(vy float32) {
	f.velocityY = vy
}

// GetVelocity returns the current velocity
func (f *FPSController) GetVelocity() (x, y, z float32) {
	return f.velocityX, f.velocityY, f.velocityZ
}
