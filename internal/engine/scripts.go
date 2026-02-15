package engine

import "fmt"

// ScriptFactory creates a Component from JSON props.
type ScriptFactory func(props map[string]any) Component

// ScriptSerializer converts a Component back to props for JSON saving.
type ScriptSerializer func(c Component) map[string]any

// ScriptApplier applies a single property value to a script component.
// Returns true if the property was applied successfully.
type ScriptApplier func(c Component, propName string, value any) bool

type scriptEntry struct {
	factory    ScriptFactory
	serializer ScriptSerializer
	applier    ScriptApplier
	fieldTypes map[string]string // Map of field name -> type (e.g., "target_button" -> "GameObjectRef")
}

var scriptRegistry = map[string]scriptEntry{}

// RegisterScript registers a named script with a factory and optional serializer.
// The serializer is used when saving the scene back to JSON.
func RegisterScript(name string, factory ScriptFactory, serializer ScriptSerializer) {
	if _, exists := scriptRegistry[name]; exists {
		panic(fmt.Sprintf("script %q already registered", name))
	}
	scriptRegistry[name] = scriptEntry{factory: factory, serializer: serializer}
}

// RegisterScriptWithApplier registers a script with factory, serializer, and property applier.
// The applier enables live property editing in the editor.
func RegisterScriptWithApplier(name string, factory ScriptFactory, serializer ScriptSerializer, applier ScriptApplier) {
	if _, exists := scriptRegistry[name]; exists {
		panic(fmt.Sprintf("script %q already registered", name))
	}
	scriptRegistry[name] = scriptEntry{factory: factory, serializer: serializer, applier: applier, fieldTypes: nil}
}

// RegisterScriptWithMetadata registers a script with full metadata including field types.
// This is used by the code generator to provide type information for inspector UI.
func RegisterScriptWithMetadata(name string, factory ScriptFactory, serializer ScriptSerializer, applier ScriptApplier, fieldTypes map[string]string) {
	if _, exists := scriptRegistry[name]; exists {
		panic(fmt.Sprintf("script %q already registered", name))
	}
	scriptRegistry[name] = scriptEntry{factory: factory, serializer: serializer, applier: applier, fieldTypes: fieldTypes}
}

// CreateScript looks up a registered script by name and creates it with the given props.
func CreateScript(name string, props map[string]any) Component {
	entry, ok := scriptRegistry[name]
	if !ok {
		return nil
	}
	return entry.factory(props)
}

// SerializeScript tries to serialize a component by checking all registered scripts.
// Returns (name, props, true) if found, ("", nil, false) otherwise.
func SerializeScript(c Component) (string, map[string]any, bool) {
	for name, entry := range scriptRegistry {
		if entry.serializer == nil {
			continue
		}
		props := entry.serializer(c)
		if props != nil {
			return name, props, true
		}
	}
	return "", nil, false
}

// GetRegisteredScripts returns a sorted list of all registered script names.
func GetRegisteredScripts() []string {
	names := make([]string, 0, len(scriptRegistry))
	for name := range scriptRegistry {
		names = append(names, name)
	}
	// Sort for consistent ordering in UI
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

// ApplyScriptProperty applies a property value to a script component.
// Returns true if the property was applied successfully.
func ApplyScriptProperty(c Component, propName string, value any) bool {
	for _, entry := range scriptRegistry {
		if entry.applier == nil {
			continue
		}
		if entry.applier(c, propName, value) {
			return true
		}
	}
	return false
}

// HasScriptApplier checks if a component has an applier registered.
func HasScriptApplier(c Component) bool {
	for _, entry := range scriptRegistry {
		if entry.applier == nil {
			continue
		}
		// Check if the serializer recognizes this component
		if entry.serializer != nil && entry.serializer(c) != nil {
			return entry.applier != nil
		}
	}
	return false
}

// GetScriptFieldType returns the type of a script field (e.g., "GameObjectRef").
// Returns empty string if the field type is not registered.
func GetScriptFieldType(c Component, fieldName string) string {
	for _, entry := range scriptRegistry {
		if entry.serializer == nil || entry.fieldTypes == nil {
			continue
		}
		// Check if this is the right script
		if entry.serializer(c) != nil {
			return entry.fieldTypes[fieldName]
		}
	}
	return ""
}
