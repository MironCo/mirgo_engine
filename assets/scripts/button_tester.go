package scripts

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"
)

// ButtonTester demonstrates a simple button click handler.
// This is a minimal example showing how to wire up buttons.
type ButtonTester struct {
	engine.BaseComponent

	clickCount int
}

func (b *ButtonTester) Start() {
	g := b.GetGameObject()
	if g == nil || g.Scene == nil {
		return
	}

	// Find the TestButton
	if buttonObj := g.Scene.FindByName("TestButton"); buttonObj != nil {
		if button := engine.GetComponent[*components.UIButton](buttonObj); button != nil {
			// Wire up multiple listeners to demonstrate Unity-style events
			button.OnClick.AddListener(b.OnButtonClicked)
			button.OnClick.AddListener(b.IncrementCounter)
			button.OnClick.AddListener(func() {
				fmt.Println("  -> Lambda function executed!")
			})

			fmt.Println("ButtonTester: Successfully wired up TestButton with 3 listeners")
		}
	}
}

func (b *ButtonTester) OnButtonClicked() {
	fmt.Println("🎉 Button was clicked!")
}

func (b *ButtonTester) IncrementCounter() {
	b.clickCount++
	fmt.Printf("  -> Total clicks: %d\n", b.clickCount)
}
