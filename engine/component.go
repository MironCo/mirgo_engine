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

// Serializable is implemented by components that can save/load themselves.
// This reduces boilerplate in scenefile.go - just implement these methods
// on your component instead of adding switch cases.
type Serializable interface {
	// TypeName returns the component type name for JSON (e.g., "BoxCollider")
	TypeName() string
	// Serialize returns the JSON-serializable representation
	Serialize() map[string]any
	// Deserialize populates the component from JSON data
	Deserialize(data map[string]any)
}

// ComponentFactory creates a new instance of a Serializable component
type ComponentFactory func() Serializable

// componentRegistry maps type names to factories
var componentRegistry = make(map[string]ComponentFactory)

// RegisterComponent registers a component type for automatic serialization.
// Call this in init() for each component that implements Serializable.
func RegisterComponent(name string, factory ComponentFactory) {
	componentRegistry[name] = factory
}

// CreateComponent creates a component by type name, or nil if not registered
func CreateComponent(typeName string) Serializable {
	if factory, ok := componentRegistry[typeName]; ok {
		return factory()
	}
	return nil
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
