package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UIProgressBar displays a fill-based progress indicator (health bar, loading, etc.)
type UIProgressBar struct {
	engine.BaseComponent

	// Current value (0 to MaxValue)
	Value    float32
	MaxValue float32

	// Colors
	BackgroundColor rl.Color
	FillColor       rl.Color
	BorderColor     rl.Color

	// Border width (0 = no border)
	BorderWidth int32

	// Fill direction
	FillFromRight bool // If true, fills from right to left
}

func NewUIProgressBar() *UIProgressBar {
	return &UIProgressBar{
		Value:           100,
		MaxValue:        100,
		BackgroundColor: rl.NewColor(40, 40, 50, 255),
		FillColor:       rl.NewColor(80, 200, 80, 255), // Green
		BorderColor:     rl.NewColor(60, 60, 75, 255),
		BorderWidth:     1,
		FillFromRight:   false,
	}
}

// GetPercent returns the fill percentage (0-1)
func (pb *UIProgressBar) GetPercent() float32 {
	if pb.MaxValue <= 0 {
		return 0
	}
	p := pb.Value / pb.MaxValue
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// SetPercent sets value based on percentage (0-1)
func (pb *UIProgressBar) SetPercent(percent float32) {
	pb.Value = percent * pb.MaxValue
}

// Draw renders the progress bar
func (pb *UIProgressBar) Draw(rect rl.Rectangle) {
	// Draw background
	rl.DrawRectangleRec(rect, pb.BackgroundColor)

	// Calculate fill rect
	percent := pb.GetPercent()
	fillWidth := rect.Width * percent

	var fillRect rl.Rectangle
	if pb.FillFromRight {
		fillRect = rl.Rectangle{
			X:      rect.X + rect.Width - fillWidth,
			Y:      rect.Y,
			Width:  fillWidth,
			Height: rect.Height,
		}
	} else {
		fillRect = rl.Rectangle{
			X:      rect.X,
			Y:      rect.Y,
			Width:  fillWidth,
			Height: rect.Height,
		}
	}

	// Draw fill
	if fillWidth > 0 {
		rl.DrawRectangleRec(fillRect, pb.FillColor)
	}

	// Draw border
	if pb.BorderWidth > 0 {
		rl.DrawRectangleLinesEx(rect, float32(pb.BorderWidth), pb.BorderColor)
	}
}

// Serialization
func (pb *UIProgressBar) TypeName() string { return "UIProgressBar" }

func (pb *UIProgressBar) Serialize() map[string]any {
	return map[string]any{
		"value":           pb.Value,
		"maxValue":        pb.MaxValue,
		"backgroundColor": []uint8{pb.BackgroundColor.R, pb.BackgroundColor.G, pb.BackgroundColor.B, pb.BackgroundColor.A},
		"fillColor":       []uint8{pb.FillColor.R, pb.FillColor.G, pb.FillColor.B, pb.FillColor.A},
		"borderColor":     []uint8{pb.BorderColor.R, pb.BorderColor.G, pb.BorderColor.B, pb.BorderColor.A},
		"borderWidth":     pb.BorderWidth,
		"fillFromRight":   pb.FillFromRight,
	}
}

func (pb *UIProgressBar) Deserialize(data map[string]any) {
	if v, ok := data["value"].(float64); ok {
		pb.Value = float32(v)
	}
	if v, ok := data["maxValue"].(float64); ok {
		pb.MaxValue = float32(v)
	}
	if v, ok := data["backgroundColor"].([]any); ok && len(v) >= 4 {
		pb.BackgroundColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["fillColor"].([]any); ok && len(v) >= 4 {
		pb.FillColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["borderColor"].([]any); ok && len(v) >= 4 {
		pb.BorderColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["borderWidth"].(float64); ok {
		pb.BorderWidth = int32(v)
	}
	if v, ok := data["fillFromRight"].(bool); ok {
		pb.FillFromRight = v
	}
}

func init() {
	engine.RegisterComponent("UIProgressBar", func() engine.Serializable {
		return NewUIProgressBar()
	})
}
