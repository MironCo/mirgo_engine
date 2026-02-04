package world

import rl "github.com/gen2brain/raylib-go/raylib"

type World struct {
	Objects []Object
}

type Object struct {
	Position rl.Vector3
	Size     rl.Vector3
	Color    rl.Color
}

func New() *World {
	return &World{
		Objects: []Object{
			{
				Position: rl.Vector3{X: 0, Y: 1, Z: 0},
				Size:     rl.Vector3{X: 2, Y: 2, Z: 2},
				Color:    rl.Red,
			},
		},
	}
}

func (w *World) AddCube(pos, size rl.Vector3, color rl.Color) {
	w.Objects = append(w.Objects, Object{
		Position: pos,
		Size:     size,
		Color:    color,
	})
}

func (w *World) Draw() {
	for _, obj := range w.Objects {
		rl.DrawCube(obj.Position, obj.Size.X, obj.Size.Y, obj.Size.Z, obj.Color)
		rl.DrawCubeWires(obj.Position, obj.Size.X, obj.Size.Y, obj.Size.Z, rl.Black)
	}
	rl.DrawGrid(10, 1)
}
