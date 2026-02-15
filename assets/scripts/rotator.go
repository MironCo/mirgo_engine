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
