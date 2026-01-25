package game

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
)

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		Players: []*Player{
			{
				Snake:       []Point{{X: config.Width / 2, Y: config.Height / 2}},
				Direction:   Point{X: 1, Y: 0},
				LastMoveDir: Point{X: 1, Y: 0},
				Name:        "Player 1",
				Brain:       &ManualController{},
				Controller:  "manual",
			},
		},
		Foods:             make([]Food, 0),
		LastFoodSpawn:     time.Now(),
		Obstacles:         make([]Obstacle, 0),
		LastObstacleSpawn: time.Now(),
		TimerStarted:      true,
		StartTime:         time.Now(),
		PauseStart:        time.Now(),
		Mode:              "battle",
	}

	// In battle mode, add the second player (AI)
	g.Players = append(g.Players, &Player{
		Snake:       []Point{{X: config.Width - 2, Y: config.Height - 2}},
		Direction:   Point{X: -1, Y: 0},
		LastMoveDir: Point{X: -1, Y: 0},
		Name:        "AI",
		Brain:       &HeuristicController{},
		Controller:  "heuristic",
	})

	// Start Global AI Inference Service if not already started
	onnxPath := "ml/checkpoints/snake_policy.onnx"
	err := StartInferenceService(onnxPath)
	if err == nil {
		g.NeuralNet = &ONNXModel{}
		log.Println("🧠 Global AI Service (ONNX) is active for this game!")
	} else {
		log.Printf("⚠️  Global AI Service Error: %v. Using Heuristic AI.\n", err)
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

// Update advances the game state
func (g *Game) Update() {
	if g.GameOver || g.Paused {
		return
	}

	g.HitPoints = nil   // Clear previous hit points
	g.ScoreEvents = nil // Clear previous score events

	// Update each player
	for i := range g.Players {
		g.UpdatePlayer(i)
	}

	g.TrySpawnFood()
	g.TrySpawnObstacle()
	g.CheckTimeLimit()
}

// UpdatePlayer advances a single player's state
func (g *Game) UpdatePlayer(idx int) {
	if idx >= len(g.Players) {
		return
	}
	p := g.Players[idx]

	// Update stun status
	p.Stunned = time.Now().Before(p.StunnedUntil)
	if p.Stunned {
		return // Skip movement if stunned
	}

	// Brain decision logic: Get action from controller
	if p.Brain != nil {
		action := p.Brain.GetAction(g, idx)
		if action.Direction.X != 0 || action.Direction.Y != 0 {
			// VALIDATE: Prevent 180-degree turns
			isOpposite := (action.Direction.X != 0 && p.LastMoveDir.X == -action.Direction.X) ||
				(action.Direction.Y != 0 && p.LastMoveDir.Y == -action.Direction.Y)

			if !isOpposite {
				p.Direction = action.Direction
			}
		}
		p.Boosting = action.Boost
		if action.Fire {
			g.FireByTypeIdx(idx)
		}
	}

	p.LastMoveDir = p.Direction

	head := p.Snake[0]
	newHead := Point{X: head.X + p.Direction.X, Y: head.Y + p.Direction.Y}

	// Collision check
	if g.checkCollision(newHead) {
		log.Printf("[Game] Collision detected for player %d at %+v (IsPVP: %v)", idx, newHead, g.IsPVP)
		if g.IsPVP {
			log.Printf("[Game] PVP COLLISION: Player %d hit something at %+v. IsPVP: %v", idx, newHead, g.IsPVP)
			// In PVP, collision means the OTHER player wins
			g.GameOver = true
			g.EndTime = time.Now()
			g.CrashPoint = newHead
			if idx == 0 {
				g.Winner = "ai" // P2 wins
			} else {
				g.Winner = "player" // P1 wins
			}
		} else {
			if idx == 0 {
				log.Printf("[Game] P1 COLLISION (Solo): Hit something at %+v", newHead)
				// Player 1 dead
				g.GameOver = true
				g.EndTime = time.Now()
				g.CrashPoint = newHead
			} else {
				// AI dead, reset it
				p.Snake = []Point{{X: config.Width - 2, Y: config.Height - 2}}
				p.Direction = Point{X: -1, Y: 0}
				p.LastMoveDir = Point{X: -1, Y: 0}
				if !g.IsPVP {
					g.SetMessage("🤖 AI 竞争者撞墙了！")
				}
			}
		}
		return
	}

	// Move snake
	p.Snake = append([]Point{newHead}, p.Snake...)

	// Food collision
	ate := g.handleFoodCollision(newHead, p, idx == 0)
	if !ate {
		p.Snake = p.Snake[:len(p.Snake)-1]
	}
}

// CheckTimeLimit checks if the game time has expired
func (g *Game) CheckTimeLimit() {
	if g.Mode == "zen" || g.GameOver || !g.TimerStarted {
		return
	}

	remaining := g.GetTimeRemaining()
	if remaining <= 0 {
		log.Printf("[Game] Time Limit Reached (IsPVP: %v)", g.IsPVP)
		log.Printf("[Game] TIME LIMIT EXPIRED! Duration: %v, Elapsed: %v, Remaining: %d", config.GameDuration, time.Since(g.StartTime), remaining)
		g.GameOver = true
		g.EndTime = time.Now()

		// Determine winner
		if len(g.Players) >= 2 {
			s1 := g.Players[0].Score
			s2 := g.Players[1].Score
			if s1 > s2 {
				g.Winner = "player"
			} else if s2 > s1 {
				g.Winner = "ai"
			} else {
				g.Winner = "draw"
			}
		} else {
			g.Winner = "none"
		}
	}
}

// GetTimeRemaining returns the remaining game time in seconds
func (g *Game) GetTimeRemaining() int {
	if !g.TimerStarted {
		return int(config.GameDuration.Seconds())
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

func (g *Game) checkCollision(p Point) bool {
	// Wall
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return true
	}
	// All Players
	for _, player := range g.Players {
		for _, s := range player.Snake {
			if s == p {
				return true
			}
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

func (g *Game) handleFoodCollision(pos Point, p *Player, isP1 bool) bool {
	for i, food := range g.Foods {
		if pos == food.Pos {
			totalScore := food.GetTotalScore(config.Width, config.Height)
			p.Score += totalScore
			p.FoodEaten++

			if isP1 {
				bonusMsg := food.GetBonusMessage(config.Width, config.Height)
				if bonusMsg != "" {
					g.SetMessageWithType(bonusMsg, "bonus")
				}
				// Record score event
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

// TogglePlayerAutoPlay toggles the AI auto-play mode for a specific player
func (g *Game) TogglePlayerAutoPlay(idx int, requestedMode string) {
	if idx >= len(g.Players) {
		return
	}
	p := g.Players[idx]

	// If it's currently manual or we want to change the agent type while running
	isSwitchingModes := p.Controller != "manual" && requestedMode != "" && p.Controller != requestedMode

	if p.Controller == "manual" || isSwitchingModes {
		modeToUse := requestedMode
		if modeToUse == "" {
			if g.NeuralNet != nil {
				modeToUse = "neural"
			} else {
				modeToUse = "heuristic"
			}
		}

		if modeToUse == "neural" && g.NeuralNet != nil {
			p.Brain = &NeuralController{}
			p.Controller = "neural"
			g.SetMessage(p.Name + ": 🧠 神经网络模型已注入")
			log.Printf("[Game] Player %d (%s) switched to NEURAL controller", idx, p.Name)
		} else {
			p.Brain = &HeuristicController{}
			p.Controller = "heuristic"
			g.SetMessage(p.Name + ": 📏 启发式规则控制器已注入")
			log.Printf("[Game] Player %d (%s) switched to HEURISTIC controller", idx, p.Name)
		}
	} else {
		// Switch back to manual
		p.Brain = &ManualController{}
		p.Controller = "manual"
		p.Boosting = false
		g.SetMessage(p.Name + ": 👤 已恢复手动模式")
		log.Printf("[Game] Player %d (%s) switched to MANUAL controller", idx, p.Name)
	}

	// Legacy flag for backward compatibility
	if idx == 0 {
		g.AutoPlay = (p.Controller != "manual")
	}
}

// ToggleBerserkerMode toggles the aggressive AI mode
func (g *Game) ToggleBerserkerMode() {
	g.BerserkerMode = !g.BerserkerMode
	if g.BerserkerMode {
		g.SetMessageWithType("👹 狂暴模式：开启！", "important")
	} else {
		g.SetMessage("👤 狂暴模式：已关闭")
	}
}

// GetMoveInterval (P1 only)
func (g *Game) GetMoveInterval(difficulty string) time.Duration {
	boosted := false
	if len(g.Players) > 0 {
		boosted = g.Players[0].Boosting
	}
	return g.GetMoveIntervalExt(difficulty, boosted)
}

func (g *Game) GetMoveIntervalExt(difficulty string, boosted bool) time.Duration {
	ticks := config.MidTicks
	boostTicks := config.MidBoostTicks

	switch difficulty {
	case "low":
		ticks = config.LowTicks
		boostTicks = config.LowBoostTicks
	case "mid":
		ticks = config.MidTicks
		boostTicks = config.MidBoostTicks
	case "high":
		ticks = config.HighTicks
		boostTicks = config.HighBoostTicks
	}

	if boosted {
		ticks = boostTicks
	}
	return time.Duration(ticks) * config.BaseTick
}

// GetAIMoveInterval (AI defaults to mid speed unless boosting)
func (g *Game) GetAIMoveInterval() time.Duration {
	boosted := false
	if len(g.Players) > 1 {
		boosted = g.Players[1].Boosting
	}
	return g.GetMoveIntervalExt("mid", boosted)
}

// SetDirection sets the direction for Player 1
func (g *Game) SetDirection(newDir Point) bool {
	return g.SetPlayerDirection(0, newDir)
}

// SetPlayerDirection sets the direction for a specific player
func (g *Game) SetPlayerDirection(idx int, newDir Point) bool {
	if idx >= len(g.Players) {
		return false
	}
	p := g.Players[idx]

	compareDir := p.LastMoveDir
	if compareDir.X == 0 && compareDir.Y == 0 {
		compareDir = p.Direction
	}

	if newDir.X != 0 && compareDir.X == -newDir.X {
		return false
	}
	if newDir.Y != 0 && compareDir.Y == -newDir.Y {
		return false
	}

	if p.Direction != newDir {
		p.Direction = newDir
		return true
	}
	return false
}

// GetEatingSpeed calculates P1 foods eaten per second
func (g *Game) GetEatingSpeed() float64 {
	if len(g.Players) == 0 {
		return 0
	}
	endTime := time.Now()
	if g.GameOver {
		endTime = g.EndTime
	}
	elapsed := endTime.Sub(g.StartTime) - g.GetTotalPausedTime()
	if elapsed.Seconds() > 0 {
		return float64(g.Players[0].FoodEaten) / elapsed.Seconds()
	}
	return 0
}

// GetTotalPausedTime returns total paused time
func (g *Game) GetTotalPausedTime() time.Duration {
	totalPaused := g.PausedTime
	if g.Paused && !g.PauseStart.IsZero() {
		endTime := time.Now()
		if g.GameOver {
			endTime = g.EndTime
		}
		totalPaused += endTime.Sub(g.PauseStart)
	}
	return totalPaused
}

// SetMessage sets a message with normal type
func (g *Game) SetMessage(message string) {
	g.SetMessageWithType(message, "normal")
}

// SetMessageWithType sets a message with specific type
func (g *Game) SetMessageWithType(message string, msgType string) {
	g.Message = message
	g.MessageType = msgType
}

// GetMessage returns the current message
func (g *Game) GetMessage() string {
	return g.Message
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
	for _, player := range g.Players {
		for _, s := range player.Snake {
			if s == p {
				return false
			}
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

// Fire allows Player 1 to shoot a fireball
func (g *Game) Fire() {
	g.FireByTypeIdx(0)
}

func (g *Game) FireByTypeIdx(idx int) {
	if g.GameOver || g.Paused || idx >= len(g.Players) {
		return
	}
	p := g.Players[idx]
	if p.Stunned || len(p.Snake) == 0 {
		return
	}

	if time.Since(p.LastFireTime) < config.FireballCooldown {
		return
	}

	owner := "player"
	if idx > 0 {
		owner = "ai" // Map P2+ to "ai" for frontend compatibility
	}

	fb := &Fireball{
		Pos:       p.Snake[0],
		Dir:       p.Direction,
		SpawnTime: time.Now(),
		Owner:     owner,
	}
	g.Fireballs = append(g.Fireballs, fb)
	p.LastFireTime = time.Now()
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

		// Wall collision
		if fb.Pos.X <= 0 || fb.Pos.X >= config.Width-1 || fb.Pos.Y <= 0 || fb.Pos.Y >= config.Height-1 {
			hit = true
			g.HitPoints = append(g.HitPoints, fb.Pos)
		}

		if !hit {
			// Check collision with all players
			for pIdx, player := range g.Players {
				for i, p := range player.Snake {
					if p == fb.Pos {
						// Don't hit own head when firing
						ownerIdx := 0
						if fb.Owner == "ai" {
							ownerIdx = 1
						}
						if pIdx == ownerIdx && i == 0 {
							continue
						}

						hit = true
						g.HitPoints = append(g.HitPoints, fb.Pos)

						// Hit logic refined
						targetPlayer := player

						// Determine attacker
						attackerIdx := 0
						if fb.Owner == "ai" {
							attackerIdx = 1
						}

						var attackerScore int
						var label string

						if i == 0 {
							// HEADSHOT: +50 to attacker, 2s stun to victim
							attackerScore = 50
							label = "🎯 HEADSHOT +50"
							targetPlayer.StunnedUntil = time.Now().Add(2 * time.Second)
							if pIdx == 0 {
								g.SetMessageWithType("😱 警告！头部被击中，麻痹2秒！", "important")
							}
						} else {
							// BODY HIT: +10 to attacker, shorten body
							attackerScore = 10
							label = "🔥 HIT +10"
							// Shorten body (remove last 1 segment if possible)
							segmentsToRemove := 1
							if len(targetPlayer.Snake) > segmentsToRemove+1 {
								targetPlayer.Snake = targetPlayer.Snake[:len(targetPlayer.Snake)-segmentsToRemove]
							}
						}

						// Award score to attacker
						if attackerIdx < len(g.Players) {
							g.Players[attackerIdx].Score += attackerScore
						}

						g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
							Pos:    fb.Pos,
							Amount: attackerScore,
							Label:  label,
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
			// Obstacle collision
			for i := range g.Obstacles {
				obs := &g.Obstacles[i]
				for j, p := range obs.Points {
					if p == fb.Pos {
						obs.Points = append(obs.Points[:j], obs.Points[j+1:]...)
						hit = true
						g.HitPoints = append(g.HitPoints, fb.Pos)

						// Add score (+10) to fireball owner for destroying obstacle
						ownerIdx := 0
						if fb.Owner == "ai" {
							ownerIdx = 1
						}
						if ownerIdx < len(g.Players) {
							g.Players[ownerIdx].Score += 10
						}

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
		Foods:         foods,
		EatingSpeed:   g.GetEatingSpeed(),
		Started:       started,
		GameOver:      g.GameOver,
		Paused:        g.Paused,
		AutoPlay:      g.AutoPlay,
		Difficulty:    difficulty,
		Message:       g.GetMessage(),
		MessageType:   g.MessageType,
		Obstacles:     g.Obstacles,
		Fireballs:     g.Fireballs,
		HitPoints:     g.HitPoints,
		TimeRemaining: g.GetTimeRemaining(),
		Winner:        g.Winner,
		Mode:          g.Mode,
		ScoreEvents:   g.ScoreEvents,
		Berserker:     g.BerserkerMode,
		IsPVP:         g.IsPVP,
	}

	// Populate P1 fields
	if len(g.Players) > 0 {
		p1 := g.Players[0]
		state.Snake = p1.Snake
		state.Score = p1.Score
		state.FoodEaten = p1.FoodEaten
		state.Boosting = p1.Boosting || serverBoosting
		state.PlayerStunned = time.Now().Before(p1.StunnedUntil)
		state.P1Name = p1.Name
	}

	// Populate P2 Fields (compatible with older AISnake frontend keys)
	if len(g.Players) > 1 {
		p2 := g.Players[1]
		state.AISnake = p2.Snake
		state.AIScore = p2.Score
		state.AIStunned = time.Now().Before(p2.StunnedUntil)
		state.P2Name = p2.Name
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
