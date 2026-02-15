package scripts

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"
)

// GameObjectRefExample demonstrates Unity-style GameObject references.
// Instead of finding GameObjects by name, you can drag-and-drop them
// in the inspector for type-safe, serializable references.
type GameObjectRefExample struct {
	engine.BaseComponent

	// Drag a GameObject with a UIButton onto this field in the inspector
	TargetButton engine.GameObjectRef

	// Drag a GameObject with UIText onto this field
	TargetText engine.GameObjectRef

	clickCount int
}

func (g *GameObjectRefExample) Start() {
	// Resolve the button reference
	if buttonObj := g.TargetButton.Get(g.GetGameObject().Scene); buttonObj != nil {
		if button := engine.GetComponent[*components.UIButton](buttonObj); button != nil {
			// Wire up the click handler
			button.OnClick.AddListener(g.OnButtonClicked)
			fmt.Printf("GameObjectRefExample: Wired up button '%s'\n", buttonObj.Name)
		} else {
			fmt.Printf("GameObjectRefExample: '%s' doesn't have a UIButton component!\n", buttonObj.Name)
		}
	} else if g.TargetButton.IsValid() {
		fmt.Println("GameObjectRefExample: TargetButton reference is broken (GameObject not found)")
	} else {
		fmt.Println("GameObjectRefExample: No TargetButton set (drag one in the inspector)")
	}

	// Resolve the text reference
	if textObj := g.TargetText.Get(g.GetGameObject().Scene); textObj != nil {
		if text := engine.GetComponent[*components.UIText](textObj); text != nil {
			text.Text = "Ready! Click the button."
			fmt.Printf("GameObjectRefExample: Found text '%s'\n", textObj.Name)
		}
	}
}

func (g *GameObjectRefExample) OnButtonClicked() {
	g.clickCount++
	fmt.Printf("Button clicked! Count: %d\n", g.clickCount)

	// Update the text if we have a reference
	if textObj := g.TargetText.Get(g.GetGameObject().Scene); textObj != nil {
		if text := engine.GetComponent[*components.UIText](textObj); text != nil {
			text.Text = fmt.Sprintf("Clicks: %d", g.clickCount)
		}
	}
}
