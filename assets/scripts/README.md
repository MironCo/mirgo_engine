# Scripts

This directory contains clean script implementations without boilerplate.

## How It Works

1. Write your scripts here with just the core logic
2. Run `make build` or `make run` - the build process automatically generates the boilerplate
3. Generated files appear in `internal/scripts/` (git-ignored)

## Example Script

```go
package scripts

import "test3d/internal/engine"

type MyScript struct {
	engine.BaseComponent
	Speed float32
	Health int
}

func (m *MyScript) Update(deltaTime float32) {
	// Your logic here
}
```

The generator will automatically create:
- `init()` function with `engine.RegisterScript()`
- Factory function that parses JSON properties
- Serializer function for saving

All exported fields (capitalized) are automatically serialized with snake_case JSON names.
