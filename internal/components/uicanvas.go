package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UICanvas is the root container for UI elements.
// Attach to a GameObject and add UI element children.
// The canvas handles layout calculation and drawing order.
type UICanvas struct {
	engine.BaseComponent

	// Render mode - for now just ScreenSpaceOverlay
	// Could add WorldSpace later for in-world UI
	SortOrder int // Higher values render on top
}

func NewUICanvas() *UICanvas {
	return &UICanvas{
		SortOrder: 0,
	}
}

// Draw renders all UI elements under this canvas
func (c *UICanvas) Draw() {
	g := c.GetGameObject()
	if g == nil {
		return
	}

	// Get screen rect as the root parent rect
	screenRect := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(rl.GetScreenWidth()),
		Height: float32(rl.GetScreenHeight()),
	}

	// Draw this object and all children recursively
	c.drawUIElement(g, screenRect)
}

// drawUIElement recursively draws a UI element and its children
func (c *UICanvas) drawUIElement(g *engine.GameObject, parentRect rl.Rectangle) {
	if g == nil || !g.Active {
		return
	}

	// Get RectTransform to calculate position
	rt := engine.GetComponent[*RectTransform](g)
	currentRect := parentRect

	if rt != nil {
		rt.CalculateRect(parentRect)
		currentRect = rt.GetScreenRect()
	}

	// Draw any UI components on this object
	if text := engine.GetComponent[*UIText](g); text != nil {
		text.Draw(currentRect)
	}
	if img := engine.GetComponent[*UIImage](g); img != nil {
		img.Draw(currentRect)
	}
	if btn := engine.GetComponent[*UIButton](g); btn != nil {
		btn.Draw(currentRect)
	}
	if panel := engine.GetComponent[*UIPanel](g); panel != nil {
		panel.Draw(currentRect)
	}
	if bar := engine.GetComponent[*UIProgressBar](g); bar != nil {
		bar.Draw(currentRect)
	}

	// Draw children
	for _, child := range g.Children {
		c.drawUIElement(child, currentRect)
	}
}

// Update handles UI interaction (clicks, hover)
func (c *UICanvas) Update(deltaTime float32) {
	g := c.GetGameObject()
	if g == nil {
		return
	}

	mousePos := rl.GetMousePosition()
	mousePressed := rl.IsMouseButtonPressed(rl.MouseLeftButton)
	mouseDown := rl.IsMouseButtonDown(rl.MouseLeftButton)
	mouseReleased := rl.IsMouseButtonReleased(rl.MouseLeftButton)

	screenRect := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(rl.GetScreenWidth()),
		Height: float32(rl.GetScreenHeight()),
	}

	c.updateUIElement(g, screenRect, mousePos, mousePressed, mouseDown, mouseReleased)
}

// updateUIElement recursively handles input for UI elements
func (c *UICanvas) updateUIElement(g *engine.GameObject, parentRect rl.Rectangle, mousePos rl.Vector2, pressed, down, released bool) {
	if g == nil || !g.Active {
		return
	}

	rt := engine.GetComponent[*RectTransform](g)
	currentRect := parentRect

	if rt != nil {
		rt.CalculateRect(parentRect)
		currentRect = rt.GetScreenRect()
	}

	// Handle button interactions
	if btn := engine.GetComponent[*UIButton](g); btn != nil {
		btn.HandleInput(currentRect, mousePos, pressed, down, released)
	}

	// Update children
	for _, child := range g.Children {
		c.updateUIElement(child, currentRect, mousePos, pressed, down, released)
	}
}

// Serialization
func (c *UICanvas) TypeName() string { return "UICanvas" }

func (c *UICanvas) Serialize() map[string]any {
	return map[string]any{
		"sortOrder": c.SortOrder,
	}
}

func (c *UICanvas) Deserialize(data map[string]any) {
	if v, ok := data["sortOrder"].(float64); ok {
		c.SortOrder = int(v)
	}
}

func init() {
	engine.RegisterComponent("UICanvas", func() engine.Serializable {
		return NewUICanvas()
	})
}
