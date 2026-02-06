package engine

type Component interface {
	Start()
	Update(deltaTime float32)
	SetGameObject(g *GameObject)
	GetGameObject() *GameObject
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
