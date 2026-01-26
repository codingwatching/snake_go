# 🐍 Snake Game (Go Premium)

A professional, high-performance Snake game engine written in Go. Features a high-fidelity **Web version**, a classic **Terminal version**, and a sophisticated **Reinforcement Learning AI**.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![Docker](https://img.shields.io/badge/Docker-Ready-blue?logo=docker)
![License](https://img.shields.io/badge/License-Non--Commercial-orange)

---

## ✨ Key Features

- 🌐 **Real-time Multiplayer**: Global **P2P Battle** mode with synchronized physics and matchmaking.
- 🧠 **Dual-Brain AI**: 
  - **Neural-RL**: 3-layer CNN trained via DQN (Reinforcement Learning).
  - **Heuristic**: Predictive spatial engine using Flood-fill and Greedy utility logic.
- 🏗️ **Clean Architecture**: Decoupled "Brain" controller system allows seamless human/AI hot-swapping.
- ⚔️ **Combat Mechanics**: Persistent scoring with **+50 headshots**, stuns, and body-shortening logic.
- 🎮 **Three Game Modes**: 
  - **Zen**: Stress-free practice with no time limits.
  - **Battle**: High-stakes match against the AI Competitor.
  - **P2P Battle**: Face off against other humans in real-time.
- 💾 **Robust Persistence**: SQLite-backed user accounts, global leaderboards, and match history.
- ⚡ **Performance**: 16ms BaseTick (60 FPS) with centralized ONNX inference queue (<1.5ms latency).
- 📼 **Data Pipeline**: Automatic frame recording to `.jsonl` for offline AI training.

---

## 📚 Technical Documentation

Explore the inner workings of the engine in the [docs/](./docs) directory:

- **Gameplay Mechanics**
  - [Neural & Hybrid AI Auto-Play](./docs/FEATURE_AI_AUTOPLAY.md)
  - [Boost & Speed Mechanics](./docs/FEATURE_BOOST.md)
  - [Scoring & Bonus System](./docs/FEATURE_POSITION_BONUS.md)
  - [Game Mode Design](./docs/GAME_MODES.md)
- **AI & Architecture**
  - [High-Performance AI Architecture](./docs/AI_ARCHITECTURE.md)
  - [RL Training & Reward Design](./docs/AI_TRAINING_DESIGN.md)
  - [Client vs Server Sync Engine](./docs/CLIENT_VS_SERVER.md)
  - [Code Structure & Package Layout](./docs/CODE_STRUCTURE.md)
- **Operations**
  - [Docker & Cloud Deployment Guide](./DEPLOY.md)
  - [ML Pipeline & Training Guide](./ml/README.md)
  - [Performance Optimizations](./docs/PERFORMANCE.md)

---

## 🚀 Quick Start

### 1. Local Development (Source)

Ensure you have **Go 1.25+** and **ONNX Runtime** installed.

```bash
# Clone and Enter
git clone https://github.com/trytobebee/snake_go.git
cd snake_go

# Install Dependencies
go mod tidy

# A. Run Modern Web Version (Recommended)
# Visit http://localhost:8080
go run ./cmd/webserver

# B. Run Classic Terminal Version
go run ./cmd/snake

# C. Run Replay Viewer (Re-watch recorded matches)
# Visit http://localhost:8081
go run ./cmd/replay
```

### 2. Docker Deployment (Cloud Ready)

The project is optimized for production deployment on Linux/Cloud servers (Alibaba Cloud, AWS, etc.) via Docker.

```bash
# Start everything in the background
docker compose up -d --build

# View logs
docker compose logs -f
```

*For detailed cloud deployment tips (proxies, mirrors, etc.), see **[DEPLOY.md](./DEPLOY.md)**.*

---

## 🛠️ System Requirements

### System Dependencies
- **Go**: 1.25.x
- **ONNX Runtime**: 1.19.2+ (Required for AI inference)
  - macOS: `brew install onnxruntime`
  - Linux: Follow [official guide](https://onnxruntime.ai/docs/install/)

### Python (For AI Training)
If you wish to retrain the neural model:
```bash
python3 -m venv venv
source venv/bin/activate
pip install -r ml/requirements.txt
```

---

## 🎨 Game Assets & Symbols

| Symbol | Meaning |
|:---:|---|
| � | **Player Head** (You) |
| 🟩 | **Player Body** |
| 🤖 | **AI / Opponent** |
| 🔴🟠🔵🟣 | **Food** (Various scores and durations) |
| 🔥 | **Fireball** (Combat / Destroy obstacles) |
| 🪨 | **Obstacle** (Destructible barrier) |
| � | **Boost** (3x Speed) |

---

## 📁 Project Layout

```text
snake_go/
├── cmd/                # Entry points (snake, webserver, replay)
├── pkg/                # Reusable Logic
│   ├── game/           # Physics, AI, Auth, DB, Recording
│   ├── renderer/       # Terminal View
│   └── config/         # Constants
├── ml/                 # Python RL Pipeline (PyTorch/DQN)
├── web/                # Frontend (HTML/CSS/JS Modules)
├── data/               # Persistent data (SQLite & Records)
└── docs/               # Technical Specs
```

---

## 📄 License & Contributing

- **License**: Non-Commercial Personal Use.
- **Contributions**: Feel free to submit PRs for new food types, AI strategies, or UI themes!

---

**Developed with ❤️ by Deepmind Advanced Agentic Coding Team.**
