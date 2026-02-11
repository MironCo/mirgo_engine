// Minimal wgpu compute shader test - completely separate from raylib
// Uses cogentcore/webgpu - mature Go WebGPU bindings
package main

import (
	"fmt"
	"unsafe"

	"github.com/cogentcore/webgpu/wgpu"
)

// Simple compute shader that doubles every number in a buffer
const computeShader = `
@group(0) @binding(0)
var<storage, read_write> data: array<f32>;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
    let idx = global_id.x;
    if (idx < arrayLength(&data)) {
        data[idx] = data[idx] * 2.0;
    }
}
`

func main() {
	// 1. Create wgpu instance (will use Metal on Mac)
	instance := wgpu.CreateInstance(nil)
	defer instance.Release()

	// 2. Get adapter (the GPU)
	adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		PowerPreference: wgpu.PowerPreferenceHighPerformance,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to get adapter: %v", err))
	}
	defer adapter.Release()

	// Print what we got
	info := adapter.GetInfo()
	fmt.Printf("Using GPU: %s (%s)\n", info.Name, info.BackendType)

	// 3. Get device (logical handle to GPU)
	device, err := adapter.RequestDevice(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to get device: %v", err))
	}
	defer device.Release()

	queue := device.GetQueue()
	defer queue.Release()

	// 4. Create compute shader module
	shaderModule, err := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "compute_shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{
			Code: computeShader,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create shader: %v", err))
	}
	defer shaderModule.Release()

	// 5. Create compute pipeline
	pipeline, err := device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
		Label: "compute_pipeline",
		Compute: wgpu.ProgrammableStageDescriptor{
			Module:     shaderModule,
			EntryPoint: "main",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create pipeline: %v", err))
	}
	defer pipeline.Release()

	// 6. Create buffer with input data
	inputData := []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	bufferSize := uint64(len(inputData) * 4) // 4 bytes per float32

	// GPU buffer for compute (storage + copy)
	gpuBuffer, err := device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "data_buffer",
		Contents: wgpu.ToBytes(inputData),
		Usage:    wgpu.BufferUsageStorage | wgpu.BufferUsageCopySrc | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create buffer: %v", err))
	}
	defer gpuBuffer.Release()

	// Staging buffer to read results back to CPU
	stagingBuffer, err := device.CreateBuffer(&wgpu.BufferDescriptor{
		Label:            "staging_buffer",
		Size:             bufferSize,
		Usage:            wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
		MappedAtCreation: false,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create staging buffer: %v", err))
	}
	defer stagingBuffer.Release()

	// 7. Create bind group (connects buffer to shader)
	bindGroup, err := device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "bind_group",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: []wgpu.BindGroupEntry{
			{
				Binding: 0,
				Buffer:  gpuBuffer,
				Size:    bufferSize,
			},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create bind group: %v", err))
	}
	defer bindGroup.Release()

	// 8. Record and submit compute commands
	encoder, err := device.CreateCommandEncoder(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create encoder: %v", err))
	}

	computePass := encoder.BeginComputePass(nil)
	computePass.SetPipeline(pipeline)
	computePass.SetBindGroup(0, bindGroup, nil)
	computePass.DispatchWorkgroups(uint32((len(inputData)+63)/64), 1, 1) // ceil(n/64) workgroups
	computePass.End()
	computePass.Release()

	// Copy results to staging buffer
	encoder.CopyBufferToBuffer(gpuBuffer, 0, stagingBuffer, 0, bufferSize)

	commands, err := encoder.Finish(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to finish encoder: %v", err))
	}
	defer commands.Release()

	queue.Submit(commands)

	// 9. Map and read results back
	done := make(chan struct{})
	err = stagingBuffer.MapAsync(wgpu.MapModeRead, 0, bufferSize, func(status wgpu.BufferMapAsyncStatus) {
		if status != wgpu.BufferMapAsyncStatusSuccess {
			panic(fmt.Sprintf("Failed to map buffer: %v", status))
		}
		close(done)
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to start buffer map: %v", err))
	}

	// Wait for GPU
	device.Poll(true, nil)
	<-done

	// Get mapped data
	mappedData := stagingBuffer.GetMappedRange(0, uint(bufferSize))
	results := make([]float32, len(inputData))
	copy(results, unsafe.Slice((*float32)(unsafe.Pointer(&mappedData[0])), len(inputData)))

	stagingBuffer.Unmap()

	fmt.Printf("Input:  %v\n", inputData)
	fmt.Printf("Output: %v\n", results)
	fmt.Println("Compute shader ran on Metal!")
}
