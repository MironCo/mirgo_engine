# Mirgo Engine

![Engine Preview](docs/images/engine%20preview.jpg)

A 3D game engine written in Go, built on top of [raylib-go](https://github.com/gen2brain/raylib-go). Features an entity-component system, real-time physics, shadow mapping, a built-in scene editor, and a JSON-based scene format.

## Features

- **PBR Rendering** - Metallic/roughness materials, normal mapping, shadow mapping, tone mapping
- **Physics** - Rigidbodies, colliders (box/sphere), spatial hashing, raycasting, collision callbacks
- **Editor** - Free-fly camera, object selection, transform gizmos, inspector, hot reload, persistent preferences
- **Unity-like Scripting** - Write clean code, automatic boilerplate generation, hot reload
- **Scene Management** - JSON scenes, hierarchies, component system, save/load, multi-scene support

## Quick Start

```bash
# Clone and run
git clone https://github.com/yourusername/mirgo_engine.git
cd mirgo_engine
make run
```

**Controls:**
- **Editor Mode**: Right-click + WASD to fly, left-click to select, Cmd/Ctrl+P to play
- **Game Mode**: WASD to move, mouse to look, Space to jump

## Writing Your First Script

Create `assets/scripts/rotator.go`:

```go
package scripts

import "test3d/internal/engine"

type Rotator struct {
    engine.BaseComponent
    Speed float32
}

func (r *Rotator) Update(deltaTime float32) {
    g := r.GetGameObject()
    if g == nil {
        return
    }
    g.Transform.Rotation.Y += r.Speed * deltaTime
}
```

That's it! Run `make build` and the engine automatically generates factory, serializer, and registration code.

### Collision Callbacks

Scripts can react to physics collisions:

```go
type Collectible struct {
    engine.BaseComponent
    Points    float32
    TargetTag string
}

func (c *Collectible) OnCollisionEnter(other *engine.GameObject) {
    if other.HasTag(c.TargetTag) {
        fmt.Printf("Collected! +%.0f points\n", c.Points)
        g := c.GetGameObject()
        g.Scene.World.Destroy(g)
    }
}

func (c *Collectible) OnCollisionExit(other *engine.GameObject) {
    // Called when collision ends
}
```

Use in your scene:

```json
{
  "type": "Script",
  "name": "Rotator",
  "props": { "speed": 45 }
}
```

## Documentation

- **[Getting Started](docs/getting-started.md)** - First steps, creating scenes
- **[Scripting Guide](docs/scripting.md)** - Writing game logic
- **[Script Generation](docs/script-generation.md)** - How automatic code generation works
- **[Scene Format](docs/scene-format.md)** - JSON structure reference
- **[Editor Guide](docs/editor.md)** - Using the visual editor
- **[API Reference](docs/api-reference.md)** - Complete type documentation

**[📚 Full Documentation Index](docs/README.md)**

## Requirements

- Go 1.24+
- GCC / C compiler (for CGO/raylib)
- OpenGL 3.3+ capable GPU

## Key Concepts

### Entity-Component System

```go
// Create a GameObject
obj := engine.NewGameObject("MyCube")
obj.Transform.Position = rl.Vector3{X: 0, Y: 5, Z: 0}

// Add components
obj.AddComponent(components.NewModelRenderer(model, rl.Red))
obj.AddComponent(components.NewBoxCollider(1, 1, 1))
obj.AddComponent(components.NewRigidbody())

// Query components
rb := engine.GetComponent[*components.Rigidbody](obj)
rb.Velocity.Y = 10  // Apply upward force
```

### Scene Queries

```go
// Find by name
player := scene.FindByName("Player")

// Find by tag
enemies := scene.FindByTag("enemy")

// Iterate all objects
for _, obj := range scene.GameObjects {
    // ...
}
```

### World Operations

```go
// Spawn objects
world.SpawnObject(newObject)

// Destroy objects
world.Destroy(target)

// Raycast
hit, ok := world.Raycast(origin, direction, maxDistance)
if ok {
    fmt.Printf("Hit %s at %.2f units\n", hit.GameObject.Name, hit.Distance)
}
```

## Project Structure

```
cmd/
  test3d/          - Entry point
  gen-scripts/     - Script code generator
internal/
  engine/          - Core ECS framework
  components/      - Built-in components (Camera, Rigidbody, etc.)
  scripts/         - Generated scripts (git-ignored)
  physics/         - Physics engine
  world/           - Scene + renderer
  game/            - Game loop + editor
assets/
  scripts/         - Source scripts (clean, no boilerplate)
  scenes/          - JSON scene files
  models/          - GLTF 3D models
  shaders/         - GLSL shaders
docs/              - Documentation
utilities/         - Rust CLI tools (build, flipnormals, newscript)
```

## Editor Shortcuts

| Shortcut | Action |
|----------|--------|
| **Cmd/Ctrl+P** | Toggle play mode (resets scene) |
| **Cmd/Ctrl+Shift+P** | Pause/resume (preserves scene state) |
| **Cmd/Ctrl+S** | Save scene |
| **Cmd/Ctrl+R** | Hot reload (regenerate + rebuild) |
| **Cmd/Ctrl+B** | Build standalone game |
| **Ctrl+Z** | Undo transform |
| **F1** | Debug overlay |
| **Delete/Backspace** | Delete selected object |
| **Double-click scene** | Open scene in asset browser |

## Building Standalone Games

### macOS .app Bundle

```bash
# In editor: Cmd/Ctrl+B
# Or via CLI:
./mirgo-utils build MyGame
```

Creates `build/MyGame.app` ready to distribute.

### Cross-Platform

```bash
make build-windows   # Windows .exe
make build-linux     # Linux binary
```

## Utilities

The `mirgo-utils` tool provides:

```bash
./mirgo-utils newscript MyScript    # Create script template
./mirgo-utils flipnormals model.gltf # Fix inverted normals
./mirgo-utils build MyGame          # Build game bundle
```

Build with: `cd utilities && cargo build --release`

## Architecture

```
Game
 └── World
      ├── Scene (GameObjects, lifecycle)
      ├── PhysicsWorld (collision, gravity)
      └── Renderer (shadows, lighting)

GameObject
 ├── Transform (position, rotation, scale)
 ├── Tags []string
 └── Components []Component (Camera, Rigidbody, Scripts, etc.)
```

Components access world through: `GetGameObject().Scene.World`

## Examples

See complete examples in the documentation:

- [Rotator Script](docs/scripting.md#rotator-script) - Simple rotation
- [Shooter Script](docs/scripting.md#shooter-script) - Projectile spawning
- [Collectible Script](docs/scripting.md#collectible-script) - Pickup system
- [Enemy AI](docs/scripting.md#state-machine) - State machine pattern

## License

[MIT License](LICENSE)

## Credits

Built with:
- [raylib](https://www.raylib.com/) - Graphics and input
- [raylib-go](https://github.com/gen2brain/raylib-go) - Go bindings
