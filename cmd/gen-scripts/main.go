package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type ScriptInfo struct {
	Name   string
	Fields []FieldInfo
}

type FieldInfo struct {
	Name     string
	Type     string
	JSONName string
}

func main() {
	sourceDir := "assets/scripts"
	outputDir := "internal/scripts"

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("❌ Source directory not found: %s\n", sourceDir)
		fmt.Println("   Create assets/scripts/ and add your script files there.")
		os.Exit(1)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Preserve doc.go if it doesn't exist
	docPath := filepath.Join(outputDir, "doc.go")
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		docContent := `// Package scripts contains game script components.
// Game-specific scripts can be placed in assets/scripts/ and will be
// copied here during build.
package scripts
`
		os.WriteFile(docPath, []byte(docContent), 0644)
	}

	// Find all .go files in source directory
	files, err := filepath.Glob(filepath.Join(sourceDir, "*.go"))
	if err != nil {
		fmt.Printf("❌ Failed to read source directory: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("⚠️  No script files found in %s\n", sourceDir)
		fmt.Println("   Add .go files to assets/scripts/ to generate scripts.")
		return
	}

	fmt.Println("🔧 Generating scripts from assets/scripts/...")

	generatedCount := 0
	skippedCount := 0
	for _, file := range files {
		result, err := processScript(file, outputDir)
		if err != nil {
			fmt.Printf("   ✗ %s: %v\n", filepath.Base(file), err)
		} else if result == "skipped" {
			skippedCount++
		} else {
			fmt.Printf("   ✓ %s\n", strings.TrimSuffix(filepath.Base(file), ".go"))
			generatedCount++
		}
	}

	if skippedCount > 0 {
		fmt.Printf("✅ Generated %d, skipped %d (cached) in %s\n", generatedCount, skippedCount, outputDir)
	} else {
		fmt.Printf("✅ Generated %d script(s) in %s\n", generatedCount, outputDir)
	}
}

func processScript(sourcePath, outputDir string) (string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	outputPath := filepath.Join(outputDir, filepath.Base(sourcePath))

	// Check if we need to regenerate (content-based hash)
	if !needsRegeneration(content, outputPath) {
		return "skipped", nil
	}

	script, err := parseScript(string(content))
	if err != nil {
		return "", err
	}

	if err := generateScriptFile(script, content, outputPath); err != nil {
		return "", err
	}

	return "generated", nil
}

func parseScript(content string) (*ScriptInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	var scriptInfo *ScriptInfo

	// Walk the AST to find struct declarations
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for type declarations
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			// Check if it's a struct type
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				scriptInfo = &ScriptInfo{
					Name:   typeSpec.Name.Name,
					Fields: []FieldInfo{},
				}

				// Extract fields
				for _, field := range structType.Fields.List {
					// Skip embedded fields (no names)
					if len(field.Names) == 0 {
						continue
					}

					for _, name := range field.Names {
						// Only export uppercase fields
						if !unicode.IsUpper(rune(name.Name[0])) {
							continue
						}

						fieldType := exprToString(field.Type)

						// Skip embedded structs with package qualifiers (except GameObjectRef)
						if strings.Contains(fieldType, ".") && fieldType != "engine.GameObjectRef" {
							continue
						}

						// Strip package prefix for known types
						cleanType := fieldType
						if fieldType == "engine.GameObjectRef" {
							cleanType = "GameObjectRef"
						}

						scriptInfo.Fields = append(scriptInfo.Fields, FieldInfo{
							Name:     name.Name,
							Type:     cleanType,
							JSONName: toSnakeCase(name.Name),
						})
					}
				}

				return false // Found our struct, stop walking
			}
		}
		return true
	})

	if scriptInfo == nil {
		return nil, fmt.Errorf("no struct definition found")
	}

	return scriptInfo, nil
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprToString(t.Len), exprToString(t.Elt))
	default:
		return "unknown"
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func generateScriptFile(script *ScriptInfo, sourceContent []byte, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	// Write original source
	if _, err := f.Write(sourceContent); err != nil {
		return fmt.Errorf("failed to write source content: %w", err)
	}

	// Generate boilerplate
	nameLower := toSnakeCase(script.Name)

	f.WriteString("\n// --- Generated boilerplate below ---\n\n")

	// Generate field types map if there are any GameObjectRef fields
	hasGameObjectRef := false
	for _, field := range script.Fields {
		if field.Type == "GameObjectRef" {
			hasGameObjectRef = true
			break
		}
	}

	if hasGameObjectRef {
		f.WriteString(fmt.Sprintf("var %sFieldTypes = map[string]string{\n", nameLower))
		for _, field := range script.Fields {
			if field.Type == "GameObjectRef" {
				f.WriteString(fmt.Sprintf("\t\"%s\": \"GameObjectRef\",\n", field.JSONName))
			}
		}
		f.WriteString("}\n\n")
		f.WriteString(fmt.Sprintf("func init() {\n\tengine.RegisterScriptWithMetadata(\"%s\", %sFactory, %sSerializer, %sApplier, %sFieldTypes)\n}\n\n",
			script.Name, nameLower, nameLower, nameLower, nameLower))
	} else {
		f.WriteString(fmt.Sprintf("func init() {\n\tengine.RegisterScriptWithApplier(\"%s\", %sFactory, %sSerializer, %sApplier)\n}\n\n",
			script.Name, nameLower, nameLower, nameLower))
	}

	// Factory function
	f.WriteString(fmt.Sprintf("func %sFactory(props map[string]any) engine.Component {\n", nameLower))
	f.WriteString(fmt.Sprintf("\tscript := &%s{}\n", script.Name))

	for _, field := range script.Fields {
		goType, conversion := getTypeConversion(field.Type)
		f.WriteString(fmt.Sprintf("\tif v, ok := props[\"%s\"].(%s); ok {\n", field.JSONName, goType))
		f.WriteString(fmt.Sprintf("\t\tscript.%s = %s\n\t}\n", field.Name, fmt.Sprintf(conversion, "v")))
	}

	f.WriteString("\treturn script\n}\n\n")

	// Serializer function
	f.WriteString(fmt.Sprintf("func %sSerializer(c engine.Component) map[string]any {\n", nameLower))
	if len(script.Fields) > 0 {
		f.WriteString(fmt.Sprintf("\ts, ok := c.(*%s)\n\tif !ok {\n\t\treturn nil\n\t}\n", script.Name))
	} else {
		f.WriteString(fmt.Sprintf("\t_, ok := c.(*%s)\n\tif !ok {\n\t\treturn nil\n\t}\n", script.Name))
	}
	f.WriteString("\treturn map[string]any{\n")

	for _, field := range script.Fields {
		// Special handling for GameObjectRef - serialize as UID
		if field.Type == "GameObjectRef" {
			f.WriteString(fmt.Sprintf("\t\t\"%s\": float64(s.%s.UID),\n", field.JSONName, field.Name))
		} else {
			f.WriteString(fmt.Sprintf("\t\t\"%s\": s.%s,\n", field.JSONName, field.Name))
		}
	}

	f.WriteString("\t}\n}\n\n")

	// Applier function for live property editing
	f.WriteString(fmt.Sprintf("func %sApplier(c engine.Component, propName string, value any) bool {\n", nameLower))
	if len(script.Fields) > 0 {
		f.WriteString(fmt.Sprintf("\ts, ok := c.(*%s)\n\tif !ok {\n\t\treturn false\n\t}\n", script.Name))
	} else {
		f.WriteString(fmt.Sprintf("\t_, ok := c.(*%s)\n\tif !ok {\n\t\treturn false\n\t}\n", script.Name))
	}
	f.WriteString("\tswitch propName {\n")

	for _, field := range script.Fields {
		goType, conversion := getTypeConversion(field.Type)
		f.WriteString(fmt.Sprintf("\tcase \"%s\":\n", field.JSONName))
		f.WriteString(fmt.Sprintf("\t\tif v, ok := value.(%s); ok {\n", goType))
		f.WriteString(fmt.Sprintf("\t\t\ts.%s = %s\n\t\t\treturn true\n\t\t}\n", field.Name, fmt.Sprintf(conversion, "v")))
	}

	f.WriteString("\t}\n\treturn false\n}\n")

	// Write hash to separate file for caching
	h := sha256.New()
	h.Write(sourceContent)
	hash := hex.EncodeToString(h.Sum(nil))
	hashPath := outputPath + ".hash"
	os.WriteFile(hashPath, []byte(hash), 0644)

	return nil
}

func getTypeConversion(fieldType string) (goType, conversion string) {
	switch fieldType {
	case "float32":
		return "float64", "float32(%s)"
	case "float64":
		return "float64", "%s"
	case "int":
		return "float64", "int(%s)"
	case "int32":
		return "float64", "int32(%s)"
	case "int64":
		return "float64", "int64(%s)"
	case "bool":
		return "bool", "%s"
	case "string":
		return "string", "%s"
	case "GameObjectRef":
		return "float64", "engine.GameObjectRef{UID: uint64(%s)}"
	default:
		return "any", "%s"
	}
}

func needsRegeneration(sourceContent []byte, outputPath string) bool {
	// Hash the source content
	h := sha256.New()
	h.Write(sourceContent)
	sourceHash := hex.EncodeToString(h.Sum(nil))

	// Check if output exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return true // Output doesn't exist, need to generate
	}

	// Read cached hash from .hash file
	hashPath := outputPath + ".hash"
	cachedHash, err := os.ReadFile(hashPath)
	if err != nil {
		return true // No hash file, regenerate
	}

	return string(cachedHash) != sourceHash
}
