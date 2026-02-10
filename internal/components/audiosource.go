package components

import (
	"test3d/internal/audio"
	"test3d/internal/engine"
)

func init() {
	engine.RegisterComponent("AudioSource", func() engine.Serializable {
		return NewAudioSource()
	})
}

type AudioSource struct {
	engine.BaseComponent

	// Serialized fields
	AudioPath   string  `json:"audioPath"`
	Volume      float32 `json:"volume"`
	MaxDistance float32 `json:"maxDistance"`
	Loop        bool    `json:"loop"`
	PlayOnStart bool    `json:"playOnStart"`
	Spatial     bool    `json:"spatial"` // 3D spatialization

	// Runtime state
	sourceID uint64
	loaded   bool
}

func NewAudioSource() *AudioSource {
	return &AudioSource{
		Volume:      1.0,
		MaxDistance: 50.0,
		Loop:        false,
		PlayOnStart: false,
		Spatial:     true,
	}
}

func (a *AudioSource) TypeName() string {
	return "AudioSource"
}

func (a *AudioSource) Serialize() map[string]any {
	return map[string]any{
		"type":        "AudioSource",
		"audioPath":   a.AudioPath,
		"volume":      a.Volume,
		"maxDistance": a.MaxDistance,
		"loop":        a.Loop,
		"playOnStart": a.PlayOnStart,
		"spatial":     a.Spatial,
	}
}

func (a *AudioSource) Deserialize(data map[string]any) {
	if v, ok := data["audioPath"].(string); ok {
		a.AudioPath = v
	}
	if v, ok := data["volume"].(float64); ok {
		a.Volume = float32(v)
	}
	if v, ok := data["maxDistance"].(float64); ok {
		a.MaxDistance = float32(v)
	}
	if v, ok := data["loop"].(bool); ok {
		a.Loop = v
	}
	if v, ok := data["playOnStart"].(bool); ok {
		a.PlayOnStart = v
	}
	if v, ok := data["spatial"].(bool); ok {
		a.Spatial = v
	}
}

func (a *AudioSource) Start() {
	if a.AudioPath != "" {
		a.Load(a.AudioPath)
		if a.PlayOnStart {
			a.Play()
		}
	}
}

func (a *AudioSource) Update(deltaTime float32) {
	if !a.loaded {
		return
	}

	// Update source position for spatial audio
	if g := a.GetGameObject(); g != nil {
		audio.SetSourcePosition(a.sourceID, g.WorldPosition())
	}
}

// Load loads an audio file
func (a *AudioSource) Load(path string) bool {
	if a.loaded {
		a.Unload()
	}

	id, ok := audio.LoadSound(path)
	if !ok {
		return false
	}

	a.sourceID = id
	a.loaded = true
	a.AudioPath = path

	// Apply settings
	audio.SetSourceVolume(a.sourceID, a.Volume)
	audio.SetSourceMaxDistance(a.sourceID, a.MaxDistance)
	audio.SetSourceLoop(a.sourceID, a.Loop)
	audio.SetSourceSpatial(a.sourceID, a.Spatial)

	return true
}

// Unload releases the audio resource
func (a *AudioSource) Unload() {
	if a.loaded {
		audio.UnloadSource(a.sourceID)
		a.loaded = false
	}
}

// Play starts playback
func (a *AudioSource) Play() {
	if a.loaded {
		audio.Play(a.sourceID)
	}
}

// Stop stops playback
func (a *AudioSource) Stop() {
	if a.loaded {
		audio.Stop(a.sourceID)
	}
}

// IsPlaying returns whether the source is currently playing
func (a *AudioSource) IsPlaying() bool {
	if !a.loaded {
		return false
	}
	return audio.IsPlaying(a.sourceID)
}

// SetVolume updates the volume
func (a *AudioSource) SetVolume(vol float32) {
	a.Volume = vol
	if a.loaded {
		audio.SetSourceVolume(a.sourceID, vol)
	}
}
