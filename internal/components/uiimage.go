package components

import (
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UIImage displays a texture or solid color rectangle
type UIImage struct {
	engine.BaseComponent

	// Texture path (loaded on Start)
	TexturePath string
	texture     rl.Texture2D

	// Fallback color if no texture
	Color rl.Color

	// Tint applied to texture
	Tint rl.Color

	// Whether to preserve aspect ratio
	PreserveAspect bool
}

func NewUIImage() *UIImage {
	return &UIImage{
		Color:          rl.White,
		Tint:           rl.White,
		PreserveAspect: false,
	}
}

func (i *UIImage) Start() {
	if i.TexturePath != "" {
		i.texture = rl.LoadTexture(i.TexturePath)
	}
}

// Draw renders the image within the given rect
func (i *UIImage) Draw(rect rl.Rectangle) {
	if i.texture.ID > 0 {
		// Draw texture
		var destRect rl.Rectangle

		if i.PreserveAspect {
			// Calculate aspect-preserving rect
			texAspect := float32(i.texture.Width) / float32(i.texture.Height)
			rectAspect := rect.Width / rect.Height

			if texAspect > rectAspect {
				// Texture is wider - fit to width
				destRect.Width = rect.Width
				destRect.Height = rect.Width / texAspect
				destRect.X = rect.X
				destRect.Y = rect.Y + (rect.Height-destRect.Height)/2
			} else {
				// Texture is taller - fit to height
				destRect.Height = rect.Height
				destRect.Width = rect.Height * texAspect
				destRect.X = rect.X + (rect.Width-destRect.Width)/2
				destRect.Y = rect.Y
			}
		} else {
			destRect = rect
		}

		sourceRect := rl.Rectangle{
			X:      0,
			Y:      0,
			Width:  float32(i.texture.Width),
			Height: float32(i.texture.Height),
		}

		rl.DrawTexturePro(i.texture, sourceRect, destRect, rl.Vector2{}, 0, i.Tint)
	} else {
		// Draw solid color rectangle
		rl.DrawRectangleRec(rect, i.Color)
	}
}

// SetTexture loads a texture from path
func (i *UIImage) SetTexture(path string) {
	if i.texture.ID > 0 {
		rl.UnloadTexture(i.texture)
	}
	i.TexturePath = path
	if path != "" {
		i.texture = rl.LoadTexture(path)
	}
}

// Serialization
func (i *UIImage) TypeName() string { return "UIImage" }

func (i *UIImage) Serialize() map[string]any {
	return map[string]any{
		"texturePath":    i.TexturePath,
		"color":          []uint8{i.Color.R, i.Color.G, i.Color.B, i.Color.A},
		"tint":           []uint8{i.Tint.R, i.Tint.G, i.Tint.B, i.Tint.A},
		"preserveAspect": i.PreserveAspect,
	}
}

func (i *UIImage) Deserialize(data map[string]any) {
	if v, ok := data["texturePath"].(string); ok {
		i.TexturePath = v
	}
	if v, ok := data["color"].([]any); ok && len(v) >= 4 {
		i.Color = rl.NewColor(
			uint8(v[0].(float64)),
			uint8(v[1].(float64)),
			uint8(v[2].(float64)),
			uint8(v[3].(float64)),
		)
	}
	if v, ok := data["tint"].([]any); ok && len(v) >= 4 {
		i.Tint = rl.NewColor(
			uint8(v[0].(float64)),
			uint8(v[1].(float64)),
			uint8(v[2].(float64)),
			uint8(v[3].(float64)),
		)
	}
	if v, ok := data["preserveAspect"].(bool); ok {
		i.PreserveAspect = v
	}
}

func init() {
	engine.RegisterComponent("UIImage", func() engine.Serializable {
		return NewUIImage()
	})
}
