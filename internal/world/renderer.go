package world

import (
	"test3d/internal/components"
	"test3d/internal/engine"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ShadowMapResolution = 2048
const MaxPointLights = 4

const (
	ShadowNear float32 = 1.0
	ShadowFar  float32 = 150.0
)

type Renderer struct {
	Shader          rl.Shader
	InstanceShader  rl.Shader
	ShadowMap       rl.RenderTexture2D
	Light           *components.DirectionalLight
	LightCamera     rl.Camera3D
	MatLightVP      rl.Matrix
	floorSize       float32
	frustum         Frustum // current frame's view frustum for culling
	CullEnabled     bool    // frustum culling toggle (default true)

	// Stats for debug display
	DrawnObjects int // objects rendered this frame
	CulledObjects int // objects culled this frame
}

func NewRenderer() *Renderer {
	return &Renderer{
		CullEnabled: true, // frustum culling on by default
	}
}

func (r *Renderer) Initialize(floorSize float32) {
	r.floorSize = floorSize

	// Load lighting shader for regular models
	r.Shader = rl.LoadShader("assets/shaders/lighting.vs", "assets/shaders/lighting.fs")

	// Set shader locations for material maps so raylib knows where to bind them
	// Normal map goes to texture slot 1 (texture1 in our shader)
	locs := unsafe.Slice(r.Shader.Locs, rl.ShaderLocMapCubemap+1) // Enough for all shader locs
	locs[rl.ShaderLocMapNormal] = rl.GetShaderLocation(r.Shader, "texture1")
	locs[rl.ShaderLocMatrixNormal] = rl.GetShaderLocation(r.Shader, "matNormal")

	// Load instancing shader for batched meshes
	r.InstanceShader = rl.LoadShader("assets/shaders/instancing.vs", "assets/shaders/lighting.fs")

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

	lightDir := []float32{r.Light.Direction.X, r.Light.Direction.Y, r.Light.Direction.Z}
	lightColor := r.Light.GetColorFloat()
	ambient := r.Light.GetAmbientFloat()

	// Update both shaders
	for _, shader := range []rl.Shader{r.Shader, r.InstanceShader} {
		lightDirLoc := rl.GetShaderLocation(shader, "lightDir")
		rl.SetShaderValue(shader, lightDirLoc, lightDir, rl.ShaderUniformVec3)

		lightColorLoc := rl.GetShaderLocation(shader, "lightColor")
		rl.SetShaderValue(shader, lightColorLoc, lightColor, rl.ShaderUniformVec4)

		ambientLoc := rl.GetShaderLocation(shader, "ambient")
		rl.SetShaderValue(shader, ambientLoc, ambient, rl.ShaderUniformVec4)
	}
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

	// Disable face culling during shadow pass for better shadow coverage
	rl.DisableBackfaceCulling()
	r.drawScene(gameObjects)
	rl.EnableBackfaceCulling()

	rl.EndMode3D()
	rl.EndTextureMode()

	rl.Viewport(0, 0, int32(rl.GetRenderWidth()), int32(rl.GetRenderHeight()))

	r.MatLightVP = rl.MatrixMultiply(lightView, lightProj)
}

func (r *Renderer) DrawWithShadows(camera rl.Camera3D, gameObjects []*engine.GameObject) {
	// Sync light uniforms and camera every frame (in case editor changed them)
	r.updateLightCamera()
	r.updateShaderUniforms()

	// Extract frustum planes for culling
	if r.CullEnabled {
		r.frustum = ExtractFrustum(camera)
	}

	viewPos := []float32{camera.Position.X, camera.Position.Y, camera.Position.Z}

	// Update both shaders with view position and light VP matrix
	for _, shader := range []rl.Shader{r.Shader, r.InstanceShader} {
		viewPosLoc := rl.GetShaderLocation(shader, "viewPos")
		rl.SetShaderValue(shader, viewPosLoc, viewPos, rl.ShaderUniformVec3)

		lightVPLoc := rl.GetShaderLocation(shader, "matLightVP")
		rl.SetShaderValueMatrix(shader, lightVPLoc, r.MatLightVP)
	}

	// Collect and set point lights
	r.updatePointLights(gameObjects)

	// Bind shadow map for both shaders
	textureSlot := int32(10)
	rl.ActiveTextureSlot(textureSlot)
	rl.EnableTexture(r.ShadowMap.Depth.ID)

	for _, shader := range []rl.Shader{r.Shader, r.InstanceShader} {
		shadowMapLoc := rl.GetShaderLocation(shader, "shadowMap")
		rl.EnableShader(shader.ID)
		rl.SetUniform(shadowMapLoc, []int32{textureSlot}, int32(rl.ShaderUniformInt), 1)
	}

	r.drawScene(gameObjects)

	// Draw light indicator
	if r.Light != nil {
		lightIndicatorPos := rl.Vector3Scale(r.Light.Direction, -r.Light.ShadowDistance)
		rl.DrawSphere(lightIndicatorPos, 0.5, rl.Yellow)
		rl.DrawLine3D(lightIndicatorPos, rl.Vector3Zero(), rl.Yellow)
	}
}

// instanceBatch groups objects by mesh type for instanced rendering
type instanceBatch struct {
	mesh      rl.Mesh
	material  rl.Material
	color     rl.Color
	transforms []rl.Matrix
}

func (r *Renderer) drawScene(gameObjects []*engine.GameObject) {
	// Reset stats
	r.DrawnObjects = 0
	r.CulledObjects = 0

	// Group objects by mesh type for instanced rendering
	batches := make(map[string]*instanceBatch)

	for _, g := range gameObjects {
		if !g.Active {
			continue
		}
		mr := engine.GetComponent[*components.ModelRenderer](g)
		if mr == nil {
			continue
		}

		// Frustum culling: skip objects outside the view frustum
		if r.CullEnabled {
			pos := g.WorldPosition()
			radius := r.getBoundingRadius(g, mr)
			if !r.frustum.ContainsSphere(pos, radius) {
				r.CulledObjects++
				continue
			}
		}
		r.DrawnObjects++

		// Only batch generated meshes (sphere, cube, plane) - file models render individually
		// Also skip batching if mesh has custom size (like the floor) since mesh geometry differs
		if mr.MeshType == "" || len(mr.MeshSize) > 0 {
			mr.Draw()
			continue
		}

		// Create batch key from mesh type + color
		key := mr.MeshType + colorKey(mr.Color)

		batch, exists := batches[key]
		if !exists {
			// Get mesh and material from the model
			mesh := mr.Model.GetMeshes()[0]
			material := mr.Model.GetMaterials()[0]
			material.Shader = r.Shader

			batch = &instanceBatch{
				mesh:     mesh,
				material: material,
				color:    mr.Color,
			}
			batches[key] = batch
		}

		// Build transform matrix for this instance
		scale := g.WorldScale()
		scaleMatrix := rl.MatrixScale(scale.X, scale.Y, scale.Z)

		rot := g.WorldRotation()
		rotX := rl.MatrixRotateX(rot.X * rl.Deg2rad)
		rotY := rl.MatrixRotateY(rot.Y * rl.Deg2rad)
		rotZ := rl.MatrixRotateZ(rot.Z * rl.Deg2rad)
		rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)

		pos := g.WorldPosition()
		transMatrix := rl.MatrixTranslate(pos.X, pos.Y, pos.Z)

		transform := rl.MatrixMultiply(rl.MatrixMultiply(scaleMatrix, rotMatrix), transMatrix)
		batch.transforms = append(batch.transforms, transform)
	}

	// Draw all batches with instanced rendering
	for _, batch := range batches {
		if len(batch.transforms) == 0 {
			continue
		}

		// Use instancing shader for batched meshes
		batch.material.Shader = r.InstanceShader

		// Set material color
		batch.material.Maps.Color = batch.color

		// Set default material uniforms for instanced objects
		metallicLoc := rl.GetShaderLocation(r.InstanceShader, "metallic")
		roughnessLoc := rl.GetShaderLocation(r.InstanceShader, "roughness")
		emissiveLoc := rl.GetShaderLocation(r.InstanceShader, "emissive")
		rl.SetShaderValue(r.InstanceShader, metallicLoc, []float32{0.0}, rl.ShaderUniformFloat)
		rl.SetShaderValue(r.InstanceShader, roughnessLoc, []float32{0.5}, rl.ShaderUniformFloat)
		rl.SetShaderValue(r.InstanceShader, emissiveLoc, []float32{0.0}, rl.ShaderUniformFloat)

		rl.DrawMeshInstanced(batch.mesh, batch.material, batch.transforms, len(batch.transforms))
	}
}

// colorKey returns a string key for a color (for batching by color)
func colorKey(c rl.Color) string {
	return string([]byte{c.R, c.G, c.B, c.A})
}

// getBoundingRadius returns a conservative bounding sphere radius for culling
func (r *Renderer) getBoundingRadius(g *engine.GameObject, mr *components.ModelRenderer) float32 {
	scale := g.WorldScale()
	// Use the largest scale axis for a conservative bounding sphere
	maxScale := scale.X
	if scale.Y > maxScale {
		maxScale = scale.Y
	}
	if scale.Z > maxScale {
		maxScale = scale.Z
	}

	// Check for custom mesh size (e.g., floor plane with meshSize [60, 60])
	if len(mr.MeshSize) >= 2 {
		// MeshSize defines the actual mesh dimensions
		meshMax := mr.MeshSize[0]
		for _, s := range mr.MeshSize[1:] {
			if s > meshMax {
				meshMax = s
			}
		}
		// Diagonal of the mesh (sqrt(2) for 2D, sqrt(3) for 3D)
		return meshMax * 0.707 * maxScale // half-diagonal for radius
	}

	// Base radius depends on mesh type (unit primitives)
	var baseRadius float32 = 1.0
	switch mr.MeshType {
	case "sphere":
		baseRadius = 1.0 // unit sphere
	case "cube":
		baseRadius = 1.732 // diagonal of unit cube (sqrt(3))
	case "plane":
		baseRadius = 1.414 // diagonal of unit plane (sqrt(2))
	default:
		// For file models, use a generous estimate
		baseRadius = 2.0
	}

	return baseRadius * maxScale
}

func (r *Renderer) updatePointLights(gameObjects []*engine.GameObject) {
	var positions []float32
	var colors []float32
	var radii []float32
	count := 0

	for _, g := range gameObjects {
		if count >= MaxPointLights {
			break
		}
		if pl := engine.GetComponent[*components.PointLight](g); pl != nil {
			pos := pl.GetPosition()
			positions = append(positions, pos.X, pos.Y, pos.Z)
			colors = append(colors, pl.GetColorFloat()...)
			radii = append(radii, pl.Radius)
			count++
		}
	}

	// Pad arrays to MaxPointLights size (shader expects fixed-size arrays)
	for i := count; i < MaxPointLights; i++ {
		positions = append(positions, 0, 0, 0)
		colors = append(colors, 0, 0, 0)
		radii = append(radii, 0)
	}

	// Update both shaders
	for _, shader := range []rl.Shader{r.Shader, r.InstanceShader} {
		countLoc := rl.GetShaderLocation(shader, "pointLightCount")
		rl.SetUniform(countLoc, []int32{int32(count)}, int32(rl.ShaderUniformInt), 1)

		posLoc := rl.GetShaderLocation(shader, "pointLightPos")
		rl.SetShaderValue(shader, posLoc, positions, rl.ShaderUniformVec3)

		colorLoc := rl.GetShaderLocation(shader, "pointLightColor")
		rl.SetShaderValue(shader, colorLoc, colors, rl.ShaderUniformVec3)

		radiusLoc := rl.GetShaderLocation(shader, "pointLightRadius")
		rl.SetShaderValue(shader, radiusLoc, radii, rl.ShaderUniformFloat)
	}
}

func (r *Renderer) MoveLightDir(dx, dy, dz float32) {
	if r.Light == nil {
		return
	}
	r.Light.MoveLightDir(dx, dy, dz)
	r.updateLightCamera()

	lightDir := []float32{r.Light.Direction.X, r.Light.Direction.Y, r.Light.Direction.Z}
	for _, shader := range []rl.Shader{r.Shader, r.InstanceShader} {
		lightDirLoc := rl.GetShaderLocation(shader, "lightDir")
		rl.SetShaderValue(shader, lightDirLoc, lightDir, rl.ShaderUniformVec3)
	}
}

func (r *Renderer) Unload(gameObjects []*engine.GameObject) {
	rl.UnloadShader(r.Shader)
	rl.UnloadShader(r.InstanceShader)
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
