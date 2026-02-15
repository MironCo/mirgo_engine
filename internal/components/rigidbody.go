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

// Sleep thresholds
const (
	SleepVelocityThreshold = 0.3 // units/sec - below this, object might sleep
	SleepAngularThreshold  = 1.0 // deg/sec - below this, object might sleep
	SleepTimeThreshold     = 0.3 // seconds of low velocity before sleeping
)

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

	// Sleep state - sleeping objects skip physics simulation
	IsSleeping bool
	sleepTimer float32 // time spent below velocity threshold
	CanSleep   bool    // whether this object can sleep (default true)
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
		CanSleep:        true,
	}
}

// Wake forces the rigidbody out of sleep state
func (r *Rigidbody) Wake() {
	r.IsSleeping = false
	r.sleepTimer = 0
}

// TrySleep checks if the rigidbody should go to sleep based on velocity
func (r *Rigidbody) TrySleep(deltaTime float32) {
	if !r.CanSleep || r.IsSleeping {
		return
	}

	// Check if velocities are below threshold
	speed := rl.Vector3Length(r.Velocity)
	angSpeed := rl.Vector3Length(r.AngularVelocity)

	if speed < SleepVelocityThreshold && angSpeed < SleepAngularThreshold {
		r.sleepTimer += deltaTime

		// Apply extra damping when nearly at rest to reduce jitter
		dampFactor := float32(0.9)
		r.Velocity = rl.Vector3Scale(r.Velocity, dampFactor)
		r.AngularVelocity = rl.Vector3Scale(r.AngularVelocity, dampFactor)

		if r.sleepTimer >= SleepTimeThreshold {
			r.IsSleeping = true
			r.Velocity = rl.Vector3{}
			r.AngularVelocity = rl.Vector3{}
		}
	} else {
		r.sleepTimer = 0
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
