package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Anchor presets for common UI layouts (like Unity)
type AnchorPreset int

const (
	AnchorTopLeft AnchorPreset = iota
	AnchorTopCenter
	AnchorTopRight
	AnchorMiddleLeft
	AnchorMiddleCenter
	AnchorMiddleRight
	AnchorBottomLeft
	AnchorBottomCenter
	AnchorBottomRight
	AnchorStretchTop
	AnchorStretchMiddle
	AnchorStretchBottom
	AnchorStretchLeft
	AnchorStretchCenter
	AnchorStretchRight
	AnchorStretchAll
)

// RectTransform positions UI elements in screen space with anchoring support.
// Works like Unity's RectTransform - anchors define relative position to parent,
// and offsets define the actual size/position relative to those anchors.
type RectTransform struct {
	engine.BaseComponent

	// Anchor points (0-1 range, relative to parent)
	// AnchorMin is bottom-left, AnchorMax is top-right
	AnchorMin rl.Vector2 // Default: {0, 0}
	AnchorMax rl.Vector2 // Default: {0, 0} (point anchor)

	// Pivot point for rotation/scaling (0-1 range within element)
	Pivot rl.Vector2 // Default: {0.5, 0.5} (center)

	// Position offset from anchor (in pixels)
	// When anchors are same point: this is position relative to anchor
	// When anchors differ: this is inset from edges
	AnchoredPosition rl.Vector2

	// Size of the element (when anchors are same point)
	SizeDelta rl.Vector2

	// Computed screen rectangle (updated each frame)
	screenRect rl.Rectangle
}

func NewRectTransform() *RectTransform {
	return &RectTransform{
		AnchorMin:        rl.Vector2{X: 0.5, Y: 0.5},
		AnchorMax:        rl.Vector2{X: 0.5, Y: 0.5},
		Pivot:            rl.Vector2{X: 0.5, Y: 0.5},
		AnchoredPosition: rl.Vector2{X: 0, Y: 0},
		SizeDelta:        rl.Vector2{X: 100, Y: 30},
	}
}

// SetAnchorPreset configures anchors using common presets
func (rt *RectTransform) SetAnchorPreset(preset AnchorPreset) {
	switch preset {
	case AnchorTopLeft:
		rt.AnchorMin = rl.Vector2{X: 0, Y: 0}
		rt.AnchorMax = rl.Vector2{X: 0, Y: 0}
		rt.Pivot = rl.Vector2{X: 0, Y: 0}
	case AnchorTopCenter:
		rt.AnchorMin = rl.Vector2{X: 0.5, Y: 0}
		rt.AnchorMax = rl.Vector2{X: 0.5, Y: 0}
		rt.Pivot = rl.Vector2{X: 0.5, Y: 0}
	case AnchorTopRight:
		rt.AnchorMin = rl.Vector2{X: 1, Y: 0}
		rt.AnchorMax = rl.Vector2{X: 1, Y: 0}
		rt.Pivot = rl.Vector2{X: 1, Y: 0}
	case AnchorMiddleLeft:
		rt.AnchorMin = rl.Vector2{X: 0, Y: 0.5}
		rt.AnchorMax = rl.Vector2{X: 0, Y: 0.5}
		rt.Pivot = rl.Vector2{X: 0, Y: 0.5}
	case AnchorMiddleCenter:
		rt.AnchorMin = rl.Vector2{X: 0.5, Y: 0.5}
		rt.AnchorMax = rl.Vector2{X: 0.5, Y: 0.5}
		rt.Pivot = rl.Vector2{X: 0.5, Y: 0.5}
	case AnchorMiddleRight:
		rt.AnchorMin = rl.Vector2{X: 1, Y: 0.5}
		rt.AnchorMax = rl.Vector2{X: 1, Y: 0.5}
		rt.Pivot = rl.Vector2{X: 1, Y: 0.5}
	case AnchorBottomLeft:
		rt.AnchorMin = rl.Vector2{X: 0, Y: 1}
		rt.AnchorMax = rl.Vector2{X: 0, Y: 1}
		rt.Pivot = rl.Vector2{X: 0, Y: 1}
	case AnchorBottomCenter:
		rt.AnchorMin = rl.Vector2{X: 0.5, Y: 1}
		rt.AnchorMax = rl.Vector2{X: 0.5, Y: 1}
		rt.Pivot = rl.Vector2{X: 0.5, Y: 1}
	case AnchorBottomRight:
		rt.AnchorMin = rl.Vector2{X: 1, Y: 1}
		rt.AnchorMax = rl.Vector2{X: 1, Y: 1}
		rt.Pivot = rl.Vector2{X: 1, Y: 1}
	case AnchorStretchAll:
		rt.AnchorMin = rl.Vector2{X: 0, Y: 0}
		rt.AnchorMax = rl.Vector2{X: 1, Y: 1}
		rt.Pivot = rl.Vector2{X: 0.5, Y: 0.5}
	}
}

// GetScreenRect returns the computed screen-space rectangle
func (rt *RectTransform) GetScreenRect() rl.Rectangle {
	return rt.screenRect
}

// CalculateRect computes screen position based on parent rect and anchors
func (rt *RectTransform) CalculateRect(parentRect rl.Rectangle) {
	// Calculate anchor positions in parent space
	anchorMinX := parentRect.X + parentRect.Width*rt.AnchorMin.X
	anchorMinY := parentRect.Y + parentRect.Height*rt.AnchorMin.Y
	anchorMaxX := parentRect.X + parentRect.Width*rt.AnchorMax.X
	anchorMaxY := parentRect.Y + parentRect.Height*rt.AnchorMax.Y

	var x, y, width, height float32

	// If anchors are the same point, use SizeDelta for size
	if rt.AnchorMin.X == rt.AnchorMax.X && rt.AnchorMin.Y == rt.AnchorMax.Y {
		// Point anchor - position relative to anchor point
		width = rt.SizeDelta.X
		height = rt.SizeDelta.Y
		x = anchorMinX + rt.AnchoredPosition.X - width*rt.Pivot.X
		y = anchorMinY + rt.AnchoredPosition.Y - height*rt.Pivot.Y
	} else {
		// Stretched anchors - SizeDelta acts as insets
		x = anchorMinX + rt.AnchoredPosition.X
		y = anchorMinY + rt.AnchoredPosition.Y
		width = (anchorMaxX - anchorMinX) + rt.SizeDelta.X
		height = (anchorMaxY - anchorMinY) + rt.SizeDelta.Y
	}

	rt.screenRect = rl.Rectangle{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

// ContainsPoint checks if a screen point is inside this rect
func (rt *RectTransform) ContainsPoint(point rl.Vector2) bool {
	return rl.CheckCollisionPointRec(point, rt.screenRect)
}

// Serialization
func (rt *RectTransform) TypeName() string { return "RectTransform" }

func (rt *RectTransform) Serialize() map[string]any {
	return map[string]any{
		"anchorMin":        []float32{rt.AnchorMin.X, rt.AnchorMin.Y},
		"anchorMax":        []float32{rt.AnchorMax.X, rt.AnchorMax.Y},
		"pivot":            []float32{rt.Pivot.X, rt.Pivot.Y},
		"anchoredPosition": []float32{rt.AnchoredPosition.X, rt.AnchoredPosition.Y},
		"sizeDelta":        []float32{rt.SizeDelta.X, rt.SizeDelta.Y},
	}
}

func (rt *RectTransform) Deserialize(data map[string]any) {
	if v, ok := data["anchorMin"].([]any); ok && len(v) >= 2 {
		rt.AnchorMin.X = float32(v[0].(float64))
		rt.AnchorMin.Y = float32(v[1].(float64))
	}
	if v, ok := data["anchorMax"].([]any); ok && len(v) >= 2 {
		rt.AnchorMax.X = float32(v[0].(float64))
		rt.AnchorMax.Y = float32(v[1].(float64))
	}
	if v, ok := data["pivot"].([]any); ok && len(v) >= 2 {
		rt.Pivot.X = float32(v[0].(float64))
		rt.Pivot.Y = float32(v[1].(float64))
	}
	if v, ok := data["anchoredPosition"].([]any); ok && len(v) >= 2 {
		rt.AnchoredPosition.X = float32(v[0].(float64))
		rt.AnchoredPosition.Y = float32(v[1].(float64))
	}
	if v, ok := data["sizeDelta"].([]any); ok && len(v) >= 2 {
		rt.SizeDelta.X = float32(v[0].(float64))
		rt.SizeDelta.Y = float32(v[1].(float64))
	}
}

func init() {
	engine.RegisterComponent("RectTransform", func() engine.Serializable {
		return NewRectTransform()
	})
}
