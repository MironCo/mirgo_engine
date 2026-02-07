package engine

// WorldAccess provides components with access to world-level operations
// without creating circular import dependencies.
type WorldAccess interface {
	GetCollidableObjects() []*GameObject
	SpawnObject(g *GameObject)
}
