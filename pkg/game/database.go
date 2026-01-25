package game

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "game.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	createTables()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			best_score INTEGER DEFAULT 0,
			total_games INTEGER DEFAULT 0,
			total_wins INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS leaderboard (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			score INTEGER,
			difficulty TEXT,
			mode TEXT,
			date DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS game_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			start_time DATETIME,
			end_time DATETIME,
			score INTEGER,
			winner TEXT,
			mode TEXT,
			difficulty TEXT
		)`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Fatalf("Failed to create table (%s): %v", query, err)
		}
	}
}
