package game

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	BestScore    int       `json:"best_score"`
	TotalGames   int       `json:"total_games"`
	TotalWins    int       `json:"total_wins"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserManager struct {
	// No longer needs in-memory map or sync.RWMutex as DB handles it
}

func NewUserManager() *UserManager {
	return &UserManager{}
}

func (um *UserManager) Register(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = DB.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, string(hash),
	)
	if err != nil {
		return errors.New("user already exists or database error")
	}

	return nil
}

func (um *UserManager) Login(username, password string) (*User, error) {
	user := &User{}
	var hash string

	err := DB.QueryRow(
		"SELECT username, password_hash, best_score, total_games, total_wins, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.Username, &hash, &user.BestScore, &user.TotalGames, &user.TotalWins, &user.CreatedAt)

	if err != nil {
		return nil, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return user, nil
}

func (um *UserManager) UpdateStats(username string, score int, won bool) (*User, error) {
	winVal := 0
	if won {
		winVal = 1
	}

	_, err := DB.Exec(`
		UPDATE users 
		SET best_score = CASE WHEN ? > best_score THEN ? ELSE best_score END,
		    total_games = total_games + 1,
		    total_wins = total_wins + ?
		WHERE username = ?`,
		score, score, winVal, username,
	)
	if err != nil {
		return nil, err
	}

	// Fetch updated user
	user := &User{}
	err = DB.QueryRow(
		"SELECT username, best_score, total_games, total_wins, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.Username, &user.BestScore, &user.TotalGames, &user.TotalWins, &user.CreatedAt)

	return user, err
}
