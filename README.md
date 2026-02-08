# Mirgo Engine

A 3D game engine written in Go, built on top of [raylib-go](https://github.com/gen2brain/raylib-go). Features an entity-component system, real-time physics, shadow mapping, a built-in scene editor, and a JSON-based scene format.

## Features

### Rendering

- **PBR-style Materials** - Metallic/roughness workflow with per-object properties
- **Normal Mapping** - Tangent-space normal maps with TBN matrix, auto-detected from GLTF models
- **Shadow Mapping** - 2048x2048 depth map, directional light, PCF 5x5 soft shadows, slope-scaled bias
- **Lighting Model** - Wrap diffuse, Blinn-Phong specular, fresnel, rim lighting, emissive
- **Post-processing** - Reinhard tone mapping, gamma correction
- **GLTF Support** - Load models with albedo textures, normal maps, and embedded materials

### Entity-Component System

- **GameObjects** - Transform (position, rotation, scale), tags, component list
- **Generic Lookups** - `GetComponent[T](gameObject)` with type safety
- **Scene Queries** - `FindByTag()`, `FindByName()`, iterate all objects
- **Lifecycle Hooks** - `Start()`, `Update(deltaTime)` per component

### Physics

- **Rigidbodies** - Mass, velocity, bounciness, friction, gravity toggle, kinematic mode
- **Colliders** - Box and sphere shapes with offset support
- **Collision Detection** - AABB intersection, spatial hashing for broad-phase
- **Raycasting** - Ray-box and ray-sphere intersection, returns hit info

### Editor

- **Free-fly Camera** - Right-click + WASD, scroll to adjust speed
- **Object Selection** - Click to select, raycast-based picking
- **Transform Gizmos** - Axis-constrained dragging for move/rotate/scale
- **Inspector Panel** - Edit component properties, add/remove components
- **Asset Browser** - Drag-and-drop GLTF models into the scene
- **Hierarchy Panel** - View and select scene objects
- **Undo System** - Ctrl+Z to undo transform changes
- **Scene Save** - Ctrl+S saves to JSON
- **Game Build** - Cmd/Ctrl+B builds standalone .app bundle

### Scripting

- **Script Registry** - Register custom components by name
- **JSON Integration** - Reference scripts from scene files with properties
- **Code Generation** - `mirgo-utils newscript` scaffolds new scripts
- **World Access** - Spawn, destroy, raycast from any component

## Requirements

- Go 1.24+
- GCC / C compiler (raylib uses CGO)
- OpenGL 3.3+ capable GPU

## Build & Run

```bash
make run      # build and run
make build    # build binary only
make clean    # remove built binaries
```

Or directly:

```bash
go run ./cmd/test3d
```

## Controls

### Game Mode

| Key | Action |
|-----|--------|
| WASD | Move |
| Mouse | Look |
| Space | Jump |
| Left Click | Shoot projectile |
| Right Click | Delete targeted object |
| Arrow Keys / Q / E | Adjust light direction |
| F1 | Toggle debug overlay |
| Cmd/Ctrl+P | Toggle editor mode |

### Editor Mode

| Key | Action |
|-----|--------|
| Right Mouse + Drag | Look around |
| Right Mouse + WASD | Fly |
| Scroll Wheel | Adjust fly speed |
| Left Click | Select object |
| Drag Gizmo Arrow | Move object along axis |
| Ctrl+S | Save scene to JSON |
| Cmd/Ctrl+B | Build game (.app bundle on macOS) |
| Cmd/Ctrl+P | Return to game mode |

## Project Structure

```
cmd/
  test3d/main.go               Entry point
utilities/                      Rust CLI tools (see below)
  src/main.rs                   mirgo-utils (subcommand tool)
internal/
  engine/                       Core framework
    component.go                Component interface + BaseComponent
    gameobject.go               GameObject (Transform, Tags, Components)
    scene.go                    Scene (object management, lifecycle)
    scripts.go                  Script registry (RegisterScript, CreateScript)
    world_access.go             WorldAccess interface (Raycast, Spawn, Destroy)
  components/                   Built-in components
    camera.go                   Perspective camera from FPSController
    fpscontroller.go            First-person movement + mouse look
    modelrenderer.go            Model/mesh rendering with shader support
    directionallight.go         Directional light for shadow casting
    rigidbody.go                Physics body (mass, velocity, bounciness)
    boxcollider.go              Box collision shape
    spherecollider.go           Sphere collision shape
    cubeanimator.go             Animated movement + rotation
    scripts/                    User scripts (one file per script)
      rotator.go                Spins objects around Y axis
      shooter.go                Projectile spawning + object deletion
  physics/                      Physics simulation
    world.go                    Gravity, spatial hashing, collision pipeline
    aabb.go                     AABB intersection + resolution
    raycast.go                  Ray-box/sphere intersection
  world/                        World management
    world.go                    Scene + physics + renderer initialization
    renderer.go                 Shadow map + main render pass
    scenefile.go                JSON scene loading/saving
    player_collision.go         Player ground check + collision
  game/                         Application layer
    game.go                     Game loop, input, debug UI
    editor.go                   Editor mode (camera, gizmos, inspector)
  assets/
    assets.go                   Cached model/texture loader
assets/
  scenes/main.json              Default scene (floor, cubes, duck, light)
  shaders/                      GLSL 330 shaders
    lighting.vs/fs              Main render (lighting + shadows)
    shadow.vs/fs                Shadow depth pass
    depth.vs/fs                 Depth pass
  models/                       3D models (GLTF)
```

## Scene File Format

Scenes are JSON files in `assets/scenes/`. Objects have a transform and a list of components:

```json
{
  "objects": [
    {
      "name": "MyCube",
      "tags": ["enemy", "destructible"],
      "position": [0, 5, 0],
      "rotation": [0, 0, 0],
      "scale": [1, 1, 1],
      "components": [
        {
          "type": "ModelRenderer",
          "mesh": "cube",
          "meshSize": [1.5, 1.5, 1.5],
          "color": "Red"
        },
        {
          "type": "BoxCollider",
          "size": [1.5, 1.5, 1.5]
        },
        {
          "type": "Rigidbody",
          "mass": 1,
          "bounciness": 0.6,
          "useGravity": true
        },
        {
          "type": "Script",
          "name": "Rotator",
          "props": { "speed": 90 }
        }
      ]
    }
  ]
}
```

### Built-in Component Types

| Type | Fields |
|------|--------|
| ModelRenderer | `mesh` (cube/plane/sphere) + `meshSize`, or `model` (file path), `color`, `metallic`, `roughness`, `emissive` |
| BoxCollider | `size` [3], `offset` [3] |
| SphereCollider | `radius` |
| Rigidbody | `mass`, `bounciness`, `friction`, `useGravity`, `isKinematic` |
| DirectionalLight | `direction` [3], `intensity` |
| Script | `name` (registry key), `props` (key-value) |

### Material Properties

ModelRenderer supports PBR-style material properties:

| Property | Range | Description |
|----------|-------|-------------|
| `metallic` | 0.0 - 1.0 | 0 = dielectric (plastic, wood), 1 = metal (gold, steel) |
| `roughness` | 0.0 - 1.0 | 0 = mirror-smooth, 1 = completely rough |
| `emissive` | 0.0+ | Glow intensity, adds to final color |

Example:
```json
{
  "type": "ModelRenderer",
  "model": "assets/models/helmet.gltf",
  "metallic": 0.9,
  "roughness": 0.3,
  "emissive": 0.0
}
```

## Script Registry

Scripts are custom behaviors that live in `internal/components/scripts/`, one file per script. Each script registers itself via `init()` with a factory (creates the component from props) and a serializer (converts it back for saving).

### Creating a New Script

Use the scaffolding tool:

```bash
# Build the utils (once)
cd utilities && cargo build --release && cp target/release/mirgo-utils ../

# Generate a new script
./mirgo-utils newscript MyScript
```

This generates `internal/components/scripts/my_script.go` with the struct, factory, serializer, and registration all wired up. Edit the struct fields and factory to match your needs, then reference it in a scene file:

```json
{
  "type": "Script",
  "name": "MyScript",
  "props": { "speed": 1.0 }
}
```

### Script Anatomy

```go
package scripts

import "test3d/internal/engine"

type Rotator struct {
    engine.BaseComponent
    Speed float32
}

func (r *Rotator) Update(deltaTime float32) {
    g := r.GetGameObject()
    g.Transform.Rotation.Y += r.Speed * deltaTime
}

func init() {
    engine.RegisterScript("Rotator", rotatorFactory, rotatorSerializer)
}

func rotatorFactory(props map[string]any) engine.Component {
    speed := float32(90)
    if v, ok := props["speed"].(float64); ok {
        speed = float32(v)
    }
    return &Rotator{Speed: speed}
}

func rotatorSerializer(c engine.Component) map[string]any {
    r, ok := c.(*Rotator)
    if !ok {
        return nil
    }
    return map[string]any{"speed": r.Speed}
}
```

### Accessing the World from Scripts

Any component can access world operations through the GameObject chain:

```go
func (s *MyScript) Update(deltaTime float32) {
    g := s.GetGameObject()

    // Access other components on this object
    rb := engine.GetComponent[*components.Rigidbody](g)

    // Access world operations (spawn, destroy, raycast)
    g.Scene.World.Destroy(someObject)
    g.Scene.World.SpawnObject(newObject)
    hit, ok := g.Scene.World.Raycast(origin, direction, 100.0)

    // Query scene
    enemies := g.Scene.FindByTag("enemy")
    light := g.Scene.FindByName("DirectionalLight")
}
```

## Architecture

```
Game
 └── World
      ├── Scene (GameObjects + WorldAccess)
      ├── PhysicsWorld (spatial hash, collision, gravity)
      └── Renderer (shadow map, lighting shader)

GameObject
 ├── Transform (Position, Rotation, Scale)
 ├── Tags []string
 └── Components []Component

Component Interface
 ├── Start()
 ├── Update(deltaTime)
 ├── SetGameObject(g)
 └── GetGameObject() → GameObject → Scene → World
```

Components access the world through `GetGameObject().Scene.World` - no singletons, no constructor injection. The `WorldAccess` interface in the engine package prevents circular dependencies between engine and world packages.

## Utilities (mirgo-utils)

A Rust CLI tool for asset processing and code generation. Build with:

```bash
cd utilities && cargo build --release && cp target/release/mirgo-utils ../
```

### Commands

#### `newscript <ScriptName>`

Generates a new script component with factory and serializer boilerplate.

```bash
./mirgo-utils newscript EnemyChaser
# Creates: internal/components/scripts/enemy_chaser.go
```

#### `flipnormals <path/to/model.gltf>`

Inverts all vertex normals in a GLTF model's binary buffer. Useful when imported models have inverted lighting (lit from below instead of above).

```bash
./mirgo-utils flipnormals assets/models/rubber_duck_toy_1k.gltf
# Flipped 2489 normal vectors in assets/models/rubber_duck_toy_1k.gltf/rubber_duck_toy.bin
```

If you pass a directory, it will find the `.gltf` file inside automatically. This command modifies the `.bin` file in place - run it again to flip back.

The editor also has a "Flip Normals" button in the inspector when a GLTF model is selected.

#### `build [name]`

Builds a standalone game executable. On macOS, creates a proper `.app` bundle that can be double-clicked or dragged to Applications.

```bash
./mirgo-utils build          # Creates build/game.app
./mirgo-utils build MyGame   # Creates build/MyGame.app
```

The build:
- Compiles with `-tags game` (excludes editor code)
- Copies assets into the bundle
- Creates Info.plist for macOS
