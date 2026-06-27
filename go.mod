module mirgo_engine

go 1.25

require (
	github.com/cogentcore/webgpu v0.23.0
	github.com/ebitengine/oto/v3 v3.4.0
	github.com/gen2brain/raylib-go/raygui v0.0.0-20260112154027-eddf36910f39
	github.com/gen2brain/raylib-go/raylib v0.55.1
)

require (
	github.com/ebitengine/purego v0.9.0 // indirect
	golang.org/x/exp v0.0.0-20241108190413-2d47ceb2692f // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/gen2brain/raylib-go/raylib => ./raylib-go/raylib
