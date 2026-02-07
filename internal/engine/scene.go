package engine

type Scene struct {
	Name        string
	GameObjects []*GameObject
}

func NewScene(name string) *Scene {
	return &Scene{
		Name:        name,
		GameObjects: make([]*GameObject, 0),
	}
}

func (s *Scene) AddGameObject(g *GameObject) {
	s.GameObjects = append(s.GameObjects, g)
}

func (s *Scene) RemoveGameObject(g *GameObject) {
	for i, obj := range s.GameObjects {
		if obj == g {
			s.GameObjects = append(s.GameObjects[:i], s.GameObjects[i+1:]...)
			return
		}
	}
}

func (s *Scene) FindByName(name string) *GameObject {
	for _, g := range s.GameObjects {
		if g.Name == name {
			return g
		}
	}
	return nil
}

func (s *Scene) FindByTag(tag string) []*GameObject {
	var result []*GameObject
	for _, g := range s.GameObjects {
		if g.HasTag(tag) {
			result = append(result, g)
		}
	}
	return result
}

func (s *Scene) Start() {
	for _, g := range s.GameObjects {
		g.Start()
	}
}

func (s *Scene) Update(deltaTime float32) {
	for _, g := range s.GameObjects {
		g.Update(deltaTime)
	}
}
