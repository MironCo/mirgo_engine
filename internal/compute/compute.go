// Package compute provides GPU compute shader functionality via WebGPU/Metal.
// This runs completely independently of raylib's OpenGL rendering.
package compute

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/cogentcore/webgpu/wgpu"
)

// System manages the WebGPU compute pipeline.
// Initialize once at startup, use throughout the engine.
type System struct {
	instance *wgpu.Instance
	adapter  *wgpu.Adapter
	device   *wgpu.Device
	queue    *wgpu.Queue

	// Cache of compiled compute pipelines
	pipelines map[string]*Pipeline
	mu        sync.RWMutex
}

// Pipeline represents a compiled compute shader ready to dispatch.
type Pipeline struct {
	shader   *wgpu.ShaderModule
	pipeline *wgpu.ComputePipeline
	layout   *wgpu.BindGroupLayout
}

// Buffer wraps a GPU buffer for compute operations.
type Buffer struct {
	buffer *wgpu.Buffer
	size   uint64
	usage  wgpu.BufferUsage
}

var (
	globalSystem *System
	initOnce     sync.Once
	initErr      error
)

// AdapterInfo contains GPU information.
type AdapterInfo struct {
	Name       string
	Vendor     string
	Backend    string
	DeviceType string
	Driver     string
}

// Initialize sets up the compute system. Safe to call multiple times.
// Returns detailed GPU info on success.
func Initialize() (info AdapterInfo, err error) {
	initOnce.Do(func() {
		globalSystem, initErr = newSystem()
	})
	if initErr != nil {
		return AdapterInfo{}, initErr
	}
	adapterInfo := globalSystem.adapter.GetInfo()
	return AdapterInfo{
		Name:       adapterInfo.Name,
		Vendor:     adapterInfo.VendorName,
		Backend:    adapterInfo.BackendType.String(),
		DeviceType: adapterInfo.AdapterType.String(),
		Driver:     adapterInfo.DriverDescription,
	}, nil
}

// Get returns the global compute system. Must call Initialize first.
func Get() *System {
	return globalSystem
}

func newSystem() (*System, error) {
	instance := wgpu.CreateInstance(nil)

	adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		PowerPreference: wgpu.PowerPreferenceHighPerformance,
	})
	if err != nil {
		instance.Release()
		return nil, fmt.Errorf("failed to get GPU adapter: %w", err)
	}

	device, err := adapter.RequestDevice(nil)
	if err != nil {
		adapter.Release()
		instance.Release()
		return nil, fmt.Errorf("failed to get GPU device: %w", err)
	}

	queue := device.GetQueue()

	return &System{
		instance:  instance,
		adapter:   adapter,
		device:    device,
		queue:     queue,
		pipelines: make(map[string]*Pipeline),
	}, nil
}

// CreatePipeline compiles a compute shader and caches it by name.
func (s *System) CreatePipeline(name, wgslCode, entryPoint string) (*Pipeline, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return cached pipeline if it exists
	if p, ok := s.pipelines[name]; ok {
		return p, nil
	}

	shaderModule, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: name,
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{
			Code: wgslCode,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create shader module: %w", err)
	}

	pipeline, err := s.device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
		Label: name,
		Compute: wgpu.ProgrammableStageDescriptor{
			Module:     shaderModule,
			EntryPoint: entryPoint,
		},
	})
	if err != nil {
		shaderModule.Release()
		return nil, fmt.Errorf("failed to create compute pipeline: %w", err)
	}

	p := &Pipeline{
		shader:   shaderModule,
		pipeline: pipeline,
		layout:   pipeline.GetBindGroupLayout(0),
	}
	s.pipelines[name] = p
	return p, nil
}

// CreateBuffer creates a GPU buffer for compute operations.
func (s *System) CreateBuffer(label string, size uint64, usage wgpu.BufferUsage) (*Buffer, error) {
	buf, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: label,
		Size:  size,
		Usage: usage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create buffer: %w", err)
	}
	return &Buffer{buffer: buf, size: size, usage: usage}, nil
}

// CreateBufferWithData creates a GPU buffer and uploads initial data.
func (s *System) CreateBufferWithData(label string, data []byte, usage wgpu.BufferUsage) (*Buffer, error) {
	buf, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    label,
		Contents: data,
		Usage:    usage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create buffer: %w", err)
	}
	return &Buffer{buffer: buf, size: uint64(len(data)), usage: usage}, nil
}

// WriteBuffer uploads data to a GPU buffer.
func (s *System) WriteBuffer(buf *Buffer, offset uint64, data []byte) {
	s.queue.WriteBuffer(buf.buffer, offset, data)
}

// Dispatch runs a compute shader with the given buffers.
type DispatchParams struct {
	Pipeline    *Pipeline
	Buffers     []*Buffer // Buffers to bind (in order of @binding)
	WorkgroupsX uint32    // Number of workgroups in X
	WorkgroupsY uint32    // Number of workgroups in Y (default 1)
	WorkgroupsZ uint32    // Number of workgroups in Z (default 1)
}

// Dispatch executes a compute shader.
func (s *System) Dispatch(params DispatchParams) error {
	if params.WorkgroupsY == 0 {
		params.WorkgroupsY = 1
	}
	if params.WorkgroupsZ == 0 {
		params.WorkgroupsZ = 1
	}

	// Build bind group entries
	entries := make([]wgpu.BindGroupEntry, len(params.Buffers))
	for i, buf := range params.Buffers {
		entries[i] = wgpu.BindGroupEntry{
			Binding: uint32(i),
			Buffer:  buf.buffer,
			Size:    buf.size,
		}
	}

	bindGroup, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   "compute_bind_group",
		Layout:  params.Pipeline.layout,
		Entries: entries,
	})
	if err != nil {
		return fmt.Errorf("failed to create bind group: %w", err)
	}
	defer bindGroup.Release()

	encoder, err := s.device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("failed to create command encoder: %w", err)
	}

	pass := encoder.BeginComputePass(nil)
	pass.SetPipeline(params.Pipeline.pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.DispatchWorkgroups(params.WorkgroupsX, params.WorkgroupsY, params.WorkgroupsZ)
	pass.End()
	pass.Release()

	commands, err := encoder.Finish(nil)
	if err != nil {
		return fmt.Errorf("failed to finish command encoder: %w", err)
	}
	defer commands.Release()

	s.queue.Submit(commands)
	return nil
}

// ReadBuffer copies GPU buffer data back to CPU.
// The buffer must have been created with BufferUsageCopySrc.
func (s *System) ReadBuffer(buf *Buffer) ([]byte, error) {
	// Create staging buffer
	staging, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "staging_read",
		Size:  buf.size,
		Usage: wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer staging.Release()

	// Copy to staging
	encoder, err := s.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create command encoder: %w", err)
	}
	encoder.CopyBufferToBuffer(buf.buffer, 0, staging, 0, buf.size)
	commands, err := encoder.Finish(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to finish encoder: %w", err)
	}
	s.queue.Submit(commands)
	commands.Release()

	// Map and read
	done := make(chan error, 1)
	err = staging.MapAsync(wgpu.MapModeRead, 0, buf.size, func(status wgpu.BufferMapAsyncStatus) {
		if status != wgpu.BufferMapAsyncStatusSuccess {
			done <- fmt.Errorf("failed to map buffer: %v", status)
		} else {
			done <- nil
		}
	})
	if err != nil {
		return nil, err
	}

	s.device.Poll(true, nil)
	if err := <-done; err != nil {
		return nil, err
	}

	mapped := staging.GetMappedRange(0, uint(buf.size))
	result := make([]byte, len(mapped))
	copy(result, mapped)
	staging.Unmap()

	return result, nil
}

// ReadBufferNonBlocking attempts to read GPU buffer data without blocking.
// Returns (nil, nil) if GPU work is still pending.
// Returns (data, nil) if data is ready.
// Returns (nil, err) on actual errors.
func (s *System) ReadBufferNonBlocking(buf *Buffer) ([]byte, error) {
	// Create staging buffer
	staging, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "staging_read_nb",
		Size:  buf.size,
		Usage: wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer staging.Release()

	// Copy to staging
	encoder, err := s.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create command encoder: %w", err)
	}
	encoder.CopyBufferToBuffer(buf.buffer, 0, staging, 0, buf.size)
	commands, err := encoder.Finish(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to finish encoder: %w", err)
	}
	s.queue.Submit(commands)
	commands.Release()

	// Map and read - NON-BLOCKING
	done := make(chan error, 1)
	mapped := false
	err = staging.MapAsync(wgpu.MapModeRead, 0, buf.size, func(status wgpu.BufferMapAsyncStatus) {
		if status != wgpu.BufferMapAsyncStatusSuccess {
			done <- fmt.Errorf("failed to map buffer: %v", status)
		} else {
			done <- nil
		}
	})
	if err != nil {
		return nil, err
	}

	// Poll once without blocking - if work isn't done, return nil
	s.device.Poll(false, nil)

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		mapped = true
	default:
		// GPU work still pending
		return nil, nil
	}

	if !mapped {
		return nil, nil
	}

	mappedData := staging.GetMappedRange(0, uint(buf.size))
	result := make([]byte, len(mappedData))
	copy(result, mappedData)
	staging.Unmap()

	return result, nil
}

// ReadBufferFloat32 is a convenience method for reading float32 data.
func (s *System) ReadBufferFloat32(buf *Buffer) ([]float32, error) {
	data, err := s.ReadBuffer(buf)
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*float32)(unsafe.Pointer(&data[0])), len(data)/4), nil
}

// Release frees all GPU resources.
func (s *System) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.pipelines {
		p.layout.Release()
		p.pipeline.Release()
		p.shader.Release()
	}
	s.pipelines = nil

	s.queue.Release()
	s.device.Release()
	s.adapter.Release()
	s.instance.Release()
}

// Release frees the buffer's GPU memory.
func (b *Buffer) Release() {
	b.buffer.Release()
}

// Size returns the buffer size in bytes.
func (b *Buffer) Size() uint64 {
	return b.size
}

// Helper to convert a slice to bytes for upload.
func ToBytes[T any](data []T) []byte {
	return wgpu.ToBytes(data)
}
