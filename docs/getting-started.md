# Getting Started with Mirgo Engine

This guide walks you through creating your first scene with Mirgo Engine, from installation to a playable game.

## Prerequisites

- **Go 1.24+** - [Download](https://go.dev/dl/)
- **GCC / C compiler** - Required for CGO (raylib binding)
  - macOS: `xcode-select --install`
  - Linux: `sudo apt install build-essential`
  - Windows: [MinGW-w64](https://www.mingw-w64.org/)
- **OpenGL 3.3+** capable GPU

## Installation

Clone the repository and build:

```bash
git clone https://github.com/yourusername/mirgo_engine.git
cd mirgo_engine
make run
```

This compiles and launches the engine with the default scene.

## Your First Scene

### Understanding the Editor

When you launch the engine, you start in **Game Mode**. Press `Cmd/Ctrl+P` to enter **Editor Mode**.

**Editor Controls:**
- **Right Mouse + Drag** - Look around
- **Right Mouse + WASD** - Fly through the scene
- **Scroll Wheel** - Adjust fly speed
- **Left Click** - Select objects
- **Ctrl+S** - Save scene
- **Cmd/Ctrl+P** - Toggle between game and editor mode

### Creating a New Scene

Scenes are JSON files in `assets/scenes/`. Create a minimal scene:

```json
{
  "objects": [
    {
      "name": "Floor",
      "position": [0, 0, 0],
      "scale": [20, 0.1, 20],
      "components": [
        {
          "type": "ModelRenderer",
          "mesh": "cube",
          "meshSize": [1, 1, 1],
          "color": "Gray"
        },
        {
          "type": "BoxCollider",
          "size": [20, 0.1, 20]
        }
      ]
    },
    {
      "name": "Player",
      "tags": ["player"],
      "position": [0, 2, 0],
      "components": [
        {
          "type": "Camera",
          "fov": 60,
          "isMain": true
        },
        {
          "type": "FPSController",
          "moveSpeed": 5,
          "lookSpeed": 0.3,
          "jumpStrength": 8
        },
        {
          "type": "BoxCollider",
          "size": [0.6, 1.8, 0.6]
        },
        {
          "type": "Rigidbody",
          "mass": 1,
          "useGravity": true,
          "isKinematic": true
        }
      ]
    },
    {
      "name": "Sun",
      "position": [0, 10, 0],
      "components": [
        {
          "type": "DirectionalLight",
          "direction": [0.5, -1, 0.3],
          "intensity": 1.0
        }
      ]
    }
  ]
}
```

Save this as `assets/scenes/myscene.json`, then modify `cmd/test3d/main.go` to load it:

```go
world.LoadScene("assets/scenes/myscene.json")
```

### Adding Objects

Let's add a spinning cube to the scene. Add this to the `objects` array:

```json
{
  "name": "SpinningCube",
  "position": [3, 1, 0],
  "components": [
    {
      "type": "ModelRenderer",
      "mesh": "cube",
      "meshSize": [1, 1, 1],
      "color": "Red",
      "metallic": 0.8,
      "roughness": 0.2
    },
    {
      "type": "BoxCollider",
      "size": [1, 1, 1]
    },
    {
      "type": "Rigidbody",
      "mass": 1,
      "bounciness": 0.5,
      "useGravity": true
    },
    {
      "type": "Script",
      "name": "Rotator",
      "props": { "speed": 45 }
    }
  ]
}
```

This creates a red metallic cube that:
- Renders with PBR materials (metallic, slightly reflective)
- Has physics (falls, bounces)
- Rotates at 45 degrees per second (via the Rotator script)

### Adding a GLTF Model

To use a 3D model instead of a primitive:

```json
{
  "name": "Duck",
  "position": [0, 1, 5],
  "scale": [2, 2, 2],
  "components": [
    {
      "type": "ModelRenderer",
      "model": "assets/models/rubber_duck_toy_1k.gltf"
    },
    {
      "type": "BoxCollider",
      "size": [1, 1, 1]
    },
    {
      "type": "Rigidbody",
      "mass": 0.5,
      "bounciness": 0.8
    }
  ]
}
```

### Object Hierarchies

Objects can have children that inherit their parent's transform:

```json
{
  "name": "Platform",
  "position": [5, 2, 0],
  "components": [
    {
      "type": "ModelRenderer",
      "mesh": "cube",
      "meshSize": [3, 0.2, 3],
      "color": "Blue"
    }
  ],
  "children": [
    {
      "name": "Pillar1",
      "position": [-1, -1, -1],
      "components": [
        {
          "type": "ModelRenderer",
          "mesh": "cube",
          "meshSize": [0.3, 2, 0.3],
          "color": "Gray"
        }
      ]
    },
    {
      "name": "Pillar2",
      "position": [1, -1, 1],
      "components": [
        {
          "type": "ModelRenderer",
          "mesh": "cube",
          "meshSize": [0.3, 2, 0.3],
          "color": "Gray"
        }
      ]
    }
  ]
}
```

Child positions are relative to their parent. When the Platform moves, the Pillars move with it.

## Game Mode Controls

Press `Cmd/Ctrl+P` to enter game mode and test your scene:

| Key | Action |
|-----|--------|
| WASD | Move |
| Mouse | Look |
| Space | Jump |
| Left Click | Shoot projectile (if Shooter script attached) |
| Right Click | Delete targeted object |
| F1 | Toggle debug overlay |

## Building a Standalone Game

When you're ready to distribute:

1. Press `Cmd/Ctrl+B` in the editor, or
2. Run `./mirgo-utils build MyGame`

This creates a standalone `.app` bundle (macOS) with all assets included.

## Next Steps

- Read the [API Reference](api-reference.md) for all available types and methods
- Learn to [write custom scripts](scripting.md) for game logic
- Explore the example scene in `assets/scenes/main.json`
