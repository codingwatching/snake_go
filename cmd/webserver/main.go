package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
)

var (
	detailedLogs = flag.Bool("detailed-logs", false, "Enable detailed session logging to database")
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Global map to track active IP connections
var (
	activeIPs     sync.Map
	totalSessions int32
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
	writeMu       sync.Mutex
	safeWriteJSON func(v interface{}) error
}

// ... (ServerMessage, ClientMessage structs unchanged)

// ... (ServerMessage, ClientMessage structs unchanged)
type ServerMessage struct {
	Type         string                  `json:"type"`
	Config       *game.GameConfig        `json:"config,omitempty"`
	State        *game.GameState         `json:"state,omitempty"`
	Leaderboard  []game.LeaderboardEntry `json:"leaderboard,omitempty"`
	WinRates     []game.WinRateEntry     `json:"win_rates,omitempty"`
	User         *game.User              `json:"user,omitempty"`
	Error        string                  `json:"error,omitempty"`
	Success      string                  `json:"success,omitempty"`
	SessionCount int                     `json:"sessionCount,omitempty"`
}

type ClientMessage struct {
	Action   string `json:"action"`
	Name     string `json:"name,omitempty"`     // For score submission
	Username string `json:"username,omitempty"` // For auth
	Password string `json:"password,omitempty"` // For auth
	Mode     string `json:"mode,omitempty"`     // For "auto" action (neural/heuristic)
}

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
	msg := ServerMessage{Type: "state"}
	st := sharedGame.GetGameStateSnapshot(true, false, "mid")
	st.Message = "⚔️ MATCH FOUND!"
	st.MessageType = "important"
	msg.State = &st
	p1.safeWriteJSON(msg)
	p2.safeWriteJSON(msg)

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
		m.Game.Message = fmt.Sprintf("🔥 STARTING IN %d...", i)
		m.Game.MessageType = "important"
		state := m.Game.GetGameStateSnapshot(true, false, m.P1.difficulty)
		m.Mu.Unlock()

		log.Printf("[PVP] 🔔 Countdown: %d... (Players: %s, %s)\n", i, m.P1.user.Username, m.P2.user.Username)
		msg := ServerMessage{Type: "state", State: &state}
		m.P1.safeWriteJSON(msg)
		m.P2.safeWriteJSON(msg)
		time.Sleep(1 * time.Second)
	}

	m.Mu.Lock()
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

		// Update game logic (using P1 as the "driver" for settings context)
		// We'll simulate a GameServer update call but on the shared game
		// Create a temporary "controller" GameServer for the update logic
		p1 := m.P1
		p1.update() // This updates the shared m.Game

		state := m.Game.GetGameStateSnapshot(true, false, p1.difficulty)

		msg := ServerMessage{
			Type:  "state",
			State: &state,
		}

		// Broadcast to both
		m.P1.safeWriteJSON(msg)
		m.P2.safeWriteJSON(msg)

		if m.Game.GameOver {
			log.Printf("[PVP] 🏁 Match Over detected in loop (%s vs %s). Winner: %s\n", m.P1.user.Username, m.P2.user.Username, m.Game.Winner)
			m.Closing = true

			// Handle stats for both players
			m.handleMatchOver()

			m.Mu.Unlock()
			return
		}

		// Reset one-shot effects
		m.Game.ScoreEvents = nil
		m.Game.HitPoints = nil
		m.Game.Message = ""
		m.Game.MessageType = ""
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
			m.P1.safeWriteJSON(ServerMessage{Type: "auth_success", User: updated})
		}
	}

	// Update Stats for P2
	if m.P2.user != nil && len(gameObj.Players) > 1 {
		won := gameObj.Winner == "ai"
		updated, _ := userManager.UpdateStats(m.P2.user.Username, gameObj.Players[1].Score, won)
		if updated != nil {
			m.P2.user = updated
			m.P2.safeWriteJSON(ServerMessage{Type: "auth_success", User: updated})
		}
	}

	// Sessions
	if *detailedLogs {
		log.Printf("[PVP] 📝 Recording detailed game sessions for both players...\n")
		if m.P1.user != nil && len(gameObj.Players) > 0 {
			game.RecordGameSession(m.P1.user.Username, m.P1.sessionStart, time.Now(), gameObj.Players[0].Score, gameObj.Winner, "pvp", m.P1.difficulty)
		}
		if m.P2.user != nil && len(gameObj.Players) > 1 {
			p2Res := "lost"
			if gameObj.Winner == "ai" {
				p2Res = "won"
			} else if gameObj.Winner == "draw" {
				p2Res = "draw"
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

func (gs *GameServer) update() {
	// Sync manual boosting state to game if not in AutoPlay
	gs.updateBoostingOnly()

	gs.tickCount++

	if !gs.started {
		return
	}

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
	// ...

	if gs.tickCount >= ticksNeeded {
		gs.tickCount = 0
		if !gs.game.GameOver && !gs.game.Paused {
			gs.game.Update()

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

	// ... (AI and Fireball update unchanged)

	if gs.started && !gs.game.GameOver && len(gs.game.Players) > 1 && gs.game.Players[1].Score > 0 {
		// Temporary debug log
	}

	// Move AI snake independently (if any)
	if !gs.game.IsPVP && len(gs.game.Players) > 1 {
		gs.aiTickCount++
		aiTicksNeeded := config.MidTicks
		if gs.game.Players[1].Boosting {
			aiTicksNeeded = config.MidBoostTicks
		}
		if gs.aiTickCount >= aiTicksNeeded {
			gs.aiTickCount = 0
			if !gs.game.GameOver && !gs.game.Paused {
				// Competitor AI is already handled in game.UpdatePlayer if IsAI is true
				// But we need to ensure the tick logic matches.
				// For simplicity, we assume UpdatePlayer handles it.
			}
		}
	}

	// Update fireballs independently at FireballSpeed
	gs.fireballTickCount++
	fbTicks := int(config.FireballSpeed / config.BaseTick)
	if gs.fireballTickCount >= fbTicks {
		gs.fireballTickCount = 0
		if !gs.game.GameOver && !gs.game.Paused {
			gs.game.UpdateFireballs()
		}
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
				if gs.role == "p1" {
					won = gs.game.Winner == "player"
				} else if gs.role == "p2" {
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
		}
	}
}

var (
	clientsMu sync.RWMutex
	clients   = make(map[string]*GameServer)
)

func broadcastSessionCount() {
	count := int(atomic.LoadInt32(&totalSessions))
	msg := ServerMessage{
		Type:         "update_counts",
		SessionCount: count,
	}
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	for _, gs := range clients {
		gs.safeWriteJSON(msg)
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
	atomic.AddInt32(&totalSessions, 1)
	broadcastSessionCount()

	// Generate a short unique ID for this connection
	connID := fmt.Sprintf("%d", time.Now().UnixNano()%10000)

	gs := NewGameServer(connID)

	// Add to global client tracking for broadcasts
	clientsMu.Lock()
	clients[connID] = gs
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, connID)
		clientsMu.Unlock()
		atomic.AddInt32(&totalSessions, -1)
		broadcastSessionCount()

		// Fix: Remove from matchmaking queue if waiting
		pvpManager.CancelSearch(gs)

		// Fix: Handle PVP match termination if in game
		if gs.match != nil {
			gs.match.Mu.Lock()
			if !gs.match.Closing {
				gs.match.Closing = true
				log.Printf("[PVP] 📡 Match terminated due to %s disconnecting\n", gs.user.Username)
			}
			gs.match.Mu.Unlock()
		}

		gs.ticker.Stop()
		gs.stopRecording()
	}()

	// Mutex to protect concurrent writes to the WebSocket connection
	gs.safeWriteJSON = func(v interface{}) error {
		gs.writeMu.Lock()
		defer gs.writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	// ... (Rest of handleWebSocket unchanged: send config, state, loops)

	// Send initial config
	gameConfig := gs.game.GetGameConfig()
	gs.safeWriteJSON(ServerMessage{
		Type:   "config",
		Config: &gameConfig,
	})

	// Send leaderboards
	gs.safeWriteJSON(ServerMessage{
		Type:        "leaderboard",
		Leaderboard: lbManager.GetEntries(),
		WinRates:    lbManager.GetWinRateEntries(),
	})

	initialState := gs.getGameState()
	gs.safeWriteJSON(ServerMessage{
		Type:  "state",
		State: &initialState,
	})

	// Input handling goroutine
	go func() {
		for {
			var msg ClientMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			if msg.Action == "login" {
				log.Printf("🔑 Login attempt: %s\n", msg.Username)
				user, err := userManager.Login(msg.Username, msg.Password)
				if err != nil {
					log.Printf("❌ Login failed: %v\n", err)
					gs.safeWriteJSON(ServerMessage{Type: "auth_error", Error: err.Error()})
				} else {
					log.Printf("✅ Login success: %s\n", msg.Username)
					gs.user = user
					gs.safeWriteJSON(ServerMessage{Type: "auth_success", User: user})
				}
				continue
			}
			if msg.Action == "register" {
				log.Printf("📝 Register attempt: %s\n", msg.Username)
				err := userManager.Register(msg.Username, msg.Password)
				if err != nil {
					log.Printf("❌ Register failed: %v\n", err)
					gs.safeWriteJSON(ServerMessage{Type: "auth_error", Error: err.Error()})
				} else {
					log.Printf("✅ Register success: %s\n", msg.Username)
					gs.safeWriteJSON(ServerMessage{Type: "auth_success", Success: "Account created! Please login."})
				}
				continue
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
				gs.safeWriteJSON(ServerMessage{
					Type:  "state",
					State: &state,
				})
			}
		}
	}()

	// Game loop
	for range gs.ticker.C {
		// If in match, P1's runPVPGame goroutine handles the updates and broadcasts.
		if gs.match != nil {
			// CRITICAL: Even in match mode, we must sync the LOCAL boosting state of this connection
			// to the shared game object, otherwise individual player boosting won't work.
			gs.updateBoostingOnly()
			continue
		}

		gs.update()

		state := gs.getGameState()
		msg := ServerMessage{
			Type:  "state",
			State: &state,
		}
		if gs.userUpdated {
			msg.User = gs.user
			gs.userUpdated = false
		}
		if gs.lbUpdated {
			msg.Leaderboard = lbManager.GetEntries()
			msg.WinRates = lbManager.GetWinRateEntries()
			gs.lbUpdated = false
		}
		err := gs.safeWriteJSON(msg)
		if err != nil {
			log.Println("Write error:", err)
			return
		}
		// Clear one-shot effects after sending
		gs.game.HitPoints = nil
	}
}

func main() {
	flag.Parse()
	game.InitDB()

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/", fs)

	// WebSocket endpoint
	http.HandleFunc("/ws", handleWebSocket)

	port := ":8080"
	log.Printf("🚀 Snake Game Web Server starting on http://localhost%s\n", port)
	log.Println("📱 Open your browser and visit: http://localhost:8080")

	log.Fatal(http.ListenAndServe(port, nil))
}
