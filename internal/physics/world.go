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

// Spatial grid cell size - objects within same or neighboring cells are checked
const CellSize = 5.0

// CellKey is a key for spatial hashing
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

// PhysicsWorld manages all physics simulation
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

// NewPhysicsWorld creates a new physics world
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

// AddObject adds a game object to the physics world
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

// RemoveObject removes a game object from the physics world
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

// Update runs one physics simulation step
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
		obj.Transform.MarkRotationDirty() // Quaternion cache needs refresh

		// Apply angular damping (time-based so it's framerate independent)
		damping := float32(1.0) - (1.0-rb.AngularDamping)*deltaTime*60
		if damping < 0 {
			damping = 0
		}
		rb.AngularVelocity = rl.Vector3Scale(rb.AngularVelocity, damping)

		// Box flattening: apply corrective torque to rotate boxes toward nearest flat orientation
		if boxCollider := engine.GetComponent[*components.BoxCollider](obj); boxCollider != nil {
			applyBoxFlatteningTorque(obj, rb, boxCollider, deltaTime)
		}

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
