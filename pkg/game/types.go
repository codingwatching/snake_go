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

// Game represents the main game state
type Game struct {
	Snake         []Point
	Foods         []Food // Multiple food items
	Direction     Point
	LastMoveDir   Point // Direction of the last performed move
	Score         int
	GameOver      bool
	Paused        bool          // Pause state
	AutoPlay      bool          // Auto-play / Demo mode active
	Boosting      bool          // Active boosting state
	CrashPoint    Point         // Collision position
	StartTime     time.Time     // Game start time
	EndTime       time.Time     // Game end time
	FoodEaten     int           // Number of foods eaten
	PausedTime    time.Duration // Accumulated pause time
	PauseStart    time.Time     // Pause start time
	LastFoodSpawn time.Time     // Last food spawn time

	// Message system
	Message         string        // Current message to display
	MessageTime     time.Time     // When message was set
	MessageDuration time.Duration // How long to show message
}
