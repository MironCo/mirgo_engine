//go:build !game

package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"test3d/internal/assets"
	"test3d/internal/components"
	"test3d/internal/engine"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// AssetEntry represents a file or folder in the asset browser
type AssetEntry struct {
	Name     string // Display name
	Path     string // Full path to the file/folder
	IsFolder bool   // True if this is a folder
	Type     string // "folder", "model", "material", "texture", "scene", etc.
}

// drawAssetBrowser draws the asset browser panel at the bottom of the screen
func (e *Editor) drawAssetBrowser() {
	panelH := int32(150)
	panelY := int32(rl.GetScreenHeight()) - panelH
	panelX := e.hierarchyWidth                                              // Start after hierarchy
	panelW := int32(rl.GetScreenWidth()) - e.hierarchyWidth - e.inspectorWidth // Between hierarchy and inspector

	// Reserve space for material editor on the right when a material is selected
	contentW := panelW
	if e.selectedMaterial != nil {
		contentW = panelW - 180
	}

	// Background with border
	rl.DrawRectangle(panelX, panelY, panelW, panelH, colorBgPanel)
	rl.DrawRectangle(panelX, panelY, panelW, 1, colorBorder)

	mousePos := rl.GetMousePosition()

	// Header with back button and path
	headerY := panelY + 6

	// Back button (only show if not at root)
	backBtnX := panelX + 10
	backBtnW := int32(26)
	backBtnH := int32(20)
	canGoBack := e.currentAssetPath != "assets" && e.currentAssetPath != ""

	if canGoBack {
		backHovered := mousePos.X >= float32(backBtnX) && mousePos.X <= float32(backBtnX+backBtnW) &&
			mousePos.Y >= float32(headerY) && mousePos.Y <= float32(headerY+backBtnH)

		backColor := colorBgElement
		if backHovered {
			backColor = colorAccent
		}
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(backBtnX), Y: float32(headerY), Width: float32(backBtnW), Height: float32(backBtnH)}, 0.3, 4, backColor)
		drawTextEx(editorFontBold, "<", backBtnX+8, headerY+3, 16, colorTextPrimary)

		if backHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			// Go up one directory
			e.currentAssetPath = filepath.Dir(e.currentAssetPath)
			e.assetBrowserScroll = 0
			e.selectedMaterial = nil
			e.selectedMaterialPath = ""
			e.scanAssets()
		}
	}

	// Current path
	pathX := backBtnX + backBtnW + 10
	if !canGoBack {
		pathX = panelX + 12
	}
	drawTextEx(editorFont, e.currentAssetPath+"/", pathX, headerY+3, 15, colorTextMuted)

	// Refresh button
	refreshBtnX := panelX + contentW - 75
	refreshBtnY := headerY
	refreshBtnW := int32(65)
	refreshBtnH := int32(20)

	refreshHovered := mousePos.X >= float32(refreshBtnX) && mousePos.X <= float32(refreshBtnX+refreshBtnW) &&
		mousePos.Y >= float32(refreshBtnY) && mousePos.Y <= float32(refreshBtnY+refreshBtnH)

	refreshColor := colorBgElement
	if refreshHovered {
		refreshColor = colorAccent
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(refreshBtnX), Y: float32(refreshBtnY), Width: float32(refreshBtnW), Height: float32(refreshBtnH)}, 0.3, 4, refreshColor)
	drawTextEx(editorFont, "Refresh", refreshBtnX+10, refreshBtnY+3, 14, colorTextSecondary)

	if refreshHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.scanAssets()
	}

	// Asset grid - larger items for better icons
	itemW := int32(80)
	itemH := int32(85)
	startX := panelX + 10
	startY := panelY + 30
	cols := (contentW - 20) / (itemW + 8)
	if cols < 1 {
		cols = 1
	}

	// Scroll handling
	mouseInPanel := mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+contentW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH)

	if mouseInPanel && !rl.IsMouseButtonDown(rl.MouseRightButton) {
		scroll := rl.GetMouseWheelMove()
		e.assetBrowserScroll -= int32(scroll * 30)
		if e.assetBrowserScroll < 0 {
			e.assetBrowserScroll = 0
		}
	}

	// Clip content
	rl.BeginScissorMode(panelX, panelY+24, contentW, panelH-24)

	for i, asset := range e.assetFiles {
		col := int32(i) % cols
		row := int32(i) / cols

		x := startX + col*(itemW+8)
		y := startY + row*(itemH+8) - e.assetBrowserScroll

		// Skip if off screen
		if y+itemH < panelY+24 || y > panelY+panelH {
			continue
		}

		// Item background - rounded
		itemHovered := mousePos.X >= float32(x) && mousePos.X <= float32(x+itemW) &&
			mousePos.Y >= float32(y) && mousePos.Y <= float32(y+itemH)

		isSelected := asset.Path == e.selectedMaterialPath

		bgColor := colorBgElement
		if isSelected {
			bgColor = colorAccent
		} else if itemHovered {
			bgColor = colorBgHover
		}
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(itemW), Height: float32(itemH)}, 0.15, 4, bgColor)

		// Draw icon based on type
		e.drawAssetIcon(x, y, itemW, asset.Type)

		// Name (truncated) - centered below icon
		name := asset.Name
		if len(name) > 10 {
			name = name[:9] + "…"
		}
		textW := rl.MeasureText(name, 13)
		drawTextEx(editorFont, name, x+(itemW-textW)/2, y+itemH-18, 13, colorTextSecondary)

		// Handle clicks
		if itemHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !e.draggingAsset {
			now := rl.GetTime()
			isDoubleClick := (now-e.lastClickTime < 0.3) && (e.lastClickedAsset == asset.Path)

			if asset.IsFolder {
				if isDoubleClick {
					// Double-click folder: navigate into it
					e.currentAssetPath = asset.Path
					e.assetBrowserScroll = 0
					e.selectedMaterial = nil
					e.selectedMaterialPath = ""
					e.scanAssets()
				}
			} else if asset.Type == "material" {
				// Start dragging material
				e.draggingAsset = true
				assetCopy := asset // Make a copy to avoid referencing loop variable
				e.draggedAsset = &assetCopy
				e.selectedMaterialPath = asset.Path
				e.selectedMaterial = assets.LoadMaterial(asset.Path)
			} else if asset.Type == "model" {
				// Click model: spawn into scene
				e.spawnModelFromAsset(asset)
			} else if asset.Type == "scene" {
				if isDoubleClick {
					// Double-click scene: open it
					e.openScene(asset.Path)
				}
			}

			e.lastClickTime = now
			e.lastClickedAsset = asset.Path
		}
	}

	rl.EndScissorMode()

	// Clamp scroll
	rows := (int32(len(e.assetFiles)) + cols - 1) / cols
	maxScroll := rows*(itemH+8) - (panelH - 30)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.assetBrowserScroll > maxScroll {
		e.assetBrowserScroll = maxScroll
	}

	// Empty state
	if len(e.assetFiles) == 0 {
		drawTextEx(editorFont, "Empty folder", panelX+20, panelY+60, 16, colorTextMuted)
	}

	// Draw material editor panel on the right
	if e.selectedMaterial != nil {
		e.drawMaterialEditor(panelX+contentW, panelY, panelW-contentW, panelH)
	}
}

// drawAssetIcon draws the icon for an asset type
func (e *Editor) drawAssetIcon(x, y, itemW int32, assetType string) {
	iconSize := int32(42)
	iconX := x + (itemW-iconSize)/2
	iconY := y + 8

	switch assetType {
	case "folder":
		// Folder icon - rounded with tab
		folderColor := rl.NewColor(220, 180, 80, 255)
		folderDark := rl.NewColor(180, 140, 50, 255)
		// Tab
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY), Width: float32(iconSize/2 + 4), Height: 8}, 0.4, 4, folderColor)
		// Body
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY + 6), Width: float32(iconSize), Height: float32(iconSize - 10)}, 0.2, 4, folderColor)
		// Shadow line
		rl.DrawRectangle(iconX+2, iconY+10, iconSize-4, 2, folderDark)

	case "material":
		// Material icon - gradient sphere effect
		centerX := iconX + iconSize/2
		centerY := iconY + iconSize/2
		radius := float32(iconSize) / 2 - 2
		// Outer ring
		rl.DrawCircle(centerX, centerY, radius, colorAccent)
		// Inner highlight
		rl.DrawCircle(centerX-4, centerY-4, radius*0.6, colorAccentLight)
		// Shine dot
		rl.DrawCircle(centerX-6, centerY-6, 4, rl.NewColor(255, 255, 255, 180))

	case "model":
		// Model icon - 3D cube with depth
		cubeColor := rl.NewColor(120, 200, 140, 255)
		cubeDark := rl.NewColor(80, 160, 100, 255)
		cubeLight := rl.NewColor(160, 230, 180, 255)
		// Main face
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 6), Y: float32(iconY + 6), Width: float32(iconSize - 12), Height: float32(iconSize - 12)}, 0.15, 4, cubeColor)
		// Top edge highlight
		rl.DrawRectangle(iconX+6, iconY+6, iconSize-12, 4, cubeLight)
		// Right edge shadow
		rl.DrawRectangle(iconX+iconSize-10, iconY+10, 4, iconSize-16, cubeDark)
		// "3D" text
		drawTextEx(editorFontBold, "3D", iconX+14, iconY+14, 16, rl.White)

	case "texture":
		// Texture icon - rounded checkerboard
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY), Width: float32(iconSize), Height: float32(iconSize)}, 0.15, 4, rl.NewColor(60, 60, 70, 255))
		half := iconSize / 2
		// Checkerboard pattern inside
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 4), Y: float32(iconY + 4), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(220, 220, 220, 255))
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + half + 2), Y: float32(iconY + half + 2), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(220, 220, 220, 255))
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + half + 2), Y: float32(iconY + 4), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(120, 120, 130, 255))
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 4), Y: float32(iconY + half + 2), Width: float32(half - 6), Height: float32(half - 6)}, 0.2, 2, rl.NewColor(120, 120, 130, 255))

	case "scene":
		// Scene icon - clapperboard style
		sceneColor := rl.NewColor(100, 180, 255, 255)  // Light blue
		sceneDark := rl.NewColor(60, 120, 200, 255)    // Darker blue
		sceneStripe := rl.NewColor(40, 80, 160, 255)   // Stripe color
		// Clapperboard top (angled stripes part)
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY), Width: float32(iconSize), Height: 14}, 0.3, 4, sceneDark)
		// Diagonal stripes on top
		for i := int32(0); i < 5; i++ {
			stripeX := iconX + i*9
			rl.DrawRectangle(stripeX, iconY+2, 4, 10, sceneStripe)
		}
		// Main body
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX), Y: float32(iconY + 12), Width: float32(iconSize), Height: float32(iconSize - 12)}, 0.15, 4, sceneColor)
		// Play triangle in center
		centerX := float32(iconX + iconSize/2)
		centerY := float32(iconY + 12 + (iconSize-12)/2)
		rl.DrawTriangle(
			rl.NewVector2(centerX-6, centerY-8),
			rl.NewVector2(centerX-6, centerY+8),
			rl.NewVector2(centerX+8, centerY),
			rl.White,
		)

	default:
		// Generic file icon - document style
		docColor := rl.NewColor(140, 140, 160, 255)
		// Main body
		rl.DrawRectangleRounded(rl.Rectangle{X: float32(iconX + 6), Y: float32(iconY), Width: float32(iconSize - 12), Height: float32(iconSize)}, 0.15, 4, docColor)
		// Corner fold
		rl.DrawTriangle(
			rl.NewVector2(float32(iconX+iconSize-6), float32(iconY)),
			rl.NewVector2(float32(iconX+iconSize-6), float32(iconY+10)),
			rl.NewVector2(float32(iconX+iconSize-16), float32(iconY)),
			rl.NewColor(100, 100, 120, 255),
		)
		// Lines to represent text
		rl.DrawRectangle(iconX+12, iconY+16, iconSize-24, 3, rl.NewColor(100, 100, 120, 255))
		rl.DrawRectangle(iconX+12, iconY+22, iconSize-28, 3, rl.NewColor(100, 100, 120, 255))
		rl.DrawRectangle(iconX+12, iconY+28, iconSize-24, 3, rl.NewColor(100, 100, 120, 255))
	}
}

// drawMaterialEditor draws the material properties editor in the asset browser
func (e *Editor) drawMaterialEditor(x, y, w, h int32) {
	// Background with border
	rl.DrawRectangle(x, y, w, h, colorBgPanel)
	rl.DrawRectangle(x, y, 1, h, colorBorder)

	// Header
	name := filepath.Base(e.selectedMaterialPath)
	drawTextEx(editorFontBold, name, x+10, y+6, 14, colorAccentLight)

	// Close button - rounded
	closeBtnX := x + w - 22
	closeBtnY := y + 5
	closeBtnSize := int32(16)
	mousePos := rl.GetMousePosition()
	closeHovered := mousePos.X >= float32(closeBtnX) && mousePos.X <= float32(closeBtnX+closeBtnSize) &&
		mousePos.Y >= float32(closeBtnY) && mousePos.Y <= float32(closeBtnY+closeBtnSize)

	closeColor := rl.NewColor(80, 50, 50, 200)
	if closeHovered {
		closeColor = rl.NewColor(180, 60, 60, 230)
	}
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(closeBtnX), Y: float32(closeBtnY), Width: float32(closeBtnSize), Height: float32(closeBtnSize)}, 0.3, 4, closeColor)
	drawTextEx(editorFontBold, "x", closeBtnX+4, closeBtnY+1, 12, colorTextPrimary)

	if closeHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.selectedMaterial = nil
		e.selectedMaterialPath = ""
		return
	}

	// Properties
	propY := y + 26
	labelW := int32(65)
	fieldW := w - labelW - 18
	fieldH := int32(18)
	indent := x + 10

	mat := e.selectedMaterial
	oldMet := mat.Metallic
	oldRough := mat.Roughness
	oldEmit := mat.Emissive

	// Material name (read-only for now)
	drawTextEx(editorFont, "Name:", indent, propY+2, 13, colorTextMuted)
	drawTextEx(editorFont, mat.Name, indent+labelW, propY+2, 13, colorTextSecondary)
	propY += fieldH + 4

	// Color (read-only display)
	drawTextEx(editorFont, "Color:", indent, propY+2, 13, colorTextMuted)
	colorNameStr := assets.LookupColorName(mat.Color)
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(indent + labelW), Y: float32(propY), Width: float32(fieldH), Height: float32(fieldH)}, 0.2, 4, mat.Color)
	drawTextEx(editorFont, colorNameStr, indent+labelW+fieldH+6, propY+2, 13, colorTextSecondary)
	propY += fieldH + 4

	// Metallic
	drawTextEx(editorFont, "Metallic:", indent, propY+3, 13, colorTextMuted)
	mat.Metallic = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.met", mat.Metallic)
	propY += fieldH + 4

	// Roughness
	drawTextEx(editorFont, "Rough:", indent, propY+3, 13, colorTextMuted)
	mat.Roughness = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.rough", mat.Roughness)
	propY += fieldH + 4

	// Emissive
	drawTextEx(editorFont, "Emissive:", indent, propY+3, 13, colorTextMuted)
	mat.Emissive = e.drawFloatField(indent+labelW, propY, fieldW, fieldH, "mated.emit", mat.Emissive)
	propY += fieldH + 4

	// Albedo texture path (editable)
	drawTextEx(editorFont, "Albedo:", indent, propY+3, 13, colorTextMuted)
	oldAlbedo := mat.AlbedoPath
	mat.AlbedoPath = e.drawTextureField(indent+labelW, propY, fieldW, fieldH, "mated.albedo", mat.AlbedoPath)

	// Load texture if path changed
	albedoChanged := mat.AlbedoPath != oldAlbedo
	if albedoChanged && mat.AlbedoPath != "" {
		mat.Albedo = assets.LoadTexture(mat.AlbedoPath)
	} else if albedoChanged && mat.AlbedoPath == "" {
		mat.Albedo = rl.Texture2D{} // Clear texture
	}

	// Auto-save if changed
	if mat.Metallic != oldMet || mat.Roughness != oldRough || mat.Emissive != oldEmit || albedoChanged {
		assets.SaveMaterial(e.selectedMaterialPath, mat)
	}
}

// handleMaterialDrop handles dropping a material onto an object
func (e *Editor) handleMaterialDrop() {
	if e.draggedAsset == nil || e.draggedAsset.Type != "material" {
		e.draggingAsset = false
		e.draggedAsset = nil
		return
	}

	mousePos := rl.GetMousePosition()
	materialPath := e.draggedAsset.Path

	// Check if dropped on hierarchy item
	panelX := int32(0)
	panelY := int32(32)
	panelW := int32(200)
	panelH := int32(rl.GetScreenHeight()) - panelY
	itemH := int32(22)

	if mousePos.X >= float32(panelX) && mousePos.X <= float32(panelX+panelW) &&
		mousePos.Y >= float32(panelY) && mousePos.Y <= float32(panelY+panelH) {
		// Find which object was dropped on
		y := panelY + 28
		for i, g := range e.world.Scene.GameObjects {
			itemY := y + int32(i)*itemH - e.hierarchyScroll
			if mousePos.Y >= float32(itemY) && mousePos.Y < float32(itemY+itemH) {
				e.applyMaterialToObject(g, materialPath)
				break
			}
		}
	} else if !e.mouseInPanel() {
		// Dropped in 3D scene area - raycast to find object
		cam := e.GetRaylibCamera()
		ray := rl.GetScreenToWorldRay(mousePos, cam)
		hit, ok := e.world.EditorRaycast(ray.Position, ray.Direction, 1000)
		if ok && hit.GameObject != nil {
			e.applyMaterialToObject(hit.GameObject, materialPath)
		}
	}

	e.draggingAsset = false
	e.draggedAsset = nil
}

// applyMaterialToObject applies a material to an object's ModelRenderer
func (e *Editor) applyMaterialToObject(obj *engine.GameObject, materialPath string) {
	mr := engine.GetComponent[*components.ModelRenderer](obj)
	if mr == nil {
		e.saveMsg = fmt.Sprintf("%s has no ModelRenderer", obj.Name)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Load and apply the material
	mat := assets.LoadMaterial(materialPath)
	mr.Material = mat
	mr.MaterialPath = materialPath

	e.saveMsg = fmt.Sprintf("Applied material to %s", obj.Name)
	e.saveMsgTime = rl.GetTime()
}

// spawnModelFromAsset creates a new GameObject with the given model
func (e *Editor) spawnModelFromAsset(asset AssetEntry) {
	obj := engine.NewGameObject(asset.Name)

	// Position in front of camera
	forward, _ := e.getDirections()
	obj.Transform.Position = rl.Vector3Add(e.camera.Position, rl.Vector3Scale(forward, 5))
	obj.Transform.Scale = rl.NewVector3(1, 1, 1)

	// Add ModelRenderer
	modelRenderer := components.NewModelRendererFromFile(asset.Path, rl.White)
	obj.AddComponent(modelRenderer)

	// Add to scene
	e.world.Scene.AddGameObject(obj)
	e.world.PhysicsWorld.AddObject(obj)
	e.Selected = obj

	e.saveMsg = fmt.Sprintf("Spawned %s", asset.Name)
	e.saveMsgTime = rl.GetTime()
}

// openScene saves the current scene and loads a new one
func (e *Editor) openScene(scenePath string) {
	// Don't reload if it's the same scene
	if scenePath == world.ScenePath {
		e.saveMsg = "Already editing this scene"
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Save current scene first
	if err := e.world.SaveScene(world.ScenePath); err != nil {
		e.saveMsg = fmt.Sprintf("Save failed: %v", err)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Unload all models from current scene
	for _, g := range e.world.Scene.GameObjects {
		if renderer := engine.GetComponent[*components.ModelRenderer](g); renderer != nil {
			renderer.Unload()
		}
	}

	// Clear scene and physics
	e.world.Scene.GameObjects = e.world.Scene.GameObjects[:0]
	e.world.PhysicsWorld.Objects = e.world.PhysicsWorld.Objects[:0]
	e.world.PhysicsWorld.Statics = e.world.PhysicsWorld.Statics[:0]
	e.world.PhysicsWorld.Kinematics = e.world.PhysicsWorld.Kinematics[:0]

	// Update the scene path
	world.ScenePath = scenePath

	// Load the new scene
	if err := e.world.LoadScene(scenePath); err != nil {
		e.saveMsg = fmt.Sprintf("Failed to load scene: %v", err)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Start all GameObjects in the new scene
	e.world.Scene.Start()

	// Clear selection and undo stack
	e.Selected = nil
	e.undoStack = e.undoStack[:0]

	e.saveMsg = fmt.Sprintf("Opened %s", filepath.Base(scenePath))
	e.saveMsgTime = rl.GetTime()
}

// scanAssets scans the current asset directory and populates the asset list
func (e *Editor) scanAssets() {
	e.assetFiles = nil

	entries, err := os.ReadDir(e.currentAssetPath)
	if err != nil {
		return
	}

	// Sort: folders first, then files
	for _, entry := range entries {
		if entry.IsDir() {
			e.assetFiles = append(e.assetFiles, AssetEntry{
				Name:     entry.Name(),
				Path:     filepath.Join(e.currentAssetPath, entry.Name()),
				IsFolder: true,
				Type:     "folder",
			})
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden/system files
		if strings.HasPrefix(name, ".") {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		fullPath := filepath.Join(e.currentAssetPath, name)

		var assetType string
		switch ext {
		case ".json":
			// Check if in materials folder
			if strings.Contains(e.currentAssetPath, "materials") {
				assetType = "material"
			} else if strings.Contains(e.currentAssetPath, "scenes") {
				assetType = "scene"
			} else {
				assetType = "json"
			}
		case ".gltf", ".glb":
			assetType = "model"
		case ".png", ".jpg", ".jpeg":
			assetType = "texture"
		default:
			assetType = "file"
		}

		e.assetFiles = append(e.assetFiles, AssetEntry{
			Name:     name,
			Path:     fullPath,
			IsFolder: false,
			Type:     assetType,
		})
	}
}
