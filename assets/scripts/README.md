# Scripts

This directory contains **clean script source files** without boilerplate - just your game logic.

## Unity-like Workflow

Write clean scripts here, and the engine automatically generates all the boilerplate for you.

### How It Works

1. **Write**: Create `.go` files here with just your struct and logic
2. **Build**: Run `make build`, `make run`, or press Cmd+R in the editor
3. **Generated**: Full implementations appear in `internal/scripts/` (git-ignored)

### Example Script

```go
package scripts

import "test3d/internal/engine"

type MyScript struct {
	engine.BaseComponent
	Speed float32
	Health int
}

func (m *MyScript) Update(deltaTime float32) {
	g := m.GetGameObject()
	if g == nil {
		return
	}
	// Your game logic here
}
```

### What Gets Generated

The system automatically creates:
- `init()` function with `engine.RegisterScript()`
- Factory function that parses JSON properties with type conversions
- Serializer function for saving scenes

### Field Serialization

- **Exported fields** (capitalized): Automatically serialized
- **JSON names**: Converted to snake_case (`Speed` → `"speed"`, `MaxHealth` → `"max_health"`)
- **Private fields** (lowercase): Ignored, won't serialize

### Supported Types

Automatic JSON conversion for:
- `float32`, `float64`
- `int`, `int32`, `int64`
- `bool`
- `string`

### Using in Scenes

Reference your scripts in scene JSON files:

```json
{
  "type": "Script",
  "name": "MyScript",
  "props": {
    "speed": 5.0,
    "health": 100
  }
}
```

### Creating New Scripts

Option 1 - Manual:
```bash
# Just create a new .go file in this directory
touch assets/scripts/enemy_ai.go
```

Option 2 - Using mirgo-utils:
```bash
./mirgo-utils newscript EnemyAI
# Creates assets/scripts/enemy_ai.go with basic template
```

### Caching

The generator uses SHA256 hashing to skip unchanged scripts. Hash files (`.hash`) are stored alongside generated files in `internal/scripts/`.

### Version Control

- ✅ **Commit**: `assets/scripts/*.go` (source files)
- ❌ **Ignore**: `internal/scripts/` (generated files)
