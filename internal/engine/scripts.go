package engine

import "fmt"

// ScriptFactory creates a Component from JSON props.
type ScriptFactory func(props map[string]any) Component

// ScriptSerializer converts a Component back to props for JSON saving.
type ScriptSerializer func(c Component) map[string]any

type scriptEntry struct {
	factory    ScriptFactory
	serializer ScriptSerializer
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
