package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("Rigidbody", func() engine.Serializable {
		return NewRigidbody()
	})
}

type Rigidbody struct {
	engine.BaseComponent
	Velocity        rl.Vector3
	AngularVelocity rl.Vector3 // degrees per second on each axis
	Mass            float32
	Bounciness      float32 // 0 = no bounce, 1 = perfect bounce
	Friction        float32 // 0 = ice, 1 = stops immediately
	AngularDamping  float32 // how fast rotation slows down
	UseGravity      bool
	IsKinematic     bool // moves but doesn't get pushed by physics
}

func NewRigidbody() *Rigidbody {
	return &Rigidbody{
		Velocity:        rl.Vector3{},
		AngularVelocity: rl.Vector3{},
		Mass:            1.0,
		Bounciness:      0.5,
		Friction:        0.1,
		AngularDamping:  0.98, // slight damping each frame
		UseGravity:      true,
		IsKinematic:     false,
	}
}

// TypeName implements engine.Serializable
func (r *Rigidbody) TypeName() string {
	return "Rigidbody"
}

// Serialize implements engine.Serializable
func (r *Rigidbody) Serialize() map[string]any {
	return map[string]any{
		"type":        "Rigidbody",
		"mass":        r.Mass,
		"bounciness":  r.Bounciness,
		"friction":    r.Friction,
		"useGravity":  r.UseGravity,
		"isKinematic": r.IsKinematic,
	}
}

// Deserialize implements engine.Serializable
func (r *Rigidbody) Deserialize(data map[string]any) {
	if m, ok := data["mass"].(float64); ok {
		r.Mass = float32(m)
	}
	if b, ok := data["bounciness"].(float64); ok {
		r.Bounciness = float32(b)
	}
	if f, ok := data["friction"].(float64); ok {
		r.Friction = float32(f)
	}
	if g, ok := data["useGravity"].(bool); ok {
		r.UseGravity = g
	}
	if k, ok := data["isKinematic"].(bool); ok {
		r.IsKinematic = k
	}
}
