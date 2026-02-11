package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("CharacterController", func() engine.Serializable {
		return NewCharacterController()
	})
}

// CharacterController handles character movement with collision detection,
// gravity, and stair stepping. Similar to Unity's CharacterController.
type CharacterController struct {
	engine.BaseComponent

	// Configuration
	Height     float32 // Total height of the capsule/box
	Radius     float32 // Radius (half-width) of the character
	StepHeight float32 // Max height of steps to climb
	SlopeLimit float32 // Max slope angle in degrees (not implemented yet)

	// Gravity
	UseGravity bool
	Gravity    float32 // Gravity strength (positive = down)

	// Runtime state (not serialized)
	velocity   rl.Vector3
	isGrounded bool
}

// TypeName implements engine.Serializable
func (c *CharacterController) TypeName() string {
	return "CharacterController"
}

// NewCharacterController creates a new character controller with defaults
func NewCharacterController() *CharacterController {
	return &CharacterController{
		Height:     1.8,
		Radius:     0.4,
		StepHeight: 0.4,
		SlopeLimit: 45.0,
		UseGravity: true,
		Gravity:    20.0,
		isGrounded: false,
	}
}

// Serialize implements engine.Serializable
func (c *CharacterController) Serialize() map[string]any {
	return map[string]any{
		"type":       "CharacterController",
		"height":     c.Height,
		"radius":     c.Radius,
		"stepHeight": c.StepHeight,
		"slopeLimit": c.SlopeLimit,
		"useGravity": c.UseGravity,
		"gravity":    c.Gravity,
	}
}

// Deserialize implements engine.Serializable
func (c *CharacterController) Deserialize(data map[string]any) {
	if v, ok := data["height"].(float64); ok {
		c.Height = float32(v)
	}
	if v, ok := data["radius"].(float64); ok {
		c.Radius = float32(v)
	}
	if v, ok := data["stepHeight"].(float64); ok {
		c.StepHeight = float32(v)
	}
	if v, ok := data["slopeLimit"].(float64); ok {
		c.SlopeLimit = float32(v)
	}
	if v, ok := data["useGravity"].(bool); ok {
		c.UseGravity = v
	}
	if v, ok := data["gravity"].(float64); ok {
		c.Gravity = float32(v)
	}
}

// Move moves the character by the given motion vector, handling collisions and steps
// Returns the actual displacement after collision resolution
func (c *CharacterController) Move(motion rl.Vector3) rl.Vector3 {
	g := c.GetGameObject()
	if g == nil {
		return rl.Vector3{}
	}

	// Get collidable objects from the scene's world
	var colliders []*engine.GameObject
	if g.Scene != nil && g.Scene.World != nil {
		colliders = g.Scene.World.GetCollidableObjects()
	}

	if len(colliders) == 0 {
		// No colliders, just move directly
		g.Transform.Position = rl.Vector3Add(g.Transform.Position, motion)
		return motion
	}

	originalPos := g.Transform.Position

	// Try to move horizontally first
	horizontalMotion := rl.Vector3{X: motion.X, Y: 0, Z: motion.Z}
	if horizontalMotion.X != 0 || horizontalMotion.Z != 0 {
		c.moveWithCollision(g, horizontalMotion, colliders)
	}

	// Then move vertically
	verticalMotion := rl.Vector3{X: 0, Y: motion.Y, Z: 0}
	if verticalMotion.Y != 0 {
		c.moveWithCollision(g, verticalMotion, colliders)
	}

	actualMotion := rl.Vector3Subtract(g.Transform.Position, originalPos)
	return actualMotion
}

// moveWithCollision attempts to move and handles collision/stepping
func (c *CharacterController) moveWithCollision(g *engine.GameObject, motion rl.Vector3, colliders []*engine.GameObject) {
	// Move to target
	g.Transform.Position = rl.Vector3Add(g.Transform.Position, motion)

	halfHeight := c.Height / 2
	halfWidth := c.Radius

	// Create character's bounding box
	charMin := rl.Vector3{
		X: g.Transform.Position.X - halfWidth,
		Y: g.Transform.Position.Y - halfHeight,
		Z: g.Transform.Position.Z - halfWidth,
	}
	charMax := rl.Vector3{
		X: g.Transform.Position.X + halfWidth,
		Y: g.Transform.Position.Y + halfHeight,
		Z: g.Transform.Position.Z + halfWidth,
	}

	for _, other := range colliders {
		// Skip self
		if other == g {
			continue
		}

		// Skip objects without rigidbody that have kinematic rigidbodies (other players)
		rb := engine.GetComponent[*Rigidbody](other)
		if rb != nil && rb.IsKinematic {
			continue // Don't collide with other kinematic objects
		}

		boxCol := engine.GetComponent[*BoxCollider](other)
		if boxCol == nil {
			continue
		}

		// Get static's bounding box
		staticCenter := boxCol.GetCenter()
		staticHalfSize := rl.Vector3Scale(boxCol.GetWorldSize(), 0.5)
		staticMin := rl.Vector3Subtract(staticCenter, staticHalfSize)
		staticMax := rl.Vector3Add(staticCenter, staticHalfSize)

		// Check AABB overlap
		if !aabbOverlap(charMin, charMax, staticMin, staticMax) {
			continue
		}

		// Calculate push-out vector
		pushOut := calculatePushOut(charMin, charMax, staticMin, staticMax)

		// Check if this is a step we can climb
		isHorizontalCollision := (pushOut.X != 0 || pushOut.Z != 0) && pushOut.Y == 0

		if isHorizontalCollision && motion.Y == 0 {
			// Check if we can step up
			charFeetY := g.Transform.Position.Y - halfHeight
			stepTopY := staticMax.Y
			stepHeight := stepTopY - charFeetY

			if stepHeight > 0 && stepHeight <= c.StepHeight {
				// Try stepping up
				testY := g.Transform.Position.Y + stepHeight + 0.01
				testMin := rl.Vector3{
					X: g.Transform.Position.X - halfWidth,
					Y: testY - halfHeight,
					Z: g.Transform.Position.Z - halfWidth,
				}
				testMax := rl.Vector3{
					X: g.Transform.Position.X + halfWidth,
					Y: testY + halfHeight,
					Z: g.Transform.Position.Z + halfWidth,
				}

				// Check if stepped position is clear
				if !aabbOverlap(testMin, testMax, staticMin, staticMax) {
					// Step up!
					g.Transform.Position.Y = testY
					c.isGrounded = true

					// Update char bounds after stepping
					charMin.Y = g.Transform.Position.Y - halfHeight
					charMax.Y = g.Transform.Position.Y + halfHeight
					continue
				}
			}
		}

		// Apply push-out
		g.Transform.Position = rl.Vector3Add(g.Transform.Position, pushOut)

		// Update char bounds after push
		charMin = rl.Vector3{
			X: g.Transform.Position.X - halfWidth,
			Y: g.Transform.Position.Y - halfHeight,
			Z: g.Transform.Position.Z - halfWidth,
		}
		charMax = rl.Vector3{
			X: g.Transform.Position.X + halfWidth,
			Y: g.Transform.Position.Y + halfHeight,
			Z: g.Transform.Position.Z + halfWidth,
		}

		// Update grounded state
		if pushOut.Y > 0 {
			c.isGrounded = true
			c.velocity.Y = 0
		}
	}
}

// SimpleMove moves the character with gravity applied automatically
func (c *CharacterController) SimpleMove(speed rl.Vector3, deltaTime float32) {
	// Apply gravity (only if not grounded, or if we have upward velocity like a jump)
	if c.UseGravity {
		if !c.isGrounded || c.velocity.Y > 0 {
			c.velocity.Y -= c.Gravity * deltaTime
		} else {
			// Grounded and not jumping - keep small downward velocity to detect ground
			c.velocity.Y = -0.1
		}
	}

	// Build motion from input speed (horizontal) and gravity (vertical)
	motion := rl.Vector3{
		X: speed.X * deltaTime,
		Y: c.velocity.Y * deltaTime,
		Z: speed.Z * deltaTime,
	}

	// Reset grounded before move (will be set if we land)
	c.isGrounded = false

	c.Move(motion)
}

// IsGrounded returns whether the character is on the ground
func (c *CharacterController) IsGrounded() bool {
	return c.isGrounded
}

// SetGrounded manually sets the grounded state
func (c *CharacterController) SetGrounded(grounded bool) {
	c.isGrounded = grounded
}

// GetVelocity returns the current velocity
func (c *CharacterController) GetVelocity() rl.Vector3 {
	return c.velocity
}

// SetVelocityY sets the vertical velocity (for jumping)
func (c *CharacterController) SetVelocityY(vy float32) {
	c.velocity.Y = vy
}

// Helper: check if two AABBs overlap
func aabbOverlap(aMin, aMax, bMin, bMax rl.Vector3) bool {
	return aMin.X <= bMax.X && aMax.X >= bMin.X &&
		aMin.Y <= bMax.Y && aMax.Y >= bMin.Y &&
		aMin.Z <= bMax.Z && aMax.Z >= bMin.Z
}

// Helper: calculate minimum push-out vector between two AABBs
func calculatePushOut(aMin, aMax, bMin, bMax rl.Vector3) rl.Vector3 {
	// Calculate overlap on each axis
	overlapX1 := aMax.X - bMin.X // A pushing left
	overlapX2 := bMax.X - aMin.X // A pushing right
	overlapY1 := aMax.Y - bMin.Y // A pushing down
	overlapY2 := bMax.Y - aMin.Y // A pushing up
	overlapZ1 := aMax.Z - bMin.Z // A pushing back
	overlapZ2 := bMax.Z - aMin.Z // A pushing forward

	// Find minimum overlap on each axis
	var pushX, pushY, pushZ float32

	if overlapX1 < overlapX2 {
		pushX = -overlapX1
	} else {
		pushX = overlapX2
	}

	if overlapY1 < overlapY2 {
		pushY = -overlapY1
	} else {
		pushY = overlapY2
	}

	if overlapZ1 < overlapZ2 {
		pushZ = -overlapZ1
	} else {
		pushZ = overlapZ2
	}

	// Return push on axis with smallest overlap
	absX := pushX
	if absX < 0 {
		absX = -absX
	}
	absY := pushY
	if absY < 0 {
		absY = -absY
	}
	absZ := pushZ
	if absZ < 0 {
		absZ = -absZ
	}

	if absX <= absY && absX <= absZ {
		return rl.Vector3{X: pushX, Y: 0, Z: 0}
	} else if absY <= absX && absY <= absZ {
		return rl.Vector3{X: 0, Y: pushY, Z: 0}
	} else {
		return rl.Vector3{X: 0, Y: 0, Z: pushZ}
	}
}
