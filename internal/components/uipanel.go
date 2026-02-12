package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UIPanel is a simple background panel/container
type UIPanel struct {
	engine.BaseComponent

	// Background color
	Color rl.Color

	// Border settings
	BorderColor  rl.Color
	BorderWidth  int32
	BorderRadius float32 // Rounded corners (0 = sharp)
}

func NewUIPanel() *UIPanel {
	return &UIPanel{
		Color:        rl.NewColor(30, 30, 40, 200),
		BorderColor:  rl.NewColor(60, 60, 75, 255),
		BorderWidth:  1,
		BorderRadius: 0,
	}
}

// Draw renders the panel background
func (p *UIPanel) Draw(rect rl.Rectangle) {
	if p.BorderRadius > 0 {
		// Rounded rectangle
		rl.DrawRectangleRounded(rect, p.BorderRadius/rect.Height, 8, p.Color)
		if p.BorderWidth > 0 {
			rl.DrawRectangleRoundedLinesEx(rect, p.BorderRadius/rect.Height, 8, float32(p.BorderWidth), p.BorderColor)
		}
	} else {
		// Sharp rectangle
		rl.DrawRectangleRec(rect, p.Color)
		if p.BorderWidth > 0 {
			rl.DrawRectangleLinesEx(rect, float32(p.BorderWidth), p.BorderColor)
		}
	}
}

// Serialization
func (p *UIPanel) TypeName() string { return "UIPanel" }

func (p *UIPanel) Serialize() map[string]any {
	return map[string]any{
		"color":        []uint8{p.Color.R, p.Color.G, p.Color.B, p.Color.A},
		"borderColor":  []uint8{p.BorderColor.R, p.BorderColor.G, p.BorderColor.B, p.BorderColor.A},
		"borderWidth":  p.BorderWidth,
		"borderRadius": p.BorderRadius,
	}
}

func (p *UIPanel) Deserialize(data map[string]any) {
	if v, ok := data["color"].([]any); ok && len(v) >= 4 {
		p.Color = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["borderColor"].([]any); ok && len(v) >= 4 {
		p.BorderColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["borderWidth"].(float64); ok {
		p.BorderWidth = int32(v)
	}
	if v, ok := data["borderRadius"].(float64); ok {
		p.BorderRadius = float32(v)
	}
}

func init() {
	engine.RegisterComponent("UIPanel", func() engine.Serializable {
		return NewUIPanel()
	})
}
