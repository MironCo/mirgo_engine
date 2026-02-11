package engine

import (
	"math"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UID counter for generating unique IDs
var uidCounter uint64

type Transform struct {
	Position rl.Vector3
	Rotation rl.Vector3 // Euler angles in degrees (XYZ order)
	Scale    rl.Vector3

	quaternion rl.Quaternion
	quatDirty  bool
}

type GameObject struct {
	UID        uint64
	Name       string
	Tags       []string
	Transform  Transform
	Active     bool
	Scene      *Scene
	Parent     *GameObject
	Children   []*GameObject
	components []Component
	started    bool
}

func NewGameObject(name string) *GameObject {
	return &GameObject{
		UID:    atomic.AddUint64(&uidCounter, 1),
		Name:   name,
		Active: true,
		Transform: Transform{
			Position:   rl.Vector3{},
			Rotation:   rl.Vector3{},
			Scale:      rl.Vector3{X: 1, Y: 1, Z: 1},
			quaternion: rl.Quaternion{X: 0, Y: 0, Z: 0, W: 1}, // Identity
			quatDirty:  false,
		},
		components: make([]Component, 0),
		Children:   make([]*GameObject, 0),
	}
}

// NewGameObjectWithUID creates a GameObject with a specific UID (for loading from files)
func NewGameObjectWithUID(name string, uid uint64) *GameObject {
	// Update counter if loaded UID is higher to avoid collisions
	for {
		current := atomic.LoadUint64(&uidCounter)
		if uid <= current {
			break
		}
		if atomic.CompareAndSwapUint64(&uidCounter, current, uid) {
			break
		}
	}
	return &GameObject{
		UID:    uid,
		Name:   name,
		Active: true,
		Transform: Transform{
			Position: rl.Vector3{},
			Rotation: rl.Vector3{},
			Scale:    rl.Vector3{X: 1, Y: 1, Z: 1},
		},
		components: make([]Component, 0),
		Children:   make([]*GameObject, 0),
	}
}

func (g *GameObject) AddComponent(c Component) {
	c.SetGameObject(g)
	g.components = append(g.components, c)
}

func (g *GameObject) GetComponent(target Component) Component {
	for _, c := range g.components {
		// Type switch to find matching component type
		switch target.(type) {
		case *BaseComponent:
			if _, ok := c.(*BaseComponent); ok {
				return c
			}
		}
	}
	return nil
}

// GetComponentOfType returns a component using a type assertion helper
func GetComponent[T Component](g *GameObject) T {
	var zero T
	for _, c := range g.components {
		if typed, ok := c.(T); ok {
			return typed
		}
	}
	return zero
}

// FindComponent finds a component that implements the given interface type.
// Unlike GetComponent which requires an exact type, this works with interfaces.
func FindComponent[T any](g *GameObject) T {
	var zero T
	for _, c := range g.components {
		if typed, ok := c.(T); ok {
			return typed
		}
	}
	return zero
}

func (g *GameObject) Start() {
	if g.started {
		return
	}
	for _, c := range g.components {
		c.Start()
	}
	g.started = true
}

func (g *GameObject) Update(deltaTime float32) {
	if !g.Active {
		return
	}
	for _, c := range g.components {
		c.Update(deltaTime)
	}
}

func (g *GameObject) Components() []Component {
	return g.components
}

// RemoveComponent removes a component from the GameObject by pointer.
// Returns true if the component was found and removed.
func (g *GameObject) RemoveComponent(c Component) bool {
	for i, comp := range g.components {
		if comp == c {
			g.components = append(g.components[:i], g.components[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveComponentByIndex removes a component at a specific index.
// Returns true if the index was valid and the component was removed.
func (g *GameObject) RemoveComponentByIndex(index int) bool {
	if index < 0 || index >= len(g.components) {
		return false
	}
	g.components = append(g.components[:index], g.components[index+1:]...)
	return true
}

func (g *GameObject) HasTag(tag string) bool {
	for _, t := range g.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (g *GameObject) AddChild(child *GameObject) {
	child.Parent = g
	g.Children = append(g.Children, child)
}

func (g *GameObject) RemoveChild(child *GameObject) {
	for i, c := range g.Children {
		if c == child {
			g.Children = append(g.Children[:i], g.Children[i+1:]...)
			child.Parent = nil
			return
		}
	}
}

func (g *GameObject) WorldPosition() rl.Vector3 {
	if g.Parent == nil {
		return g.Transform.Position
	}
	parentPos := g.Parent.WorldPosition()
	parentRot := g.Parent.WorldRotation()
	parentScale := g.Parent.WorldScale()

	// Scale local position by parent's world scale
	scaled := rl.Vector3{
		X: g.Transform.Position.X * parentScale.X,
		Y: g.Transform.Position.Y * parentScale.Y,
		Z: g.Transform.Position.Z * parentScale.Z,
	}

	// Rotate by parent rotation (same convention as ModelRenderer: X then Y then Z)
	rx := float64(parentRot.X) * math.Pi / 180
	ry := float64(parentRot.Y) * math.Pi / 180
	rz := float64(parentRot.Z) * math.Pi / 180
	rotX := rl.MatrixRotateX(float32(rx))
	rotY := rl.MatrixRotateY(float32(ry))
	rotZ := rl.MatrixRotateZ(float32(rz))
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

	rotated := rl.Vector3Transform(scaled, rotMatrix)
	return rl.Vector3Add(parentPos, rotated)
}

func (g *GameObject) WorldRotation() rl.Vector3 {
	if g.Parent == nil {
		return g.Transform.Rotation
	}
	return rl.Vector3Add(g.Parent.WorldRotation(), g.Transform.Rotation)
}

func (g *GameObject) WorldScale() rl.Vector3 {
	if g.Parent == nil {
		return g.Transform.Scale
	}
	ps := g.Parent.WorldScale()
	return rl.Vector3{
		X: ps.X * g.Transform.Scale.X,
		Y: ps.Y * g.Transform.Scale.Y,
		Z: ps.Z * g.Transform.Scale.Z,
	}
}

// GetQuaternion returns the quaternion representation of the rotation.
// Converts from Euler angles if needed (lazy evaluation).
func (t *Transform) GetQuaternion() rl.Quaternion {
	if t.quatDirty {
		radX := t.Rotation.X * rl.Deg2rad
		radY := t.Rotation.Y * rl.Deg2rad
		radZ := t.Rotation.Z * rl.Deg2rad

		t.quaternion = rl.QuaternionFromEuler(radX, radY, radZ)
		t.quatDirty = false
	}
	return t.quaternion
}

// SetQuaternion sets the rotation from a quaternion and updates Euler angles.
func (t *Transform) SetQuaternion(q rl.Quaternion) {
	t.quaternion = q
	t.quatDirty = false

	euler := rl.QuaternionToEuler(q)
	t.Rotation = rl.Vector3{
		X: euler.X * rl.Rad2deg,
		Y: euler.Y * rl.Rad2deg,
		Z: euler.Z * rl.Rad2deg,
	}
}

// MarkRotationDirty marks that the Euler angles have changed and quaternion needs updating.
func (t *Transform) MarkRotationDirty() {
	t.quatDirty = true
}
