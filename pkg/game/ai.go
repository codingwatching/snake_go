package game

import (
	"math/rand"

	"github.com/trytobebee/snake_go/pkg/config"
)

// UpdateAI decides the next move for the player snake when in AutoPlay mode
func (g *Game) UpdateAI() {
	if !g.AutoPlay || g.GameOver || g.Paused {
		return
	}

	newDir, boosting := g.CalculateBestMove(g.Snake, g.LastMoveDir)
	g.Boosting = boosting
	g.SetDirection(newDir)

	// Fireball logic for player AI
	g.handleAIFire(g.Snake[0], g.Direction, "player")
}

// UpdateCompetitorAI decides the next move for the AI competitor
func (g *Game) UpdateCompetitorAI() {
	if g.GameOver || g.Paused {
		return
	}

	newDir, boosting := g.CalculateBestMove(g.AISnake, g.AILastDir)

	// Aggressive AI logic (only in Berserker Mode):
	// AI boosts if it's far from its target OR if it wants to race the player
	if g.BerserkerMode && !boosting && len(g.AISnake) > 0 {
		head := g.AISnake[0]

		// Find closest food to evaluate distance
		closestDist := 1000
		for _, f := range g.Foods {
			d := abs(f.Pos.X-head.X) + abs(f.Pos.Y-head.Y)
			if d < closestDist {
				closestDist = d
			}

			// If player is also close to this food, boost to compete!
			if len(g.Snake) > 0 {
				playerDist := abs(f.Pos.X-g.Snake[0].X) + abs(f.Pos.Y-g.Snake[0].Y)
				if d < 8 && playerDist < 8 {
					boosting = true
					break
				}
			}
		}

		// If food is far away, occasionally boost to close the gap
		if closestDist > 10 && rand.Float32() < 0.2 {
			boosting = true
		}
	}

	g.AIBoosting = boosting

	// Set AI direction (bypass SetDirection validation which is for player)
	if newDir.X != 0 || newDir.Y != 0 {
		g.AIDirection = newDir
	}

	// Fireball logic for AI competitor
	if len(g.AISnake) > 0 {
		if g.BerserkerMode {
			g.handleAIFire(g.AISnake[0], g.AIDirection, "ai")
		} else {
			// Normal AI only clears obstacles, doesn't shoot at player
			g.handleNormalAIFire(g.AISnake[0], g.AIDirection)
		}
	}
}

// handleNormalAIFire only shoots at obstacles, not players
func (g *Game) handleNormalAIFire(head Point, dir Point) {
	for dist := 1; dist <= 5; dist++ {
		lookAhead := Point{X: head.X + dir.X*dist, Y: head.Y + dir.Y*dist}
		if lookAhead.X <= 0 || lookAhead.X >= config.Width-1 || lookAhead.Y <= 0 || lookAhead.Y >= config.Height-1 {
			break
		}

		isObstacle := false
		for _, obs := range g.Obstacles {
			for _, op := range obs.Points {
				if op == lookAhead {
					isObstacle = true
					break
				}
			}
			if isObstacle {
				break
			}
		}

		if isObstacle {
			g.FireByType("ai")
			break
		}

		isFood := false
		for _, f := range g.Foods {
			if f.Pos == lookAhead {
				isFood = true
				break
			}
		}
		if isFood {
			break
		}
	}
}

// CalculateBestMove computes the best next move for a given snake
func (g *Game) CalculateBestMove(snake []Point, lastMoveDir Point) (Point, bool) {
	if len(snake) == 0 {
		return Point{X: 1, Y: 0}, false
	}

	head := snake[0]
	var target Food
	foundFood := false
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

		// Time estimation
		normalInterval := g.GetMoveIntervalExt(currentDiff, false)
		timeNeededNormal := float64(dist) * normalInterval.Seconds()

		boostInterval := g.GetMoveIntervalExt(currentDiff, true)
		timeNeededBoost := float64(dist) * boostInterval.Seconds()

		if timeNeededBoost > float64(remainingSec) && len(g.Foods) > 1 {
			continue
		}

		totalScore := food.GetTotalScore(config.Width, config.Height)
		utility := float64(totalScore) / dist

		if utility > maxUtility {
			maxUtility = utility
			target = food
			foundFood = true
			shouldBoost = timeNeededNormal > float64(remainingSec)
		}
	}

	if !foundFood {
		return lastMoveDir, false
	}

	// Pathfinding logic
	possibleDirs := []Point{
		{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0},
	}
	// Shuffle dirs to avoid deterministic behavior when scores are equal
	rand.Shuffle(len(possibleDirs), func(i, j int) {
		possibleDirs[i], possibleDirs[j] = possibleDirs[j], possibleDirs[i]
	})

	bestDir := lastMoveDir
	bestScore := -1000000.0
	snakeLen := len(snake)

	for _, dir := range possibleDirs {
		// Prevent 180-degree turns
		if dir.X != 0 && lastMoveDir.X == -dir.X {
			continue
		}
		if dir.Y != 0 && lastMoveDir.Y == -dir.Y {
			continue
		}

		nextPos := Point{X: head.X + dir.X, Y: head.Y + dir.Y}
		if !g.isSafe(nextPos) {
			continue
		}

		reachableSpace := g.countReachableSpace(nextPos)
		score := float64(reachableSpace) * 50.0

		if reachableSpace < snakeLen {
			score -= 5000.0
		}

		distToFood := float64(abs(target.Pos.X-nextPos.X) + abs(target.Pos.Y-nextPos.Y))
		score += (100.0 - distToFood) * 2.0

		if nextPos == target.Pos {
			score += 1000.0
		}

		survivalThreshold := snakeLen + 10
		if reachableSpace < survivalThreshold {
			tail := snake[snakeLen-1]
			distToTail := float64(abs(tail.X-nextPos.X) + abs(tail.Y-nextPos.Y))
			urgency := float64(survivalThreshold - reachableSpace)
			score += (100.0 - distToTail) * urgency * 0.5
		}

		if score > bestScore {
			bestScore = score
			bestDir = dir
		}
	}

	return bestDir, shouldBoost
}

func (g *Game) handleAIFire(head Point, dir Point, owner string) {
	// Look further for targets (up to 10 tiles)
	for dist := 1; dist <= 10; dist++ {
		lookAhead := Point{X: head.X + dir.X*dist, Y: head.Y + dir.Y*dist}
		if lookAhead.X <= 0 || lookAhead.X >= config.Width-1 || lookAhead.Y <= 0 || lookAhead.Y >= config.Height-1 {
			break
		}

		// Check for obstacles
		isObstacle := false
		for _, obs := range g.Obstacles {
			for _, op := range obs.Points {
				if op == lookAhead {
					isObstacle = true
					break
				}
			}
			if isObstacle {
				break
			}
		}

		// Check for enemy snakes
		isTarget := false
		if owner == "ai" {
			// AI targets player
			for _, s := range g.Snake {
				if s == lookAhead {
					isTarget = true
					break
				}
			}
		} else {
			// Player AI targets AI competitor
			for _, s := range g.AISnake {
				if s == lookAhead {
					isTarget = true
					break
				}
			}
		}

		if isObstacle || isTarget {
			g.FireByType(owner)
			break
		}

		// Don't shoot through food
		isFood := false
		for _, f := range g.Foods {
			if f.Pos == lookAhead {
				isFood = true
				break
			}
		}
		if isFood {
			break
		}
	}
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
	for _, p := range g.AISnake {
		occupied[p] = true
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		count++

		if count > 450 {
			return count
		}

		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		for _, d := range dirs {
			next := Point{curr.X + d.X, curr.Y + d.Y}

			if next.X <= 0 || next.X >= config.Width-1 || next.Y <= 0 || next.Y >= config.Height-1 {
				continue
			}

			if occupied[next] {
				// Simple check: ignore tail positions if they might move
				isTail := false
				if len(g.Snake) > 0 && next == g.Snake[len(g.Snake)-1] {
					isTail = true
				}
				if len(g.AISnake) > 0 && next == g.AISnake[len(g.AISnake)-1] {
					isTail = true
				}
				if !isTail {
					continue
				}
			}

			// Obstacle check
			isObs := false
			for _, obs := range g.Obstacles {
				for _, op := range obs.Points {
					if op == next {
						isObs = true
						break
					}
				}
				if isObs {
					break
				}
			}
			if isObs {
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

// isSafe checks if a position is not a wall, snake body or obstacle
func (g *Game) isSafe(p Point) bool {
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return false
	}

	for _, s := range g.Snake {
		if s == p {
			return false
		}
	}
	for _, s := range g.AISnake {
		if s == p {
			return false
		}
	}

	for _, obs := range g.Obstacles {
		for _, op := range obs.Points {
			if p == op {
				return false
			}
		}
	}

	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
