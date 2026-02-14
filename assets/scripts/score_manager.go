package scripts

import (
	"fmt"
	"test3d/internal/components"
	"test3d/internal/engine"
)

// ScoreManager demonstrates how to wire up UI buttons and text.
// Add this to a GameObject in your scene (can be empty GameObject).
// It will find UI elements by name and wire them up.
type ScoreManager struct {
	engine.BaseComponent

	// Score tracking
	Score int

	// UI element names to find
	ScoreTextName   string // Name of GameObject with UIText showing score
	AddButtonName   string // Name of GameObject with UIButton to add score
	ResetButtonName string // Name of GameObject with UIButton to reset score

	// Internal references (set in Start)
	scoreText   *components.UIText
	addButton   *components.UIButton
	resetButton *components.UIButton
}

func (s *ScoreManager) Start() {
	// Set defaults
	if s.ScoreTextName == "" {
		s.ScoreTextName = "ScoreText"
	}
	if s.AddButtonName == "" {
		s.AddButtonName = "AddScoreButton"
	}
	if s.ResetButtonName == "" {
		s.ResetButtonName = "ResetScoreButton"
	}

	// Find UI elements by name
	g := s.GetGameObject()
	if g == nil || g.Scene == nil {
		fmt.Println("ScoreManager: No scene found")
		return
	}

	// Find score text
	if scoreObj := g.Scene.FindByName(s.ScoreTextName); scoreObj != nil {
		s.scoreText = engine.GetComponent[*components.UIText](scoreObj)
		if s.scoreText != nil {
			s.UpdateScoreDisplay()
		}
	}

	// Find and wire up add button
	if addObj := g.Scene.FindByName(s.AddButtonName); addObj != nil {
		s.addButton = engine.GetComponent[*components.UIButton](addObj)
		if s.addButton != nil {
			s.addButton.OnClick.AddListener(s.OnAddScoreClicked)
			fmt.Printf("ScoreManager: Wired up Add Score button (%s)\n", s.AddButtonName)
		}
	}

	// Find and wire up reset button
	if resetObj := g.Scene.FindByName(s.ResetButtonName); resetObj != nil {
		s.resetButton = engine.GetComponent[*components.UIButton](resetObj)
		if s.resetButton != nil {
			s.resetButton.OnClick.AddListener(s.OnResetClicked)
			fmt.Printf("ScoreManager: Wired up Reset button (%s)\n", s.ResetButtonName)
		}
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
