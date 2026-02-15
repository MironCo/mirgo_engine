package engine

import "testing"

// Mock script for testing
type MockScript struct {
	BaseComponent
	Speed  float32
	Health int
}

func mockFactory(props map[string]any) Component {
	script := &MockScript{}
	if v, ok := props["speed"].(float64); ok {
		script.Speed = float32(v)
	}
	if v, ok := props["health"].(float64); ok {
		script.Health = int(v)
	}
	return script
}

func mockSerializer(c Component) map[string]any {
	s, ok := c.(*MockScript)
	if !ok {
		return nil
	}
	return map[string]any{
		"speed":  s.Speed,
		"health": s.Health,
	}
}

func mockApplier(c Component, propName string, value any) bool {
	s, ok := c.(*MockScript)
	if !ok {
		return false
	}
	switch propName {
	case "speed":
		if v, ok := value.(float64); ok {
			s.Speed = float32(v)
			return true
		}
	case "health":
		if v, ok := value.(float64); ok {
			s.Health = int(v)
			return true
		}
	}
	return false
}

func TestRegisterScript(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	RegisterScript("MockScript", mockFactory, mockSerializer)

	if _, exists := scriptRegistry["MockScript"]; !exists {
		t.Error("Script not registered")
	}
}

func TestRegisterScriptDuplicate(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	RegisterScript("Duplicate", mockFactory, mockSerializer)

	// Should panic on duplicate registration
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on duplicate registration")
		}
	}()

	RegisterScript("Duplicate", mockFactory, mockSerializer)
}

func TestCreateScript(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	RegisterScript("MockScript", mockFactory, mockSerializer)

	props := map[string]any{
		"speed":  float64(10.5),
		"health": float64(100),
	}

	component := CreateScript("MockScript", props)
	if component == nil {
		t.Fatal("CreateScript returned nil")
	}

	script, ok := component.(*MockScript)
	if !ok {
		t.Fatal("CreateScript didn't return MockScript")
	}

	if script.Speed != 10.5 {
		t.Errorf("Expected Speed 10.5, got %f", script.Speed)
	}

	if script.Health != 100 {
		t.Errorf("Expected Health 100, got %d", script.Health)
	}
}

func TestCreateScriptNotFound(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	component := CreateScript("DoesNotExist", nil)
	if component != nil {
		t.Error("CreateScript should return nil for non-existent script")
	}
}

func TestSerializeScript(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	RegisterScript("MockScript", mockFactory, mockSerializer)

	script := &MockScript{
		Speed:  15.0,
		Health: 200,
	}

	name, props, ok := SerializeScript(script)
	if !ok {
		t.Fatal("SerializeScript failed")
	}

	if name != "MockScript" {
		t.Errorf("Expected name 'MockScript', got '%s'", name)
	}

	if props["speed"] != float32(15.0) {
		t.Errorf("Expected speed 15.0, got %v", props["speed"])
	}

	if props["health"] != 200 {
		t.Errorf("Expected health 200, got %v", props["health"])
	}
}

func TestGetRegisteredScripts(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	RegisterScript("ScriptA", mockFactory, mockSerializer)
	RegisterScript("ScriptB", mockFactory, mockSerializer)
	RegisterScript("ScriptC", mockFactory, mockSerializer)

	scripts := GetRegisteredScripts()

	if len(scripts) != 3 {
		t.Errorf("Expected 3 scripts, got %d", len(scripts))
	}

	// Verify sorted order
	if scripts[0] != "ScriptA" || scripts[1] != "ScriptB" || scripts[2] != "ScriptC" {
		t.Errorf("Scripts not in sorted order: %v", scripts)
	}
}

func TestApplyScriptProperty(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	RegisterScriptWithApplier("MockScript", mockFactory, mockSerializer, mockApplier)

	script := &MockScript{Speed: 5.0, Health: 50}

	// Apply speed property
	ok := ApplyScriptProperty(script, "speed", float64(20.0))
	if !ok {
		t.Error("ApplyScriptProperty should return true for valid property")
	}

	if script.Speed != 20.0 {
		t.Errorf("Expected Speed 20.0 after apply, got %f", script.Speed)
	}

	// Apply health property
	ok = ApplyScriptProperty(script, "health", float64(150))
	if !ok {
		t.Error("ApplyScriptProperty should return true for valid property")
	}

	if script.Health != 150 {
		t.Errorf("Expected Health 150 after apply, got %d", script.Health)
	}

	// Try invalid property
	ok = ApplyScriptProperty(script, "nonexistent", float64(99))
	if ok {
		t.Error("ApplyScriptProperty should return false for invalid property")
	}
}

func TestRegisterScriptWithMetadata(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	fieldTypes := map[string]string{
		"target_ref": "GameObjectRef",
	}

	RegisterScriptWithMetadata("MockScript", mockFactory, mockSerializer, mockApplier, fieldTypes)

	entry, exists := scriptRegistry["MockScript"]
	if !exists {
		t.Fatal("Script not registered")
	}

	if entry.fieldTypes == nil {
		t.Error("fieldTypes not set")
	}

	if entry.fieldTypes["target_ref"] != "GameObjectRef" {
		t.Error("Field type not registered correctly")
	}
}

func TestGetScriptFieldType(t *testing.T) {
	// Clear registry for clean test
	scriptRegistry = map[string]scriptEntry{}

	fieldTypes := map[string]string{
		"target_ref": "GameObjectRef",
		"speed":      "float32",
	}

	RegisterScriptWithMetadata("MockScript", mockFactory, mockSerializer, mockApplier, fieldTypes)

	script := &MockScript{}

	fieldType := GetScriptFieldType(script, "target_ref")
	if fieldType != "GameObjectRef" {
		t.Errorf("Expected 'GameObjectRef', got '%s'", fieldType)
	}

	// Test non-existent field
	emptyType := GetScriptFieldType(script, "nonexistent")
	if emptyType != "" {
		t.Errorf("Expected empty string for non-existent field, got '%s'", emptyType)
	}
}
