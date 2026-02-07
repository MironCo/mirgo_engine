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

// importModel safely imports a GLTF/GLB file into assets/models/
func (e *Editor) importModel(srcPath string) {
	filename := filepath.Base(srcPath)
	ext := strings.ToLower(filepath.Ext(filename))
	name := strings.TrimSuffix(filename, ext)

	dstDir := filepath.Join("assets/models", name)

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		e.setMsg("Failed to create model dir: %v", err)
		return
	}

	dstModelPath := filepath.Join(dstDir, filename)

	// Copy main file (.gltf or .glb)
	if err := copyFile(srcPath, dstModelPath); err != nil {
		e.setMsg("Failed to copy model: %v", err)
		return
	}

	// Handle glTF dependencies
	if ext == ".gltf" {
		srcDir := filepath.Dir(srcPath)

		// Copy .bin files
		binFiles, _ := filepath.Glob(filepath.Join(srcDir, "*.bin"))
		if len(binFiles) == 0 {
			e.setMsg("Invalid glTF: missing .bin file")
			return
		}

		for _, bin := range binFiles {
			copyFile(bin, filepath.Join(dstDir, filepath.Base(bin)))
		}

		// Copy textures directory if present
		texturesSrc := filepath.Join(srcDir, "textures")
		texturesDst := filepath.Join(dstDir, "textures")
		if dirExists(texturesSrc) {
			if err := copyDir(texturesSrc, texturesDst); err != nil {
				e.setMsg("Failed to copy textures: %v", err)
				return
			}
		}
	}

	// Create a new GameObject
	obj := engine.NewGameObject(name)
	obj.Transform.Position = e.camera.Position
	obj.Transform.Scale = rl.NewVector3(10, 10, 10)

	// Add ModelRenderer component (safe path)
	modelRenderer := components.NewModelRendererFromFile(dstModelPath, rl.White)
	obj.AddComponent(modelRenderer)

	// Add to scene
	e.world.Scene.AddGameObject(obj)
	e.Selected = obj

	e.setMsg("Imported: %s", filename)
}

// ---------- Helpers ----------

func (e *Editor) setMsg(format string, args ...any) {
	e.saveMsg = fmt.Sprintf(format, args...)
	e.saveMsgTime = rl.GetTime()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		return copyFile(path, target)
	})
}
