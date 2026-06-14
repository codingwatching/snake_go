package main

// hub.go tracks all active connections for server-wide broadcasts.

import (
	"sync"

	pb "github.com/trytobebee/snake_go/pkg/proto"
)

const MaxPlayers = 500

var (
	clientsMu sync.RWMutex
	clients   = make(map[string]*GameServer)
)

// registerClient adds a connection and returns the player count after adding.
func registerClient(connID string, gs *GameServer) int {
	clientsMu.Lock()
	clients[connID] = gs
	count := len(clients)
	clientsMu.Unlock()
	return count
}

// unregisterClient removes a connection.
func unregisterClient(connID string) {
	clientsMu.Lock()
	delete(clients, connID)
	clientsMu.Unlock()
}

// playerCount returns the number of active connections.
func playerCount() int {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	return len(clients)
}

// findSessionByUser returns another connection logged in as username, if any.
func findSessionByUser(username, exceptConnID string) *GameServer {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	for id, c := range clients {
		if id == exceptConnID {
			continue
		}
		if u := c.currentUser(); u != nil && u.Username == username {
			return c
		}
	}
	return nil
}

// broadcastSessionCount pushes the current player count to all connections.
func broadcastSessionCount() {
	clientsMu.RLock()
	count := len(clients)
	targets := make([]*GameServer, 0, count)
	for _, gs := range clients {
		targets = append(targets, gs)
	}
	clientsMu.RUnlock()

	msg := pb.ToProtoServerMessage("update_counts", nil, nil, nil, nil, nil, "", "", count)
	for _, gs := range targets {
		go gs.sendMsg(msg)
	}
}
