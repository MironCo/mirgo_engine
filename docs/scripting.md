# Mirgo Engine Scripting Guide

This guide covers everything you need to write custom game logic in Mirgo Engine.

## Table of Contents

- [Overview](#overview)
- [Creating Your First Script](#creating-your-first-script)
- [Script Structure](#script-structure)
- [Accessing GameObjects and Components](#accessing-gameobjects-and-components)
- [World Operations](#world-operations)
- [Input Handling](#input-handling)
- [Common Patterns](#common-patterns)
- [Complete Examples](#complete-examples)

---

## Overview

Scripts are custom components that implement game logic. With Mirgo Engine's automatic code generation:

1. **You write**: Clean source files in `assets/scripts/` with just your game logic
2. **Engine generates**: Factory and serializer functions automatically
3. **Result**: Full implementation in `internal/scripts/` (git-ignored)

This Unity-like workflow eliminates boilerplate while maintaining Go's type safety

---

## Creating Your First Script

### The Unity-like Way

Mirgo Engine uses automatic code generation - you write clean scripts without boilerplate.

Create a new file in `assets/scripts/`:

```go
package scripts

import "test3d/internal/engine"

type EnemyAI struct {
    engine.BaseComponent
    Speed float32
}

func (e *EnemyAI) Update(deltaTime float32) {
    g := e.GetGameObject()
    if g == nil {
        return
    }
    // Your logic here
}
```

That's it! When you run `make build`, `make run`, or press Cmd+R in the editor, the system automatically:

1. Parses your struct using Go's AST parser
2. Generates factory and serializer functions
3. Registers the script with the engine
4. Outputs to `internal/scripts/enemy_ai.go` (git-ignored)

### Using the Script Generator (Optional)

For convenience, you can use `mirgo-utils` to create the initial file:

```bash
# Build utils (once)
cd utilities && cargo build --release && cp target/release/mirgo-utils ../

# Generate a new script template
./mirgo-utils newscript EnemyAI
```

This creates `assets/scripts/enemy_ai.go` with basic structure

### Using in a Scene

Add the script to any object in your scene JSON:

```json
{
  "name": "Enemy",
  "position": [5, 0, 5],
  "components": [
    {
      "type": "ModelRenderer",
      "mesh": "cube",
      "meshSize": [1, 1, 1],
      "color": "Red"
    },
    {
      "type": "Script",
      "name": "EnemyAI",
      "props": {
        "speed": 3.5
      }
    }
  ]
}
```

---

## Script Structure

### What You Write (in assets/scripts/)

Your source scripts need only two things:

#### 1. The Struct (with BaseComponent)

```go
type MyScript struct {
    engine.BaseComponent  // REQUIRED - provides GetGameObject()

    // Your custom fields
    Speed     float32
    Target    string
    Health    int
    isActive  bool  // lowercase = private, won't be serialized
}
```

**Field Serialization**:
- Exported fields (capitalized) are automatically serialized
- Converted to snake_case JSON names (`Speed` → `"speed"`, `MaxHealth` → `"max_health"`)
- Private fields (lowercase) are ignored

#### 2. The Update Method

```go
func (m *MyScript) Update(deltaTime float32) {
    g := m.GetGameObject()
    if g == nil {
        return
    }

    // Per-frame logic here
    // deltaTime is seconds since last frame (typically ~0.016 for 60fps)
}
```

### What Gets Generated (in internal/scripts/)

The system automatically generates:

#### 3. The Factory Function

Creates the component from JSON properties with proper type conversions:

```go
func myScriptFactory(props map[string]any) engine.Component {
    script := &MyScript{}
    if v, ok := props["speed"].(float64); ok {
        script.Speed = float32(v)  // Converts JSON float64 to Go float32
    }
    if v, ok := props["target"].(string); ok {
        script.Target = v
    }
    if v, ok := props["health"].(float64); ok {
        script.Health = int(v)  // Converts JSON float64 to Go int
    }
    return script
}
```

#### 4. The Serializer Function

Converts component back to JSON for saving:

```go
func myScriptSerializer(c engine.Component) map[string]any {
    m, ok := c.(*MyScript)
    if !ok {
        return nil
    }
    return map[string]any{
        "speed":  m.Speed,
        "target": m.Target,
        "health": m.Health,
    }
}
```

#### 5. Registration (in init)

```go
func init() {
    engine.RegisterScript("MyScript", myScriptFactory, myScriptSerializer)
}
```

### How Generation Works

- **Trigger**: Runs automatically on `make build`, `make run`, or Cmd+R in editor
- **Parser**: Uses Go's `go/ast` and `go/parser` packages
- **Caching**: SHA256 hashes in `.hash` files prevent regenerating unchanged scripts
- **Output**: Source + generated boilerplate in `internal/scripts/` (git-ignored)
- **Type Conversions**: Handles float32, float64, int, int32, int64, bool, string automatically

### Optional: Start Method

For one-time initialization:

```go
func (m *MyScript) Start() {
    // Called once when object enters scene
    // Good for caching references, initial setup
    m.cachedTarget = m.GetGameObject().Scene.FindByName(m.TargetName)
}
```

---

## Accessing GameObjects and Components

### Getting Your Own GameObject

```go
func (m *MyScript) Update(deltaTime float32) {
    g := m.GetGameObject()

    // Access transform
    g.Transform.Position.X += m.Speed * deltaTime
    g.Transform.Rotation.Y += 90 * deltaTime

    // Check tags
    if g.HasTag("enemy") {
        // ...
    }
}
```

### Getting Components on the Same Object

Use the generic `GetComponent` function:

```go
import "test3d/internal/components"

func (m *MyScript) Update(deltaTime float32) {
    g := m.GetGameObject()

    // Get rigidbody
    rb := engine.GetComponent[*components.Rigidbody](g)
    if rb != nil {
        rb.Velocity.Y = 10  // Apply upward force
    }

    // Get other components
    fps := engine.GetComponent[*components.FPSController](g)
    cam := engine.GetComponent[*components.Camera](g)
    box := engine.GetComponent[*components.BoxCollider](g)
}
```

### Finding Other Objects

```go
func (m *MyScript) Update(deltaTime float32) {
    scene := m.GetGameObject().Scene

    // Find by name (returns first match or nil)
    player := scene.FindByName("Player")

    // Find by UID (for persistent references)
    obj := scene.FindByUID(12345)

    // Find by tag (returns slice)
    enemies := scene.FindByTag("enemy")
    for _, enemy := range enemies {
        // Do something with each enemy
    }
}
```

### Iterating All Objects

```go
func (m *MyScript) Update(deltaTime float32) {
    for _, obj := range m.GetGameObject().Scene.GameObjects {
        if obj.HasTag("collectible") {
            // Check distance, collect, etc.
        }
    }
}
```

---

## World Operations

Access world operations through `GetGameObject().Scene.World`:

### Spawning Objects

```go
func (m *MyScript) SpawnBullet() {
    g := m.GetGameObject()
    world := g.Scene.World

    // Create the object
    bullet := engine.NewGameObject("Bullet")
    bullet.Transform.Position = g.WorldPosition()

    // Add components
    mesh := rl.GenMeshSphere(0.2, 16, 16)
    model := rl.LoadModelFromMesh(mesh)
    renderer := components.NewModelRenderer(model, rl.Yellow)
    renderer.SetShader(world.GetShader())  // Apply lighting shader
    bullet.AddComponent(renderer)

    bullet.AddComponent(components.NewSphereCollider(0.2))

    rb := components.NewRigidbody()
    rb.Velocity = rl.Vector3{X: 0, Y: 0, Z: 20}  // Forward velocity
    bullet.AddComponent(rb)

    // Initialize and spawn
    bullet.Start()
    world.SpawnObject(bullet)
}
```

### Destroying Objects

```go
func (m *MyScript) Update(deltaTime float32) {
    world := m.GetGameObject().Scene.World

    // Destroy another object
    enemy := m.GetGameObject().Scene.FindByName("Enemy")
    if enemy != nil {
        world.Destroy(enemy)
    }

    // Destroy self
    world.Destroy(m.GetGameObject())
}
```

### Raycasting

```go
func (m *MyScript) Update(deltaTime float32) {
    g := m.GetGameObject()
    world := g.Scene.World

    origin := g.WorldPosition()
    direction := rl.Vector3{X: 0, Y: 0, Z: 1}  // Forward
    maxDistance := float32(100.0)

    if hit, ok := world.Raycast(origin, direction, maxDistance); ok {
        // hit.GameObject - what was hit
        // hit.Point      - world position of hit
        // hit.Normal     - surface normal
        // hit.Distance   - distance from origin

        fmt.Printf("Hit %s at %.2f units\n", hit.GameObject.Name, hit.Distance)
    }
}
```

---

## Input Handling

Use raylib input functions directly:

### Keyboard

```go
import rl "github.com/gen2brain/raylib-go/raylib"

func (m *MyScript) Update(deltaTime float32) {
    // Check if key is currently held
    if rl.IsKeyDown(rl.KeyE) {
        m.Interact()
    }

    // Check if key was just pressed this frame
    if rl.IsKeyPressed(rl.KeyR) {
        m.Reload()
    }

    // Check if key was just released
    if rl.IsKeyReleased(rl.KeyShift) {
        m.StopSprinting()
    }
}
```

### Mouse

```go
func (m *MyScript) Update(deltaTime float32) {
    // Mouse buttons
    if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
        m.Shoot()
    }
    if rl.IsMouseButtonDown(rl.MouseRightButton) {
        m.Aim()
    }

    // Mouse position
    pos := rl.GetMousePosition()

    // Mouse movement (for look controls)
    delta := rl.GetMouseDelta()
}
```

### Common Key Codes

| Key | Code |
|-----|------|
| WASD | `rl.KeyW`, `rl.KeyA`, `rl.KeyS`, `rl.KeyD` |
| Space | `rl.KeySpace` |
| Shift | `rl.KeyLeftShift` |
| Ctrl | `rl.KeyLeftControl` |
| E | `rl.KeyE` |
| R | `rl.KeyR` |
| Escape | `rl.KeyEscape` |
| Numbers | `rl.KeyOne` through `rl.KeyZero` |

---

## Common Patterns

### Cooldown Timer

```go
type Shooter struct {
    engine.BaseComponent
    Cooldown     float64
    lastShotTime float64
}

func (s *Shooter) Update(deltaTime float32) {
    currentTime := rl.GetTime()

    if rl.IsMouseButtonDown(rl.MouseLeftButton) {
        if currentTime - s.lastShotTime >= s.Cooldown {
            s.Shoot()
            s.lastShotTime = currentTime
        }
    }
}
```

### Move Toward Target

```go
type Chaser struct {
    engine.BaseComponent
    TargetName string
    Speed      float32
}

func (c *Chaser) Update(deltaTime float32) {
    g := c.GetGameObject()
    target := g.Scene.FindByName(c.TargetName)
    if target == nil {
        return
    }

    // Calculate direction
    myPos := g.WorldPosition()
    targetPos := target.WorldPosition()
    direction := rl.Vector3Subtract(targetPos, myPos)
    direction = rl.Vector3Normalize(direction)

    // Move toward target
    movement := rl.Vector3Scale(direction, c.Speed * deltaTime)
    g.Transform.Position = rl.Vector3Add(g.Transform.Position, movement)
}
```

### Distance Check

```go
func (m *MyScript) Update(deltaTime float32) {
    g := m.GetGameObject()
    player := g.Scene.FindByName("Player")
    if player == nil {
        return
    }

    myPos := g.WorldPosition()
    playerPos := player.WorldPosition()
    distance := rl.Vector3Distance(myPos, playerPos)

    if distance < 5.0 {
        m.OnPlayerNear()
    }
}
```

### Look At Target

```go
import "math"

func (m *MyScript) LookAt(target rl.Vector3) {
    g := m.GetGameObject()
    pos := g.WorldPosition()

    direction := rl.Vector3Subtract(target, pos)

    // Calculate yaw (Y rotation)
    yaw := float32(math.Atan2(float64(direction.X), float64(direction.Z)))
    g.Transform.Rotation.Y = yaw * rl.Rad2deg
}
```

### State Machine

```go
type EnemyState int

const (
    StateIdle EnemyState = iota
    StateChasing
    StateAttacking
)

type Enemy struct {
    engine.BaseComponent
    State       EnemyState
    DetectRange float32
    AttackRange float32
}

func (e *Enemy) Update(deltaTime float32) {
    player := e.GetGameObject().Scene.FindByName("Player")
    if player == nil {
        return
    }

    distance := rl.Vector3Distance(
        e.GetGameObject().WorldPosition(),
        player.WorldPosition(),
    )

    switch e.State {
    case StateIdle:
        if distance < e.DetectRange {
            e.State = StateChasing
        }
    case StateChasing:
        e.MoveToward(player)
        if distance < e.AttackRange {
            e.State = StateAttacking
        } else if distance > e.DetectRange * 1.5 {
            e.State = StateIdle
        }
    case StateAttacking:
        e.Attack()
        if distance > e.AttackRange {
            e.State = StateChasing
        }
    }
}
```

---

## Complete Examples

### Rotator Script

Spins an object around the Y axis:

```go
package scripts

import "test3d/internal/engine"

type Rotator struct {
    engine.BaseComponent
    Speed float32  // Degrees per second
}

func (r *Rotator) Update(deltaTime float32) {
    g := r.GetGameObject()
    if g == nil {
        return
    }

    g.Transform.Rotation.Y += r.Speed * deltaTime

    // Keep rotation in 0-360 range
    if g.Transform.Rotation.Y > 360 {
        g.Transform.Rotation.Y -= 360
    }
}

func init() {
    engine.RegisterScript("Rotator", rotatorFactory, rotatorSerializer)
}

func rotatorFactory(props map[string]any) engine.Component {
    speed := float32(90)  // Default: 90 deg/sec
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

**Usage:**
```json
{
  "type": "Script",
  "name": "Rotator",
  "props": { "speed": 45 }
}
```

### Shooter Script

Shoots projectiles on left-click, deletes targets on right-click:

```go
package scripts

import (
    "fmt"
    "test3d/internal/components"
    "test3d/internal/engine"
    rl "github.com/gen2brain/raylib-go/raylib"
)

type Shooter struct {
    engine.BaseComponent
    Cooldown     float64
    lastShotTime float64
    shotCounter  int
}

func (s *Shooter) Update(deltaTime float32) {
    // Shoot on left click (with cooldown)
    if rl.IsMouseButtonDown(rl.MouseLeftButton) {
        if rl.GetTime() - s.lastShotTime >= s.Cooldown {
            s.Shoot()
            s.lastShotTime = rl.GetTime()
        }
    }

    // Delete target on right click
    if rl.IsMouseButtonPressed(rl.MouseRightButton) {
        s.DeleteTarget()
    }
}

func (s *Shooter) Shoot() {
    g := s.GetGameObject()
    fps := engine.GetComponent[*components.FPSController](g)
    if fps == nil {
        return
    }

    s.shotCounter++

    // Calculate spawn position (in front of camera)
    eyePos := g.Transform.Position
    eyePos.Y += fps.EyeHeight
    lookDir := fps.GetLookDirection()
    spawnPos := rl.Vector3Add(eyePos, rl.Vector3Scale(lookDir, 3))

    // Create projectile
    radius := float32(0.5)
    sphere := engine.NewGameObject(fmt.Sprintf("Shot_%d", s.shotCounter))
    sphere.Transform.Position = spawnPos

    // Add renderer
    mesh := rl.GenMeshSphere(radius, 16, 16)
    model := rl.LoadModelFromMesh(mesh)
    renderer := components.NewModelRenderer(model, rl.Orange)
    renderer.SetShader(g.Scene.World.GetShader())
    sphere.AddComponent(renderer)

    // Add collider
    sphere.AddComponent(components.NewSphereCollider(radius))

    // Add rigidbody with initial velocity
    rb := components.NewRigidbody()
    rb.Bounciness = 0.6
    rb.Velocity = rl.Vector3Scale(lookDir, 30)
    sphere.AddComponent(rb)

    // Spawn into world
    sphere.Start()
    g.Scene.World.SpawnObject(sphere)
}

func (s *Shooter) DeleteTarget() {
    g := s.GetGameObject()
    fps := engine.GetComponent[*components.FPSController](g)
    if fps == nil {
        return
    }

    // Raycast from camera
    origin := g.Transform.Position
    origin.Y += fps.EyeHeight
    direction := fps.GetLookDirection()

    hit, ok := g.Scene.World.Raycast(origin, direction, 100.0)
    if !ok {
        return
    }

    // Don't delete floor or player
    if hit.GameObject.Name == "Floor" || hit.GameObject.Name == "Player" {
        return
    }

    g.Scene.World.Destroy(hit.GameObject)
}

func init() {
    engine.RegisterScript("Shooter", shooterFactory, shooterSerializer)
}

func shooterFactory(props map[string]any) engine.Component {
    cooldown := 0.15
    if v, ok := props["cooldown"].(float64); ok {
        cooldown = v
    }
    return &Shooter{Cooldown: cooldown}
}

func shooterSerializer(c engine.Component) map[string]any {
    s, ok := c.(*Shooter)
    if !ok {
        return nil
    }
    return map[string]any{"cooldown": s.Cooldown}
}
```

**Usage:**
```json
{
  "type": "Script",
  "name": "Shooter",
  "props": { "cooldown": 0.1 }
}
```

### Collectible Script

Destroys self when player gets close, could trigger score increase:

```go
package scripts

import (
    "test3d/internal/engine"
    rl "github.com/gen2brain/raylib-go/raylib"
)

type Collectible struct {
    engine.BaseComponent
    CollectRadius float32
    RotateSpeed   float32
}

func (c *Collectible) Update(deltaTime float32) {
    g := c.GetGameObject()

    // Rotate for visual effect
    g.Transform.Rotation.Y += c.RotateSpeed * deltaTime

    // Find player
    player := g.Scene.FindByName("Player")
    if player == nil {
        return
    }

    // Check distance
    distance := rl.Vector3Distance(g.WorldPosition(), player.WorldPosition())
    if distance < c.CollectRadius {
        c.Collect()
    }
}

func (c *Collectible) Collect() {
    // TODO: Add score, play sound, etc.
    g := c.GetGameObject()
    g.Scene.World.Destroy(g)
}

func init() {
    engine.RegisterScript("Collectible", collectibleFactory, collectibleSerializer)
}

func collectibleFactory(props map[string]any) engine.Component {
    radius := float32(1.5)
    rotateSpeed := float32(90)

    if v, ok := props["collectRadius"].(float64); ok {
        radius = float32(v)
    }
    if v, ok := props["rotateSpeed"].(float64); ok {
        rotateSpeed = float32(v)
    }

    return &Collectible{
        CollectRadius: radius,
        RotateSpeed:   rotateSpeed,
    }
}

func collectibleSerializer(c engine.Component) map[string]any {
    col, ok := c.(*Collectible)
    if !ok {
        return nil
    }
    return map[string]any{
        "collectRadius": col.CollectRadius,
        "rotateSpeed":   col.RotateSpeed,
    }
}
```

---

## Tips and Best Practices

1. **Always null-check `GetGameObject()`** - It can be nil during teardown.

2. **Cache references in `Start()`** - Don't call `FindByName()` every frame.

3. **Use `deltaTime` for movement** - Ensures consistent speed regardless of framerate.

4. **Keep scripts focused** - One behavior per script. Compose by adding multiple scripts.

5. **Handle edge cases** - What if the target doesn't exist? What if the player is destroyed?

6. **Use tags for categories** - Instead of checking names, use tags like "enemy", "collectible".

7. **Private fields for state** - Use lowercase names for internal state that shouldn't serialize.

---

## See Also

- [API Reference](api-reference.md) - Complete type documentation
- [Getting Started](getting-started.md) - First steps with Mirgo Engine
