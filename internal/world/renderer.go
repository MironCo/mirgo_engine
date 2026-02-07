package world

import (
	"test3d/internal/components"
	"test3d/internal/engine"
	"unsafe"

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
	Light       *components.DirectionalLight
	LightCamera rl.Camera3D
	MatLightVP  rl.Matrix
	floorSize   float32
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Initialize(floorSize float32) {
	r.floorSize = floorSize

	// Load lighting shader
	r.Shader = rl.LoadShader("assets/shaders/lighting.vs", "assets/shaders/lighting.fs")

	// Set shader locations for material maps so raylib knows where to bind them
	// Normal map goes to texture slot 1 (texture1 in our shader)
	locs := unsafe.Slice(r.Shader.Locs, rl.ShaderLocMapCubemap+1) // Enough for all shader locs
	locs[rl.ShaderLocMapNormal] = rl.GetShaderLocation(r.Shader, "texture1")

	// Create shadowmap render texture
	r.ShadowMap = loadShadowmapRenderTexture(ShadowMapResolution, ShadowMapResolution)
}

func (r *Renderer) SetLight(light *components.DirectionalLight) {
	r.Light = light
	r.updateLightCamera()
	r.updateShaderUniforms()
}

func (r *Renderer) updateLightCamera() {
	if r.Light == nil {
		return
	}
	r.LightCamera = r.Light.GetLightCamera(r.floorSize + 20)
}

func (r *Renderer) updateShaderUniforms() {
	if r.Light == nil {
		return
	}

	lightDirLoc := rl.GetShaderLocation(r.Shader, "lightDir")
	rl.SetShaderValue(r.Shader, lightDirLoc, []float32{r.Light.Direction.X, r.Light.Direction.Y, r.Light.Direction.Z}, rl.ShaderUniformVec3)

	lightColorLoc := rl.GetShaderLocation(r.Shader, "lightColor")
	rl.SetShaderValue(r.Shader, lightColorLoc, r.Light.GetColorFloat(), rl.ShaderUniformVec4)

	ambientLoc := rl.GetShaderLocation(r.Shader, "ambient")
	rl.SetShaderValue(r.Shader, ambientLoc, r.Light.GetAmbientFloat(), rl.ShaderUniformVec4)
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
	if r.Light != nil {
		lightIndicatorPos := rl.Vector3Scale(r.Light.Direction, -r.Light.ShadowDistance)
		rl.DrawSphere(lightIndicatorPos, 0.5, rl.Yellow)
		rl.DrawLine3D(lightIndicatorPos, rl.Vector3Zero(), rl.Yellow)
	}
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
	if r.Light == nil {
		return
	}
	r.Light.MoveLightDir(dx, dy, dz)
	r.updateLightCamera()

	lightDirLoc := rl.GetShaderLocation(r.Shader, "lightDir")
	rl.SetShaderValue(r.Shader, lightDirLoc, []float32{r.Light.Direction.X, r.Light.Direction.Y, r.Light.Direction.Z}, rl.ShaderUniformVec3)
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
