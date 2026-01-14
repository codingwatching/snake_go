package game

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
)

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		Snake:             []Point{{X: config.Width / 2, Y: config.Height / 2}},
		Direction:         Point{X: 1, Y: 0},
		LastMoveDir:       Point{X: 1, Y: 0},
		Score:             0,
		GameOver:          false,
		StartTime:         time.Now(),
		FoodEaten:         0,
		Foods:             make([]Food, 0),
		LastFoodSpawn:     time.Now(),
		Obstacles:         make([]Obstacle, 0),
		LastObstacleSpawn: time.Now(),
		// Initialize AI competitor snake
		AISnake:      []Point{{X: config.Width - 2, Y: config.Height - 2}},
		AIDirection:  Point{X: -1, Y: 0},
		AILastDir:    Point{X: -1, Y: 0},
		AIScore:      0,
		TimerStarted: true,
		Mode:         "battle",
	}
	// Spawn initial food
	g.spawnOneFood()
	return g
}

// spawnOneFood generates one food of random type
func (g *Game) spawnOneFood() {
	if len(g.Foods) >= config.MaxFoodsOnBoard {
		return // Max foods reached
	}

	// Randomly select food type with weighted probability
	randNum := rand.Intn(100)
	var foodType FoodType
	if randNum < 15 { // 15% red (high score)
		foodType = FoodRed
	} else if randNum < 35 { // 20% orange
		foodType = FoodOrange
	} else if randNum < 65 { // 25% blue
		foodType = FoodBlue
	} else { // 35% purple (low score)
		foodType = FoodPurple
	}

	// Find position that doesn't overlap with snakes, foods or obstacles
	for attempts := 0; attempts < 100; attempts++ {
		pos := Point{
			X: rand.Intn(config.Width-2) + 1,
			Y: rand.Intn(config.Height-2) + 1,
		}

		if !g.isCellEmpty(pos) {
			continue
		}

		// Found valid position, spawn food
		g.Foods = append(g.Foods, Food{
			Pos:               pos,
			FoodType:          foodType,
			SpawnTime:         time.Now(),
			PausedTimeAtSpawn: g.GetTotalPausedTime(),
		})
		g.LastFoodSpawn = time.Now()
		return
	}
}

// removeExpiredFoods removes expired foods from the board
func (g *Game) removeExpiredFoods() {
	newFoods := make([]Food, 0)
	totalPaused := g.GetTotalPausedTime()
	for _, food := range g.Foods {
		if !food.IsExpired(totalPaused) {
			newFoods = append(newFoods, food)
		}
	}
	g.Foods = newFoods
}

// TrySpawnFood attempts to spawn new food
func (g *Game) TrySpawnFood() {
	if g.GameOver {
		return
	}

	g.removeExpiredFoods()

	if len(g.Foods) == 0 {
		g.spawnOneFood()
		return
	}

	if time.Since(g.LastFoodSpawn) > config.FoodSpawnInterval && len(g.Foods) < config.MaxFoodsOnBoard {
		g.spawnOneFood()
	}
}

// Update advances the player game state
func (g *Game) Update() {
	if g.GameOver {
		return
	}

	g.HitPoints = nil   // Clear previous hit points
	g.ScoreEvents = nil // Clear previous score events

	if g.AutoPlay {
		g.UpdateAI()
	}
	g.LastMoveDir = g.Direction

	head := g.Snake[0]
	newHead := Point{X: head.X + g.Direction.X, Y: head.Y + g.Direction.Y}

	// Collision check for player
	if g.checkCollision(newHead) {
		g.GameOver = true
		g.EndTime = time.Now()
		g.CrashPoint = newHead
		return
	}

	// Move snake
	g.Snake = append([]Point{newHead}, g.Snake...)

	// Food collision
	ate := g.handleFoodCollision(newHead, &g.Score, &g.FoodEaten, true)
	if !ate {
		g.Snake = g.Snake[:len(g.Snake)-1]
	}

	g.TrySpawnFood()
	g.TrySpawnObstacle()
	g.CheckTimeLimit()
}

// CheckTimeLimit checks if the game time has expired
func (g *Game) CheckTimeLimit() {
	if g.Mode == "zen" || g.GameOver || !g.TimerStarted {
		return
	}

	remaining := g.GetTimeRemaining()
	if remaining <= 0 {
		g.GameOver = true
		g.EndTime = time.Now()

		// Determine winner
		if g.Score > g.AIScore {
			g.Winner = "player"
			g.SetMessage("🏁 时间到！你赢了！🏆", 5*time.Second)
		} else if g.AIScore > g.Score {
			g.Winner = "ai"
			g.SetMessage("🏁 时间到！AI 赢了！🤖", 5*time.Second)
		} else {
			g.Winner = "draw"
			g.SetMessage("🏁 时间到！平局！🤝", 5*time.Second)
		}
	}
}

// GetTimeRemaining returns the remaining game time in seconds
func (g *Game) GetTimeRemaining() int {
	if !g.TimerStarted {
		return int(config.GameDuration.Seconds())
	}
	if g.GameOver && g.Winner != "" {
		return 0
	}
	endTime := time.Now()
	if g.GameOver {
		endTime = g.EndTime
	}
	elapsed := endTime.Sub(g.StartTime) - g.GetTotalPausedTime()
	remaining := config.GameDuration - elapsed
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// UpdateAISnake advances the AI competitor snake state
func (g *Game) UpdateAISnake() {
	if g.Mode == "zen" || g.GameOver || g.Paused {
		return
	}

	// Update stun status
	g.AIStunned = time.Now().Before(g.AIStunnedUntil)
	if g.AIStunned {
		return // Skip movement if stunned
	}

	g.CheckTimeLimit()
	if g.GameOver {
		return
	}

	// Decision logic
	g.UpdateCompetitorAI()
	g.AILastDir = g.AIDirection

	// Move
	if len(g.AISnake) == 0 {
		return
	}
	head := g.AISnake[0]
	newHead := Point{X: head.X + g.AIDirection.X, Y: head.Y + g.AIDirection.Y}

	// Collision check for AI
	if g.checkCollision(newHead) {
		// AI died, respawn in a corner after some time?
		// For now just teleport to a safe spot or reset
		g.AISnake = []Point{{X: config.Width - 2, Y: config.Height - 2}}
		g.AIDirection = Point{X: -1, Y: 0}
		g.AILastDir = Point{X: -1, Y: 0}
		g.SetMessage("🤖 AI 竞争者撞墙了！", 2*time.Second)
		return
	}

	g.AISnake = append([]Point{newHead}, g.AISnake...)
	dummyEaten := 0
	ate := g.handleFoodCollision(newHead, &g.AIScore, &dummyEaten, false)
	if !ate {
		g.AISnake = g.AISnake[:len(g.AISnake)-1]
	}
}

func (g *Game) checkCollision(p Point) bool {
	// Wall
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return true
	}
	// Player Body
	for _, s := range g.Snake {
		if s == p {
			return true
		}
	}
	// AI Body
	for _, s := range g.AISnake {
		if s == p {
			return true
		}
	}
	// Obstacles
	for _, obs := range g.Obstacles {
		for _, op := range obs.Points {
			if op == p {
				return true
			}
		}
	}
	return false
}

func (g *Game) handleFoodCollision(pos Point, score *int, eaten *int, isPlayer bool) bool {
	for i, food := range g.Foods {
		if pos == food.Pos {
			totalScore := food.GetTotalScore(config.Width, config.Height)
			*score += totalScore
			*eaten++

			if isPlayer {
				bonusMsg := food.GetBonusMessage(config.Width, config.Height)
				if bonusMsg != "" {
					g.SetMessage(bonusMsg, 3*time.Second)
				}
				// Record score event for player
				g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
					Pos:    pos,
					Amount: totalScore,
					Label:  fmt.Sprintf("+%d", totalScore),
				})
			}

			g.Foods = append(g.Foods[:i], g.Foods[i+1:]...)
			return true
		}
	}
	return false
}

// TogglePause toggles the pause state
func (g *Game) TogglePause() {
	if g.GameOver {
		return
	}
	if !g.Paused {
		g.PauseStart = time.Now()
	} else {
		g.PausedTime += time.Since(g.PauseStart)
	}
	g.Paused = !g.Paused
}

// ToggleAutoPlay toggles the AI auto-play mode
func (g *Game) ToggleAutoPlay() {
	g.AutoPlay = !g.AutoPlay
	if g.AutoPlay {
		g.SetMessage("🤖 自动模式已开启", 2*time.Second)
	} else {
		g.SetMessage("👤 手动模式已恢复", 2*time.Second)
		g.Boosting = false
	}
}

// GetMoveInterval (Player only)
func (g *Game) GetMoveInterval(difficulty string) time.Duration {
	return g.GetMoveIntervalExt(difficulty, g.Boosting)
}

func (g *Game) GetMoveIntervalExt(difficulty string, boosted bool) time.Duration {
	ticks := 13 // Default Medium
	boostTicks := 4
	switch difficulty {
	case "low":
		ticks = 18
		boostTicks = 6
	case "mid":
		ticks = 13
		boostTicks = 4
	case "high":
		ticks = 9
		boostTicks = 3
	}
	if boosted {
		ticks = boostTicks
	}
	return time.Duration(ticks) * config.BaseTick
}

// GetAIMoveInterval (AI always mid speed unless boosting)
func (g *Game) GetAIMoveInterval() time.Duration {
	return g.GetMoveIntervalExt("mid", g.AIBoosting)
}

// ToggleBoost allows manual control of the boosting state
func (g *Game) ToggleBoost(active bool) {
	g.Boosting = active
}

// SetDirection sets the player snake direction
func (g *Game) SetDirection(newDir Point) bool {
	if newDir.X != 0 && g.Direction.X == -newDir.X {
		return false
	}
	if newDir.Y != 0 && g.Direction.Y == -newDir.Y {
		return false
	}
	if newDir.X != 0 && g.LastMoveDir.X == -newDir.X {
		return false
	}
	if newDir.Y != 0 && g.LastMoveDir.Y == -newDir.Y {
		return false
	}

	if g.Direction != newDir {
		g.Direction = newDir
		return true
	}
	return false
}

// GetEatingSpeed calculates player foods eaten per second
func (g *Game) GetEatingSpeed() float64 {
	endTime := time.Now()
	if g.GameOver {
		endTime = g.EndTime
	}
	elapsed := endTime.Sub(g.StartTime) - g.GetTotalPausedTime()
	if elapsed.Seconds() > 0 {
		return float64(g.FoodEaten) / elapsed.Seconds()
	}
	return 0
}

// GetTotalPausedTime returns total paused time
func (g *Game) GetTotalPausedTime() time.Duration {
	totalPaused := g.PausedTime
	if g.Paused {
		endTime := time.Now()
		if g.GameOver {
			endTime = g.EndTime
		}
		totalPaused += endTime.Sub(g.PauseStart)
	}
	return totalPaused
}

// SetMessage sets a temporary message
func (g *Game) SetMessage(message string, duration time.Duration) {
	g.Message = message
	g.MessageTime = time.Now()
	g.MessageDuration = duration
}

// GetMessage returns the current message
func (g *Game) GetMessage() string {
	if g.HasActiveMessage() {
		return g.Message
	}
	return ""
}

// HasActiveMessage checks if message is active
func (g *Game) HasActiveMessage() bool {
	if g.Message == "" {
		return false
	}
	return time.Since(g.MessageTime) < g.MessageDuration
}

// TrySpawnObstacle
func (g *Game) TrySpawnObstacle() {
	if g.GameOver {
		return
	}
	newObs := make([]Obstacle, 0)
	totalPaused := g.GetTotalPausedTime()
	for _, obs := range g.Obstacles {
		pausedSinceSpawn := totalPaused - obs.PausedTimeAtSpawn
		elapsed := time.Since(obs.SpawnTime) - pausedSinceSpawn
		if elapsed.Seconds() <= obs.Duration && len(obs.Points) > 0 {
			newObs = append(newObs, obs)
		}
	}
	g.Obstacles = newObs
	if len(g.Obstacles) < config.MaxObstacles && time.Since(g.LastObstacleSpawn) > config.ObstacleSpawnInterval {
		g.spawnOneObstacle()
	}
}

func (g *Game) spawnOneObstacle() {
	var start Point
	found := false
	for attempts := 0; attempts < 50; attempts++ {
		p := Point{X: rand.Intn(config.Width-4) + 2, Y: rand.Intn(config.Height-4) + 2}
		if g.isCellEmpty(p) {
			start = p
			found = true
			break
		}
	}
	if !found {
		return
	}

	points := []Point{start}
	numPoints := rand.Intn(5) + 1
	dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for i := 1; i < numPoints; i++ {
		base := points[rand.Intn(len(points))]
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			next := Point{base.X + d.X, base.Y + d.Y}
			if next.X > 1 && next.X < config.Width-2 && next.Y > 1 && next.Y < config.Height-2 && g.isCellEmpty(next) {
				already := false
				for _, op := range points {
					if op == next {
						already = true
						break
					}
				}
				if !already {
					points = append(points, next)
					break
				}
			}
		}
	}
	g.Obstacles = append(g.Obstacles, Obstacle{
		Points: points, SpawnTime: time.Now(), Duration: config.ObstacleDuration.Seconds(), PausedTimeAtSpawn: g.GetTotalPausedTime(),
	})
	g.LastObstacleSpawn = time.Now()
}

func (g *Game) isCellEmpty(p Point) bool {
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
	for _, f := range g.Foods {
		if f.Pos == p {
			return false
		}
	}
	for _, obs := range g.Obstacles {
		for _, op := range obs.Points {
			if op == p {
				return false
			}
		}
	}
	return true
}

// Fire
func (g *Game) Fire() {
	if g.GameOver || g.Paused {
		return
	}
	if time.Since(g.LastFireTime) < config.FireballCooldown {
		return
	}
	head := g.Snake[0]
	fb := &Fireball{Pos: head, Dir: g.Direction, SpawnTime: time.Now(), Owner: "player"}
	g.Fireballs = append(g.Fireballs, fb)
	g.LastFireTime = time.Now()
}

// UpdateFireballs
func (g *Game) UpdateFireballs() {
	if g.Paused || g.GameOver {
		return
	}
	g.HitPoints = make([]Point, 0)
	activeFbs := make([]*Fireball, 0)
	for _, fb := range g.Fireballs {
		fb.Pos.X += fb.Dir.X
		fb.Pos.Y += fb.Dir.Y
		hit := false
		if fb.Pos.X <= 0 || fb.Pos.X >= config.Width-1 || fb.Pos.Y <= 0 || fb.Pos.Y >= config.Height-1 {
			hit = true
			g.HitPoints = append(g.HitPoints, fb.Pos)
		}
		if !hit {
			for i, p := range g.Snake {
				if p == fb.Pos {
					if fb.Owner == "player" && i == 0 {
						continue // Don't hit own head when firing
					}
					hit = true
					g.HitPoints = append(g.HitPoints, fb.Pos)
					break
				}
			}
		}
		if !hit {
			for i, p := range g.AISnake {
				if p == fb.Pos {
					if fb.Owner == "ai" && i == 0 {
						continue // Don't hit own head if we allow AI to fire later
					}
					hit = true
					g.HitPoints = append(g.HitPoints, fb.Pos)
					if i == 0 {
						// Hit head: Stun for 2 seconds
						g.AIStunnedUntil = time.Now().Add(2 * time.Second)
						g.Score += 50
						g.SetMessage("🎯 爆头！AI 被眩晕了！", 2*time.Second)
						g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
							Pos:    fb.Pos,
							Amount: 50,
							Label:  "🎯 HEADSHOT +50",
						})
					} else {
						// Hit body: Remove segments and some score
						g.Score += 20
						// Shrink AI: remove 1 segment
						if len(g.AISnake) > 1 {
							g.AISnake = g.AISnake[:len(g.AISnake)-1]
						}
						// Removed middle message for body hits to keep UI cleaner
						g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
							Pos:    fb.Pos,
							Amount: 20,
							Label:  "🔥 HIT +20",
						})
					}
					break
				}
			}
		}
		if !hit {
			for i := range g.Obstacles {
				obs := &g.Obstacles[i]
				for j, p := range obs.Points {
					if p == fb.Pos {
						obs.Points = append(obs.Points[:j], obs.Points[j+1:]...)
						hit = true
						g.HitPoints = append(g.HitPoints, fb.Pos)
						g.Score += 10
						g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
							Pos:    fb.Pos,
							Amount: 10,
							Label:  "+10",
						})
						break
					}
				}
				if hit {
					break
				}
			}
		}
		if !hit {
			activeFbs = append(activeFbs, fb)
		}
	}
	g.Fireballs = activeFbs
}

// GetGameStateSnapshot returns a copy of the current game state for serialization
func (g *Game) GetGameStateSnapshot(started bool, serverBoosting bool, difficulty string) GameState {
	foods := make([]FoodInfo, len(g.Foods))
	totalPaused := g.GetTotalPausedTime()
	for i, f := range g.Foods {
		foods[i] = FoodInfo{
			Pos:              f.Pos,
			FoodType:         int(f.FoodType),
			RemainingSeconds: f.GetRemainingSeconds(totalPaused),
		}
	}

	state := GameState{
		Snake:         g.Snake,
		Foods:         foods,
		Score:         g.Score,
		FoodEaten:     g.FoodEaten,
		EatingSpeed:   g.GetEatingSpeed(),
		Started:       started,
		GameOver:      g.GameOver,
		Paused:        g.Paused,
		Boosting:      g.Boosting || serverBoosting,
		AutoPlay:      g.AutoPlay,
		Difficulty:    difficulty,
		Message:       g.GetMessage(),
		Obstacles:     g.Obstacles,
		Fireballs:     g.Fireballs,
		HitPoints:     g.HitPoints,
		AISnake:       g.AISnake,
		AIScore:       g.AIScore,
		TimeRemaining: g.GetTimeRemaining(),
		Winner:        g.Winner,
		AIStunned:     g.AIStunned,
		Mode:          g.Mode,
		ScoreEvents:   g.ScoreEvents,
	}

	if g.GameOver {
		state.CrashPoint = &g.CrashPoint
	}

	return state
}

// GetGameConfig returns the current game configuration
func (g *Game) GetGameConfig() GameConfig {
	return GameConfig{
		Width:            config.Width,
		Height:           config.Height,
		GameDuration:     int(config.GameDuration.Seconds()),
		FireballCooldown: int(config.FireballCooldown.Milliseconds()),
	}
}
