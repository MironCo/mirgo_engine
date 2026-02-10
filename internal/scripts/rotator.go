package scripts

import "test3d/internal/engine"

// Rotator is a simple script that spins an object around the Y axis.
type Rotator struct {
	engine.BaseComponent
	Speed float32
}

func (r *Rotator) Update(deltaTime float32) {
	g := r.GetGameObject()
	if g == nil {
		return
	}
	g.Transform.Rotation.Y += r.Speed * deltaTime
	if g.Transform.Rotation.Y > 360 {
		g.Transform.Rotation.Y -= 360
	}
}
// test

// --- Generated boilerplate below ---

func init() {
	engine.RegisterScriptWithApplier("Rotator", rotatorFactory, rotatorSerializer, rotatorApplier)
}

func rotatorFactory(props map[string]any) engine.Component {
	script := &Rotator{}
	if v, ok := props["speed"].(float64); ok {
		script.Speed = float32(v)
	}
	return script
}

func rotatorSerializer(c engine.Component) map[string]any {
	s, ok := c.(*Rotator)
	if !ok {
		return nil
	}
	return map[string]any{
		"speed": s.Speed,
	}
}

func rotatorApplier(c engine.Component, propName string, value any) bool {
	s, ok := c.(*Rotator)
	if !ok {
		return false
	}
	switch propName {
	case "speed":
		if v, ok := value.(float64); ok {
			s.Speed = float32(v)
			return true
		}
	}
	return false
}
