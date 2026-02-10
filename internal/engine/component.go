package engine

type Component interface {
	Start()
	Update(deltaTime float32)
	SetGameObject(g *GameObject)
	GetGameObject() *GameObject
}

// LookProvider is implemented by components that control camera look direction.
// Used by Camera and other components that need to follow a look direction.
type LookProvider interface {
	GetLookDirection() (x, y, z float32)
	GetEyeHeight() float32
}

// PlayerController is implemented by components that handle player movement.
// Used by collision systems to sync velocity and grounded state.
type PlayerController interface {
	LookProvider
	GetVelocity() (x, y, z float32)
	SetVelocityY(vy float32)
	Grounded() bool
	SetGrounded(grounded bool)
}

// CollisionHandler is implemented by components that want to receive collision callbacks.
// Scripts can implement these methods to react to collisions.
type CollisionHandler interface {
	OnCollisionEnter(other *GameObject)
	OnCollisionExit(other *GameObject)
}

// BaseComponent provides default implementation for Component interface
type BaseComponent struct {
	gameObject *GameObject
}

func (b *BaseComponent) Start() {}

func (b *BaseComponent) Update(deltaTime float32) {}

func (b *BaseComponent) SetGameObject(g *GameObject) {
	b.gameObject = g
}

func (b *BaseComponent) GetGameObject() *GameObject {
	return b.gameObject
}
