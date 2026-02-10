package scripts

import (
	"fmt"
	"test3d/internal/engine"
)

// Collectible is a script that detects when the player touches it.
// Add this to any object with a collider to make it collectible.
type Collectible struct {
	engine.BaseComponent
	// Points awarded when collected
	Points float32
	// Tag to check for (default: "Player")
	TargetTag string
	// Internal state
	collected bool
}

func (c *Collectible) Start() {
	if c.Points == 0 {
		c.Points = 10
	}
	if c.TargetTag == "" {
		c.TargetTag = "Player"
	}
}

func (c *Collectible) OnCollisionEnter(other *engine.GameObject) {
	if c.collected {
		return
	}

	// Check if the other object has the target tag
	if other.HasTag(c.TargetTag) {
		c.collected = true
		fmt.Printf("Collected! +%.0f points\n", c.Points)

		// Destroy this object
		g := c.GetGameObject()
		if g != nil && g.Scene != nil && g.Scene.World != nil {
			g.Scene.World.Destroy(g)
		}
	}
}

func (c *Collectible) OnCollisionExit(other *engine.GameObject) {
	// Optional: do something when player leaves (if not collected)
}
