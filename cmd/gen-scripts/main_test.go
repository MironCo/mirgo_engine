package main

import (
	"strings"
	"testing"
)

func TestParseScriptBasic(t *testing.T) {
	source := `package scripts

type TestScript struct {
	Speed float32
	Name  string
}
`

	script, err := parseScript(source)
	if err != nil {
		t.Fatalf("parseScript failed: %v", err)
	}

	if script.Name != "TestScript" {
		t.Errorf("Expected name 'TestScript', got '%s'", script.Name)
	}

	if len(script.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(script.Fields))
	}

	// Check Speed field
	if script.Fields[0].Name != "Speed" || script.Fields[0].Type != "float32" {
		t.Errorf("Speed field incorrect: %+v", script.Fields[0])
	}

	// Check Name field
	if script.Fields[1].Name != "Name" || script.Fields[1].Type != "string" {
		t.Errorf("Name field incorrect: %+v", script.Fields[1])
	}
}

func TestParseScriptWithGameObjectRef(t *testing.T) {
	source := `package scripts

import "test3d/internal/engine"

type TargetScript struct {
	engine.BaseComponent
	TargetRef engine.GameObjectRef
	Speed     float32
}
`

	script, err := parseScript(source)
	if err != nil {
		t.Fatalf("parseScript failed: %v", err)
	}

	// Should have 2 fields (BaseComponent is embedded, so skipped)
	if len(script.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(script.Fields))
	}

	// Find TargetRef field
	var foundRef bool
	for _, field := range script.Fields {
		if field.Name == "TargetRef" {
			foundRef = true
			if field.Type != "GameObjectRef" {
				t.Errorf("Expected type 'GameObjectRef', got '%s'", field.Type)
			}
			if field.JSONName != "target_ref" {
				t.Errorf("Expected JSONName 'target_ref', got '%s'", field.JSONName)
			}
		}
	}

	if !foundRef {
		t.Error("GameObjectRef field not found")
	}
}

func TestParseScriptSkipsPrivateFields(t *testing.T) {
	source := `package scripts

type TestScript struct {
	PublicField  string
	privateField int
}
`

	script, err := parseScript(source)
	if err != nil {
		t.Fatalf("parseScript failed: %v", err)
	}

	// Should only have PublicField
	if len(script.Fields) != 1 {
		t.Fatalf("Expected 1 field (private field should be skipped), got %d", len(script.Fields))
	}

	if script.Fields[0].Name != "PublicField" {
		t.Errorf("Expected 'PublicField', got '%s'", script.Fields[0].Name)
	}
}

func TestParseScriptSkipsEmbeddedStructs(t *testing.T) {
	source := `package scripts

import "test3d/internal/engine"

type TestScript struct {
	engine.BaseComponent
	Speed float32
}
`

	script, err := parseScript(source)
	if err != nil {
		t.Fatalf("parseScript failed: %v", err)
	}

	// Should only have Speed (BaseComponent is embedded)
	if len(script.Fields) != 1 {
		t.Fatalf("Expected 1 field (embedded struct should be skipped), got %d", len(script.Fields))
	}

	if script.Fields[0].Name != "Speed" {
		t.Errorf("Expected 'Speed', got '%s'", script.Fields[0].Name)
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Speed", "speed"},
		{"TargetRef", "target_ref"},
		{"IsActive", "is_active"},
		{"HTTPServer", "h_t_t_p_server"},
		{"name", "name"},
	}

	for _, test := range tests {
		result := toSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("toSnakeCase(%s): expected '%s', got '%s'", test.input, test.expected, result)
		}
	}
}

func TestGetTypeConversion(t *testing.T) {
	tests := []struct {
		fieldType      string
		expectedGoType string
		expectedConv   string
	}{
		{"float32", "float64", "float32(%s)"},
		{"float64", "float64", "%s"},
		{"int", "float64", "int(%s)"},
		{"int32", "float64", "int32(%s)"},
		{"bool", "bool", "%s"},
		{"string", "string", "%s"},
		{"GameObjectRef", "float64", "engine.GameObjectRef{UID: uint64(%s)}"},
	}

	for _, test := range tests {
		goType, conv := getTypeConversion(test.fieldType)
		if goType != test.expectedGoType {
			t.Errorf("getTypeConversion(%s): expected goType '%s', got '%s'", test.fieldType, test.expectedGoType, goType)
		}
		if conv != test.expectedConv {
			t.Errorf("getTypeConversion(%s): expected conversion '%s', got '%s'", test.fieldType, test.expectedConv, conv)
		}
	}
}

func TestExprToString(t *testing.T) {
	source := `package scripts

type TestScript struct {
	SimpleField   string
	SliceField    []int
	PointerField  *bool
	QualifiedField engine.GameObjectRef
}
`

	script, err := parseScript(source)
	if err != nil {
		t.Fatalf("parseScript failed: %v", err)
	}

	expectedTypes := map[string]string{
		"SimpleField":    "string",
		"SliceField":     "[]int",
		"PointerField":   "*bool",
		"QualifiedField": "GameObjectRef", // Stripped package prefix
	}

	for _, field := range script.Fields {
		expected, exists := expectedTypes[field.Name]
		if !exists {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}
		if field.Type != expected {
			t.Errorf("Field %s: expected type '%s', got '%s'", field.Name, expected, field.Type)
		}
	}
}

func TestParseScriptNoStruct(t *testing.T) {
	source := `package scripts

func MyFunction() {
	// Just a function, no struct
}
`

	_, err := parseScript(source)
	if err == nil {
		t.Error("Expected error when parsing file with no struct, got nil")
	}

	if !strings.Contains(err.Error(), "no struct definition found") {
		t.Errorf("Expected 'no struct definition found' error, got: %v", err)
	}
}

func TestParseScriptMultipleTypes(t *testing.T) {
	source := `package scripts

type FirstScript struct {
	Speed float32
}

type SecondScript struct {
	Health int
}
`

	script, err := parseScript(source)
	if err != nil {
		t.Fatalf("parseScript failed: %v", err)
	}

	// Should parse a struct (order may vary based on AST traversal)
	// Just verify it finds one of them
	if script.Name != "FirstScript" && script.Name != "SecondScript" {
		t.Errorf("Expected 'FirstScript' or 'SecondScript', got '%s'", script.Name)
	}
}
