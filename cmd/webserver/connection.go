package main

// connection.go owns a single WebSocket connection: the handshake, the input
// goroutine that reads client messages, and the game-loop goroutine that ticks
// the game and broadcasts state.

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
	pb "github.com/trytobebee/snake_go/pkg/proto"
	"google.golang.org/protobuf/proto"
)

var mobileKeywords = []string{"Mobile", "Android", "iPhone", "iPad", "Windows Phone", "Mobi"}

func isMobileUserAgent(userAgent string) bool {
	for _, kw := range mobileKeywords {
		if bytes.Contains([]byte(userAgent), []byte(kw)) {
			return true
		}
	}
	return false
}

func newConnID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x-%d", b, time.Now().UnixNano())
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("New WebSocket connection from:", r.RemoteAddr)
	connID := newConnID()

	// Desktop gets a larger board; mobile uses the standard size.
	width, height := config.StandardWidth, config.StandardHeight
	if isMobileUserAgent(r.Header.Get("User-Agent")) {
		log.Printf("[Server] Mobile detected. Using standard game space: %dx%d\n", width, height)
	} else {
		width, height = config.LargeWidth, config.LargeHeight
		log.Printf("[Server] Desktop detected. Using large game space: %dx%d\n", width, height)
	}

	gs := NewGameServer(connID, width, height)
	gs.sendMsg = func(v *pb.ServerMessage) error {
		gs.writeMu.Lock()
		defer gs.writeMu.Unlock()
		data, err := proto.Marshal(v)
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.BinaryMessage, data)
	}
	gs.close = func() { conn.Close() }

	// Enforce the player limit.
	if playerCount() >= MaxPlayers {
		log.Printf("🚫 Connection rejected: server full (%d/%d)\n", MaxPlayers, MaxPlayers)
		msg := pb.ToProtoServerMessage("error", nil, nil, nil, nil, nil, "Server is full (500/500). Please wait for a player to leave and try refreshing.", "", 0)
		if data, err := proto.Marshal(msg); err == nil {
			conn.WriteMessage(websocket.BinaryMessage, data)
		}
		return
	}

	registerClient(connID, gs)
	broadcastSessionCount()

	defer cleanupConnection(gs, connID)

	sendInitialState(gs)

	done := make(chan struct{})
	go gs.readLoop(conn, done)
	gs.gameLoop(done)
}

// cleanupConnection runs when the connection ends: it deregisters the client,
// cancels matchmaking, tears down any active match, and stops the ticker.
func cleanupConnection(gs *GameServer, connID string) {
	unregisterClient(connID)
	broadcastSessionCount()

	pvpManager.CancelSearch(gs)

	if m := gs.currentMatch(); m != nil {
		m.Mu.Lock()
		if !m.Closing {
			m.Closing = true
			log.Printf("[PVP] 📡 Match terminated due to %s disconnecting\n", usernameOf(gs))
			m.handleMatchOver() // resets the opponent to solo mode
		}
		m.Mu.Unlock()
	}

	gs.ticker.Stop()

	// The read loop may still be draining (e.g. if the game loop exited on a
	// write error), so guard the recorder with gs.mu.
	gs.mu.Lock()
	gs.stopRecording()
	gs.mu.Unlock()
}

func sendInitialState(gs *GameServer) {
	gameConfig := gs.game.GetGameConfig()
	gs.sendMsg(pb.ToProtoServerMessage("config", &gameConfig, nil, nil, nil, nil, "", "", 0))
	gs.sendMsg(pb.ToProtoServerMessage("leaderboard", nil, nil, lbManager.GetEntries(), lbManager.GetWinRateEntries(), nil, "", "", 0))

	gs.mu.Lock()
	initialState := gs.getGameState()
	gs.mu.Unlock()
	gs.sendMsg(pb.ToProtoServerMessage("state", nil, &initialState, nil, nil, nil, "", "", 0))
}

// readLoop reads and dispatches client messages until the connection closes.
func (gs *GameServer) readLoop(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		var msg pb.ClientMessage
		if err := proto.Unmarshal(data, &msg); err != nil {
			log.Println("Protobuf unmarshal error:", err)
			continue
		}

		// Control messages (auth, feedback, ping, logout) are handled inline;
		// everything else is a game action.
		if handled := gs.handleControlMessage(&msg); handled {
			if msg.Action == "logout" {
				return // terminate the read loop
			}
			continue
		}

		gs.handleAction(msg.Action, msg.Mode)

		// Immediate state echo for UI responsiveness (solo only; PVP is driven
		// by the shared game loop).
		gs.mu.Lock()
		echo := gs.match == nil && !gs.searching
		var st game.GameState
		if echo {
			st = gs.getGameState()
		}
		gs.mu.Unlock()
		if echo {
			gs.sendMsg(pb.ToProtoServerMessage("state", nil, &st, nil, nil, nil, "", "", 0))
		}
	}
}

// handleControlMessage handles non-gameplay messages. It returns true if the
// message was a control message (and thus fully handled here).
func (gs *GameServer) handleControlMessage(msg *pb.ClientMessage) bool {
	switch msg.Action {
	case "login":
		log.Printf("🔑 Login attempt: %s\n", msg.Username)
		user, err := userManager.Login(msg.Username, msg.Password)
		if err != nil {
			log.Printf("❌ Login failed: %v\n", err)
			gs.sendMsg(pb.ToProtoServerMessage("auth_error", nil, nil, nil, nil, nil, err.Error(), "", 0))
			return true
		}
		log.Printf("✅ Login success: %s\n", msg.Username)

		// Kick any older session for this user.
		if killee := findSessionByUser(msg.Username, gs.connID); killee != nil {
			log.Printf("⚠️ Kicking old session for user: %s\n", msg.Username)
			go func(c *GameServer) {
				c.sendMsg(pb.ToProtoServerMessage("error", nil, nil, nil, nil, nil, "Logged in from another location.", "", 0))
				if c.close != nil {
					c.close()
				}
			}(killee)
		}

		gs.setUser(user)
		gs.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, user, "", "", 0))
		return true

	case "register":
		log.Printf("📝 Register attempt: %s\n", msg.Username)
		if err := userManager.Register(msg.Username, msg.Password); err != nil {
			log.Printf("❌ Register failed: %v\n", err)
			gs.sendMsg(pb.ToProtoServerMessage("auth_error", nil, nil, nil, nil, nil, err.Error(), "", 0))
		} else {
			log.Printf("✅ Register success: %s\n", msg.Username)
			gs.sendMsg(pb.ToProtoServerMessage("auth_success", nil, nil, nil, nil, nil, "", "Account created! Please login.", 0))
		}
		return true

	case "submit_feedback":
		log.Printf("📩 Feedback received from %s: %s\n", msg.Username, msg.Feedback)
		if _, err := game.DB.Exec("INSERT INTO feedback (username, message) VALUES (?, ?)", msg.Username, msg.Feedback); err != nil {
			log.Printf("❌ Error saving feedback: %v\n", err)
		} else {
			go notifyFeishu(msg.Username, msg.Feedback)
			gs.sendMsg(pb.ToProtoServerMessage("state", nil, nil, nil, nil, nil, "", "Thank you for your feedback!", 0))
		}
		return true

	case "ping":
		gs.sendMsg(pb.ToProtoServerMessage("pong", nil, nil, nil, nil, nil, "", "", 0))
		return true

	case "logout":
		log.Printf("👋 Logout received for user: %v\n", gs.currentUser())
		if gs.close != nil {
			gs.close()
		}
		return true
	}
	return false
}

// gameLoop ticks the game until the connection closes. For solo games it
// advances state under gs.mu; for PVP it only syncs this player's boost state
// (the shared loop in runPVPGame drives the match).
func (gs *GameServer) gameLoop(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-gs.ticker.C:
			// Decide the mode under gs.mu so a PVP match cannot be attached
			// (which also takes gs.mu) between the check and the solo update.
			gs.mu.Lock()
			if gs.match != nil {
				gs.mu.Unlock()
				// In PVP the shared loop drives the game; here we only sync
				// this player's boost state under the match lock.
				if m := gs.currentMatch(); m != nil {
					m.Mu.Lock()
					gs.updateBoostingOnly()
					m.Mu.Unlock()
				}
				continue
			}

			changed := gs.update()

			var (
				send bool
				st   game.GameState
				lb   []game.LeaderboardEntry
				wr   []game.WinRateEntry
				usr  *game.User
			)
			if changed || gs.userUpdated || gs.lbUpdated {
				st = gs.getGameState()
				if gs.userUpdated {
					usr = gs.user
					gs.userUpdated = false
				}
				if gs.lbUpdated {
					lb = lbManager.GetEntries()
					wr = lbManager.GetWinRateEntries()
					gs.lbUpdated = false
				}
				// Clear one-shot effects (already copied into st).
				gs.game.HitPoints = nil
				gs.game.ScoreEvents = nil
				gs.game.Message = ""
				gs.game.MessageType = ""
				send = true
			}
			gs.mu.Unlock()

			if send {
				if err := gs.sendMsg(pb.ToProtoServerMessage("state", nil, &st, lb, wr, usr, "", "", 0)); err != nil {
					log.Println("Write error:", err)
					return
				}
			}
		}
	}
}
