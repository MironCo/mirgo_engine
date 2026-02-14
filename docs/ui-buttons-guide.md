# UI Buttons Guide

## Overview

Buttons in Mirgo Engine work through the `UIButton` component combined with scripts that wire up the `OnClick` callback.

## Quick Start

### 1. Create Button UI Elements

In your scene JSON or via the editor:

```json
{
  "uid": 2001,
  "name": "AddScoreButton",
  "components": [
    {
      "type": "RectTransform",
      "anchoredPosition": [100, -100],
      "sizeDelta": [150, 40],
      "anchorMin": [0, 1],
      "anchorMax": [0, 1],
      "pivot": [0, 1]
    },
    {
      "type": "UIButton",
      "normalColor": [60, 60, 70, 255],
      "hoverColor": [80, 80, 95, 255],
      "pressedColor": [100, 100, 120, 255]
    }
  ],
  "children": [
    {
      "uid": 2002,
      "name": "ButtonText",
      "components": [
        {
          "type": "RectTransform",
          "anchoredPosition": [0, 0],
          "sizeDelta": [150, 40]
        },
        {
          "type": "UIText",
          "text": "+10 Points",
          "fontSize": 20,
          "alignment": 1,
          "color": [255, 255, 255, 255]
        }
      ]
    }
  ]
}
```

### 2. Create a Script to Handle Button Clicks

See `internal/scripts/score_manager.go` for a complete example. Key pattern:

```go
type MyScript struct {
    engine.BaseComponent

    ButtonName string
    button *components.UIButton
}

func (s *MyScript) Start() {
    g := s.GetGameObject()
    if buttonObj := g.Scene.FindByName(s.ButtonName); buttonObj != nil {
        s.button = engine.GetComponent[*components.UIButton](buttonObj)
        if s.button != nil {
            // Unity-style: Add listener to the event
            s.button.OnClick.AddListener(s.OnButtonClicked)

            // You can add multiple listeners!
            s.button.OnClick.AddListener(s.PlayClickSound)
            s.button.OnClick.AddListener(s.LogClick)
        }
    }
}

func (s *MyScript) OnButtonClicked() {
    fmt.Println("Button was clicked!")
}

func (s *MyScript) PlayClickSound() {
    // Play sound effect
}

func (s *MyScript) LogClick() {
    // Log analytics
}
```

### 3. Add Script to Scene

Add your script component to a GameObject (can be an empty GameObject):

```json
{
  "uid": 3000,
  "name": "GameManager",
  "components": [
    {
      "type": "script",
      "script": "MyScript",
      "button_name": "AddScoreButton"
    }
  ]
}
```

## How It Works

1. **UICanvas.Update()** is called every frame for all canvases in the scene
2. Canvas calls **UIButton.HandleInput()** for all buttons
3. HandleInput tracks hover/pressed state and detects clicks
4. When clicked, button calls **OnClick.Invoke()** which triggers all listeners
5. All registered callback functions execute in order

## Unity-Style Events

Buttons use `engine.Event` which is a multi-cast delegate system like Unity's UnityEvent:

```go
// Add multiple listeners to one button
button.OnClick.AddListener(func1)
button.OnClick.AddListener(func2)
button.OnClick.AddListener(func3)

// All three functions will be called when button is clicked

// Clear all listeners (rarely needed)
button.OnClick.RemoveAllListeners()

// Check how many listeners are registered
count := button.OnClick.GetListenerCount()
```

## Example: ScoreManager

The `ScoreManager` script demonstrates:
- Finding UI elements by name
- Wiring up multiple buttons
- Updating UI text from code
- Managing game state

Add it to your scene:

```json
{
  "uid": 3000,
  "name": "ScoreManager",
  "components": [
    {
      "type": "script",
      "script": "ScoreManager",
      "score": 0,
      "score_text_name": "ScoreText",
      "add_button_name": "AddScoreButton",
      "reset_button_name": "ResetButton"
    }
  ]
}
```

Then create the corresponding UI elements with those names.

## Tips

- **Button + Text**: Buttons typically have a child GameObject with UIText for the button label
- **Find by Name**: Use `scene.FindByName()` to get references to UI GameObjects
- **OnClick is not serialized**: Listeners must be added in code (in Start method)
- **Multiple handlers**: Add as many listeners as you want with `AddListener()`
- **Disable buttons**: Set `button.Disabled = true` to prevent interaction
- **Lambda functions**: You can use anonymous functions: `button.OnClick.AddListener(func() { score += 10 })`

## Common Patterns

### Update Text from Script

```go
if textObj := g.Scene.FindByName("MyText"); textObj != nil {
    if text := engine.GetComponent[*components.UIText](textObj); text != nil {
        text.Text = fmt.Sprintf("Lives: %d", lives)
    }
}
```

### Disable Button Conditionally

```go
func (s *MyScript) Update(dt float32) {
    if s.button != nil {
        s.button.Disabled = (s.lives <= 0)
    }
}
```

### Change Button Colors

```go
s.button.HoverColor = rl.NewColor(255, 100, 100, 255)
```
