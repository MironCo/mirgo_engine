package engine

import "testing"

func TestGameObjectRefGet(t *testing.T) {
	scene := NewScene("Test")
	obj := NewGameObject("Target")
	scene.AddGameObject(obj)

	ref := GameObjectRef{UID: obj.UID}

	found := ref.Get(scene)
	if found != obj {
		t.Errorf("Get() failed: expected %v, got %v", obj, found)
	}
}

func TestGameObjectRefGetNil(t *testing.T) {
	scene := NewScene("Test")
	ref := GameObjectRef{UID: 0}

	found := ref.Get(scene)
	if found != nil {
		t.Error("Get() with UID=0 should return nil")
	}

	// Test with non-existent UID
	ref2 := GameObjectRef{UID: 99999}
	found2 := ref2.Get(scene)
	if found2 != nil {
		t.Error("Get() with non-existent UID should return nil")
	}

	// Test with nil scene
	ref3 := GameObjectRef{UID: 123}
	found3 := ref3.Get(nil)
	if found3 != nil {
		t.Error("Get() with nil scene should return nil")
	}
}

func TestGameObjectRefIsValid(t *testing.T) {
	validRef := GameObjectRef{UID: 123}
	if !validRef.IsValid() {
		t.Error("GameObjectRef with UID > 0 should be valid")
	}

	invalidRef := GameObjectRef{UID: 0}
	if invalidRef.IsValid() {
		t.Error("GameObjectRef with UID = 0 should be invalid")
	}
}

func TestGameObjectRefSerialization(t *testing.T) {
	// Test that GameObjectRef can be serialized as float64 (JSON compatibility)
	ref := GameObjectRef{UID: 12345}

	// Serialize
	serialized := float64(ref.UID)
	if serialized != 12345.0 {
		t.Errorf("Expected serialized value 12345.0, got %f", serialized)
	}

	// Deserialize
	deserialized := GameObjectRef{UID: uint64(serialized)}
	if deserialized.UID != ref.UID {
		t.Errorf("Serialization roundtrip failed: expected UID %d, got %d", ref.UID, deserialized.UID)
	}
}

func TestGameObjectRefGetName(t *testing.T) {
	scene := NewScene("Test")
	obj := NewGameObject("MyObject")
	scene.AddGameObject(obj)

	ref := GameObjectRef{UID: obj.UID}
	resolved := ref.Get(scene)
	if resolved == nil {
		t.Fatal("Get() should return object")
	}

	if resolved.Name != "MyObject" {
		t.Errorf("Expected name 'MyObject', got '%s'", resolved.Name)
	}

	// Test with invalid ref
	invalidRef := GameObjectRef{UID: 0}
	invalidResolved := invalidRef.Get(scene)
	if invalidResolved != nil {
		t.Error("Invalid ref should return nil")
	}

	// Test with non-existent UID
	missingRef := GameObjectRef{UID: 99999}
	missingResolved := missingRef.Get(scene)
	if missingResolved != nil {
		t.Error("Missing ref should return nil")
	}
}

func TestGameObjectRefMultipleRefs(t *testing.T) {
	scene := NewScene("Test")
	obj1 := NewGameObject("First")
	obj2 := NewGameObject("Second")

	scene.AddGameObject(obj1)
	scene.AddGameObject(obj2)

	ref1 := GameObjectRef{UID: obj1.UID}
	ref2 := GameObjectRef{UID: obj2.UID}

	// Verify both refs work independently
	found1 := ref1.Get(scene)
	found2 := ref2.Get(scene)

	if found1 != obj1 {
		t.Error("First ref didn't return correct object")
	}
	if found2 != obj2 {
		t.Error("Second ref didn't return correct object")
	}
	if found1 == found2 {
		t.Error("Different refs should return different objects")
	}
}
