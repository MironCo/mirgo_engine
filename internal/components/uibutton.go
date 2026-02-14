package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ButtonState tracks the current visual state of a button
type ButtonState int

const (
	ButtonNormal ButtonState = iota
	ButtonHovered
	ButtonPressed
	ButtonDisabled
)

// UIButton is an interactive button element
type UIButton struct {
	engine.BaseComponent

	// Visual colors for each state
	NormalColor   rl.Color
	HoverColor    rl.Color
	PressedColor  rl.Color
	DisabledColor rl.Color

	// Border
	BorderColor rl.Color
	BorderWidth int32

	// Current state
	State    ButtonState
	Disabled bool

	// Unity-style event - supports multiple listeners
	OnClick engine.Event

	// For detecting click (press and release on same button)
	wasPressed bool
}

func NewUIButton() *UIButton {
	return &UIButton{
		NormalColor:   rl.NewColor(60, 60, 70, 255),
		HoverColor:    rl.NewColor(80, 80, 95, 255),
		PressedColor:  rl.NewColor(100, 100, 120, 255),
		DisabledColor: rl.NewColor(40, 40, 45, 255),
		BorderColor:   rl.NewColor(100, 100, 115, 255),
		BorderWidth:   1,
		State:         ButtonNormal,
	}
}

// Draw renders the button background
func (b *UIButton) Draw(rect rl.Rectangle) {
	var color rl.Color

	if b.Disabled {
		color = b.DisabledColor
	} else {
		switch b.State {
		case ButtonHovered:
			color = b.HoverColor
		case ButtonPressed:
			color = b.PressedColor
		default:
			color = b.NormalColor
		}
	}

	// Draw background
	rl.DrawRectangleRec(rect, color)

	// Draw border
	if b.BorderWidth > 0 {
		rl.DrawRectangleLinesEx(rect, float32(b.BorderWidth), b.BorderColor)
	}
}

// HandleInput processes mouse input for the button
func (b *UIButton) HandleInput(rect rl.Rectangle, mousePos rl.Vector2, pressed, down, released bool) {
	if b.Disabled {
		b.State = ButtonDisabled
		return
	}

	isHovered := rl.CheckCollisionPointRec(mousePos, rect)

	if isHovered {
		if down {
			b.State = ButtonPressed
			b.wasPressed = true
		} else {
			b.State = ButtonHovered
		}

		// Click detection: released while hovering and was pressed on this button
		if released && b.wasPressed {
			b.OnClick.Invoke()
			b.wasPressed = false
		}
	} else {
		b.State = ButtonNormal
		if released {
			b.wasPressed = false
		}
	}
}

// Serialization
func (b *UIButton) TypeName() string { return "UIButton" }

func (b *UIButton) Serialize() map[string]any {
	return map[string]any{
		"normalColor":   []uint8{b.NormalColor.R, b.NormalColor.G, b.NormalColor.B, b.NormalColor.A},
		"hoverColor":    []uint8{b.HoverColor.R, b.HoverColor.G, b.HoverColor.B, b.HoverColor.A},
		"pressedColor":  []uint8{b.PressedColor.R, b.PressedColor.G, b.PressedColor.B, b.PressedColor.A},
		"disabledColor": []uint8{b.DisabledColor.R, b.DisabledColor.G, b.DisabledColor.B, b.DisabledColor.A},
		"borderColor":   []uint8{b.BorderColor.R, b.BorderColor.G, b.BorderColor.B, b.BorderColor.A},
		"borderWidth":   b.BorderWidth,
		"disabled":      b.Disabled,
	}
}

func (b *UIButton) Deserialize(data map[string]any) {
	if v, ok := data["normalColor"].([]any); ok && len(v) >= 4 {
		b.NormalColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["hoverColor"].([]any); ok && len(v) >= 4 {
		b.HoverColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["pressedColor"].([]any); ok && len(v) >= 4 {
		b.PressedColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["disabledColor"].([]any); ok && len(v) >= 4 {
		b.DisabledColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["borderColor"].([]any); ok && len(v) >= 4 {
		b.BorderColor = rl.NewColor(uint8(v[0].(float64)), uint8(v[1].(float64)), uint8(v[2].(float64)), uint8(v[3].(float64)))
	}
	if v, ok := data["borderWidth"].(float64); ok {
		b.BorderWidth = int32(v)
	}
	if v, ok := data["disabled"].(bool); ok {
		b.Disabled = v
	}
}

func init() {
	engine.RegisterComponent("UIButton", func() engine.Serializable {
		return NewUIButton()
	})
}
