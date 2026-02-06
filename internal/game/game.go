package game

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/physics"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Game struct {
	Player    *engine.GameObject
	World     *world.World
	DebugMode bool
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

	rl.SetTargetFPS(60)
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

	g.Player.Start()
}

func (g *Game) Update() {
	deltaTime := rl.GetFrameTime()

	// Update player
	g.Player.Update(deltaTime)

	// Get player components
	fps := engine.GetComponent[*components.FPSController](g.Player)
	collider := engine.GetComponent[*components.BoxCollider](g.Player)

	if fps == nil || collider == nil {
		return
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
	playerAABB := collider.GetAABB()
	// Adjust AABB center for eye height offset
	playerAABB = physics.NewAABBFromCenter(
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

		objAABB := objCollider.GetAABB()
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

	// Light controls
	lightSpeed := float32(1.0) * deltaTime
	if rl.IsKeyDown(rl.KeyLeft) {
		g.World.MoveLightDir(-lightSpeed, 0, 0)
	}
	if rl.IsKeyDown(rl.KeyRight) {
		g.World.MoveLightDir(lightSpeed, 0, 0)
	}
	if rl.IsKeyDown(rl.KeyUp) {
		g.World.MoveLightDir(0, 0, lightSpeed)
	}
	if rl.IsKeyDown(rl.KeyDown) {
		g.World.MoveLightDir(0, 0, -lightSpeed)
	}
	if rl.IsKeyDown(rl.KeyQ) {
		g.World.MoveLightDir(0, -lightSpeed, 0)
	}
	if rl.IsKeyDown(rl.KeyE) {
		g.World.MoveLightDir(0, lightSpeed, 0)
	}
}

func (g *Game) Draw() {
	cam := engine.GetComponent[*components.Camera](g.Player)
	if cam == nil {
		return
	}

	camera := cam.GetRaylibCamera()

	// Shadow pass
	g.World.DrawShadowMap()

	// Main render
	rl.BeginDrawing()
	rl.ClearBackground(rl.NewColor(20, 20, 30, 255))

	rl.BeginMode3D(camera)
	g.World.DrawWithShadows(camera.Position)
	rl.EndMode3D()

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
			g.World.ShadowMap.Depth,
			rl.Rectangle{X: 0, Y: 0, Width: float32(g.World.ShadowMap.Depth.Width), Height: float32(-g.World.ShadowMap.Depth.Height)},
			rl.Rectangle{X: float32(screenW - previewSize - 10), Y: 10, Width: float32(previewSize), Height: float32(previewSize)},
			rl.Vector2{X: 0, Y: 0},
			0,
			rl.White,
		)
		rl.DrawRectangleLines(screenW-previewSize-10, 10, previewSize, previewSize, rl.Green)
		rl.DrawText("Shadow Map", screenW-previewSize-10, previewSize+15, 16, rl.Green)

		lightDir := g.World.LightDir
		rl.DrawText(fmt.Sprintf("Light Dir: (%.2f, %.2f, %.2f)", lightDir.X, lightDir.Y, lightDir.Z), 10, 85, 16, rl.Yellow)
	}
}
