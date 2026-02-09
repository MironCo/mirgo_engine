# Script Generation System

Mirgo Engine uses automatic code generation to eliminate boilerplate from game scripts, providing a Unity-like developer experience.

## Overview

**What you write** (in `assets/scripts/`):
```go
type MyScript struct {
    engine.BaseComponent
    Speed float32
}

func (m *MyScript) Update(deltaTime float32) {
    // Your game logic
}
```

**What gets generated** (in `internal/scripts/`):
- Factory function (creates component from JSON)
- Serializer function (saves component to JSON)
- Registration code (`init()` function)
- Type conversions (JSON ↔ Go types)

## How It Works

### 1. Source Files

Place clean scripts in `assets/scripts/`:

```
assets/scripts/
├── rotator.go
├── enemy_ai.go
└── collectible.go
```

These files contain **only**:
- Struct definition with `engine.BaseComponent`
- Your logic methods (`Update`, `Start`, etc.)
- No factory, no serializer, no registration

### 2. Build Trigger

Generation runs automatically when you:
- Run `make build` or `make run`
- Press **Cmd+R** in the editor (hot reload)
- Manually run `go run ./cmd/gen-scripts`

### 3. Generation Process

The `cmd/gen-scripts` tool:

1. **Parses** your source files using Go's `go/ast` package
2. **Extracts** struct name and exported fields
3. **Generates** factory and serializer functions
4. **Writes** complete implementation to `internal/scripts/`
5. **Caches** SHA256 hash to skip unchanged files

### 4. Output

Generated files in `internal/scripts/`:

```
internal/scripts/
├── doc.go              # Package documentation (preserved)
├── rotator.go          # Source + generated boilerplate
├── rotator.go.hash     # SHA256 for caching
├── enemy_ai.go
├── enemy_ai.go.hash
└── ...
```

## Field Serialization

### Exported Fields

Only **exported** (capitalized) fields are serialized:

```go
type MyScript struct {
    engine.BaseComponent

    Speed    float32  // ✅ Serialized as "speed"
    Health   int      // ✅ Serialized as "health"
    MaxAmmo  int      // ✅ Serialized as "max_ammo"

    target   string   // ❌ Private, not serialized
    internal bool     // ❌ Private, not serialized
}
```

### JSON Name Conversion

Field names are converted to **snake_case**:

| Go Field | JSON Property |
|----------|---------------|
| `Speed` | `"speed"` |
| `MaxHealth` | `"max_health"` |
| `AttackDamage` | `"attack_damage"` |
| `IsEnabled` | `"is_enabled"` |

### Type Conversions

The generator handles automatic type conversions between JSON and Go:

| Go Type | JSON Type | Factory Conversion |
|---------|-----------|-------------------|
| `float32` | `float64` | `float32(v)` |
| `float64` | `float64` | `v` (direct) |
| `int` | `float64` | `int(v)` |
| `int32` | `float64` | `int32(v)` |
| `int64` | `float64` | `int64(v)` |
| `bool` | `bool` | `v` (direct) |
| `string` | `string` | `v` (direct) |

**Why float64?** JSON numbers are always decoded as `float64` in Go.

## Generated Code Structure

### Factory Function

Creates your script from JSON properties:

```go
func myScriptFactory(props map[string]any) engine.Component {
    script := &MyScript{}
    if v, ok := props["speed"].(float64); ok {
        script.Speed = float32(v)
    }
    if v, ok := props["health"].(float64); ok {
        script.Health = int(v)
    }
    return script
}
```

**Default values**: Fields use Go's zero values (0, false, "") if not in JSON.

### Serializer Function

Converts your script back to JSON:

```go
func myScriptSerializer(c engine.Component) map[string]any {
    s, ok := c.(*MyScript)
    if !ok {
        return nil
    }
    return map[string]any{
        "speed":  s.Speed,
        "health": s.Health,
    }
}
```

### Registration

Registers the script with the engine:

```go
func init() {
    engine.RegisterScript("MyScript", myScriptFactory, myScriptSerializer)
}
```

The script name matches your struct name.

## Caching System

### Hash Files

Each generated script has a corresponding `.hash` file:

```
internal/scripts/
├── rotator.go
├── rotator.go.hash    # SHA256 of assets/scripts/rotator.go
```

### How Caching Works

1. Generator reads source file
2. Computes SHA256 hash of contents
3. Compares with cached hash
4. **Skips generation** if hashes match
5. **Regenerates** if different or missing

### Cache Invalidation

Cache is invalidated when:
- Source file is modified
- Source file is newly created
- Hash file is deleted
- Output file is missing

### Benefits

- **Fast rebuilds**: Only changed scripts regenerate
- **Clean output**: No "nothing changed" spam
- **Reliable**: Content-based, not timestamp-based

## File Organization

### Version Control

**Commit** (tracked in git):
- ✅ `assets/scripts/*.go` - Source files
- ✅ `.gitignore` - Contains `internal/scripts/`

**Ignore** (git-ignored):
- ❌ `internal/scripts/` - Generated files
- ❌ `internal/scripts/*.hash` - Cache files

### Why This Structure?

- **Source of truth**: Clean scripts in `assets/scripts/`
- **Reproducible**: Anyone can regenerate from source
- **No conflicts**: Generated files never in git
- **Clean diffs**: Only meaningful changes tracked

## Generator Implementation

### Tool Location

```
cmd/gen-scripts/main.go
```

Built-in Go tool, not external dependency.

### AST Parsing

Uses Go's standard library:

```go
import (
    "go/ast"
    "go/parser"
    "go/token"
)
```

**Why AST?** Handles all Go syntax correctly, no fragile regex parsing.

### Workflow

```
Source File
    ↓
go/parser → AST
    ↓
Extract struct & fields
    ↓
Generate factory & serializer
    ↓
Write source + boilerplate
    ↓
Write hash file
```

## Usage in Scenes

Once generated, reference scripts in scene JSON:

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

The `name` must match your struct name. Properties use snake_case JSON names.

## Creating New Scripts

### Option 1: Manual

```bash
touch assets/scripts/my_new_script.go
```

Edit with your struct and logic, then run `make build`.

### Option 2: Using mirgo-utils

```bash
./mirgo-utils newscript MyNewScript
```

Creates `assets/scripts/my_new_script.go` with basic template.

## Limitations

### Supported Field Types

Only these types auto-serialize:
- Numeric: `float32`, `float64`, `int`, `int32`, `int64`
- `bool`
- `string`

**Unsupported**:
- Arrays/slices
- Maps
- Structs
- Pointers (except `engine.BaseComponent`)
- Custom types

For complex data, use manual serialization or multiple simple fields.

### Single Struct Per File

The generator expects:
- One struct definition per file
- Struct embeds `engine.BaseComponent`
- Struct is exported (capitalized name)

Multiple structs in one file: only the first is used.

### No Default Values

Generated factories use Go zero values. For custom defaults:

```go
func (m *MyScript) Start() {
    if m.Speed == 0 {
        m.Speed = 5.0  // Apply default after creation
    }
}
```

## Troubleshooting

### Script Not Found

**Error**: `Script "MyScript" not found in registry`

**Fix**:
1. Check struct name matches scene JSON `name` field
2. Run `make build` to regenerate
3. Verify source file is in `assets/scripts/`

### Properties Not Loading

**Error**: Script created but properties are zero values

**Fix**:
1. Check JSON property names are snake_case
2. Check field types match (e.g., `float32` not `int`)
3. Verify fields are exported (capitalized)

### Generation Fails

**Error**: Parse error or generation fails

**Fix**:
1. Check Go syntax is valid: `go build ./assets/scripts/...`
2. Ensure struct embeds `engine.BaseComponent`
3. Check for unsupported field types

### Cache Not Working

**Symptom**: Scripts regenerate every time

**Fix**:
1. Check `.hash` files exist in `internal/scripts/`
2. Verify file permissions (should be readable/writable)
3. Delete all `.hash` files to reset

## Advanced Usage

### Multiple Scripts

Create multiple scripts in `assets/scripts/`:

```
assets/scripts/
├── enemy_ai.go
├── player_controller.go
├── collectible.go
└── door_trigger.go
```

All generate automatically in one build.

### Script Composition

Attach multiple scripts to one object:

```json
{
  "components": [
    { "type": "Script", "name": "Health", "props": { "max_hp": 100 } },
    { "type": "Script", "name": "Movement", "props": { "speed": 5 } },
    { "type": "Script", "name": "Weapon", "props": { "damage": 10 } }
  ]
}
```

Each script is independent.

### Hot Reload Workflow

1. Edit script in `assets/scripts/my_script.go`
2. Press **Cmd+R** in editor
3. Generator runs automatically
4. Engine rebuilds
5. Scene reloads with new script behavior

Fast iteration cycle!

## See Also

- [Scripting Guide](scripting.md) - Writing game logic
- [Scene Format](scene-format.md) - Using scripts in scenes
- [Getting Started](getting-started.md) - First script tutorial
