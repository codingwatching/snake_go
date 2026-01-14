package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Global map to track active IP connections
var activeIPs sync.Map

type GameServer struct {
	game       *game.Game
	started    bool
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
}

type ServerMessage struct {
	Type   string           `json:"type"`
	Config *game.GameConfig `json:"config,omitempty"`
	State  *game.GameState  `json:"state,omitempty"`
}

type ClientMessage struct {
	Action string `json:"action"`
}

// FoodInfo moved to pkg/game
// GameState moved to pkg/game

func NewGameServer() *GameServer {
	gs := &GameServer{
		game:        game.NewGame(),
		ticker:      time.NewTicker(config.BaseTick),
		difficulty:  "mid",
		currentMode: "battle",
	}
	gs.game.TimerStarted = false
	return gs
}

func (gs *GameServer) getGameState() game.GameState {
	state := gs.game.GetGameStateSnapshot(gs.started, gs.boosting, gs.difficulty)

	// Important: Clear events after they are captured for the current state update
	// to prevent the client from creating duplicate floating bubbles.
	gs.game.ScoreEvents = nil

	return state
}

func (gs *GameServer) handleAction(action string) {
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
				gs.started = true
				gs.tickCount = 0
				gs.game.TimerStarted = true
				gs.game.StartTime = time.Now()
				gs.game.LastFoodSpawn = time.Now()
				if len(gs.game.Foods) > 0 {
					gs.game.Foods[0].SpawnTime = time.Now()
					gs.game.Foods[0].PausedTimeAtSpawn = gs.game.GetTotalPausedTime()
				}
			} else {
				gs.game.TogglePause()
			}
		}
	case "start":
		gs.started = true
	case "restart":
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
		gs.game.AISnake = nil
	case "mode_battle":
		gs.currentMode = "battle"
		gs.game.Mode = "battle"
		if len(gs.game.AISnake) == 0 {
			gs.game.AISnake = []game.Point{{X: config.Width - 2, Y: config.Height - 2}}
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
			gs.game.ToggleAutoPlay()
		}
	case "fire":
		if !gs.game.GameOver && !gs.game.Paused {
			gs.game.Fire()
		}
	case "toggleBerserker":
		if !gs.game.GameOver {
			gs.game.ToggleBerserkerMode()
		}
	}

	if isDirection {
		if !gs.started {
			gs.started = true
			gs.game.TimerStarted = true
			gs.tickCount = 0
			gs.game.StartTime = time.Now()
			gs.game.LastFoodSpawn = time.Now()
			if len(gs.game.Foods) > 0 {
				gs.game.Foods[0].SpawnTime = time.Now()
			}
		}
		dirChanged := gs.game.SetDirection(inputDir)

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

func (gs *GameServer) checkBoostKey(inputDir game.Point) {
	now := time.Now()

	if inputDir == gs.lastDirKeyDir && time.Since(gs.lastDirKeyTime) < config.KeyRepeatWindow {
		gs.consecutiveKeyCount++
	} else {
		gs.consecutiveKeyCount = 1
	}

	gs.lastDirKeyDir = inputDir
	gs.lastDirKeyTime = now

	if gs.consecutiveKeyCount >= config.BoostThreshold && inputDir == gs.game.Direction {
		gs.boosting = true
		gs.lastBoostKeyTime = now
	}
}

func (gs *GameServer) update() {
	// Sync manual boosting state to game if not in AutoPlay
	if !gs.game.AutoPlay {
		// Check manual boost timeout
		if gs.boosting && time.Since(gs.lastBoostKeyTime) > config.BoostTimeout {
			gs.boosting = false
		}
		gs.game.Boosting = gs.boosting
	}

	gs.tickCount++

	if !gs.started {
		return
	}

	ticksNeeded := 13 // Default Medium
	boostTicks := 4

	switch gs.difficulty {
	case "low":
		ticksNeeded = 18 // 288ms
		boostTicks = 6   // 96ms
	case "mid":
		ticksNeeded = 13 // 208ms (approx 216ms)
		boostTicks = 4   // 64ms
	case "high":
		ticksNeeded = 9 // 144ms
		boostTicks = 3  // 48ms
	}

	if gs.game.Boosting {
		ticksNeeded = boostTicks
	}

	if gs.tickCount >= ticksNeeded {
		gs.tickCount = 0
		if !gs.game.GameOver && !gs.game.Paused {
			gs.game.Update()
		}
	}

	if gs.started && !gs.game.GameOver && gs.game.AIScore > 0 {
		// Temporary debug log
		// log.Printf("Debug: User Score: %d, AI Score: %d\n", gs.game.Score, gs.game.AIScore)
	}

	// Move AI snake independently
	gs.aiTickCount++
	aiTicksNeeded := 13 // Mid speed for AI
	if gs.game.AIBoosting {
		aiTicksNeeded = 4
	}
	if gs.aiTickCount >= aiTicksNeeded {
		gs.aiTickCount = 0
		if !gs.game.GameOver && !gs.game.Paused {
			gs.game.UpdateAISnake()
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
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("New WebSocket connection from:", r.RemoteAddr)

	// Get base IP address (remove port)
	ip := r.RemoteAddr
	for i := len(r.RemoteAddr) - 1; i >= 0; i-- {
		if r.RemoteAddr[i] == ':' {
			ip = r.RemoteAddr[:i]
			break
		}
	}

	// Double check if this IP is already connected
	if _, loaded := activeIPs.LoadOrStore(ip, true); loaded {
		log.Printf("Connection rejected: IP %s is already connected\n", ip)
		// Optionally send a reason before closing, but simple close is safer
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Already connected"))
		return
	}

	// Defer removal of IP from active list when connection closes
	defer activeIPs.Delete(ip)

	gs := NewGameServer()
	defer gs.ticker.Stop()

	// Mutex to protect concurrent writes to the WebSocket connection
	var writeMu sync.Mutex
	safeWriteJSON := func(v interface{}) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	// Send initial config
	gameConfig := gs.game.GetGameConfig()
	safeWriteJSON(ServerMessage{
		Type:   "config",
		Config: &gameConfig,
	})

	// Send initial state
	initialState := gs.getGameState()
	safeWriteJSON(ServerMessage{
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
			gs.handleAction(msg.Action)
			// Trigger immediate state update for UI responsiveness
			state := gs.getGameState()
			safeWriteJSON(ServerMessage{
				Type:  "state",
				State: &state,
			})
		}
	}()

	// Game loop
	for range gs.ticker.C {
		gs.update()

		state := gs.getGameState()
		err := safeWriteJSON(ServerMessage{
			Type:  "state",
			State: &state,
		})
		if err != nil {
			log.Println("Write error:", err)
			return
		}
		// Clear one-shot effects after sending
		gs.game.HitPoints = nil
	}
}

func main() {
	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/", fs)

	// WebSocket endpoint
	http.HandleFunc("/ws", handleWebSocket)

	port := ":8080"
	fmt.Printf("🚀 Snake Game Web Server starting on http://localhost%s\n", port)
	fmt.Println("📱 Open your browser and visit: http://localhost:8080")

	log.Fatal(http.ListenAndServe(port, nil))
}
