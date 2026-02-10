# Mirgo Engine API Reference

Complete reference for all core types, interfaces, and built-in components.

## Table of Contents

- [Core Types](#core-types)
  - [Component](#component)
  - [CollisionHandler](#collisionhandler)
  - [BaseComponent](#basecomponent)
  - [GameObject](#gameobject)
  - [Transform](#transform)
  - [Scene](#scene)
  - [WorldAccess](#worldaccess)
  - [RaycastResult](#raycastresult)
- [Built-in Components](#built-in-components)
  - [ModelRenderer](#modelrenderer)
  - [Camera](#camera)
  - [FPSController](#fpscontroller)
  - [DirectionalLight](#directionallight)
  - [Rigidbody](#rigidbody)
  - [BoxCollider](#boxcollider)
  - [SphereCollider](#spherecollider)
- [Generic Functions](#generic-functions)

---

## Core Types

### Component

The base interface that all components must implement.

```go
type Component interface {
    Start()                        // Called once when the object enters the scene
    Update(deltaTime float32)      // Called every frame
    SetGameObject(g *GameObject)   // Called by engine when component is added
    GetGameObject() *GameObject    // Returns the owning GameObject
}
```

**Lifecycle:**
1. `SetGameObject()` - Called immediately when added via `AddComponent()`
2. `Start()` - Called once on the first frame the object is active
3. `Update()` - Called every frame while the object is active

---

### CollisionHandler

Optional interface for components that want to receive collision callbacks.

```go
type CollisionHandler interface {
    OnCollisionEnter(other *GameObject)  // Called when collision starts
    OnCollisionExit(other *GameObject)   // Called when collision ends
}
```

**Usage:**
```go
type Collectible struct {
    engine.BaseComponent
    Points float32
}

func (c *Collectible) OnCollisionEnter(other *engine.GameObject) {
    if other.HasTag("Player") {
        fmt.Printf("+%.0f points!\n", c.Points)
        c.GetGameObject().Scene.World.Destroy(c.GetGameObject())
    }
}

func (c *Collectible) OnCollisionExit(other *engine.GameObject) {
    // Optional: called when objects separate
}
```

**Note:** You don't need to explicitly implement this interface. Just define the methods on your component and the physics engine will detect and call them automatically.

---

### BaseComponent

A default implementation of `Component` that you should embed in your own components.

```go
type BaseComponent struct {
    gameObject *GameObject
}

func (b *BaseComponent) Start()                        {}
func (b *BaseComponent) Update(deltaTime float32)     {}
func (b *BaseComponent) SetGameObject(g *GameObject)  { b.gameObject = g }
func (b *BaseComponent) GetGameObject() *GameObject   { return b.gameObject }
```

**Usage:**
```go
type MyComponent struct {
    engine.BaseComponent  // Embed this
    Speed float32
}

func (m *MyComponent) Update(deltaTime float32) {
    g := m.GetGameObject()  // Available via BaseComponent
    g.Transform.Rotation.Y += m.Speed * deltaTime
}
```

---

### GameObject

An entity in the scene with a transform, tags, and components.

```go
type GameObject struct {
    UID        uint64           // Unique identifier (persists across save/load)
    Name       string           // Human-readable name
    Tags       []string         // For querying (e.g., "enemy", "player")
    Transform  Transform        // Position, rotation, scale
    Active     bool             // If false, Update() is not called
    Scene      *Scene           // Back-reference to containing scene
    Parent     *GameObject      // Parent in hierarchy (nil if root)
    Children   []*GameObject    // Child objects
}
```

**Constructor:**
```go
func NewGameObject(name string) *GameObject
func NewGameObjectWithUID(name string, uid uint64) *GameObject
```

**Methods:**

| Method | Description |
|--------|-------------|
| `AddComponent(c Component)` | Adds a component, calls `SetGameObject()` |
| `RemoveComponent(c Component) bool` | Removes a component by pointer |
| `RemoveComponentByIndex(i int) bool` | Removes component at index |
| `Components() []Component` | Returns all components |
| `HasTag(tag string) bool` | Checks if object has a tag |
| `AddChild(child *GameObject)` | Adds a child object |
| `RemoveChild(child *GameObject)` | Removes a child object |
| `WorldPosition() rl.Vector3` | Position in world space (accounts for parent) |
| `WorldRotation() rl.Vector3` | Rotation in world space (accounts for parent) |
| `WorldScale() rl.Vector3` | Scale in world space (accounts for parent) |
| `Start()` | Calls `Start()` on all components (once) |
| `Update(deltaTime float32)` | Calls `Update()` on all components |

**Example:**
```go
cube := engine.NewGameObject("MyCube")
cube.Transform.Position = rl.Vector3{X: 0, Y: 5, Z: 0}
cube.Tags = []string{"enemy", "destructible"}
cube.AddComponent(components.NewRigidbody())
scene.AddGameObject(cube)
```

---

### Transform

Position, rotation, and scale of a GameObject.

```go
type Transform struct {
    Position rl.Vector3  // World position (X, Y, Z)
    Rotation rl.Vector3  // Euler angles in degrees (pitch, yaw, roll)
    Scale    rl.Vector3  // Scale factors (1.0 = normal size)
}
```

**Rotation order:** X (pitch) → Y (yaw) → Z (roll)

**Default values:**
- Position: `{0, 0, 0}`
- Rotation: `{0, 0, 0}`
- Scale: `{1, 1, 1}`

---

### Scene

A collection of GameObjects with query methods.

```go
type Scene struct {
    Name        string
    World       WorldAccess       // Interface to world operations
    GameObjects []*GameObject
}
```

**Constructor:**
```go
func NewScene(name string) *Scene
```

**Methods:**

| Method | Description |
|--------|-------------|
| `AddGameObject(g *GameObject)` | Adds object to scene, sets `g.Scene` |
| `RemoveGameObject(g *GameObject)` | Removes object and its children |
| `FindByName(name string) *GameObject` | Find first object with name |
| `FindByUID(uid uint64) *GameObject` | Find object by unique ID |
| `FindByTag(tag string) []*GameObject` | Find all objects with tag |
| `Start()` | Calls `Start()` on all objects |
| `Update(deltaTime float32)` | Calls `Update()` on all objects |

**Example:**
```go
// Find the player
player := scene.FindByName("Player")

// Find all enemies
enemies := scene.FindByTag("enemy")
for _, enemy := range enemies {
    // Do something with each enemy
}
```

---

### WorldAccess

Interface for world-level operations. Access via `gameObject.Scene.World`.

```go
type WorldAccess interface {
    GetCollidableObjects() []*GameObject
    SpawnObject(g *GameObject)
    Destroy(g *GameObject)
    Raycast(origin, direction rl.Vector3, maxDistance float32) (RaycastResult, bool)
    GetShader() rl.Shader
}
```

**Methods:**

| Method | Description |
|--------|-------------|
| `SpawnObject(g)` | Adds object to scene and physics world |
| `Destroy(g)` | Removes object from scene and physics |
| `Raycast(origin, dir, maxDist)` | Casts ray, returns hit info and success |
| `GetCollidableObjects()` | Returns all objects with colliders |
| `GetShader()` | Returns the main lighting shader |

**Example:**
```go
func (s *MyScript) Update(deltaTime float32) {
    g := s.GetGameObject()
    world := g.Scene.World

    // Raycast from camera
    origin := g.WorldPosition()
    direction := rl.Vector3{X: 0, Y: 0, Z: 1}

    if hit, ok := world.Raycast(origin, direction, 100.0); ok {
        fmt.Printf("Hit %s at distance %.2f\n", hit.GameObject.Name, hit.Distance)
    }

    // Destroy an object
    world.Destroy(someObject)

    // Spawn a new object
    bullet := engine.NewGameObject("Bullet")
    bullet.AddComponent(components.NewRigidbody())
    world.SpawnObject(bullet)
}
```

---

### RaycastResult

Information about a raycast hit.

```go
type RaycastResult struct {
    GameObject *GameObject  // Object that was hit
    Point      rl.Vector3   // World position of hit
    Normal     rl.Vector3   // Surface normal at hit point
    Distance   float32      // Distance from ray origin
}
```

---

## Built-in Components

### ModelRenderer

Renders a 3D model or generated mesh with PBR materials.

```go
type ModelRenderer struct {
    engine.BaseComponent
    Model     rl.Model      // The raylib model
    Color     rl.Color      // Tint color (for generated meshes)
    FilePath  string        // Path for file-loaded models
    MeshType  string        // "cube", "plane", "sphere" for generated
    MeshSize  []float32     // Size parameters for generated mesh
    Metallic  float32       // 0.0 (plastic) to 1.0 (metal)
    Roughness float32       // 0.0 (mirror) to 1.0 (matte)
    Emissive  float32       // Glow intensity (0.0+)
}
```

**Constructors:**
```go
func NewModelRenderer(model rl.Model, color rl.Color) *ModelRenderer
func NewModelRendererFromFile(path string, color rl.Color) *ModelRenderer
```

**Methods:**

| Method | Description |
|--------|-------------|
| `SetShader(shader rl.Shader)` | Sets shader for all materials |
| `Draw()` | Renders the model (called by renderer) |
| `Unload()` | Frees GPU resources |

**JSON Properties:**

| Property | Type | Description |
|----------|------|-------------|
| `mesh` | string | "cube", "plane", or "sphere" |
| `meshSize` | [3]float | Size for generated meshes |
| `model` | string | File path for GLTF models |
| `color` | string | Color name (Red, Blue, Gray, etc.) |
| `metallic` | float | 0.0-1.0 |
| `roughness` | float | 0.0-1.0 |
| `emissive` | float | 0.0+ |

**Example (JSON):**
```json
{
  "type": "ModelRenderer",
  "mesh": "sphere",
  "meshSize": [0.5, 32, 32],
  "color": "Gold",
  "metallic": 0.9,
  "roughness": 0.1
}
```

---

### Camera

Perspective camera for rendering the scene.

```go
type Camera struct {
    engine.BaseComponent
    FOV        float32              // Field of view in degrees
    Near       float32              // Near clipping plane
    Far        float32              // Far clipping plane
    Projection rl.CameraProjection  // Perspective or Orthographic
    IsMain     bool                 // If true, this is the active camera
}
```

**Constructor:**
```go
func NewCamera() *Camera  // FOV=45, Near=0.1, Far=1000
```

**Methods:**

| Method | Description |
|--------|-------------|
| `GetRaylibCamera() rl.Camera3D` | Returns raylib camera for rendering |

**JSON Properties:**

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `fov` | float | 45 | Field of view in degrees |
| `near` | float | 0.1 | Near clip distance |
| `far` | float | 1000 | Far clip distance |
| `isMain` | bool | false | Active game camera |

---

### FPSController

First-person movement and mouse look.

```go
type FPSController struct {
    engine.BaseComponent
    Yaw          float32     // Horizontal look angle (degrees)
    Pitch        float32     // Vertical look angle (degrees)
    MoveSpeed    float32     // Units per second
    LookSpeed    float32     // Mouse sensitivity
    Velocity     rl.Vector3  // Current velocity
    Gravity      float32     // Gravity strength
    JumpStrength float32     // Jump impulse
    Grounded     bool        // On ground?
    EyeHeight    float32     // Camera height offset
}
```

**Constructor:**
```go
func NewFPSController() *FPSController
// Defaults: MoveSpeed=8, LookSpeed=0.1, Gravity=20, JumpStrength=8, EyeHeight=1.6
```

**Methods:**

| Method | Description |
|--------|-------------|
| `GetLookDirection() rl.Vector3` | Returns normalized look vector |

**Controls (hardcoded):**
- WASD: Move
- Mouse: Look
- Space: Jump

**JSON Properties:**

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `moveSpeed` | float | 8.0 | Movement speed |
| `lookSpeed` | float | 0.1 | Mouse sensitivity |
| `jumpStrength` | float | 8.0 | Jump force |
| `gravity` | float | 20.0 | Gravity acceleration |
| `eyeHeight` | float | 1.6 | Camera Y offset |

---

### DirectionalLight

Directional light source for shadows and illumination.

```go
type DirectionalLight struct {
    engine.BaseComponent
    Direction      rl.Vector3  // Light direction (normalized)
    Color          rl.Color    // Light color
    Intensity      float32     // Brightness multiplier
    AmbientColor   rl.Color    // Ambient fill light
    ShadowDistance float32     // Shadow map range
}
```

**Constructor:**
```go
func NewDirectionalLight() *DirectionalLight
// Defaults: Direction={0.35, -1, -0.35}, Intensity=1.0, ShadowDistance=50
```

**Methods:**

| Method | Description |
|--------|-------------|
| `GetLightCamera(orthoSize float32) rl.Camera3D` | Camera for shadow map rendering |
| `MoveLightDir(dx, dy, dz float32)` | Adjusts and renormalizes direction |
| `GetColorFloat() []float32` | Color * intensity as RGBA floats |
| `GetAmbientFloat() []float32` | Ambient color as RGBA floats |

**JSON Properties:**

| Property | Type | Description |
|----------|------|-------------|
| `direction` | [3]float | Light direction vector |
| `intensity` | float | Brightness (1.0 = normal) |

---

### Rigidbody

Physics body with mass, velocity, and collision response.

```go
type Rigidbody struct {
    engine.BaseComponent
    Velocity        rl.Vector3  // Linear velocity (units/sec)
    AngularVelocity rl.Vector3  // Rotation speed (degrees/sec per axis)
    Mass            float32     // Object mass
    Bounciness      float32     // 0 = no bounce, 1 = perfect bounce
    Friction        float32     // 0 = ice, 1 = stops immediately
    AngularDamping  float32     // Rotation slowdown per frame
    UseGravity      bool        // Apply gravity?
    IsKinematic     bool        // Moves but isn't pushed by physics
}
```

**Constructor:**
```go
func NewRigidbody() *Rigidbody
// Defaults: Mass=1, Bounciness=0.5, Friction=0.1, AngularDamping=0.98, UseGravity=true
```

**JSON Properties:**

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `mass` | float | 1.0 | Object mass |
| `bounciness` | float | 0.5 | Restitution coefficient |
| `friction` | float | 0.1 | Friction coefficient |
| `useGravity` | bool | true | Enable gravity |
| `isKinematic` | bool | false | Kinematic mode |

**Kinematic vs Dynamic:**
- **Dynamic** (default): Affected by forces, collisions, gravity
- **Kinematic**: Moves via code, pushes other objects, not affected by forces

Use kinematic for players and moving platforms.

---

### BoxCollider

Axis-aligned box collision shape.

```go
type BoxCollider struct {
    engine.BaseComponent
    Size   rl.Vector3  // Box dimensions
    Offset rl.Vector3  // Offset from object center
}
```

**Constructor:**
```go
func NewBoxCollider(size rl.Vector3) *BoxCollider
```

**Methods:**

| Method | Description |
|--------|-------------|
| `GetCenter() rl.Vector3` | World-space center (position + scaled offset) |
| `GetWorldSize() rl.Vector3` | Size scaled by object transform |

**JSON Properties:**

| Property | Type | Description |
|----------|------|-------------|
| `size` | [3]float | Box dimensions [x, y, z] |
| `offset` | [3]float | Offset from center |

---

### SphereCollider

Sphere collision shape.

```go
type SphereCollider struct {
    engine.BaseComponent
    Radius float32     // Sphere radius
    Offset rl.Vector3  // Offset from object center
}
```

**Constructor:**
```go
func NewSphereCollider(radius float32) *SphereCollider
```

**Methods:**

| Method | Description |
|--------|-------------|
| `GetCenter() rl.Vector3` | World-space center |

**JSON Properties:**

| Property | Type | Description |
|----------|------|-------------|
| `radius` | float | Sphere radius |

---

## Generic Functions

### GetComponent

Type-safe component lookup using Go generics.

```go
func GetComponent[T Component](g *GameObject) T
```

**Usage:**
```go
// Get specific component types
rb := engine.GetComponent[*components.Rigidbody](gameObject)
if rb != nil {
    rb.Velocity.Y = 10
}

cam := engine.GetComponent[*components.Camera](gameObject)
light := engine.GetComponent[*components.DirectionalLight](gameObject)
fps := engine.GetComponent[*components.FPSController](gameObject)
```

Returns `nil` if the component type is not found on the object.

---

## Color Names

Available color names for JSON scene files:

| Name | RGB |
|------|-----|
| White | 255, 255, 255 |
| Black | 0, 0, 0 |
| Gray | 128, 128, 128 |
| Red | 255, 0, 0 |
| Green | 0, 255, 0 |
| Blue | 0, 0, 255 |
| Yellow | 255, 255, 0 |
| Orange | 255, 165, 0 |
| Purple | 128, 0, 128 |
| Cyan | 0, 255, 255 |
| Magenta | 255, 0, 255 |
| Gold | 255, 215, 0 |
| Brown | 139, 69, 19 |
| Pink | 255, 192, 203 |

---

## See Also

- [Getting Started Guide](getting-started.md)
- [Scripting Guide](scripting.md)
