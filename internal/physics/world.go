package physics

import (
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type PhysicsWorld struct {
	Gravity rl.Vector3
	Objects []*engine.GameObject // objects with rigidbodies
	Statics []*engine.GameObject // objects without rigidbodies (walls, floor, etc)
}

func NewPhysicsWorld() *PhysicsWorld {
	return &PhysicsWorld{
		Gravity: rl.Vector3{X: 0, Y: -20.0, Z: 0},
		Objects: make([]*engine.GameObject, 0),
		Statics: make([]*engine.GameObject, 0),
	}
}

func (p *PhysicsWorld) AddObject(g *engine.GameObject) {
	rb := engine.GetComponent[*components.Rigidbody](g)
	if rb != nil && !rb.IsKinematic {
		p.Objects = append(p.Objects, g)
	} else {
		p.Statics = append(p.Statics, g)
	}
}

func (p *PhysicsWorld) Update(deltaTime float32) {
	// 1. Apply gravity and integrate velocity
	for _, obj := range p.Objects {
		rb := engine.GetComponent[*components.Rigidbody](obj)
		if rb == nil {
			continue
		}

		// Apply gravity
		if rb.UseGravity {
			rb.Velocity = rl.Vector3Add(rb.Velocity, rl.Vector3Scale(p.Gravity, deltaTime))
		}

		// Integrate position
		obj.Transform.Position = rl.Vector3Add(
			obj.Transform.Position,
			rl.Vector3Scale(rb.Velocity, deltaTime),
		)
	}

	// 2. Rigidbody vs Rigidbody collision (O(n²) baby)
	for i := 0; i < len(p.Objects); i++ {
		for j := i + 1; j < len(p.Objects); j++ {
			p.resolveCollision(p.Objects[i], p.Objects[j])
		}
	}

	// 4. Rigidbody vs Static collision
	for _, obj := range p.Objects {
		for _, static := range p.Statics {
			p.resolveStaticCollision(obj, static)
		}
	}
}

func (p *PhysicsWorld) resolveCollision(a, b *engine.GameObject) {
	rbA := engine.GetComponent[*components.Rigidbody](a)
	rbB := engine.GetComponent[*components.Rigidbody](b)
	colA := engine.GetComponent[*components.BoxCollider](a)
	colB := engine.GetComponent[*components.BoxCollider](b)

	if rbA == nil || rbB == nil || colA == nil || colB == nil {
		return
	}

	aabbA := NewAABBFromCenter(a.Transform.Position, colA.Size)
	aabbB := NewAABBFromCenter(b.Transform.Position, colB.Size)

	pushOut := aabbA.Resolve(aabbB)
	if pushOut.X == 0 && pushOut.Y == 0 && pushOut.Z == 0 {
		return
	}

	// Split the push based on mass ratio
	totalMass := rbA.Mass + rbB.Mass
	ratioA := rbB.Mass / totalMass
	ratioB := rbA.Mass / totalMass

	a.Transform.Position = rl.Vector3Add(a.Transform.Position, rl.Vector3Scale(pushOut, ratioA))
	b.Transform.Position = rl.Vector3Subtract(b.Transform.Position, rl.Vector3Scale(pushOut, ratioB))

	// Bounce velocities
	// Find collision normal (normalize pushOut)
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
}

func (p *PhysicsWorld) resolveStaticCollision(obj, static *engine.GameObject) {
	rb := engine.GetComponent[*components.Rigidbody](obj)
	colObj := engine.GetComponent[*components.BoxCollider](obj)
	colStatic := engine.GetComponent[*components.BoxCollider](static)

	if rb == nil || colObj == nil || colStatic == nil {
		return
	}

	aabbObj := NewAABBFromCenter(obj.Transform.Position, colObj.Size)
	aabbStatic := NewAABBFromCenter(static.Transform.Position, colStatic.Size)

	pushOut := aabbObj.Resolve(aabbStatic)
	if pushOut.X == 0 && pushOut.Y == 0 && pushOut.Z == 0 {
		return
	}

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
	}
}
