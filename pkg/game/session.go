package game

import (
	"log"
	"time"
)

func RecordGameSession(username string, startTime, endTime time.Time, score int, winner, mode, difficulty string) {
	_, err := DB.Exec(`
		INSERT INTO game_sessions (username, start_time, end_time, score, winner, mode, difficulty)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		username, startTime, endTime, score, winner, mode, difficulty,
	)
	if err != nil {
		log.Printf("❌ Error recording game session: %v\n", err)
	} else {
		log.Printf("📝 Detailed session recorded for %s: Result=%s, Score=%d\n", username, winner, score)
	}
}
