package fonts

import "embed"

//go:embed *.ttf
var FS embed.FS

func MustLoad(name string) []byte {
	data, err := FS.ReadFile(name)
	if err != nil {
		panic("missing embedded font: " + name)
	}
	return data
}
