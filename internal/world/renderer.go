package world

import (
	"math"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ShadowMapResolution = 2048

const (
	ShadowNear float32 = 1.0
	ShadowFar  float32 = 150.0
)

type Renderer struct {
	Shader      rl.Shader
	ShadowMap   rl.RenderTexture2D
	LightDir    rl.Vector3
	LightCamera rl.Camera3D
	MatLightVP  rl.Matrix
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Initialize(floorSize float32) {
	// Load lighting shader
	r.Shader = rl.LoadShader("assets/shaders/lighting.vs", "assets/shaders/lighting.fs")

	// Create shadowmap render texture
	r.ShadowMap = loadShadowmapRenderTexture(ShadowMapResolution, ShadowMapResolution)

	// Set directional light direction
	r.LightDir = rl.Vector3Normalize(rl.Vector3{X: 0.35, Y: -1.0, Z: -0.35})

	// Light camera
	r.LightCamera = rl.Camera3D{
		Position:   rl.Vector3Scale(r.LightDir, -50.0),
		Target:     rl.Vector3Zero(),
		Up:         lightCameraUp(r.LightDir),
		Fovy:       floorSize + 20,
		Projection: rl.CameraOrthographic,
	}

	// Set shader uniforms
	lightDirLoc := rl.GetShaderLocation(r.Shader, "lightDir")
	rl.SetShaderValue(r.Shader, lightDirLoc, []float32{r.LightDir.X, r.LightDir.Y, r.LightDir.Z}, rl.ShaderUniformVec3)

	lightColorLoc := rl.GetShaderLocation(r.Shader, "lightColor")
	rl.SetShaderValue(r.Shader, lightColorLoc, []float32{1.0, 1.0, 1.0, 1.0}, rl.ShaderUniformVec4)

	ambientLoc := rl.GetShaderLocation(r.Shader, "ambient")
	rl.SetShaderValue(r.Shader, ambientLoc, []float32{0.1, 0.1, 0.1, 1.0}, rl.ShaderUniformVec4)

}

func (r *Renderer) DrawShadowMap(gameObjects []*engine.GameObject) {
	rl.BeginTextureMode(r.ShadowMap)
	rl.ClearBackground(rl.White)

	rl.BeginMode3D(r.LightCamera)

	halfSize := r.LightCamera.Fovy / 2.0
	shadowProj := rl.MatrixOrtho(
		-halfSize, halfSize,
		-halfSize, halfSize,
		ShadowNear, ShadowFar,
	)
	rl.SetMatrixProjection(shadowProj)

	lightView := rl.GetMatrixModelview()
	lightProj := rl.GetMatrixProjection()

	rl.SetCullFace(0)
	r.drawScene(gameObjects)
	rl.SetCullFace(1)

	rl.EndMode3D()
	rl.EndTextureMode()

	rl.Viewport(0, 0, int32(rl.GetRenderWidth()), int32(rl.GetRenderHeight()))

	r.MatLightVP = rl.MatrixMultiply(lightView, lightProj)
}

func (r *Renderer) DrawWithShadows(cameraPos rl.Vector3, gameObjects []*engine.GameObject) {
	viewPosLoc := rl.GetShaderLocation(r.Shader, "viewPos")
	rl.SetShaderValue(r.Shader, viewPosLoc, []float32{cameraPos.X, cameraPos.Y, cameraPos.Z}, rl.ShaderUniformVec3)

	lightVPLoc := rl.GetShaderLocation(r.Shader, "matLightVP")
	rl.SetShaderValueMatrix(r.Shader, lightVPLoc, r.MatLightVP)

	shadowMapLoc := rl.GetShaderLocation(r.Shader, "shadowMap")
	rl.EnableShader(r.Shader.ID)

	textureSlot := int32(10)
	rl.ActiveTextureSlot(textureSlot)
	rl.EnableTexture(r.ShadowMap.Depth.ID)
	rl.SetUniform(shadowMapLoc, []int32{textureSlot}, int32(rl.ShaderUniformInt), 1)

	r.drawScene(gameObjects)

	// Draw light indicator
	lightIndicatorPos := rl.Vector3Scale(r.LightDir, -50)
	rl.DrawSphere(lightIndicatorPos, 0.5, rl.Yellow)
	rl.DrawLine3D(lightIndicatorPos, rl.Vector3Zero(), rl.Yellow)
}

func (r *Renderer) drawScene(gameObjects []*engine.GameObject) {
	// Draw all GameObjects with ModelRenderer
	for _, g := range gameObjects {
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Draw()
		}
	}
}

func (r *Renderer) MoveLightDir(dx, dy, dz float32) {
	r.LightDir.X += dx
	r.LightDir.Y += dy
	r.LightDir.Z += dz
	r.LightDir = rl.Vector3Normalize(r.LightDir)

	r.LightCamera.Position = rl.Vector3Scale(r.LightDir, -50.0)
	r.LightCamera.Up = lightCameraUp(r.LightDir)

	lightDirLoc := rl.GetShaderLocation(r.Shader, "lightDir")
	rl.SetShaderValue(r.Shader, lightDirLoc, []float32{r.LightDir.X, r.LightDir.Y, r.LightDir.Z}, rl.ShaderUniformVec3)
}

func (r *Renderer) Unload(gameObjects []*engine.GameObject) {
	rl.UnloadShader(r.Shader)
	rl.UnloadRenderTexture(r.ShadowMap)

	for _, g := range gameObjects {
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Unload()
		}
	}
}

func lightCameraUp(lightDir rl.Vector3) rl.Vector3 {
	if math.Abs(float64(lightDir.Y)) > 0.9 {
		return rl.Vector3{X: 0, Y: 0, Z: 1}
	}
	return rl.Vector3{X: 0, Y: 1, Z: 0}
}

func loadShadowmapRenderTexture(width, height int32) rl.RenderTexture2D {
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
