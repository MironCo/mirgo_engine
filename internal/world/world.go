package world

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ShadowMapResolution = 2048

type World struct {
	Objects     []AnimatedCube
	LightDir    rl.Vector3
	Shader      rl.Shader
	DepthShader rl.Shader
	FloorModel  rl.Model
	ShadowMap   rl.RenderTexture2D
	LightCamera rl.Camera3D
	MatLightVP  rl.Matrix
	time        float32
}

type AnimatedCube struct {
	Position        rl.Vector3
	StartPosition   rl.Vector3
	Size            rl.Vector3
	Color           rl.Color
	Model           rl.Model
	RotationAxis    rl.Vector3
	RotationSpeed   float32
	CurrentRotation float32
	MovementRadius  float32
	MovementSpeed   float32
	Phase           float32
}

func New() *World {
	w := &World{
		Objects: make([]AnimatedCube, 0),
	}

	numCubes := 15
	colors := []rl.Color{
		rl.Red, rl.Blue, rl.Green, rl.Purple, rl.Orange,
		rl.Yellow, rl.Pink, rl.SkyBlue, rl.Lime, rl.Magenta,
	}

	for i := range numCubes {
		angle := float32(i) * (2 * math.Pi / float32(numCubes))
		radius := float32(8 + rand.Float64()*5)

		pos := rl.Vector3{
			X: float32(math.Cos(float64(angle))) * radius,
			Y: float32(2 + rand.Float64()*3),
			Z: float32(math.Sin(float64(angle))) * radius,
		}

		cube := AnimatedCube{
			Position:       pos,
			StartPosition:  pos,
			Size:           rl.Vector3{X: 1.5, Y: 1.5, Z: 1.5},
			Color:          colors[i%len(colors)],
			RotationAxis:   rl.Vector3Normalize(rl.Vector3{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()}),
			RotationSpeed:  float32(30 + rand.Float64()*60),
			MovementRadius: float32(2 + rand.Float64()*3),
			MovementSpeed:  float32(0.5 + rand.Float64()*1.5),
			Phase:          float32(rand.Float64() * 2 * math.Pi),
		}

		w.Objects = append(w.Objects, cube)
	}

	return w
}

func (w *World) Initialize() {
	// Load shaders from files
	w.Shader = rl.LoadShader("assets/shaders/lighting.vs", "assets/shaders/lighting.fs")
	w.DepthShader = rl.LoadShader("assets/shaders/depth.vs", "assets/shaders/depth.fs")

	// Load shadow map
	w.ShadowMap = rl.LoadRenderTexture(ShadowMapResolution, ShadowMapResolution)

	// Set directional light direction
	w.LightDir = rl.Vector3Normalize(rl.Vector3{X: 1.0, Y: 1.0, Z: 1.0})

	// Setup light camera for shadowmap
	lightPos := rl.Vector3Scale(w.LightDir, 40)
	w.LightCamera = rl.Camera3D{
		Position:   lightPos,
		Target:     rl.Vector3{X: 0, Y: 0, Z: 0},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       60,
		Projection: rl.CameraOrthographic,
	}

	// Set shader uniforms
	lightDirLoc := rl.GetShaderLocation(w.Shader, "lightDir")
	rl.SetShaderValue(w.Shader, lightDirLoc, []float32{w.LightDir.X, w.LightDir.Y, w.LightDir.Z}, rl.ShaderUniformVec3)

	lightColorLoc := rl.GetShaderLocation(w.Shader, "lightColor")
	rl.SetShaderValue(w.Shader, lightColorLoc, []float32{1.0, 0.95, 0.9, 1.0}, rl.ShaderUniformVec4)

	ambientLoc := rl.GetShaderLocation(w.Shader, "ambient")
	rl.SetShaderValue(w.Shader, ambientLoc, []float32{0.2, 0.2, 0.25, 1.0}, rl.ShaderUniformVec4)

	// Initialize cube models
	for i := range w.Objects {
		mesh := rl.GenMeshCube(w.Objects[i].Size.X, w.Objects[i].Size.Y, w.Objects[i].Size.Z)
		w.Objects[i].Model = rl.LoadModelFromMesh(mesh)
		w.Objects[i].Model.Materials.Shader = w.Shader
		w.Objects[i].Model.Materials.Maps.Color = w.Objects[i].Color
	}

	// Create floor
	floorMesh := rl.GenMeshPlane(60, 60, 10, 10)
	w.FloorModel = rl.LoadModelFromMesh(floorMesh)
	w.FloorModel.Materials.Shader = w.Shader
	w.FloorModel.Materials.Maps.Color = rl.NewColor(80, 80, 90, 255)
}

func (w *World) Update(deltaTime float32) {
	w.time += deltaTime

	for i := range w.Objects {
		cube := &w.Objects[i]

		t := w.time*cube.MovementSpeed + cube.Phase
		offset := rl.Vector3{
			X: float32(math.Cos(float64(t))) * cube.MovementRadius,
			Y: float32(math.Sin(float64(t*2))) * 1.5,
			Z: float32(math.Sin(float64(t))) * cube.MovementRadius,
		}

		cube.Position = rl.Vector3Add(cube.StartPosition, offset)

		cube.CurrentRotation += cube.RotationSpeed * deltaTime
		if cube.CurrentRotation > 360 {
			cube.CurrentRotation -= 360
		}
	}
}

// DrawShadowMap renders the scene from light's perspective
func (w *World) DrawShadowMap() {
	rl.BeginTextureMode(w.ShadowMap)
	rl.ClearBackground(rl.White)

	rl.BeginMode3D(w.LightCamera)

	lightOrthoSize := float32(40)
	rl.SetMatrixProjection(rl.MatrixOrtho(
		-lightOrthoSize, lightOrthoSize,
		-lightOrthoSize, lightOrthoSize,
		0.1, 100.0,
	))

	w.MatLightVP = rl.MatrixMultiply(rl.GetMatrixModelview(), rl.GetMatrixProjection())

	// Draw with depth shader
	w.FloorModel.Materials.Shader = w.DepthShader
	for i := range w.Objects {
		w.Objects[i].Model.Materials.Shader = w.DepthShader
	}

	rl.DrawModel(w.FloorModel, rl.Vector3Zero(), 1.0, rl.White)
	for _, obj := range w.Objects {
		rl.DrawModel(obj.Model, obj.Position, 1.0, rl.White)
	}

	// Restore main shader
	w.FloorModel.Materials.Shader = w.Shader
	for i := range w.Objects {
		w.Objects[i].Model.Materials.Shader = w.Shader
	}

	rl.EndMode3D()
	rl.EndTextureMode()
}

// DrawWithShadows renders the scene with lighting
func (w *World) DrawWithShadows(cameraPos rl.Vector3) {
	// Update view position for specular
	viewPosLoc := rl.GetShaderLocation(w.Shader, "viewPos")
	rl.SetShaderValue(w.Shader, viewPosLoc, []float32{cameraPos.X, cameraPos.Y, cameraPos.Z}, rl.ShaderUniformVec3)

	// Draw floor
	rl.DrawModel(w.FloorModel, rl.Vector3Zero(), 1.0, rl.White)

	// Draw cubes
	for _, obj := range w.Objects {
		rl.DrawModel(obj.Model, obj.Position, 1.0, rl.White)
	}

	// Draw light indicator
	lightIndicatorPos := rl.Vector3Scale(w.LightDir, 15)
	rl.DrawSphere(lightIndicatorPos, 0.5, rl.Yellow)
	rl.DrawLine3D(lightIndicatorPos, rl.Vector3Zero(), rl.Yellow)
}

func (w *World) Unload() {
	rl.UnloadShader(w.Shader)
	rl.UnloadShader(w.DepthShader)
	rl.UnloadRenderTexture(w.ShadowMap)
	for i := range w.Objects {
		rl.UnloadModel(w.Objects[i].Model)
	}
	rl.UnloadModel(w.FloorModel)
}
