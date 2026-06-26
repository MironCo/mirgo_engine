package engine

// GameObjectRef is a serializable reference to a GameObject by UID.
// Use this in scripts when you want Unity-style drag-and-drop GameObject references.
//
// Example:
//
//	type MyScript struct {
//	    engine.BaseComponent
//	    TargetButton engine.GameObjectRef
//	}
//
//	func (s *MyScript) Start() {
//	    if button := s.TargetButton.Get(s.GetGameObject().Scene); button != nil {
//	        // Use the button...
//	    }
//	}
type GameObjectRef struct {
	UID uint64 // UID of the referenced GameObject (0 = none)
}

// Get resolves the reference to the actual GameObject.
// Returns nil if the reference is empty (UID = 0) or if the GameObject doesn't exist.
func (r GameObjectRef) Get(scene *Scene) *GameObject {
	if r.UID == 0 || scene == nil {
		return nil
	}
	return scene.FindByUID(r.UID)
}

// IsValid returns true if the reference points to something (UID != 0).
// Note: This doesn't check if the GameObject actually exists in the scene.
func (r GameObjectRef) IsValid() bool {
	return r.UID != 0
}

// Set sets the reference to point to the given GameObject.
// Pass nil to clear the reference.
func (r *GameObjectRef) Set(g *GameObject) {
	if g == nil {
		r.UID = 0
	} else {
		r.UID = g.UID
	}
}

// Clear clears the reference (sets UID to 0).
func (r *GameObjectRef) Clear() {
	r.UID = 0
}
