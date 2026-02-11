//go:build !game

package game

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"test3d/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// EditorRestoreState holds editor state for hot-reload restoration
type EditorRestoreState struct {
	CameraPosition  rl.Vector3 `json:"cameraPosition"`
	CameraYaw       float32    `json:"cameraYaw"`
	CameraPitch     float32    `json:"cameraPitch"`
	CameraMoveSpeed float32    `json:"cameraMoveSpeed"`
	GizmoMode       int        `json:"gizmoMode"`
	SelectedUID     uint64     `json:"selectedUID,omitempty"`
}

const editorRestoreFile = ".editor_restore.json"

// RestoreState restores editor camera state from the restore file after hot-reload
func (e *Editor) RestoreState() {
	data, err := os.ReadFile(editorRestoreFile)
	if err != nil {
		return // No restore file, that's fine
	}

	var state EditorRestoreState
	if err := json.Unmarshal(data, &state); err != nil {
		fmt.Printf("Failed to parse restore state: %v\n", err)
		os.Remove(editorRestoreFile)
		return
	}

	// Apply restored state
	e.camera.Position = state.CameraPosition
	e.camera.Yaw = state.CameraYaw
	e.camera.Pitch = state.CameraPitch
	if state.CameraMoveSpeed > 0 {
		e.camera.MoveSpeed = state.CameraMoveSpeed
	}
	e.gizmoMode = GizmoMode(state.GizmoMode)

	// Restore selected object by UID
	if state.SelectedUID > 0 {
		e.Selected = e.world.Scene.FindByUID(state.SelectedUID)
	}

	// Clean up the restore file
	os.Remove(editorRestoreFile)

	fmt.Println("Scripts reloaded successfully")
}

// rebuildAndRelaunch saves state, rebuilds the binary, and relaunches
func (e *Editor) rebuildAndRelaunch() {
	// Don't allow rebuild while paused (scene has runtime modifications)
	if e.Paused {
		e.saveMsg = "Cannot rebuild while paused"
		e.saveMsgTime = rl.GetTime()
		return
	}

	// Check if a rebuild is already in progress
	e.rebuildMutex.Lock()
	if e.rebuildInProgress {
		e.rebuildMutex.Unlock()
		return
	}
	e.rebuildInProgress = true
	e.rebuildProgress = 0.0
	e.rebuildStage = "Starting..."
	e.rebuildMutex.Unlock()

	// Run rebuild asynchronously
	go func() {
		// Helper to update progress
		updateProgress := func(progress float32, stage string) {
			e.rebuildMutex.Lock()
			e.rebuildProgress = progress
			e.rebuildStage = stage
			e.rebuildMutex.Unlock()
		}

		// Helper to handle errors
		handleError := func(msg string) {
			e.rebuildMutex.Lock()
			e.rebuildInProgress = false
			e.saveMsg = msg
			e.saveMsgTime = rl.GetTime()
			e.rebuildMutex.Unlock()
		}

		// Get the current executable path
		execPath, err := os.Executable()
		if err != nil {
			handleError(fmt.Sprintf("Failed to get executable: %v", err))
			return
		}

		// Build to a temp file first to check if it compiles
		tempExec := execPath + ".new"

		// Stage 1: Generate scripts (0% - 20%)
		updateProgress(0.0, "Generating scripts...")
		fmt.Println("Generating scripts...")

		genCmd := exec.Command("go", "run", "./cmd/gen-scripts")
		genOutput, genErr := genCmd.CombinedOutput()
		if genErr != nil {
			handleError("Script generation failed!")
			fmt.Printf("Script generation error:\n%s\n", string(genOutput))
			return
		}

		// Stage 2: Compile (20% - 80%)
		updateProgress(0.2, "Compiling...")
		fmt.Println("Compiling...")

		cmd := exec.Command("go", "build", "-o", tempExec, "./cmd/test3d")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Build failed - show error, keep window open
			handleError("Build failed!")
			fmt.Printf("Build error:\n%s\n", string(output))
			os.Remove(tempExec)
			return
		}

		// Stage 3: Save state and prepare relaunch (80% - 100%)
		updateProgress(0.8, "Preparing reload...")

		// Save the scene
		if err := e.world.SaveScene(world.ScenePath); err != nil {
			handleError(fmt.Sprintf("Save failed: %v", err))
			os.Remove(tempExec)
			return
		}

		// Save editor state for restoration
		state := EditorRestoreState{
			CameraPosition:  e.camera.Position,
			CameraYaw:       e.camera.Yaw,
			CameraPitch:     e.camera.Pitch,
			CameraMoveSpeed: e.camera.MoveSpeed,
			GizmoMode:       int(e.gizmoMode),
		}
		if e.Selected != nil {
			state.SelectedUID = e.Selected.UID
		}
		stateJSON, _ := json.MarshalIndent(state, "", "  ")
		if err := os.WriteFile(editorRestoreFile, stateJSON, 0644); err != nil {
			handleError(fmt.Sprintf("Failed to save state: %v", err))
			os.Remove(tempExec)
			return
		}

		// Replace old binary with new one
		if err := os.Rename(tempExec, execPath); err != nil {
			handleError(fmt.Sprintf("Failed to replace binary: %v", err))
			os.Remove(tempExec)
			os.Remove(editorRestoreFile)
			return
		}

		updateProgress(1.0, "Reloading...")
		fmt.Println("Relaunching...")

		// Signal main thread to close window and relaunch
		// (can't call CloseWindow from goroutine - must be on main thread)
		e.rebuildMutex.Lock()
		e.rebuildReadyToExit = true
		e.rebuildExecPath = execPath
		e.rebuildMutex.Unlock()
	}()
}

// checkRebuildExit checks if rebuild is ready to relaunch and handles it on main thread
func (e *Editor) checkRebuildExit() {
	e.rebuildMutex.Lock()
	if e.rebuildReadyToExit {
		execPath := e.rebuildExecPath
		e.rebuildMutex.Unlock()

		// Close the window (must be on main thread)
		rl.CloseWindow()

		// Replace current process with new binary
		err := execNewBinary(execPath, []string{execPath, "--restore-editor"})
		if err != nil {
			fmt.Printf("Failed to exec: %v\n", err)
			os.Exit(1)
		}
	} else {
		e.rebuildMutex.Unlock()
	}
}

// buildGame runs the Rust utility to build and package the game
func (e *Editor) buildGame() {
	e.saveMsg = "Building game..."
	e.saveMsgTime = rl.GetTime()

	cmd := exec.Command("./mirgo-utils", "build")
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.saveMsg = fmt.Sprintf("Build failed: %v", err)
		fmt.Printf("Build error: %v\nOutput: %s\n", err, string(output))
	} else {
		e.saveMsg = "Build complete! See build/"
		fmt.Printf("Build output:\n%s\n", string(output))
	}
	e.saveMsgTime = rl.GetTime()
}

// scanScriptModTimes records the modification times of all script files
func (e *Editor) scanScriptModTimes() {
	e.scriptModTimes = make(map[string]int64)
	e.scriptsChanged = false
	scriptsDir := "internal/scripts"

	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(scriptsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		e.scriptModTimes[path] = info.ModTime().UnixNano()
	}
	e.lastScriptCheck = rl.GetTime()
}

// checkScriptChanges checks if any script files have been modified
func (e *Editor) checkScriptChanges() {
	// Only check every 0.5 seconds
	if rl.GetTime()-e.lastScriptCheck < 0.5 {
		return
	}
	e.lastScriptCheck = rl.GetTime()

	scriptsDir := "internal/scripts"
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(scriptsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime().UnixNano()

		// Check if file is new or modified
		if oldTime, exists := e.scriptModTimes[path]; !exists || modTime != oldTime {
			e.scriptsChanged = true
			return
		}
	}
}

// EditorPrefs holds persistent editor preferences saved between sessions
type EditorPrefs struct {
	WindowWidth      int        `json:"windowWidth"`
	WindowHeight     int        `json:"windowHeight"`
	WindowX          int        `json:"windowX"`
	WindowY          int        `json:"windowY"`
	CameraPosition   rl.Vector3 `json:"cameraPosition"`
	CameraYaw        float32    `json:"cameraYaw"`
	CameraPitch      float32    `json:"cameraPitch"`
	CameraMoveSpeed  float32    `json:"cameraMoveSpeed"`
	ScenePath        string     `json:"scenePath"`
	AssetBrowserOpen bool       `json:"assetBrowserOpen"`
	AssetBrowserPath string     `json:"assetBrowserPath"`
	HierarchyWidth   int32      `json:"hierarchyWidth"`
	InspectorWidth   int32      `json:"inspectorWidth"`
}

const editorPrefsFile = ".editor_prefs.json"

// LoadEditorPrefs loads editor preferences from disk
func LoadEditorPrefs() *EditorPrefs {
	data, err := os.ReadFile(editorPrefsFile)
	if err != nil {
		return nil
	}

	var prefs EditorPrefs
	if err := json.Unmarshal(data, &prefs); err != nil {
		fmt.Printf("Failed to parse editor prefs: %v\n", err)
		return nil
	}

	return &prefs
}

// SavePrefs saves the current editor state to disk
func (e *Editor) SavePrefs() {
	prefs := EditorPrefs{
		WindowWidth:      rl.GetScreenWidth(),
		WindowHeight:     rl.GetScreenHeight(),
		WindowX:          int(rl.GetWindowPosition().X),
		WindowY:          int(rl.GetWindowPosition().Y),
		CameraPosition:   e.camera.Position,
		CameraYaw:        e.camera.Yaw,
		CameraPitch:      e.camera.Pitch,
		CameraMoveSpeed:  e.camera.MoveSpeed,
		ScenePath:        world.ScenePath,
		AssetBrowserOpen: e.showAssetBrowser,
		AssetBrowserPath: e.currentAssetPath,
		HierarchyWidth:   e.hierarchyWidth,
		InspectorWidth:   e.inspectorWidth,
	}

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal editor prefs: %v\n", err)
		return
	}

	if err := os.WriteFile(editorPrefsFile, data, 0644); err != nil {
		fmt.Printf("Failed to save editor prefs: %v\n", err)
	}
}

// ApplyPrefs applies loaded preferences to the editor
func (e *Editor) ApplyPrefs(prefs *EditorPrefs) {
	if prefs == nil {
		return
	}

	e.camera.Position = prefs.CameraPosition
	e.camera.Yaw = prefs.CameraYaw
	e.camera.Pitch = prefs.CameraPitch
	if prefs.CameraMoveSpeed > 0 {
		e.camera.MoveSpeed = prefs.CameraMoveSpeed
	}
	if prefs.HierarchyWidth > 0 {
		e.hierarchyWidth = prefs.HierarchyWidth
	}
	if prefs.InspectorWidth > 0 {
		e.inspectorWidth = prefs.InspectorWidth
	}
	e.showAssetBrowser = prefs.AssetBrowserOpen
	if prefs.AssetBrowserPath != "" {
		e.currentAssetPath = prefs.AssetBrowserPath
		if e.showAssetBrowser {
			e.scanAssets()
		}
	}
}
