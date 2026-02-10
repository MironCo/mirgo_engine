package components

import (
	"math"
	"test3d/internal/audio"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("AudioListener", func() engine.Serializable {
		return NewAudioListener()
	})
}

type AudioListener struct {
	engine.BaseComponent
}

func NewAudioListener() *AudioListener {
	return &AudioListener{}
}

func (a *AudioListener) TypeName() string {
	return "AudioListener"
}

func (a *AudioListener) Serialize() map[string]any {
	return map[string]any{
		"type": "AudioListener",
	}
}

func (a *AudioListener) Deserialize(data map[string]any) {
	// No fields to deserialize
}

func (a *AudioListener) Update(deltaTime float32) {
	g := a.GetGameObject()
	if g == nil {
		return
	}

	pos := g.WorldPosition()
	up := rl.Vector3{X: 0, Y: 1, Z: 0}

	var forward rl.Vector3

	// Try to get look direction from a LookProvider (like FPSController)
	if lookProvider := engine.FindComponent[engine.LookProvider](g); lookProvider != nil {
		x, y, z := lookProvider.GetLookDirection()
		forward = rl.Vector3{X: x, Y: y, Z: z}
	} else {
		// Fallback to rotation-based forward
		rot := g.WorldRotation()
		yaw := float64(rot.Y) * math.Pi / 180
		pitch := float64(rot.X) * math.Pi / 180
		forward = rl.Vector3{
			X: float32(math.Cos(yaw) * math.Cos(pitch)),
			Y: float32(math.Sin(pitch)),
			Z: float32(math.Sin(yaw) * math.Cos(pitch)),
		}
	}

	audio.SetListener(pos, forward, up)
}
