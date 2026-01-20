# 🐍 Snake Game (Go)

A modern Snake game written in Go, featuring both **Terminal** and **Web** versions with rich gameplay mechanics.

![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ Features

- 🌐 **Dual Mode**: Terminal CLI and Web Browser versions
- 🧠 **Deep Learning AI**: Neural-network driven decision making using ONNX Runtime
- 🎮 **New Game Modes**: **Zen** (Infinite practice) and **Battle** (AI competition)
- ✨ **Floating Score Effects**: Animated score bubbles with glass-morphic design
- 🔥 **Fireball Combat System**: Shoot fireballs to destroy obstacles and stun AI
- 🐍 **AI Competitive Snake**: Battle against an intelligent AI rival powered by RL
- 🚀 **High-Performance Inference**: Global task queue with <2ms latency
- 🔊 **Dynamic Sound Effects** (Web Audio synthesized)
- 📳 **Haptic Feedback** for mobile devices
- ⚡ **Boost Mode**: Hold direction key for 3x speed
- 🍎 **Multi-Type Food System** with different scores and expiry times
- 📊 **Real-time Statistics** (score, eating speed, food count)
- 📱 **Mobile-Friendly** web interface with touch controls
- 💾 **High Score Persistence** (localStorage for web)
- 📼 **Session Recording**: Full JSONL-based game capture for ML training

## 📚 Documentation

Detailed documentation on features and architecture can be found in the [docs/](./docs) directory:

- **Features**
  - [Neural & Hybrid AI Auto-Play](./docs/FEATURE_AI_AUTOPLAY.md)
  - [Boost Mechanism](./docs/FEATURE_BOOST.md)
  - [Position Scores & Bonuses](./docs/FEATURE_POSITION_BONUS.md)
- **Architecture & AI**
  - [High-Performance AI Architecture](./docs/AI_ARCHITECTURE.md)
  - [RL Training & Reward Design](./docs/AI_TRAINING_DESIGN.md)
  - [ML Pipeline & Dataset Guide](./ml/README.md)
  - [Web Version Overview](./docs/WEB_VERSION.md)
  - [Client vs Server Sync Engine](./docs/CLIENT_VS_SERVER.md)
  - [Code Structure](./docs/CODE_STRUCTURE.md)
- **Optimization & Debugging**
  - [Performance Optimizations](./docs/PERFORMANCE.md)
  - [Config Synchronization Logic](./docs/SYNC_ENGINE.md)
  - [Game Mode Design](./docs/GAME_MODES.md)

## 🎯 Game Features

### New Game Modes 🎮
- **🧘 Zen Mode**: No time limit, no AI opponent. Perfect for practicing controls, exploring the multi-food system, and enjoying a relaxed experience.
- **⚔️ Battle Mode**: Compete against an AI snake within a 30-second time limit. Includes combat mechanics (stun AI with fireballs!).

### Floating Score Feedback 📈
- **Dynamic Bubbles**: Floating score labels pop up exactly where points are earned.
- **Visual Design**: Sleek glass-morphic capsule design with smooth upward-floating and fading animations.
- **Contextual Colors**: Blue for food, red for combat hits, and **gold** for critical AI headshots.

### Multi-Type Food System
- 🔴 Red (40 points, 10s) - 15% spawn rate
- 🟠 Orange (30 points, 15s) - 20% spawn rate
- 🔵 Blue (20 points, 18s) - 25% spawn rate
- 🟣 Purple (10 points, 20s) - 35% spawn rate

### 🔥 High-Performance AI System
- **Brain**: 3-layer Convolutional Neural Network (CNN) trained via Reinforcement Learning (DQN).
- **Inference**: Powered by **ONNX Runtime** with C++ acceleration.
- **Latency**: Centralized task queue + worker pattern achieving **<1.5ms** total latency.
- **Hybrid Control**: Deep learning strategic movement + Heuristic safety fallback.
- **Safety Gate**: Physical collision look-ahead prevents suicidal moves during inference.

### 🌐 High-Fidelity Sync Engine
- **BaseTick**: 16ms (matches 60 FPS display refresh rate).
- **Dual-Bus Logic**: Instant input response (Reader Goroutine) + Precise physical updates (Main Loop).
- **UI Responsiveness**: Immediate state push on user action to eliminate perceived input lag.

### 📼 Recording & Replay
- **Data Capture**: Every frame is recorded to `.jsonl` files (S, A, R, S' transitions).
- **AI Training**: Seamless pipeline from game logs to PyTorch training and ONNX export.
- **Visual Replay**: High-fidelity replay tool running on port 8081.

### Countdown Display
Foods show a visual countdown timer with pulsating animation in the last 5 seconds

### Obstacle System 🪨
- **Dynamic Spawning**: Obstacles appear randomly during gameplay
- **Destructible**: Use fireballs to destroy obstacle blocks
- **Temporary**: Obstacles expire after a set duration
- **Strategic Challenge**: Navigate around or destroy them for bonus points

### Fireball Combat System 🔥
- **Shoot Projectiles**: Fire fireballs in your current direction
- **Destroy Obstacles**: Earn +10 points for each block destroyed
- **Cooldown**: 300ms between shots
- **Fast Travel**: Fireballs move faster than the snake
- **Smart Collision**: Fireballs pass through snake head but hit body/walls/obstacles
- **Visual Effects**: Explosion animations on impact

### Difficulty Levels
- **Low**: Slower snake speed, easier gameplay
- **Mid**: Moderate speed, balanced challenge
- **High**: Fast-paced action for experienced players
- Change difficulty before starting or after game over

### Boost Mechanism
Hold the current direction key to trigger 3x speed boost 🚀

### Real-time Stats
- Current score
- Eating speed (foods/second)
- Total foods eaten
- Best score (persisted)

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

- [ ] Boss battles or giant mode
- [ ] Power-ups (shield, ghost mode, etc.)
- [ ] Multiplayer mode (Real-time P2P)
- [ ] Skins and customization
