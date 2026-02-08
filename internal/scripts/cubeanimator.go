package scripts

import (
	"test3d/internal/components"
	"test3d/internal/engine"
)

func init() {
	engine.RegisterScript("CubeAnimatorScript", cubeAnimatorFactory, cubeAnimatorSerializer)
}

func cubeAnimatorFactory(props map[string]any) engine.Component {
	getFloat := func(key string, fallback float32) float32 {
		if v, ok := props[key].(float64); ok {
			return float32(v)
		}
		return fallback
	}

	return &components.CubeAnimator{
		RotationSpeed:  getFloat("rotationSpeed", 45),
		MovementRadius: getFloat("movementRadius", 0),
		MovementSpeed:  getFloat("movementSpeed", 1),
		Phase:          getFloat("phase", 0),
	}
}

func cubeAnimatorSerializer(c engine.Component) map[string]any {
	ca, ok := c.(*components.CubeAnimator)
	if !ok {
		return nil
	}
	return map[string]any{
		"rotationSpeed":  ca.RotationSpeed,
		"movementRadius": ca.MovementRadius,
		"movementSpeed":  ca.MovementSpeed,
		"phase":          ca.Phase,
	}
}
