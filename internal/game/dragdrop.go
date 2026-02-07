//go:build !game

package game

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleFileDrop checks for dropped files and imports supported assets
func (e *Editor) handleFileDrop() {
	if !rl.IsFileDropped() {
		return
	}

	files := rl.LoadDroppedFiles()
	defer rl.UnloadDroppedFiles()

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		switch ext {
		case ".gltf", ".glb":
			e.importModel(file)
		default:
			e.saveMsg = fmt.Sprintf("Unsupported file type: %s", ext)
			e.saveMsgTime = rl.GetTime()
		}
	}
}

// importModel copies a GLTF/GLB file to assets/models/ and spawns an object
func (e *Editor) importModel(srcPath string) {
	// Get just the filename
	filename := filepath.Base(srcPath)

	// Destination path in assets/models/
	dstDir := "assets/models"
	dstPath := filepath.Join(dstDir, filename)

	// Create models directory if needed
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		e.saveMsg = fmt.Sprintf("Failed to create models dir: %v", err)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Copy file
	if err := copyFile(srcPath, dstPath); err != nil {
		e.saveMsg = fmt.Sprintf("Failed to copy: %v", err)
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Create a new GameObject with the model
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	obj := engine.NewGameObject(name)
	obj.Transform.Position = e.camera.Position

	// Add ModelRenderer component
	modelRenderer := components.NewModelRendererFromFile(dstPath, rl.White)
	obj.AddComponent(modelRenderer)

	// Add to scene
	e.world.Scene.AddGameObject(obj)
	e.Selected = obj

	e.saveMsg = fmt.Sprintf("Imported: %s", filename)
	e.saveMsgTime = rl.GetTime()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
