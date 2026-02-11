package game

import (
	"fmt"
	"time"

	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Game struct {
	World     *world.World
	editor    *Editor
	DebugMode bool

	// Debug timing (ms)
	updateMs float64
	shadowMs float64
	drawMs   float64
}

func New() *Game {
	return &Game{
		World:     world.New(),
		DebugMode: false,
	}
}

func (g *Game) Run(restoreEditor bool) {
	// Load editor preferences before creating window
	prefs := LoadEditorPrefs()

	rl.SetConfigFlags(rl.FlagWindowHighdpi | rl.FlagWindowResizable)

	// Use saved window size or default
	windowW, windowH := 1280, 720
	if prefs != nil && prefs.WindowWidth > 0 && prefs.WindowHeight > 0 {
		windowW = prefs.WindowWidth
		windowH = prefs.WindowHeight
	}
	rl.InitWindow(int32(windowW), int32(windowH), "Mirgo Engine")

	// Restore window position if we have valid prefs
	if prefs != nil && (prefs.WindowX > 0 || prefs.WindowY > 0) {
		rl.SetWindowPosition(prefs.WindowX, prefs.WindowY)
	}

	rl.SetTargetFPS(120)

	// Load the scene from prefs if available
	if prefs != nil && prefs.ScenePath != "" {
		world.ScenePath = prefs.ScenePath
	}

	// Initialize world after OpenGL context is created
	g.World.Initialize()
	defer g.World.Unload()

	g.editor = NewEditor(g.World)

	// Start in editor mode by default
	cam := g.World.FindMainCamera()
	if cam != nil {
		g.editor.Enter(cam.GetRaylibCamera())
	} else {
		// No camera in scene, start with default editor camera
		g.editor.Enter(rl.Camera3D{
			Position:   rl.Vector3{X: 10, Y: 10, Z: 10},
			Target:     rl.Vector3{},
			Up:         rl.Vector3{Y: 1},
			Fovy:       45,
			Projection: rl.CameraPerspective,
		})
	}

	// Apply editor preferences (camera position, panel sizes, etc.)
	if prefs != nil && !restoreEditor {
		g.editor.ApplyPrefs(prefs)
	}

	// Restore editor state if relaunching after hot-reload
	if restoreEditor {
		g.editor.RestoreState()
	}

	for !rl.WindowShouldClose() {
		g.Update()
		g.Draw()
	}

	// Save editor preferences before closing
	g.editor.SavePrefs()
	rl.CloseWindow()
}

func (g *Game) Update() {
	updateStart := time.Now()
	deltaTime := rl.GetFrameTime()

	// Mode toggles (always active)
	if rl.IsKeyPressed(rl.KeyF1) {
		g.DebugMode = !g.DebugMode
	}

	// Check for modifier keys
	cmdOrCtrl := rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper) || rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

	// Cmd/Ctrl+Shift+P to pause/unpause (preserves scene state)
	if rl.IsKeyPressed(rl.KeyP) && cmdOrCtrl && shift {
		if g.editor.Active {
			// Resume game from pause
			g.editor.Exit()
		} else {
			// Pause game (enter editor without reset)
			cam := g.World.FindMainCamera()
			if cam != nil {
				g.editor.Pause(cam.GetRaylibCamera())
			}
		}
	} else if rl.IsKeyPressed(rl.KeyP) && cmdOrCtrl && !shift {
		// Cmd/Ctrl+P to toggle play mode (resets scene when entering editor)
		if g.editor.Active {
			// Enter game mode
			g.editor.Exit()
		} else {
			// Return to editor (resets scene)
			cam := g.World.FindMainCamera()
			if cam != nil {
				g.editor.Enter(cam.GetRaylibCamera())
			}
		}
	}

	if g.editor.Active {
		g.editor.Update(deltaTime)
		g.updateMs = float64(time.Since(updateStart).Microseconds()) / 1000.0
		return
	}

	// Update world (physics + all game objects including player)
	g.World.Update(deltaTime)

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
	// Get camera based on mode
	var camera rl.Camera3D
	if g.editor.Active {
		camera = g.editor.GetRaylibCamera()
	} else {
		cam := g.World.FindMainCamera()
		if cam == nil {
			// No camera, just draw editor UI
			rl.BeginDrawing()
			rl.ClearBackground(rl.NewColor(20, 20, 30, 255))
			rl.DrawText("No Camera in scene! Add a Camera component to a GameObject.", 10, 10, 20, rl.Red)
			rl.EndDrawing()
			return
		}
		camera = cam.GetRaylibCamera()
	}

	// Check if in UI edit mode - skip 3D rendering, draw 2D UI instead
	uiEditMode := g.editor.Active && g.editor.IsUIEditModeActive()

	if !uiEditMode {
		// Shadow pass (only in 3D mode)
		shadowStart := time.Now()
		g.World.Renderer.DrawShadowMap(g.World.Scene.GameObjects)
		g.shadowMs = float64(time.Since(shadowStart).Microseconds()) / 1000.0
	}

	// Main render
	rl.BeginDrawing()
	rl.ClearBackground(rl.NewColor(20, 20, 30, 255))

	if uiEditMode {
		// Draw 2D UI editor view instead of 3D scene
		g.editor.Draw3DForUIMode()
	} else {
		// Normal 3D rendering
		drawStart := time.Now()
		rl.BeginMode3D(camera)
		g.World.Renderer.DrawWithShadows(camera, g.World.Scene.GameObjects)
		if g.editor.Active {
			g.editor.Draw3D()
		}
		rl.EndMode3D()
		g.drawMs = float64(time.Since(drawStart).Microseconds()) / 1000.0
	}

	// Draw editor UI (panels, etc) - same for both modes
	if g.editor.Active {
		g.editor.DrawUI()
		// Draw UI edit mode overlay (2D badge, cursor handling)
		g.editor.DrawUIEditModeOverlay()
	} else {
		g.DrawUI()
	}
	rl.EndDrawing()
}

func (g *Game) DrawUI() {
	// Draw UI canvases from the scene
	for _, obj := range g.World.Scene.GameObjects {
		if canvas := engine.GetComponent[*components.UICanvas](obj); canvas != nil {
			canvas.Draw()
		}
	}

	rl.DrawText("WASD to move, Space to jump, Mouse to look", 10, 10, 20, rl.DarkGray)
	rl.DrawText("F1: debug | P: pause | Cmd+P: editor (reset)", 10, 35, 20, rl.DarkGray)
	rl.DrawFPS(10, 60)

	// Crosshair
	cx := int32(rl.GetScreenWidth() / 2)
	cy := int32(rl.GetScreenHeight() / 2)
	size := int32(10)
	rl.DrawLine(cx-size, cy, cx+size, cy, rl.White)
	rl.DrawLine(cx, cy-size, cx, cy+size, rl.White)

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

		if light := g.World.Renderer.Light; light != nil {
			rl.DrawText(fmt.Sprintf("Light Dir: (%.2f, %.2f, %.2f)", light.Direction.X, light.Direction.Y, light.Direction.Z), 10, 85, 16, rl.Yellow)
		}

		rl.DrawText(fmt.Sprintf("Update:  %.2f ms", g.updateMs), 10, 110, 16, rl.Green)
		rl.DrawText(fmt.Sprintf("Shadows: %.2f ms", g.shadowMs), 10, 130, 16, rl.Green)
		rl.DrawText(fmt.Sprintf("Draw:    %.2f ms", g.drawMs), 10, 150, 16, rl.Green)
		rl.DrawText(fmt.Sprintf("Total:   %.2f ms", g.updateMs+g.shadowMs+g.drawMs), 10, 170, 16, rl.Lime)

		// Frustum culling stats
		drawn := g.World.Renderer.DrawnObjects
		culled := g.World.Renderer.CulledObjects
		total := drawn + culled
		rl.DrawText(fmt.Sprintf("Drawn: %d / %d (culled: %d)", drawn, total, culled), 10, 195, 16, rl.SkyBlue)
	}
}
