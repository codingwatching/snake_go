package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"crypto/rand"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
	pb "github.com/trytobebee/snake_go/pkg/proto"
	"google.golang.org/protobuf/proto"
)

var (
	detailedLogs = flag.Bool("detailed-logs", false, "Enable detailed session logging to database")
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

const (
	MaxPlayers = 500
)

// Global map to track active IP connections
var (
	clientsMu sync.RWMutex
	clients   = make(map[string]*GameServer)
)

// Global leaderboard manager
var lbManager = game.NewLeaderboardManager()

// Global user manager
var userManager = game.NewUserManager()

// PVP Matchmaking
type Match struct {
	Game    *game.Game
	P1      *GameServer
	P2      *GameServer
	Mu      sync.Mutex
	Closing bool
}

type MatchMaker struct {
	mu      sync.Mutex
	waiting *GameServer
}

var pvpManager = &MatchMaker{}

type GameServer struct {
	game       *game.Game
	match      *Match // Shared match if in PVP
	role       string // "p1" or "p2" for PVP, "solo" for others
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

	// Connection management
	writeMu sync.Mutex
	sendMsg func(v *pb.ServerMessage) error
	close   func() // Function to close the connection
}

// ... (ServerMessage, ClientMessage structs unchanged)

// ... (ServerMessage, ClientMessage structs unchanged)
// No longer using internal ServerMessage/ClientMessage as they are now in proto package

func NewGameServer(connID string) *GameServer {
	gs := &GameServer{
		game:        game.NewGame(),
		ticker:      time.NewTicker(config.BaseTick),
		difficulty:  "mid",
		currentMode: "battle",
		connID:      connID,
	}
	gs.game.TimerStarted = false
	return gs
}

func (gs *GameServer) getGameState() game.GameState {
	state := gs.game.GetGameStateSnapshot(gs.started, gs.boosting, gs.difficulty)

	// Important: Clear events after they are captured for the current state update
	// to prevent the client from creating duplicate floating bubbles.
	gs.game.ScoreEvents = nil
	gs.game.Message = ""
	gs.game.MessageType = ""

	return state
}

func (gs *GameServer) startRecording() {
	if gs.game.Recorder != nil {
		return // Already recording
	}
	sessionID := fmt.Sprintf("game_%d_conn_%s", time.Now().UnixNano(), gs.connID)
	// Sanitize filename safe chars
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

func (mm *MatchMaker) FindMatch(gs *GameServer) {
	mm.mu.Lock()
	if mm.waiting == nil {
		gs.searching = true
		mm.waiting = gs
		mm.mu.Unlock()
		log.Printf("[PVP] ⏳ Player %s entered matchmaking queue (waiting for opponent)\n", gs.user.Username)
		return
	}

	// Double-check: Prevent matching with oneself (same user, different connection)
	if mm.waiting.user.Username == gs.user.Username {
		log.Printf("[PVP] ⚠️ Player %s tried to match with themselves (another session). Keeping original session in queue.\n", gs.user.Username)
		mm.mu.Unlock()
		// Optionally: Send a message to the client saying "Still waiting for others..."
		return
	}

	// Found a pair!
	p1 := mm.waiting
	p2 := gs
	mm.waiting = nil
	p1.searching = false
	p2.searching = false
	mm.mu.Unlock()

	log.Printf("[PVP] ⚔️ Match found: %s (P1) vs %s (P2). Initializing shared game state...\n", p1.user.Username, p2.user.Username)

	// Create shared game
	sharedGame := game.NewGame()
	sharedGame.Mode = "pvp"
	sharedGame.IsPVP = true
	sharedGame.Paused = true // Start paused for countdown

	// Reset players for PVP symmetry - Start them at different Y positions to avoid head-on crash
	sharedGame.Players = []*game.Player{
		{
			Snake:       []game.Point{{X: config.Width / 4, Y: config.Height / 3}},
			Direction:   game.Point{X: 1, Y: 0},
			LastMoveDir: game.Point{X: 1, Y: 0},
			Name:        p1.user.Username,
			Brain:       &game.ManualController{},
			Controller:  "manual",
		},
		{
			Snake:       []game.Point{{X: (config.Width * 3) / 4, Y: (config.Height * 2) / 3}},
			Direction:   game.Point{X: -1, Y: 0},
			LastMoveDir: game.Point{X: -1, Y: 0},
			Name:        p2.user.Username,
			Brain:       &game.ManualController{},
			Controller:  "manual",
		},
	}

	match := &Match{
		Game: sharedGame,
		P1:   p1,
		P2:   p2,
	}

	p1.match = match
	p1.role = "p1"
	p1.game = sharedGame

	p2.match = match
	p2.role = "p2"
	p2.game = sharedGame

	log.Printf("[PVP] 🔗 Both players attached to Match. P1: %s, P2: %s. Sending initial MATCH FOUND msg.\n", p1.user.Username, p2.user.Username)

	// Initial broadcast with "MATCH FOUND"
	st := sharedGame.GetGameStateSnapshot(true, false, "mid")
	st.Message = "⚔️ MATCH FOUND!"
	st.MessageType = "important"

	p1.sendMsg(pb.ToProtoServerMessage("state", nil, &st, nil, nil, nil, "", "", 0))
	p2.sendMsg(pb.ToProtoServerMessage("state", nil, &st, nil, nil, nil, "", "", 0))

	log.Printf("[PVP] ⏱️ Starting 3-second countdown for %s vs %s\n", p1.user.Username, p2.user.Username)
	go mm.runPVPCountdown(match)
}

func (mm *MatchMaker) CancelSearch(gs *GameServer) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	if mm.waiting == gs {
		mm.waiting = nil
		gs.searching = false
		log.Printf("👋 Player %s removed from matchmaking queue (disconnected/left)\n", gs.user.Username)
	}
}

func (mm *MatchMaker) runPVPCountdown(m *Match) {
	for i := 3; i > 0; i-- {
		m.Mu.Lock()
		if m.Closing {
			m.Mu.Unlock()
			return
		}
		m.Game.Message = fmt.Sprintf("🔥 STARTING IN %d...", i)
		m.Game.MessageType = "important"
		state := m.Game.GetGameStateSnapshot(true, false, m.P1.difficulty)
		m.Mu.Unlock()

		log.Printf("[PVP] 🔔 Countdown: %d... (Players: %s, %s)\n", i, m.P1.user.Username, m.P2.user.Username)

		// Send personalized state to P1
		stateP1 := state
		stateP1.Message = fmt.Sprintf("🟢 YOU ARE PLAYER 1 (GREEN)\nSTARTING IN %d...", i)
		m.P1.sendMsg(pb.ToProtoServerMessage("state", nil, &stateP1, nil, nil, nil, "", "", 0))

		// Send personalized state to P2
		stateP2 := state
		stateP2.Message = fmt.Sprintf("🟣 YOU ARE PLAYER 2 (PURPLE)\nSTARTING IN %d...", i)
		m.P2.sendMsg(pb.ToProtoServerMessage("state", nil, &stateP2, nil, nil, nil, "", "", 0))

		time.Sleep(1 * time.Second)
	}

	m.Mu.Lock()
	if m.Closing {
		m.Mu.Unlock()
		return
	}
	m.Game.Message = "🚀 GO!"
	m.Game.MessageType = "important"
	m.Game.Paused = false
	m.Game.TimerStarted = true
	m.Game.StartTime = time.Now()
	// Set both participants to started state so their update logic runs
	m.P1.started = true
	m.P2.started = true
	m.Mu.Unlock()

	log.Printf("[PVP] 🚀 Rocket Start! Game is now UNPAUSED for %s vs %s\n", m.P1.user.Username, m.P2.user.Username)
	go mm.runPVPGame(m)
}

func (mm *MatchMaker) runPVPGame(m *Match) {
	ticker := time.NewTicker(config.BaseTick)
	defer ticker.Stop()

	for range ticker.C {
		m.Mu.Lock()
		if m.Closing {
			m.Mu.Unlock()
			return
		}

		// Update game logic for both players
		c1 := m.P1.update()
		c2 := m.P2.update()
		changed := c1 || c2

		if changed {
			state := m.Game.GetGameStateSnapshot(true, false, m.P1.difficulty)

			// Broadcast to both
			m.P1.sendMsg(pb.ToProtoServerMessage("state", nil, &state, nil, nil, nil, "", "", 0))
			m.P2.sendMsg(pb.ToProtoServerMessage("state", nil, &state, nil, nil, nil, "", "", 0))

			// Reset one-shot effects ONLY after broadcast
			m.Game.ScoreEvents = nil
			m.Game.HitPoints = nil
			m.Game.Message = ""
			m.Game.MessageType = ""
		}

		if m.Game.GameOver {
			log.Printf("[PVP] 🏁 Match Over detected in loop (%s vs %s). Winner: %s\n", m.P1.user.Username, m.P2.user.Username, m.Game.Winner)
			m.Closing = true

			// Handle stats for both players
			m.handleMatchOver()

			m.Mu.Unlock()
			return
		}
		m.Mu.Unlock()
	}

	// Double check: if match ended without disconnect, ensure roles are reset
	m.Mu.Lock()
	if !m.Closing {
		m.handleMatchOver()
	}
	m.Mu.Unlock()
}

func (m *Match) handleMatchOver() {
	gameObj := m.Game

	// Update Stats for P1
	if m.P1.user != nil && len(gameObj.Players) > 0 {
		won := gameObj.Winner == "player"
		updated, _ := userManager.UpdateStats(m.P1.user.Username, gameObj.Players[0].Score, won)
		if updated != nil {
			m.P1.user = updated
			m.P1.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, updated, "", "", 0))
		}
	}

	// Update Stats for P2
	if m.P2.user != nil && len(gameObj.Players) > 1 {
		won := gameObj.Winner == "ai"
		updated, _ := userManager.UpdateStats(m.P2.user.Username, gameObj.Players[1].Score, won)
		if updated != nil {
			m.P2.user = updated
			m.P2.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, updated, "", "", 0))
		}
	}

	// Sessions
	if *detailedLogs {
		log.Printf("[PVP] 📝 Recording detailed game sessions for both players...\n")
		if m.P1.user != nil && len(gameObj.Players) > 0 {
			p1Res := "lost"
			if gameObj.Winner == "player" {
				p1Res = "won"
			} else if gameObj.Winner == "draw" {
				p1Res = "draw"
			}
			game.RecordGameSession(m.P1.user.Username, m.P1.sessionStart, time.Now(), gameObj.Players[0].Score, p1Res, "pvp", m.P1.difficulty)
		}
		if m.P2.user != nil && len(gameObj.Players) > 1 {
			var p2Res string
			switch gameObj.Winner {
			case "ai":
				p2Res = "won"
			case "draw":
				p2Res = "draw"
			default:
				p2Res = "lost"
			}
			game.RecordGameSession(m.P2.user.Username, m.P2.sessionStart, time.Now(), gameObj.Players[1].Score, p2Res, "pvp", m.P2.difficulty)
		}
	}

	log.Printf("[PVP] 🔓 Detaching players from match and resetting to solo state (%s, %s)\n", m.P1.user.Username, m.P2.user.Username)
	// Crucial: Detach players from the match so they can resume solo or re-queue
	m.P1.match = nil
	m.P1.role = "solo"
	m.P1.started = false

	m.P2.match = nil
	m.P2.role = "solo"
	m.P2.started = false
}

func (gs *GameServer) handleAction(action string, mode string) {
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
		// Force stop even if game wasn't over (shouldn't happen with current UI logic)
		gs.stopRecording()

		if gs.game.GameOver {
			gs.game = game.NewGame()
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
			gs.game.Players = append(gs.game.Players, &game.Player{
				Snake:       []game.Point{{X: config.Width - 2, Y: config.Height - 2}},
				Direction:   game.Point{X: -1, Y: 0},
				LastMoveDir: game.Point{X: -1, Y: 0},
				Name:        "AI",
				Brain:       &game.HeuristicController{},
				Controller:  "heuristic",
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
	case "find_match":
		if gs.user != nil {
			pvpManager.FindMatch(gs)
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
	case "submit_score":
		// Removed manual submission as it's now automatic
	case "register":
		// Handled in auth loop
	case "login":
		// Handled in auth loop
	case "cancel_match":
		pvpManager.CancelSearch(gs)
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
				// We still need to know for local boost logic if the direction actually changed on the body
				dirChanged = gs.game.SetPlayerDirection(pIdx, inputDir)
			}
		}

		if dirChanged {
			// Direction changed, reset boost
			gs.consecutiveKeyCount = 1
			gs.lastDirKeyDir = inputDir
			gs.lastDirKeyTime = time.Now()
			gs.boosting = false
		} else {
			// Same direction, check for boost
			gs.checkBoostKey(inputDir)
		}
	}
}

// ... (checkBoostKey unchanged)

func (gs *GameServer) updateBoostingOnly() {
	// Sync manual boosting state to game controller
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

// Check if any other player has the TIMEWARP effect active
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

func (gs *GameServer) update() bool {
	changed := false

	// Sync manual boosting state to game if not in AutoPlay
	gs.updateBoostingOnly()

	// 1. Clear per-frame events at the start of the update cycle
	gs.game.HitPoints = nil
	gs.game.ScoreEvents = nil

	gs.tickCount++

	// 2. Movement logic (Only if started)
	if gs.started {
		// Determine Tick threshold based on difficulty and boosting
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

		// --- PROP EFFECTS: SPEED & TIMEWARP ---
		// If OTHERS have TimeWarp, I am slowed down
		if gs.isOthersTimeWarpActive() {
			ticksNeeded = ticksNeeded * 2
		}
		// --------------------------------------

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

				// --- Recording Logic ---
				if gs.game.Recorder != nil && len(gs.game.Players) > 0 {
					p1 := gs.game.Players[0]
					snapshot := gs.game.GetGameStateSnapshot(gs.started, gs.boosting, gs.difficulty)

					// Reward Calculation
					reward := float64(p1.Score - gs.game.LastScore)
					if gs.game.GameOver && gs.game.Winner != "player" {
						reward -= 100.0 // Death penalty
					} else if !gs.game.GameOver {
						reward += 0.1 // Survival bonus
					}
					gs.game.LastScore = p1.Score

					// Capture Action
					actionData := game.ActionData{
						Direction: p1.LastMoveDir,
						Boost:     p1.Boosting,
						Fire:      gs.firedThisStep,
					}
					gs.firedThisStep = false // Reset for next step

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
				// -----------------------
			}
		}

	}

	// Move AI snake independently (if any)
	if gs.started && !gs.game.IsPVP && len(gs.game.Players) > 1 {
		gs.aiTickCount++
		aiTicksNeeded := config.MidTicks
		if gs.game.Players[1].Boosting {
			aiTicksNeeded = config.MidBoostTicks
		}

		// If P1 has TimeWarp, AI is slowed
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

	// Periodic World Update (Food, Obstacles, Time Limit) - always check if game is active
	if !gs.game.GameOver && !gs.game.Paused && (gs.started || gs.game.Mode == "pvp") {
		gs.game.TrySpawnFood()
		gs.game.TrySpawnProp()
		gs.game.TrySpawnObstacle()
		gs.game.CheckTimeLimit()
		// Note: we don't necessarily set changed=true here to avoid flooding,
		// but if food or time changed significantly it will be sent in the next snake move anyway.
	}

	// Update fireballs independently at FireballSpeed
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

	// IMPORTANT: Any message or special event also counts as a change that MUST be sent
	if gs.game.Message != "" || len(gs.game.HitPoints) > 0 || len(gs.game.ScoreEvents) > 0 || gs.game.GameOver {
		changed = true
	}

	// Handle Game Over logic (Stats, Leaderboard, Recording)
	if gs.game.GameOver {
		// 1. Stop recording if it's still running
		if gs.game.Recorder != nil && len(gs.game.Players) > 0 {
			p1 := gs.game.Players[0]
			// Capture final state
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

		// 2. Automatic score/stats submission (only once per game session)
		// We use gs.started as a flag since it's set to false on restart
		if gs.started && gs.user != nil && len(gs.game.Players) > 0 {
			log.Printf("🏁 Game Over detected for user %s. Processing stats (Winner: %s, IsPVP: %v)...\n", gs.user.Username, gs.game.Winner, gs.game.IsPVP)
			isBattle := gs.game.Mode == "battle"

			// In PVP, 'player' means P1 wins, 'ai' means P2 wins
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

			p1Score := gs.game.Players[0].Score
			if gs.role == "p2" && len(gs.game.Players) > 1 {
				p1Score = gs.game.Players[1].Score
			}

			// Always update stats
			updatedUser, err := userManager.UpdateStats(gs.user.Username, p1Score, won)
			if err == nil {
				gs.user = updatedUser
				gs.userUpdated = true
				log.Printf("📈 Updated stats for %s: Best Score = %d\n", gs.user.Username, gs.user.BestScore)
			}

			// Only Battle Mode goes to the Global Leaderboard
			if isBattle && p1Score > 0 {
				log.Printf("🏆 Submitting Battle Mode score (%d) to leaderboard...\n", p1Score)
				if lbManager.AddEntry(gs.user.Username, p1Score, gs.difficulty, gs.game.Mode) {
					gs.lbUpdated = true
				}
			}

			// Detailed session logging if enabled
			if *detailedLogs {
				game.RecordGameSession(
					gs.user.Username,
					gs.sessionStart,
					time.Now(),
					p1Score,
					gs.game.Winner,
					gs.game.Mode,
					gs.difficulty,
				)
			}

			// Mark as processed for this session
			gs.started = false
			changed = true
		}
	}
	return changed
}

func notifyFeishu(username, feedback string) {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	// 构造卡片消息，这种格式在飞书中显示最稳定且美观
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": "🐍 贪吃蛇游戏 - 新用户反馈",
				},
				"template": "blue", // 蓝色页眉
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag": "lark_md",
						"content": fmt.Sprintf("**用户ID:** %s\n**反馈时间:** %s\n\n**反馈详情:**\n%s",
							username, time.Now().Format("2006-01-02 15:04:05"), feedback),
					},
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("❌ Failed to send Feishu notification: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ Feishu returned non-OK status: %s\n", resp.Status)
	} else {
		log.Printf("🔔 Feishu card notification sent successfully!\n")
	}
}

func broadcastSessionCount() {
	clientsMu.RLock()
	count := len(clients)
	var targets []*GameServer
	for _, gs := range clients {
		targets = append(targets, gs)
	}
	clientsMu.RUnlock()

	msg := pb.ToProtoServerMessage("update_counts", nil, nil, nil, nil, nil, "", "", count)
	for _, gs := range targets {
		go gs.sendMsg(msg)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("New WebSocket connection from:", r.RemoteAddr)
	// Generate a unique ID for this connection
	b := make([]byte, 8)
	rand.Read(b)
	connID := fmt.Sprintf("%x-%d", b, time.Now().UnixNano())

	gs := NewGameServer(connID)

	// Mutex to protect concurrent writes to the WebSocket connection
	gs.sendMsg = func(v *pb.ServerMessage) error {
		gs.writeMu.Lock()
		defer gs.writeMu.Unlock()
		data, err := proto.Marshal(v)
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.BinaryMessage, data)
	}

	gs.close = func() {
		conn.Close()
	}

	// Check for player limit
	clientsMu.RLock()
	currentCount := len(clients)
	clientsMu.RUnlock()

	if currentCount >= MaxPlayers {
		log.Printf("🚫 Connection rejected: server full (%d/%d)\n", currentCount, MaxPlayers)
		msg := pb.ToProtoServerMessage("error", nil, nil, nil, nil, nil, "Server is full (500/500). Please wait for a player to leave and try refreshing.", "", 0)
		data, _ := proto.Marshal(msg)
		conn.WriteMessage(websocket.BinaryMessage, data)
		return
	}

	// Add to global client tracking for broadcasts
	clientsMu.Lock()
	clients[connID] = gs
	clientsMu.Unlock()

	broadcastSessionCount()

	defer func() {
		clientsMu.Lock()
		delete(clients, connID)
		clientsMu.Unlock()
		broadcastSessionCount()

		// Fix: Remove from matchmaking queue if waiting
		pvpManager.CancelSearch(gs)

		// Fix: Handle PVP match termination if in game
		if gs.match != nil {
			gs.match.Mu.Lock()
			if !gs.match.Closing {
				gs.match.Closing = true
				log.Printf("[PVP] 📡 Match terminated due to %s disconnecting\n", gs.user.Username)
				gs.match.handleMatchOver() // Ensure P2 gets reset to solo mode
			}
			gs.match.Mu.Unlock()
		}

		gs.ticker.Stop()
		gs.stopRecording()
	}()

	// Send initial config
	gameConfig := gs.game.GetGameConfig()
	gs.sendMsg(pb.ToProtoServerMessage("config", &gameConfig, nil, nil, nil, nil, "", "", 0))

	// Send leaderboards
	gs.sendMsg(pb.ToProtoServerMessage("leaderboard", nil, nil, lbManager.GetEntries(), lbManager.GetWinRateEntries(), nil, "", "", 0))

	initialState := gs.getGameState()
	gs.sendMsg(pb.ToProtoServerMessage("state", nil, &initialState, nil, nil, nil, "", "", 0))

	// Signal for loop termination
	done := make(chan struct{})

	// Input handling goroutine
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}

			var msg pb.ClientMessage
			err = proto.Unmarshal(data, &msg)
			if err != nil {
				log.Println("Protobuf unmarshal error:", err)
				continue
			}

			if msg.Action == "login" {
				log.Printf("🔑 Login attempt: %s\n", msg.Username)
				user, err := userManager.Login(msg.Username, msg.Password)
				if err != nil {
					log.Printf("❌ Login failed: %v\n", err)
					gs.sendMsg(pb.ToProtoServerMessage("auth_error", nil, nil, nil, nil, nil, err.Error(), "", 0))
				} else {
					log.Printf("✅ Login success: %s\n", msg.Username)

					// Kick old connection for this user if it exists
					clientsMu.Lock()
					var killee *GameServer
					for id, c := range clients {
						if c.user != nil && c.user.Username == msg.Username && id != connID {
							log.Printf("⚠️ Found old session for user: %s (connID: %s)\n", msg.Username, id)
							killee = c
							break
						}
					}
					clientsMu.Unlock()

					if killee != nil {
						log.Printf("⚠️ Kicking old session for user: %s\n", msg.Username)
						go func(c *GameServer) {
							msg := pb.ToProtoServerMessage("error", nil, nil, nil, nil, nil, "Logged in from another location.", "", 0)
							c.sendMsg(msg)
							if c.close != nil {
								c.close()
							}
						}(killee)
					}

					gs.user = user
					gs.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, user, "", "", 0))
				}
				continue
			}
			if msg.Action == "register" {
				log.Printf("📝 Register attempt: %s\n", msg.Username)
				err := userManager.Register(msg.Username, msg.Password)
				if err != nil {
					log.Printf("❌ Register failed: %v\n", err)
					gs.sendMsg(pb.ToProtoServerMessage("auth_error", nil, nil, nil, nil, nil, err.Error(), "", 0))
				} else {
					log.Printf("✅ Register success: %s\n", msg.Username)
					gs.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, nil, "", "Account created! Please login.", 0))
				}
				continue
			}

			if msg.Action == "submit_feedback" {
				log.Printf("📩 Feedback received from %s: %s\n", msg.Username, msg.Feedback)
				_, err := game.DB.Exec("INSERT INTO feedback (username, message) VALUES (?, ?)", msg.Username, msg.Feedback)
				if err != nil {
					log.Printf("❌ Error saving feedback: %v\n", err)
				} else {
					// Trigger Feishu notification
					go notifyFeishu(msg.Username, msg.Feedback)
					gs.sendMsg(pb.ToProtoServerMessage("state", nil, nil, nil, nil, nil, "", "Thank you for your feedback!", 0))
				}
				continue
			}

			if msg.Action == "ping" {
				gs.sendMsg(pb.ToProtoServerMessage("pong", nil, nil, nil, nil, nil, "", "", 0))
				continue
			}

			if msg.Action == "logout" {
				log.Printf("👋 Logout received for user: %v\n", gs.user)
				if gs.close != nil {
					gs.close()
				}
				return // Terminate the read loop
			}

			if msg.Action == "submit_score" {
				// No-op: automatic on game over
			} else {
				// Only allow game actions if not in a state where we should be logged in?
				// For now, let's just let it run, but typically you'd want auth for leaderboard.
				gs.handleAction(msg.Action, msg.Mode)
			}
			// Trigger immediate state update for UI responsiveness
			if gs.match == nil && !gs.searching {
				state := gs.getGameState()
				gs.sendMsg(pb.ToProtoServerMessage("state", nil, &state, nil, nil, nil, "", "", 0))
			}
		}
	}()

	// Game loop
	for {
		select {
		case <-done:
			return
		case <-gs.ticker.C:
			// If in match, P1's runPVPGame goroutine handles the updates and broadcasts.
			if gs.match != nil {
				// CRITICAL: Even in match mode, we must sync the LOCAL boosting state of this connection
				// to the shared game object, otherwise individual player boosting won't work.
				gs.updateBoostingOnly()
				continue
			}

			changed := gs.update()

			if changed || gs.userUpdated || gs.lbUpdated {
				state := gs.getGameState()
				var leaderboard []game.LeaderboardEntry
				var winRates []game.WinRateEntry
				var user *game.User

				if gs.userUpdated {
					user = gs.user
					gs.userUpdated = false
				}
				if gs.lbUpdated {
					leaderboard = lbManager.GetEntries()
					winRates = lbManager.GetWinRateEntries()
					gs.lbUpdated = false
				}

				err := gs.sendMsg(pb.ToProtoServerMessage("state", nil, &state, leaderboard, winRates, user, "", "", 0))
				if err != nil {
					log.Println("Write error:", err)
					return
				}
				// Clear one-shot effects after sending
				gs.game.HitPoints = nil
				gs.game.ScoreEvents = nil
				gs.game.Message = ""
				gs.game.MessageType = ""
			}
		}
	}
}

func main() {
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, relying on system environment variables")
	}

	game.InitDB()

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/", fs)

	// Single Source of Truth: serve the proto file to the frontend from its server-side home
	http.HandleFunc("/proto/snake.proto", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pkg/proto/snake.proto")
	})

	// WebSocket endpoint
	http.HandleFunc("/ws", handleWebSocket)

	port := ":8080"
	log.Printf("🚀 Snake Game Web Server starting on http://localhost%s\n", port)
	http.HandleFunc("/admin/feedback", func(w http.ResponseWriter, r *http.Request) {
		// Simple basic security: checking for a secret query param
		secret := os.Getenv("ADMIN_SECRET")
		if secret == "" {
			secret = "admin123" // Fallback if not configured
		}
		if r.URL.Query().Get("key") != secret {
			http.Error(w, "Unauthorized. Please provide valid ?key=...", http.StatusUnauthorized)
			return
		}

		rows, err := game.DB.Query("SELECT username, message, created_at FROM feedback ORDER BY created_at DESC LIMIT 50")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html><head><title>Admin - Feedback</title><style>body{font-family:sans-serif;background:#1a1a2e;color:#fff;padding:20px;} table{width:100%%;border-collapse:collapse;} th,td{border:1px solid #444;padding:12px;text-align:left;} th{background:#333;}</style></head><body>")
		fmt.Fprintf(w, "<h1>📩 Recent User Feedback</h1><p>Welcome, Admin. Showing last 50 entries.</p><table><tr><th>Time</th><th>User</th><th>Message</th></tr>")
		for rows.Next() {
			var user, msg, timeStr string
			rows.Scan(&user, &msg, &timeStr)
			fmt.Fprintf(w, "<tr><td>%s</td><td><strong>%s</strong></td><td>%s</td></tr>", timeStr, user, msg)
		}
		fmt.Fprintf(w, "</table></body></html>")
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
