# 🐍 Snake Game (Go Terminal)

A terminal-based Snake game written in Go, featuring emoji rendering and cross-platform support.

![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ Features

- 🤖 **AI Auto-Play (Demo Mode)** with space-awareness (Flood Fill)
- 🔊 **Dynamic Sound Effects** (Web Audio synthesized)
- 📳 **Haptic Feedback** for mobile devices
- 🚀 **Boost Mode** (Hold direction key)
- 🍎 Multiple food types with different scores and expiry times
- 📊 Real-time statistics (score, eating speed, food count)
- 📦 Single binary, no runtime dependencies
- 🌐 **Web & Terminal Support**

## 📚 Documentation

Detailed documentation on features and architecture can be found in the [docs/](./docs) directory:

- **Features**
  - [AI Auto-Play & Pathfinding](./docs/FEATURE_AI_AUTOPLAY.md)
  - [Boost Mechanism](./docs/FEATURE_BOOST.md)
  - [Position Scores & Bonuses](./docs/FEATURE_POSITION_BONUS.md)
- **Architecture**
  - [Web Version Overview](./docs/WEB_VERSION.md)
  - [Web Architecture & Design](./docs/WEB_ARCHITECTURE.md)
  - [Client vs Server Communication](./docs/CLIENT_VS_SERVER.md)
  - [Code Structure](./docs/CODE_STRUCTURE.md)
- **Optimization & Debugging**
  - [Performance Optimizations](./docs/PERFORMANCE.md)
  - [Bugfix: Pause Timer Logic](./docs/BUGFIX_PAUSE_TIMER.md)
  - [Bugfix: Screen Drift Fix](./docs/BUGFIX_SCREEN_DRIFT.md)

## 🎯 Game Features

### Multi-Type Food System
- 🔴 Red (40 points, 10s) - 15% spawn rate
- 🟠 Orange (30 points, 15s) - 20% spawn rate
- 🔵 Blue (20 points, 18s) - 25% spawn rate
- 🟣 Purple (10 points, 20s) - 35% spawn rate

### Countdown Display
Foods show a countdown timer in the last 5 seconds (🔴⁵ → 🔴¹)

### Boost Mechanism
Hold the current direction key to trigger 3x speed boost 🚀

### Real-time Stats
- Current score
- Eating speed (foods/second)
- Total foods eaten

## 🚀 Quick Start

### Run from Source

```bash
# Clone repository
git clone https://github.com/trytobebee/snake_go.git
cd snake_go

# Install dependencies
go mod tidy

# Run game
go run ./cmd/snake
```

### Build Executable

```bash
# Build for current platform
go build -o snake ./cmd/snake

# Run
./snake
```

### Cross-Platform Build

Use the build script to compile for all platforms:

```bash
chmod +x build.sh
./build.sh
```

This creates executables in `dist/`:
- `snake_game_mac_arm64` - macOS Apple Silicon
- `snake_game_mac_amd64` - macOS Intel
- `snake_game_windows.exe` - Windows
- `snake_game_linux` - Linux

## 🎮 Game Controls

| Key | Action |
|------|--------|
| ↑ / W | Move up |
| ↓ / S | Move down |
| ← / A | Move left |
| → / D | Move right |
| P / Space | Pause/resume |
| Q | Quit game |
| R | Restart (after game over) |

## 🎨 Game Elements

| Emoji | Meaning |
|-------|---------|
| ⬜ | Wall |
| 🟢 | Snake head |
| 🟩 | Snake body |
| 🔴🟠🔵🟣 | Food (different types) |
| 💥 | Crash point |

## 📁 Project Structure

The project follows a clean package architecture:

```
snake_go/
├── cmd/
│   └── snake/
│       └── main.go           # Entry point, game loop orchestration
├── pkg/
│   ├── game/                 # Core game logic
│   │   ├── types.go         # Game data structures
│   │   ├── game.go          # Game state management
│   │   ├── food.go          # Food-related logic
│   │   └── ai.go            # AI & Auto-play logic (New)
│   ├── renderer/             # Rendering layer
│   │   └── terminal.go      # Terminal-based renderer
│   ├── input/                # Input handling
│   │   └── keyboard.go      # Keyboard input management
│   └── config/               # Configuration
│       └── config.go        # Game constants and settings
├── build.sh                  # Multi-platform build script
├── go.mod                    # Go module definition
└── README.md                 # This file
```

### Package Responsibilities

- **`cmd/snake`**: Main entry point, coordinates all components
- **`pkg/game`**: Core game logic (snake movement, collision, scoring)
- **`pkg/renderer`**: Rendering abstraction (could support multiple renderers)
- **`pkg/input`**: Input handling abstraction
- **`pkg/config`**: Centralized configuration and constants

## 🔧 Dependencies

- [github.com/eiannone/keyboard](https://github.com/eiannone/keyboard) - Terminal keyboard input

## 📝 Implementation Details

### Snake Movement
Snake represented as coordinate array. Movement adds new head position and removes tail, creating smooth motion.

### Game Loop
Event-driven loop using Go's `time.Ticker` + `select`, handling both timed updates and keyboard input.

### Rendering
- ANSI escape codes for fast screen clearing (no external `clear` command)
- `strings.Builder` for buffered output (single write operation)
- Pre-allocated board to reduce GC pressure
- Emoji characters solve terminal aspect ratio issues

### Performance Optimizations
- Pre-allocated rendering buffers
- Single stdout write per frame
- Reusable data structures
- ANSI codes instead of shell commands

## 🏗️ Development

### Building
```bash
go build -o snake ./cmd/snake
```

### Testing
```bash
go test ./...
```

### Linting
```bash
golangci-lint run
```

## 📄 License

MIT License

## 🤝 Contributing

Contributions welcome! Please feel free to submit issues and pull requests.

## 🎯 Future Enhancements

- [ ] High score persistence
- [ ] Difficulty levels
- [ ] Power-ups (shield, time freeze, etc.)
- [ ] Obstacles
- [ ] Combo scoring system
- [ ] Sound effects (optional)
- [ ] Web-based renderer
- [ ] Multiplayer mode
