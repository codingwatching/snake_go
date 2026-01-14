# Changelog

All notable changes to this project will be documented in this file.

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
