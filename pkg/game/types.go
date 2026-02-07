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

// PropType represents different types of item props
type PropType int

const (
	PropShield      PropType = iota // Shield (Protects against 1 collision/fireball)
	PropTimeWarp                    // Time Warp (Slows down everyone else)
	PropTrimmer                     // Shorten snake (Removes last 3 segments)
	PropMagnet                      // Magnet (Attracts food nearby)
	PropChestBig                    // Big Treasure Chest (120 points)
	PropChestSmall                  // Small Treasure Chest (20 points)
	PropRapidFire                   // Rapid Fire (Reduces fireball cooldown)
	PropScatterShot                 // Scatter Shot (Fire 3 fireballs at once)
)

// Prop represents an item on the board
type Prop struct {
	Pos               Point
	Type              PropType
	SpawnTime         time.Time
	PausedTimeAtSpawn time.Duration
}

// EffectType represents duration-based status effects
type EffectType string

const (
	EffectShield      EffectType = "SHIELD"
	EffectTimeWarp    EffectType = "TIMEWARP"
	EffectMagnet      EffectType = "MAGNET"
	EffectRapidFire   EffectType = "RAPIDFIRE"
	EffectScatterShot EffectType = "SCATTER"
	EffectNone        EffectType = "NONE"
)

// ActiveEffect represents a status effect currently affecting a player
type ActiveEffect struct {
	Type     EffectType `json:"type"`
	ExpireAt time.Time  `json:"-"`
	Duration float64    `json:"duration"` // remaining seconds for client
}

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

// Player represents a participant in the game (human or AI)
type Player struct {
	Snake        []Point         `json:"snake"`
	Direction    Point           `json:"direction"`
	LastMoveDir  Point           `json:"lastMoveDir"`
	Score        int             `json:"score"`
	FoodEaten    int             `json:"foodEaten"`
	StunnedUntil time.Time       `json:"-"`
	Stunned      bool            `json:"stunned"`
	Boosting     bool            `json:"boosting"`
	LastFireTime time.Time       `json:"-"`
	Name         string          `json:"name"`
	Brain        Controller      `json:"-"`
	Controller   string          `json:"controllerType"` // "manual", "heuristic", "neural"
	Effects      []*ActiveEffect `json:"effects"`        // Status effects
}

// Game represents the main game state
type Game struct {
	Players           []*Player
	Foods             []Food       // Multiple food items
	LastScore         int          `json:"-"`           // Total score from previous frame (for reward calc - primarily for P1)
	ScoreEvents       []ScoreEvent `json:"scoreEvents"` // Recent scoring events
	GameOver          bool
	Paused            bool          // Pause state
	AutoPlay          bool          // Auto-play / Demo mode active (controls P1)
	CrashPoint        Point         // Collision position
	StartTime         time.Time     // Game start time
	EndTime           time.Time     // Game end time
	PausedTime        time.Duration // Accumulated pause time
	PauseStart        time.Time     // Pause start time
	LastFoodSpawn     time.Time     // Last food spawn time
	LastPropSpawn     time.Time     // Last item spawn time
	LastObstacleSpawn time.Time     // Last obstacle spawn time
	TimerStarted      bool          // Whether the竞技 timer has started

	// Message system
	Message     string // Current message to display
	MessageType string // Type of message: "normal", "bonus", "important"

	// Obstacle & Prop system
	Obstacles []Obstacle // Temporary walls in the middle of the board
	Props     []Prop     // Active items on board

	// Fireball system
	Fireballs []*Fireball // Active projectiles
	HitPoints []Point     `json:"hitPoints"` // Points where fireballs hit something
	Winner    string      `json:"winner"`    // "player", "ai", or "draw"
	Mode      string      `json:"mode"`      // "zen", "battle", or "pvp"
	IsPVP     bool        `json:"isPVP"`

	// Recording support
	CurrentAIContext AIContext     `json:"-"` // Last calculated AI context
	Recorder         *GameRecorder `json:"-"` // Active recorder
	NeuralNet        *ONNXModel    `json:"-"` // Loaded AI Model (ONNX Runtime)

	// Legacy support / Internal
	BerserkerMode bool `json:"berserker"` // Whether AI (if any) is in aggressive mode
}

// FoodInfo is a DTO for food items sent to client
type FoodInfo struct {
	Pos              Point `json:"pos"`
	FoodType         int   `json:"foodType"`
	RemainingSeconds int   `json:"remainingSeconds"`
}

// GameState is a snapshot of the current game for client synchronization
type GameState struct {
	Snake         []Point         `json:"snake"`
	Foods         []FoodInfo      `json:"foods"`
	Score         int             `json:"score"`
	FoodEaten     int             `json:"foodEaten"`
	EatingSpeed   float64         `json:"eatingSpeed"`
	Started       bool            `json:"started"`
	GameOver      bool            `json:"gameOver"`
	Paused        bool            `json:"paused"`
	Boosting      bool            `json:"boosting"`
	AutoPlay      bool            `json:"autoPlay"`
	Difficulty    string          `json:"difficulty"`
	Message       string          `json:"message,omitempty"`
	MessageType   string          `json:"messageType,omitempty"` // "normal", "bonus", "important"
	CrashPoint    *Point          `json:"crashPoint,omitempty"`
	Obstacles     []Obstacle      `json:"obstacles"`
	Fireballs     []*Fireball     `json:"fireballs"`
	HitPoints     []Point         `json:"hitPoints"`
	AISnake       []Point         `json:"aiSnake"`
	AIScore       int             `json:"aiScore"`
	TimeRemaining int             `json:"timeRemaining"`
	Winner        string          `json:"winner"`
	AIStunned     bool            `json:"aiStunned"`
	PlayerStunned bool            `json:"playerStunned"`
	Mode          string          `json:"mode"`
	ScoreEvents   []ScoreEvent    `json:"scoreEvents"`
	Berserker     bool            `json:"berserker"`
	IsPVP         bool            `json:"isPVP"`
	Props         []Prop          `json:"props"`
	P1Effects     []*ActiveEffect `json:"p1Effects"`
	P2Effects     []*ActiveEffect `json:"p2Effects"`
	P1Name        string          `json:"p1Name"`
	P2Name        string          `json:"p2Name"`
}

// GameConfig is a DTO for game settings sent to client on connect
type GameConfig struct {
	Width            int `json:"width"`
	Height           int `json:"height"`
	GameDuration     int `json:"gameDuration"`
	FireballCooldown int `json:"fireballCooldown"`
}

// --- Recording & AI Training Structures ---

// AIIntent represents the high-level strategic intent
type AIIntent string

const (
	IntentHunt    AIIntent = "HUNT"    // Actively seeking food
	IntentSurvive AIIntent = "SURVIVE" // Avoiding dead ends or threats
	IntentAttack  AIIntent = "ATTACK"  // Engaging enemy
	IntentIdle    AIIntent = "IDLE"    // No specific target
)

// AIContext records the internal state/decision of the AI
type AIContext struct {
	Intent    AIIntent `json:"intent"`
	TargetPos *Point   `json:"target_pos,omitempty"`
	Urgency   float64  `json:"urgency"` // 0.0 - 1.0
}

// ActionData represents the discrete action taken in a step
type ActionData struct {
	Direction Point `json:"dir"`
	Boost     bool  `json:"boost"`
	Fire      bool  `json:"fire"`
}

// StepRecord represents a single frame of game data for training
type StepRecord struct {
	StepID    int        `json:"step_id"`
	Timestamp int64      `json:"ts"` // Unix Milli
	State     GameState  `json:"state"`
	Action    ActionData `json:"action"`
	AIContext AIContext  `json:"ai_context"`
	Reward    float64    `json:"reward"`
	Done      bool       `json:"done"`
}

// LeaderboardEntry represents a single entry in the global leaderboard
type LeaderboardEntry struct {
	Name       string    `json:"name"`
	Score      int       `json:"score"`
	Date       time.Time `json:"date"`
	Difficulty string    `json:"difficulty"`
	Mode       string    `json:"mode"`
}
