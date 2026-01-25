# Changelog

All notable changes to this project will be documented in this file.

## [5.0.0] - 2026-01-26

### 👥 Real-Time P2P Matchmaking
- **MatchMaker Service**: Added a centralized lobby system for pairing players.
  - Automated queue management with background matching logic.
  - Interactive "Match Found" countdown and lobby state sync.
- **Shared Game Instances**: Implemented a "Master-Observer" synchronization pattern for PVP.
  - Two players share a single `Game` instance on the server for zero-offset state consistency.
  - Role-based input routing (P1 vs P2) within the same engine tick.

### 🏗️ Controller-Based Architecture (The Brain Overhaul)
- **Controller Interface**: Replaced hardcoded input with a modular `Controller` pattern.
  - **ManualController**: Handles human input via WebSockets.
  - **HeuristicController**: Advanced rule-based AI (now works for any player slot).
  - **NeuralController**: Reinforcement Learning (RL) agent.
- **Dynamic Brain Swapping**: Support for real-time AI assistance.
  - Toggle between Manual and AI modes mid-game.
  - UI-selectable AI flavors (Neural vs Heuristic) during Auto-Play.
- **Decoupled Viewports**: AI now perceives the game relative to its own head, enabling RL models to drive P2 snakes correctly.

### ⚔️ Combat & Gameplay Overhaul
- **Advanced Scoring & Penalties**:
  - **Headshots**: Reward attacker with **+50 pts** and stun victim for **2 seconds** (No victim score loss).
  - **Body Hits**: Reward attacker with **+10 pts** and remove **1 segment** from victim (No victim stun).
- **Persistent Game-Over State**: 
  - The game timer now **freezes** at the exact moment of collision instead of clearing to zero.
- **Control Optimization**:
  - Remapped Auto-Play toggle to **`P`** (Play) to avoid WASD conflicts.
  - Spacebar is now the dedicated **Pause** key.
- **Safety Polish**: Fixed the "180-degree turn" bug that caused accidental self-collisions during rapid input.

### 💾 Persistence & Backend
- **SQLite Integration**: Replaced raw file storage with a robust database layer.
  - Fully persistent `users` table for accounts/passwords.
  - `game_sessions` table for detailed match history.
  - `leaderboard` table for high scores and win rates.
- **Deployment**:
  - Added full **Docker** & **Docker Compose** support for one-click deployment.
  - Updated build scripts for multi-arch releases (Arm64/Amd64).

---

## [4.1.0] - 2026-01-21
 
### 🛠️ Developer Experience & Environment
- **Documentation Overhaul**:
  - Added comprehensive "Dev Environment Setup" to README.
  - Clarified system dependencies (ONNX Runtime) and Python environment setup.
  - Added specific instructions for macOS Apple Silicon vs Intel.
- **Python Compatibility**:
  - Implemented automatic runtime Monkey Patch for `onnxscript` to support Python 3.14.
  - Re-enabled high-performance ONNX Opset 18 export for modern PyTorch versions.
 
### 🤖 Global AI Service
- **Robustness**:
  - Enhanced `StartInferenceService` to automatically detect ONNX Runtime libraries in multiple standard locations (Homebrew paths for M-series/Intel Macs & Linux).
  - Added standalone diagnostic tool `cmd/check_onnx` to verify library linkage.
 
### 🐛 Stability & Fixes
- **Game Recording Logic**:
  - Fixed a critical issue where game recordings were not saved if the game ended via AI victory or timeout (loop exit prevented final save).
  - Removed artificial "+50/-100" rewards from the final frame to ensure cleaner, score-based ground truth for RL training.
- **Logging**:
  - Standardized all server output to use `log` (with timestamps) instead of raw `fmt.Print`.
  - Removed misleading "Game Over" print statements that cluttered the terminal.
 
---
 
## [4.0.0] - 2026-01-20

### 🧠 Deep Learning AI Integration
- **ONNX Runtime Engine**:
  - Replaced legacy heuristic pathfinding with a **3-Layer Convolutional Neural Network (CNN)**.
  - Implemented high-performance inference using the Microsoft **ONNX Runtime** Go bindings.
  - Native C++ acceleration for matrix operations, achieving sub-millisecond computation time.
- **Strategic Intelligence**:
  - AI model trained via **Reinforcement Learning (DQN)** on human and bot game session data.
  - 6-channel grid representation for advanced spatial awareness (Player, AI, Food, Obstacles, Fireballs).
- **Safety Intercept Layer**:
  - Added a "Logical Safety Gate" that overrides AI neural network predictions if a collision is imminent, ensuring robust performance even in unstable model states.

### 🚀 High-Performance Inference Architecture
- **Centralized Inference Service**:
  - Transitioned from per-game model instances to a **Singleton Global Worker** pattern.
  - Eliminated memory overhead and resource contention across multiple concurrent users.
- **Task Queue & Channel Pipeline**:
  - Implemented an asynchronous **Inference Queue** using Go channels.
  - Dedicated worker thread processes requests sequentially, maximizing CPU cache locality and eliminating mutex locking overhead.
- **Latency Optimization**:
  - Reduced total end-to-end inference latency (Go -> Queue -> ONNX -> Results) to **~1.3ms**.
  - System throughput increased to **750+ inferences/sec**, capable of serving 30+ simultaneous AI sessions on a single core.

### 🛠️ Core Infrastructure
- **Training Pipeline**:
  - Added `ml/train.py` for PyTorch-based DQN training.
  - Optimized ONNX export with stable opset versions and static shape configurations.
- **Multi-User Stability**:
  - Thread-safe environment initialization for ONNX Runtime.
  - Improved WebServer response time by decoupling input handling from the 16ms physical game loop.

---

## [3.3.0] - 2026-01-16

### 📼 Game Recording System
- **Full Session Recording**:
  - Automatically captures every game session from start to finish.
  - Data stored in **JSONL** format (one JSON object per line) for easy parsing and ML training.
  - Records full game state, player actions, AI context, and rewards at every frame.
- **Data Engineering**:
  - Unique session IDs based on timestamp and connection UUID.
  - Asynchronous, non-blocking disk writing to ensure zero impact on game server performance.
  - Captures complex events: `Action`, `Reward` (sparse & dense), `AI Stun`, `Collision`.

### 📺 Replay & Visualization Tool
- **Dedicated Replay Server**:
  - New binary `replay` running on port **8081**.
  - Web interface to list and browse all recorded game sessions.
- **High-Fidelity Playback**:
  - Pixel-perfect recreation of the original game using the exact same frontend engine (`game.js`).
  - Supports full visual effects: particles, floating texts, animations.
  - Displays auxiliary AI data: **Step Count** and **AI Intent** (e.g., HUNT, ATTACK).
- **Control Features**:
  - Pause/Resume during playback.
  - Seek bar logic (implicit via step tracking).
  - Handles edge cases like varying canvas sizes and network configs via recorded `config` packets.

### 🐛 Fixes & Polish
- **Stability**:
  - Fixed binary ignore rules in `.gitignore` to prevent committing build artifacts.
  - Removed accidental binary commitments from history.
- **Replay UX**:
  - Solved "Black Screen" issue by correctly replaying `config` packets to initialization canvas.
  - Fixed Overlay display logic to ensure Game Over screens render correctly in replay mode (re-attached DOM elements).

---

## [3.2.0] - 2026-01-14

### 👺 Berserker AI (Aggressive Mode)

- **AI Auto-Boost**: AI competitor can now trigger 3x speed boost to snatch food, making it much more competitive.
- **Offensive Fireball AI**:
  - AI now actively targets and shoots at the player when in range.
  - AI uses fireballs to clear obstacles blocking its path.
- **Combat Balance**:
  - Implemented player penalties when hit by AI fireballs:
    - **Headshot**: -30 points and visual warning.
    - **Body Hit**: -10 points and length reduction (-1 segment).
- **Core Improvements**:
  - Refactored AI movement to support sub-frame iterations for true speed boosting.
  - Improved AI targeting range (up to 10 tiles).

---

## [3.1.0] - 2026-01-14

### � Game Modes & Features

- **🧘 Zen Mode**: Infinite practice mode with no time limit and no AI opponent
  - Perfect for learning controls and exploring the food system
  - Relaxed gameplay without competitive pressure
- **⚔️ Battle Mode**: Competitive 2-minute showdown against AI
  - Time-limited matches with score comparison
  - AI opponent with intelligent pathfinding
  - Fireball combat system for tactical gameplay
- **🎯 Fireball Combat System**:
  - Shoot projectiles to stun AI (headshot) or shrink its body
  - Destroy obstacles for bonus points
  - 300ms cooldown between shots
  - Visual explosion effects on impact
- **📈 Floating Score Bubbles**:
  - Animated score labels appear at exact point of earning
  - Glass-morphic capsule design with smooth animations
  - Contextual colors: blue (food), red (combat), gold (headshot)
  - Upward floating with fade-out effect

### �🏗️ Architecture

- **Frontend Modularization**: Refactored monolithic `game.js` (1100+ lines) into clean ES modules
  - Created `modules/audio.js` for sound management (150 lines)
  - Created `modules/renderer.js` for pure rendering logic (340 lines)
  - Main `game.js` now acts as orchestrator (470 lines)
  - Clear separation of concerns: rendering, audio, game logic
- **Message System Simplification**: 
  - Removed `MessageTime` and `MessageDuration` fields from backend
  - Simplified `SetMessage()` API (removed duration parameter)
  - Frontend now controls all message display timing
  - Single source of truth for message configuration
- **Dynamic Config Synchronization**:
  - Server sends game configuration on connection
  - Client syncs board size, duration, cooldowns dynamically
  - Eliminates hardcoded values in frontend

### ✨ Visual & UX Improvements

- **Optimized Message Display**:
  - Reduced font size (18px for normal, 16px for bonus)
  - Moved position higher (top 1/5 instead of 1/3)
  - Shortened display time (bonus: 800ms, normal: 1000ms)
  - Fixed fade-out animation timing (500ms smooth transition)
  - Less obstructive to gameplay
- **Unified Game-Over Experience**:
  - All end states use consistent overlay design
  - Victory states: 🏆 YOU WIN (gold), 🤖 AI WINS (purple), 🤝 DRAW (blue)
  - Crash state: ❌ GAME OVER (red)
  - Clear restart instructions on all overlays
  - Confetti effects visible behind semi-transparent overlay
- **Enhanced Confetti Effects**:
  - Multi-point burst system (3 simultaneous explosions)
  - Particle variety (circles, strips, squares)
  - Physics simulation (gravity, air resistance, rotation)
  - Two-stage effect (initial burst + delayed confetti rain from top)
  - Extended duration with slower decay
- **Food Timer Improvements**:
  - Countdown rings freeze correctly during pause/game-over
  - Pulsating animation in last 5 seconds
  - Visual clarity improvements

### 🐛 Bug Fixes

- Fixed food countdown animation continuing during pause/game-over
- Fixed message fade-out timing inconsistency (was 4000ms, now 1500ms)
- Fixed overlay layout vertical stacking with `flex-direction: column`
- Fixed CSS cache issues with version query parameters
- Eliminated duplicate `maxDuration` calculations in renderer
- Fixed confetti animation not updating (moved to `updateVisuals()`)
- Fixed overlay not showing for AI victory (now shows for all winners)

### 🎨 Code Quality

- Applied DRY principle to message timing logic
- Improved separation of concerns across modules
- Added comprehensive code documentation
- Updated unit tests for new simplified API
- Reduced main game.js from 1100 to 470 lines
- Created reusable, testable modules

### 🔄 Breaking Changes

- `SetMessage(msg, duration)` → `SetMessage(msg)` (duration removed)
- `SetMessageWithType(msg, duration, type)` → `SetMessageWithType(msg, type)`
- Removed `HasActiveMessage()` method (no longer needed)
- Removed `MessageTime` and `MessageDuration` from Game struct

---

## [3.0.0] - 2026-01-13

### 🚀 Added
- **Intelligent AI (Auto-Play Mode)**:
  - Implemented Flood Fill algorithm for spatial awareness and trap avoidance.
  - Added utility-based greedy logic to prioritize high-value foods (corners/edges).
  - Added time-awareness: AI calculates if it can reach food before expiry and auto-triggers Boost.
- **Web Audio System**: 
  - Real-time sound synthesis for move, eat, boost, and game over using Web Audio API (zero external assets).
- **Mobile Experience & Haptics**: 
  - Integrated Vibration API for Android devices (haptic feedback on turns, eating, and crashes).
  - Added "Long Press to Boost" for mobile touch controls.
- **Infrastructure**:
  - Organized documentation into a dedicated `docs/` directory.
  - Implemented IP-based connection limiting to prevent duplicate game instances.

### ⚡ Changed
- Refined mobile UI layout to prevent Cumulative Layout Shift (CLS) when Boost is active.
- Improved difficulty/auto panel to fit better on small screens.
- Standardized coordinate system and direction validation to prevent 180-degree self-collisions.

---

## [2.0.0] - 2026-01-09
### Added
- Web-based version with High Performance Canvas rendering.
- Glassmorphism UI design.
- WebSocket-based real-time state synchronization.

---

## [1.0.0] - 2026-01-07
### Added
- Original Go terminal implementation.
- Multiple food types and scoring.
