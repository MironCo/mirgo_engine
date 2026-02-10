package components

import (
	"math"
	"test3d/internal/engine"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	engine.RegisterComponent("MeshCollider", func() engine.Serializable {
		return NewMeshCollider()
	})
}

// Triangle represents a single triangle with precomputed normal
type Triangle struct {
	V0, V1, V2 rl.Vector3
	Normal     rl.Vector3
}

// AABB represents an axis-aligned bounding box
type AABB struct {
	Min, Max rl.Vector3
}

// BVHNode is a node in the bounding volume hierarchy
type BVHNode struct {
	Bounds    AABB
	Left      *BVHNode
	Right     *BVHNode
	Triangles []int // indices into the triangle array (only for leaf nodes)
}

// MeshCollider provides collision detection against mesh triangles.
// This is for STATIC geometry only - moving the object won't update the collider.
type MeshCollider struct {
	engine.BaseComponent
	Triangles []Triangle
	Root      *BVHNode
	built     bool
}

// NewMeshCollider creates a mesh collider (must call BuildFromModel after)
func NewMeshCollider() *MeshCollider {
	return &MeshCollider{}
}

// BuildFromModel extracts triangles from a raylib Model and builds the BVH
func (m *MeshCollider) BuildFromModel(model rl.Model) {
	g := m.GetGameObject()
	if g == nil {
		return
	}

	// Get world transform
	worldPos := g.WorldPosition()
	worldRot := g.WorldRotation()
	worldScale := g.WorldScale()

	// Build transform matrix
	scaleMatrix := rl.MatrixScale(worldScale.X, worldScale.Y, worldScale.Z)
	rotX := rl.MatrixRotateX(worldRot.X * rl.Deg2rad)
	rotY := rl.MatrixRotateY(worldRot.Y * rl.Deg2rad)
	rotZ := rl.MatrixRotateZ(worldRot.Z * rl.Deg2rad)
	rotMatrix := rl.MatrixMultiply(rl.MatrixMultiply(rotX, rotY), rotZ)
	transMatrix := rl.MatrixTranslate(worldPos.X, worldPos.Y, worldPos.Z)
	transform := rl.MatrixMultiply(rl.MatrixMultiply(scaleMatrix, rotMatrix), transMatrix)

	// Extract triangles from all meshes
	m.Triangles = nil
	meshes := unsafe.Slice(model.Meshes, model.MeshCount)

	for _, mesh := range meshes {
		vertices := unsafe.Slice(mesh.Vertices, mesh.VertexCount*3)

		if mesh.Indices != nil {
			// Indexed mesh
			indices := unsafe.Slice(mesh.Indices, mesh.TriangleCount*3)
			for i := int32(0); i < mesh.TriangleCount; i++ {
				i0 := indices[i*3+0]
				i1 := indices[i*3+1]
				i2 := indices[i*3+2]

				v0 := rl.Vector3{X: vertices[i0*3+0], Y: vertices[i0*3+1], Z: vertices[i0*3+2]}
				v1 := rl.Vector3{X: vertices[i1*3+0], Y: vertices[i1*3+1], Z: vertices[i1*3+2]}
				v2 := rl.Vector3{X: vertices[i2*3+0], Y: vertices[i2*3+1], Z: vertices[i2*3+2]}

				// Transform to world space
				v0 = rl.Vector3Transform(v0, transform)
				v1 = rl.Vector3Transform(v1, transform)
				v2 = rl.Vector3Transform(v2, transform)

				// Compute normal
				edge1 := rl.Vector3Subtract(v1, v0)
				edge2 := rl.Vector3Subtract(v2, v0)
				normal := rl.Vector3CrossProduct(edge1, edge2)
				normal = rl.Vector3Normalize(normal)

				m.Triangles = append(m.Triangles, Triangle{V0: v0, V1: v1, V2: v2, Normal: normal})
			}
		} else {
			// Non-indexed mesh (every 3 vertices = 1 triangle)
			triCount := mesh.VertexCount / 3
			for i := int32(0); i < triCount; i++ {
				v0 := rl.Vector3{X: vertices[i*9+0], Y: vertices[i*9+1], Z: vertices[i*9+2]}
				v1 := rl.Vector3{X: vertices[i*9+3], Y: vertices[i*9+4], Z: vertices[i*9+5]}
				v2 := rl.Vector3{X: vertices[i*9+6], Y: vertices[i*9+7], Z: vertices[i*9+8]}

				// Transform to world space
				v0 = rl.Vector3Transform(v0, transform)
				v1 = rl.Vector3Transform(v1, transform)
				v2 = rl.Vector3Transform(v2, transform)

				// Compute normal
				edge1 := rl.Vector3Subtract(v1, v0)
				edge2 := rl.Vector3Subtract(v2, v0)
				normal := rl.Vector3CrossProduct(edge1, edge2)
				normal = rl.Vector3Normalize(normal)

				m.Triangles = append(m.Triangles, Triangle{V0: v0, V1: v1, V2: v2, Normal: normal})
			}
		}
	}

	// Build BVH
	m.buildBVH()
	m.built = true
}

// buildBVH constructs a bounding volume hierarchy for fast queries
func (m *MeshCollider) buildBVH() {
	if len(m.Triangles) == 0 {
		return
	}

	// Create indices for all triangles
	indices := make([]int, len(m.Triangles))
	for i := range indices {
		indices[i] = i
	}

	m.Root = m.buildBVHNode(indices, 0)
}

func (m *MeshCollider) buildBVHNode(indices []int, depth int) *BVHNode {
	node := &BVHNode{}

	// Compute bounds for all triangles in this node
	node.Bounds = m.computeBounds(indices)

	// If few triangles or max depth, make leaf
	if len(indices) <= 4 || depth > 20 {
		node.Triangles = indices
		return node
	}

	// Find longest axis
	size := rl.Vector3Subtract(node.Bounds.Max, node.Bounds.Min)
	axis := 0
	if size.Y > size.X {
		axis = 1
	}
	if size.Z > getAxisValue(size, axis) {
		axis = 2
	}

	// Sort by centroid on longest axis
	mid := m.partitionTriangles(indices, axis)

	if mid == 0 || mid == len(indices) {
		// Couldn't split, make leaf
		node.Triangles = indices
		return node
	}

	node.Left = m.buildBVHNode(indices[:mid], depth+1)
	node.Right = m.buildBVHNode(indices[mid:], depth+1)

	return node
}

func (m *MeshCollider) computeBounds(indices []int) AABB {
	bounds := AABB{
		Min: rl.Vector3{X: math.MaxFloat32, Y: math.MaxFloat32, Z: math.MaxFloat32},
		Max: rl.Vector3{X: -math.MaxFloat32, Y: -math.MaxFloat32, Z: -math.MaxFloat32},
	}

	for _, idx := range indices {
		tri := &m.Triangles[idx]
		bounds.Min = vector3Min(bounds.Min, tri.V0)
		bounds.Min = vector3Min(bounds.Min, tri.V1)
		bounds.Min = vector3Min(bounds.Min, tri.V2)
		bounds.Max = vector3Max(bounds.Max, tri.V0)
		bounds.Max = vector3Max(bounds.Max, tri.V1)
		bounds.Max = vector3Max(bounds.Max, tri.V2)
	}

	return bounds
}

func (m *MeshCollider) partitionTriangles(indices []int, axis int) int {
	// Find median centroid
	center := float32(0)
	for _, idx := range indices {
		tri := &m.Triangles[idx]
		centroid := rl.Vector3Scale(rl.Vector3Add(rl.Vector3Add(tri.V0, tri.V1), tri.V2), 1.0/3.0)
		center += getAxisValue(centroid, axis)
	}
	center /= float32(len(indices))

	// Partition around median
	left := 0
	right := len(indices) - 1
	for left <= right {
		tri := &m.Triangles[indices[left]]
		centroid := rl.Vector3Scale(rl.Vector3Add(rl.Vector3Add(tri.V0, tri.V1), tri.V2), 1.0/3.0)
		if getAxisValue(centroid, axis) < center {
			left++
		} else {
			indices[left], indices[right] = indices[right], indices[left]
			right--
		}
	}
	return left
}

func getAxisValue(v rl.Vector3, axis int) float32 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

func vector3Min(a, b rl.Vector3) rl.Vector3 {
	return rl.Vector3{
		X: float32(math.Min(float64(a.X), float64(b.X))),
		Y: float32(math.Min(float64(a.Y), float64(b.Y))),
		Z: float32(math.Min(float64(a.Z), float64(b.Z))),
	}
}

func vector3Max(a, b rl.Vector3) rl.Vector3 {
	return rl.Vector3{
		X: float32(math.Max(float64(a.X), float64(b.X))),
		Y: float32(math.Max(float64(a.Y), float64(b.Y))),
		Z: float32(math.Max(float64(a.Z), float64(b.Z))),
	}
}

// SphereIntersect tests if a sphere intersects the mesh and returns push-out vector
func (m *MeshCollider) SphereIntersect(center rl.Vector3, radius float32) (bool, rl.Vector3) {
	if !m.built || m.Root == nil {
		return false, rl.Vector3{}
	}

	// Expand sphere to AABB for BVH query
	sphereAABB := AABB{
		Min: rl.Vector3{X: center.X - radius, Y: center.Y - radius, Z: center.Z - radius},
		Max: rl.Vector3{X: center.X + radius, Y: center.Y + radius, Z: center.Z + radius},
	}

	// Find all potentially colliding triangles
	candidates := m.queryBVH(m.Root, sphereAABB)

	// Test each triangle
	var totalPush rl.Vector3
	hit := false

	for _, idx := range candidates {
		tri := &m.Triangles[idx]
		if collides, push := sphereTriangleIntersect(center, radius, tri); collides {
			// Accumulate push vectors (take the largest in each direction)
			if math.Abs(float64(push.X)) > math.Abs(float64(totalPush.X)) {
				totalPush.X = push.X
			}
			if math.Abs(float64(push.Y)) > math.Abs(float64(totalPush.Y)) {
				totalPush.Y = push.Y
			}
			if math.Abs(float64(push.Z)) > math.Abs(float64(totalPush.Z)) {
				totalPush.Z = push.Z
			}
			hit = true
		}
	}

	return hit, totalPush
}

func (m *MeshCollider) queryBVH(node *BVHNode, query AABB) []int {
	if node == nil {
		return nil
	}

	// Check if query intersects this node's bounds
	if !aabbIntersects(node.Bounds, query) {
		return nil
	}

	// If leaf, return triangles
	if node.Triangles != nil {
		return node.Triangles
	}

	// Otherwise, recurse
	left := m.queryBVH(node.Left, query)
	right := m.queryBVH(node.Right, query)
	return append(left, right...)
}

func aabbIntersects(a, b AABB) bool {
	return a.Min.X <= b.Max.X && a.Max.X >= b.Min.X &&
		a.Min.Y <= b.Max.Y && a.Max.Y >= b.Min.Y &&
		a.Min.Z <= b.Max.Z && a.Max.Z >= b.Min.Z
}

// sphereTriangleIntersect tests sphere vs triangle and returns push vector
func sphereTriangleIntersect(center rl.Vector3, radius float32, tri *Triangle) (bool, rl.Vector3) {
	// Find closest point on triangle to sphere center
	closest := closestPointOnTriangle(center, tri.V0, tri.V1, tri.V2)

	// Check if closest point is within sphere
	diff := rl.Vector3Subtract(center, closest)
	distSq := rl.Vector3DotProduct(diff, diff)
	radiusSq := radius * radius

	if distSq >= radiusSq {
		return false, rl.Vector3{}
	}

	// Calculate push-out vector
	dist := float32(math.Sqrt(float64(distSq)))
	if dist < 0.0001 {
		// Center is on triangle, push along normal
		return true, rl.Vector3Scale(tri.Normal, radius)
	}

	// Push out along the direction from closest point to center
	pushDir := rl.Vector3Scale(diff, 1.0/dist)
	penetration := radius - dist
	return true, rl.Vector3Scale(pushDir, penetration)
}

// closestPointOnTriangle finds the closest point on a triangle to point p
func closestPointOnTriangle(p, a, b, c rl.Vector3) rl.Vector3 {
	// Check if P in vertex region outside A
	ab := rl.Vector3Subtract(b, a)
	ac := rl.Vector3Subtract(c, a)
	ap := rl.Vector3Subtract(p, a)

	d1 := rl.Vector3DotProduct(ab, ap)
	d2 := rl.Vector3DotProduct(ac, ap)
	if d1 <= 0 && d2 <= 0 {
		return a // barycentric coordinates (1,0,0)
	}

	// Check if P in vertex region outside B
	bp := rl.Vector3Subtract(p, b)
	d3 := rl.Vector3DotProduct(ab, bp)
	d4 := rl.Vector3DotProduct(ac, bp)
	if d3 >= 0 && d4 <= d3 {
		return b // barycentric coordinates (0,1,0)
	}

	// Check if P in edge region of AB
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		v := d1 / (d1 - d3)
		return rl.Vector3Add(a, rl.Vector3Scale(ab, v)) // barycentric coordinates (1-v,v,0)
	}

	// Check if P in vertex region outside C
	cp := rl.Vector3Subtract(p, c)
	d5 := rl.Vector3DotProduct(ab, cp)
	d6 := rl.Vector3DotProduct(ac, cp)
	if d6 >= 0 && d5 <= d6 {
		return c // barycentric coordinates (0,0,1)
	}

	// Check if P in edge region of AC
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		w := d2 / (d2 - d6)
		return rl.Vector3Add(a, rl.Vector3Scale(ac, w)) // barycentric coordinates (1-w,0,w)
	}

	// Check if P in edge region of BC
	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return rl.Vector3Add(b, rl.Vector3Scale(rl.Vector3Subtract(c, b), w)) // barycentric coordinates (0,1-w,w)
	}

	// P inside face region
	denom := 1.0 / (va + vb + vc)
	v := vb * denom
	w := vc * denom
	return rl.Vector3Add(a, rl.Vector3Add(rl.Vector3Scale(ab, v), rl.Vector3Scale(ac, w)))
}

// IsBuilt returns true if the BVH has been built
func (m *MeshCollider) IsBuilt() bool {
	return m.built
}

// TriangleCount returns the number of triangles in the collider
func (m *MeshCollider) TriangleCount() int {
	return len(m.Triangles)
}

// GetBounds returns the AABB of the entire mesh collider
func (m *MeshCollider) GetBounds() AABB {
	if m.Root == nil {
		return AABB{}
	}
	return m.Root.Bounds
}

// TypeName implements engine.Serializable
func (m *MeshCollider) TypeName() string {
	return "MeshCollider"
}

// Serialize implements engine.Serializable
func (m *MeshCollider) Serialize() map[string]any {
	return map[string]any{"type": "MeshCollider"}
}

// Deserialize implements engine.Serializable
func (m *MeshCollider) Deserialize(data map[string]any) {
	// MeshCollider rebuilds from ModelRenderer, nothing to deserialize
}
