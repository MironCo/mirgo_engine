package world

import (
	"encoding/json"
	"fmt"
	"os"
	"test3d/internal/assets"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- JSON types ---

type SceneFile struct {
	Objects []ObjectDef `json:"objects"`
}

type ObjectDef struct {
	UID        uint64            `json:"uid,omitempty"`
	Name       string            `json:"name"`
	Tags       []string          `json:"tags,omitempty"`
	Position   [3]float32        `json:"position"`
	Rotation   [3]float32        `json:"rotation"`
	Scale      [3]float32        `json:"scale"`
	Components []json.RawMessage `json:"components"`
	Children   []ObjectDef       `json:"children,omitempty"`
}

type componentHeader struct {
	Type string `json:"type"`
}

type modelRendererDef struct {
	Type      string    `json:"type"`
	Mesh      string    `json:"mesh,omitempty"`
	MeshSize  []float32 `json:"meshSize,omitempty"`
	Model     string    `json:"model,omitempty"`
	Material  string    `json:"material,omitempty"` // path to material JSON file
	Color     string    `json:"color,omitempty"`    // inline color (used if no material)
	Metallic  float32   `json:"metallic,omitempty"` // inline (used if no material)
	Roughness float32   `json:"roughness,omitempty"`
	Emissive  float32   `json:"emissive,omitempty"`
}

type boxColliderDef struct {
	Type   string     `json:"type"`
	Size   [3]float32 `json:"size"`
	Offset [3]float32 `json:"offset,omitempty"`
}

type sphereColliderDef struct {
	Type   string  `json:"type"`
	Radius float32 `json:"radius"`
}

type rigidbodyDef struct {
	Type        string  `json:"type"`
	Mass        float32 `json:"mass,omitempty"`
	Bounciness  float32 `json:"bounciness,omitempty"`
	Friction    float32 `json:"friction,omitempty"`
	UseGravity  *bool   `json:"useGravity,omitempty"`
	IsKinematic bool    `json:"isKinematic,omitempty"`
}

type directionalLightDef struct {
	Type      string     `json:"type"`
	Direction [3]float32 `json:"direction,omitempty"`
	Intensity float32    `json:"intensity,omitempty"`
}

type scriptDef struct {
	Type  string         `json:"type"`
	Name  string         `json:"name"`
	Props map[string]any `json:"props,omitempty"`
}

type cameraDef struct {
	Type   string  `json:"type"`
	FOV    float32 `json:"fov,omitempty"`
	Near   float32 `json:"near,omitempty"`
	Far    float32 `json:"far,omitempty"`
	IsMain bool    `json:"isMain,omitempty"`
}

// --- Color mapping ---

var colorByName = map[string]rl.Color{
	"Red":       rl.Red,
	"Blue":      rl.Blue,
	"Green":     rl.Green,
	"Purple":    rl.Purple,
	"Orange":    rl.Orange,
	"Yellow":    rl.Yellow,
	"Pink":      rl.Pink,
	"SkyBlue":   rl.SkyBlue,
	"Lime":      rl.Lime,
	"Magenta":   rl.Magenta,
	"White":     rl.White,
	"LightGray": rl.LightGray,
	"Gray":      rl.Gray,
	"DarkGray":  rl.DarkGray,
	"Black":     rl.Black,
	"Brown":     rl.Brown,
	"Beige":     rl.Beige,
	"Maroon":    rl.Maroon,
	"Gold":      rl.Gold,
}

var nameByColor map[rl.Color]string

func init() {
	nameByColor = make(map[rl.Color]string, len(colorByName))
	for name, c := range colorByName {
		nameByColor[c] = name
	}
}

func lookupColor(name string) rl.Color {
	if c, ok := colorByName[name]; ok {
		return c
	}
	return rl.White
}

func lookupColorName(c rl.Color) string {
	if name, ok := nameByColor[c]; ok {
		return name
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// --- Loading ---

func (w *World) LoadScene(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read scene: %w", err)
	}

	var sf SceneFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parse scene: %w", err)
	}

	for _, objDef := range sf.Objects {
		w.loadObject(objDef, nil)
	}

	return nil
}

func (w *World) loadObject(objDef ObjectDef, parent *engine.GameObject) {
	var g *engine.GameObject
	if objDef.UID > 0 {
		g = engine.NewGameObjectWithUID(objDef.Name, objDef.UID)
	} else {
		g = engine.NewGameObject(objDef.Name)
	}
	g.Tags = objDef.Tags
	g.Transform.Position = rl.Vector3{X: objDef.Position[0], Y: objDef.Position[1], Z: objDef.Position[2]}
	g.Transform.Rotation = rl.Vector3{X: objDef.Rotation[0], Y: objDef.Rotation[1], Z: objDef.Rotation[2]}

	// Default scale to 1 if zero
	if objDef.Scale == [3]float32{} {
		g.Transform.Scale = rl.Vector3{X: 1, Y: 1, Z: 1}
	} else {
		g.Transform.Scale = rl.Vector3{X: objDef.Scale[0], Y: objDef.Scale[1], Z: objDef.Scale[2]}
	}

	for _, raw := range objDef.Components {
		var header componentHeader
		if err := json.Unmarshal(raw, &header); err != nil {
			continue
		}

		// Try component registry first (for Serializable components)
		if comp := engine.CreateComponent(header.Type); comp != nil {
			var data map[string]any
			json.Unmarshal(raw, &data)
			comp.Deserialize(data)
			g.AddComponent(comp.(engine.Component))

			// Post-load hooks for components that need extra setup
			switch header.Type {
			case "MeshCollider":
				if mc, ok := comp.(*components.MeshCollider); ok {
					if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
						mc.BuildFromModel(renderer.Model)
					}
				}
			case "DirectionalLight":
				if light, ok := comp.(*components.DirectionalLight); ok {
					w.Light = g
					w.Renderer.SetLight(light)
				}
			}
			continue
		}

		switch header.Type {
		case "ModelRenderer":
			w.loadModelRenderer(g, raw)
		case "Script":
			loadScript(g, raw)
		}
	}

	if parent != nil {
		parent.AddChild(g)
	}

	// All objects go in the flat list for Start/Update/Draw
	w.Scene.AddGameObject(g)

	// Only root objects participate in physics
	if parent == nil {
		w.PhysicsWorld.AddObject(g)
	}

	// Recursively load children
	for _, childDef := range objDef.Children {
		w.loadObject(childDef, g)
	}
}

func (w *World) loadModelRenderer(g *engine.GameObject, raw json.RawMessage) {
	var def modelRendererDef
	if err := json.Unmarshal(raw, &def); err != nil {
		return
	}

	color := lookupColor(def.Color)

	var renderer *components.ModelRenderer
	if def.Model != "" {
		renderer = components.NewModelRendererFromFile(def.Model, color)
	} else {
		var model rl.Model
		switch def.Mesh {
		case "cube":
			if len(def.MeshSize) >= 3 {
				model = rl.LoadModelFromMesh(rl.GenMeshCube(def.MeshSize[0], def.MeshSize[1], def.MeshSize[2]))
			}
		case "plane":
			if len(def.MeshSize) >= 2 {
				model = rl.LoadModelFromMesh(rl.GenMeshPlane(def.MeshSize[0], def.MeshSize[1], 1, 1))
			}
		case "sphere":
			if len(def.MeshSize) >= 1 {
				model = rl.LoadModelFromMesh(rl.GenMeshSphere(def.MeshSize[0], 16, 16))
			}
		default:
			return
		}
		renderer = components.NewModelRenderer(model, color)
		renderer.MeshType = def.Mesh
		renderer.MeshSize = def.MeshSize
	}

	// Load material from file if specified, otherwise use inline properties
	if def.Material != "" {
		renderer.Material = assets.LoadMaterial(def.Material)
		renderer.MaterialPath = def.Material
	} else {
		renderer.Metallic = def.Metallic
		if def.Roughness > 0 {
			renderer.Roughness = def.Roughness
		}
		renderer.Emissive = def.Emissive
	}

	renderer.SetShader(w.Renderer.Shader)
	g.AddComponent(renderer)
}

func loadScript(g *engine.GameObject, raw json.RawMessage) {
	var def scriptDef
	if err := json.Unmarshal(raw, &def); err != nil {
		return
	}
	if comp := engine.CreateScript(def.Name, def.Props); comp != nil {
		g.AddComponent(comp)
	}
}

// --- Duplicating ---

// DuplicateObject creates a deep copy of a GameObject and adds it to the scene.
// Returns the new root object.
func (w *World) DuplicateObject(original *engine.GameObject) *engine.GameObject {
	// Serialize the object (including children)
	objDef := serializeObject(original)

	// Clear UIDs so new ones are generated
	clearUIDs(&objDef)

	// Rename to indicate copy
	objDef.Name = objDef.Name + "_copy"

	// Offset position slightly so it's visible
	objDef.Position[0] += 1.0

	// Load as new object with same parent
	return w.loadObjectAndReturn(objDef, original.Parent)
}

func clearUIDs(def *ObjectDef) {
	def.UID = 0
	for i := range def.Children {
		clearUIDs(&def.Children[i])
	}
}

// loadObjectAndReturn is like loadObject but returns the created object
func (w *World) loadObjectAndReturn(objDef ObjectDef, parent *engine.GameObject) *engine.GameObject {
	g := engine.NewGameObject(objDef.Name)
	g.Tags = objDef.Tags
	g.Transform.Position = rl.Vector3{X: objDef.Position[0], Y: objDef.Position[1], Z: objDef.Position[2]}
	g.Transform.Rotation = rl.Vector3{X: objDef.Rotation[0], Y: objDef.Rotation[1], Z: objDef.Rotation[2]}

	if objDef.Scale == [3]float32{} {
		g.Transform.Scale = rl.Vector3{X: 1, Y: 1, Z: 1}
	} else {
		g.Transform.Scale = rl.Vector3{X: objDef.Scale[0], Y: objDef.Scale[1], Z: objDef.Scale[2]}
	}

	for _, raw := range objDef.Components {
		var header componentHeader
		if err := json.Unmarshal(raw, &header); err != nil {
			continue
		}

		// Try component registry first (for Serializable components)
		if comp := engine.CreateComponent(header.Type); comp != nil {
			var data map[string]any
			json.Unmarshal(raw, &data)
			comp.Deserialize(data)
			g.AddComponent(comp.(engine.Component))

			// Post-load hooks for components that need extra setup
			switch header.Type {
			case "MeshCollider":
				if mc, ok := comp.(*components.MeshCollider); ok {
					if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
						mc.BuildFromModel(renderer.Model)
					}
				}
			case "DirectionalLight":
				if light, ok := comp.(*components.DirectionalLight); ok {
					w.Light = g
					w.Renderer.SetLight(light)
				}
			}
			continue
		}

		switch header.Type {
		case "ModelRenderer":
			w.loadModelRenderer(g, raw)
		case "Script":
			loadScript(g, raw)
		}
	}

	if parent != nil {
		parent.AddChild(g)
	}

	w.Scene.AddGameObject(g)

	if parent == nil {
		w.PhysicsWorld.AddObject(g)
	}

	// Recursively load children
	for _, childDef := range objDef.Children {
		w.loadObjectAndReturn(childDef, g)
	}

	return g
}

// --- Saving ---

func (w *World) SaveScene(path string) error {
	var sf SceneFile

	for _, g := range w.Scene.GameObjects {
		// Skip children (saved recursively under their parent)
		if g.Parent != nil {
			continue
		}

		sf.Objects = append(sf.Objects, serializeObject(g))
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scene: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write scene: %w", err)
	}

	return nil
}

func serializeObject(g *engine.GameObject) ObjectDef {
	objDef := ObjectDef{
		UID:      g.UID,
		Name:     g.Name,
		Tags:     g.Tags,
		Position: [3]float32{g.Transform.Position.X, g.Transform.Position.Y, g.Transform.Position.Z},
		Rotation: [3]float32{g.Transform.Rotation.X, g.Transform.Rotation.Y, g.Transform.Rotation.Z},
		Scale:    [3]float32{g.Transform.Scale.X, g.Transform.Scale.Y, g.Transform.Scale.Z},
	}

	for _, c := range g.Components() {
		if raw := serializeComponent(c); raw != nil {
			objDef.Components = append(objDef.Components, raw)
		}
	}

	for _, child := range g.Children {
		objDef.Children = append(objDef.Children, serializeObject(child))
	}

	return objDef
}

func serializeComponent(c engine.Component) json.RawMessage {
	var def any

	switch comp := c.(type) {
	case *components.ModelRenderer:
		d := modelRendererDef{
			Type: "ModelRenderer",
		}
		if comp.FilePath != "" {
			d.Model = comp.FilePath
		} else {
			d.Mesh = comp.MeshType
			d.MeshSize = comp.MeshSize
		}
		// Save material path if set, otherwise save inline properties
		if comp.MaterialPath != "" {
			d.Material = comp.MaterialPath
		} else {
			d.Color = lookupColorName(comp.Color)
			d.Metallic = comp.Metallic
			d.Roughness = comp.Roughness
			d.Emissive = comp.Emissive
		}
		def = d

	default:
		// Try Serializable interface first
		if s, ok := c.(engine.Serializable); ok {
			data := s.Serialize()
			// Add the type field from TypeName
			data["type"] = s.TypeName()
			def = data
		} else if name, props, ok := engine.SerializeScript(c); ok {
			// Try script registry
			def = scriptDef{Type: "Script", Name: name, Props: props}
		} else {
			return nil
		}
	}

	data, err := json.Marshal(def)
	if err != nil {
		return nil
	}
	return data
}
