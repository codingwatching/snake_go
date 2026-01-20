# 🐍 Snake Game (Go)

A modern Snake game written in Go, featuring both **Terminal** and **Web** versions with rich gameplay mechanics.

![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/License-Non--Commercial-orange)

## ✨ Features

- 🌐 **Dual Mode**: Terminal CLI and Web Browser versions
- 🧠 **Deep Learning AI**: Neural-network driven decision making for **Auto-Play mode**
- 🎮 **New Game Modes**: **Zen** (Infinite practice) and **Battle** (AI competition)
- ✨ **Floating Score Effects**: Animated score bubbles with glass-morphic design
- 🔥 **Fireball Combat System**: Shoot fireballs to destroy obstacles and stun AI
- 🐍 **AI Competitive Snake**: Battle against an intelligent, heuristic-driven AI rival
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

### 🔥 Intelligent AI Strategy
- **Auto-Play Brain**: 3-layer Convolutional Neural Network (CNN) trained via Reinforcement Learning (DQN).
- **Competitor AI**: Robust Heuristic engine using Flood-fill spatial awareness and Greedy utility logic.
- **Inference**: High-speed **ONNX Runtime** with C++ acceleration for neural models.
- **Latency**: Centralized task queue + worker pattern achieving **<1.5ms** total latency.
- **Hybrid Control**: Strategic movement + Heuristic safety fallback for autonomous play.
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

The project provides three main binaries: the **Terminal Game**, the **Web Server**, and the **Replay Viewer**.

### Run from Source

```bash
# Clone repository
git clone https://github.com/trytobebee/snake_go.git
cd snake_go

# Install dependencies
go mod tidy

# 1. Run Terminal version
go run ./cmd/snake

# 2. Run Web version (Server + UI)
# Then open http://localhost:8080
go run ./cmd/webserver

# 3. Run Replay Viewer
# Then open http://localhost:8081
go run ./cmd/replay
```

### Build Executables

```bash
# Build for current platform
go build -o snake ./cmd/snake
go build -o webserver ./cmd/webserver
go build -o replay ./cmd/replay
```

### Multi-Platform Distribution

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

The project follows a clean, modular architecture:

```
snake_go/
├── cmd/
│   ├── snake/               # Terminal game entry point
│   ├── webserver/           # Web version entry point (WebSocket server)
│   └── replay/              # Session replay tool entry point
├── pkg/
│   ├── game/                # Core game logic (Movement, Collisions, State)
│   │   ├── ai.go            # Heuristic & Hybrid AI logic
│   │   ├── ai_model.go      # ONNX Runtime inference service
│   │   ├── recorder.go      # Session recording system
│   │   └── food.go          # Multi-type food system
│   ├── renderer/            # Rendering layer (Terminal-based)
│   ├── input/               # Terminal input handling
│   └── config/              # Centralized game constants & settings
├── ml/                      # Machine Learning pipeline
│   ├── train.py             # DQN training script
│   ├── model.py             # CNN architecture
│   └── dataset.py           # JSONL to tensor conversion
├── web/
│   └── static/              # Frontend (HTML, CSS, JS modules)
├── docs/                    # Detailed technical documentation
├── build.sh                 # Multi-platform build script
└── README.md                # This file
```

### 🎯 Core Components

- **`cmd/webserver`**: Orchestrates the multi-user WebSocket server and serves the frontend.
- **`pkg/game`**: The "Physics Engine" of the game, shared across all versions.
- **`pkg/game/ai_model.go`**: Implements the high-performance singleton inference worker.
- **`ml/`**: Comprehensive Python environment for training the neural "brain".
- **`web/static/modules`**: Modularized ES6 JavaScript frontend for the canvas game.

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

Non-Commercial Personal Use License

## 🤝 Contributing

Contributions welcome! Please feel free to submit issues and pull requests.

## 🎯 Future Enhancements

- [ ] Boss battles or giant mode
- [ ] Power-ups (shield, ghost mode, etc.)
- [ ] Multiplayer mode (Real-time P2P)
- [ ] Skins and customization
