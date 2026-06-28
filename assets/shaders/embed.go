package shaders

import "embed"

//go:embed *.vs *.fs
var FS embed.FS

func MustLoad(name string) string {
	data, err := FS.ReadFile(name)
	if err != nil {
		panic("missing embedded shader: " + name)
	}
	return string(data)
}
