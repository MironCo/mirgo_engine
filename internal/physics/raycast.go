package physics

import (
	"math"
	"test3d/internal/components"
	"test3d/internal/engine"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type RaycastHit struct {
	GameObject *engine.GameObject
	Point      rl.Vector3
	Normal     rl.Vector3
	Distance   float32
}

// Raycast checks for intersection with all collidable objects and returns the closest hit
func (p *PhysicsWorld) Raycast(origin, direction rl.Vector3, maxDistance float32) (RaycastHit, bool) {
	direction = rl.Vector3Normalize(direction)
	var closestHit RaycastHit
	closestHit.Distance = maxDistance
	hit := false

	// Check all object lists
	allObjects := make([]*engine.GameObject, 0, len(p.Objects)+len(p.Kinematics)+len(p.Statics))
	allObjects = append(allObjects, p.Objects...)
	allObjects = append(allObjects, p.Kinematics...)
	allObjects = append(allObjects, p.Statics...)

	for _, obj := range allObjects {
		// Check box collider
		if box := engine.GetComponent[*components.BoxCollider](obj); box != nil {
			if hitInfo, ok := raycastBox(origin, direction, box, maxDistance); ok {
				if hitInfo.Distance < closestHit.Distance {
					closestHit = hitInfo
					closestHit.GameObject = obj
					hit = true
				}
			}
		}
		// Check sphere collider
		if sphere := engine.GetComponent[*components.SphereCollider](obj); sphere != nil {
			if hitInfo, ok := raycastSphere(origin, direction, sphere, maxDistance); ok {
				if hitInfo.Distance < closestHit.Distance {
					closestHit = hitInfo
					closestHit.GameObject = obj
					hit = true
				}
			}
		}
	}

	return closestHit, hit
}

func raycastBox(origin, direction rl.Vector3, box *components.BoxCollider, maxDistance float32) (RaycastHit, bool) {
	center := box.GetCenter()
	// Use world-scaled size with absolute values to handle negative sizes
	worldSize := box.GetWorldSize()
	halfSize := rl.Vector3{X: abs(worldSize.X) / 2, Y: abs(worldSize.Y) / 2, Z: abs(worldSize.Z) / 2}

	min := rl.Vector3{X: center.X - halfSize.X, Y: center.Y - halfSize.Y, Z: center.Z - halfSize.Z}
	max := rl.Vector3{X: center.X + halfSize.X, Y: center.Y + halfSize.Y, Z: center.Z + halfSize.Z}

	var tmin, tmax float32

	// X slab
	if direction.X != 0 {
		t1 := (min.X - origin.X) / direction.X
		t2 := (max.X - origin.X) / direction.X
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tmin = t1
		tmax = t2
	} else if origin.X < min.X || origin.X > max.X {
		return RaycastHit{}, false
	} else {
		tmin = -1e30
		tmax = 1e30
	}

	// Y slab
	if direction.Y != 0 {
		t1 := (min.Y - origin.Y) / direction.Y
		t2 := (max.Y - origin.Y) / direction.Y
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tmin {
			tmin = t1
		}
		if t2 < tmax {
			tmax = t2
		}
	} else if origin.Y < min.Y || origin.Y > max.Y {
		return RaycastHit{}, false
	}

	if tmin > tmax {
		return RaycastHit{}, false
	}

	// Z slab
	if direction.Z != 0 {
		t1 := (min.Z - origin.Z) / direction.Z
		t2 := (max.Z - origin.Z) / direction.Z
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tmin {
			tmin = t1
		}
		if t2 < tmax {
			tmax = t2
		}
	} else if origin.Z < min.Z || origin.Z > max.Z {
		return RaycastHit{}, false
	}

	if tmin > tmax || tmax < 0 || tmin > maxDistance {
		return RaycastHit{}, false
	}

	t := tmin
	if t < 0 {
		t = tmax
	}
	if t < 0 || t > maxDistance {
		return RaycastHit{}, false
	}

	point := rl.Vector3Add(origin, rl.Vector3Scale(direction, t))

	// Calculate normal based on which face was hit
	var normal rl.Vector3
	epsilon := float32(0.001)
	if abs(point.X-min.X) < epsilon {
		normal = rl.Vector3{X: -1, Y: 0, Z: 0}
	} else if abs(point.X-max.X) < epsilon {
		normal = rl.Vector3{X: 1, Y: 0, Z: 0}
	} else if abs(point.Y-min.Y) < epsilon {
		normal = rl.Vector3{X: 0, Y: -1, Z: 0}
	} else if abs(point.Y-max.Y) < epsilon {
		normal = rl.Vector3{X: 0, Y: 1, Z: 0}
	} else if abs(point.Z-min.Z) < epsilon {
		normal = rl.Vector3{X: 0, Y: 0, Z: -1}
	} else {
		normal = rl.Vector3{X: 0, Y: 0, Z: 1}
	}

	return RaycastHit{Point: point, Normal: normal, Distance: t}, true
}

func raycastSphere(origin, direction rl.Vector3, sphere *components.SphereCollider, maxDistance float32) (RaycastHit, bool) {
	center := sphere.GetCenter()
	radius := sphere.Radius

	oc := rl.Vector3Subtract(origin, center)
	a := rl.Vector3DotProduct(direction, direction)
	b := 2.0 * rl.Vector3DotProduct(oc, direction)
	c := rl.Vector3DotProduct(oc, oc) - radius*radius

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return RaycastHit{}, false
	}

	t := (-b - float32(math.Sqrt(float64(discriminant)))) / (2 * a)
	if t < 0 {
		t = (-b + float32(math.Sqrt(float64(discriminant)))) / (2 * a)
	}
	if t < 0 || t > maxDistance {
		return RaycastHit{}, false
	}

	point := rl.Vector3Add(origin, rl.Vector3Scale(direction, t))
	normal := rl.Vector3Normalize(rl.Vector3Subtract(point, center))

	return RaycastHit{Point: point, Normal: normal, Distance: t}, true
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
