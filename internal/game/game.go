package game

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/physics"
	"test3d/internal/world"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Game struct {
	Player      *engine.GameObject
	World       *world.World
	DebugMode   bool
	cubeCounter int

	// Debug timing (ms)
	updateMs  float64
	shadowMs  float64
	drawMs    float64

	lastShotTime float64
}

func New() *Game {
	return &Game{
		World:     world.New(),
		DebugMode: false,
	}
}

func (g *Game) Run() {
	rl.SetConfigFlags(rl.FlagWindowHighdpi)
	rl.InitWindow(1280, 720, "3D Animated Cubes with Lighting")
	defer rl.CloseWindow()

	rl.SetTargetFPS(120)
	rl.DisableCursor()

	// Initialize world after OpenGL context is created
	g.World.Initialize()
	defer g.World.Unload()

	// Create player GameObject
	g.createPlayer()

	for !rl.WindowShouldClose() {
		g.Update()
		g.Draw()
	}
}

func (g *Game) createPlayer() {
	g.Player = engine.NewGameObject("Player")
	g.Player.Transform.Position = rl.Vector3{X: 10, Y: 10, Z: 10}

	// Add FPS controller
	fps := components.NewFPSController()
	g.Player.AddComponent(fps)

	// Add camera
	cam := components.NewCamera()
	g.Player.AddComponent(cam)

	// Add collider for player body
	collider := components.NewBoxCollider(rl.Vector3{X: 0.6, Y: 1.8, Z: 0.6})
	g.Player.AddComponent(collider)

	// Add kinematic rigidbody so player can push things
	rb := components.NewRigidbody()
	rb.IsKinematic = true
	rb.UseGravity = false // FPSController handles gravity
	g.Player.AddComponent(rb)

	// Add to physics world
	g.World.PhysicsWorld.AddObject(g.Player)

	g.Player.Start()
}

func (g *Game) Update() {
	updateStart := time.Now()
	deltaTime := rl.GetFrameTime()

	// Update player
	g.Player.Update(deltaTime)

	// Get player components
	fps := engine.GetComponent[*components.FPSController](g.Player)
	collider := engine.GetComponent[*components.BoxCollider](g.Player)
	playerRb := engine.GetComponent[*components.Rigidbody](g.Player)

	if fps == nil || collider == nil {
		return
	}

	// Sync FPSController velocity to rigidbody for physics pushing
	if playerRb != nil {
		playerRb.Velocity = fps.Velocity
	}

	// Ground check - floor is at Y=0
	floorY := float32(0.0)
	feetY := g.Player.Transform.Position.Y - fps.EyeHeight
	if feetY <= floorY {
		g.Player.Transform.Position.Y = floorY + fps.EyeHeight
		fps.Velocity.Y = 0
		fps.Grounded = true
	} else {
		fps.Grounded = false
	}

	// Collision with world objects
	// Build AABB with eye height offset
	playerAABB := physics.NewAABBFromCenter(
		rl.Vector3{
			X: g.Player.Transform.Position.X,
			Y: g.Player.Transform.Position.Y - fps.EyeHeight + collider.Size.Y/2,
			Z: g.Player.Transform.Position.Z,
		},
		collider.Size,
	)

	for _, obj := range g.World.GetCollidableObjects() {
		objCollider := engine.GetComponent[*components.BoxCollider](obj)
		if objCollider == nil {
			continue
		}

		objAABB := physics.NewAABBFromCenter(obj.Transform.Position, objCollider.Size)
		pushOut := playerAABB.Resolve(objAABB)

		if pushOut.X != 0 || pushOut.Y != 0 || pushOut.Z != 0 {
			g.Player.Transform.Position = rl.Vector3Add(g.Player.Transform.Position, pushOut)

			if pushOut.Y > 0 {
				fps.Velocity.Y = 0
				fps.Grounded = true
			}
			if pushOut.Y < 0 && fps.Velocity.Y > 0 {
				fps.Velocity.Y = 0
			}

			// Update AABB for subsequent checks
			playerAABB = physics.NewAABBFromCenter(
				rl.Vector3{
					X: g.Player.Transform.Position.X,
					Y: g.Player.Transform.Position.Y - fps.EyeHeight + collider.Size.Y/2,
					Z: g.Player.Transform.Position.Z,
				},
				collider.Size,
			)
		}
	}

	// Update world
	g.World.Update(deltaTime)

	// Toggle debug mode
	if rl.IsKeyPressed(rl.KeyF1) {
		g.DebugMode = !g.DebugMode
	}

	// Shoot sphere with left mouse button (with cooldown)
	const shootCooldown = 0.15
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.GetTime()-g.lastShotTime >= shootCooldown {
		g.ShootSphere(fps)
		g.lastShotTime = rl.GetTime()
	}

	// Light controls
	lightSpeed := float32(1.0) * deltaTime
	if rl.IsKeyDown(rl.KeyLeft) {
		g.World.Renderer.MoveLightDir(-lightSpeed, 0, 0)
	}
	if rl.IsKeyDown(rl.KeyRight) {
		g.World.Renderer.MoveLightDir(lightSpeed, 0, 0)
	}
	if rl.IsKeyDown(rl.KeyUp) {
		g.World.Renderer.MoveLightDir(0, 0, lightSpeed)
	}
	if rl.IsKeyDown(rl.KeyDown) {
		g.World.Renderer.MoveLightDir(0, 0, -lightSpeed)
	}
	if rl.IsKeyDown(rl.KeyQ) {
		g.World.Renderer.MoveLightDir(0, -lightSpeed, 0)
	}
	if rl.IsKeyDown(rl.KeyE) {
		g.World.Renderer.MoveLightDir(0, lightSpeed, 0)
	}

	g.updateMs = float64(time.Since(updateStart).Microseconds()) / 1000.0
}

func (g *Game) Draw() {
	cam := engine.GetComponent[*components.Camera](g.Player)
	if cam == nil {
		return
	}

	camera := cam.GetRaylibCamera()

	// Shadow pass
	shadowStart := time.Now()
	g.World.Renderer.DrawShadowMap(g.World.Scene.GameObjects)
	g.shadowMs = float64(time.Since(shadowStart).Microseconds()) / 1000.0

	// Main render
	rl.BeginDrawing()
	rl.ClearBackground(rl.NewColor(20, 20, 30, 255))

	drawStart := time.Now()
	rl.BeginMode3D(camera)
	g.World.Renderer.DrawWithShadows(camera.Position, g.World.Scene.GameObjects)
	rl.EndMode3D()
	g.drawMs = float64(time.Since(drawStart).Microseconds()) / 1000.0

	g.DrawUI()
	rl.EndDrawing()
}

func (g *Game) DrawUI() {
	rl.DrawText("WASD to move, Space to jump, Mouse to look", 10, 10, 20, rl.DarkGray)
	rl.DrawText("F1 to toggle debug view", 10, 35, 20, rl.DarkGray)
	rl.DrawFPS(10, 60)

	if g.DebugMode {
		previewSize := int32(256)
		screenW := int32(rl.GetScreenWidth())
		rl.DrawTexturePro(
			g.World.Renderer.ShadowMap.Depth,
			rl.Rectangle{X: 0, Y: 0, Width: float32(g.World.Renderer.ShadowMap.Depth.Width), Height: float32(-g.World.Renderer.ShadowMap.Depth.Height)},
			rl.Rectangle{X: float32(screenW - previewSize - 10), Y: 10, Width: float32(previewSize), Height: float32(previewSize)},
			rl.Vector2{X: 0, Y: 0},
			0,
			rl.White,
		)
		rl.DrawRectangleLines(screenW-previewSize-10, 10, previewSize, previewSize, rl.Green)
		rl.DrawText("Shadow Map", screenW-previewSize-10, previewSize+15, 16, rl.Green)

		lightDir := g.World.Renderer.LightDir
		rl.DrawText(fmt.Sprintf("Light Dir: (%.2f, %.2f, %.2f)", lightDir.X, lightDir.Y, lightDir.Z), 10, 85, 16, rl.Yellow)

		rl.DrawText(fmt.Sprintf("Update:  %.2f ms", g.updateMs), 10, 110, 16, rl.Green)
		rl.DrawText(fmt.Sprintf("Shadows: %.2f ms", g.shadowMs), 10, 130, 16, rl.Green)
		rl.DrawText(fmt.Sprintf("Draw:    %.2f ms", g.drawMs), 10, 150, 16, rl.Green)
		rl.DrawText(fmt.Sprintf("Total:   %.2f ms", g.updateMs+g.shadowMs+g.drawMs), 10, 170, 16, rl.Lime)
	}
}

func (g *Game) ShootSphere(fps *components.FPSController) {
	g.cubeCounter++

	// Spawn position: in front of player
	lookDir := fps.GetLookDirection()
	spawnPos := rl.Vector3Add(g.Player.Transform.Position, rl.Vector3Scale(lookDir, 3))

	radius := float32(0.5)

	// Create sphere GameObject
	sphere := engine.NewGameObject(fmt.Sprintf("Shot_%d", g.cubeCounter))
	sphere.Transform.Position = spawnPos

	// Create sphere model and renderer
	mesh := rl.GenMeshSphere(radius, 16, 16)
	model := rl.LoadModelFromMesh(mesh)
	renderer := components.NewModelRenderer(model, rl.Orange)
	renderer.SetShader(g.World.Renderer.Shader)
	sphere.AddComponent(renderer)

	// Add sphere collider
	sphere.AddComponent(components.NewSphereCollider(radius))

	// Add rigidbody with initial velocity in look direction
	rb := components.NewRigidbody()
	rb.Bounciness = 0.6
	rb.Friction = 0.1
	rb.Velocity = rl.Vector3Scale(lookDir, 30) // yeet it
	sphere.AddComponent(rb)

	sphere.Start()
	g.World.Scene.AddGameObject(sphere)
	g.World.PhysicsWorld.AddObject(sphere)
}
