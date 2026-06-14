package main

// matchmaker.go implements PVP matchmaking and the shared-game loop.
//
// The shared game is guarded by Match.Mu. Per-player scalar fields (match,
// role, game, started, user) are guarded by each player's gs.mu. When both
// are needed the order is always Match.Mu -> gs.mu (see server.go).

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
	pb "github.com/trytobebee/snake_go/pkg/proto"
)

// Match is a shared PVP game between two connections.
type Match struct {
	Game    *game.Game
	P1      *GameServer
	P2      *GameServer
	Mu      sync.Mutex // guards Game and Closing
	Closing bool
}

// MatchMaker pairs waiting players.
type MatchMaker struct {
	mu      sync.Mutex
	waiting *GameServer
}

var pvpManager = &MatchMaker{}

// attachToMatch wires a connection into a match, taking the player's own lock
// so its game-loop goroutine observes a consistent transition.
func attachToMatch(gs *GameServer, m *Match, role string, sharedGame *game.Game) {
	gs.mu.Lock()
	gs.match = m
	gs.role = role
	gs.game = sharedGame
	gs.searching = false
	gs.mu.Unlock()
}

// FindMatch enqueues the player or pairs them with a waiting opponent.
func (mm *MatchMaker) FindMatch(gs *GameServer) {
	mm.mu.Lock()

	if mm.waiting == nil {
		mm.waiting = gs
		mm.mu.Unlock()

		// Read the username inside the same critical section (do NOT call
		// usernameOf here: gs.mu is non-reentrant).
		gs.mu.Lock()
		gs.searching = true
		username := "(anonymous)"
		if gs.user != nil {
			username = gs.user.Username
		}
		gs.mu.Unlock()

		log.Printf("[PVP] ⏳ Player %s entered matchmaking queue (waiting for opponent)\n", username)
		return
	}

	// Prevent matching with oneself (same user, different connection).
	if usernameOf(mm.waiting) == usernameOf(gs) {
		log.Printf("[PVP] ⚠️ Player %s tried to match with themselves (another session). Keeping original session in queue.\n", usernameOf(gs))
		mm.mu.Unlock()
		return
	}

	// Found a pair!
	p1 := mm.waiting
	p2 := gs
	mm.waiting = nil
	mm.mu.Unlock()

	log.Printf("[PVP] ⚔️ Match found: %s (P1) vs %s (P2). Initializing shared game state...\n", usernameOf(p1), usernameOf(p2))

	// Create shared game. Use Standard size for PVP to ensure mobile compatibility.
	sharedGame := game.NewGame(config.StandardWidth, config.StandardHeight)
	sharedGame.Mode = "pvp"
	sharedGame.IsPVP = true
	sharedGame.Paused = true // Start paused for countdown

	// Start players at different positions to avoid a head-on crash.
	sharedGame.Players = []*game.Player{
		{
			Snake:       []game.Point{{X: sharedGame.Width / 4, Y: sharedGame.Height / 3}},
			Direction:   game.Point{X: 1, Y: 0},
			LastMoveDir: game.Point{X: 1, Y: 0},
			Name:        usernameOf(p1),
			Brain:       &game.ManualController{},
			Controller:  "manual",
		},
		{
			Snake:       []game.Point{{X: (sharedGame.Width * 3) / 4, Y: (sharedGame.Height * 2) / 3}},
			Direction:   game.Point{X: -1, Y: 0},
			LastMoveDir: game.Point{X: -1, Y: 0},
			Name:        usernameOf(p2),
			Brain:       &game.ManualController{},
			Controller:  "manual",
		},
	}

	match := &Match{Game: sharedGame, P1: p1, P2: p2}

	attachToMatch(p1, match, "p1", sharedGame)
	attachToMatch(p2, match, "p2", sharedGame)

	log.Printf("[PVP] 🔗 Both players attached to Match. P1: %s, P2: %s. Sending initial MATCH FOUND msg.\n", usernameOf(p1), usernameOf(p2))

	// Initial broadcast with "MATCH FOUND".
	st := sharedGame.GetGameStateSnapshot(true, false, "mid")
	st.Message = "⚔️ MATCH FOUND!"
	st.MessageType = "important"

	p1.sendMsg(pb.ToProtoServerMessage("state", nil, &st, nil, nil, nil, "", "", 0))
	p2.sendMsg(pb.ToProtoServerMessage("state", nil, &st, nil, nil, nil, "", "", 0))

	log.Printf("[PVP] ⏱️ Starting 3-second countdown for %s vs %s\n", usernameOf(p1), usernameOf(p2))
	go mm.runPVPCountdown(match)
}

// CancelSearch removes a player from the matchmaking queue.
func (mm *MatchMaker) CancelSearch(gs *GameServer) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	if mm.waiting == gs {
		mm.waiting = nil
		gs.mu.Lock()
		gs.searching = false
		gs.mu.Unlock()
		log.Printf("👋 Player %s removed from matchmaking queue (disconnected/left)\n", usernameOf(gs))
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

		log.Printf("[PVP] 🔔 Countdown: %d... (Players: %s, %s)\n", i, usernameOf(m.P1), usernameOf(m.P2))

		stateP1 := state
		stateP1.Message = fmt.Sprintf("🟢 YOU ARE PLAYER 1 (GREEN)\nSTARTING IN %d...", i)
		m.P1.sendMsg(pb.ToProtoServerMessage("state", nil, &stateP1, nil, nil, nil, "", "", 0))

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
	m.Mu.Unlock()

	// Set both participants to started state so their update logic runs.
	setStarted(m.P1, true)
	setStarted(m.P2, true)

	log.Printf("[PVP] 🚀 Rocket Start! Game is now UNPAUSED for %s vs %s\n", usernameOf(m.P1), usernameOf(m.P2))
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

		// Advance game logic for both players.
		changed := m.P1.update() || m.P2.update()

		if changed {
			state := m.Game.GetGameStateSnapshot(true, false, m.P1.difficulty)

			m.P1.sendMsg(pb.ToProtoServerMessage("state", nil, &state, nil, nil, nil, "", "", 0))
			m.P2.sendMsg(pb.ToProtoServerMessage("state", nil, &state, nil, nil, nil, "", "", 0))

			// Reset one-shot effects only after broadcast.
			m.Game.ScoreEvents = nil
			m.Game.HitPoints = nil
			m.Game.Message = ""
			m.Game.MessageType = ""
		}

		if m.Game.GameOver {
			log.Printf("[PVP] 🏁 Match Over detected in loop (%s vs %s). Winner: %s\n", usernameOf(m.P1), usernameOf(m.P2), m.Game.Winner)
			m.Closing = true
			m.handleMatchOver()
			m.Mu.Unlock()
			return
		}
		m.Mu.Unlock()
	}
}

// handleMatchOver records stats and detaches both players. The caller MUST
// hold m.Mu.
func (m *Match) handleMatchOver() {
	gameObj := m.Game

	// Update stats for P1.
	if u := userOf(m.P1); u != nil && len(gameObj.Players) > 0 {
		won := gameObj.Winner == "player"
		if updated, _ := userManager.UpdateStats(u.Username, gameObj.Players[0].Score, won); updated != nil {
			m.P1.setUser(updated)
			m.P1.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, updated, "", "", 0))
		}
	}

	// Update stats for P2.
	if u := userOf(m.P2); u != nil && len(gameObj.Players) > 1 {
		won := gameObj.Winner == "ai"
		if updated, _ := userManager.UpdateStats(u.Username, gameObj.Players[1].Score, won); updated != nil {
			m.P2.setUser(updated)
			m.P2.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, updated, "", "", 0))
		}
	}

	if *detailedLogs {
		log.Printf("[PVP] 📝 Recording detailed game sessions for both players...\n")
		if u := userOf(m.P1); u != nil && len(gameObj.Players) > 0 {
			game.RecordGameSession(u.Username, m.P1.sessionStart, time.Now(), gameObj.Players[0].Score, pvpResult(gameObj.Winner, "player"), "pvp", m.P1.difficulty)
		}
		if u := userOf(m.P2); u != nil && len(gameObj.Players) > 1 {
			game.RecordGameSession(u.Username, m.P2.sessionStart, time.Now(), gameObj.Players[1].Score, pvpResult(gameObj.Winner, "ai"), "pvp", m.P2.difficulty)
		}
	}

	log.Printf("[PVP] 🔓 Detaching players from match and resetting to solo state (%s, %s)\n", usernameOf(m.P1), usernameOf(m.P2))
	detachToSolo(m.P1)
	detachToSolo(m.P2)
}

// pvpResult maps the game winner to "won"/"lost"/"draw" for the given side
// ("player" for P1, "ai" for P2).
func pvpResult(winner, winSide string) string {
	switch winner {
	case winSide:
		return "won"
	case "draw":
		return "draw"
	default:
		return "lost"
	}
}

// --- Per-player field helpers (take gs.mu) ----------------------------------

func usernameOf(gs *GameServer) string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.user != nil {
		return gs.user.Username
	}
	return "(anonymous)"
}

func userOf(gs *GameServer) *game.User {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.user
}

func setStarted(gs *GameServer, started bool) {
	gs.mu.Lock()
	gs.started = started
	gs.mu.Unlock()
}

func detachToSolo(gs *GameServer) {
	gs.mu.Lock()
	gs.match = nil
	gs.role = "solo"
	gs.started = false
	gs.mu.Unlock()
}
