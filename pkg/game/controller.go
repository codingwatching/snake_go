package game

import (
	"math/rand"
)

// Controller defines the brain of a player (Human, AI, or Model)
type Controller interface {
	GetAction(g *Game, playerIdx int) ActionData
}

// --- Implementation: Manual Controller (Human) ---

type ManualController struct {
	PendingAction ActionData
}

func (c *ManualController) GetAction(g *Game, playerIdx int) ActionData {
	// Return the action set by external WebSocket events
	return c.PendingAction
}

func (c *ManualController) SetDirection(dir Point) {
	c.PendingAction.Direction = dir
}

func (c *ManualController) SetBoosting(boosting bool) {
	c.PendingAction.Boost = boosting
}

func (c *ManualController) SetFire(fire bool) {
	c.PendingAction.Fire = fire
}

// --- Implementation: Heuristic AI Controller ---

type HeuristicController struct{}

func (c *HeuristicController) GetAction(g *Game, playerIdx int) ActionData {
	if playerIdx >= len(g.Players) {
		return ActionData{}
	}
	p := g.Players[playerIdx]

	newDir, boosting, _ := g.CalculateBestMove(p.Snake, p.LastMoveDir)

	// Intelligent Firing Module
	fire := g.shouldAIFire(playerIdx, newDir)
	if !fire && !g.IsPVP {
		// Rare random shots in solo mode only
		fire = rand.Float32() < 0.01
	}

	return ActionData{
		Direction: newDir,
		Boost:     boosting,
		Fire:      fire,
	}
}

// --- Implementation: Neural Network Controller ---

type NeuralController struct{}

func (c *NeuralController) GetAction(g *Game, playerIdx int) ActionData {
	if playerIdx >= len(g.Players) || g.NeuralNet == nil {
		// Fallback to Heuristic if NN not available
		hc := &HeuristicController{}
		return hc.GetAction(g, playerIdx)
	}

	p := g.Players[playerIdx]
	input := g.GetFeatureGrid(playerIdx)
	logits := Predict(input)

	// Simple argmax
	bestIdx := 0
	var maxVal float32 = -1e9
	for i, v := range logits {
		if v > maxVal {
			maxVal = v
			bestIdx = i
		}
	}

	var newDir Point
	switch bestIdx {
	case 0:
		newDir = Point{X: 0, Y: -1}
	case 1:
		newDir = Point{X: 0, Y: 1}
	case 2:
		newDir = Point{X: -1, Y: 0}
	case 3:
		newDir = Point{X: 1, Y: 0}
	}

	// Safety check - if NN suggests suicide, fallback
	nextHead := Point{X: p.Snake[0].X + newDir.X, Y: p.Snake[0].Y + newDir.Y}
	if !g.isSafe(nextHead) {
		hc := &HeuristicController{}
		return hc.GetAction(g, playerIdx)
	}

	// Intelligent Firing Module for Neural Controller (Auto-Aim)
	fire := false
	if len(p.Snake) > 0 {
		fire = g.shouldAIFire(playerIdx, newDir)
	}

	return ActionData{
		Direction: newDir,
		Boost:     false,
		Fire:      fire,
	}
}

// shouldAIFire is a helper that checks if there's a target in front of the snake
func (g *Game) shouldAIFire(idx int, dir Point) bool {
	if idx >= len(g.Players) {
		return false
	}
	p := g.Players[idx]
	if len(p.Snake) == 0 {
		return false
	}

	head := p.Snake[0]
	// Range: 8 tiles
	for dist := 1; dist <= 8; dist++ {
		look := Point{X: head.X + dir.X*dist, Y: head.Y + dir.Y*dist}

		// If it's a wall, stop looking
		if look.X <= 0 || look.X >= 24 || look.Y <= 0 || look.Y >= 24 {
			break
		}

		// Check for obstacles
		for _, obs := range g.Obstacles {
			for _, op := range obs.Points {
				if op == look {
					return true
				}
			}
		}

		// Check for other players
		for i, other := range g.Players {
			if i == idx {
				continue
			}
			for _, s := range other.Snake {
				if s == look {
					return true
				}
			}
		}
	}
	return false
}
