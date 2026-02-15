package scripts

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"
)

// ScoreManager demonstrates how to wire up UI buttons and text using GameObjectRef.
// Add this to a GameObject in your scene (can be empty GameObject).
// Drag UI elements onto the fields in the inspector.
type ScoreManager struct {
	engine.BaseComponent

	// Score tracking
	Score int

	// Drag UI GameObjects onto these fields in the inspector
	ScoreText   engine.GameObjectRef // GameObject with UIText showing score
	AddButton   engine.GameObjectRef // GameObject with UIButton to add score
	ResetButton engine.GameObjectRef // GameObject with UIButton to reset score

	// Internal references (set in Start)
	scoreText   *components.UIText
	addButton   *components.UIButton
	resetButton *components.UIButton
}

func (s *ScoreManager) Start() {
	g := s.GetGameObject()
	if g == nil || g.Scene == nil {
		fmt.Println("ScoreManager: No scene found")
		return
	}

	// Resolve score text reference
	if scoreObj := s.ScoreText.Get(g.Scene); scoreObj != nil {
		s.scoreText = engine.GetComponent[*components.UIText](scoreObj)
		if s.scoreText != nil {
			s.UpdateScoreDisplay()
			fmt.Printf("ScoreManager: Found score text '%s'\n", scoreObj.Name)
		} else {
			fmt.Printf("ScoreManager: '%s' doesn't have a UIText component!\n", scoreObj.Name)
		}
	} else if s.ScoreText.IsValid() {
		fmt.Println("ScoreManager: ScoreText reference is broken (GameObject not found)")
	} else {
		fmt.Println("ScoreManager: No ScoreText set (drag one in the inspector)")
	}

	// Resolve and wire up add button
	if addObj := s.AddButton.Get(g.Scene); addObj != nil {
		s.addButton = engine.GetComponent[*components.UIButton](addObj)
		if s.addButton != nil {
			s.addButton.OnClick.AddListener(s.OnAddScoreClicked)
			fmt.Printf("ScoreManager: Wired up Add Score button '%s'\n", addObj.Name)
		} else {
			fmt.Printf("ScoreManager: '%s' doesn't have a UIButton component!\n", addObj.Name)
		}
	} else if s.AddButton.IsValid() {
		fmt.Println("ScoreManager: AddButton reference is broken (GameObject not found)")
	} else {
		fmt.Println("ScoreManager: No AddButton set (drag one in the inspector)")
	}

	// Resolve and wire up reset button
	if resetObj := s.ResetButton.Get(g.Scene); resetObj != nil {
		s.resetButton = engine.GetComponent[*components.UIButton](resetObj)
		if s.resetButton != nil {
			s.resetButton.OnClick.AddListener(s.OnResetClicked)
			fmt.Printf("ScoreManager: Wired up Reset button '%s'\n", resetObj.Name)
		} else {
			fmt.Printf("ScoreManager: '%s' doesn't have a UIButton component!\n", resetObj.Name)
		}
	} else if s.ResetButton.IsValid() {
		fmt.Println("ScoreManager: ResetButton reference is broken (GameObject not found)")
	} else {
		fmt.Println("ScoreManager: No ResetButton set (drag one in the inspector)")
	}

	fmt.Printf("ScoreManager initialized - Score: %d\n", s.Score)
}

// OnAddScoreClicked is called when the add score button is clicked
func (s *ScoreManager) OnAddScoreClicked() {
	s.Score += 10
	s.UpdateScoreDisplay()
	fmt.Printf("Score increased! New score: %d\n", s.Score)
}

// OnResetClicked is called when the reset button is clicked
func (s *ScoreManager) OnResetClicked() {
	s.Score = 0
	s.UpdateScoreDisplay()
	fmt.Println("Score reset!")
}

// UpdateScoreDisplay updates the score text UI
func (s *ScoreManager) UpdateScoreDisplay() {
	if s.scoreText != nil {
		s.scoreText.Text = fmt.Sprintf("Score: %d", s.Score)
	}
}
