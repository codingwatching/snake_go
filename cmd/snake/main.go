package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
	"github.com/trytobebee/snake_go/pkg/input"
	"github.com/trytobebee/snake_go/pkg/renderer"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Initialize input handler
	inputHandler := input.NewKeyboardHandler()
	if err := inputHandler.Start(); err != nil {
		fmt.Println("Error opening keyboard:", err)
		return
	}
	defer inputHandler.Stop()

	// Initialize renderer
	render := renderer.NewTerminalRenderer()

	// Create new game
	g := game.NewGame()

	// Get input channel
	inputChan := inputHandler.GetInputChan()

	// Game loop ticker
	ticker := time.NewTicker(config.BaseTick)
	defer ticker.Stop()

	// Boost tracking state
	var (
		tickCount           = 0
		boosting            = false
		lastBoostKeyTime    time.Time
		lastDirKeyTime      time.Time
		lastDirKeyDir       game.Point
		consecutiveKeyCount = 0
	)

	// Check if boost should be triggered
	checkBoostKey := func(inputDir game.Point) {
		now := time.Now()

		if inputDir == lastDirKeyDir && time.Since(lastDirKeyTime) < config.KeyRepeatWindow {
			consecutiveKeyCount++
		} else {
			consecutiveKeyCount = 1
		}

		lastDirKeyDir = inputDir
		lastDirKeyTime = now

		if len(g.Players) > 0 {
			if consecutiveKeyCount >= config.BoostThreshold && inputDir == g.Players[0].Direction {
				boosting = true
				lastBoostKeyTime = now
			}
		}
	}

	// Initial render
	render.Render(g, false)

	// Main game loop
	for {
		select {
		case inputEvent := <-inputChan:
			if input.IsQuit(inputEvent) {
				fmt.Println("\n  Thanks for playing! 👋")
				return
			}

			if input.IsRestart(inputEvent) {
				if g.GameOver {
					g = game.NewGame()
					boosting = false
					tickCount = 0
					consecutiveKeyCount = 0
				}
			}

			if input.IsPause(inputEvent) {
				if !g.GameOver {
					g.TogglePause()
					render.Render(g, boosting)
				}
			}

			if inputDir, isValid := input.ParseDirection(inputEvent); isValid {
				dirChanged := g.SetDirection(inputDir)

				if dirChanged {
					consecutiveKeyCount = 1
					lastDirKeyDir = inputDir
					lastDirKeyTime = time.Now()
					boosting = false
				} else {
					checkBoostKey(inputDir)
				}
			}

		case <-ticker.C:
			if boosting && time.Since(lastBoostKeyTime) > config.BoostTimeout {
				boosting = false
			}

			if len(g.Players) > 0 {
				g.Players[0].Boosting = boosting
			}

			tickCount++
			ticksNeeded := config.NormalTicksPerUpdate
			if boosting {
				ticksNeeded = config.BoostTicksPerUpdate
			}

			if tickCount >= ticksNeeded {
				tickCount = 0
				if !g.GameOver && !g.Paused {
					g.Update()
				}
				render.Render(g, boosting)
			}
		}
	}
}
