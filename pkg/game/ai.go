package game

import (
	"github.com/trytobebee/snake_go/pkg/config"
)

// UpdateAI decides the next move for the snake when in AutoPlay mode
func (g *Game) UpdateAI() {
	if !g.AutoPlay || g.GameOver || g.Paused {
		return
	}

	head := g.Snake[0]
	var target Food
	foundFood := false

	// Determine current difficulty from GameServer context
	// Since UpdateAI is called from Game.Update, we need the interval
	// In the web version, the server manages the tick.
	// We'll use "mid" as a stable assumption for internal AI calculation
	currentDiff := "mid"

	// Find best target based on (Score / Distance) and (Time Check)
	maxUtility := -1.0
	shouldBoost := false

	for _, food := range g.Foods {
		dist := float64(abs(food.Pos.X-head.X) + abs(food.Pos.Y-head.Y))
		if dist == 0 {
			dist = 0.5
		}

		remainingSec := food.GetRemainingSeconds(g.GetTotalPausedTime())

		// Time estimation: Use explicit states for estimation to avoid flickering
		normalInterval := g.GetMoveIntervalExt(currentDiff, false)
		timeNeededNormal := float64(dist) * normalInterval.Seconds()

		boostInterval := g.GetMoveIntervalExt(currentDiff, true)
		timeNeededBoost := float64(dist) * boostInterval.Seconds()

		// If unreachable even with boost, skip this food (unless it's the only one)
		if timeNeededBoost > float64(remainingSec) && len(g.Foods) > 1 {
			continue
		}

		// Calculate utility
		totalScore := food.GetTotalScore(config.Width, config.Height)
		utility := float64(totalScore) / dist

		if utility > maxUtility {
			maxUtility = utility
			target = food
			foundFood = true

			// If normal speed is too slow, we need to boost
			if timeNeededNormal > float64(remainingSec) {
				shouldBoost = true
			} else {
				shouldBoost = false
			}
		}
	}

	if !foundFood {
		g.Boosting = false
		return
	}

	// Apply boost decision
	g.Boosting = shouldBoost

	// Pathfinding logic...
	possibleDirs := []Point{
		{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0},
	}

	bestDir := g.Direction
	bestScore := -1000000.0
	snakeLen := len(g.Snake)

	for _, dir := range possibleDirs {
		// Prevent 180-degree turns
		if dir.X != 0 && g.LastMoveDir.X == -dir.X {
			continue
		}
		if dir.Y != 0 && g.LastMoveDir.Y == -dir.Y {
			continue
		}

		nextPos := Point{X: head.X + dir.X, Y: head.Y + dir.Y}
		if !g.isSafe(nextPos) {
			continue
		}

		// Calculate Space Rating (Flood Fill)
		// This is the most important "smart" part: how much room do we have?
		reachableSpace := g.countReachableSpace(nextPos)

		// Base score from space
		score := float64(reachableSpace) * 50.0

		// Trap detection: If space is smaller than our body, it's a death trap
		if reachableSpace < snakeLen {
			score -= 5000.0 // Heavy penalty for suicide moves
		}

		// Reward moving towards food only if it's safe-ish
		distToFood := float64(abs(target.Pos.X-nextPos.X) + abs(target.Pos.Y-nextPos.Y))
		score += (100.0 - distToFood) * 2.0

		// Huge bonus for eating
		if nextPos == target.Pos {
			score += 1000.0
		}

		// Survival fallback: If we are getting cramped, try to move toward tail
		// As snake grows, we should be less picky, but ensure we have enough to survive
		survivalThreshold := snakeLen + 20
		if reachableSpace < survivalThreshold {
			tail := g.Snake[snakeLen-1]
			distToTail := float64(abs(tail.X-nextPos.X) + abs(tail.Y-nextPos.Y))
			// Only follow tail if space is actually getting tight
			urgency := float64(survivalThreshold - reachableSpace)
			score += (100.0 - distToTail) * urgency * 0.5
		}

		if score > bestScore {
			bestScore = score
			bestDir = dir
		}
	}

	g.SetDirection(bestDir)
}

// countReachableSpace uses a simple flood fill to count safe tiles
func (g *Game) countReachableSpace(start Point) int {
	visited := make(map[Point]bool)
	queue := []Point{start}
	visited[start] = true
	count := 0

	// Create a temporary "occupied" map for faster lookups
	occupied := make(map[Point]bool)
	for _, p := range g.Snake {
		occupied[p] = true
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		count++

		// Max early exit for performance.
		// For 25x25 board (625), we want to count enough to know if we can fit
		if count > 450 {
			return count
		}

		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		for _, d := range dirs {
			next := Point{curr.X + d.X, curr.Y + d.Y}

			// Wall check
			if next.X <= 0 || next.X >= config.Width-1 || next.Y <= 0 || next.Y >= config.Height-1 {
				continue
			}

			// Body check (simple: ignore tail might move, but safer to assume occupied)
			if occupied[next] && next != g.Snake[len(g.Snake)-1] {
				continue
			}

			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return count
}

// isSafe checks if a position is not a wall or snake body
func (g *Game) isSafe(p Point) bool {
	// Wall check
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return false
	}

	// Snake body check (excluding tail if not eating, but simple check is safer for demo)
	for _, segment := range g.Snake {
		if p == segment {
			return false
		}
	}

	return true
}

// countFreeNeighbors counts how many adjacent cells are safe
func (g *Game) countFreeNeighbors(p Point) int {
	count := 0
	dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for _, d := range dirs {
		if g.isSafe(Point{p.X + d.X, p.Y + d.Y}) {
			count++
		}
	}
	return count
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
