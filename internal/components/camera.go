package components

import (
	"math"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("Camera", func() engine.Serializable {
		return NewCamera()
	})
}

type Camera struct {
	engine.BaseComponent
	FOV        float32
	Near       float32
	Far        float32
	Projection rl.CameraProjection
	IsMain     bool // If true, this is the active game camera
}

func NewCamera() *Camera {
	return &Camera{
		FOV:        45.0,
		Near:       0.1,
		Far:        1000.0,
		Projection: rl.CameraPerspective,
		IsMain:     false,
	}
}

// TypeName implements engine.Serializable
func (c *Camera) TypeName() string {
	return "Camera"
}

// Serialize implements engine.Serializable
func (c *Camera) Serialize() map[string]any {
	return map[string]any{
		"type":   "Camera",
		"fov":    c.FOV,
		"near":   c.Near,
		"far":    c.Far,
		"isMain": c.IsMain,
	}
}

// Deserialize implements engine.Serializable
func (c *Camera) Deserialize(data map[string]any) {
	if f, ok := data["fov"].(float64); ok {
		c.FOV = float32(f)
	}
	if n, ok := data["near"].(float64); ok {
		c.Near = float32(n)
	}
	if f, ok := data["far"].(float64); ok {
		c.Far = float32(f)
	}
	if m, ok := data["isMain"].(bool); ok {
		c.IsMain = m
	}
}

func (c *Camera) GetRaylibCamera() rl.Camera3D {
	g := c.GetGameObject()
	if g == nil {
		return rl.Camera3D{}
	}

	// Get eye position from world position
	eyePos := g.WorldPosition()

	// Look for any LookProvider component on this object or parents
	var lookProvider engine.LookProvider
	for obj := g; obj != nil; obj = obj.Parent {
		if lp := engine.FindComponent[engine.LookProvider](obj); lp != nil {
			lookProvider = lp
			break
		}
	}

	// If camera is a child, use parent's position + eye height offset
	// If camera is on the same object as LookProvider, add eye height
	if lookProvider != nil {
		if g.Parent != nil {
			// Camera is a child - use parent position + local offset
			eyePos = g.WorldPosition()
		} else {
			// Camera is on same object as controller - add eye height
			eyePos.Y += lookProvider.GetEyeHeight()
		}
	}

	var target rl.Vector3
	if lookProvider != nil {
		x, y, z := lookProvider.GetLookDirection()
		lookDir := rl.Vector3{X: x, Y: y, Z: z}
		target = rl.Vector3Add(eyePos, lookDir)
	} else {
		// Default: look forward based on object's rotation
		rot := g.WorldRotation()
		// Convert euler to forward vector (simplified - just yaw for now)
		yawRad := float64(rot.Y) * 3.14159265 / 180.0
		forward := rl.Vector3{
			X: float32(-sin(yawRad)),
			Y: 0,
			Z: float32(-cos(yawRad)),
		}
		target = rl.Vector3Add(eyePos, forward)
	}

	return rl.Camera3D{
		Position:   eyePos,
		Target:     target,
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       c.FOV,
		Projection: c.Projection,
	}
}

func sin(x float64) float64 {
	return math.Sin(x)
}

func cos(x float64) float64 {
	return math.Cos(x)
}
