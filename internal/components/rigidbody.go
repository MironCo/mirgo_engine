package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
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
