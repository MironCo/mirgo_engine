package main

import (
	"os"
	"test3d/internal/game"
)

func main() {
	restoreEditor := len(os.Args) > 1 && os.Args[1] == "--restore-editor"
	g := game.New()
	g.Run(restoreEditor)
}
