package physics

import (
	"log"
	"test3d/internal/components"
	"test3d/internal/compute"
	"test3d/internal/engine"
	"time"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Cross product of two vectors
func cross(a, b rl.Vector3) rl.Vector3 {
	return rl.Vector3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// Estimate contact point on object's surface given push direction
func estimateContactPoint(center rl.Vector3, halfSize rl.Vector3, pushDir rl.Vector3) rl.Vector3 {
	// Contact is on the face in the direction of the push
	// Use the push direction components scaled by half size
	contact := center
	contact.X -= pushDir.X * halfSize.X
	contact.Y -= pushDir.Y * halfSize.Y
	contact.Z -= pushDir.Z * halfSize.Z
	return contact
}

// Spatial grid cell size - objects within same or neighboring cells are checked
const CellSize = 5.0

// Cell key for spatial hashing
type CellKey struct {
	X, Y, Z int
}

func posToCell(pos rl.Vector3) CellKey {
	return CellKey{
		X: int(pos.X / CellSize),
		Y: int(pos.Y / CellSize),
		Z: int(pos.Z / CellSize),
	}
}

// CollisionPair represents two objects that are colliding
type CollisionPair struct {
	A, B *engine.GameObject
}

// makePair creates a consistent collision pair (smaller pointer first)
func makePair(a, b *engine.GameObject) CollisionPair {
	ptrA, ptrB := uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b))
	if ptrA > ptrB {
		return CollisionPair{A: b, B: a}
	}
	return CollisionPair{A: a, B: b}
}

type PhysicsWorld struct {
	Gravity    rl.Vector3
	Objects    []*engine.GameObject // dynamic rigidbodies
	Kinematics []*engine.GameObject // kinematic rigidbodies (player, moving platforms)
	Statics    []*engine.GameObject // no rigidbody (walls, floor)
	grid       map[CellKey][]*engine.GameObject

	// Collision tracking for callbacks
	activeCollisions  map[CollisionPair]bool // collisions from last frame
	currentCollisions map[CollisionPair]bool // collisions this frame

	// Normal forces - accumulated during collision resolution, applied before gravity
	normalForces map[*engine.GameObject]rl.Vector3

	// GPU broad-phase (nil if compute unavailable or object count too low)
	gpuBroadPhase   *compute.BroadPhase
	useGPU          bool      // switches on when object count exceeds threshold
	lastLoggedCount int       // prevents duplicate logs at same object count
	lastLogTime     time.Time // rate-limit collision pair logs
}

// GPUBroadPhaseThreshold is the minimum object count before GPU broad-phase kicks in.
// Below this, CPU spatial hashing is faster due to GPU overhead.
const GPUBroadPhaseThreshold = 750

// MaxPhysicsObjects is the maximum objects the GPU broad-phase can handle.
const MaxPhysicsObjects = 50000

func NewPhysicsWorld() *PhysicsWorld {
	return &PhysicsWorld{
		Gravity:           rl.Vector3{X: 0, Y: -20.0, Z: 0},
		Objects:           make([]*engine.GameObject, 0),
		Kinematics:        make([]*engine.GameObject, 0),
		Statics:           make([]*engine.GameObject, 0),
		grid:              make(map[CellKey][]*engine.GameObject),
		activeCollisions:  make(map[CollisionPair]bool),
		currentCollisions: make(map[CollisionPair]bool),
		normalForces:      make(map[*engine.GameObject]rl.Vector3),
	}
}

// InitGPU initializes GPU broad-phase. Call after compute.Initialize().
func (p *PhysicsWorld) InitGPU() {
	if p.gpuBroadPhase != nil {
		return // Already initialized
	}
	bp, err := compute.NewBroadPhase(MaxPhysicsObjects, MaxPhysicsObjects*20)
	if err == nil && bp != nil {
		p.gpuBroadPhase = bp
		log.Printf("Physics: GPU broad-phase ready (threshold: %d objects)", GPUBroadPhaseThreshold)
	}
}

// rebuildGrid clears and repopulates the spatial hash grid
func (p *PhysicsWorld) rebuildGrid() {
	// Clear grid
	for k := range p.grid {
		delete(p.grid, k)
	}

	// Insert all dynamic objects
	for _, obj := range p.Objects {
		cell := posToCell(obj.Transform.Position)
		p.grid[cell] = append(p.grid[cell], obj)
	}
}

// buildBoundingSpheres creates sphere bounds for all dynamic objects
func (p *PhysicsWorld) buildBoundingSpheres() []compute.Sphere {
	spheres := make([]compute.Sphere, len(p.Objects))

	for i, obj := range p.Objects {
		pos := obj.Transform.Position
		var radius float32 = 0.5 // default

		// Get actual collider radius
		if sphere := engine.GetComponent[*components.SphereCollider](obj); sphere != nil {
			radius = sphere.Radius
		} else if box := engine.GetComponent[*components.BoxCollider](obj); box != nil {
			// Use half-diagonal of box as bounding sphere radius
			size := box.GetWorldSize()
			radius = rl.Vector3Length(size) * 0.5
		}

		spheres[i] = compute.Sphere{
			X:      pos.X,
			Y:      pos.Y,
			Z:      pos.Z,
			Radius: radius,
		}
	}

	return spheres
}

// getNeighborObjects returns all objects in same cell and 26 neighboring cells
func (p *PhysicsWorld) getNeighborObjects(obj *engine.GameObject) []*engine.GameObject {
	cell := posToCell(obj.Transform.Position)
	var neighbors []*engine.GameObject

	// Check 3x3x3 cube of cells centered on object's cell
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				key := CellKey{cell.X + dx, cell.Y + dy, cell.Z + dz}
				neighbors = append(neighbors, p.grid[key]...)
			}
		}
	}
	return neighbors
}

func (p *PhysicsWorld) AddObject(g *engine.GameObject) {
	rb := engine.GetComponent[*components.Rigidbody](g)
	if rb == nil {
		p.Statics = append(p.Statics, g)
	} else if rb.IsKinematic {
		p.Kinematics = append(p.Kinematics, g)
	} else {
		p.Objects = append(p.Objects, g)
	}
}

func (p *PhysicsWorld) RemoveObject(g *engine.GameObject) {
	// Remove from Objects
	for i, obj := range p.Objects {
		if obj == g {
			p.Objects = append(p.Objects[:i], p.Objects[i+1:]...)
			return
		}
	}
	// Remove from Kinematics
	for i, obj := range p.Kinematics {
		if obj == g {
			p.Kinematics = append(p.Kinematics[:i], p.Kinematics[i+1:]...)
			return
		}
	}
	// Remove from Statics
	for i, obj := range p.Statics {
		if obj == g {
			p.Statics = append(p.Statics[:i], p.Statics[i+1:]...)
			return
		}
	}
}

// Release frees GPU resources
func (p *PhysicsWorld) Release() {
	if p.gpuBroadPhase != nil {
		p.gpuBroadPhase.Release()
		p.gpuBroadPhase = nil
	}
}

// UsingGPU returns true if GPU broad-phase is currently active
func (p *PhysicsWorld) UsingGPU() bool {
	return p.useGPU
}

// DynamicObjectCount returns the number of dynamic physics objects
func (p *PhysicsWorld) DynamicObjectCount() int {
	return len(p.Objects)
}

func (p *PhysicsWorld) Update(deltaTime float32) {
	// Reset current frame collisions
	p.currentCollisions = make(map[CollisionPair]bool)

	// 1. Apply forces (gravity + normal forces from previous frame) and integrate velocity
	for _, obj := range p.Objects {
		rb := engine.GetComponent[*components.Rigidbody](obj)
		if rb == nil {
			continue
		}

		// Skip sleeping objects
		if rb.IsSleeping {
			continue
		}

		// Apply gravity
		if rb.UseGravity {
			gravityAccel := rl.Vector3Scale(p.Gravity, deltaTime)

			// Apply normal force from last frame to counter gravity (prevents sinking)
			if normalForce, hasNormal := p.normalForces[obj]; hasNormal {
				// Normal force counters gravity
				normalAccel := rl.Vector3Scale(normalForce, deltaTime/rb.Mass)
				gravityAccel = rl.Vector3Add(gravityAccel, normalAccel)
			}

			rb.Velocity = rl.Vector3Add(rb.Velocity, gravityAccel)
		}

		// Integrate position
		obj.Transform.Position = rl.Vector3Add(
			obj.Transform.Position,
			rl.Vector3Scale(rb.Velocity, deltaTime),
		)

		// Integrate rotation for all rigidbodies (now that we have OBB collision)
		obj.Transform.Rotation = rl.Vector3Add(
			obj.Transform.Rotation,
			rl.Vector3Scale(rb.AngularVelocity, deltaTime),
		)

		// Apply angular damping (time-based so it's framerate independent)
		damping := float32(1.0) - (1.0-rb.AngularDamping)*deltaTime*60
		if damping < 0 {
			damping = 0
		}
		rb.AngularVelocity = rl.Vector3Scale(rb.AngularVelocity, damping)

		// Check if object should go to sleep
		rb.TrySleep(deltaTime)
	}

	// Clear normal forces - they will be recalculated during collision resolution
	p.normalForces = make(map[*engine.GameObject]rl.Vector3)

	// 2. Broad-phase collision detection
	// Use GPU when object count is high enough to benefit
	wasUsingGPU := p.useGPU
	p.useGPU = p.gpuBroadPhase != nil && len(p.Objects) >= GPUBroadPhaseThreshold

	// Log when GPU kicks in or out, and periodically show object count
	if p.useGPU && !wasUsingGPU {
		log.Printf("Physics: GPU broad-phase ON (%d objects)", len(p.Objects))
	} else if !p.useGPU && wasUsingGPU {
		log.Printf("Physics: GPU broad-phase OFF (%d objects)", len(p.Objects))
	} else if len(p.Objects)%100 == 0 && len(p.Objects) > 0 && len(p.Objects) != p.lastLoggedCount {
		p.lastLoggedCount = len(p.Objects)
		mode := "CPU"
		if p.useGPU {
			mode = "GPU"
		}
		log.Printf("Physics: %d objects (%s)", len(p.Objects), mode)
	}

	if p.useGPU {
		// GPU broad-phase: get collision pairs from compute shader
		spheres := p.buildBoundingSpheres()
		pairs, err := p.gpuBroadPhase.DetectPairs(spheres)
		if err == nil {
			// Log collision pairs once per second
			if len(pairs) > 0 && time.Since(p.lastLogTime) >= time.Second {
				p.lastLogTime = time.Now()
				log.Printf("Physics: GPU detected %d collision pairs (%d objects)", len(pairs), len(p.Objects))
			}
			// Narrow-phase only on pairs the GPU found
			for _, pair := range pairs {
				if int(pair.A) < len(p.Objects) && int(pair.B) < len(p.Objects) {
					p.resolveCollision(p.Objects[pair.A], p.Objects[pair.B])
				}
			}
		}
	} else {
		// CPU broad-phase: spatial hashing
		p.rebuildGrid()

		// Track checked pairs to avoid duplicate checks
		checked := make(map[[2]uintptr]bool)

		for _, obj := range p.Objects {
			neighbors := p.getNeighborObjects(obj)
			for _, other := range neighbors {
				if obj == other {
					continue
				}
				// Create consistent pair key using pointer addresses (smaller first)
				ptrA, ptrB := uintptr(unsafe.Pointer(obj)), uintptr(unsafe.Pointer(other))
				if ptrA > ptrB {
					ptrA, ptrB = ptrB, ptrA
				}
				key := [2]uintptr{ptrA, ptrB}
				if checked[key] {
					continue
				}
				checked[key] = true
				p.resolveCollision(obj, other)
			}
		}
	}

	// 3. Kinematic vs Dynamic collision (kinematic pushes dynamic)
	for _, kinematic := range p.Kinematics {
		for _, obj := range p.Objects {
			p.resolveKinematicCollision(kinematic, obj)
		}
	}

	// 4. Rigidbody vs Static collision
	for _, obj := range p.Objects {
		for _, static := range p.Statics {
			p.resolveStaticCollision(obj, static)
		}
	}

	// 5. Kinematic vs Static collision (player vs walls/static objects)
	for _, kinematic := range p.Kinematics {
		for _, static := range p.Statics {
			p.resolveKinematicStaticCollision(kinematic, static)
		}
	}

	// 6. Kinematic vs MeshCollider (player vs terrain/complex geometry)
	for _, kinematic := range p.Kinematics {
		for _, static := range p.Statics {
			p.resolveKinematicMeshCollision(kinematic, static)
		}
	}

	// 7. Dynamic vs MeshCollider
	for _, obj := range p.Objects {
		for _, static := range p.Statics {
			p.resolveDynamicMeshCollision(obj, static)
		}
	}

	// 8. Dispatch collision callbacks
	p.dispatchCollisionCallbacks()
}

// recordCollision marks a collision pair as active this frame and wakes sleeping objects
func (p *PhysicsWorld) recordCollision(a, b *engine.GameObject) {
	pair := makePair(a, b)
	p.currentCollisions[pair] = true

	// Wake sleeping rigidbodies only if collision has significant relative velocity
	// This prevents micro-collisions from waking settled stacks
	rbA := engine.GetComponent[*components.Rigidbody](a)
	rbB := engine.GetComponent[*components.Rigidbody](b)

	if rbA != nil && rbB != nil {
		relVel := rl.Vector3Subtract(rbA.Velocity, rbB.Velocity)
		relSpeed := rl.Vector3Length(relVel)

		// Only wake if relative velocity is significant (> 2x sleep threshold)
		wakeThreshold := float32(components.SleepVelocityThreshold * 2.0)

		if relSpeed > wakeThreshold {
			if rbA.IsSleeping {
				rbA.Wake()
			}
			if rbB.IsSleeping {
				rbB.Wake()
			}
		}
	}
}

// dispatchCollisionCallbacks sends OnCollisionEnter/Exit to handlers
func (p *PhysicsWorld) dispatchCollisionCallbacks() {
	// Find new collisions (enter)
	for pair := range p.currentCollisions {
		if !p.activeCollisions[pair] {
			// New collision - call OnCollisionEnter
			p.notifyCollisionEnter(pair.A, pair.B)
			p.notifyCollisionEnter(pair.B, pair.A)
		}
	}

	// Find ended collisions (exit)
	for pair := range p.activeCollisions {
		if !p.currentCollisions[pair] {
			// Collision ended - call OnCollisionExit
			p.notifyCollisionExit(pair.A, pair.B)
			p.notifyCollisionExit(pair.B, pair.A)
		}
	}

	// Swap buffers
	p.activeCollisions = p.currentCollisions
}

// notifyCollisionEnter calls OnCollisionEnter on all handlers in obj
func (p *PhysicsWorld) notifyCollisionEnter(obj, other *engine.GameObject) {
	for _, comp := range obj.Components() {
		if handler, ok := comp.(engine.CollisionHandler); ok {
			handler.OnCollisionEnter(other)
		}
	}
}

// notifyCollisionExit calls OnCollisionExit on all handlers in obj
func (p *PhysicsWorld) notifyCollisionExit(obj, other *engine.GameObject) {
	for _, comp := range obj.Components() {
		if handler, ok := comp.(engine.CollisionHandler); ok {
			handler.OnCollisionExit(other)
		}
	}
}

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

// Sphere vs Sphere collision
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

// Sphere vs Box collision (supports rotated boxes via OBB)
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

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

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
