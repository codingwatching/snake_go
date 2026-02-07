package game

import (
	"math/rand"

	"github.com/trytobebee/snake_go/pkg/config"
)

// UpdateAI decides the next move for the player snake when in AutoPlay mode
// --- Obsolete functions removed (logic moved to Controller) ---

// GetFeatureGrid generates the 6-channel input for the Neural Network
// Channels: 0:PlayerHead, 1:PlayerBody, 2:EnemyHead, 3:EnemyBody, 4:Food, 5:Hazard
func (g *Game) GetFeatureGrid(playerIdx int) []float64 {
	W, H := config.Width, config.Height // 25x25
	size := W * H
	grid := make([]float64, 6*size)

	set := func(c, x, y int) {
		if x >= 0 && x < W && y >= 0 && y < H {
			grid[c*size+y*W+x] = 1.0
		}
	}

	if playerIdx >= len(g.Players) {
		return grid
	}

	me := g.Players[playerIdx]

	// Ch 0: Player Head
	if len(me.Snake) > 0 {
		set(0, me.Snake[0].X, me.Snake[0].Y)
	}
	// Ch 1: Player Body
	if len(me.Snake) > 1 {
		for _, p := range me.Snake[1:] {
			set(1, p.X, p.Y)
		}
	}

	// Iterate over other players as enemies
	for i, other := range g.Players {
		if i == playerIdx {
			continue
		}
		// Ch 2: Enemy Head
		if len(other.Snake) > 0 {
			set(2, other.Snake[0].X, other.Snake[0].Y)
		}
		// Ch 3: Enemy Body
		if len(other.Snake) > 1 {
			for _, p := range other.Snake[1:] {
				set(3, p.X, p.Y)
			}
		}
	}

	// Ch 4: Food
	for _, f := range g.Foods {
		set(4, f.Pos.X, f.Pos.Y)
	}

	// Ch 5: Hazard (Obstacles + Fireballs)
	for _, obs := range g.Obstacles {
		for _, p := range obs.Points {
			set(5, p.X, p.Y)
		}
	}
	for _, fb := range g.Fireballs {
		set(5, fb.Pos.X, fb.Pos.Y)
	}

	return grid
}

// UpdateCompetitorAI decides the next move for the AI competitor (p2)
func (g *Game) UpdateCompetitorAI(p *Player) {
	if g.GameOver || g.Paused || p == nil {
		return
	}

	newDir, boosting, _ := g.CalculateBestMove(1, p.Snake, p.LastMoveDir)

	// Aggressive AI logic (only in Berserker Mode):
	// AI boosts if it's far from its target OR if it wants to race the player
	if g.BerserkerMode && !boosting && len(p.Snake) > 0 {
		head := p.Snake[0]

		// Find closest food to evaluate distance
		closestDist := 1000
		for _, f := range g.Foods {
			d := abs(f.Pos.X-head.X) + abs(f.Pos.Y-head.Y)
			if d < closestDist {
				closestDist = d
			}

			// If player is also close to this food, boost to compete!
			if len(g.Players) > 1 {
				// AI is usually p2, so p1 is g.Players[0]
				p1 := g.Players[0]
				if len(p1.Snake) > 0 {
					playerDist := abs(f.Pos.X-p1.Snake[0].X) + abs(f.Pos.Y-p1.Snake[0].Y)
					if d < 8 && playerDist < 8 {
						boosting = true
						break
					}
				}
			}
		}

		// If food is far away, occasionally boost to close the gap
		if closestDist > 10 && rand.Float32() < 0.2 {
			boosting = true
		}
	}

	p.Boosting = boosting

	// Set AI direction (bypass SetDirection validation which is for player1)
	if newDir.X != 0 || newDir.Y != 0 {
		p.Direction = newDir
	}

	// Fireball logic for AI competitor
	if len(p.Snake) > 0 {
		if g.BerserkerMode {
			g.handleAIFire(p, 1)
		} else {
			// Normal AI only clears obstacles, doesn't shoot at player
			g.handleNormalAIFire(p)
		}
	}
}

// handleNormalAIFire only shoots at obstacles, not players
func (g *Game) handleNormalAIFire(p *Player) {
	head := p.Snake[0]
	dir := p.Direction
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
			g.FireByTypeIdx(1) // P2 (AI)
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
func (g *Game) CalculateBestMove(playerIdx int, snake []Point, lastMoveDir Point) (Point, bool, AIContext) {
	ctx := AIContext{
		Intent:  IntentIdle,
		Urgency: 0.0,
	}

	if len(snake) == 0 {
		return Point{X: 1, Y: 0}, false, ctx
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
		normalInterval := g.GetMoveIntervalExt(currentDiff, false)
		timeNeededBoost := float64(dist) * g.GetMoveIntervalExt(currentDiff, true).Seconds()

		if timeNeededBoost > float64(remainingSec) && len(g.Foods) > 1 {
			continue
		}

		totalScore := food.GetTotalScore(config.Width, config.Height)
		utility := float64(totalScore) / dist

		if utility > maxUtility {
			maxUtility = utility
			target = food
			foundFood = true
			shouldBoost = (float64(dist) * normalInterval.Seconds()) > float64(remainingSec)
		}
	}

	// --- NEW: Prop Targeting ---
	var targetProp *Prop
	for i := range g.Props {
		p := &g.Props[i]
		dist := float64(abs(p.Pos.X-head.X) + abs(p.Pos.Y-head.Y))
		if dist == 0 {
			dist = 0.5
		}

		// Treat props as high-value targets (base "score" of 80 for utility calc)
		propUtility := 80.0 / dist
		if propUtility > maxUtility {
			maxUtility = propUtility
			targetProp = p
			foundFood = true // Reuse flag to indicate we have a destination
		}
	}

	var targetPos Point
	if foundFood {
		ctx.Intent = IntentHunt
		if targetProp != nil {
			targetPos = targetProp.Pos
		} else {
			targetPos = target.Pos
		}
		ctx.TargetPos = &targetPos

		// --- NEW: Competitive Boosting ---
		distToTarget := abs(targetPos.X-head.X) + abs(targetPos.Y-head.Y)

		// 1. Race logic: if an enemy is also close to our target, boost!
		for i, other := range g.Players {
			if i == playerIdx || len(other.Snake) == 0 {
				continue
			}
			enemyDist := abs(targetPos.X-other.Snake[0].X) + abs(targetPos.Y-other.Snake[0].Y)
			if distToTarget < 8 && enemyDist < 8 {
				shouldBoost = true
				break
			}
		}

		// 2. Catch-up logic (Berserker Mode): if food is far, occasionally boost to close gap
		if !shouldBoost && distToTarget > 10 && g.BerserkerMode && rand.Float32() < 0.2 {
			shouldBoost = true
		}
	}

	if !foundFood {
		return lastMoveDir, false, ctx
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
		if !g.isSafe(nextPos, playerIdx) {
			continue
		}

		reachableSpace := g.countReachableSpace(nextPos, playerIdx)
		score := float64(reachableSpace) * 50.0

		isSurvive := false
		if reachableSpace < snakeLen {
			score -= 5000.0
			isSurvive = true
		}

		distToTarget := float64(abs(targetPos.X-nextPos.X) + abs(targetPos.Y-nextPos.Y))
		score += (100.0 - distToTarget) * 2.0

		if nextPos == targetPos {
			score += 1000.0
		}

		survivalThreshold := snakeLen + 10
		if reachableSpace < survivalThreshold {
			tail := snake[snakeLen-1]
			distToTail := float64(abs(tail.X-nextPos.X) + abs(tail.Y-nextPos.Y))
			urgency := float64(survivalThreshold - reachableSpace)
			score += (100.0 - distToTail) * urgency * 0.5

			if urgency > 5 {
				isSurvive = true
			}
		}

		if score > bestScore {
			bestScore = score
			bestDir = dir

			if isSurvive {
				ctx.Intent = IntentSurvive
				if snakeLen > 0 {
					ctx.Urgency = 1.0 - float64(reachableSpace)/float64(snakeLen*2)
					if ctx.Urgency < 0 {
						ctx.Urgency = 0
					}
				}
			} else {
				if foundFood {
					ctx.Intent = IntentHunt
				} else {
					ctx.Intent = IntentIdle
				}
				ctx.Urgency = 0.0
			}
		}
	}

	return bestDir, shouldBoost, ctx
}

func (g *Game) handleAIFire(p *Player, ownerIdx int) bool {
	head := p.Snake[0]
	dir := p.Direction
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
		for i, other := range g.Players {
			if i == ownerIdx {
				continue
			}
			for _, s := range other.Snake {
				if s == lookAhead {
					isTarget = true
					break
				}
			}
			if isTarget {
				break
			}
		}

		if isObstacle || isTarget {
			g.FireByTypeIdx(ownerIdx)
			return true
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
	return false
}

// countReachableSpace uses a simple flood fill to count safe tiles.
// It is now more optimistic about its own tail.
func (g *Game) countReachableSpace(start Point, ownerIdx int) int {
	visited := make(map[Point]bool)

	// Pre-fill visited with all current obstacles
	for i, p := range g.Players {
		body := p.Snake
		if i == ownerIdx && len(body) > 1 {
			// For our own snake, we assume the tail will move.
			// This allows the AI to enter loops following its own tail.
			body = body[:len(body)-1]
		}
		for _, s := range body {
			visited[s] = true
		}
	}
	for _, obs := range g.Obstacles {
		for _, op := range obs.Points {
			visited[op] = true
		}
	}

	if visited[start] {
		return 0
	}

	queue := []Point{start}
	visited[start] = true
	count := 0

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		count++

		if count > 400 { // Performance limit
			return count
		}

		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		for _, d := range dirs {
			next := Point{curr.X + d.X, curr.Y + d.Y}

			// Wall check
			if next.X <= 0 || next.X >= config.Width-1 || next.Y <= 0 || next.Y >= config.Height-1 {
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

// isSafe checks if a position is not a wall, snake body or obstacle.
// It also checks for "threat zones" created by other players' heads.
func (g *Game) isSafe(p Point, ownerIdx int) bool {
	// 1. Boundary check
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return false
	}

	// 2. Obstacle check
	for _, obs := range g.Obstacles {
		for _, op := range obs.Points {
			if p == op {
				return false
			}
		}
	}

	// 3. Players check (Body and Head Proximity)
	for i, player := range g.Players {
		if len(player.Snake) == 0 {
			continue
		}
		bodyToCheck := player.Snake
		if i == ownerIdx {
			// For our own snake, we can follow our tail if we are long enough
			if len(bodyToCheck) > 1 {
				bodyToCheck = bodyToCheck[:len(bodyToCheck)-1]
			} else {
				bodyToCheck = []Point{}
			}

			// Self-collision check
			for _, s := range bodyToCheck {
				if s == p {
					return false
				}
			}
		} else {
			// Enemy check
			// Enemy Body check (all except tail which is about to move)
			enemyBody := player.Snake
			if len(enemyBody) > 1 {
				enemyBody = enemyBody[:len(enemyBody)-1]
			}
			for _, s := range enemyBody {
				if s == p {
					return false
				}
			}

			// --- THE CRITICAL FIX: Enemy Head Proximity ---
			// If p is adjacent to the enemy head, they could move into p in the same tick!
			enemyHead := player.Snake[0]
			distToEnemyHead := abs(enemyHead.X-p.X) + abs(enemyHead.Y-p.Y)

			if distToEnemyHead <= 1 {
				// Only be cautious in non-berserker modes.
				// In Berserker mode, we 'dare' to challenge for the same spot!
				if !g.BerserkerMode {
					return false
				}
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
