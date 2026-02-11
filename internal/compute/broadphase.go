// GPU-accelerated broad-phase collision detection
package compute

import (
	"github.com/cogentcore/webgpu/wgpu"
)

// BroadPhase handles GPU-accelerated collision pair detection.
// Uses sphere bounding volumes for fast culling.
type BroadPhase struct {
	system   *System
	pipeline *Pipeline

	// Buffers
	sphereBuffer *Buffer // Input: positions + radii
	pairBuffer   *Buffer // Output: collision pairs
	countBuffer  *Buffer // Output: number of pairs found

	maxObjects uint32
	maxPairs   uint32
}

// Sphere represents a bounding sphere for broad-phase detection.
// Packed as vec4: xyz = position, w = radius
type Sphere struct {
	X, Y, Z float32
	Radius  float32
}

// CollisionPair represents two objects that may be colliding.
type CollisionPair struct {
	A, B uint32
}

const broadPhaseShader = `
// Broad-phase collision detection shader
// Each thread checks one object against all others with higher indices
// This gives us n*(n-1)/2 checks with no duplicates

struct Sphere {
    pos: vec3<f32>,
    radius: f32,
}

struct Pair {
    a: u32,
    b: u32,
}

@group(0) @binding(0) var<storage, read> spheres: array<Sphere>;
@group(0) @binding(1) var<storage, read_write> pairs: array<Pair>;
@group(0) @binding(2) var<storage, read_write> pairCount: atomic<u32>;
@group(0) @binding(3) var<uniform> objectCount: u32;

@compute @workgroup_size(256)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
    let i = global_id.x;
    if (i >= objectCount) {
        return;
    }

    let sphereA = spheres[i];

    // Check against all objects with higher index (avoids duplicates)
    for (var j = i + 1u; j < objectCount; j = j + 1u) {
        let sphereB = spheres[j];

        // Distance check
        let diff = sphereA.pos - sphereB.pos;
        let distSq = dot(diff, diff);
        let radiusSum = sphereA.radius + sphereB.radius;

        if (distSq < radiusSum * radiusSum) {
            // Collision! Atomically add to output
            let idx = atomicAdd(&pairCount, 1u);

            // Bounds check (don't overflow pair buffer)
            if (idx < arrayLength(&pairs)) {
                pairs[idx] = Pair(i, j);
            }
        }
    }
}
`

// NewBroadPhase creates a GPU broad-phase system.
// maxObjects: maximum number of objects to track
// maxPairs: maximum collision pairs to output (should be generous, e.g., maxObjects * 10)
func NewBroadPhase(maxObjects, maxPairs uint32) (*BroadPhase, error) {
	sys := Get()
	if sys == nil {
		return nil, nil // Compute not available
	}

	pipeline, err := sys.CreatePipeline("broadphase", broadPhaseShader, "main")
	if err != nil {
		return nil, err
	}

	// Create buffers
	sphereSize := uint64(maxObjects * 16) // 4 floats * 4 bytes
	pairSize := uint64(maxPairs * 8)      // 2 uint32s * 4 bytes
	countSize := uint64(4)                // 1 uint32

	sphereBuffer, err := sys.CreateBuffer("spheres", sphereSize,
		wgpu.BufferUsageStorage|wgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}

	pairBuffer, err := sys.CreateBuffer("pairs", pairSize,
		wgpu.BufferUsageStorage|wgpu.BufferUsageCopySrc)
	if err != nil {
		sphereBuffer.Release()
		return nil, err
	}

	countBuffer, err := sys.CreateBuffer("pairCount", countSize,
		wgpu.BufferUsageStorage|wgpu.BufferUsageCopySrc|wgpu.BufferUsageCopyDst)
	if err != nil {
		sphereBuffer.Release()
		pairBuffer.Release()
		return nil, err
	}

	return &BroadPhase{
		system:       sys,
		pipeline:     pipeline,
		sphereBuffer: sphereBuffer,
		pairBuffer:   pairBuffer,
		countBuffer:  countBuffer,
		maxObjects:   maxObjects,
		maxPairs:     maxPairs,
	}, nil
}

// DetectPairs finds all potentially colliding pairs.
// Returns slice of (indexA, indexB) pairs where indices correspond to input sphere order.
func (bp *BroadPhase) DetectPairs(spheres []Sphere) ([]CollisionPair, error) {
	if len(spheres) == 0 {
		return nil, nil
	}
	if uint32(len(spheres)) > bp.maxObjects {
		spheres = spheres[:bp.maxObjects]
	}

	// Upload sphere data
	bp.system.WriteBuffer(bp.sphereBuffer, 0, ToBytes(spheres))

	// Reset pair count to 0
	bp.system.WriteBuffer(bp.countBuffer, 0, ToBytes([]uint32{0}))

	// Create uniform buffer for object count
	objectCount := uint32(len(spheres))
	uniformBuffer, err := bp.system.CreateBufferWithData("objectCount",
		ToBytes([]uint32{objectCount}),
		wgpu.BufferUsageUniform|wgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}
	defer uniformBuffer.Release()

	// We need a custom dispatch since we have 4 buffers including uniform
	err = bp.dispatchWithUniform(objectCount, uniformBuffer)
	if err != nil {
		return nil, err
	}

	// Read back pair count
	countData, err := bp.system.ReadBuffer(bp.countBuffer)
	if err != nil {
		return nil, err
	}
	pairCount := toSlice[uint32](countData)[0]

	if pairCount == 0 {
		return nil, nil
	}

	// Clamp to max pairs
	if pairCount > bp.maxPairs {
		pairCount = bp.maxPairs
	}

	// Read back pairs
	pairData, err := bp.system.ReadBuffer(bp.pairBuffer)
	if err != nil {
		return nil, err
	}

	// Convert to pairs slice
	pairs := make([]CollisionPair, pairCount)
	rawPairs := toSlice[CollisionPair](pairData)
	copy(pairs, rawPairs[:pairCount])

	return pairs, nil
}

// dispatchWithUniform handles the 4-buffer dispatch for broad-phase
func (bp *BroadPhase) dispatchWithUniform(objectCount uint32, uniformBuffer *Buffer) error {
	device := bp.system.device
	queue := bp.system.queue

	// Create bind group layout for 4 bindings
	layoutDesc := wgpu.BindGroupLayoutDescriptor{
		Label: "broadphase_layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageCompute,
				Buffer: wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeReadOnlyStorage}},
			{Binding: 1, Visibility: wgpu.ShaderStageCompute,
				Buffer: wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeStorage}},
			{Binding: 2, Visibility: wgpu.ShaderStageCompute,
				Buffer: wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeStorage}},
			{Binding: 3, Visibility: wgpu.ShaderStageCompute,
				Buffer: wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	}
	layout, err := device.CreateBindGroupLayout(&layoutDesc)
	if err != nil {
		return err
	}
	defer layout.Release()

	bindGroup, err := device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "broadphase_bindgroup",
		Layout: layout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: bp.sphereBuffer.buffer, Size: bp.sphereBuffer.size},
			{Binding: 1, Buffer: bp.pairBuffer.buffer, Size: bp.pairBuffer.size},
			{Binding: 2, Buffer: bp.countBuffer.buffer, Size: bp.countBuffer.size},
			{Binding: 3, Buffer: uniformBuffer.buffer, Size: uniformBuffer.size},
		},
	})
	if err != nil {
		return err
	}
	defer bindGroup.Release()

	// Create pipeline layout
	pipelineLayout, err := device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:           "broadphase_pipeline_layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{layout},
	})
	if err != nil {
		return err
	}
	defer pipelineLayout.Release()

	// Create compute pipeline with explicit layout
	shaderModule, err := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "broadphase_shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: broadPhaseShader},
	})
	if err != nil {
		return err
	}
	defer shaderModule.Release()

	pipeline, err := device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
		Label:  "broadphase_pipeline",
		Layout: pipelineLayout,
		Compute: wgpu.ProgrammableStageDescriptor{
			Module:     shaderModule,
			EntryPoint: "main",
		},
	})
	if err != nil {
		return err
	}
	defer pipeline.Release()

	// Dispatch
	encoder, err := device.CreateCommandEncoder(nil)
	if err != nil {
		return err
	}

	pass := encoder.BeginComputePass(nil)
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	workgroups := (objectCount + 255) / 256
	pass.DispatchWorkgroups(workgroups, 1, 1)
	pass.End()
	pass.Release()

	commands, err := encoder.Finish(nil)
	if err != nil {
		return err
	}
	defer commands.Release()

	queue.Submit(commands)
	return nil
}

// Release frees GPU resources.
func (bp *BroadPhase) Release() {
	if bp.sphereBuffer != nil {
		bp.sphereBuffer.Release()
	}
	if bp.pairBuffer != nil {
		bp.pairBuffer.Release()
	}
	if bp.countBuffer != nil {
		bp.countBuffer.Release()
	}
}

// Helper to get unsafe pointer
func toPtr(data []byte) *byte {
	if len(data) == 0 {
		return nil
	}
	return &data[0]
}

// Helper to convert bytes to slice
func toSlice[T any](data []byte) []T {
	return wgpu.FromBytes[T](data)
}