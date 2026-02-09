# Scene File Format

Scenes are JSON files in `assets/scenes/`. Objects have a transform and a list of components.

## Basic Scene Structure

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

## Object Properties

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | Object name (used for `FindByName()`) |
| `tags` | string[] | Tags for categorization (used for `FindByTag()`) |
| `position` | [x, y, z] | Local position relative to parent |
| `rotation` | [x, y, z] | Euler angles in degrees |
| `scale` | [x, y, z] | Local scale multiplier |
| `components` | Component[] | Array of component definitions |
| `children` | Object[] | Child objects (inherit parent transform) |

## Built-in Components

### ModelRenderer

Renders a 3D model or primitive mesh.

**Primitive mesh:**
```json
{
  "type": "ModelRenderer",
  "mesh": "cube",
  "meshSize": [1, 1, 1],
  "color": "Red",
  "metallic": 0.0,
  "roughness": 0.5,
  "emissive": 0.0
}
```

**GLTF model:**
```json
{
  "type": "ModelRenderer",
  "model": "assets/models/helmet.gltf",
  "metallic": 0.9,
  "roughness": 0.3
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mesh` | string | - | Primitive type: "cube", "sphere", "plane" |
| `meshSize` | [x, y, z] | [1, 1, 1] | Size of primitive mesh |
| `model` | string | - | Path to GLTF model file |
| `color` | string | "White" | Color name or hex (#FF0000) |
| `metallic` | float | 0.0 | 0.0 = dielectric, 1.0 = metal |
| `roughness` | float | 0.5 | 0.0 = smooth, 1.0 = rough |
| `emissive` | float | 0.0 | Glow intensity |

### BoxCollider

Box-shaped collision volume.

```json
{
  "type": "BoxCollider",
  "size": [1, 2, 1],
  "offset": [0, 1, 0]
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `size` | [x, y, z] | [1, 1, 1] | Dimensions of the box |
| `offset` | [x, y, z] | [0, 0, 0] | Local position offset |

### SphereCollider

Sphere-shaped collision volume.

```json
{
  "type": "SphereCollider",
  "radius": 0.5
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `radius` | float | 0.5 | Sphere radius |

### Rigidbody

Physics simulation component.

```json
{
  "type": "Rigidbody",
  "mass": 1.0,
  "bounciness": 0.5,
  "friction": 0.5,
  "useGravity": true,
  "isKinematic": false
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mass` | float | 1.0 | Object mass (affects collision response) |
| `bounciness` | float | 0.3 | 0.0 = no bounce, 1.0 = perfect bounce |
| `friction` | float | 0.5 | Surface friction coefficient |
| `useGravity` | bool | true | Apply gravity force |
| `isKinematic` | bool | false | Kinematic bodies don't respond to forces |

### DirectionalLight

Directional light source (sun/moon).

```json
{
  "type": "DirectionalLight",
  "direction": [0.5, -1, 0.3],
  "intensity": 1.0
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `direction` | [x, y, z] | [0, -1, 0] | Light direction vector (normalized) |
| `intensity` | float | 1.0 | Light brightness multiplier |

### Camera

Perspective camera.

```json
{
  "type": "Camera",
  "fov": 60,
  "isMain": true
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `fov` | float | 60 | Field of view in degrees |
| `isMain` | bool | false | Set as the main rendering camera |

### FPSController

First-person character controller.

```json
{
  "type": "FPSController",
  "moveSpeed": 5.0,
  "lookSpeed": 0.3,
  "jumpStrength": 8.0
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `moveSpeed` | float | 5.0 | Movement speed (units/second) |
| `lookSpeed` | float | 0.3 | Mouse sensitivity |
| `jumpStrength` | float | 8.0 | Jump impulse strength |

### Script

Custom script component.

```json
{
  "type": "Script",
  "name": "Rotator",
  "props": {
    "speed": 45.0,
    "enabled": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Script type name (registered with engine) |
| `props` | object | Key-value properties passed to script factory |

Properties in `props` are automatically parsed based on your script's struct fields. See the [Scripting Guide](scripting.md) for details.

## Object Hierarchies

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
      "name": "Pillar",
      "position": [-1, -1, -1],
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

Child positions are **relative to their parent**. When the Platform moves, the Pillar moves with it.

## Material Properties

ModelRenderer supports PBR-style materials:

### Metallic

- **Range**: 0.0 - 1.0
- **0.0**: Dielectric materials (plastic, wood, rubber)
- **1.0**: Metallic materials (gold, steel, chrome)

```json
{ "metallic": 0.9 }  // Shiny metal
{ "metallic": 0.0 }  // Matte plastic
```

### Roughness

- **Range**: 0.0 - 1.0
- **0.0**: Mirror-smooth, sharp reflections
- **1.0**: Completely rough, diffuse surface

```json
{ "roughness": 0.2 }  // Polished surface
{ "roughness": 0.8 }  // Rough surface
```

### Emissive

- **Range**: 0.0+
- **Effect**: Adds glow to the material
- **Use**: For light sources, neon signs, glowing objects

```json
{ "emissive": 2.0, "color": "Cyan" }  // Glowing cyan object
```

## Complete Examples

### Physics Object

```json
{
  "name": "Crate",
  "position": [0, 5, 0],
  "components": [
    {
      "type": "ModelRenderer",
      "mesh": "cube",
      "meshSize": [1, 1, 1],
      "color": "Brown",
      "roughness": 0.8
    },
    {
      "type": "BoxCollider",
      "size": [1, 1, 1]
    },
    {
      "type": "Rigidbody",
      "mass": 2.0,
      "bounciness": 0.3,
      "useGravity": true
    }
  ]
}
```

### Player Character

```json
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
    },
    {
      "type": "Script",
      "name": "Shooter",
      "props": { "cooldown": 0.15 }
    }
  ]
}
```

### GLTF Model with Physics

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
    },
    {
      "type": "Script",
      "name": "Rotator",
      "props": { "speed": 45 }
    }
  ]
}
```

## See Also

- [Getting Started](getting-started.md) - Create your first scene
- [Scripting Guide](scripting.md) - Writing custom script components
- [API Reference](api-reference.md) - Complete type documentation