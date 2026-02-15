//go:build !game

package game

import (
	"log"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Editor fonts - Outfit for UI, JetBrains Mono for values
var editorFont rl.Font     // Outfit Regular - main UI font
var editorFontBold rl.Font // Outfit Bold - headers
var editorFontMono rl.Font // JetBrains Mono - numeric values
var editorFontsLoaded bool

// Theme colors - Indigo/purple dark theme matching the website
var (
	// Base backgrounds (dark with slight blue tint)
	colorBgDark    = rl.NewColor(10, 10, 15, 255) // Darkest - nav bg
	colorBgPanel   = rl.NewColor(18, 18, 24, 245) // Panel backgrounds
	colorBgElement = rl.NewColor(28, 28, 38, 255) // Input fields, buttons
	colorBgHover   = rl.NewColor(38, 38, 52, 255) // Hover state
	colorBgActive  = rl.NewColor(48, 48, 65, 255) // Active/pressed state

	// Accent colors - indigo/purple gradient
	colorAccent       = rl.NewColor(108, 99, 255, 255)  // Primary indigo #6c63ff
	colorAccentLight  = rl.NewColor(167, 139, 250, 255) // Light purple #a78bfa
	colorAccentHover  = rl.NewColor(130, 120, 255, 255) // Hover indigo
	colorAccentActive = rl.NewColor(90, 80, 220, 255)   // Pressed indigo

	// Text colors
	colorTextPrimary   = rl.NewColor(255, 255, 255, 255) // White
	colorTextSecondary = rl.NewColor(200, 200, 208, 255) // Light gray #c8c8d0
	colorTextMuted     = rl.NewColor(119, 119, 119, 255) // Muted #777

	// Borders
	colorBorder      = rl.NewColor(255, 255, 255, 13) // rgba(255,255,255,0.05)
	colorBorderHover = rl.NewColor(108, 99, 255, 100) // Indigo border on hover

	// Selection highlight (indigo tinted)
	colorSelection = rl.NewColor(108, 99, 255, 60) // Indigo with transparency
)

// initRayguiStyle sets up the modern indigo dark theme
func initRayguiStyle() {
	// Load fonts at high resolution for smooth scaling
	if !editorFontsLoaded {
		editorFontsLoaded = true

		// Load Outfit Regular for main UI (high res for smooth scaling)
		editorFont = rl.LoadFontEx("assets/fonts/Outfit-Regular.ttf", 48, nil)
		if editorFont.Texture.ID > 0 {
			rl.SetTextureFilter(editorFont.Texture, rl.FilterBilinear)
			gui.SetFont(editorFont)
			log.Println("Loaded Outfit-Regular font")
		} else {
			log.Println("Failed to load Outfit-Regular font")
		}

		// Load Outfit Bold for headers
		editorFontBold = rl.LoadFontEx("assets/fonts/Outfit-Bold.ttf", 48, nil)
		if editorFontBold.Texture.ID > 0 {
			rl.SetTextureFilter(editorFontBold.Texture, rl.FilterBilinear)
			log.Println("Loaded Outfit-Bold font")
		} else {
			log.Println("Failed to load Outfit-Bold font")
		}

		// Load JetBrains Mono for numeric values
		editorFontMono = rl.LoadFontEx("assets/fonts/JetBrainsMono-Regular.ttf", 48, nil)
		if editorFontMono.Texture.ID > 0 {
			rl.SetTextureFilter(editorFontMono.Texture, rl.FilterBilinear)
			log.Println("Loaded JetBrainsMono font")
		} else {
			log.Println("Failed to load JetBrainsMono font")
		}
	}

	// Background colors - dark with blue tint
	gui.SetStyle(gui.DEFAULT, gui.BACKGROUND_COLOR, gui.NewColorPropertyValue(colorBgDark))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_NORMAL, gui.NewColorPropertyValue(colorBgElement))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_FOCUSED, gui.NewColorPropertyValue(colorBgHover))
	gui.SetStyle(gui.DEFAULT, gui.BASE_COLOR_PRESSED, gui.NewColorPropertyValue(colorAccent))

	// Text colors
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_NORMAL, gui.NewColorPropertyValue(colorTextSecondary))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_FOCUSED, gui.NewColorPropertyValue(colorTextPrimary))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_COLOR_PRESSED, gui.NewColorPropertyValue(colorTextPrimary))

	// Border colors - subtle with indigo on focus
	gui.SetStyle(gui.DEFAULT, gui.BORDER_COLOR_NORMAL, gui.NewColorPropertyValue(rl.NewColor(50, 50, 65, 255)))
	gui.SetStyle(gui.DEFAULT, gui.BORDER_COLOR_FOCUSED, gui.NewColorPropertyValue(colorAccent))

	// Line color (for separators)
	gui.SetStyle(gui.DEFAULT, gui.LINE_COLOR, gui.NewColorPropertyValue(rl.NewColor(40, 40, 55, 255)))

	// Text size
	gui.SetStyle(gui.DEFAULT, gui.TEXT_SIZE, 15)
}

// drawTextEx draws text using the specified font scaled to the requested size
func drawTextEx(font rl.Font, text string, x, y int32, size float32, color rl.Color) {
	if font.Texture.ID > 0 {
		rl.DrawTextEx(font, text, rl.Vector2{X: float32(x), Y: float32(y)}, size, 0, color)
	} else {
		rl.DrawText(text, x, y, int32(size), color)
	}
}

// colorName returns a human-readable name for common colors.
func colorName(c rl.Color) string {
	switch c {
	case rl.Red:
		return "Red"
	case rl.Blue:
		return "Blue"
	case rl.Green:
		return "Green"
	case rl.Purple:
		return "Purple"
	case rl.Orange:
		return "Orange"
	case rl.Yellow:
		return "Yellow"
	case rl.Pink:
		return "Pink"
	case rl.SkyBlue:
		return "SkyBlue"
	case rl.Lime:
		return "Lime"
	case rl.Magenta:
		return "Magenta"
	case rl.White:
		return "White"
	case rl.LightGray:
		return "LightGray"
	case rl.Gray:
		return "Gray"
	case rl.DarkGray:
		return "DarkGray"
	case rl.Black:
		return "Black"
	case rl.Brown:
		return "Brown"
	case rl.Beige:
		return "Beige"
	case rl.Maroon:
		return "Maroon"
	case rl.Gold:
		return "Gold"
	default:
		return "#" + string([]byte{
			hexChar(c.R >> 4), hexChar(c.R & 0xF),
			hexChar(c.G >> 4), hexChar(c.G & 0xF),
			hexChar(c.B >> 4), hexChar(c.B & 0xF),
		})
	}
}

func hexChar(n uint8) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}
