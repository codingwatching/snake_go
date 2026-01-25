package game

import (
	"log"
)

const MaxLeaderboardEntries = 10

type LeaderboardManager struct{}

type WinRateEntry struct {
	Name       string  `json:"name"`
	WinRate    float64 `json:"win_rate"`
	TotalWins  int     `json:"total_wins"`
	TotalGames int     `json:"total_games"`
}

func NewLeaderboardManager() *LeaderboardManager {
	return &LeaderboardManager{}
}

func (lm *LeaderboardManager) AddEntry(name string, score int, difficulty string, mode string) bool {
	log.Printf("📊 Attempting to add leaderboard entry: Player=%s, Score=%d, Mode=%s\n", name, score, mode)

	// Check if this score makes it to top 10
	entries := lm.GetEntries()
	if len(entries) >= MaxLeaderboardEntries && score <= entries[len(entries)-1].Score {
		log.Printf("⏭️ Score %d too low for top %d (lowest is %d). Skipping.\n", score, MaxLeaderboardEntries, entries[len(entries)-1].Score)
		return false
	}

	_, err := DB.Exec(
		"INSERT INTO leaderboard (name, score, difficulty, mode) VALUES (?, ?, ?, ?)",
		name, score, difficulty, mode,
	)
	if err != nil {
		log.Printf("❌ Error adding leaderboard entry to DB: %v\n", err)
		return false
	}

	log.Printf("✅ Success! Score %d saved for %s.\n", score, name)

	// Keep only top 10 in DB
	res, _ := DB.Exec(`
		DELETE FROM leaderboard 
		WHERE id NOT IN (
			SELECT id FROM leaderboard 
			ORDER BY score DESC 
			LIMIT ?
		)`, MaxLeaderboardEntries)

	if affected, err := res.RowsAffected(); err == nil && affected > 0 {
		log.Printf("🧹 Trimmed %d older entries from leaderboard.\n", affected)
	}

	return true
}

func (lm *LeaderboardManager) GetEntries() []LeaderboardEntry {
	rows, err := DB.Query(
		"SELECT name, score, date, difficulty, mode FROM leaderboard ORDER BY score DESC LIMIT ?",
		MaxLeaderboardEntries,
	)
	if err != nil {
		log.Printf("❌ Error querying leaderboard: %v\n", err)
		return []LeaderboardEntry{}
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.Name, &e.Score, &e.Date, &e.Difficulty, &e.Mode); err != nil {
			log.Printf("❌ Error scanning leaderboard row: %v\n", err)
			continue
		}
		entries = append(entries, e)
	}
	log.Printf("🔍 Fetched %d score leaderboard entries.\n", len(entries))
	return entries
}

func (lm *LeaderboardManager) GetWinRateEntries() []WinRateEntry {
	// Filter for users who have played at least 3 games to avoid 100% win rate from 1 game
	rows, err := DB.Query(`
		SELECT username, total_wins, total_games, 
		       (CAST(total_wins AS FLOAT) / total_games * 100) as win_rate
		FROM users 
		WHERE total_games >= 1
		ORDER BY win_rate DESC, total_games DESC
		LIMIT ?`,
		MaxLeaderboardEntries,
	)
	if err != nil {
		log.Printf("❌ Error querying win rate leaderboard: %v\n", err)
		return []WinRateEntry{}
	}
	defer rows.Close()

	var entries []WinRateEntry
	for rows.Next() {
		var e WinRateEntry
		if err := rows.Scan(&e.Name, &e.TotalWins, &e.TotalGames, &e.WinRate); err != nil {
			log.Printf("❌ Error scanning win rate row: %v\n", err)
			continue
		}
		entries = append(entries, e)
	}
	log.Printf("🔍 Fetched %d win rate entries.\n", len(entries))
	return entries
}
