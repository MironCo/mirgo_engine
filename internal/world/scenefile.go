package world

import (
	"encoding/json"
	"fmt"
	"os"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- JSON types ---

type SceneFile struct {
	Objects []ObjectDef `json:"objects"`
}

type ObjectDef struct {
	Name       string            `json:"name"`
	Tags       []string          `json:"tags,omitempty"`
	Position   [3]float32        `json:"position"`
	Rotation   [3]float32        `json:"rotation"`
	Scale      [3]float32        `json:"scale"`
	Components []json.RawMessage `json:"components"`
}

type componentHeader struct {
	Type string `json:"type"`
}

type modelRendererDef struct {
	Type     string    `json:"type"`
	Mesh     string    `json:"mesh,omitempty"`
	MeshSize []float32 `json:"meshSize,omitempty"`
	Model    string    `json:"model,omitempty"`
	Color    string    `json:"color"`
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
		g := engine.NewGameObject(objDef.Name)
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

			switch header.Type {
			case "ModelRenderer":
				w.loadModelRenderer(g, raw)
			case "BoxCollider":
				w.loadBoxCollider(g, raw)
			case "SphereCollider":
				w.loadSphereCollider(g, raw)
			case "Rigidbody":
				w.loadRigidbody(g, raw)
			case "DirectionalLight":
				w.loadDirectionalLight(g, raw)
			case "Script":
				loadScript(g, raw)
			}
		}

		w.Scene.AddGameObject(g)
		w.PhysicsWorld.AddObject(g)
	}

	return nil
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

	renderer.SetShader(w.Renderer.Shader)
	g.AddComponent(renderer)
}

func (w *World) loadBoxCollider(g *engine.GameObject, raw json.RawMessage) {
	var def boxColliderDef
	if err := json.Unmarshal(raw, &def); err != nil {
		return
	}
	col := components.NewBoxCollider(rl.Vector3{X: def.Size[0], Y: def.Size[1], Z: def.Size[2]})
	col.Offset = rl.Vector3{X: def.Offset[0], Y: def.Offset[1], Z: def.Offset[2]}
	g.AddComponent(col)
}

func (w *World) loadSphereCollider(g *engine.GameObject, raw json.RawMessage) {
	var def sphereColliderDef
	if err := json.Unmarshal(raw, &def); err != nil {
		return
	}
	g.AddComponent(components.NewSphereCollider(def.Radius))
}

func (w *World) loadRigidbody(g *engine.GameObject, raw json.RawMessage) {
	var def rigidbodyDef
	if err := json.Unmarshal(raw, &def); err != nil {
		return
	}
	rb := components.NewRigidbody()
	if def.Mass > 0 {
		rb.Mass = def.Mass
	}
	if def.Bounciness > 0 {
		rb.Bounciness = def.Bounciness
	}
	if def.Friction > 0 {
		rb.Friction = def.Friction
	}
	if def.UseGravity != nil {
		rb.UseGravity = *def.UseGravity
	}
	rb.IsKinematic = def.IsKinematic
	g.AddComponent(rb)
}

func (w *World) loadDirectionalLight(g *engine.GameObject, raw json.RawMessage) {
	var def directionalLightDef
	if err := json.Unmarshal(raw, &def); err != nil {
		return
	}
	light := components.NewDirectionalLight()
	if def.Direction != [3]float32{} {
		light.Direction = rl.Vector3Normalize(rl.Vector3{X: def.Direction[0], Y: def.Direction[1], Z: def.Direction[2]})
	}
	if def.Intensity > 0 {
		light.Intensity = def.Intensity
	}
	g.AddComponent(light)

	// Wire light to renderer
	w.Light = g
	w.Renderer.SetLight(light)
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

// --- Saving ---

func (w *World) SaveScene(path string) error {
	var sf SceneFile

	for _, g := range w.Scene.GameObjects {
		// Skip player (code-managed)
		if engine.GetComponent[*components.FPSController](g) != nil {
			continue
		}
		// Skip runtime-spawned projectiles
		if engine.GetComponent[*components.Shooter](g) != nil {
			continue
		}

		objDef := ObjectDef{
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

		sf.Objects = append(sf.Objects, objDef)
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

func serializeComponent(c engine.Component) json.RawMessage {
	var def any

	switch comp := c.(type) {
	case *components.ModelRenderer:
		d := modelRendererDef{
			Type:  "ModelRenderer",
			Color: lookupColorName(comp.Color),
		}
		if comp.FilePath != "" {
			d.Model = comp.FilePath
		} else {
			d.Mesh = comp.MeshType
			d.MeshSize = comp.MeshSize
		}
		def = d

	case *components.BoxCollider:
		def = boxColliderDef{
			Type:   "BoxCollider",
			Size:   [3]float32{comp.Size.X, comp.Size.Y, comp.Size.Z},
			Offset: [3]float32{comp.Offset.X, comp.Offset.Y, comp.Offset.Z},
		}

	case *components.SphereCollider:
		def = sphereColliderDef{
			Type:   "SphereCollider",
			Radius: comp.Radius,
		}

	case *components.Rigidbody:
		useGravity := comp.UseGravity
		def = rigidbodyDef{
			Type:        "Rigidbody",
			Mass:        comp.Mass,
			Bounciness:  comp.Bounciness,
			Friction:    comp.Friction,
			UseGravity:  &useGravity,
			IsKinematic: comp.IsKinematic,
		}

	case *components.DirectionalLight:
		def = directionalLightDef{
			Type:      "DirectionalLight",
			Direction: [3]float32{comp.Direction.X, comp.Direction.Y, comp.Direction.Z},
			Intensity: comp.Intensity,
		}

	default:
		// Try script registry
		if name, props, ok := engine.SerializeScript(c); ok {
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
