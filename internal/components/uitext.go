package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TextAlignment controls horizontal text alignment
type TextAlignment int

const (
	TextAlignLeft TextAlignment = iota
	TextAlignCenter
	TextAlignRight
)

// UIText displays text on screen
type UIText struct {
	engine.BaseComponent

	Text      string
	FontSize  int32
	Color     rl.Color
	Alignment TextAlignment
}

func NewUIText() *UIText {
	return &UIText{
		Text:      "Text",
		FontSize:  20,
		Color:     rl.White,
		Alignment: TextAlignLeft,
	}
}

// Draw renders the text within the given rect
func (t *UIText) Draw(rect rl.Rectangle) {
	if t.Text == "" {
		return
	}

	// Measure text for alignment
	textWidth := float32(rl.MeasureText(t.Text, t.FontSize))

	var x float32
	switch t.Alignment {
	case TextAlignLeft:
		x = rect.X
	case TextAlignCenter:
		x = rect.X + (rect.Width-textWidth)/2
	case TextAlignRight:
		x = rect.X + rect.Width - textWidth
	}

	// Vertically center text in rect
	y := rect.Y + (rect.Height-float32(t.FontSize))/2

	rl.DrawText(t.Text, int32(x), int32(y), t.FontSize, t.Color)
}

// Serialization
func (t *UIText) TypeName() string { return "UIText" }

func (t *UIText) Serialize() map[string]any {
	return map[string]any{
		"text":      t.Text,
		"fontSize":  t.FontSize,
		"color":     []uint8{t.Color.R, t.Color.G, t.Color.B, t.Color.A},
		"alignment": int(t.Alignment),
	}
}

func (t *UIText) Deserialize(data map[string]any) {
	if v, ok := data["text"].(string); ok {
		t.Text = v
	}
	if v, ok := data["fontSize"].(float64); ok {
		t.FontSize = int32(v)
	}
	if v, ok := data["color"].([]any); ok && len(v) >= 4 {
		t.Color = rl.NewColor(
			uint8(v[0].(float64)),
			uint8(v[1].(float64)),
			uint8(v[2].(float64)),
			uint8(v[3].(float64)),
		)
	}
	if v, ok := data["alignment"].(float64); ok {
		t.Alignment = TextAlignment(int(v))
	}
}

func init() {
	engine.RegisterComponent("UIText", func() engine.Serializable {
		return NewUIText()
	})
}
