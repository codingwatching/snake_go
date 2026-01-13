package game

import (
	"testing"
	"time"
)

// TestFoodExpirationWithPause tests that food timers pause correctly
func TestFoodExpirationWithPause(t *testing.T) {
	// Create a red food (10 second duration)
	food := Food{
		Pos:       Point{X: 5, Y: 5},
		FoodType:  FoodRed,
		SpawnTime: time.Now(),
	}

	// Wait 2 seconds
	time.Sleep(2 * time.Second)

	// Check remaining time without pause (should be ~8 seconds)
	// Since PausedTimeAtSpawn is 0, passing 0 pause time means no pause during its life
	remaining1 := food.GetRemainingSeconds(0)
	if remaining1 < 7 || remaining1 > 9 {
		t.Errorf("Expected ~8 seconds remaining, got %d", remaining1)
	}
	t.Logf("After 2s real time, 0s pause: %d seconds remaining", remaining1)

	// Now simulate that those 2 seconds actually happened WHILE the game was paused.
	// So we pass 2 seconds as the current total game pause time.
	// Since food.PausedTimeAtSpawn is 0, it calculates: elapsed = 2s - (2s - 0s) = 0s.
	// Remaining should be the full 10s.
	pausedTime := 2 * time.Second
	remaining2 := food.GetRemainingSeconds(pausedTime)
	if remaining2 < 9 || remaining2 > 11 {
		t.Errorf("Expected ~10 seconds remaining with 2s pause, got %d", remaining2)
	}
	t.Logf("After 2s real time, 2s pause: %d seconds remaining (correctly compensated!)", remaining2)
}

// TestFoodExpirationDuringPause tests expiration check during active pause
func TestFoodExpirationDuringPause(t *testing.T) {
	// Create a food that will expire in 2 seconds
	food := Food{
		Pos:       Point{X: 5, Y: 5},
		FoodType:  FoodRed,                          // 10 second duration
		SpawnTime: time.Now().Add(-9 * time.Second), // Spawned 9 seconds ago
	}

	// Without pause, should have 1 second remaining
	if food.IsExpired(0) {
		t.Error("Food should not be expired yet without pause")
	}

	// Now simulate that the game has BEEN paused for 5 seconds total,
	// BUT the food was spawned when the game had already been paused for 5 seconds.
	// (e.g. food spawned during a pause, or pause happened before spawn)
	food.PausedTimeAtSpawn = 5 * time.Second

	// If we pass 5s as current pause, pausedSinceSpawn = 5 - 5 = 0.
	// Elapsed = 9s - 0 = 9s. Remaining = 10s - 9s = 1s.
	if food.IsExpired(5 * time.Second) {
		t.Error("Food should not be expired because it hasn't experienced any pause yet")
	}

	// Now simulate a NEW 5-second pause occurring AFTER spawn.
	// Current total pause = 10s.
	// pausedSinceSpawn = 10 - 5 = 5s.
	// Elapsed = 9s - 5s = 4s. Remaining = 10s - 4s = 6s.
	remainingWithPause := food.GetRemainingSeconds(10 * time.Second)
	if remainingWithPause < 5 || remainingWithPause > 7 {
		t.Errorf("Expected ~6 seconds remaining with post-spawn pause, got %d", remainingWithPause)
	}
}

// TestGamePauseIntegration tests the full integration with Game struct
func TestGamePauseIntegration(t *testing.T) {
	g := NewGame()

	// Get initial total paused time (should be 0)
	if g.GetTotalPausedTime() != 0 {
		t.Errorf("Expected 0 initial paused time, got %v", g.GetTotalPausedTime())
	}

	// Start pause
	g.TogglePause()
	if !g.Paused {
		t.Error("Game should be paused")
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Total paused time should now include current pause
	totalPaused := g.GetTotalPausedTime()
	if totalPaused < 90*time.Millisecond || totalPaused > 110*time.Millisecond {
		t.Errorf("Expected ~100ms paused, got %v", totalPaused)
	}
	t.Logf("While paused: total paused time = %v", totalPaused)

	// Resume
	g.TogglePause()
	if g.Paused {
		t.Error("Game should not be paused")
	}

	// Total paused time should now be accumulated
	accumulatedPause := g.GetTotalPausedTime()
	if accumulatedPause < 90*time.Millisecond || accumulatedPause > 110*time.Millisecond {
		t.Errorf("Expected ~100ms accumulated pause, got %v", accumulatedPause)
	}
	t.Logf("After resume: accumulated paused time = %v", accumulatedPause)

	// Wait a bit more
	time.Sleep(50 * time.Millisecond)

	// Total paused time should still be the same (not actively paused)
	if g.GetTotalPausedTime() != accumulatedPause {
		t.Error("Paused time should not increase when not paused")
	}

	t.Log("✅ Pause integration test passed!")
}

// TestDirectionValidation tests that 180-degree turns are prevented,
// even when multiple direction changes are attempted within a single tick.
func TestDirectionValidation(t *testing.T) {
	g := NewGame()
	// Initial state: Direction={1,0}, LastMoveDir={1,0} (Moving Right)

	// 1. Test basic 180-degree turn prevention
	if g.SetDirection(Point{X: -1, Y: 0}) {
		t.Error("Should have rejected immediate 180-degree turn (Left while moving Right)")
	}

	// 2. Test rapid input bug: Right -> Up -> Left
	// Press Up
	if !g.SetDirection(Point{X: 0, Y: -1}) {
		t.Error("Should have allowed turning Up while moving Right")
	}
	// At this point: Direction={0,-1}, LastMoveDir={1,0}

	// Press Left (Rapidly, before Update())
	if g.SetDirection(Point{X: -1, Y: 0}) {
		t.Error("Should have rejected turning Left because it's a 180-degree turn relative to the last move (Right)")
	}

	// 3. Test that it works correctly after Update()
	g.Update() // Move Up. LastMoveDir becomes {0,-1}
	if !g.SetDirection(Point{X: -1, Y: 0}) {
		t.Error("Should have allowed turning Left after moving Up")
	}
	if g.SetDirection(Point{X: 0, Y: 1}) {
		t.Error("Should have rejected turning Down because it's now a 180-degree turn relative to the last move (Up)")
	}

	t.Log("✅ Direction validation test passed!")
}
