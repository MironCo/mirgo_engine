package physics

import (
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// resolveCollision handles collision between two dynamic rigidbodies
func (p *PhysicsWorld) resolveCollision(a, b *engine.GameObject) {
	rbA := engine.GetComponent[*components.Rigidbody](a)
	rbB := engine.GetComponent[*components.Rigidbody](b)
	if rbA == nil || rbB == nil {
		return
	}

	// Skip if both objects are sleeping
	if rbA.IsSleeping && rbB.IsSleeping {
		return
	}

	// Check for sphere colliders first
	sphereA := engine.GetComponent[*components.SphereCollider](a)
	sphereB := engine.GetComponent[*components.SphereCollider](b)

	// Sphere vs Sphere
	if sphereA != nil && sphereB != nil {
		p.resolveSphereVsSphere(a, b, rbA, rbB, sphereA, sphereB)
		return
	}

	// Sphere vs Box
	boxA := engine.GetComponent[*components.BoxCollider](a)
	boxB := engine.GetComponent[*components.BoxCollider](b)

	if sphereA != nil && boxB != nil {
		p.resolveSphereVsBox(a, b, rbA, rbB, sphereA, boxB)
		return
	}
	if boxA != nil && sphereB != nil {
		p.resolveSphereVsBox(b, a, rbB, rbA, sphereB, boxA)
		return
	}

	// Box vs Box - use OBB for rotated collision
	if boxA == nil || boxB == nil {
		return
	}

	obbA := NewOBBFromBox(boxA.GetCenter(), boxA.Size, a.WorldRotation(), a.WorldScale())
	obbB := NewOBBFromBox(boxB.GetCenter(), boxB.Size, b.WorldRotation(), b.WorldScale())

	pushOut := obbA.ResolveOBB(obbB)
	if pushOut.X == 0 && pushOut.Y == 0 && pushOut.Z == 0 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(a, b)

	// Split the push based on mass ratio
	totalMass := rbA.Mass + rbB.Mass
	ratioA := rbB.Mass / totalMass
	ratioB := rbA.Mass / totalMass

	a.Transform.Position = rl.Vector3Add(a.Transform.Position, rl.Vector3Scale(pushOut, ratioA))
	b.Transform.Position = rl.Vector3Subtract(b.Transform.Position, rl.Vector3Scale(pushOut, ratioB))

	// Bounce velocities
	pushLen := rl.Vector3Length(pushOut)
	if pushLen < 0.0001 {
		return
	}
	normal := rl.Vector3Scale(pushOut, 1/pushLen)

	// Relative velocity
	relVel := rl.Vector3Subtract(rbA.Velocity, rbB.Velocity)
	velAlongNormal := rl.Vector3DotProduct(relVel, normal)

	// Only resolve if objects are moving toward each other
	if velAlongNormal > 0 {
		return
	}

	// Restitution (bounciness)
	e := (rbA.Bounciness + rbB.Bounciness) / 2

	// Impulse magnitude
	j := -(1 + e) * velAlongNormal
	j /= (1/rbA.Mass + 1/rbB.Mass)

	// Apply impulse
	impulse := rl.Vector3Scale(normal, j)
	rbA.Velocity = rl.Vector3Add(rbA.Velocity, rl.Vector3Scale(impulse, 1/rbA.Mass))
	rbB.Velocity = rl.Vector3Subtract(rbB.Velocity, rl.Vector3Scale(impulse, 1/rbB.Mass))

	// Apply torque to boxes - contact point is on surface in direction of normal
	halfSizeA := rl.Vector3{X: boxA.Size.X / 2, Y: boxA.Size.Y / 2, Z: boxA.Size.Z / 2}
	halfSizeB := rl.Vector3{X: boxB.Size.X / 2, Y: boxB.Size.Y / 2, Z: boxB.Size.Z / 2}
	rA := estimateContactPoint(rl.Vector3{}, halfSizeA, rl.Vector3Scale(normal, -1))
	rB := estimateContactPoint(rl.Vector3{}, halfSizeB, normal)

	// Convert to degrees and scale up significantly
	torqueScale := float32(500.0)
	torqueA := cross(rA, impulse)
	torqueB := cross(rB, rl.Vector3Scale(impulse, -1))

	rbA.AngularVelocity = rl.Vector3Add(rbA.AngularVelocity, rl.Vector3Scale(torqueA, torqueScale/rbA.Mass))
	rbB.AngularVelocity = rl.Vector3Add(rbB.AngularVelocity, rl.Vector3Scale(torqueB, torqueScale/rbB.Mass))
}

// resolveSphereVsSphere handles collision between two spheres
func (p *PhysicsWorld) resolveSphereVsSphere(a, b *engine.GameObject, rbA, rbB *components.Rigidbody, sA, sB *components.SphereCollider) {
	diff := rl.Vector3Subtract(a.Transform.Position, b.Transform.Position)
	dist := rl.Vector3Length(diff)
	minDist := sA.Radius + sB.Radius

	if dist >= minDist || dist < 0.0001 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(a, b)

	// Collision normal
	normal := rl.Vector3Scale(diff, 1/dist)
	penetration := minDist - dist

	// Split push based on mass
	totalMass := rbA.Mass + rbB.Mass
	ratioA := rbB.Mass / totalMass
	ratioB := rbA.Mass / totalMass

	a.Transform.Position = rl.Vector3Add(a.Transform.Position, rl.Vector3Scale(normal, penetration*ratioA))
	b.Transform.Position = rl.Vector3Subtract(b.Transform.Position, rl.Vector3Scale(normal, penetration*ratioB))

	// Relative velocity
	relVel := rl.Vector3Subtract(rbA.Velocity, rbB.Velocity)
	velAlongNormal := rl.Vector3DotProduct(relVel, normal)

	// Static friction and normal force - if objects are nearly at rest and stacked
	relSpeed := rl.Vector3Length(relVel)
	if relSpeed < 0.5 && penetration < 0.1 {
		// Apply strong damping to settle stacked objects
		friction := (rbA.Friction + rbB.Friction) / 2
		rbA.Velocity = rl.Vector3Scale(rbA.Velocity, 1.0-friction)
		rbB.Velocity = rl.Vector3Scale(rbB.Velocity, 1.0-friction)

		// Apply normal force to counter gravity if contact normal points upward
		// This prevents slow sinking through stacked objects
		if normal.Y > 0.5 { // Contact points upward (B supports A)
			// A is on top of B - apply upward normal force to A
			normalForce := rl.Vector3Scale(rl.Vector3{X: 0, Y: 1, Z: 0}, -p.Gravity.Y*rbA.Mass)
			if existing, ok := p.normalForces[a]; ok {
				p.normalForces[a] = rl.Vector3Add(existing, normalForce)
			} else {
				p.normalForces[a] = normalForce
			}
		} else if normal.Y < -0.5 { // Contact points downward (A supports B)
			// B is on top of A - apply upward normal force to B
			normalForce := rl.Vector3Scale(rl.Vector3{X: 0, Y: 1, Z: 0}, -p.Gravity.Y*rbB.Mass)
			if existing, ok := p.normalForces[b]; ok {
				p.normalForces[b] = rl.Vector3Add(existing, normalForce)
			} else {
				p.normalForces[b] = normalForce
			}
		}

		// If velocity is very low, zero it out to stop jitter
		if rl.Vector3Length(rbA.Velocity) < 0.1 {
			rbA.Velocity = rl.Vector3{}
		}
		if rl.Vector3Length(rbB.Velocity) < 0.1 {
			rbB.Velocity = rl.Vector3{}
		}
		return
	}

	if velAlongNormal > 0 {
		return
	}

	// Restitution
	e := (rbA.Bounciness + rbB.Bounciness) / 2

	// Impulse
	j := -(1 + e) * velAlongNormal
	j /= (1/rbA.Mass + 1/rbB.Mass)

	impulse := rl.Vector3Scale(normal, j)
	rbA.Velocity = rl.Vector3Add(rbA.Velocity, rl.Vector3Scale(impulse, 1/rbA.Mass))
	rbB.Velocity = rl.Vector3Subtract(rbB.Velocity, rl.Vector3Scale(impulse, 1/rbB.Mass))

	// Torque for spheres - contact point is on surface along normal
	rA := rl.Vector3Scale(normal, -sA.Radius)
	rB := rl.Vector3Scale(normal, sB.Radius)

	torqueScale := float32(50.0)
	torqueA := cross(rA, impulse)
	torqueB := cross(rB, rl.Vector3Scale(impulse, -1))

	rbA.AngularVelocity = rl.Vector3Add(rbA.AngularVelocity, rl.Vector3Scale(torqueA, torqueScale/rbA.Mass))
	rbB.AngularVelocity = rl.Vector3Add(rbB.AngularVelocity, rl.Vector3Scale(torqueB, torqueScale/rbB.Mass))
}

// resolveSphereVsBox handles collision between a sphere and a box (supports rotated boxes via OBB)
func (p *PhysicsWorld) resolveSphereVsBox(sphereObj, boxObj *engine.GameObject, rbSphere, rbBox *components.Rigidbody, sphere *components.SphereCollider, box *components.BoxCollider) {
	sphereCenter := sphereObj.Transform.Position
	obb := NewOBBFromBox(box.GetCenter(), box.Size, boxObj.WorldRotation(), boxObj.WorldScale())

	// Find closest point on OBB to sphere center
	closest := ClosestPointOnOBB(obb, sphereCenter)

	diff := rl.Vector3Subtract(sphereCenter, closest)
	dist := rl.Vector3Length(diff)

	if dist >= sphere.Radius || dist < 0.0001 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(sphereObj, boxObj)

	// Normal points from box to sphere
	normal := rl.Vector3Scale(diff, 1/dist)
	penetration := sphere.Radius - dist

	// Split push based on mass
	totalMass := rbSphere.Mass + rbBox.Mass
	ratioSphere := rbBox.Mass / totalMass
	ratioBox := rbSphere.Mass / totalMass

	sphereObj.Transform.Position = rl.Vector3Add(sphereObj.Transform.Position, rl.Vector3Scale(normal, penetration*ratioSphere))
	boxObj.Transform.Position = rl.Vector3Subtract(boxObj.Transform.Position, rl.Vector3Scale(normal, penetration*ratioBox))

	// Relative velocity
	relVel := rl.Vector3Subtract(rbSphere.Velocity, rbBox.Velocity)
	velAlongNormal := rl.Vector3DotProduct(relVel, normal)

	if velAlongNormal > 0 {
		return
	}

	// Restitution
	e := (rbSphere.Bounciness + rbBox.Bounciness) / 2

	// Impulse
	j := -(1 + e) * velAlongNormal
	j /= (1/rbSphere.Mass + 1/rbBox.Mass)

	impulse := rl.Vector3Scale(normal, j)
	rbSphere.Velocity = rl.Vector3Add(rbSphere.Velocity, rl.Vector3Scale(impulse, 1/rbSphere.Mass))
	rbBox.Velocity = rl.Vector3Subtract(rbBox.Velocity, rl.Vector3Scale(impulse, 1/rbBox.Mass))

	// Torque only for spheres (AABB boxes don't rotate)
	rSphere := rl.Vector3Scale(normal, -sphere.Radius)
	torqueScale := float32(50.0)
	torqueSphere := cross(rSphere, impulse)
	rbSphere.AngularVelocity = rl.Vector3Add(rbSphere.AngularVelocity, rl.Vector3Scale(torqueSphere, torqueScale/rbSphere.Mass))
}

// resolveStaticCollision handles dynamic object colliding with static object
func (p *PhysicsWorld) resolveStaticCollision(obj, static *engine.GameObject) {
	rb := engine.GetComponent[*components.Rigidbody](obj)
	if rb == nil {
		return
	}

	// Check for sphere collider
	sphere := engine.GetComponent[*components.SphereCollider](obj)
	colStatic := engine.GetComponent[*components.BoxCollider](static)

	if sphere != nil && colStatic != nil {
		p.resolveSphereVsStaticBox(obj, static, rb, sphere, colStatic)
		return
	}

	// Box vs static box - use OBB for rotated collision
	colObj := engine.GetComponent[*components.BoxCollider](obj)
	if colObj == nil || colStatic == nil {
		return
	}

	obbObj := NewOBBFromBox(colObj.GetCenter(), colObj.Size, obj.WorldRotation(), obj.WorldScale())
	obbStatic := NewOBBFromBox(colStatic.GetCenter(), colStatic.Size, static.WorldRotation(), static.WorldScale())

	pushOut := obbObj.ResolveOBB(obbStatic)
	if pushOut.X == 0 && pushOut.Y == 0 && pushOut.Z == 0 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(obj, static)

	// Push fully out (static doesn't move)
	obj.Transform.Position = rl.Vector3Add(obj.Transform.Position, pushOut)

	// Reflect velocity
	pushLen := rl.Vector3Length(pushOut)
	if pushLen < 0.0001 {
		return
	}
	normal := rl.Vector3Scale(pushOut, 1/pushLen)

	velAlongNormal := rl.Vector3DotProduct(rb.Velocity, normal)
	if velAlongNormal < 0 {
		// Reflect and apply bounciness
		reflect := rl.Vector3Scale(normal, -2*velAlongNormal*rb.Bounciness)
		rb.Velocity = rl.Vector3Add(rb.Velocity, reflect)

		// Apply friction perpendicular to normal
		rb.Velocity.X *= (1 - rb.Friction)
		rb.Velocity.Z *= (1 - rb.Friction)

		// Apply torque - contact point is on surface in direction of normal
		halfSize := rl.Vector3{X: colObj.Size.X / 2, Y: colObj.Size.Y / 2, Z: colObj.Size.Z / 2}
		r := estimateContactPoint(rl.Vector3{}, halfSize, rl.Vector3Scale(normal, -1))
		torque := cross(r, reflect)
		// Convert to degrees and scale up significantly
		torqueScale := float32(500.0) // Much higher to make rotation visible
		rb.AngularVelocity = rl.Vector3Add(rb.AngularVelocity, rl.Vector3Scale(torque, torqueScale/rb.Mass))

		// Friction on angular velocity when on ground
		if normal.Y > 0.5 {
			rb.AngularVelocity.X *= (1 - rb.Friction*0.5)
			rb.AngularVelocity.Z *= (1 - rb.Friction*0.5)
		}
	}
}

// resolveSphereVsStaticBox handles sphere colliding with static box (floor, walls)
func (p *PhysicsWorld) resolveSphereVsStaticBox(obj, static *engine.GameObject, rb *components.Rigidbody, sphere *components.SphereCollider, box *components.BoxCollider) {
	sphereCenter := obj.Transform.Position
	obb := NewOBBFromBox(box.GetCenter(), box.Size, static.WorldRotation(), static.WorldScale())

	// Find closest point on OBB to sphere center
	closest := ClosestPointOnOBB(obb, sphereCenter)

	diff := rl.Vector3Subtract(sphereCenter, closest)
	dist := rl.Vector3Length(diff)

	if dist >= sphere.Radius || dist < 0.0001 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(obj, static)

	// Normal points from box to sphere
	normal := rl.Vector3Scale(diff, 1/dist)
	penetration := sphere.Radius - dist

	// Push sphere out
	obj.Transform.Position = rl.Vector3Add(obj.Transform.Position, rl.Vector3Scale(normal, penetration))

	// Reflect velocity
	velAlongNormal := rl.Vector3DotProduct(rb.Velocity, normal)
	if velAlongNormal < 0 {
		reflect := rl.Vector3Scale(normal, -2*velAlongNormal*rb.Bounciness)
		rb.Velocity = rl.Vector3Add(rb.Velocity, reflect)

		// Apply friction
		rb.Velocity.X *= (1 - rb.Friction)
		rb.Velocity.Z *= (1 - rb.Friction)

		// Apply torque - contact point is on sphere surface
		r := rl.Vector3Scale(normal, -sphere.Radius)
		torque := cross(r, reflect)
		torqueScale := float32(30.0)
		rb.AngularVelocity = rl.Vector3Add(rb.AngularVelocity, rl.Vector3Scale(torque, torqueScale/rb.Mass))

		// Friction on angular velocity when on ground
		if normal.Y > 0.5 {
			rb.AngularVelocity.X *= (1 - rb.Friction*0.5)
			rb.AngularVelocity.Z *= (1 - rb.Friction*0.5)
		}
	}
}

// resolveKinematicCollision handles kinematic (player) pushing dynamic objects
func (p *PhysicsWorld) resolveKinematicCollision(kinematic, obj *engine.GameObject) {
	rbKin := engine.GetComponent[*components.Rigidbody](kinematic)
	rbObj := engine.GetComponent[*components.Rigidbody](obj)
	colKin := engine.GetComponent[*components.BoxCollider](kinematic)
	colObj := engine.GetComponent[*components.BoxCollider](obj)

	if rbKin == nil || rbObj == nil || colKin == nil || colObj == nil {
		return
	}

	obbKin := NewOBBFromBox(colKin.GetCenter(), colKin.Size, kinematic.WorldRotation(), kinematic.WorldScale())
	obbObj := NewOBBFromBox(colObj.GetCenter(), colObj.Size, obj.WorldRotation(), obj.WorldScale())

	pushOut := obbKin.ResolveOBB(obbObj)
	if pushOut.X == 0 && pushOut.Y == 0 && pushOut.Z == 0 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(kinematic, obj)

	// Push the dynamic object fully out (kinematic doesn't move)
	obj.Transform.Position = rl.Vector3Subtract(obj.Transform.Position, pushOut)

	// Transfer velocity from kinematic to dynamic
	pushLen := rl.Vector3Length(pushOut)
	if pushLen < 0.0001 {
		return
	}
	normal := rl.Vector3Scale(pushOut, 1/pushLen)

	// Add kinematic's velocity to the object in the push direction
	kinVelAlongNormal := rl.Vector3DotProduct(rbKin.Velocity, normal)
	if kinVelAlongNormal > 0 {
		// Push the object with some of the kinematic's velocity
		impulse := rl.Vector3Scale(normal, kinVelAlongNormal*1.5)
		rbObj.Velocity = rl.Vector3Subtract(rbObj.Velocity, impulse)
	}
}

// resolveKinematicStaticCollision handles kinematic objects (player) colliding with static objects (walls)
func (p *PhysicsWorld) resolveKinematicStaticCollision(kinematic, static *engine.GameObject) {
	colKin := engine.GetComponent[*components.BoxCollider](kinematic)
	colStatic := engine.GetComponent[*components.BoxCollider](static)

	if colKin == nil || colStatic == nil {
		return
	}

	obbKin := NewOBBFromBox(colKin.GetCenter(), colKin.Size, kinematic.WorldRotation(), kinematic.WorldScale())
	obbStatic := NewOBBFromBox(colStatic.GetCenter(), colStatic.Size, static.WorldRotation(), static.WorldScale())

	pushOut := obbKin.ResolveOBB(obbStatic)
	if pushOut.X == 0 && pushOut.Y == 0 && pushOut.Z == 0 {
		return
	}

	// Record collision for callbacks
	p.recordCollision(kinematic, static)

	// Push kinematic out of static (static doesn't move)
	kinematic.Transform.Position = rl.Vector3Add(kinematic.Transform.Position, pushOut)
}

// resolveKinematicMeshCollision handles kinematic objects (player) colliding with mesh colliders
func (p *PhysicsWorld) resolveKinematicMeshCollision(kinematic, static *engine.GameObject) {
	meshCol := engine.GetComponent[*components.MeshCollider](static)
	if meshCol == nil || !meshCol.IsBuilt() {
		return
	}

	// Get the kinematic's collider - try box first, then sphere
	boxCol := engine.GetComponent[*components.BoxCollider](kinematic)
	sphereCol := engine.GetComponent[*components.SphereCollider](kinematic)

	if boxCol != nil {
		// Approximate box as sphere for mesh collision
		center := boxCol.GetCenter()
		size := boxCol.GetWorldSize()
		// Use half the smallest dimension as radius (conservative)
		radius := size.X
		if size.Y < radius {
			radius = size.Y
		}
		if size.Z < radius {
			radius = size.Z
		}
		radius *= 0.5

		if hit, push := meshCol.SphereIntersect(center, radius); hit {
			p.recordCollision(kinematic, static)
			kinematic.Transform.Position = rl.Vector3Add(kinematic.Transform.Position, push)
		}
	} else if sphereCol != nil {
		center := sphereCol.GetCenter()
		radius := sphereCol.Radius

		if hit, push := meshCol.SphereIntersect(center, radius); hit {
			p.recordCollision(kinematic, static)
			kinematic.Transform.Position = rl.Vector3Add(kinematic.Transform.Position, push)
		}
	}
}

// resolveDynamicMeshCollision handles dynamic rigidbodies colliding with mesh colliders
func (p *PhysicsWorld) resolveDynamicMeshCollision(obj, static *engine.GameObject) {
	meshCol := engine.GetComponent[*components.MeshCollider](static)
	if meshCol == nil || !meshCol.IsBuilt() {
		return
	}

	rb := engine.GetComponent[*components.Rigidbody](obj)
	if rb == nil {
		return
	}

	// Get the object's collider
	sphereCol := engine.GetComponent[*components.SphereCollider](obj)
	boxCol := engine.GetComponent[*components.BoxCollider](obj)

	var center rl.Vector3
	var radius float32

	if sphereCol != nil {
		center = sphereCol.GetCenter()
		radius = sphereCol.Radius
	} else if boxCol != nil {
		center = boxCol.GetCenter()
		size := boxCol.GetWorldSize()
		radius = size.X
		if size.Y < radius {
			radius = size.Y
		}
		if size.Z < radius {
			radius = size.Z
		}
		radius *= 0.5
	} else {
		return
	}

	if hit, push := meshCol.SphereIntersect(center, radius); hit {
		p.recordCollision(obj, static)

		// Push out
		obj.Transform.Position = rl.Vector3Add(obj.Transform.Position, push)

		// Reflect velocity
		pushLen := rl.Vector3Length(push)
		if pushLen > 0.0001 {
			normal := rl.Vector3Scale(push, 1.0/pushLen)
			dot := rl.Vector3DotProduct(rb.Velocity, normal)
			if dot < 0 {
				// Reflect with bounciness
				reflect := rl.Vector3Scale(normal, -2*dot*rb.Bounciness)
				rb.Velocity = rl.Vector3Add(rb.Velocity, reflect)
				// Apply friction
				rb.Velocity = rl.Vector3Scale(rb.Velocity, 1.0-rb.Friction)
			}
		}
	}
}
