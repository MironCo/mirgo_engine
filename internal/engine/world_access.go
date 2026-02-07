package engine

import rl "github.com/gen2brain/raylib-go/raylib"

// RaycastResult holds information about a raycast hit.
// Defined here to avoid circular imports with physics package.
type RaycastResult struct {
	GameObject *GameObject
	Point      rl.Vector3
	Normal     rl.Vector3
	Distance   float32
}

// WorldAccess provides components with access to world-level operations
// without creating circular import dependencies.
type WorldAccess interface {
	GetCollidableObjects() []*GameObject
	SpawnObject(g *GameObject)
	Destroy(g *GameObject)
	Raycast(origin, direction rl.Vector3, maxDistance float32) (RaycastResult, bool)
}
