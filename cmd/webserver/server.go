package main

// server.go holds the per-connection GameServer: the state owned by a single
// WebSocket client and the logic that mutates its game.
//
// Concurrency model
// -----------------
// Every connection runs two goroutines that touch the same game state: the
// input goroutine (reads client messages) and the game-loop goroutine (ticks
// the game). They are serialized as follows:
//
//   - Solo modes (zen / battle): the game belongs exclusively to this
//     connection. All access goes through gs.mu.
//   - PVP: the game is shared between two connections, so it is guarded by the
//     match-wide Match.Mu instead. gs.mu still guards this connection's own
//     scalar fields.
//
// Lock ordering (never violate, or we risk deadlock):
//
//	clientsMu -> gs.mu          (kicking a stale session)
//	Match.Mu  -> gs.mu          (match teardown writing per-player fields)
//	MatchMaker.mu -> gs.mu      (pairing writes per-player fields)
//
// gs.mu is NEVER held while acquiring Match.Mu, clientsMu or MatchMaker.mu:
// callers read gs.match under gs.mu, release it, then take the match lock.
//
// Helper methods (update, getGameState, startGame, recording, boost, ...) all
// assume the appropriate lock is ALREADY held by the caller and never lock
// themselves. Only the entry points (handleAction and the loops in
// connection.go) acquire locks.

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
	pb "github.com/trytobebee/snake_go/pkg/proto"
)

// GameServer is the per-connection game session.
type GameServer struct {
	// mu guards the game (in solo mode) and every mutable field below.
	// In PVP the shared game is guarded by Match.Mu instead; mu still guards
	// this connection's own scalar fields (match, role, user, started, ...).
	mu sync.Mutex

	game       *game.Game
	match      *Match // non-nil while in a PVP match
	role       string // "p1" or "p2" for PVP, "solo" otherwise
	user       *game.User
	started    bool
	searching  bool
	boosting   bool
	difficulty string
	ticker     *time.Ticker

	// Boost tracking
	tickCount           int
	lastBoostKeyTime    time.Time
	lastDirKeyTime      time.Time
	lastDirKeyDir       game.Point
	consecutiveKeyCount int
	fireballTickCount   int
	aiTickCount         int
	currentMode         string
	userUpdated         bool
	lbUpdated           bool

	// Recording info
	stepID        int
	firedThisStep bool
	connID        string
	sessionStart  time.Time

	// Connection management. writeMu serializes concurrent writes to the
	// underlying WebSocket and is independent of mu.
	writeMu sync.Mutex
	sendMsg func(v *pb.ServerMessage) error
	close   func() // closes the connection
}

func NewGameServer(connID string, width, height int) *GameServer {
	gs := &GameServer{
		game:        game.NewGame(width, height),
		ticker:      time.NewTicker(config.BaseTick),
		difficulty:  "mid",
		currentMode: "battle",
		connID:      connID,
	}
	gs.game.TimerStarted = false
	return gs
}

// --- Locked accessors -------------------------------------------------------
// These take gs.mu briefly so other goroutines can safely read a field. They
// must NOT be called while gs.mu is already held.

func (gs *GameServer) currentMatch() *Match {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.match
}

func (gs *GameServer) currentUser() *game.User {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.user
}

func (gs *GameServer) setUser(u *game.User) {
	gs.mu.Lock()
	gs.user = u
	gs.mu.Unlock()
}

// --- Game state snapshot ----------------------------------------------------
// getGameState assumes the active game's lock is held by the caller.

func (gs *GameServer) getGameState() game.GameState {
	state := gs.game.GetGameStateSnapshot(gs.started, gs.boosting, gs.difficulty)

	// Clear one-shot events after capture so the client does not render
	// duplicate floating bubbles on the next frame.
	gs.game.ScoreEvents = nil
	gs.game.Message = ""
	gs.game.MessageType = ""

	return state
}

// --- Recording (lock held by caller) ----------------------------------------

func (gs *GameServer) startRecording() {
	if gs.game.Recorder != nil {
		return // already recording
	}
	sessionID := fmt.Sprintf("game_%d_conn_%s", time.Now().UnixNano(), gs.connID)
	recorder, err := game.NewRecorder(sessionID)
	if err == nil {
		gs.game.Recorder = recorder
		gs.stepID = 0
		log.Printf("🔴 Recording started: %s\n", sessionID)
	} else {
		log.Println("❌ Failed to start recording:", err)
	}
}

func (gs *GameServer) stopRecording() {
	if gs.game.Recorder != nil {
		gs.game.Recorder.Close()
		gs.game.Recorder = nil
		log.Println("⏹️ Recording stopped")
	}
}

// --- Boost detection (lock held by caller) ----------------------------------

func (gs *GameServer) checkBoostKey(inputDir game.Point) {
	now := time.Now()

	if inputDir == gs.lastDirKeyDir && time.Since(gs.lastDirKeyTime) < config.KeyRepeatWindow {
		gs.consecutiveKeyCount++
	} else {
		gs.consecutiveKeyCount = 1
	}

	gs.lastDirKeyDir = inputDir
	gs.lastDirKeyTime = now

	if len(gs.game.Players) > 0 {
		p := gs.game.Players[0]
		if gs.role == "p2" && len(gs.game.Players) > 1 {
			p = gs.game.Players[1]
		}

		if gs.consecutiveKeyCount >= config.BoostThreshold && inputDir == p.Direction {
			gs.boosting = true
			gs.lastBoostKeyTime = now
		}
	}
}

func (gs *GameServer) startGame() {
	if gs.started || gs.game.GameOver {
		return
	}
	gs.started = true
	gs.tickCount = 0
	gs.game.TimerStarted = true
	gs.game.StartTime = time.Now()
	gs.sessionStart = time.Now()
	gs.game.LastFoodSpawn = time.Now()
	if len(gs.game.Foods) > 0 {
		gs.game.Foods[0].SpawnTime = time.Now()
		gs.game.Foods[0].PausedTimeAtSpawn = gs.game.GetTotalPausedTime()
	}
	gs.startRecording()
}

// --- Action handling --------------------------------------------------------

// handleAction is an entry point: it acquires the correct lock for the current
// mode and applies the action. Matchmaking actions manage their own
// synchronization and are dispatched before any game lock is taken.
func (gs *GameServer) handleAction(action string, mode string) {
	switch action {
	case "find_match":
		if gs.currentUser() != nil {
			pvpManager.FindMatch(gs)
		}
		return
	case "cancel_match":
		pvpManager.CancelSearch(gs)
		return
	case "submit_score", "register", "login":
		// No-op here: handled in the auth loop or automatic on game over.
		return
	}

	// Hold gs.mu while deciding the mode. In solo we apply the action under
	// gs.mu directly: because attachToMatch also takes gs.mu, the match cannot
	// be created underneath us. In PVP we hand off to the match lock.
	gs.mu.Lock()
	if gs.match == nil {
		gs.applyAction(action, mode)
		gs.mu.Unlock()
		return
	}
	m := gs.match
	gs.mu.Unlock()

	m.Mu.Lock()
	gs.applyAction(action, mode)
	m.Mu.Unlock()
}

// applyAction mutates the active game. The caller MUST hold the active game's
// lock (gs.mu for solo, Match.Mu for PVP).
func (gs *GameServer) applyAction(action string, mode string) {
	var inputDir game.Point
	var isDirection bool

	switch action {
	case "up":
		inputDir = game.Point{X: 0, Y: -1}
		isDirection = true
	case "down":
		inputDir = game.Point{X: 0, Y: 1}
		isDirection = true
	case "left":
		inputDir = game.Point{X: -1, Y: 0}
		isDirection = true
	case "right":
		inputDir = game.Point{X: 1, Y: 0}
		isDirection = true
	case "pause":
		if !gs.game.GameOver {
			if !gs.started {
				gs.startGame()
			} else {
				gs.game.TogglePause()
			}
		}
	case "start":
		gs.startGame()
	case "restart":
		// Force stop even if the game wasn't over (shouldn't happen with current UI logic).
		gs.stopRecording()

		if gs.game.GameOver {
			gs.game = game.NewGame(gs.game.Width, gs.game.Height)
			gs.game.Mode = gs.currentMode
			gs.game.TimerStarted = false
			gs.started = false
			gs.boosting = false
			gs.tickCount = 0
			gs.consecutiveKeyCount = 0
		}
	case "mode_zen":
		gs.currentMode = "zen"
		gs.game.Mode = "zen"
		if len(gs.game.Players) > 1 {
			gs.game.Players = gs.game.Players[:1] // Remove AI
		}
	case "mode_battle":
		gs.currentMode = "battle"
		gs.game.Mode = "battle"
		if len(gs.game.Players) < 2 {
			// Decide which AI brain to use based on dimensions.
			var brain game.Controller = &game.HeuristicController{}
			controller := "heuristic"
			if gs.game.NeuralNet != nil && gs.game.Width == config.StandardWidth && gs.game.Height == config.StandardHeight {
				brain = &game.NeuralController{}
				controller = "neural"
			}

			gs.game.Players = append(gs.game.Players, &game.Player{
				Snake:       []game.Point{{X: gs.game.Width - 2, Y: gs.game.Height - 2}},
				Direction:   game.Point{X: -1, Y: 0},
				LastMoveDir: game.Point{X: -1, Y: 0},
				Name:        "AI",
				Brain:       brain,
				Controller:  controller,
			})
		}
	case "diff_low":
		if !gs.started || gs.game.GameOver {
			gs.difficulty = "low"
		}
	case "diff_mid":
		if !gs.started || gs.game.GameOver {
			gs.difficulty = "mid"
		}
	case "diff_high":
		if !gs.started || gs.game.GameOver {
			gs.difficulty = "high"
		}
	case "auto":
		if !gs.game.GameOver {
			pIdx := 0
			if gs.role == "p2" {
				pIdx = 1
			}
			gs.game.TogglePlayerAutoPlay(pIdx, mode)
		}
	case "fire":
		if !gs.game.GameOver && !gs.game.Paused {
			if gs.role == "p2" {
				gs.game.FireByTypeIdx(1)
			} else {
				gs.game.FireByTypeIdx(0)
			}
			gs.firedThisStep = true
		}
	case "toggleBerserker":
		if !gs.game.GameOver {
			gs.game.ToggleBerserkerMode()
		}
	}

	if isDirection {
		if !gs.started && gs.role != "p1" && gs.role != "p2" {
			gs.startGame()
		}

		var dirChanged bool
		pIdx := 0
		if gs.role == "p2" {
			pIdx = 1
		}

		if pIdx < len(gs.game.Players) {
			p := gs.game.Players[pIdx]
			if mc, ok := p.Brain.(*game.ManualController); ok {
				mc.SetDirection(inputDir)
				// We still need to know for local boost logic if the direction actually changed on the body.
				dirChanged = gs.game.SetPlayerDirection(pIdx, inputDir)
			}
		}

		if dirChanged {
			// Direction changed, reset boost.
			gs.consecutiveKeyCount = 1
			gs.lastDirKeyDir = inputDir
			gs.lastDirKeyTime = time.Now()
			gs.boosting = false
		} else {
			// Same direction, check for boost.
			gs.checkBoostKey(inputDir)
		}
	}
}

// updateBoostingOnly syncs this connection's boost state into the shared game
// controller. The caller MUST hold the active game's lock.
func (gs *GameServer) updateBoostingOnly() {
	if gs.boosting && time.Since(gs.lastBoostKeyTime) > config.BoostTimeout {
		gs.boosting = false
	}
	pIdx := 0
	if gs.role == "p2" {
		pIdx = 1
	}
	if pIdx < len(gs.game.Players) {
		p := gs.game.Players[pIdx]
		if mc, ok := p.Brain.(*game.ManualController); ok {
			mc.SetBoosting(gs.boosting)
		}
	}
}

// isOthersTimeWarpActive reports whether an opponent has TIMEWARP active.
// The caller MUST hold the active game's lock.
func (gs *GameServer) isOthersTimeWarpActive() bool {
	myIdx := 0
	if gs.role == "p2" {
		myIdx = 1
	}
	for i, p := range gs.game.Players {
		if i == myIdx {
			continue
		}
		for _, e := range p.Effects {
			if e.Type == game.EffectTimeWarp {
				return true
			}
		}
	}
	return false
}

// update advances the solo game by one tick and reports whether anything
// changed. The caller MUST hold gs.mu.
func (gs *GameServer) update() bool {
	changed := false

	// Sync manual boosting state to game if not in AutoPlay.
	gs.updateBoostingOnly()

	// 1. Clear per-frame events at the start of the update cycle.
	gs.game.HitPoints = nil
	gs.game.ScoreEvents = nil

	gs.tickCount++

	// 2. Movement logic (only if started).
	if gs.started {
		// Determine tick threshold based on difficulty and boosting.
		ticksNeeded := config.MidTicks
		boostTicks := config.MidBoostTicks

		switch gs.difficulty {
		case "low":
			ticksNeeded = config.LowTicks
			boostTicks = config.LowBoostTicks
		case "mid":
			ticksNeeded = config.MidTicks
			boostTicks = config.MidBoostTicks
		case "high":
			ticksNeeded = config.HighTicks
			boostTicks = config.HighBoostTicks
		}

		isBoosted := false
		if len(gs.game.Players) > 0 {
			isBoosted = gs.game.Players[0].Boosting
			if gs.role == "p2" && len(gs.game.Players) > 1 {
				isBoosted = gs.game.Players[1].Boosting
			}
		}

		if isBoosted {
			ticksNeeded = boostTicks
		}

		// Prop effects: if opponents have TimeWarp, this player is slowed.
		if gs.isOthersTimeWarpActive() {
			ticksNeeded = ticksNeeded * 2
		}

		if gs.tickCount >= ticksNeeded {
			gs.tickCount = 0
			if !gs.game.GameOver && !gs.game.Paused {
				playerIdx := 0
				if gs.role == "p2" {
					playerIdx = 1
				}
				if playerIdx < len(gs.game.Players) {
					gs.game.UpdatePlayer(playerIdx)
					changed = true
				}

				// --- Recording ---
				if gs.game.Recorder != nil && len(gs.game.Players) > 0 {
					p1 := gs.game.Players[0]
					snapshot := gs.game.GetGameStateSnapshot(gs.started, gs.boosting, gs.difficulty)

					reward := float64(p1.Score - gs.game.LastScore)
					if gs.game.GameOver && gs.game.Winner != "player" {
						reward -= 100.0 // Death penalty
					} else if !gs.game.GameOver {
						reward += 0.1 // Survival bonus
					}
					gs.game.LastScore = p1.Score

					actionData := game.ActionData{
						Direction: p1.LastMoveDir,
						Boost:     p1.Boosting,
						Fire:      gs.firedThisStep,
					}
					gs.firedThisStep = false

					rec := game.StepRecord{
						StepID:    gs.stepID,
						Timestamp: time.Now().UnixMilli(),
						State:     snapshot,
						Action:    actionData,
						AIContext: gs.game.CurrentAIContext,
						Reward:    reward,
						Done:      gs.game.GameOver,
					}
					gs.game.Recorder.RecordStep(rec)
					gs.stepID++
				}
			}
		}
	}

	// Move AI snake independently (if any).
	if gs.started && !gs.game.IsPVP && len(gs.game.Players) > 1 {
		gs.aiTickCount++
		aiTicksNeeded := config.MidTicks
		if gs.game.Players[1].Boosting {
			aiTicksNeeded = config.MidBoostTicks
		}

		// If P1 has TimeWarp, the AI is slowed.
		p1HasWarp := false
		if len(gs.game.Players) > 0 {
			for _, e := range gs.game.Players[0].Effects {
				if e.Type == game.EffectTimeWarp {
					p1HasWarp = true
					break
				}
			}
		}
		if p1HasWarp {
			aiTicksNeeded = aiTicksNeeded * 2
		}

		if gs.aiTickCount >= aiTicksNeeded {
			gs.aiTickCount = 0
			if !gs.game.GameOver && !gs.game.Paused {
				gs.game.UpdatePlayer(1)
				changed = true
			}
		}
	}

	// Periodic world update (food, obstacles, time limit).
	if !gs.game.GameOver && !gs.game.Paused && (gs.started || gs.game.Mode == "pvp") {
		gs.game.TrySpawnFood()
		gs.game.TrySpawnProp()
		gs.game.TrySpawnObstacle()
		gs.game.CheckTimeLimit()
	}

	// Update fireballs independently at FireballSpeed.
	if gs.started {
		gs.fireballTickCount++
		fbTicks := int(config.FireballSpeed / config.BaseTick)
		if gs.fireballTickCount >= fbTicks {
			gs.fireballTickCount = 0
			if !gs.game.GameOver && !gs.game.Paused {
				gs.game.UpdateFireballs()
				changed = true
			}
		}
	}

	// Any message or special event also counts as a change that must be sent.
	if gs.game.Message != "" || len(gs.game.HitPoints) > 0 || len(gs.game.ScoreEvents) > 0 || gs.game.GameOver {
		changed = true
	}

	// Handle game-over logic (stats, leaderboard, recording).
	if gs.game.GameOver {
		if gs.handleGameOver() {
			changed = true
		}
	}
	return changed
}

// handleGameOver finalizes recording, stats and leaderboard for the game. It
// returns true if per-session stats were submitted (which the caller treats as
// a change worth broadcasting). The caller MUST hold the active game's lock.
func (gs *GameServer) handleGameOver() bool {
	// 1. Stop recording if it's still running.
	if gs.game.Recorder != nil && len(gs.game.Players) > 0 {
		p1 := gs.game.Players[0]
		snapshot := gs.game.GetGameStateSnapshot(gs.started, gs.boosting, gs.difficulty)
		reward := float64(p1.Score - gs.game.LastScore)

		rec := game.StepRecord{
			StepID:    gs.stepID,
			Timestamp: time.Now().UnixMilli(),
			State:     snapshot,
			Action: game.ActionData{
				Direction: p1.LastMoveDir,
				Boost:     p1.Boosting,
				Fire:      false,
			},
			AIContext: gs.game.CurrentAIContext,
			Reward:    reward,
			Done:      true,
		}
		gs.game.Recorder.RecordStep(rec)
		gs.stopRecording()
	}

	// 2. Automatic score/stats submission, once per session (gs.started is the guard).
	if !gs.started || gs.user == nil || len(gs.game.Players) == 0 {
		return false
	}

	log.Printf("🏁 Game Over detected for user %s. Processing stats (Winner: %s, IsPVP: %v)...\n", gs.user.Username, gs.game.Winner, gs.game.IsPVP)
	isBattle := gs.game.Mode == "battle"

	// In PVP, 'player' means P1 wins, 'ai' means P2 wins.
	won := false
	if gs.game.IsPVP {
		switch gs.role {
		case "p1":
			won = gs.game.Winner == "player"
		case "p2":
			won = gs.game.Winner == "ai"
		}
	} else {
		won = gs.game.Winner == "player"
	}

	playerScore := gs.game.Players[0].Score
	if gs.role == "p2" && len(gs.game.Players) > 1 {
		playerScore = gs.game.Players[1].Score
	}

	if updatedUser, err := userManager.UpdateStats(gs.user.Username, playerScore, won); err == nil {
		gs.user = updatedUser
		gs.userUpdated = true
		log.Printf("📈 Updated stats for %s: Best Score = %d\n", gs.user.Username, gs.user.BestScore)
	}

	// Only Battle Mode goes to the global leaderboard.
	if isBattle && playerScore > 0 {
		log.Printf("🏆 Submitting Battle Mode score (%d) to leaderboard...\n", playerScore)
		if lbManager.AddEntry(gs.user.Username, playerScore, gs.difficulty, gs.game.Mode) {
			gs.lbUpdated = true
		}
	}

	if *detailedLogs {
		game.RecordGameSession(gs.user.Username, gs.sessionStart, time.Now(), playerScore, gs.game.Winner, gs.game.Mode, gs.difficulty)
	}

	// Mark as processed for this session.
	gs.started = false
	return true
}
