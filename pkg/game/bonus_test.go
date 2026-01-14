package game

import (
	"testing"

	"github.com/trytobebee/snake_go/pkg/config"
)

// TestPositionBonus tests position-based scoring
func TestPositionBonus(t *testing.T) {
	// Test corner positions (should get +100)
	corners := []Point{
		{X: 1, Y: 1},                                // Top-left
		{X: config.Width - 2, Y: 1},                 // Top-right
		{X: 1, Y: config.Height - 2},                // Bottom-left
		{X: config.Width - 2, Y: config.Height - 2}, // Bottom-right
	}

	for _, pos := range corners {
		food := Food{
			Pos:      pos,
			FoodType: FoodRed,
		}
		bonus := food.GetPositionBonus(config.Width, config.Height)
		if bonus != 100 {
			t.Errorf("Corner position %v should have +100 bonus, got %d", pos, bonus)
		}

		msg := food.GetBonusMessage(config.Width, config.Height)
		if msg == "" {
			t.Errorf("Corner position should have congratulatory message")
		}
		t.Logf("Corner %v: bonus=%d, message='%s'", pos, bonus, msg)
	}

	// Test edge positions (should get +30)
	edges := []Point{
		{X: 10, Y: 1},                 // Top edge
		{X: 10, Y: config.Height - 2}, // Bottom edge
		{X: 1, Y: 10},                 // Left edge
		{X: config.Width - 2, Y: 10},  // Right edge
	}

	for _, pos := range edges {
		food := Food{
			Pos:      pos,
			FoodType: FoodBlue,
		}
		bonus := food.GetPositionBonus(config.Width, config.Height)
		if bonus != 30 {
			t.Errorf("Edge position %v should have +30 bonus, got %d", pos, bonus)
		}

		msg := food.GetBonusMessage(config.Width, config.Height)
		if msg == "" {
			t.Errorf("Edge position should have congratulatory message")
		}
		t.Logf("Edge %v: bonus=%d, message='%s'", pos, bonus, msg)
	}

	// Test normal positions (should get 0)
	normal := Point{X: 10, Y: 10} // Center
	food := Food{
		Pos:      normal,
		FoodType: FoodPurple,
	}
	bonus := food.GetPositionBonus(config.Width, config.Height)
	if bonus != 0 {
		t.Errorf("Normal position %v should have 0 bonus, got %d", normal, bonus)
	}

	msg := food.GetBonusMessage(config.Width, config.Height)
	if msg != "" {
		t.Errorf("Normal position should have no message, got '%s'", msg)
	}
	t.Logf("Normal %v: bonus=%d (no message)", normal, bonus)
}

// TestTotalScore tests combined base + position scoring
func TestTotalScore(t *testing.T) {
	tests := []struct {
		name     string
		pos      Point
		foodType FoodType
		expected int
	}{
		{"Red corner", Point{X: 1, Y: 1}, FoodRed, 140},       // 40 + 100
		{"Red edge", Point{X: 1, Y: 10}, FoodRed, 70},         // 40 + 30
		{"Red normal", Point{X: 10, Y: 10}, FoodRed, 40},      // 40 + 0
		{"Purple corner", Point{X: 1, Y: 1}, FoodPurple, 110}, // 10 + 100
		{"Blue edge", Point{X: 10, Y: 1}, FoodBlue, 50},       // 20 + 30
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			food := Food{
				Pos:      tc.pos,
				FoodType: tc.foodType,
			}
			score := food.GetTotalScore(config.Width, config.Height)
			if score != tc.expected {
				t.Errorf("%s: expected %d, got %d", tc.name, tc.expected, score)
			}
			t.Logf("%s: %d points (base=%d, bonus=%d)",
				tc.name, score, food.GetBaseScore(),
				food.GetPositionBonus(config.Width, config.Height))
		})
	}
}

// TestMessageSystem tests the message display system
func TestMessageSystem(t *testing.T) {
	g := NewGame()

	// Initially no message
	if g.GetMessage() != "" {
		t.Error("Should return empty string when no message set")
	}

	// Set a message
	g.SetMessage("Test message")

	if g.GetMessage() != "Test message" {
		t.Errorf("Expected 'Test message', got '%s'", g.GetMessage())
	}

	t.Log("✅ Message system works correctly!")
}

// TestVisualIndicators tests emoji display (now always shows original color)
func TestVisualIndicators(t *testing.T) {
	// Corner food should still show original color (bonus only in message)
	cornerFood := Food{
		Pos:      Point{X: 1, Y: 1},
		FoodType: FoodRed,
	}
	emoji := cornerFood.GetEmojiWithTimer(config.Width, config.Height)
	if emoji != "🔴" {
		t.Errorf("Corner food should show 🔴, got %s", emoji)
	}
	t.Logf("Corner food emoji: %s (bonus shown in message)", emoji)

	// Edge food should also show original color (bonus only in message)
	edgeFood := Food{
		Pos:      Point{X: 1, Y: 10},
		FoodType: FoodBlue,
	}
	emoji = edgeFood.GetEmojiWithTimer(config.Width, config.Height)
	if emoji != "🔵" {
		t.Errorf("Edge food should show 🔵, got %s", emoji)
	}
	t.Logf("Edge food emoji: %s (bonus shown in message)", emoji)

	// Normal food shows type emoji
	normalFood := Food{
		Pos:      Point{X: 10, Y: 10},
		FoodType: FoodRed,
	}
	emoji = normalFood.GetEmojiWithTimer(config.Width, config.Height)
	if emoji != "🔴" {
		t.Errorf("Normal red food should show 🔴, got %s", emoji)
	}
	t.Logf("Normal food emoji: %s", emoji)

	// But messages should contain trophy/star
	msg := cornerFood.GetBonusMessage(config.Width, config.Height)
	if msg == "" || !contains(msg, "🏆") {
		t.Errorf("Corner food message should contain 🏆, got: %s", msg)
	}
	t.Logf("Corner bonus message: %s", msg)

	msg = edgeFood.GetBonusMessage(config.Width, config.Height)
	if msg == "" || !contains(msg, "⭐") {
		t.Errorf("Edge food message should contain ⭐, got: %s", msg)
	}
	t.Logf("Edge bonus message: %s", msg)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
