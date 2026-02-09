# Mirgo Engine Documentation

Welcome to the Mirgo Engine documentation. This collection of guides will help you build 3D games with Go and raylib.

## Getting Started

**New to Mirgo Engine?** Start here:

- **[Getting Started Guide](getting-started.md)** - Installation, first scene, basic concepts
- **[Editor Guide](editor.md)** - Using the built-in scene editor

## Core Guides

### Scripting

Learn to write game logic with automatic code generation:

- **[Scripting Guide](scripting.md)** - Complete guide to writing scripts
- **[Script Generation System](script-generation.md)** - How automatic code generation works

### Scene Creation

Build and configure your game world:

- **[Scene File Format](scene-format.md)** - JSON structure, components, properties
- **[Editor Guide](editor.md)** - Visual scene editing

### Reference

- **[API Reference](api-reference.md)** - Complete type and function documentation

## Quick Links

### By Task

**I want to...**

- **Create a new scene** → [Getting Started](getting-started.md#creating-a-new-scene)
- **Write a script** → [Scripting Guide](scripting.md#creating-your-first-script)
- **Add physics** → [Scene Format](scene-format.md#rigidbody)
- **Use the editor** → [Editor Guide](editor.md)
- **Understand code generation** → [Script Generation System](script-generation.md)
- **Load a GLTF model** → [Scene Format](scene-format.md#modelrenderer)
- **Build a standalone game** → [Editor Guide](editor.md#building-standalone-games)

### By Topic

**Components**
- [ModelRenderer](scene-format.md#modelrenderer) - Rendering meshes and models
- [Rigidbody](scene-format.md#rigidbody) - Physics simulation
- [BoxCollider](scene-format.md#boxcollider) - Box collision
- [SphereCollider](scene-format.md#spherecollider) - Sphere collision
- [DirectionalLight](scene-format.md#directionallight) - Lighting
- [Camera](scene-format.md#camera) - Perspective camera
- [FPSController](scene-format.md#fpscontroller) - First-person controls
- [Script](scene-format.md#script) - Custom components

**Scripting**
- [Creating Scripts](scripting.md#creating-your-first-script) - Write your first script
- [Script Structure](scripting.md#script-structure) - Required parts
- [Accessing GameObjects](scripting.md#accessing-gameobjects-and-components) - Query and modify objects
- [World Operations](scripting.md#world-operations) - Spawn, destroy, raycast
- [Input Handling](scripting.md#input-handling) - Keyboard and mouse
- [Common Patterns](scripting.md#common-patterns) - Timers, state machines, movement
- [Complete Examples](scripting.md#complete-examples) - Full script implementations

**Code Generation**
- [How It Works](script-generation.md#how-it-works) - Generation pipeline
- [Field Serialization](script-generation.md#field-serialization) - Automatic JSON conversion
- [Caching System](script-generation.md#caching-system) - Fast rebuilds
- [Troubleshooting](script-generation.md#troubleshooting) - Common issues

**Editor**
- [Controls](editor.md#editor-controls) - Keyboard and mouse shortcuts
- [Working with Objects](editor.md#working-with-objects) - Select, move, edit
- [Hot Reload](editor.md#hot-reload-cmdr) - Fast iteration
- [Building Games](editor.md#building-standalone-games) - Distribution

## Document Structure

Each guide focuses on a specific area:

| Guide | Purpose |
|-------|---------|
| [getting-started.md](getting-started.md) | Installation and first steps |
| [scripting.md](scripting.md) | Writing game logic with scripts |
| [script-generation.md](script-generation.md) | Deep dive into code generation |
| [scene-format.md](scene-format.md) | JSON scene file reference |
| [editor.md](editor.md) | Using the visual editor |
| [api-reference.md](api-reference.md) | Complete API documentation |

## Examples

### Minimal Scene

```json
{
  "objects": [
    {
      "name": "Floor",
      "components": [
        { "type": "ModelRenderer", "mesh": "cube", "meshSize": [20, 0.1, 20] },
        { "type": "BoxCollider", "size": [20, 0.1, 20] }
      ]
    },
    {
      "name": "Player",
      "position": [0, 2, 0],
      "components": [
        { "type": "Camera", "fov": 60, "isMain": true },
        { "type": "FPSController" }
      ]
    }
  ]
}
```

### Simple Script

```go
// assets/scripts/rotator.go
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

Run `make build` and the boilerplate is generated automatically!

## Contributing

Found an error in the docs? Want to add examples?

1. Docs are in `docs/*.md`
2. Edit the relevant file
3. Submit a pull request

Keep docs:
- **Clear** - Simple language, avoid jargon
- **Practical** - Show code examples
- **Complete** - Cover edge cases
- **Accurate** - Test examples before documenting

## Getting Help

- **Read the guides** - Most questions are answered here
- **Check examples** - Look at `assets/scenes/main.json` and `assets/scripts/`
- **Search docs** - Use Cmd+F in your browser
- **GitHub Issues** - Report bugs or request features

## Version

Documentation for **Mirgo Engine v0.1** (2024)

Last updated: 2024

---

**[← Back to Main README](../README.md)**
