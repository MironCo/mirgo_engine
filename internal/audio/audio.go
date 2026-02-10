package audio

import (
	"math"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Listener represents the audio listener position and orientation
type Listener struct {
	Position rl.Vector3
	Forward  rl.Vector3
	Right    rl.Vector3
}

// Source represents an audio source in the world
type Source struct {
	ID          uint64
	Position    rl.Vector3
	Sound       rl.Sound
	Volume      float32
	MaxDistance float32
	Loop        bool
	Spatial     bool
	playing     bool
	wantsToPlay bool // True if Play() was called but playModeEnabled was false
}

// Manager handles audio playback
type Manager struct {
	mu       sync.Mutex
	listener Listener
	sources  map[uint64]*Source
	nextID   uint64
}

var globalManager *Manager
var playModeEnabled bool // Only play audio when in play mode

// Init initializes the audio system
func Init() {
	rl.InitAudioDevice()
	globalManager = &Manager{
		sources: make(map[uint64]*Source),
	}
}

// SetPlayMode enables or disables audio playback (for editor vs play mode)
func SetPlayMode(enabled bool) {
	playModeEnabled = enabled
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if enabled {
		// Start any sounds that wanted to play
		for _, src := range globalManager.sources {
			if src.wantsToPlay && !src.playing {
				rl.PlaySound(src.Sound)
				src.playing = true
			}
		}
	} else {
		// Stop all sounds when exiting play mode
		for _, src := range globalManager.sources {
			if src.playing {
				rl.StopSound(src.Sound)
				src.playing = false
			}
		}
	}
}

// Close shuts down the audio system
func Close() {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	for _, src := range globalManager.sources {
		rl.UnloadSound(src.Sound)
	}
	globalManager.sources = nil
	globalManager.mu.Unlock()
	rl.CloseAudioDevice()
}

// SetListener updates the listener position and orientation
func SetListener(pos, forward, up rl.Vector3) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	globalManager.listener.Position = pos

	// Normalize forward, default to -Z if zero
	fwdLen := rl.Vector3Length(forward)
	if fwdLen > 0.001 {
		globalManager.listener.Forward = rl.Vector3Scale(forward, 1.0/fwdLen)
	} else {
		globalManager.listener.Forward = rl.Vector3{X: 0, Y: 0, Z: -1}
	}

	// Calculate right vector (up × forward)
	right := rl.Vector3CrossProduct(up, globalManager.listener.Forward)
	rightLen := rl.Vector3Length(right)
	if rightLen > 0.001 {
		globalManager.listener.Right = rl.Vector3Scale(right, 1.0/rightLen)
	} else {
		globalManager.listener.Right = rl.Vector3{X: 1, Y: 0, Z: 0}
	}
}

// LoadSound loads audio from a file and returns a source ID
func LoadSound(path string) (uint64, bool) {
	if globalManager == nil {
		return 0, false
	}

	sound := rl.LoadSound(path)
	if !rl.IsSoundValid(sound) {
		return 0, false
	}

	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	id := globalManager.nextID
	globalManager.nextID++

	globalManager.sources[id] = &Source{
		ID:          id,
		Sound:       sound,
		Volume:      1.0,
		MaxDistance: 50.0,
		Spatial:     true,
	}

	return id, true
}

// Play starts playing a source
func Play(id uint64) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		src.wantsToPlay = true
		if playModeEnabled {
			rl.PlaySound(src.Sound)
			src.playing = true
		}
	}
}

// Stop stops a source
func Stop(id uint64) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		rl.StopSound(src.Sound)
		src.playing = false
		src.wantsToPlay = false
	}
}

// SetSourcePosition updates a source's position
func SetSourcePosition(id uint64, pos rl.Vector3) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		src.Position = pos
	}
}

// SetSourceVolume sets the volume for a source
func SetSourceVolume(id uint64, volume float32) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		src.Volume = volume
	}
}

// SetSourceLoop sets whether a source loops
func SetSourceLoop(id uint64, loop bool) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		src.Loop = loop
	}
}

// SetSourceMaxDistance sets the max audible distance
func SetSourceMaxDistance(id uint64, dist float32) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		src.MaxDistance = dist
	}
}

// SetSourceSpatial sets whether a source uses 3D spatialization
func SetSourceSpatial(id uint64, spatial bool) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		src.Spatial = spatial
	}
}

// UnloadSource removes a source
func UnloadSource(id uint64) {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		rl.UnloadSound(src.Sound)
		delete(globalManager.sources, id)
	}
}

// Update updates spatial audio parameters for all sources
func Update() {
	if globalManager == nil {
		return
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	listener := globalManager.listener

	for _, src := range globalManager.sources {
		if !src.playing {
			continue
		}

		// Handle looping
		if src.Loop && !rl.IsSoundPlaying(src.Sound) {
			rl.PlaySound(src.Sound)
		} else if !src.Loop && !rl.IsSoundPlaying(src.Sound) {
			src.playing = false
			continue
		}

		if !src.Spatial {
			// 2D audio - center pan, full volume
			rl.SetSoundVolume(src.Sound, src.Volume)
			rl.SetSoundPan(src.Sound, 0.5)
			continue
		}

		// Calculate spatial audio
		toSource := rl.Vector3Subtract(src.Position, listener.Position)
		distance := rl.Vector3Length(toSource)

		// Distance attenuation
		var volume float32 = 0
		if distance < src.MaxDistance {
			// Linear falloff
			volume = src.Volume * (1.0 - distance/src.MaxDistance)
		}

		// Pan based on angle to listener's right vector
		var pan float32 = 0.5 // center
		if distance > 0.001 {
			direction := rl.Vector3Scale(toSource, 1.0/distance)
			rightDot := rl.Vector3DotProduct(direction, listener.Right)
			// rightDot: -1 = full left, +1 = full right
			// pan: 0 = full left, 0.5 = center, 1 = full right
			pan = 0.5 + rightDot*0.5

			// Clamp pan to valid range
			if pan < 0.0 {
				pan = 0.0
			} else if pan > 1.0 {
				pan = 1.0
			}

			// Also factor in front/back - sounds behind are slightly quieter
			frontDot := rl.Vector3DotProduct(direction, listener.Forward)
			if frontDot < 0 {
				// Sound is behind, reduce volume slightly
				volume *= 0.7 + 0.3*float32(math.Abs(float64(frontDot)))
			}
		}

		rl.SetSoundVolume(src.Sound, volume)
		rl.SetSoundPan(src.Sound, pan)
	}
}

// IsPlaying returns whether a source is currently playing
func IsPlaying(id uint64) bool {
	if globalManager == nil {
		return false
	}
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if src, ok := globalManager.sources[id]; ok {
		return rl.IsSoundPlaying(src.Sound)
	}
	return false
}
