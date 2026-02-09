# Editor Guide

The Mirgo Engine editor provides a Unity-like interface for building and testing 3D scenes.

## Toggling Editor Mode

Press **Cmd/Ctrl+P** to switch between:
- **Game Mode**: Play and test your game
- **Editor Mode**: Edit scene, move objects, tweak properties

## Editor Controls

| Action | Control |
|--------|---------|
| **Look Around** | Right Mouse + Drag |
| **Fly Camera** | Right Mouse + WASD |
| **Adjust Fly Speed** | Scroll Wheel (while holding Right Mouse) |
| **Select Object** | Left Click |
| **Move Object** | Drag Gizmo Arrow |
| **Save Scene** | Cmd/Ctrl+S |
| **Hot Reload** | Cmd/Ctrl+R (rebuilds + regenerates scripts) |
| **Build Game** | Cmd/Ctrl+B |
| **Toggle Game Mode** | Cmd/Ctrl+P |

## Editor Panels

### Hierarchy Panel

- Shows all objects in the scene
- Click to select an object
- Displays object names and hierarchy

### Inspector Panel

Shows properties of the selected object:

- **Transform**: Position, Rotation, Scale
- **Components**: Add, remove, or edit components
- **Tags**: Add/remove tags for categorization
- **Properties**: Edit component-specific values

### Asset Browser

- Drag-and-drop GLTF models into the scene
- Browse available assets
- "Flip Normals" button for GLTF models with inverted lighting

## Working with Objects

### Selecting Objects

1. Left-click on an object in the scene
2. Or click its name in the Hierarchy panel
3. Selected object highlights in the Inspector

### Moving Objects

1. Select an object
2. Drag the colored gizmo arrows:
   - **Red arrow**: Move along X axis
   - **Green arrow**: Move along Y axis
   - **Blue arrow**: Move along Z axis

### Transform Modes

The editor supports different transform modes (future feature):
- **Translate** (Move)
- **Rotate**
- **Scale**

Currently only translation is implemented.

### Adding Components

In the Inspector:
1. Select an object
2. Click "Add Component"
3. Choose component type
4. Configure properties

### Editing Properties

Click on property values in the Inspector to edit:
- **Numbers**: Click and type
- **Booleans**: Click checkbox
- **Colors**: Choose from palette
- **Vectors**: Edit X, Y, Z individually

## Scene Management

### Saving Scenes

1. Press **Cmd/Ctrl+S**
2. Scene saved to current JSON file
3. Confirmation message appears

All changes are saved:
- Object transforms
- Component properties
- Script properties
- Hierarchy structure

### Loading Scenes

Modify `cmd/test3d/main.go`:

```go
world.LoadScene("assets/scenes/your_scene.json")
```

Then rebuild with `make run`.

### Creating New Scenes

1. Copy an existing scene JSON file
2. Edit in text editor or start from scratch
3. Load it in the game

See [Scene Format](scene-format.md) for JSON structure.

## Hot Reload (Cmd+R)

Press **Cmd/Ctrl+R** to:

1. **Regenerate scripts** from `assets/scripts/`
2. **Rebuild** the engine
3. **Reload** the current scene

This is faster than exiting and restarting. Use it when:
- You modify a script in `assets/scripts/`
- You change Go code in the engine
- You want to test changes quickly

**Note**: Scene state is reset (objects return to their saved positions).

## Undo System

Press **Ctrl+Z** to undo recent transform changes.

Currently supports:
- Position changes
- Rotation changes
- Scale changes

**Limitation**: Only one level of undo is implemented.

## Debugging

### Debug Overlay

Press **F1** in Game Mode to show:
- FPS counter
- Frame time
- Object count
- Physics stats

### Console Output

The terminal shows:
- Script generation results
- Build errors
- Runtime warnings
- Physics collision events

## Building Standalone Games

### macOS .app Bundle

1. Press **Cmd/Ctrl+B** in the editor, or
2. Run `./mirgo-utils build MyGame`

Creates `build/MyGame.app` with:
- Compiled executable
- All assets embedded
- Proper macOS app structure
- Info.plist

Double-click to run or drag to Applications.

### Other Platforms

Use the Makefile targets:

```bash
make build-windows   # Requires MinGW cross-compiler
make build-linux     # Requires Linux cross-compiler
```

See [README](../README.md#utilities-mirgo-utils) for build tool details.

## Tips and Tricks

### Camera Navigation

- **Hold Right Mouse** before pressing WASD to fly
- **Scroll while holding Right Mouse** to adjust speed
- Higher speed = faster navigation for large scenes
- Lower speed = precise positioning

### Selecting Small Objects

- Zoom in close with the fly camera
- Use the Hierarchy panel to click by name
- Adjust camera speed for precision

### Asset Organization

- Keep models in `assets/models/`
- Keep scenes in `assets/scenes/`
- Keep scripts in `assets/scripts/`
- Use descriptive names

### Working with GLTF Models

- If a model appears dark/inverted, use "Flip Normals" in Inspector
- Scale models with object's `scale` property
- Models with embedded textures load automatically
- Normal maps are auto-detected

## Common Workflows

### Prototyping

1. Start in Editor Mode
2. Add primitive shapes (cubes, spheres)
3. Add physics components
4. Test in Game Mode (Cmd+P)
5. Iterate quickly

### Lighting Setup

1. Add DirectionalLight object
2. Adjust `direction` vector
3. Toggle Game Mode to see shadows
4. Tweak `intensity` for brightness

### Script Testing

1. Write script in `assets/scripts/`
2. Add to object in scene JSON
3. Press Cmd+R to hot reload
4. Test in Game Mode
5. Edit script, Cmd+R again

### Level Design

1. Block out geometry with cubes
2. Add GLTF models for detail
3. Set up physics colliders
4. Add lighting
5. Test player movement
6. Iterate

## Keyboard Shortcuts Summary

| Shortcut | Action |
|----------|--------|
| **Cmd/Ctrl+S** | Save scene |
| **Cmd/Ctrl+R** | Hot reload (regenerate + rebuild) |
| **Cmd/Ctrl+B** | Build standalone game |
| **Cmd/Ctrl+P** | Toggle editor/game mode |
| **Ctrl+Z** | Undo transform |
| **F1** | Toggle debug overlay (Game Mode) |
| **Right Mouse** | Activate fly camera |
| **Scroll** | Adjust fly speed |

## See Also

- [Getting Started](getting-started.md) - First steps with the editor
- [Scene Format](scene-format.md) - Understanding scene JSON files
- [Scripting Guide](scripting.md) - Adding custom behaviors
