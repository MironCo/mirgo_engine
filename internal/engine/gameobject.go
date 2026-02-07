package engine

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Transform struct {
	Position rl.Vector3
	Rotation rl.Vector3 // Euler angles in degrees
	Scale    rl.Vector3
}

type GameObject struct {
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
