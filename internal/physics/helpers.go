package physics

import (
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// cross computes the cross product of two vectors
func cross(a, b rl.Vector3) rl.Vector3 {
	return rl.Vector3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// clamp restricts a value to a range
func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// estimateContactPoint estimates the contact point on an object's surface given a push direction
func estimateContactPoint(center rl.Vector3, halfSize rl.Vector3, pushDir rl.Vector3) rl.Vector3 {
	// Contact is on the face in the direction of the push
	// Use the push direction components scaled by half size
	contact := center
	contact.X -= pushDir.X * halfSize.X
	contact.Y -= pushDir.Y * halfSize.Y
	contact.Z -= pushDir.Z * halfSize.Z
	return contact
}

// applyBoxFlatteningTorque applies off-center gravity torque to naturally tip boxes toward flat faces
func applyBoxFlatteningTorque(obj *engine.GameObject, rb *components.Rigidbody, box *components.BoxCollider, deltaTime float32) {
	// Only apply when box is nearly at rest and grounded
	speed := rl.Vector3Length(rb.Velocity)
	if speed > 2.0 {
		return // Moving too fast, let physics play out naturally
	}
	if rb.Velocity.Y < -0.5 || rb.Velocity.Y > 0.5 {
		return // Not grounded (falling or jumping)
	}

	quat := obj.Transform.GetQuaternion()
	worldUp := rl.Vector3{X: 0, Y: 1, Z: 0}

	// Box's 6 face normals in local space
	localFaces := []rl.Vector3{
		{X: 0, Y: 1, Z: 0},  // Top
		{X: 0, Y: -1, Z: 0}, // Bottom
		{X: 1, Y: 0, Z: 0},  // Right
		{X: -1, Y: 0, Z: 0}, // Left
		{X: 0, Y: 0, Z: 1},  // Front
		{X: 0, Y: 0, Z: -1}, // Back
	}

	// Find which face is closest to pointing up
	var bestFaceLocal rl.Vector3
	bestDot := float32(-2.0)

	for _, localFace := range localFaces {
		worldFace := rl.Vector3RotateByQuaternion(localFace, quat)
		dot := rl.Vector3DotProduct(worldFace, worldUp)
		if dot > bestDot {
			bestDot = dot
			bestFaceLocal = localFace
		}
	}

	// If flat enough, we're done
	if bestDot > 0.995 {
		return
	}

	// Calculate the tipping torque from off-center gravity
	// When tilted, gravity acts at center of mass but support is at the contact edge
	// This creates a natural torque that tips the box
	bestFaceWorld := rl.Vector3RotateByQuaternion(bestFaceLocal, quat)

	// The torque axis is perpendicular to both the face normal and world up
	torqueAxis := rl.Vector3CrossProduct(bestFaceWorld, worldUp)
	axisLength := rl.Vector3Length(torqueAxis)

	if axisLength < 0.001 {
		return
	}

	torqueAxis = rl.Vector3Scale(torqueAxis, 1.0/axisLength)

	// Torque magnitude based on how tilted we are (sin of tilt angle)
	// More tilt = more torque, just like real physics
	tiltAmount := axisLength // This is sin(angle) from the cross product

	// Apply torque - gravity * lever arm * tilt
	// Increased multiplier to overcome angular damping
	gravityMag := float32(20.0) // Match world gravity
	leverArm := (box.Size.X + box.Size.Y + box.Size.Z) / 6.0
	torqueMag := gravityMag * leverArm * tiltAmount * 2.0

	// Apply as angular acceleration (in degrees)
	angularAccel := rl.Vector3Scale(torqueAxis, torqueMag*deltaTime*rl.Rad2deg)
	rb.AngularVelocity = rl.Vector3Add(rb.AngularVelocity, angularAccel)
}
