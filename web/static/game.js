// Snake Game - Web Version
// WebSocket connection and rendering

class SoundManager {
    constructor() {
        this.ctx = null;
        this.enabled = true;
    }

    init() {
        if (!this.ctx) {
            this.ctx = new (window.AudioContext || window.webkitAudioContext)();
        }
    }

    playTone(freq, type, duration, volume = 0.1) {
        if (!this.enabled || !this.ctx) return;

        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();

        osc.type = type;
        osc.frequency.setValueAtTime(freq, this.ctx.currentTime);

        gain.gain.setValueAtTime(volume, this.ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.0001, this.ctx.currentTime + duration);

        osc.connect(gain);
        gain.connect(this.ctx.destination);

        osc.start();
        osc.stop(this.ctx.currentTime + duration);
    }

    playMove() {
        this.playTone(150, 'sine', 0.1, 0.05);
    }

    playEat(type) {
        // Different pitches for different foods
        const freqs = { 0: 440, 1: 554, 2: 659, 3: 880 };
        const f = freqs[type] || 440;
        this.playTone(f, 'triangle', 0.3, 0.15);
        setTimeout(() => this.playTone(f * 1.5, 'triangle', 0.2, 0.1), 50);
    }

    playCrash() {
        this.playTone(100, 'sawtooth', 0.5, 0.2);
        this.playTone(50, 'square', 0.8, 0.1);
    }

    playBoost(active) {
        if (active) {
            this.playTone(200, 'sine', 0.1, 0.03);
        }
    }

    toggle() {
        this.enabled = !this.enabled;
        return this.enabled;
    }
}

class SnakeGameClient {
    constructor() {
        this.canvas = document.getElementById('gameCanvas');
        this.ctx = this.canvas.getContext('2d');
        this.ws = null;
        this.gameState = null;
        this.sounds = new SoundManager();

        // Canvas settings
        this.cellSize = 20;
        this.boardWidth = 25;
        this.boardHeight = 25;

        this.canvas.width = this.boardWidth * this.cellSize;
        this.canvas.height = this.boardHeight * this.cellSize;

        // Sound init on first interaction (browser requirement)
        const initAudio = () => {
            this.sounds.init();
            document.removeEventListener('click', initAudio);
            document.removeEventListener('keydown', initAudio);
        };
        document.addEventListener('click', initAudio);
        document.addEventListener('keydown', initAudio);

        // UI elements
        this.scoreEl = document.getElementById('score');
        this.bestScoreEl = document.getElementById('bestScore');
        this.speedEl = document.getElementById('speed');
        this.eatenEl = document.getElementById('eaten');
        this.boostIndicator = document.getElementById('boostIndicator');
        this.gameOverlay = document.getElementById('gameOverlay');
        this.overlayTitle = document.getElementById('overlayTitle');
        this.overlayMessage = document.getElementById('overlayMessage');
        this.connectionStatus = document.getElementById('connectionStatus');

        // High score persistence with safety check
        this.bestScore = 0;
        try {
            const savedScore = localStorage.getItem('snake_best_score');
            if (savedScore) {
                this.bestScore = parseInt(savedScore) || 0;
            }
        } catch (e) {
            console.warn('LocalStorage access denied:', e);
        }
        this.bestScoreEl.textContent = this.bestScore;

        this.setupWebSocket();
        this.setupKeyboard();
        this.setupMobileControls();
        this.setupDifficulty();
        this.setupAutoPlay();

        // Message state for Canvas
        this.currentMessage = '';
        this.messageAlpha = 0;
        this.messageStartTime = 0;
        this.lastMessage = '';
        this.messageTimeout = null;
        this.lastGameOver = false;
        this.previousScore = 0;
        this.previousEaten = 0;

        // Start animation loop once
        this.startAnimationLoop();
    }

    // Helper for haptic feedback
    triggerHaptic(pattern = 10) {
        if ('vibrate' in navigator) {
            try {
                navigator.vibrate(pattern);
            } catch (e) {
                // Ignore vibration errors as they are non-critical
            }
        }
    }

    startAnimationLoop() {
        const frame = () => {
            this.render();
            requestAnimationFrame(frame);
        };
        requestAnimationFrame(frame);
    }

    setupMobileControls() {
        const buttons = {
            'btn-up': 'up',
            'btn-down': 'down',
            'btn-left': 'left',
            'btn-right': 'right',
            'btn-pause': 'pause'
        };

        this.pressTimer = null;

        Object.entries(buttons).forEach(([id, action]) => {
            const btn = document.getElementById(id);
            if (!btn) return;

            const startAction = (e) => {
                e.preventDefault();
                if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

                // Initial move
                this.ws.send(JSON.stringify({ action }));
                this.triggerHaptic(15);
                this.sounds.playMove();

                // Setup repeat for Boost
                if (action !== 'pause') {
                    if (this.pressTimer) clearInterval(this.pressTimer);
                    this.pressTimer = setInterval(() => {
                        this.ws.send(JSON.stringify({ action }));
                        // Subtle haptic for boosting
                        if (this.gameState && this.gameState.boosting) {
                            this.triggerHaptic(5);
                        }
                    }, 80); // Send every 80ms while holding
                }
            };

            const endAction = (e) => {
                e.preventDefault();
                if (this.pressTimer) {
                    clearInterval(this.pressTimer);
                    this.pressTimer = null;
                }
            };

            btn.addEventListener('touchstart', startAction);
            btn.addEventListener('touchend', endAction);
            btn.addEventListener('touchcancel', endAction);

            btn.addEventListener('mousedown', startAction);
            btn.addEventListener('mouseup', endAction);
            btn.addEventListener('mouseleave', endAction);
        });

        // Restart support via overlay tap/touch
        const handleStartRestart = () => {
            if (this.gameState && this.gameState.gameOver) {
                this.ws.send(JSON.stringify({ action: 'restart' }));
                this.triggerHaptic([30, 50, 30]); // Distinct pattern for restart
            } else if (this.gameState && !this.gameState.started) {
                this.ws.send(JSON.stringify({ action: 'start' }));
                this.triggerHaptic(20);
            }
        };

        this.gameOverlay.addEventListener('click', handleStartRestart);
        this.gameOverlay.addEventListener('touchstart', (e) => {
            e.preventDefault();
            handleStartRestart();
        });
    }

    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(wsURL);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus('connected');
        };

        this.ws.onmessage = (event) => {
            const newState = JSON.parse(event.data);
            // Only update UI if state changed enough or periodically
            this.gameState = newState;
            this.updateUI();
            // REMOVED: this.render() is now called by the animation loop
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus('disconnected');
        };

        this.ws.onclose = () => {
            console.log('WebSocket closed');
            this.updateConnectionStatus('disconnected');
            setTimeout(() => this.setupWebSocket(), 3000);
        };
    }

    setupKeyboard() {
        document.addEventListener('keydown', (e) => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

            const key = e.key.toLowerCase();
            let action = '';

            // Direction keys
            if (key === 'arrowup' || key === 'w') action = 'up';
            else if (key === 'arrowdown' || key === 's') action = 'down';
            else if (key === 'arrowleft') action = 'left'; // Removed 'a' to allow auto-play toggle
            else if (key === 'arrowright' || key === 'd') action = 'right';
            else if (key === ' ' || key === 'p') action = 'pause';
            else if (key === 'r') action = 'restart';
            else if (key === 'q') action = 'quit';
            else if (key === 'a') action = 'auto'; // 'a' is for auto-play

            if (action) {
                e.preventDefault();
                this.ws.send(JSON.stringify({ action }));

                // Play sounds/haptics for actions
                if (['up', 'down', 'left', 'right'].includes(action)) {
                    this.sounds.playMove();
                    this.triggerHaptic(15);
                } else if (action === 'auto') {
                    this.sounds.playEat(0); // Slight feedback for toggle
                    this.triggerHaptic(25);
                }
            }
        });
    }

    setupDifficulty() {
        ['low', 'mid', 'high'].forEach(d => {
            const btn = document.getElementById(`diff-${d}`);
            if (btn) {
                btn.addEventListener('click', () => {
                    if (this.ws && (this.gameState && (!this.gameState.started || this.gameState.gameOver))) {
                        this.ws.send(JSON.stringify({ action: `diff_${d}` }));
                        this.triggerHaptic(20);
                    } else if (this.gameState && this.gameState.started && !this.gameState.gameOver) {
                        this.showTempMessage("Can't change difficulty during game!");
                        this.triggerHaptic([50, 100, 50]); // Error vibration
                    }
                });
            }
        });
    }

    setupAutoPlay() {
        const btn = document.getElementById('btn-auto');
        if (btn) {
            btn.addEventListener('click', () => {
                if (this.ws && (this.ws.readyState === WebSocket.OPEN)) {
                    this.ws.send(JSON.stringify({ action: 'auto' }));
                    this.triggerHaptic(25);
                }
            });
        }
    }

    showTempMessage(msg) {
        if (!msg) return;
        this.currentMessage = msg;
        this.messageAlpha = 1.0;
        this.messageStartTime = Date.now();

        // Note: We no longer clear this.lastMessage here.
        // this.lastMessage acts as a buffer for what we've already reacted to.

        if (this.messageTimeout) {
            clearTimeout(this.messageTimeout);
        }

        // Logic for canvas rendering duration (1.5s total with fade)
        this.messageTimeout = setTimeout(() => {
            this.currentMessage = '';
            // We DON'T clear this.lastMessage here to prevent re-triggering 
            // the same message still being broadcast by the server.
        }, 1500);
    }

    updateUI() {
        if (!this.gameState) return;

        // 1. Update Current Stats
        const currentScore = this.gameState.score || 0;
        const currentEaten = this.gameState.foodEaten || 0;

        // Haptic on eating food
        if (currentEaten > this.previousEaten) {
            this.triggerHaptic(20);
            // Play eat sound for the last food eaten
            const lastFoodType = this.gameState.foods && this.gameState.foods.length > 0 ?
                this.gameState.foods[0].foodType : 0;
            this.sounds.playEat(lastFoodType);
        }
        this.previousEaten = currentEaten;
        this.previousScore = currentScore;

        this.scoreEl.textContent = currentScore;
        this.speedEl.textContent = (this.gameState.eatingSpeed || 0).toFixed(2);
        this.eatenEl.textContent = this.gameState.foodEaten || 0;

        // 2. High Score Logic
        if (currentScore > this.bestScore) {
            this.bestScore = currentScore;
            this.bestScoreEl.textContent = this.bestScore;
            try {
                localStorage.setItem('snake_best_score', this.bestScore);
            } catch (e) {
                console.error('Failed to save high score:', e);
            }
            this.bestScoreEl.parentElement.classList.add('new-record');
        } else {
            this.bestScoreEl.parentElement.classList.remove('new-record');
        }

        // 3. Game Over Special Message
        if (this.gameState.gameOver && !this.lastGameOver) {
            this.triggerHaptic([100, 50, 100]); // Heavy vibration on crash
            this.sounds.playCrash();
            if (currentScore >= this.bestScore && currentScore > 0) {
                this.showTempMessage("🎊 NEW HIGH SCORE! 🎊");
            }
        }
        this.lastGameOver = this.gameState.gameOver;

        // 4. Boost indicator
        if (this.gameState.boosting) {
            this.boostIndicator.classList.add('active');
            this.sounds.playBoost(true);
        } else {
            this.boostIndicator.classList.remove('active');
        }

        // 4.5 Update difficulty labels
        const diffs = ['low', 'mid', 'high'];
        diffs.forEach(d => {
            const btn = document.getElementById(`diff-${d}`);
            if (btn) {
                if (this.gameState.difficulty === d) {
                    btn.classList.add('active');
                } else {
                    btn.classList.remove('active');
                }
            }
        });

        // 4.6 Update auto-play button
        const autoBtn = document.getElementById('btn-auto');
        if (autoBtn) {
            if (this.gameState.autoPlay) {
                autoBtn.classList.add('active');
                autoBtn.innerHTML = '🤖 Auto: ON';
            } else {
                autoBtn.classList.remove('active');
                autoBtn.innerHTML = '👤 Auto: OFF';
            }
        }

        // 5. Intercept server messages for Canvas display
        const currentMsg = this.gameState.message || '';
        if (currentMsg && currentMsg !== this.lastMessage) {
            this.lastMessage = currentMsg; // Update seen message
            this.showTempMessage(currentMsg);
        } else if (!currentMsg) {
            this.lastMessage = '';
        }

        // 6. Game overlay
        if (!this.gameState.started) {
            const msg = window.innerWidth <= 768 ? 'Tap to start' : 'Press SPACE to start';
            this.showOverlay('🐍 READY?', msg);
        } else if (this.gameState.gameOver) {
            const msg = window.innerWidth <= 768 ? 'Tap to restart' : 'Press R to restart';
            this.showOverlay('💀 GAME OVER!', msg);
        } else if (this.gameState.paused) {
            const msg = window.innerWidth <= 768 ? 'Tap to resume' : 'Press P to continue';
            this.showOverlay('⏸️ PAUSED', msg);
        } else {
            this.hideOverlay();
        }
    }

    showOverlay(title, message) {
        this.overlayTitle.textContent = title;
        this.overlayMessage.textContent = message;
        this.gameOverlay.classList.add('show');
    }

    hideOverlay() {
        this.gameOverlay.classList.remove('show');
    }

    updateConnectionStatus(status) {
        this.connectionStatus.className = `connection-status ${status}`;
        const statusText = this.connectionStatus.querySelector('.status-text');

        if (status === 'connected') {
            statusText.textContent = 'Connected';
        } else if (status === 'disconnected') {
            statusText.textContent = 'Disconnected';
        } else {
            statusText.textContent = 'Connecting...';
        }
    }

    render() {
        // Clear canvas
        this.ctx.fillStyle = '#1a1a2e';
        this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);

        if (!this.gameState) {
            // Show waiting message
            this.ctx.fillStyle = '#fff';
            this.ctx.font = '20px Inter, sans-serif';
            this.ctx.textAlign = 'center';
            this.ctx.fillText('Connecting to server...', this.canvas.width / 2, this.canvas.height / 2);
            return;
        }

        // Draw walls
        this.ctx.fillStyle = '#4a5568';
        for (let x = 0; x < this.boardWidth; x++) {
            this.drawCell(x, 0);
            this.drawCell(x, this.boardHeight - 1);
        }
        for (let y = 0; y < this.boardHeight; y++) {
            this.drawCell(0, y);
            this.drawCell(this.boardWidth - 1, y);
        }

        // Draw foods
        if (this.gameState.foods) {
            this.gameState.foods.forEach(food => {
                this.drawFood(food);
            });
        }

        // Draw snake
        if (this.gameState.snake) {
            this.gameState.snake.forEach((segment, index) => {
                if (index === 0) {
                    // Head
                    this.ctx.fillStyle = '#48bb78';
                    this.drawCell(segment.x, segment.y);
                    // Add eyes
                    this.ctx.fillStyle = '#000';
                    const centerX = segment.x * this.cellSize + this.cellSize / 2;
                    const centerY = segment.y * this.cellSize + this.cellSize / 2;
                    this.ctx.beginPath();
                    this.ctx.arc(centerX - 4, centerY - 4, 2, 0, Math.PI * 2);
                    this.ctx.arc(centerX + 4, centerY - 4, 2, 0, Math.PI * 2);
                    this.ctx.fill();
                } else {
                    // Body
                    this.ctx.fillStyle = '#68d391';
                    this.drawCell(segment.x, segment.y);
                }
            });
        }

        if (this.gameState.gameOver && this.gameState.crashPoint) {
            this.ctx.font = `${this.cellSize}px sans-serif`;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'middle';
            this.ctx.fillText('💥',
                this.gameState.crashPoint.x * this.cellSize + this.cellSize / 2,
                this.gameState.crashPoint.y * this.cellSize + this.cellSize / 2
            );
        }

        // --- Render Canvas Message ---
        this.drawCanvasMessage();
    }

    drawCanvasMessage() {
        if (!this.currentMessage) return;

        const now = Date.now();
        const age = now - this.messageStartTime;

        // Settings: Show for 1.5s total (1s static + 0.5s fade/float)
        if (age > 1500) return;

        let displayAlpha = 1.0;
        let yOffset = 0;

        if (age > 1000) {
            // Fade and Float in last 500ms
            const fadeProgress = (age - 1000) / 500;
            displayAlpha = 1.0 - fadeProgress;
            yOffset = -fadeProgress * 30; // Float up by 30px
        }

        const x = this.canvas.width / 2;
        const y = this.canvas.height / 3 + yOffset; // Position in upper third

        this.ctx.save();
        this.ctx.globalAlpha = displayAlpha;

        // Draw pill background
        this.ctx.font = 'bold 20px Inter, sans-serif';
        const padding = 20;
        const metrics = this.ctx.measureText(this.currentMessage);
        const rectWidth = metrics.width + padding * 2;
        const rectHeight = 44;

        // Fill
        this.ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
        this.ctx.beginPath();
        if (this.ctx.roundRect) {
            this.ctx.roundRect(x - rectWidth / 2, y - rectHeight / 2, rectWidth, rectHeight, 22);
        } else {
            this.ctx.rect(x - rectWidth / 2, y - rectHeight / 2, rectWidth, rectHeight);
        }
        this.ctx.fill();

        // Glow/Border
        this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.4)';
        this.ctx.lineWidth = 2;
        this.ctx.stroke();

        // Text
        this.ctx.fillStyle = '#ffffff';
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';
        this.ctx.shadowBlur = 4;
        this.ctx.shadowColor = 'rgba(0,0,0,0.5)';
        this.ctx.fillText(this.currentMessage, x, y);

        this.ctx.restore();
    }

    drawCell(x, y) {
        this.ctx.fillRect(
            x * this.cellSize + 1,
            y * this.cellSize + 1,
            this.cellSize - 2,
            this.cellSize - 2
        );
    }

    drawFood(food) {
        this.ctx.save();
        const centerX = food.pos.x * this.cellSize + this.cellSize / 2;
        const centerY = food.pos.y * this.cellSize + this.cellSize / 2;

        const emojis = {
            0: '🟣',
            1: '🔵',
            2: '🟠',
            3: '🔴'
        };

        // 1. Calculate Pulsating Effect
        let scale = 1.0;
        if (food.remainingSeconds > 0 && food.remainingSeconds <= 5) {
            // Speed up pulse as time runs out (from 2Hz to 5Hz)
            const frequency = 5 - (food.remainingSeconds - 1) * 0.5;
            scale = 1 + 0.15 * Math.sin(Date.now() * 0.005 * frequency);
        }

        // 2. Draw Food Emoji
        this.ctx.font = `${(this.cellSize - 4) * scale}px sans-serif`;
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';
        this.ctx.fillText(
            emojis[food.foodType] || '🟣',
            centerX,
            centerY
        );

        // 3. Draw Premium Radial Progress Ring
        if (food.remainingSeconds > 0 && food.remainingSeconds <= 5) {
            const radius = this.cellSize / 2 - 1;
            const startAngle = -Math.PI / 2; // Top
            const progress = food.remainingSeconds / 5;
            const endAngle = startAngle + (progress * Math.PI * 2);

            // Draw Background Track (Subtle white)
            this.ctx.beginPath();
            this.ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.15)';
            this.ctx.lineWidth = 2;
            this.ctx.stroke();

            // Draw Progress Arc (Glowing white)
            this.ctx.beginPath();
            this.ctx.arc(centerX, centerY, radius, startAngle, endAngle);
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.8)';
            this.ctx.lineWidth = 2;
            this.ctx.lineCap = 'round';

            // Add subtle glow
            this.ctx.shadowBlur = 6;
            this.ctx.shadowColor = '#fff';
            this.ctx.stroke();
        }
        this.ctx.restore();
    }
}

// Initialize game when page loads
document.addEventListener('DOMContentLoaded', () => {
    new SnakeGameClient();
});
