package world

import (
	"fmt"
	"math"
	"math/rand"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ShadowMapResolution = 1024

type World struct {
	Scene       *engine.Scene
	LightDir    rl.Vector3
	Shader      rl.Shader
	FloorModel  rl.Model
	ShadowMap   rl.RenderTexture2D
	LightCamera rl.Camera3D
	MatLightVP  rl.Matrix
}

func New() *World {
	w := &World{
		Scene: engine.NewScene("Main"),
	}
	return w
}

const FloorSize = 60.0

func (w *World) Initialize() {
	// Load lighting shader
	w.Shader = rl.LoadShader("assets/shaders/lighting.vs", "assets/shaders/lighting.fs")

	// Create shadowmap render texture
	w.ShadowMap = LoadShadowmapRenderTexture(ShadowMapResolution, ShadowMapResolution)

	// Set directional light direction
	w.LightDir = rl.Vector3Normalize(rl.Vector3{X: 0.35, Y: -1.0, Z: -0.35})

	// Light camera
	w.LightCamera = rl.Camera3D{
		Position:   rl.Vector3Scale(w.LightDir, -50.0),
		Target:     rl.Vector3Zero(),
		Up:         getLightCameraUp(w.LightDir),
		Fovy:       FloorSize + 20,
		Projection: rl.CameraOrthographic,
	}

	// Set shader uniforms
	lightDirLoc := rl.GetShaderLocation(w.Shader, "lightDir")
	rl.SetShaderValue(w.Shader, lightDirLoc, []float32{w.LightDir.X, w.LightDir.Y, w.LightDir.Z}, rl.ShaderUniformVec3)

	lightColorLoc := rl.GetShaderLocation(w.Shader, "lightColor")
	rl.SetShaderValue(w.Shader, lightColorLoc, []float32{1.0, 1.0, 1.0, 1.0}, rl.ShaderUniformVec4)

	ambientLoc := rl.GetShaderLocation(w.Shader, "ambient")
	rl.SetShaderValue(w.Shader, ambientLoc, []float32{0.1, 0.1, 0.1, 1.0}, rl.ShaderUniformVec4)

	// Create floor with lighting shader
	floorMesh := rl.GenMeshPlane(FloorSize, FloorSize, 1, 1)
	w.FloorModel = rl.LoadModelFromMesh(floorMesh)
	w.FloorModel.Materials.Shader = w.Shader
	w.FloorModel.Materials.Maps.Color = rl.LightGray

	// Create cube GameObjects
	w.createCubes()

	// Start all GameObjects
	w.Scene.Start()
}

func (w *World) createCubes() {
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

		size := rl.Vector3{X: 1.5, Y: 1.5, Z: 1.5}
		color := colors[i%len(colors)]

		// Create GameObject
		cube := engine.NewGameObject(fmt.Sprintf("Cube_%d", i))
		cube.Transform.Position = pos

		// Create model and renderer
		mesh := rl.GenMeshCube(size.X, size.Y, size.Z)
		model := rl.LoadModelFromMesh(mesh)
		renderer := components.NewModelRenderer(model, color)
		renderer.SetShader(w.Shader)
		cube.AddComponent(renderer)

		// Add collider
		collider := components.NewBoxCollider(size)
		cube.AddComponent(collider)

		// Add animator
		animator := components.NewCubeAnimator(
			pos,
			rl.Vector3Normalize(rl.Vector3{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()}),
			float32(30+rand.Float64()*60),
			float32(2+rand.Float64()*3),
			float32(0.5+rand.Float64()*1.5),
			float32(rand.Float64()*2*math.Pi),
		)
		cube.AddComponent(animator)

		w.Scene.AddGameObject(cube)
	}
}

// LoadShadowmapRenderTexture creates a framebuffer with only depth attachment
func LoadShadowmapRenderTexture(width, height int32) rl.RenderTexture2D {
	target := rl.RenderTexture2D{}

	target.ID = rl.LoadFramebuffer()
	target.Texture.Width = width
	target.Texture.Height = height

	if target.ID > 0 {
		rl.EnableFramebuffer(target.ID)

		target.Depth.ID = rl.LoadTextureDepth(width, height, false)
		target.Depth.Width = width
		target.Depth.Height = height
		target.Depth.Format = 19
		target.Depth.Mipmaps = 1

		rl.FramebufferAttach(target.ID, target.Depth.ID, rl.AttachmentDepth, rl.AttachmentTexture2d, 0)

		rl.DisableFramebuffer()
	}

	return target
}

func (w *World) MoveLightDir(dx, dy, dz float32) {
	w.LightDir.X += dx
	w.LightDir.Y += dy
	w.LightDir.Z += dz
	w.LightDir = rl.Vector3Normalize(w.LightDir)

	w.LightCamera.Position = rl.Vector3Scale(w.LightDir, -50.0)
	w.LightCamera.Up = getLightCameraUp(w.LightDir)

	lightDirLoc := rl.GetShaderLocation(w.Shader, "lightDir")
	rl.SetShaderValue(w.Shader, lightDirLoc, []float32{w.LightDir.X, w.LightDir.Y, w.LightDir.Z}, rl.ShaderUniformVec3)
}

func getLightCameraUp(lightDir rl.Vector3) rl.Vector3 {
	if math.Abs(float64(lightDir.Y)) > 0.9 {
		return rl.Vector3{X: 0, Y: 0, Z: 1}
	}
	return rl.Vector3{X: 0, Y: 1, Z: 0}
}

func (w *World) Update(deltaTime float32) {
	w.Scene.Update(deltaTime)
}

const (
	ShadowNear float32 = 1.0
	ShadowFar  float32 = 150.0
)

func (w *World) DrawShadowMap() {
	rl.BeginTextureMode(w.ShadowMap)
	rl.ClearBackground(rl.White)

	rl.BeginMode3D(w.LightCamera)

	halfSize := w.LightCamera.Fovy / 2.0
	shadowProj := rl.MatrixOrtho(
		-halfSize, halfSize,
		-halfSize, halfSize,
		ShadowNear, ShadowFar,
	)
	rl.SetMatrixProjection(shadowProj)

	lightView := rl.GetMatrixModelview()
	lightProj := rl.GetMatrixProjection()

	rl.SetCullFace(0)
	w.drawScene()
	rl.SetCullFace(1)

	rl.EndMode3D()
	rl.EndTextureMode()

	rl.Viewport(0, 0, int32(rl.GetRenderWidth()), int32(rl.GetRenderHeight()))

	w.MatLightVP = rl.MatrixMultiply(lightView, lightProj)
}

func (w *World) drawScene() {
	// Draw floor
	rl.DrawModel(w.FloorModel, rl.Vector3Zero(), 1.0, rl.White)

	// Draw all GameObjects with ModelRenderer
	for _, g := range w.Scene.GameObjects {
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Draw()
		}
	}
}

func (w *World) DrawWithShadows(cameraPos rl.Vector3) {
	viewPosLoc := rl.GetShaderLocation(w.Shader, "viewPos")
	rl.SetShaderValue(w.Shader, viewPosLoc, []float32{cameraPos.X, cameraPos.Y, cameraPos.Z}, rl.ShaderUniformVec3)

	lightVPLoc := rl.GetShaderLocation(w.Shader, "matLightVP")
	rl.SetShaderValueMatrix(w.Shader, lightVPLoc, w.MatLightVP)

	shadowMapLoc := rl.GetShaderLocation(w.Shader, "shadowMap")
	rl.EnableShader(w.Shader.ID)

	textureSlot := int32(10)
	rl.ActiveTextureSlot(textureSlot)
	rl.EnableTexture(w.ShadowMap.Depth.ID)
	rl.SetUniform(shadowMapLoc, []int32{textureSlot}, int32(rl.ShaderUniformInt), 1)

	w.drawScene()

	// Draw light indicator
	lightIndicatorPos := rl.Vector3Scale(w.LightDir, -50)
	rl.DrawSphere(lightIndicatorPos, 0.5, rl.Yellow)
	rl.DrawLine3D(lightIndicatorPos, rl.Vector3Zero(), rl.Yellow)
}

// GetCollidableObjects returns all GameObjects that have BoxColliders
func (w *World) GetCollidableObjects() []*engine.GameObject {
	var result []*engine.GameObject
	for _, g := range w.Scene.GameObjects {
		if collider := engine.GetComponent[*components.BoxCollider](g); collider != nil {
			result = append(result, g)
		}
	}
	return result
}

func (w *World) Unload() {
	rl.UnloadShader(w.Shader)
	rl.UnloadRenderTexture(w.ShadowMap)

	// Unload all ModelRenderers
	for _, g := range w.Scene.GameObjects {
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Unload()
		}
	}
	rl.UnloadModel(w.FloorModel)
}
