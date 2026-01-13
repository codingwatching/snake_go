package game

import (
	"math/rand"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
)

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		Snake:         []Point{{X: config.Width / 2, Y: config.Height / 2}},
		Direction:     Point{X: 1, Y: 0},
		LastMoveDir:   Point{X: 1, Y: 0},
		Score:         0,
		GameOver:      false,
		StartTime:     time.Now(),
		FoodEaten:     0,
		Foods:         make([]Food, 0),
		LastFoodSpawn: time.Now(),
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

	// Find position that doesn't overlap with snake or other foods
	for attempts := 0; attempts < 100; attempts++ {
		pos := Point{
			X: rand.Intn(config.Width-2) + 1,
			Y: rand.Intn(config.Height-2) + 1,
		}

		// Check snake collision
		onSnake := false
		for _, p := range g.Snake {
			if p == pos {
				onSnake = true
				break
			}
		}
		if onSnake {
			continue
		}

		// Check food collision
		onFood := false
		for _, f := range g.Foods {
			if f.Pos == pos {
				onFood = true
				break
			}
		}
		if onFood {
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
	for _, food := range g.Foods {
		if !food.IsExpired(g.GetTotalPausedTime()) {
			newFoods = append(newFoods, food)
		}
	}
	g.Foods = newFoods
}

// TrySpawnFood attempts to spawn new food
func (g *Game) TrySpawnFood() {
	// Don't spawn when game is over
	if g.GameOver {
		return
	}

	// Remove expired foods
	g.removeExpiredFoods()

	// If no foods, spawn immediately
	if len(g.Foods) == 0 {
		g.spawnOneFood()
		return
	}

	// Spawn new food if interval passed and not at max
	if time.Since(g.LastFoodSpawn) > config.FoodSpawnInterval && len(g.Foods) < config.MaxFoodsOnBoard {
		g.spawnOneFood()
	}
}

// Update advances the game state by one tick
func (g *Game) Update() {
	if g.GameOver {
		return
	}

	// Auto-play logic
	if g.AutoPlay {
		g.UpdateAI()
	}

	// Update last move direction
	g.LastMoveDir = g.Direction

	// Calculate new head position
	head := g.Snake[0]
	newHead := Point{
		X: head.X + g.Direction.X,
		Y: head.Y + g.Direction.Y,
	}

	// Check wall collision
	if newHead.X <= 0 || newHead.X >= config.Width-1 || newHead.Y <= 0 || newHead.Y >= config.Height-1 {
		g.GameOver = true
		g.EndTime = time.Now()
		g.CrashPoint = newHead
		return
	}

	// Check self collision
	for _, p := range g.Snake {
		if p == newHead {
			g.GameOver = true
			g.EndTime = time.Now()
			g.CrashPoint = newHead
			return
		}
	}

	// Move snake
	g.Snake = append([]Point{newHead}, g.Snake...)

	// Check food collision
	ateFood := false
	for i, food := range g.Foods {
		if newHead == food.Pos {
			// Calculate score with position bonus
			totalScore := food.GetTotalScore(config.Width, config.Height)
			g.Score += totalScore
			g.FoodEaten++

			// Show congratulatory message if there's a position bonus
			bonusMsg := food.GetBonusMessage(config.Width, config.Height)
			if bonusMsg != "" {
				g.SetMessage(bonusMsg, 3*time.Second)
			}

			// Remove eaten food
			g.Foods = append(g.Foods[:i], g.Foods[i+1:]...)
			ateFood = true
			break
		}
	}

	if !ateFood {
		// Remove tail if no food eaten
		g.Snake = g.Snake[:len(g.Snake)-1]
	}

	// Try to spawn new food
	g.TrySpawnFood()
}

// TogglePause toggles the pause state
func (g *Game) TogglePause() {
	if g.GameOver {
		return
	}

	if !g.Paused {
		// Start pause
		g.PauseStart = time.Now()
	} else {
		// End pause, accumulate paused time
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
		g.Boosting = false // Reset boosting when leaving auto
	}
}

// GetMoveInterval returns the current expected time between moves
func (g *Game) GetMoveInterval(difficulty string) time.Duration {
	return g.GetMoveIntervalExt(difficulty, g.Boosting)
}

// GetMoveIntervalExt returns the expected time between moves for a specific boost state
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

// ToggleBoost allows manual control of the boosting state (used by AI)
func (g *Game) ToggleBoost(active bool) {
	g.Boosting = active
}

// SetDirection sets the snake direction (with validation)
func (g *Game) SetDirection(newDir Point) bool {
	// Prevent reversing into self relative to current direction
	if newDir.X != 0 && g.Direction.X == -newDir.X {
		return false
	}
	if newDir.Y != 0 && g.Direction.Y == -newDir.Y {
		return false
	}

	// Prevent reversing into self relative to last moved direction
	// This handles the "rapid key press" bug
	if newDir.X != 0 && g.LastMoveDir.X == -newDir.X {
		return false
	}
	if newDir.Y != 0 && g.LastMoveDir.Y == -newDir.Y {
		return false
	}

	// Direction changed
	if g.Direction != newDir {
		g.Direction = newDir
		return true
	}

	return false
}

// GetEatingSpeed calculates foods eaten per second
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

// GetTotalPausedTime returns total paused time including current pause if active
func (g *Game) GetTotalPausedTime() time.Duration {
	totalPaused := g.PausedTime
	// If currently paused, add the current pause duration
	if g.Paused {
		endTime := time.Now()
		if g.GameOver {
			endTime = g.EndTime
		}
		totalPaused += endTime.Sub(g.PauseStart)
	}
	return totalPaused
}

// SetMessage sets a temporary message to display
func (g *Game) SetMessage(message string, duration time.Duration) {
	g.Message = message
	g.MessageTime = time.Now()
	g.MessageDuration = duration
}

// GetMessage returns the current message if it's still active
func (g *Game) GetMessage() string {
	if g.HasActiveMessage() {
		return g.Message
	}
	return ""
}

// HasActiveMessage checks if there's an active message to display
func (g *Game) HasActiveMessage() bool {
	if g.Message == "" {
		return false
	}
	return time.Since(g.MessageTime) < g.MessageDuration
}
