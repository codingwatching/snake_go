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

type GameServer struct {
	game     *game.Game
	started  bool
	boosting bool
	ticker   *time.Ticker

	// Boost tracking
	tickCount           int
	lastBoostKeyTime    time.Time
	lastDirKeyTime      time.Time
	lastDirKeyDir       game.Point
	consecutiveKeyCount int
}

type ClientMessage struct {
	Action string `json:"action"`
}

type GameState struct {
	Snake       []game.Point `json:"snake"`
	Foods       []FoodInfo   `json:"foods"`
	Score       int          `json:"score"`
	FoodEaten   int          `json:"foodEaten"`
	EatingSpeed float64      `json:"eatingSpeed"`
	Started     bool         `json:"started"`
	GameOver    bool         `json:"gameOver"`
	Paused      bool         `json:"paused"`
	Boosting    bool         `json:"boosting"`
	Message     string       `json:"message,omitempty"`
	CrashPoint  *game.Point  `json:"crashPoint,omitempty"`
}

type FoodInfo struct {
	Pos              game.Point `json:"pos"`
	FoodType         int        `json:"foodType"`
	RemainingSeconds int        `json:"remainingSeconds"`
}

func NewGameServer() *GameServer {
	return &GameServer{
		game:   game.NewGame(),
		ticker: time.NewTicker(config.BaseTick),
	}
}

func (gs *GameServer) getGameState() GameState {
	foods := make([]FoodInfo, len(gs.game.Foods))
	for i, f := range gs.game.Foods {
		foods[i] = FoodInfo{
			Pos:              f.Pos,
			FoodType:         int(f.FoodType),
			RemainingSeconds: f.GetRemainingSeconds(gs.game.GetTotalPausedTime()),
		}
	}

	state := GameState{
		Snake:       gs.game.Snake,
		Foods:       foods,
		Score:       gs.game.Score,
		FoodEaten:   gs.game.FoodEaten,
		EatingSpeed: gs.game.GetEatingSpeed(),
		Started:     gs.started,
		GameOver:    gs.game.GameOver,
		Paused:      gs.game.Paused,
		Boosting:    gs.boosting,
		Message:     gs.game.GetMessage(),
	}

	if gs.game.GameOver {
		state.CrashPoint = &gs.game.CrashPoint
	}

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
				gs.game.StartTime = time.Now()
				gs.game.LastFoodSpawn = time.Now()
				if len(gs.game.Foods) > 0 {
					gs.game.Foods[0].SpawnTime = time.Now()
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
			gs.started = false
			gs.boosting = false
			gs.tickCount = 0
			gs.consecutiveKeyCount = 0
		}
	}

	if isDirection {
		if !gs.started {
			gs.started = true
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
	// Check boost timeout
	if gs.boosting && time.Since(gs.lastBoostKeyTime) > config.BoostTimeout {
		gs.boosting = false
	}

	gs.tickCount++

	if !gs.started {
		return
	}

	ticksNeeded := config.NormalTicksPerUpdate
	if gs.boosting {
		ticksNeeded = config.BoostTicksPerUpdate
	}

	if gs.tickCount >= ticksNeeded {
		gs.tickCount = 0
		if !gs.game.GameOver && !gs.game.Paused {
			gs.game.Update()
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

	log.Println("New WebSocket connection")

	gs := NewGameServer()
	defer gs.ticker.Stop()

	// Mutex to protect concurrent writes to the WebSocket connection
	var writeMu sync.Mutex
	safeWriteJSON := func(v interface{}) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	// Send initial state
	safeWriteJSON(gs.getGameState())

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
			safeWriteJSON(gs.getGameState())
		}
	}()

	// Game loop
	for range gs.ticker.C {
		gs.update()

		state := gs.getGameState()
		err := safeWriteJSON(state)
		if err != nil {
			log.Println("Write error:", err)
			return
		}
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
