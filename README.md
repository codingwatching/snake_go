# 🐍 Snake Game (Go)

A modern Snake game written in Go, featuring both **Terminal** and **Web** versions with rich gameplay mechanics.

![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/License-Non--Commercial-orange)

## ✨ Features

- 🌐 **Online Multiplayer**: Real-time **P2P Battle** with matchmaking and shared game state
- 🧠 **Multi-Brain AI**: Choose between **Neural-RL** and **Heuristic** agents for Auto-Play
- 🏗️ **Modular Architecture**: Decoupled "Brain" (Controller) system for human/AI interchangeability
- ⚔️ **Advanced Combat**: Persistent scoring with **+50 headshots**, stuns, and body shortening
- 🎮 **Three Modes**: **Zen** (Practice), **Battle** (vs AI), and **P2P Battle** (vs Humans)
- 💾 **SQLite Persistence**: Robust tracking for accounts, match history, and leaderboards
- 🚀 **Docker Ready**: One-click deployment with `docker-compose`
- 📼 **Session Recording**: Full JSONL-based game capture for ML training
- ⚡ **Boost Mode**: Hold direction key for 3x speed boost
- 📳 **Haptic Feedback** & synthesized sound effects

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
- **👥 P2P Battle Mode**: Real-time online matchmaking. Play against another human on the same board with synchronized physics.

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

## 🛠️ Dev Environment Setup
26: 
27: Follow these steps to set up a complete development environment, including the AI training pipeline.
28: 
29: ### 1. Install Go
30: Ensure you have Go installed (version 1.21 or higher recommended).
31: - **macOS**: `brew install go`
32: - **Official Download**: [go.dev/dl](https://go.dev/dl/)
33: 
34: ### 2. Install System Dependencies (Crucial for AI)
35: The AI inference engine relies on the C++ ONNX Runtime library. You **must** install this system-level dependency.
36: 
37: - **macOS (Homebrew)**:
38:   ```bash
39:   brew install onnxruntime
40:   ```
41: - **Linux/Windows**: Follow instructions at [ONNX Runtime Get Started](https://onnxruntime.ai/docs/install/).
42: 
43: ### 3. Initialize Project & Go Modules
44: 
45: ```bash
46: # Clone repository
47: git clone https://github.com/trytobebee/snake_go.git
48: cd snake_go
49: 
50: # Download and verify Go dependencies
51: go mod tidy
52: ```
53: 
54: ### 4. Setup Python ML Environment
55: If you want to train your own AI models or run the training pipeline, you need a Python environment.
56: *Note: Supports Python 3.9 through 3.14. (Patches included for 3.14+ compatibility)*
57: 
58: ```bash
59: # 1. Create a virtual environment (recommended)
60: python3 -m venv venv
61: 
62: # 2. Activate it
63: source venv/bin/activate  # macOS/Linux
64: # .\venv\Scripts\activate # Windows
65: 
66: # 3. Install Python dependencies
67: pip install -r ml/requirements.txt
68: ```
69: 
70: ### 5. Initialize the AI Brain (Optional)
71: The game comes with a pre-trained model, but if you want to generate a fresh one from your recorded gameplay:
72: 
73: ```bash
74: cd ml
75: python3 train.py
76: # This will generate 'checkpoints/snake_policy.onnx' used by the game.
77: cd ..
78: ```
79: 
80: ## 🚀 Quick Start

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
| Space | Pause/resume |
| P | **Auto-Play Toggle** (Neural/Heuristic) |
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
- [x] Multiplayer mode (Real-time P2P)
- [ ] Skins and customization
