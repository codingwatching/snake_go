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
	switch {
	case randNum < 15: // 15% red (high score)
		foodType = FoodRed
	case randNum < 35: // 20% orange
		foodType = FoodOrange
	case randNum < 65: // 25% blue
		foodType = FoodBlue
	default: // 35% purple (low score)
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

	g.HitPoints = nil
	g.ScoreEvents = nil

	// Update each player. In synchronized mode, they still move "together" in the same tick,
	// but UpdatePlayer handles their individual logic.
	for i := range g.Players {
		g.UpdatePlayer(i)
	}

	g.TrySpawnFood()
	g.TrySpawnProp()
	g.TrySpawnObstacle()
	g.CheckTimeLimit()
	g.updateActiveEffects()
}

func (g *Game) updateActiveEffects() {
	now := time.Now()
	for _, p := range g.Players {
		var active []*ActiveEffect
		for _, e := range p.Effects {
			if now.Before(e.ExpireAt) {
				e.Duration = e.ExpireAt.Sub(now).Seconds()
				active = append(active, e)
			}
		}
		p.Effects = active
	}

	// Clean up expired props on board
	totalPaused := g.GetTotalPausedTime()
	var remainingProps []Prop
	for _, pr := range g.Props {
		if !pr.IsExpired(totalPaused) {
			remainingProps = append(remainingProps, pr)
		}
	}
	g.Props = remainingProps
}

// UpdatePlayer moves a single player and handles its collisions
func (g *Game) UpdatePlayer(idx int) {
	if g.GameOver || g.Paused || idx >= len(g.Players) {
		return
	}

	p := g.Players[idx]
	p.Stunned = time.Now().Before(p.StunnedUntil)
	if p.Stunned {
		return
	}

	// 1. Brain decision
	if p.Brain != nil {
		action := p.Brain.GetAction(g, idx)
		if action.Direction.X != 0 || action.Direction.Y != 0 {
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

	// 2. Calculate next head
	nextHead := Point{X: p.Snake[0].X + p.Direction.X, Y: p.Snake[0].Y + p.Direction.Y}

	// 3. Collision Check
	if g.checkCollisionForPlayer(idx, nextHead) {
		// --- THE SHIELD BUFF CHECK ---
		hasShield := false
		for i, e := range p.Effects {
			if e.Type == EffectShield {
				// Use up the shield!
				p.Effects = append(p.Effects[:i], p.Effects[i+1:]...)
				hasShield = true
				g.SetMessage("🛡️ 保险丝生效！护盾抵消了一次碰撞")
				g.HitPoints = append(g.HitPoints, nextHead)
				break
			}
		}

		if !hasShield {
			if g.IsPVP || idx == 0 {
				// Player or PVP participant died
				g.GameOver = true
				g.EndTime = time.Now()
				g.CrashPoint = nextHead
				if g.IsPVP {
					if idx == 0 {
						g.Winner = "ai"
					} else {
						g.Winner = "player"
					}
				}
			} else {
				// AI competitor died, reset it
				p.Snake = []Point{{X: config.Width - 2, Y: config.Height - 2}}
				p.Direction = Point{X: -1, Y: 0}
				p.LastMoveDir = Point{X: -1, Y: 0}
				g.SetMessage("🤖 AI 竞争者撞墙了！")
			}
			return
		}
		// Shield was used: "Brake" by returning early without updating position
		return
	}

	// 4. Move
	p.Snake = append([]Point{nextHead}, p.Snake...)
	ate := g.handleFoodCollision(nextHead, p, idx == 0)

	// --- MAGNET EFFECT ---
	if !ate {
		hasMagnet := false
		for _, e := range p.Effects {
			if e.Type == EffectMagnet {
				hasMagnet = true
				break
			}
		}
		if hasMagnet {
			// Check nearby food (radius 3)
			for i := 0; i < len(g.Foods); i++ {
				food := g.Foods[i]
				dx := food.Pos.X - nextHead.X
				dy := food.Pos.Y - nextHead.Y
				if dx*dx+dy*dy <= 9 { // Radius 3 (squared)
					// Magnetize!
					ate = g.handleFoodCollision(food.Pos, p, idx == 0)
					if ate {
						// Food was eaten via magnet, break to avoid multiple eat per turn
						break
					}
				}
			}
		}
	}

	g.handlePropCollision(nextHead, p)
	if !ate {
		p.Snake = p.Snake[:len(p.Snake)-1]
	}
}

func (g *Game) handlePropCollision(pos Point, p *Player) {
	var remaining []Prop
	for _, pr := range g.Props {
		if pr.Pos == pos {
			// Collected!
			if pr.Type == PropTrimmer {
				// Instant effect: shorten snake
				if len(p.Snake) > 5 {
					p.Snake = p.Snake[:len(p.Snake)-3]
					g.SetMessage("✂️ 剪刀手生效！蛇身缩短了")
				} else {
					g.SetMessage("✂️ 太短了，剪不动了")
				}
			} else if pr.Type == PropChestBig {
				p.Score += 120
				g.SetMessageWithType(fmt.Sprintf("%s 哇！开启大宝箱：+120分", pr.GetEmoji()), "bonus")
				g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
					Pos:    pos,
					Amount: 120,
					Label:  "+120",
				})
			} else if pr.Type == PropChestSmall {
				p.Score += 20
				g.SetMessageWithType(fmt.Sprintf("%s 捡到钱袋：+20分", pr.GetEmoji()), "bonus")
				g.ScoreEvents = append(g.ScoreEvents, ScoreEvent{
					Pos:    pos,
					Amount: 20,
					Label:  "+20",
				})
			} else {
				effectType := pr.GetEffectType()
				duration := pr.GetDuration()

				if effectType != EffectNone && duration > 0 {
					found := false
					for _, e := range p.Effects {
						if e.Type == effectType {
							e.ExpireAt = time.Now().Add(duration)
							found = true
							break
						}
					}
					if !found {
						p.Effects = append(p.Effects, &ActiveEffect{
							Type:     effectType,
							ExpireAt: time.Now().Add(duration),
						})
					}
					g.SetMessageWithType(fmt.Sprintf("%s 拾取道具: %s!", pr.GetEmoji(), effectType), "bonus")
				} else if pr.Type != PropTrimmer {
					// Fallback for props that are not instant and have no effect type (shouldn't happen with current enum)
					log.Printf("Warning: Prop collected with no action: %v", pr.Type)
				}
			}
		} else {
			remaining = append(remaining, pr)
		}
	}
	g.Props = remaining
}

func (g *Game) checkCollisionForPlayer(idx int, p Point) bool {
	// Wall
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return true
	}

	// Check against all snake bodies
	for i, player := range g.Players {
		body := player.Snake
		// If it's another player, they are static right now, but we don't collide with their tail
		// if they are updated in a regular loop where tails move.
		// For simplicity, we collide with the whole body except the tail if length > 1
		if len(body) > 1 {
			body = body[:len(body)-1]
		}
		for _, s := range body {
			if s == p {
				return true
			}
		}

		// Head-on collision check (against current head positions of others)
		if i != idx && len(player.Snake) > 0 && player.Snake[0] == p {
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

func (g *Game) checkCollisionFair(idx int, newHeads []Point) bool {
	p := newHeads[idx]
	// Wall
	if p.X <= 0 || p.X >= config.Width-1 || p.Y <= 0 || p.Y >= config.Height-1 {
		return true
	}

	// Check against all snake bodies
	for _, player := range g.Players {
		// If snake is longer than 1, the tail will move out unless it ate food.
		// For fairness and fluidity, we usually don't collide with the very last segment
		// if the snake is moving.
		bodyToCheck := player.Snake
		if len(bodyToCheck) > 1 {
			bodyToCheck = bodyToCheck[:len(bodyToCheck)-1]
		}
		for _, s := range bodyToCheck {
			if s == p {
				return true
			}
		}
	}

	// Check against other players' new heads (Head-on)
	for i, otherHead := range newHeads {
		if i != idx && otherHead == p {
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
			switch {
			case s1 > s2:
				g.Winner = "player"
			case s2 > s1:
				g.Winner = "ai"
			default:
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

	cooldown := config.FireballCooldown
	for _, e := range p.Effects {
		if e.Type == EffectRapidFire {
			cooldown = config.FireballCooldown / 2
			break
		}
	}

	if time.Since(p.LastFireTime) < cooldown {
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

	// SCATTER SHOT: Shoot two diagonal bullets if effect is active
	for _, e := range p.Effects {
		if e.Type == EffectScatterShot {
			// Diagonal 1
			d1 := Point{X: p.Direction.X, Y: p.Direction.Y}
			if d1.X != 0 {
				d1.Y = 1
			} else {
				d1.X = 1
			}

			// Diagonal 2
			d2 := Point{X: p.Direction.X, Y: p.Direction.Y}
			if d2.X != 0 {
				d2.Y = -1
			} else {
				d2.X = -1
			}

			g.Fireballs = append(g.Fireballs, &Fireball{
				Pos:       p.Snake[0],
				Dir:       d1,
				SpawnTime: time.Now(),
				Owner:     owner,
			}, &Fireball{
				Pos:       p.Snake[0],
				Dir:       d2,
				SpawnTime: time.Now(),
				Owner:     owner,
			})
			break
		}
	}

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
		hit := false
		steps := 1
		ownerIdx := 0
		if fb.Owner == "ai" {
			ownerIdx = 1
		}
		if ownerIdx < len(g.Players) {
			for _, e := range g.Players[ownerIdx].Effects {
				if e.Type == EffectRapidFire {
					steps = 2
					break
				}
			}
		}

		for s := 0; s < steps; s++ {
			fb.Pos.X += fb.Dir.X
			fb.Pos.Y += fb.Dir.Y

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
							selfIdx := 0
							if fb.Owner == "ai" {
								selfIdx = 1
							}
							if pIdx == selfIdx && i == 0 {
								continue
							}

							hit = true
							g.HitPoints = append(g.HitPoints, fb.Pos)

							// Hit logic
							targetPlayer := player
							attackerIdx := 0
							if fb.Owner == "ai" {
								attackerIdx = 1
							}

							var attackerScore int
							var label string

							if i == 0 {
								attackerScore = 50
								label = "🎯 HEADSHOT +50"
								targetPlayer.StunnedUntil = time.Now().Add(2 * time.Second)
								if pIdx == 0 {
									g.SetMessageWithType("😱 警告！头部被击中，麻痹2秒！", "important")
								}
							} else {
								attackerScore = 10
								label = "🔥 HIT +10"
								segmentsToRemove := 1
								if len(targetPlayer.Snake) > segmentsToRemove+1 {
									targetPlayer.Snake = targetPlayer.Snake[:len(targetPlayer.Snake)-segmentsToRemove]
								}
							}

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

							curOwnerIdx := 0
							if fb.Owner == "ai" {
								curOwnerIdx = 1
							}
							if curOwnerIdx < len(g.Players) {
								g.Players[curOwnerIdx].Score += 10
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

			if hit {
				break
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
	}
	state.IsPVP = g.IsPVP
	state.Props = g.Props
	if len(g.Players) > 0 {
		state.P1Effects = g.Players[0].Effects
	}
	if len(g.Players) > 1 {
		state.P2Effects = g.Players[1].Effects
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
