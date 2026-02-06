package components

import (
	"math"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type CubeAnimator struct {
	engine.BaseComponent
	StartPosition   rl.Vector3
	RotationAxis    rl.Vector3
	RotationSpeed   float32
	CurrentRotation float32
	MovementRadius  float32
	MovementSpeed   float32
	Phase           float32
	time            float32
}

func NewCubeAnimator(startPos rl.Vector3, rotAxis rl.Vector3, rotSpeed, moveRadius, moveSpeed, phase float32) *CubeAnimator {
	return &CubeAnimator{
		StartPosition:  startPos,
		RotationAxis:   rotAxis,
		RotationSpeed:  rotSpeed,
		MovementRadius: moveRadius,
		MovementSpeed:  moveSpeed,
		Phase:          phase,
	}
}

func (c *CubeAnimator) Update(deltaTime float32) {
	g := c.GetGameObject()
	if g == nil {
		return
	}

	c.time += deltaTime

	t := c.time*c.MovementSpeed + c.Phase
	offset := rl.Vector3{
		X: float32(math.Cos(float64(t))) * c.MovementRadius,
		Y: float32(math.Sin(float64(t*2))) * 1.5,
		Z: float32(math.Sin(float64(t))) * c.MovementRadius,
	}

	g.Transform.Position = rl.Vector3Add(c.StartPosition, offset)

	c.CurrentRotation += c.RotationSpeed * deltaTime
	if c.CurrentRotation > 360 {
		c.CurrentRotation -= 360
	}
	g.Transform.Rotation.Y = c.CurrentRotation
}
