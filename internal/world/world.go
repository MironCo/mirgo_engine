package world

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ShadowMapResolution = 1024

type World struct {
	Objects     []AnimatedCube
	LightDir    rl.Vector3
	Shader      rl.Shader
	FloorModel  rl.Model
	ShadowMap   rl.RenderTexture2D // Use raylib's RenderTexture2D
	LightCamera rl.Camera3D
	MatLightVP  rl.Matrix // Light view-projection matrix, captured each frame
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

const FloorSize = 60.0

func (w *World) Initialize() {
	// Load lighting shader
	w.Shader = rl.LoadShader("assets/shaders/lighting.vs", "assets/shaders/lighting.fs")

	// Create shadowmap render texture (matching raylib example)
	w.ShadowMap = LoadShadowmapRenderTexture(ShadowMapResolution, ShadowMapResolution)

	// Set directional light direction (pointing DOWN and toward scene, like raylib example)
	w.LightDir = rl.Vector3Normalize(rl.Vector3{X: 0.35, Y: -1.0, Z: -0.35})

	// Light camera - positioned opposite to light direction
	// Distance needs to be far enough to see the whole scene
	// Fovy for orthographic = size of the view (needs to cover FloorSize)
	w.LightCamera = rl.Camera3D{
		Position:   rl.Vector3Scale(w.LightDir, -50.0), // Further back
		Target:     rl.Vector3Zero(),
		Up:         getLightCameraUp(w.LightDir), // Use dynamic up vector
		Fovy:       FloorSize + 20,               // Cover entire floor plus margin
		Projection: rl.CameraOrthographic,
	}

	// Set shader uniforms
	lightDirLoc := rl.GetShaderLocation(w.Shader, "lightDir")
	rl.SetShaderValue(w.Shader, lightDirLoc, []float32{w.LightDir.X, w.LightDir.Y, w.LightDir.Z}, rl.ShaderUniformVec3)

	lightColorLoc := rl.GetShaderLocation(w.Shader, "lightColor")
	rl.SetShaderValue(w.Shader, lightColorLoc, []float32{1.0, 1.0, 1.0, 1.0}, rl.ShaderUniformVec4)

	ambientLoc := rl.GetShaderLocation(w.Shader, "ambient")
	rl.SetShaderValue(w.Shader, ambientLoc, []float32{0.1, 0.1, 0.1, 1.0}, rl.ShaderUniformVec4)

	// Initialize cube models with lighting shader
	for i := range w.Objects {
		mesh := rl.GenMeshCube(w.Objects[i].Size.X, w.Objects[i].Size.Y, w.Objects[i].Size.Z)
		w.Objects[i].Model = rl.LoadModelFromMesh(mesh)
		w.Objects[i].Model.Materials.Shader = w.Shader
		w.Objects[i].Model.Materials.Maps.Color = w.Objects[i].Color
	}

	// Create floor with lighting shader
	floorMesh := rl.GenMeshPlane(FloorSize, FloorSize, 1, 1)
	w.FloorModel = rl.LoadModelFromMesh(floorMesh)
	w.FloorModel.Materials.Shader = w.Shader
	w.FloorModel.Materials.Maps.Color = rl.LightGray
}

// LoadShadowmapRenderTexture creates a framebuffer with only depth attachment (matching raylib example)
func LoadShadowmapRenderTexture(width, height int32) rl.RenderTexture2D {
	target := rl.RenderTexture2D{}

	target.ID = rl.LoadFramebuffer()
	target.Texture.Width = width
	target.Texture.Height = height

	if target.ID > 0 {
		rl.EnableFramebuffer(target.ID)

		// Create depth texture (NO color texture needed for shadowmap)
		target.Depth.ID = rl.LoadTextureDepth(width, height, false)
		target.Depth.Width = width
		target.Depth.Height = height
		target.Depth.Format = 19 // DEPTH_COMPONENT_24BIT
		target.Depth.Mipmaps = 1

		// Attach depth texture to FBO
		rl.FramebufferAttach(target.ID, target.Depth.ID, rl.AttachmentDepth, rl.AttachmentTexture2d, 0)

		rl.DisableFramebuffer()
	}

	return target
}

// MoveLightDir adjusts the light direction and updates the light camera
func (w *World) MoveLightDir(dx, dy, dz float32) {
	w.LightDir.X += dx
	w.LightDir.Y += dy
	w.LightDir.Z += dz
	w.LightDir = rl.Vector3Normalize(w.LightDir)

	// Update light camera position (same distance as initialization)
	w.LightCamera.Position = rl.Vector3Scale(w.LightDir, -50.0)

	// Fix Up vector for vertical light directions
	// When light is nearly vertical, use Z-forward as up instead
	w.LightCamera.Up = getLightCameraUp(w.LightDir)

	// Update shader uniform
	lightDirLoc := rl.GetShaderLocation(w.Shader, "lightDir")
	rl.SetShaderValue(w.Shader, lightDirLoc, []float32{w.LightDir.X, w.LightDir.Y, w.LightDir.Z}, rl.ShaderUniformVec3)
}

// getLightCameraUp returns an appropriate up vector for the light camera
// avoiding the degenerate case when light direction is parallel to Y-up
func getLightCameraUp(lightDir rl.Vector3) rl.Vector3 {
	// If light is nearly vertical (pointing up or down), use Z as up
	// Threshold of 0.9 gives buffer before view matrix becomes degenerate
	if math.Abs(float64(lightDir.Y)) > 0.9 {
		return rl.Vector3{X: 0, Y: 0, Z: 1}
	}
	// Otherwise use standard Y-up
	return rl.Vector3{X: 0, Y: 1, Z: 0}
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

// Shadow camera near/far planes - tight range for better depth precision
const (
	ShadowNear float32 = 1.0
	ShadowFar  float32 = 150.0
)

// DrawShadowMap renders the scene from light's perspective (PASS 01 in raylib example)
func (w *World) DrawShadowMap() {
	rl.BeginTextureMode(w.ShadowMap)
	rl.ClearBackground(rl.White)

	rl.BeginMode3D(w.LightCamera)

	// Override projection matrix with proper near/far planes for depth precision
	// Default raylib uses 0.01-1000 which compresses all depth to nearly the same value
	halfSize := w.LightCamera.Fovy / 2.0
	shadowProj := rl.MatrixOrtho(
		-halfSize, halfSize, // left, right
		-halfSize, halfSize, // bottom, top
		ShadowNear, ShadowFar, // near, far - CRITICAL for depth precision
	)
	rl.SetMatrixProjection(shadowProj)

	// Capture matrices AFTER setting custom projection
	lightView := rl.GetMatrixModelview()
	lightProj := rl.GetMatrixProjection()

	// Cull front faces during shadow pass — writes back-face depth into the shadowmap,
	// which naturally prevents self-shadowing (shadow acne) on lit surfaces
	rl.SetCullFace(0) // 0 = RL_CULL_FACE_FRONT

	// Draw the scene (same objects as main pass)
	w.drawScene()

	// Restore default back-face culling for the main render pass
	rl.SetCullFace(1) // 1 = RL_CULL_FACE_BACK

	rl.EndMode3D()
	rl.EndTextureMode()

	// Reset viewport to full framebuffer size for main rendering (use RenderWidth/Height for Retina)
	rl.Viewport(0, 0, int32(rl.GetRenderWidth()), int32(rl.GetRenderHeight()))

	// Calculate lightViewProj AFTER EndTextureMode (like raylib example)
	w.MatLightVP = rl.MatrixMultiply(lightView, lightProj)
}

// drawScene draws all objects (used by both shadow and main pass)
func (w *World) drawScene() {
	// Draw floor
	rl.DrawModel(w.FloorModel, rl.Vector3Zero(), 1.0, rl.White)

	// Draw cubes
	for _, obj := range w.Objects {
		rl.DrawModel(obj.Model, obj.Position, 1.0, rl.White)
	}
}

// DrawWithShadows renders the main scene with shadowmapping (PASS 02 in raylib example)
func (w *World) DrawWithShadows(cameraPos rl.Vector3) {
	// Update view position for specular
	viewPosLoc := rl.GetShaderLocation(w.Shader, "viewPos")
	rl.SetShaderValue(w.Shader, viewPosLoc, []float32{cameraPos.X, cameraPos.Y, cameraPos.Z}, rl.ShaderUniformVec3)

	// Pass light view-projection matrix
	lightVPLoc := rl.GetShaderLocation(w.Shader, "matLightVP")
	rl.SetShaderValueMatrix(w.Shader, lightVPLoc, w.MatLightVP)

	// Enable shader and bind shadowmap texture (matching raylib example exactly)
	shadowMapLoc := rl.GetShaderLocation(w.Shader, "shadowMap")
	rl.EnableShader(w.Shader.ID)

	textureSlot := int32(10) // Use slot 10 like raylib example
	rl.ActiveTextureSlot(textureSlot)
	rl.EnableTexture(w.ShadowMap.Depth.ID)
	rl.SetUniform(shadowMapLoc, []int32{textureSlot}, int32(rl.ShaderUniformInt), 1)

	// Draw the same scene
	w.drawScene()

	// Draw light indicator (without shadows)
	lightIndicatorPos := rl.Vector3Scale(w.LightDir, -50)
	rl.DrawSphere(lightIndicatorPos, 0.5, rl.Yellow)
	rl.DrawLine3D(lightIndicatorPos, rl.Vector3Zero(), rl.Yellow)
}

func (w *World) Unload() {
	rl.UnloadShader(w.Shader)
	rl.UnloadRenderTexture(w.ShadowMap)

	for i := range w.Objects {
		rl.UnloadModel(w.Objects[i].Model)
	}
	rl.UnloadModel(w.FloorModel)
}
