package game

import (
	"test3d/internal/camera"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Game struct {
	Camera *camera.FPSCamera
	World  *world.World
}

func New() *Game {
	return &Game{
		Camera: camera.New(rl.Vector3{X: 10, Y: 10, Z: 10}),
		World:  world.New(),
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
	g.World.Update(deltaTime)
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
	rl.DrawText("WASD to move, Mouse to look, Space/Shift for up/down", 10, 10, 20, rl.DarkGray)
	rl.DrawFPS(10, 40)
}
