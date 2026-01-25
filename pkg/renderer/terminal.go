package renderer

import (
	"fmt"
	"strings"

	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
)

// TerminalRenderer handles terminal-based rendering
type TerminalRenderer struct {
	board  [][]int
	buffer strings.Builder
}

// Cell types for the board
const (
	cellEmpty = iota
	cellWall
	cellHead
	cellBody
	cellCrash
	cellAIHead
	cellAIBody
)

// NewTerminalRenderer creates a new terminal renderer
func NewTerminalRenderer() *TerminalRenderer {
	// Pre-allocate board to reduce GC pressure
	board := make([][]int, config.Height)
	for i := range board {
		board[i] = make([]int, config.Width)
	}

	return &TerminalRenderer{
		board: board,
	}
}

// clearScreen clears the terminal using ANSI escape codes
func (r *TerminalRenderer) clearScreen() {
	fmt.Print("\033[H\033[2J\033[3J")
}

// ShowCursor shows the cursor (call on exit)
func (r *TerminalRenderer) ShowCursor() {
	fmt.Print("\033[?25h")
}

// HideCursor hides the cursor (call on start)
func (r *TerminalRenderer) HideCursor() {
	fmt.Print("\033[?25l")
}

// Render renders the game state to the terminal
func (r *TerminalRenderer) Render(g *game.Game, boosting bool) {
	r.clearScreen()
	r.buffer.Reset()

	// Reset board
	for y := range r.board {
		for x := range r.board[y] {
			r.board[y][x] = cellEmpty
		}
	}

	// Create food emoji maps
	foodEmojis := make(map[game.Point]string)
	timerEmojis := make(map[game.Point]string)
	for _, food := range g.Foods {
		foodEmojis[food.Pos] = food.GetEmojiWithTimer(config.Width, config.Height)
		// Show timer only when game is not over
		if !g.GameOver {
			timerEmoji := food.GetTimerEmoji(g.GetTotalPausedTime())
			if timerEmoji != "" {
				timerPos := game.Point{X: food.Pos.X + 1, Y: food.Pos.Y}
				timerEmojis[timerPos] = timerEmoji
			}
		}
	}

	// Draw walls
	for x := 0; x < config.Width; x++ {
		r.board[0][x] = cellWall
		r.board[config.Height-1][x] = cellWall
	}
	for y := 0; y < config.Height; y++ {
		r.board[y][0] = cellWall
		r.board[y][config.Width-1] = cellWall
	}

	// Draw snake (P1)
	if len(g.Players) > 0 {
		for i, p := range g.Players[0].Snake {
			if i == 0 {
				r.board[p.Y][p.X] = cellHead
			} else {
				r.board[p.Y][p.X] = cellBody
			}
		}
	}

	// Draw AI/P2 snake
	if len(g.Players) > 1 {
		for i, p := range g.Players[1].Snake {
			if i == 0 {
				r.board[p.Y][p.X] = cellAIHead
			} else {
				r.board[p.Y][p.X] = cellAIBody
			}
		}
	}

	// Draw crash point if game over
	if g.GameOver {
		if g.CrashPoint.X >= 0 && g.CrashPoint.X < config.Width &&
			g.CrashPoint.Y >= 0 && g.CrashPoint.Y < config.Height {
			r.board[g.CrashPoint.Y][g.CrashPoint.X] = cellCrash
		}
	}

	// Build output using string builder
	r.buffer.WriteString("\n  🐍 SNAKE GAME 🐍\n")

	// Header with stats
	boostStr := ""
	if boosting {
		boostStr = "  |  🚀 BOOST!"
	}
	p1Score := 0
	p1FoodEaten := 0
	if len(g.Players) > 0 {
		p1Score = g.Players[0].Score
		p1FoodEaten = g.Players[0].FoodEaten
	}
	p2Score := 0
	if len(g.Players) > 1 {
		p2Score = g.Players[1].Score
	}

	r.buffer.WriteString(fmt.Sprintf("  Score: %d  |  AI/P2 Score: %d  |  Time Left: %ds  |  吃豆速度: %.2f 个/秒  |  已吃: %d 个%s\n",
		p1Score, p2Score, g.GetTimeRemaining(), g.GetEatingSpeed(), p1FoodEaten, boostStr))

	if msg := g.GetMessage(); msg != "" {
		r.buffer.WriteString("  " + msg + "\n")
	} else {
		r.buffer.WriteString("\n")
	}
	r.buffer.WriteString("\n")

	// Render board
	for y, row := range r.board {
		r.buffer.WriteString("  ")
		for x, cell := range row {
			pos := game.Point{X: x, Y: y}
			if timer, hasTimer := timerEmojis[pos]; hasTimer && cell == cellEmpty {
				r.buffer.WriteString(timer)
				r.buffer.WriteString(" ")
			} else if emoji, hasFood := foodEmojis[pos]; hasFood && cell == cellEmpty {
				r.buffer.WriteString(emoji)
			} else {
				switch cell {
				case cellEmpty:
					r.buffer.WriteString(config.CharEmpty)
				case cellWall:
					r.buffer.WriteString(config.CharWall)
				case cellHead:
					r.buffer.WriteString(config.CharHead)
				case cellBody:
					r.buffer.WriteString(config.CharBody)
				case cellCrash:
					r.buffer.WriteString(config.CharCrash)
				case cellAIHead:
					r.buffer.WriteString("🤖")
				case cellAIBody:
					r.buffer.WriteString("🤖")
				}
			}
		}
		r.buffer.WriteString("\n")
	}

	r.buffer.WriteString("\n  Use WASD or Arrow keys to move, hold direction key to boost 🚀\n")
	r.buffer.WriteString("  P to pause, Q to quit\n")

	if g.Paused {
		r.buffer.WriteString("\n  ⏸️  PAUSED - Press P to continue\n")
	}

	if g.GameOver {
		r.buffer.WriteString("\n  💀 GAME OVER! Press R to restart or Q to quit\n")
	}

	fmt.Print(r.buffer.String())
}
