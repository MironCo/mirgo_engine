package world

import (
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/physics"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type PlayerCollision struct {
	engine.BaseComponent
}

func (p *PlayerCollision) Update(deltaTime float32) {
	g := p.GetGameObject()
	if g == nil {
		return
	}

	fps := engine.GetComponent[*components.FPSController](g)
	collider := engine.GetComponent[*components.BoxCollider](g)
	rb := engine.GetComponent[*components.Rigidbody](g)

	if fps == nil || collider == nil {
		return
	}

	// Sync FPSController velocity to rigidbody for physics pushing
	if rb != nil {
		rb.Velocity = fps.Velocity
	}

	// Ground check - floor is at Y=0
	floorY := float32(0.0)
	feetY := g.Transform.Position.Y - fps.EyeHeight
	if feetY <= floorY {
		g.Transform.Position.Y = floorY + fps.EyeHeight
		fps.Velocity.Y = 0
		fps.Grounded = true
	} else {
		fps.Grounded = false
	}

	// Collision with world objects - use OBB for rotated box support
	playerCenter := rl.Vector3{
		X: g.Transform.Position.X,
		Y: g.Transform.Position.Y - fps.EyeHeight + collider.Size.Y/2,
		Z: g.Transform.Position.Z,
	}
	playerOBB := physics.NewAABBasOBB(playerCenter, collider.Size)

	for _, obj := range p.GetGameObject().Scene.World.GetCollidableObjects() {
		if obj == g {
			continue
		}

		objCollider := engine.GetComponent[*components.BoxCollider](obj)
		if objCollider == nil {
			continue
		}

		// Use OBB for rotated box collision
		objOBB := physics.NewOBBFromBox(objCollider.GetCenter(), objCollider.Size, obj.WorldRotation(), obj.WorldScale())
		pushOut := playerOBB.ResolveOBB(objOBB)

		if pushOut.X != 0 || pushOut.Y != 0 || pushOut.Z != 0 {
			g.Transform.Position = rl.Vector3Add(g.Transform.Position, pushOut)

			if pushOut.Y > 0 {
				fps.Velocity.Y = 0
				fps.Grounded = true
			}
			if pushOut.Y < 0 && fps.Velocity.Y > 0 {
				fps.Velocity.Y = 0
			}

			// Update OBB for subsequent checks
			playerOBB = physics.NewAABBasOBB(
				rl.Vector3{
					X: g.Transform.Position.X,
					Y: g.Transform.Position.Y - fps.EyeHeight + collider.Size.Y/2,
					Z: g.Transform.Position.Z,
				},
				collider.Size,
			)
		}
	}
}
