package engine

import "testing"

func TestSceneAddGameObject(t *testing.T) {
	scene := NewScene("Test")
	obj := NewGameObject("Player")

	scene.AddGameObject(obj)

	if len(scene.GameObjects) != 1 {
		t.Errorf("Expected 1 GameObject, got %d", len(scene.GameObjects))
	}

	if scene.GameObjects[0] != obj {
		t.Error("GameObject not added to scene")
	}

	if obj.Scene != scene {
		t.Error("GameObject.Scene not set")
	}
}

func TestSceneUIDLookup(t *testing.T) {
	scene := NewScene("Test")
	obj := NewGameObject("Player")

	scene.AddGameObject(obj)

	// Test O(1) lookup
	found := scene.FindByUID(obj.UID)
	if found != obj {
		t.Errorf("FindByUID failed: expected %v, got %v", obj, found)
	}

	// Test non-existent UID
	notFound := scene.FindByUID(99999)
	if notFound != nil {
		t.Error("FindByUID should return nil for non-existent UID")
	}
}

func TestSceneRemoveGameObject(t *testing.T) {
	scene := NewScene("Test")
	obj1 := NewGameObject("Player")
	obj2 := NewGameObject("Enemy")

	scene.AddGameObject(obj1)
	scene.AddGameObject(obj2)

	scene.RemoveGameObject(obj1)

	if len(scene.GameObjects) != 1 {
		t.Errorf("Expected 1 GameObject after removal, got %d", len(scene.GameObjects))
	}

	if scene.GameObjects[0] != obj2 {
		t.Error("Wrong GameObject removed")
	}

	// Verify UID map was updated
	if scene.FindByUID(obj1.UID) != nil {
		t.Error("Removed GameObject still in UID map")
	}

	if scene.FindByUID(obj2.UID) != obj2 {
		t.Error("Remaining GameObject not in UID map")
	}
}

func TestSceneFindByName(t *testing.T) {
	scene := NewScene("Test")
	obj := NewGameObject("UniquePlayer")

	scene.AddGameObject(obj)

	found := scene.FindByName("UniquePlayer")
	if found != obj {
		t.Error("FindByName failed")
	}

	notFound := scene.FindByName("DoesNotExist")
	if notFound != nil {
		t.Error("FindByName should return nil for non-existent name")
	}
}

func TestSceneFindByTag(t *testing.T) {
	scene := NewScene("Test")
	obj1 := NewGameObject("Enemy1")
	obj2 := NewGameObject("Enemy2")
	obj3 := NewGameObject("Player")

	obj1.Tags = []string{"enemy", "ai"}
	obj2.Tags = []string{"enemy"}
	obj3.Tags = []string{"player"}

	scene.AddGameObject(obj1)
	scene.AddGameObject(obj2)
	scene.AddGameObject(obj3)

	enemies := scene.FindByTag("enemy")
	if len(enemies) != 2 {
		t.Errorf("Expected 2 enemies, got %d", len(enemies))
	}

	players := scene.FindByTag("player")
	if len(players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(players))
	}

	notFound := scene.FindByTag("nonexistent")
	if len(notFound) != 0 {
		t.Error("FindByTag should return empty slice for non-existent tag")
	}
}

func TestSceneRemoveWithChildren(t *testing.T) {
	scene := NewScene("Test")
	parent := NewGameObject("Parent")
	child := NewGameObject("Child")

	scene.AddGameObject(parent)
	scene.AddGameObject(child)
	parent.AddChild(child)

	scene.RemoveGameObject(parent)

	// Both parent and child should be removed
	if len(scene.GameObjects) != 0 {
		t.Errorf("Expected 0 GameObjects, got %d", len(scene.GameObjects))
	}

	// Verify UID map cleaned up
	if scene.FindByUID(parent.UID) != nil {
		t.Error("Parent still in UID map after removal")
	}
	if scene.FindByUID(child.UID) != nil {
		t.Error("Child still in UID map after removal")
	}
}

func TestSceneUIDMapInitialization(t *testing.T) {
	scene := NewScene("Test")

	if scene.uidMap == nil {
		t.Error("uidMap should be initialized in NewScene")
	}

	// Test adding to uninitialized map (defensive programming check)
	scene.uidMap = nil
	obj := NewGameObject("Test")
	scene.AddGameObject(obj) // Should not panic

	if scene.uidMap == nil {
		t.Error("uidMap should be initialized on first AddGameObject")
	}
}
