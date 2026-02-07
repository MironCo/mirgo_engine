package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const scriptsDir = "internal/components/scripts"

const tmpl = `package scripts

import "test3d/internal/engine"

type {{.Name}} struct {
	engine.BaseComponent
	Speed float32
}

func (s *{{.Name}}) Update(deltaTime float32) {
	g := s.GetGameObject()
	if g == nil {
		return
	}
	// TODO: implement behavior
}

func init() {
	engine.RegisterScript("{{.Name}}", {{.Lower}}Factory, {{.Lower}}Serializer)
}

func {{.Lower}}Factory(props map[string]any) engine.Component {
	speed := float32(1)
	if v, ok := props["speed"].(float64); ok {
		speed = float32(v)
	}
	return &{{.Name}}{Speed: speed}
}

func {{.Lower}}Serializer(c engine.Component) map[string]any {
	s, ok := c.(*{{.Name}})
	if !ok {
		return nil
	}
	return map[string]any{
		"speed": s.Speed,
	}
}
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run ./cmd/newscript <ScriptName>\n")
		fmt.Fprintf(os.Stderr, "Example: go run ./cmd/newscript EnemyChaser\n")
		os.Exit(1)
	}

	name := os.Args[1]
	if name == "" || !unicode.IsUpper(rune(name[0])) {
		fmt.Fprintf(os.Stderr, "Error: script name must start with an uppercase letter\n")
		os.Exit(1)
	}

	lower := string(unicode.ToLower(rune(name[0]))) + name[1:]
	filename := toSnakeCase(name) + ".go"
	outPath := filepath.Join(scriptsDir, filename)

	if _, err := os.Stat(outPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s already exists\n", outPath)
		os.Exit(1)
	}

	content := tmpl
	content = strings.ReplaceAll(content, "{{.Name}}", name)
	content = strings.ReplaceAll(content, "{{.Lower}}", lower)

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s\n", outPath)
	fmt.Printf("Script \"%s\" registered. Add it to a scene object:\n\n", name)
	fmt.Printf("  {\n")
	fmt.Printf("    \"type\": \"Script\",\n")
	fmt.Printf("    \"name\": \"%s\",\n", name)
	fmt.Printf("    \"props\": { \"speed\": 1.0 }\n")
	fmt.Printf("  }\n")
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
