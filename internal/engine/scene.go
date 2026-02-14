package engine

type Scene struct {
	Name        string
	World       WorldAccess
	GameObjects []*GameObject
	uidMap      map[uint64]*GameObject // Fast UID lookup
}

func NewScene(name string) *Scene {
	return &Scene{
		Name:        name,
		GameObjects: make([]*GameObject, 0),
		uidMap:      make(map[uint64]*GameObject),
	}
}

func (s *Scene) AddGameObject(g *GameObject) {
	g.Scene = s
	s.GameObjects = append(s.GameObjects, g)

	// Add to UID map for fast lookup
	if s.uidMap == nil {
		s.uidMap = make(map[uint64]*GameObject)
	}
	s.uidMap[g.UID] = g
}

func (s *Scene) RemoveGameObject(g *GameObject) {
	// Detach from parent
	if g.Parent != nil {
		g.Parent.RemoveChild(g)
	}

	// Remove from UID map
	delete(s.uidMap, g.UID)

	// Remove from flat list
	for i, obj := range s.GameObjects {
		if obj == g {
			s.GameObjects = append(s.GameObjects[:i], s.GameObjects[i+1:]...)
			break
		}
	}

	// Recursively remove children from flat list and UID map
	for _, child := range g.Children {
		delete(s.uidMap, child.UID)
		for i, obj := range s.GameObjects {
			if obj == child {
				s.GameObjects = append(s.GameObjects[:i], s.GameObjects[i+1:]...)
				break
			}
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

func (s *Scene) FindByUID(uid uint64) *GameObject {
	// O(1) lookup using hashmap
	return s.uidMap[uid]
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

// FindGameObjectByTag returns the first GameObject with the given tag, or nil if not found.
// This is useful when you expect only one GameObject with a tag (e.g., "Player").
func (s *Scene) FindGameObjectByTag(tag string) *GameObject {
	for _, g := range s.GameObjects {
		if g.HasTag(tag) {
			return g
		}
	}
	return nil
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
