# 🎮 Release Notes - v3.1.0

**Release Date**: January 14, 2026

## 🎉 Major Updates

### 🏗️ Architecture Improvements

#### **Frontend Modularization**
- **ES Modules Migration**: Refactored monolithic `game.js` (1100+ lines) into clean, single-responsibility modules
  - `modules/audio.js`: Sound management system
  - `modules/renderer.js`: Pure rendering engine
  - Main `game.js`: Orchestrator for game logic and state management
- **Separation of Concerns**: Clear boundaries between rendering, audio, and game logic
- **Maintainability**: Easier to extend and debug with modular architecture

#### **Message System Simplification**
- **Backend Cleanup**: Removed duration management from server-side
  - Deleted `MessageTime` and `MessageDuration` fields
  - Simplified `SetMessage()` API
- **Frontend Control**: Unified message display timing in renderer
  - Bonus messages: 800ms display time
  - Normal messages: 1000ms display time
  - All messages: 500ms fade-out animation

### ✨ Visual & UX Enhancements

#### **Optimized Message Display**
- **Reduced Obstruction**: Messages are now smaller and positioned higher
  - Font size: 18px (down from 24px)
  - Position: Top 1/5 of screen (up from 1/3)
  - Tighter padding for compact appearance
- **Faster Feedback**: Shorter display times keep gameplay flowing
- **Smooth Animations**: Fixed fade-out bugs for consistent visual polish

#### **Unified Game Over Experience**
- **Consistent Overlays**: All game-end states use the same overlay design
  - 🏆 **YOU WIN!** (Gold) - Player victory
  - 🤖 **AI WINS!** (Purple) - AI victory
  - 🤝 **DRAW!** (Blue) - Tie game
  - ❌ **GAME OVER** (Red) - Crash/unexpected end
- **Victory Celebrations**: Confetti effects visible behind semi-transparent overlay
- **Clear Instructions**: "Tap or Press 'R' to Restart" on all end screens

#### **Enhanced Confetti System**
- **Multi-Point Bursts**: Fireworks explode from 3 locations simultaneously
- **Particle Variety**: Circles, strips, and squares for visual richness
- **Physics Simulation**: Gravity, air resistance, and rotation for realistic motion
- **Two-Stage Effect**: Initial burst + delayed "confetti rain" from top
- **Extended Duration**: Slower decay for longer celebration

#### **Food Timer Improvements**
- **Pause-Aware Animation**: Countdown rings freeze when game is paused/ended
- **Consistent Behavior**: No more "ghost animations" during game-over state

### 🐛 Bug Fixes

- Fixed food countdown animation continuing after game pause/end
- Fixed message fade-out timing inconsistency (was using 4000ms instead of 1500ms)
- Fixed overlay layout issues (vertical stacking now enforced with `flex-direction: column`)
- Fixed CSS cache issues by adding version query parameters
- Removed duplicate `maxDuration` calculations in renderer

### 🎨 Code Quality

- **DRY Principle**: Eliminated code duplication in message timing logic
- **Type Safety**: Consistent use of `messageType` parameter throughout
- **Comments**: Added clear documentation for complex logic
- **Testing**: Updated unit tests to match new simplified API

---

## 📊 Technical Details

### Breaking Changes
- `SetMessage(msg, duration)` → `SetMessage(msg)` (duration removed)
- `SetMessageWithType(msg, duration, type)` → `SetMessageWithType(msg, type)`
- Removed `HasActiveMessage()` method (no longer needed)

### Migration Guide
If you've forked this project:
1. Update all `SetMessage()` calls to remove duration parameter
2. Remove any references to `MessageTime` or `MessageDuration`
3. Message display timing is now controlled in `modules/renderer.js` line 271-272

---

## 🎯 What's Next?

### Planned Features
- Global leaderboard system
- Achievement/badge system
- Custom themes and skins
- Daily challenges
- Multiplayer mode

### Performance Goals
- Further optimize rendering pipeline
- Reduce memory footprint
- Improve mobile performance

---

## 🙏 Acknowledgments

Thanks to all players who provided feedback and bug reports!

---

## 📝 Full Changelog

### Added
- ES Module architecture for frontend
- Separate renderer module (`modules/renderer.js`)
- Separate audio module (`modules/audio.js`)
- Multi-stage confetti effects
- Particle variety (circles, strips, squares)
- Physics-based confetti motion
- Unified game-over overlay system

### Changed
- Message display duration (bonus: 800ms, normal: 1000ms)
- Message font size (18px for normal, 16px for bonus)
- Message position (top 1/5 instead of 1/3)
- Simplified backend message API
- Refactored frontend into modules

### Fixed
- Food timer animation during pause/game-over
- Message fade-out timing bug
- Overlay layout vertical stacking
- CSS cache issues
- Code duplication in renderer

### Removed
- `MessageTime` field from Game struct
- `MessageDuration` field from Game struct
- `HasActiveMessage()` method
- Duration parameter from `SetMessage()` methods

---

**Download**: [Latest Release](https://github.com/trytobebee/snake_go/releases/latest)
**Documentation**: [README.md](./README.md)
**Issues**: [GitHub Issues](https://github.com/trytobebee/snake_go/issues)
