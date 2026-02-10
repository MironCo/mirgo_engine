package components

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"test3d/internal/audio"
	"test3d/internal/engine"

	"github.com/ebitengine/oto/v3"
)

const (
	hrtfSampleRate   = 44100
	hrtfHeadRadius   = 0.0875 // Average human head radius in meters
	hrtfSpeedOfSound = 343.0  // Speed of sound in m/s
)

// Global oto context - shared across all HRTF sources
var (
	otoContext     *oto.Context
	otoContextOnce sync.Once
	otoContextErr  error
)

func initOtoContext() {
	otoContextOnce.Do(func() {
		op := &oto.NewContextOptions{
			SampleRate:   hrtfSampleRate,
			ChannelCount: 2,
			Format:       oto.FormatFloat32LE,
		}
		var ready chan struct{}
		otoContext, ready, otoContextErr = oto.NewContext(op)
		if otoContextErr != nil {
			fmt.Printf("HRTF: Failed to create oto context: %v\n", otoContextErr)
			return
		}
		<-ready
		fmt.Println("HRTF: Oto audio context initialized")
	})
}

func init() {
	engine.RegisterComponent("HRTFAudioSource", func() engine.Serializable {
		return NewHRTFAudioSource()
	})
}

// HRTFAudioSource provides HRTF-based 3D audio spatialization
type HRTFAudioSource struct {
	engine.BaseComponent

	// Serialized fields
	AudioPath   string
	Volume      float32
	MaxDistance float32
	Loop        bool
	PlayOnStart bool

	// Runtime state
	samples    []float32 // Mono samples normalized to [-1, 1]
	sampleRate int
	player     *oto.Player
	reader     *hrtfReader
}

// hrtfReader implements io.Reader for streaming HRTF-processed audio
type hrtfReader struct {
	source      *HRTFAudioSource
	playhead    int
	leftDelay   []float32
	rightDelay  []float32
	delayWrite  int
	prevLeftG   float32
	prevRightG  float32
	playing     bool
	wantsToPlay bool
	mu          sync.Mutex
}

func NewHRTFAudioSource() *HRTFAudioSource {
	return &HRTFAudioSource{
		Volume:      1.0,
		MaxDistance: 50.0,
		Loop:        false,
		PlayOnStart: false,
	}
}

func (h *HRTFAudioSource) TypeName() string {
	return "HRTFAudioSource"
}

func (h *HRTFAudioSource) Serialize() map[string]any {
	return map[string]any{
		"type":        "HRTFAudioSource",
		"audioPath":   h.AudioPath,
		"volume":      h.Volume,
		"maxDistance": h.MaxDistance,
		"loop":        h.Loop,
		"playOnStart": h.PlayOnStart,
	}
}

func (h *HRTFAudioSource) Deserialize(data map[string]any) {
	if v, ok := data["audioPath"].(string); ok {
		h.AudioPath = v
	}
	if v, ok := data["volume"].(float64); ok {
		h.Volume = float32(v)
	}
	if v, ok := data["maxDistance"].(float64); ok {
		h.MaxDistance = float32(v)
	}
	if v, ok := data["loop"].(bool); ok {
		h.Loop = v
	}
	if v, ok := data["playOnStart"].(bool); ok {
		h.PlayOnStart = v
	}
}

func (h *HRTFAudioSource) Start() {
	if h.AudioPath == "" {
		return
	}

	// Initialize oto context
	initOtoContext()
	if otoContext == nil {
		return
	}

	// Load WAV file manually
	samples, sampleRate, err := loadWavFile(h.AudioPath)
	if err != nil {
		fmt.Printf("HRTF: Failed to load %s: %v\n", h.AudioPath, err)
		return
	}

	h.samples = samples
	h.sampleRate = sampleRate
	fmt.Printf("HRTF: Loaded %s - %d samples at %d Hz\n", h.AudioPath, len(samples), sampleRate)

	// Create reader for HRTF processing
	maxDelaySamples := 128 // ~3ms at 44100Hz
	h.reader = &hrtfReader{
		source:     h,
		leftDelay:  make([]float32, maxDelaySamples),
		rightDelay: make([]float32, maxDelaySamples),
		prevLeftG:  1.0,
		prevRightG: 1.0,
	}

	// Create player
	h.player = otoContext.NewPlayer(h.reader)

	if h.PlayOnStart {
		h.reader.wantsToPlay = true
	}
}

func (h *HRTFAudioSource) Play() {
	if h.reader == nil {
		return
	}
	h.reader.mu.Lock()
	h.reader.playing = true
	h.reader.wantsToPlay = true
	h.reader.mu.Unlock()

	h.player.Play()
	fmt.Println("HRTF: Started playback")
}

func (h *HRTFAudioSource) Stop() {
	if h.reader == nil {
		return
	}
	h.reader.mu.Lock()
	h.reader.playing = false
	h.reader.wantsToPlay = false
	h.reader.playhead = 0
	h.reader.mu.Unlock()

	h.player.Pause()
}

func (h *HRTFAudioSource) Update(deltaTime float32) {
	if h.reader == nil {
		return
	}

	// Handle play mode changes
	if !audio.IsPlayModeEnabled() {
		h.reader.mu.Lock()
		if h.reader.playing {
			h.player.Pause()
			h.reader.playing = false
		}
		h.reader.mu.Unlock()
		return
	}

	h.reader.mu.Lock()
	wantsToPlay := h.reader.wantsToPlay
	playing := h.reader.playing
	h.reader.mu.Unlock()

	if wantsToPlay && !playing {
		h.Play()
	}
}

func (h *HRTFAudioSource) Destroy() {
	if h.player != nil {
		h.player.Close()
	}
}

// Read implements io.Reader for oto player
func (r *hrtfReader) Read(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.playing || len(r.source.samples) == 0 {
		// Output silence
		for i := range buf {
			buf[i] = 0
		}
		return len(buf), nil
	}

	// Get spatial parameters
	listener := audio.GetListener()
	sourcePos := r.source.GetGameObject().WorldPosition()

	// Calculate direction and distance
	dx := sourcePos.X - listener.Position.X
	dy := sourcePos.Y - listener.Position.Y
	dz := sourcePos.Z - listener.Position.Z
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	// Normalize direction
	var dirX, dirZ float32
	if distance > 0.001 {
		dirX = dx / distance
		dirZ = dz / distance
	}

	// Calculate right vector from listener orientation
	// Right = Forward × Up (in 2D on XZ plane: rotate forward 90° clockwise)
	rightX := -listener.Forward.Z
	rightZ := listener.Forward.X
	rightLen := float32(math.Sqrt(float64(rightX*rightX + rightZ*rightZ)))
	if rightLen > 0.001 {
		rightX /= rightLen
		rightZ /= rightLen
	}

	// Calculate pan (-1 to 1, left to right)
	pan := dirX*rightX + dirZ*rightZ

	// Distance attenuation using inverse square law with reference distance
	// At refDist, volume is 100%. Beyond that, falls off with 1/r²
	refDist := float32(2.0) // Reference distance (2 meters for game scale)
	attenuation := float32(1.0)
	if distance > refDist {
		// Inverse square: attenuation = (refDist / distance)²
		ratio := refDist / distance
		attenuation = ratio * ratio
	}
	// Fade out smoothly near max distance instead of hard cutoff
	if distance > r.source.MaxDistance*0.8 {
		fadeStart := r.source.MaxDistance * 0.8
		fadeRange := r.source.MaxDistance * 0.2
		fadeFactor := 1.0 - (distance-fadeStart)/fadeRange
		if fadeFactor < 0 {
			fadeFactor = 0
		}
		attenuation *= fadeFactor
	}

	// ITD: Interaural Time Difference (delay in samples)
	// Maximum ITD is about 0.7ms (~31 samples at 44100Hz)
	itdSeconds := float64(pan) * float64(hrtfHeadRadius) / float64(hrtfSpeedOfSound)
	itdSamples := float32(itdSeconds * float64(hrtfSampleRate))

	var leftDelaySamples, rightDelaySamples float32
	if pan > 0 {
		// Sound is to the right, delay left ear
		leftDelaySamples = itdSamples
		rightDelaySamples = 0
	} else {
		// Sound is to the left, delay right ear
		leftDelaySamples = 0
		rightDelaySamples = -itdSamples
	}

	// ILD: Interaural Level Difference (head shadow effect)
	// More aggressive panning - up to 70% reduction on far ear
	headShadow := float32(0.7)
	absPan := pan
	if absPan < 0 {
		absPan = -absPan
	}

	leftGain := attenuation * r.source.Volume
	rightGain := attenuation * r.source.Volume

	if pan > 0 {
		// Sound to the right - reduce left ear
		leftGain *= (1.0 - headShadow*absPan)
	} else {
		// Sound to the left - reduce right ear
		rightGain *= (1.0 - headShadow*absPan)
	}

	// No smoothing - direct response to position changes
	r.prevLeftG = leftGain
	r.prevRightG = rightGain

	// Process samples (stereo float32 = 8 bytes per frame)
	numFrames := len(buf) / 8

	for i := 0; i < numFrames; i++ {
		// Get source sample
		if r.playhead >= len(r.source.samples) {
			if r.source.Loop {
				r.playhead = 0
			} else {
				r.playing = false
				// Fill rest with silence
				for j := i * 8; j < len(buf); j++ {
					buf[j] = 0
				}
				return len(buf), nil
			}
		}

		sample := r.source.samples[r.playhead]
		r.playhead++

		// Write to delay buffers
		delayLen := len(r.leftDelay)
		r.leftDelay[r.delayWrite] = sample
		r.rightDelay[r.delayWrite] = sample

		// Read from delay buffers with interpolation
		leftReadPos := float32(r.delayWrite) - leftDelaySamples
		rightReadPos := float32(r.delayWrite) - rightDelaySamples

		leftSample := r.interpolateDelay(r.leftDelay, leftReadPos, delayLen) * leftGain
		rightSample := r.interpolateDelay(r.rightDelay, rightReadPos, delayLen) * rightGain

		r.delayWrite = (r.delayWrite + 1) % delayLen

		// Write stereo float32 samples to buffer
		writeFloat32LE(buf[i*8:], leftSample)
		writeFloat32LE(buf[i*8+4:], rightSample)
	}

	return len(buf), nil
}

func (r *hrtfReader) interpolateDelay(buf []float32, pos float32, bufLen int) float32 {
	for pos < 0 {
		pos += float32(bufLen)
	}
	idx0 := int(pos) % bufLen
	idx1 := (idx0 + 1) % bufLen
	frac := pos - float32(int(pos))
	return buf[idx0]*(1-frac) + buf[idx1]*frac
}

func writeFloat32LE(b []byte, v float32) {
	bits := math.Float32bits(v)
	binary.LittleEndian.PutUint32(b, bits)
}

// loadWavFile loads a WAV file and returns normalized float32 mono samples
func loadWavFile(path string) ([]float32, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	// Read RIFF header
	header := make([]byte, 44)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, 0, fmt.Errorf("failed to read WAV header: %w", err)
	}

	// Verify RIFF/WAVE
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a valid WAV file")
	}

	// Parse format
	channels := int(binary.LittleEndian.Uint16(header[22:24]))
	sampleRate := int(binary.LittleEndian.Uint32(header[24:28]))
	bitsPerSample := int(binary.LittleEndian.Uint16(header[34:36]))

	// Find data chunk
	dataSize := int(binary.LittleEndian.Uint32(header[40:44]))

	// Read all data
	data := make([]byte, dataSize)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, 0, fmt.Errorf("failed to read WAV data: %w", err)
	}

	// Convert to float32 mono
	var samples []float32

	if bitsPerSample == 16 {
		numSamples := dataSize / (2 * channels)
		samples = make([]float32, numSamples)
		for i := 0; i < numSamples; i++ {
			var sum float32
			for ch := 0; ch < channels; ch++ {
				idx := (i*channels + ch) * 2
				sample := int16(binary.LittleEndian.Uint16(data[idx:]))
				sum += float32(sample) / 32768.0
			}
			samples[i] = sum / float32(channels)
		}
	} else if bitsPerSample == 32 {
		numSamples := dataSize / (4 * channels)
		samples = make([]float32, numSamples)
		for i := 0; i < numSamples; i++ {
			var sum float32
			for ch := 0; ch < channels; ch++ {
				idx := (i*channels + ch) * 4
				bits := binary.LittleEndian.Uint32(data[idx:])
				sum += math.Float32frombits(bits)
			}
			samples[i] = sum / float32(channels)
		}
	} else {
		return nil, 0, fmt.Errorf("unsupported bits per sample: %d", bitsPerSample)
	}

	return samples, sampleRate, nil
}
