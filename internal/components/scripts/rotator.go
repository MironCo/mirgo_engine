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

func init() {
	engine.RegisterScript("Rotator", rotatorFactory, rotatorSerializer)
}

func rotatorFactory(props map[string]any) engine.Component {
	speed := float32(90)
	if v, ok := props["speed"].(float64); ok {
		speed = float32(v)
	}
	return &Rotator{Speed: speed}
}

func rotatorSerializer(c engine.Component) map[string]any {
	r, ok := c.(*Rotator)
	if !ok {
		return nil
	}
	return map[string]any{
		"speed": r.Speed,
	}
}
