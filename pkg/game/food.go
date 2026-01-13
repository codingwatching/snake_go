package game

import "time"

// GetBaseScore returns the base score value of the food (without position bonus)
func (f *Food) GetBaseScore() int {
	switch f.FoodType {
	case FoodRed:
		return 40
	case FoodOrange:
		return 30
	case FoodBlue:
		return 20
	case FoodPurple:
		return 10
	default:
		return 10
	}
}

// GetPositionBonus returns bonus points based on food position difficulty
// Corner food: +100, Edge food: +30, Normal: 0
func (f *Food) GetPositionBonus(boardWidth, boardHeight int) int {
	x, y := f.Pos.X, f.Pos.Y

	// Check if in corner (most difficult)
	isTopLeft := x == 1 && y == 1
	isTopRight := x == boardWidth-2 && y == 1
	isBottomLeft := x == 1 && y == boardHeight-2
	isBottomRight := x == boardWidth-2 && y == boardHeight-2

	if isTopLeft || isTopRight || isBottomLeft || isBottomRight {
		return 100 // Corner bonus
	}

	// Check if on edge (difficult)
	isOnEdge := x == 1 || x == boardWidth-2 || y == 1 || y == boardHeight-2
	if isOnEdge {
		return 30 // Edge bonus
	}

	return 0 // Normal position
}

// GetTotalScore returns total score including base score and position bonus
func (f *Food) GetTotalScore(boardWidth, boardHeight int) int {
	return f.GetBaseScore() + f.GetPositionBonus(boardWidth, boardHeight)
}

// GetBonusMessage returns congratulatory message based on position bonus
func (f *Food) GetBonusMessage(boardWidth, boardHeight int) string {
	bonus := f.GetPositionBonus(boardWidth, boardHeight)

	if bonus == 100 {
		return "🏆 恭喜！角落挑战 +100 分！"
	} else if bonus == 30 {
		return "⭐ 不错！靠边奖励 +30 分！"
	}

	return ""
}

// GetDuration returns the food's lifetime duration
func (f *Food) GetDuration() time.Duration {
	switch f.FoodType {
	case FoodRed:
		return 10 * time.Second
	case FoodOrange:
		return 15 * time.Second
	case FoodBlue:
		return 18 * time.Second
	case FoodPurple:
		return 20 * time.Second
	default:
		return 20 * time.Second
	}
}

// IsExpired checks if the food has expired, accounting for paused time that occurred AFTER spawn
func (f *Food) IsExpired(currentTotalPaused time.Duration) bool {
	pausedSinceSpawn := currentTotalPaused - f.PausedTimeAtSpawn
	elapsed := time.Since(f.SpawnTime) - pausedSinceSpawn
	return elapsed > f.GetDuration()
}

// GetRemainingSeconds returns remaining seconds before expiration, accounting for paused time AFTER spawn
func (f *Food) GetRemainingSeconds(currentTotalPaused time.Duration) int {
	pausedSinceSpawn := currentTotalPaused - f.PausedTimeAtSpawn
	elapsed := time.Since(f.SpawnTime) - pausedSinceSpawn
	remaining := f.GetDuration() - elapsed
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// GetEmoji returns the emoji for the food type
func (f *Food) GetEmoji() string {
	switch f.FoodType {
	case FoodRed:
		return "🔴"
	case FoodOrange:
		return "🟠"
	case FoodBlue:
		return "🔵"
	case FoodPurple:
		return "🟣"
	default:
		return "🟣"
	}
}

// GetEmojiWithTimer returns the food emoji (always shows original color)
func (f *Food) GetEmojiWithTimer(boardWidth, boardHeight int) string {
	return f.GetEmoji()
}

// GetTimerEmoji returns countdown number emoji if in countdown phase
func (f *Food) GetTimerEmoji(pausedTime time.Duration) string {
	remaining := f.GetRemainingSeconds(pausedTime)

	// Show countdown for last 5 seconds
	if remaining <= 5 && remaining > 0 {
		circledNums := map[int]string{
			1: "①",
			2: "②",
			3: "③",
			4: "④",
			5: "⑤",
		}
		return circledNums[remaining]
	}

	return "" // Not in countdown phase
}
