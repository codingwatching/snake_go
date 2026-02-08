package config

import "time"

// Game board dimensions
const (
	StandardWidth  = 25
	StandardHeight = 25
	LargeWidth     = 38
	LargeHeight    = 38
	GameDuration   = 60 * time.Second
)

// Food spawn settings
const (
	FoodSpawnInterval = 5 * time.Second
	MaxFoodsOnBoard   = 12 // Increased for larger board
)

// Obstacle settings
const (
	ObstacleSpawnInterval = 10 * time.Second
	ObstacleDuration      = 30 * time.Second
	MaxObstacles          = 8 // Increased for larger board
)

// Prop settings
const (
	PropSpawnInterval = 12 * time.Second
	PropSpawnChance   = 20               // 20% chance every interval
	MaxPropsOnBoard   = 4                // Increased for larger board
	PropDuration      = 15 * time.Second // Time before disappearing
)

// Fireball settings
const (
	FireballSpeed    = 48 * time.Millisecond  // Time between fireball moves
	FireballCooldown = 300 * time.Millisecond // Time between shots
)

// Speed and boost settings
const (
	BaseTick = 16 * time.Millisecond // Base tick interval (~60 FPS)
	// Difficulty Settings (Number of BaseTicks per Move)
	LowTicks  = 22 // Decreased speed (was 18)
	MidTicks  = 16 // Decreased speed (was 13)
	HighTicks = 11 // Decreased speed (was 9)

	// Boost settings: how many ticks during boost (usually ~1/3 of normal)
	LowBoostTicks        = 8
	MidBoostTicks        = 5
	HighBoostTicks       = 4
	NormalTicksPerUpdate = 22                     // Normal speed: 16ms * 22 = 352ms
	BoostTicksPerUpdate  = 8                      // Boost: 16ms * 8 = 128ms
	BoostTimeout         = 150 * time.Millisecond // Boost timeout duration
	BoostThreshold       = 2                      // Consecutive key presses to trigger boost
	KeyRepeatWindow      = 200 * time.Millisecond // Time window for consecutive key detection
)

// Emoji characters for rendering
const (
	CharEmpty = "  " // Two spaces to match emoji width
	CharWall  = "⬜"
	CharHead  = "🟢"
	CharBody  = "🟩"
	CharCrash = "💥"
)
