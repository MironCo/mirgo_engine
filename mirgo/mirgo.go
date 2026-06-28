package mirgo

import "mirgo_engine/internal/game"

func Run() {
	g := game.New()
	g.Run(false)
}
