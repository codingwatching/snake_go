package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	width  = 25 // 减小宽度，因为 emoji 是双倍宽度
	height = 25
)

type Point struct {
	x, y int
}

type Game struct {
	snake      []Point
	food       Point
	direction  Point
	score      int
	gameOver   bool
	paused     bool  // 暂停状态
	crashPoint Point // 碰撞位置
}

func NewGame() *Game {
	g := &Game{
		snake:     []Point{{x: width / 2, y: height / 2}},
		direction: Point{x: 1, y: 0},
		score:     0,
		gameOver:  false,
	}
	g.spawnFood()
	return g
}

func (g *Game) spawnFood() {
	for {
		g.food = Point{
			x: rand.Intn(width-2) + 1,
			y: rand.Intn(height-2) + 1,
		}
		// Make sure food doesn't spawn on snake
		onSnake := false
		for _, p := range g.snake {
			if p == g.food {
				onSnake = true
				break
			}
		}
		if !onSnake {
			break
		}
	}
}

func (g *Game) update() {
	if g.gameOver {
		return
	}

	// Calculate new head position
	head := g.snake[0]
	newHead := Point{
		x: head.x + g.direction.x,
		y: head.y + g.direction.y,
	}

	// Check wall collision
	if newHead.x <= 0 || newHead.x >= width-1 || newHead.y <= 0 || newHead.y >= height-1 {
		g.gameOver = true
		g.crashPoint = newHead
		return
	}

	// Check self collision
	for _, p := range g.snake {
		if p == newHead {
			g.gameOver = true
			g.crashPoint = newHead
			return
		}
	}

	// Move snake
	g.snake = append([]Point{newHead}, g.snake...)

	// Check food collision
	if newHead == g.food {
		g.score += 10
		g.spawnFood()
	} else {
		// Remove tail if no food eaten
		g.snake = g.snake[:len(g.snake)-1]
	}
}

func (g *Game) render() {
	clearScreen()

	// Cell types for coloring
	const (
		cellEmpty = iota
		cellWall
		cellFood
		cellHead
		cellBody
		cellCrash
	)

	// Create the board
	board := make([][]int, height)
	for i := range board {
		board[i] = make([]int, width)
	}

	// Draw walls
	for x := 0; x < width; x++ {
		board[0][x] = cellWall
		board[height-1][x] = cellWall
	}
	for y := 0; y < height; y++ {
		board[y][0] = cellWall
		board[y][width-1] = cellWall
	}

	// Draw food
	board[g.food.y][g.food.x] = cellFood

	// Draw snake
	for i, p := range g.snake {
		if i == 0 {
			board[p.y][p.x] = cellHead
		} else {
			board[p.y][p.x] = cellBody
		}
	}

	// Draw crash point if game over
	if g.gameOver {
		// 确保碰撞点在边界内才显示
		if g.crashPoint.x >= 0 && g.crashPoint.x < width && g.crashPoint.y >= 0 && g.crashPoint.y < height {
			board[g.crashPoint.y][g.crashPoint.x] = cellCrash
		}
	}

	// Emoji squares (these are typically rendered as perfect squares)
	const (
		charEmpty = "  " // Two spaces to match emoji width
		charWall  = "⬜"
		charFood  = "🔴"
		charHead  = "🟢"
		charBody  = "🟩"
		charCrash = "💥"
	)

	// Print board
	fmt.Println("\n  🐍 SNAKE GAME 🐍")
	fmt.Printf("  Score: %d\n\n", g.score)
	for _, row := range board {
		fmt.Print("  ")
		for _, cell := range row {
			switch cell {
			case cellEmpty:
				fmt.Print(charEmpty)
			case cellWall:
				fmt.Print(charWall)
			case cellFood:
				fmt.Print(charFood)
			case cellHead:
				fmt.Print(charHead)
			case cellBody:
				fmt.Print(charBody)
			case cellCrash:
				fmt.Print(charCrash)
			}
		}
		fmt.Println()
	}
	fmt.Println("\n  Use WASD or Arrow keys to move, P to pause, Q to quit")

	if g.paused {
		fmt.Println("\n  ⏸️  PAUSED - Press P to continue")
	}

	if g.gameOver {
		fmt.Println("\n  💀 GAME OVER! Press R to restart or Q to quit")
	}
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := keyboard.Open(); err != nil {
		fmt.Println("Error opening keyboard:", err)
		return
	}
	defer keyboard.Close()

	game := NewGame()

	// Input channel - sends both char and key
	type keyInput struct {
		char rune
		key  keyboard.Key
	}
	inputChan := make(chan keyInput)
	go func() {
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				return
			}
			inputChan <- keyInput{char: char, key: key}
		}
	}()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	game.render()

	for {
		select {
		case input := <-inputChan:
			// Handle arrow keys
			switch input.key {
			case keyboard.KeyArrowUp:
				if game.direction.y != 1 {
					game.direction = Point{x: 0, y: -1}
				}
			case keyboard.KeyArrowDown:
				if game.direction.y != -1 {
					game.direction = Point{x: 0, y: 1}
				}
			case keyboard.KeyArrowLeft:
				if game.direction.x != 1 {
					game.direction = Point{x: -1, y: 0}
				}
			case keyboard.KeyArrowRight:
				if game.direction.x != -1 {
					game.direction = Point{x: 1, y: 0}
				}
			}
			// Handle character keys
			switch input.char {
			case 'w', 'W':
				if game.direction.y != 1 {
					game.direction = Point{x: 0, y: -1}
				}
			case 's', 'S':
				if game.direction.y != -1 {
					game.direction = Point{x: 0, y: 1}
				}
			case 'a', 'A':
				if game.direction.x != 1 {
					game.direction = Point{x: -1, y: 0}
				}
			case 'd', 'D':
				if game.direction.x != -1 {
					game.direction = Point{x: 1, y: 0}
				}
			case 'q', 'Q':
				fmt.Println("\n  Thanks for playing! 👋")
				return
			case 'r', 'R':
				if game.gameOver {
					game = NewGame()
				}
			case 'p', 'P', ' ':
				if !game.gameOver {
					game.paused = !game.paused
					game.render()
				}
			}
		case <-ticker.C:
			if !game.gameOver && !game.paused {
				game.update()
			}
			game.render()
		}
	}
}
