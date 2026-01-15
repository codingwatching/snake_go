package game

import "time"

// Point represents a coordinate on the game board
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// FoodType represents different types of food
type FoodType int

const (
	FoodPurple FoodType = iota // Purple, 10 points, 20s
	FoodBlue                   // Blue, 20 points, 18s
	FoodOrange                 // Orange, 30 points, 15s
	FoodRed                    // Red, 40 points, 10s
)

// Food represents a food item on the board
type Food struct {
	Pos               Point
	FoodType          FoodType
	SpawnTime         time.Time
	PausedTimeAtSpawn time.Duration // Total game pause time when this food was spawned
}

// Obstacle represents a temporary wall/stone unit on the board
type Obstacle struct {
	Points            []Point       `json:"points"`
	SpawnTime         time.Time     `json:"spawnTime"`
	Duration          float64       `json:"duration"` // in seconds
	PausedTimeAtSpawn time.Duration `json:"-"`
}

// Fireball represents a projectile shot by the snake
type Fireball struct {
	Pos       Point     `json:"pos"`
	Dir       Point     `json:"dir"`
	SpawnTime time.Time `json:"-"`
	Owner     string    `json:"owner"` // "player" or "ai"
}

// ScoreEvent represents a point-earning event for visual feedback
type ScoreEvent struct {
	Pos    Point  `json:"pos"`
	Amount int    `json:"amount"`
	Label  string `json:"label"`
}

// Game represents the main game state
type Game struct {
	Snake              []Point
	Foods              []Food // Multiple food items
	Direction          Point
	LastMoveDir        Point // Direction of the last performed move
	Score              int
	ScoreEvents        []ScoreEvent `json:"scoreEvents"` // Recent scoring events
	GameOver           bool
	PlayerStunnedUntil time.Time     `json:"-"`
	PlayerStunned      bool          `json:"playerStunned"`
	Paused             bool          // Pause state
	AutoPlay           bool          // Auto-play / Demo mode active
	Boosting           bool          // Active boosting state
	CrashPoint         Point         // Collision position
	StartTime          time.Time     // Game start time
	EndTime            time.Time     // Game end time
	FoodEaten          int           // Number of foods eaten
	PausedTime         time.Duration // Accumulated pause time
	PauseStart         time.Time     // Pause start time
	LastFoodSpawn      time.Time     // Last food spawn time
	LastObstacleSpawn  time.Time     // Last obstacle spawn time
	TimerStarted       bool          // Whether the竞技 timer has started

	// Message system
	Message     string // Current message to display
	MessageType string // Type of message: "normal", "bonus", "important"

	// AI Competitor Snake
	AISnake        []Point   `json:"aiSnake"`     // Body of the AI competitor
	AIDirection    Point     `json:"aiDirection"` // Current direction of AI
	AILastDir      Point     `json:"aiLastDir"`   // Last moved direction of AI
	AIBoosting     bool      `json:"aiBoosting"`  // Whether AI is boosting
	AIScore        int       `json:"aiScore"`     // AI's score
	AIStunnedUntil time.Time `json:"-"`           // When AI will recover from stun
	AIStunned      bool      `json:"aiStunned"`   // Whether AI is currently stunned
	AILastFireTime time.Time `json:"-"`           // Last time AI fired a fireball
	BerserkerMode  bool      `json:"berserker"`   // Whether AI is in aggressive mode

	// Obstacle system
	Obstacles []Obstacle // Temporary walls in the middle of the board

	// Fireball system
	Fireballs    []*Fireball // Active projectiles
	LastFireTime time.Time   // Cooldown management
	HitPoints    []Point     `json:"hitPoints"` // Points where fireballs hit something
	Winner       string      `json:"winner"`    // "player", "ai", or "draw"
	Mode         string      `json:"mode"`      // "zen" or "battle"
}

// FoodInfo is a DTO for food items sent to client
type FoodInfo struct {
	Pos              Point `json:"pos"`
	FoodType         int   `json:"foodType"`
	RemainingSeconds int   `json:"remainingSeconds"`
}

// GameState is a snapshot of the current game for client synchronization
type GameState struct {
	Snake         []Point      `json:"snake"`
	Foods         []FoodInfo   `json:"foods"`
	Score         int          `json:"score"`
	FoodEaten     int          `json:"foodEaten"`
	EatingSpeed   float64      `json:"eatingSpeed"`
	Started       bool         `json:"started"`
	GameOver      bool         `json:"gameOver"`
	Paused        bool         `json:"paused"`
	Boosting      bool         `json:"boosting"`
	AutoPlay      bool         `json:"autoPlay"`
	Difficulty    string       `json:"difficulty"`
	Message       string       `json:"message,omitempty"`
	MessageType   string       `json:"messageType,omitempty"` // "normal", "bonus", "important"
	CrashPoint    *Point       `json:"crashPoint,omitempty"`
	Obstacles     []Obstacle   `json:"obstacles"`
	Fireballs     []*Fireball  `json:"fireballs"`
	HitPoints     []Point      `json:"hitPoints"`
	AISnake       []Point      `json:"aiSnake"`
	AIScore       int          `json:"aiScore"`
	TimeRemaining int          `json:"timeRemaining"`
	Winner        string       `json:"winner"`
	AIStunned     bool         `json:"aiStunned"`
	PlayerStunned bool         `json:"playerStunned"`
	Mode          string       `json:"mode"`
	ScoreEvents   []ScoreEvent `json:"scoreEvents"`
	Berserker     bool         `json:"berserker"`
}

// GameConfig is a DTO for game settings sent to client on connect
type GameConfig struct {
	Width            int `json:"width"`
	Height           int `json:"height"`
	GameDuration     int `json:"gameDuration"`
	FireballCooldown int `json:"fireballCooldown"`
}
