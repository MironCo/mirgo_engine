package engine

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Transform struct {
	Position rl.Vector3
	Rotation rl.Vector3 // Euler angles in degrees
	Scale    rl.Vector3
}

type GameObject struct {
	Name       string
	Transform  Transform
	Active     bool
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
