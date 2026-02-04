package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"test3d/internal/camera"
	"test3d/internal/world"
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
	rl.InitWindow(800, 600, "3D Game")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.DisableCursor()

	for !rl.WindowShouldClose() {
		g.Update()
		g.Draw()
	}
}

func (g *Game) Update() {
	g.Camera.Update()
}

func (g *Game) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)

	rl.BeginMode3D(g.Camera.GetRaylibCamera())
	g.World.Draw()
	rl.EndMode3D()

	g.DrawUI()
	rl.EndDrawing()
}

func (g *Game) DrawUI() {
	rl.DrawText("WASD to move, Mouse to look, Space/Shift for up/down", 10, 10, 20, rl.DarkGray)
	rl.DrawFPS(10, 40)
}
