package game

import (
	"fmt"
	"test3d/internal/camera"
	"test3d/internal/physics"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Game struct {
	Camera    *camera.FPSCamera
	World     *world.World
	DebugMode bool
}

func New() *Game {
	return &Game{
		Camera:    camera.New(rl.Vector3{X: 10, Y: 10, Z: 10}),
		World:     world.New(),
		DebugMode: false,
	}
}

func (g *Game) Run() {
	rl.InitWindow(1280, 720, "3D Animated Cubes with Lighting")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.DisableCursor()

	// Initialize world after OpenGL context is created
	g.World.Initialize()
	defer g.World.Unload()

	for !rl.WindowShouldClose() {
		g.Update()
		g.Draw()
	}
}

func (g *Game) Update() {
	deltaTime := rl.GetFrameTime()
	g.Camera.Update()

	// Resolve player collision against world objects
	playerSize := rl.Vector3{X: 0.6, Y: 1.8, Z: 0.6}
	playerAABB := physics.NewAABBFromCenter(g.Camera.Position, playerSize)
	for _, obj := range g.World.Objects {
		objAABB := physics.NewAABBFromCenter(obj.Position, obj.Size)
		pushOut := playerAABB.Resolve(objAABB)
		if pushOut.X != 0 || pushOut.Y != 0 || pushOut.Z != 0 {
			g.Camera.Position = rl.Vector3Add(g.Camera.Position, pushOut)
			playerAABB = physics.NewAABBFromCenter(g.Camera.Position, playerSize)
		}
	}

	g.World.Update(deltaTime)

	// Toggle debug mode with F1
	if rl.IsKeyPressed(rl.KeyF1) {
		g.DebugMode = !g.DebugMode
	}

	// Move light with arrow keys + Q/E for up/down
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
		g.World.MoveLightDir(0, -lightSpeed, 0) // Light points more down
	}
	if rl.IsKeyDown(rl.KeyE) {
		g.World.MoveLightDir(0, lightSpeed, 0) // Light points more up
	}
}

func (g *Game) Draw() {
	camera := g.Camera.GetRaylibCamera()

	// First: render shadow map (before BeginDrawing to avoid state conflicts)
	g.World.DrawShadowMap()

	// Second: main scene rendering
	rl.BeginDrawing()
	rl.ClearBackground(rl.NewColor(20, 20, 30, 255))

	rl.BeginMode3D(camera)
	g.World.DrawWithShadows(camera.Position)
	rl.EndMode3D()

	g.DrawUI()
	rl.EndDrawing()
}

func (g *Game) DrawUI() {
	rl.DrawText("WASD to move, Mouse to look, Arrows+Q/E to move light", 10, 10, 20, rl.DarkGray)
	rl.DrawText("F1 to toggle debug view", 10, 35, 20, rl.DarkGray)
	rl.DrawFPS(10, 60)

	// Debug: draw shadowmap preview
	if g.DebugMode {
		previewSize := int32(256)
		screenW := int32(rl.GetScreenWidth())
		// Draw shadowmap depth texture as preview in corner
		rl.DrawTexturePro(
			g.World.ShadowMap.Depth,
			rl.Rectangle{X: 0, Y: 0, Width: float32(g.World.ShadowMap.Depth.Width), Height: float32(-g.World.ShadowMap.Depth.Height)}, // Flip Y
			rl.Rectangle{X: float32(screenW - previewSize - 10), Y: 10, Width: float32(previewSize), Height: float32(previewSize)},
			rl.Vector2{X: 0, Y: 0},
			0,
			rl.White,
		)
		rl.DrawRectangleLines(screenW-previewSize-10, 10, previewSize, previewSize, rl.Green)
		rl.DrawText("Shadow Map", screenW-previewSize-10, previewSize+15, 16, rl.Green)

		// Show light direction
		lightDir := g.World.LightDir
		rl.DrawText(fmt.Sprintf("Light Dir: (%.2f, %.2f, %.2f)", lightDir.X, lightDir.Y, lightDir.Z), 10, 85, 16, rl.Yellow)
	}
}
