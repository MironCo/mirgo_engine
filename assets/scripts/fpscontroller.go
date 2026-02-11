package scripts

import (
	"math"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// FPSController provides first-person camera controls with WASD movement and mouse look.
// Requires a CharacterController component on the same GameObject for collision/movement.
type FPSController struct {
	engine.BaseComponent
	Yaw          float32
	Pitch        float32
	MoveSpeed    float32
	LookSpeed    float32
	JumpStrength float32
	EyeHeight    float32

	// Cached CharacterController reference
	charController *components.CharacterController
}

func (f *FPSController) Start() {
	// Set defaults if not loaded from scene
	if f.MoveSpeed == 0 {
		f.MoveSpeed = 8.0
	}
	if f.LookSpeed == 0 {
		f.LookSpeed = 0.1
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

	// Cache CharacterController reference
	g := f.GetGameObject()
	if g != nil {
		f.charController = engine.GetComponent[*components.CharacterController](g)
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

	// Use CharacterController if available
	if f.charController != nil {
		// Build speed vector (CharacterController handles deltaTime internally in SimpleMove)
		speed := rl.Vector3{
			X: moveDirX * f.MoveSpeed,
			Y: 0,
			Z: moveDirZ * f.MoveSpeed,
		}

		// Jump
		if rl.IsKeyPressed(rl.KeySpace) && f.charController.IsGrounded() {
			f.charController.SetVelocityY(f.JumpStrength)
		}

		// Let CharacterController handle movement, collision, and gravity
		f.charController.SimpleMove(speed, deltaTime)
	} else {
		// Fallback: direct movement without collision (for backwards compatibility)
		g.Transform.Position.X += moveDirX * f.MoveSpeed * deltaTime
		g.Transform.Position.Z += moveDirZ * f.MoveSpeed * deltaTime
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
	if f.charController != nil {
		return f.charController.IsGrounded()
	}
	return false
}
