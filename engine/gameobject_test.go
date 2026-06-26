package engine

import "testing"

func TestNewGameObject(t *testing.T) {
	obj := NewGameObject("TestObject")

	if obj.Name != "TestObject" {
		t.Errorf("Expected name 'TestObject', got '%s'", obj.Name)
	}

	if obj.UID == 0 {
		t.Error("UID should not be 0")
	}

	if obj.components == nil {
		t.Error("components slice should be initialized")
	}
}

func TestGameObjectUniqueUIDs(t *testing.T) {
	obj1 := NewGameObject("First")
	obj2 := NewGameObject("Second")
	obj3 := NewGameObject("Third")

	if obj1.UID == obj2.UID {
		t.Error("GameObjects should have unique UIDs")
	}
	if obj2.UID == obj3.UID {
		t.Error("GameObjects should have unique UIDs")
	}
	if obj1.UID == obj3.UID {
		t.Error("GameObjects should have unique UIDs")
	}
}

func TestGameObjectHasTag(t *testing.T) {
	obj := NewGameObject("Test")
	obj.Tags = []string{"enemy", "ai", "dangerous"}

	if !obj.HasTag("enemy") {
		t.Error("HasTag should return true for existing tag")
	}

	if !obj.HasTag("ai") {
		t.Error("HasTag should return true for existing tag")
	}

	if obj.HasTag("player") {
		t.Error("HasTag should return false for non-existent tag")
	}

	// Test empty tags
	obj2 := NewGameObject("Test2")
	if obj2.HasTag("anything") {
		t.Error("HasTag should return false when Tags is nil/empty")
	}
}

func TestGameObjectParentChild(t *testing.T) {
	parent := NewGameObject("Parent")
	child := NewGameObject("Child")

	parent.AddChild(child)

	if child.Parent != parent {
		t.Error("Child.Parent should be set")
	}

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}

	if parent.Children[0] != child {
		t.Error("Child not added to parent's Children slice")
	}
}

func TestGameObjectRemoveChild(t *testing.T) {
	parent := NewGameObject("Parent")
	child1 := NewGameObject("Child1")
	child2 := NewGameObject("Child2")

	parent.AddChild(child1)
	parent.AddChild(child2)

	parent.RemoveChild(child1)

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child after removal, got %d", len(parent.Children))
	}

	if parent.Children[0] != child2 {
		t.Error("Wrong child removed")
	}

	if child1.Parent != nil {
		t.Error("Removed child should have nil parent")
	}
}

func TestGameObjectAddComponent(t *testing.T) {
	obj := NewGameObject("Test")
	comp := &BaseComponent{}

	obj.AddComponent(comp)

	if len(obj.components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(obj.components))
	}

	if comp.gameObject != obj {
		t.Error("Component.gameObject should be set")
	}
}

func TestGameObjectGetComponent(t *testing.T) {
	obj := NewGameObject("Test")
	comp := &BaseComponent{}

	obj.AddComponent(comp)

	found := GetComponent[*BaseComponent](obj)
	if found != comp {
		t.Error("GetComponent failed to find component")
	}
}

func TestGameObjectStartCalledOnce(t *testing.T) {
	obj := NewGameObject("Test")

	// First call should set started = true
	obj.Start()
	if !obj.started {
		t.Error("started flag should be true after Start()")
	}

	// Second call should be a no-op (no panic, no re-initialization)
	obj.Start() // Should not panic or cause issues
}
