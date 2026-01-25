package renderer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
	"github.com/trytobebee/snake_go/pkg/game"
)

// BenchmarkANSIClear benchmarks ANSI escape code clear (NEW)
func BenchmarkANSIClear(b *testing.B) {
	// Redirect stdout to discard
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Print("\033[H\033[2J")
	}
}

// BenchmarkExecClear benchmarks exec.Command("clear") (OLD)
func BenchmarkExecClear(b *testing.B) {
	// Redirect stdout to suppress output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	// Consume pipe output
	go func() {
		io.Copy(io.Discard, r)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// BenchmarkStringBuilderRender benchmarks buffered rendering (NEW)
func BenchmarkStringBuilderRender(b *testing.B) {
	// Redirect stdout to suppress output
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	g := game.NewGame()
	renderer := NewTerminalRenderer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(g, false)
	}
}

// BenchmarkNaiveRender benchmarks naive rendering with multiple fmt.Print (OLD)
func BenchmarkNaiveRender(b *testing.B) {
	// Redirect stdout to suppress output
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	g := game.NewGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		naiveRender(g, false)
	}
}

// naiveRender simulates old rendering approach with multiple fmt.Print calls
func naiveRender(g *game.Game, boosting bool) {
	// Old approach: multiple syscalls
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()

	board := make([][]int, config.Height)
	for i := range board {
		board[i] = make([]int, config.Width)
	}

	// Draw walls
	for x := 0; x < config.Width; x++ {
		board[0][x] = cellWall
		board[config.Height-1][x] = cellWall
	}
	for y := 0; y < config.Height; y++ {
		board[y][0] = cellWall
		board[y][config.Width-1] = cellWall
	}

	// Draw snake
	if len(g.Players) > 0 {
		for i, p := range g.Players[0].Snake {
			if i == 0 {
				board[p.Y][p.X] = cellHead
			} else {
				board[p.Y][p.X] = cellBody
			}
		}
	}

	// Multiple print calls (OLD WAY)
	fmt.Println("\n  🐍 SNAKE GAME 🐍")
	score := 0
	foodEaten := 0
	if len(g.Players) > 0 {
		score = g.Players[0].Score
		foodEaten = g.Players[0].FoodEaten
	}

	if boosting {
		fmt.Printf("  Score: %d  |  已吃: %d 个  |  🚀 BOOST!\n\n", score, foodEaten)
	} else {
		fmt.Printf("  Score: %d  |  已吃: %d 个\n\n", score, foodEaten)
	}

	for _, row := range board {
		fmt.Print("  ")
		for _, cell := range row {
			switch cell {
			case cellEmpty:
				fmt.Print(config.CharEmpty)
			case cellWall:
				fmt.Print(config.CharWall)
			case cellHead:
				fmt.Print(config.CharHead)
			case cellBody:
				fmt.Print(config.CharBody)
			}
		}
		fmt.Println()
	}

	fmt.Println("\n  Controls: WASD/Arrows")
}

// BenchmarkMemoryAllocation benchmarks memory allocation (pre-allocated vs new each time)
func BenchmarkPreAllocatedBoard(b *testing.B) {
	renderer := NewTerminalRenderer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset board (reusing memory)
		for y := range renderer.board {
			for x := range renderer.board[y] {
				renderer.board[y][x] = cellEmpty
			}
		}
	}
}

func BenchmarkNewBoardEachTime(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Allocate new board each time (OLD WAY)
		board := make([][]int, config.Height)
		for i := range board {
			board[i] = make([]int, config.Width)
		}
		// Reset it
		for y := range board {
			for x := range board[y] {
				board[y][x] = cellEmpty
			}
		}
	}
}

// BenchmarkStringConcatenation compares string building methods
func BenchmarkStringsBuilder(b *testing.B) {
	var buf strings.Builder
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		for j := 0; j < 100; j++ {
			buf.WriteString("test ")
		}
		_ = buf.String()
	}
}

func BenchmarkStringConcatenation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := ""
		for j := 0; j < 100; j++ {
			s += "test "
		}
		_ = s
	}
}

func BenchmarkBytesBuffer(b *testing.B) {
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		for j := 0; j < 100; j++ {
			buf.WriteString("test ")
		}
		_ = buf.String()
	}
}

// TestRenderingPerformance is a visual test showing timing differences
func TestRenderingPerformance(t *testing.T) {
	g := game.NewGame()
	renderer := NewTerminalRenderer()

	// Test optimized rendering
	start := time.Now()
	for i := 0; i < 100; i++ {
		renderer.Render(g, false)
	}
	optimizedTime := time.Since(start)

	// Test naive rendering
	start = time.Now()
	for i := 0; i < 100; i++ {
		naiveRender(g, false)
	}
	naiveTime := time.Since(start)

	improvement := float64(naiveTime) / float64(optimizedTime)

	t.Logf("Optimized rendering (100 frames): %v", optimizedTime)
	t.Logf("Naive rendering (100 frames): %v", naiveTime)
	t.Logf("Performance improvement: %.2fx faster", improvement)

	if improvement < 2.0 {
		t.Logf("Warning: Expected at least 2x improvement, got %.2fx", improvement)
	}
}
