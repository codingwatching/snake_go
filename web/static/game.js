import { SoundManager } from './modules/audio.js';
import { GameRenderer } from './modules/renderer.js';

class SnakeGameClient {
    constructor() {
        this.canvas = document.getElementById('gameCanvas');
        this.ctx = this.canvas.getContext('2d');
        this.ws = null;
        this.gameState = null;
        this.sounds = new SoundManager();

        // Constants & Config
        this.cellSize = 20;
        this.boardWidth = 25;
        this.boardHeight = 25;
        this.gameDuration = 30;
        this.fireCooldown = 300;

        // Initialize Renderer
        this.renderer = new GameRenderer(this.canvas, this.ctx, this.cellSize);

        // Sound init on first interaction
        const initAudio = () => {
            this.sounds.init();
            document.removeEventListener('click', initAudio);
            document.removeEventListener('keydown', initAudio);
        };
        document.addEventListener('click', initAudio);
        document.addEventListener('keydown', initAudio);

        // UI elements
        this.initUIElements();

        // High score
        this.bestScore = 0;
        this.loadBestScore();

        // State for animations
        this.currentMessage = '';
        this.messageType = 'normal';
        this.messageStartTime = 0;
        this.lastGameOver = false;
        this.previousScore = 0;
        this.previousEaten = 0;
        this.previousAIScore = 0;
        this.lastAIStunned = false;
        this.explosions = [];
        this.confetti = [];
        this.floatingScores = [];
        this.lastFireTime = 0;

        // Start systems
        this.setupWebSocket();
        this.setupKeyboard();
        this.setupMobileControls();
        this.setupDifficulty();
        this.setupAutoPlay();
        this.setupMode();
        this.startAnimationLoop();
    }

    initUIElements() {
        this.scoreEl = document.getElementById('score');
        this.bestScoreEl = document.getElementById('bestScore');
        this.speedEl = document.getElementById('speed');
        this.eatenEl = document.getElementById('eaten');
        this.aiScoreEl = document.getElementById('aiScore');
        this.boostIndicator = document.getElementById('boostIndicator');
        this.gameOverlay = document.getElementById('gameOverlay');
        this.overlayTitle = document.getElementById('overlayTitle');
        this.overlayMessage = document.getElementById('overlayMessage');
        this.connectionStatus = document.getElementById('connectionStatus');
        this.timeLeftEl = document.getElementById('timeLeft');
        this.aiStatEl = document.querySelector('.ai-stat');
        this.timerEl = document.querySelector('.timer');
    }

    loadBestScore() {
        try {
            const savedScore = localStorage.getItem('snake_best_score');
            if (savedScore) {
                this.bestScore = parseInt(savedScore) || 0;
                this.bestScoreEl.textContent = this.bestScore;
            }
        } catch (e) {
            console.warn('LocalStorage error:', e);
        }
    }

    triggerHaptic(pattern = 10) {
        if ('vibrate' in navigator) {
            try { navigator.vibrate(pattern); } catch (e) { }
        }
    }

    fire() {
        const now = Date.now();
        if (now - this.lastFireTime < this.fireCooldown) return;
        this.ws.send(JSON.stringify({ action: 'fire' }));
        this.sounds.playFire('cannon');
        this.triggerHaptic(20);
        this.lastFireTime = now;
    }

    startAnimationLoop() {
        const frame = () => {
            this.updateVisuals();
            this.renderer.render(
                this.gameState,
                this.boardWidth,
                this.boardHeight,
                this.explosions,
                this.confetti,
                this.floatingScores,
                this.currentMessage,
                this.messageStartTime,
                this.messageType
            );
            requestAnimationFrame(frame);
        };
        requestAnimationFrame(frame);
    }

    updateVisuals() {
        const now = Date.now();

        // 1. Filter and Update Explosions
        this.explosions = this.explosions.filter(exp => now - exp.startTime < exp.duration);

        // 2. Update and Filter Floating Scores
        this.floatingScores = this.floatingScores.filter(s => s.life > 0);
        this.floatingScores.forEach(s => {
            s.y -= 0.8; // Float up
            s.life -= 0.015; // Fade out
        });

        // 3. Update and Filter Confetti
        this.confetti = this.confetti.filter(p => p.life > 0);
        this.confetti.forEach(p => {
            p.x += p.vx;
            p.y += p.vy;
            p.vy += 0.15; // Gravity
            p.vx *= 0.99; // Air resistance
            p.rotation += p.spin;
            p.life -= p.decay;
        });
    }

    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

        this.ws.onopen = () => this.updateConnectionStatus('connected');
        this.ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (msg.type === 'config') {
                this.handleConfig(msg.config);
            } else if (msg.type === 'state') {
                this.gameState = msg.state;
                this.updateUI();
            }
        };
        this.ws.onerror = () => this.updateConnectionStatus('disconnected');
        this.ws.onclose = () => {
            this.updateConnectionStatus('disconnected');
            setTimeout(() => this.setupWebSocket(), 3000);
        };
    }

    handleConfig(config) {
        if (!config) return;
        this.boardWidth = config.width;
        this.boardHeight = config.height;
        this.gameDuration = config.gameDuration;
        this.fireCooldown = config.fireballCooldown || 300;
        this.canvas.width = this.boardWidth * this.cellSize;
        this.canvas.height = this.boardHeight * this.cellSize;
    }

    updateUI() {
        if (!this.gameState) return;

        const currentScore = this.gameState.score || 0;
        const currentEaten = this.gameState.foodEaten || 0;

        // Sound/Haptic feedback
        if (currentEaten > this.previousEaten) {
            this.triggerHaptic(20);
            const lastType = this.gameState.foods?.[0]?.foodType || 0;
            this.sounds.playEat(lastType);
        }
        this.previousEaten = currentEaten;

        const currentAIScore = this.gameState.aiScore || 0;
        if (currentAIScore > this.previousAIScore) this.sounds.playAIConsume();
        this.previousAIScore = currentAIScore;

        if (this.gameState.aiStunned && !this.lastAIStunned) {
            this.sounds.playAIStun();
            this.triggerHaptic([30, 20, 30]);
        }
        this.lastAIStunned = this.gameState.aiStunned;

        if (this.gameState.playerStunned && !this.lastPlayerStunned) {
            this.triggerHaptic([50, 50, 50]);
            // You can add a specific hurt sound here if available
        }
        this.lastPlayerStunned = this.gameState.playerStunned;

        // Update elements
        this.scoreEl.textContent = currentScore;
        this.speedEl.textContent = (this.gameState.eatingSpeed || 0).toFixed(2);
        this.eatenEl.textContent = currentEaten;

        const timeRemaining = this.gameState.timeRemaining ?? this.gameDuration;
        this.timeLeftEl.textContent = timeRemaining;
        this.timeLeftEl.parentElement.classList.toggle('low-time', timeRemaining <= 10);

        if (this.gameState.mode === 'zen') {
            this.aiStatEl.style.display = 'none';
            this.timerEl.style.display = 'none';
        } else {
            this.aiStatEl.style.display = 'flex';
            this.timerEl.style.display = 'flex';
            this.aiScoreEl.textContent = this.gameState.aiScore || 0;
        }

        this.processEvents();
        this.updateOverlay();
        this.checkGameOver(currentScore);

        // Update Difficulty buttons state
        const currentDiff = this.gameState.difficulty || 'mid';
        ['low', 'mid', 'high'].forEach(d => {
            const btn = document.getElementById(`diff-${d}`);
            if (btn) btn.classList.toggle('active', currentDiff === d);
        });

        // Update Mode buttons state
        const currentMode = this.gameState.mode || 'battle';
        ['battle', 'zen'].forEach(m => {
            const btn = document.getElementById(`mode-${m}`);
            if (btn) btn.classList.toggle('active', currentMode === m);
        });

        // Update Berserker toggle UI
        const berserkerToggle = document.getElementById('berserker-toggle');
        if (berserkerToggle) {
            berserkerToggle.classList.toggle('active', !!this.gameState.berserker);
        }

        // Update Auto-Play button state
        const autoBtn = document.getElementById('btn-auto');
        if (autoBtn) {
            autoBtn.classList.toggle('active', !!this.gameState.autoPlay);
        }
    }

    updateOverlay() {
        if (!this.gameState) return;

        const isGameOver = this.gameState.gameOver;
        const isStarted = this.gameState.started;
        const isPaused = this.gameState.paused;

        if (isGameOver) {
            this.gameOverlay.style.display = 'flex';

            // If there's a winner (normal game end by time)
            if (this.gameState.winner) {
                if (this.gameState.winner === 'player') {
                    this.overlayTitle.textContent = '🏆 YOU WIN!';
                    this.overlayTitle.style.color = '#f6e05e';
                } else if (this.gameState.winner === 'ai') {
                    this.overlayTitle.textContent = '🤖 AI WINS!';
                    this.overlayTitle.style.color = '#9f7aea';
                } else {
                    this.overlayTitle.textContent = '🤝 DRAW!';
                    this.overlayTitle.style.color = '#4299e1';
                }
                this.overlayMessage.innerHTML = `<span class="tap-hint">Tap or Press 'R' to Restart</span>`;
            } else {
                // Crash/unexpected game over
                this.overlayTitle.textContent = 'GAME OVER';
                this.overlayTitle.style.color = '#f56565';
                this.overlayMessage.innerHTML = `<span class="tap-hint">Tap or Press 'R' to Restart</span>`;
            }
        } else if (!isStarted) {
            this.gameOverlay.style.display = 'flex';
            this.overlayTitle.textContent = 'READY?';
            this.overlayTitle.style.color = '#48bb78';
            this.overlayMessage.innerHTML = 'Press any direction to Start';
        } else if (isPaused) {
            this.gameOverlay.style.display = 'flex';
            this.overlayTitle.textContent = 'PAUSED';
            this.overlayTitle.style.color = '#4299e1';
            this.overlayMessage.innerHTML = 'Press P or Space to Resume';
        } else {
            this.gameOverlay.style.display = 'none';
        }
        this.gameOverlay.style.flexDirection = 'column';
    }

    processEvents() {
        // Floating scores
        if (this.gameState.scoreEvents) {
            this.gameState.scoreEvents.forEach(ev => {
                this.floatingScores.push({
                    x: ev.pos.x * this.cellSize + this.cellSize / 2,
                    y: ev.pos.y * this.cellSize,
                    text: ev.label,
                    life: 1.0,
                    color: ev.label.includes('HEADSHOT') ? '#f6e05e' : (ev.label.includes('HIT') ? '#fc8181' : '#63b3ed')
                });
            });
        }

        // Explosions
        if (this.gameState.hitPoints) {
            this.gameState.hitPoints.forEach(hp => {
                this.explosions.push({ x: hp.x, y: hp.y, startTime: Date.now(), duration: 500 });
            });
            if (this.gameState.hitPoints.length > 0) {
                this.sounds.playExplosion();
                this.triggerHaptic(30);
            }
        }

        // Server Messages
        if (this.gameState.message && this.gameState.message !== this.lastMessage) {
            this.messageType = this.gameState.messageType || 'normal';
            this.showTempMessage(this.gameState.message);
            this.lastMessage = this.gameState.message;
        } else if (!this.gameState.message) {
            this.lastMessage = '';
        }
    }

    checkGameOver(currentScore) {
        if (this.gameState.gameOver && !this.lastGameOver) {
            this.triggerHaptic([100, 50, 100]);
            if (this.gameState.winner) {
                this.sounds.playWin();
                this.createConfetti();
            } else {
                this.sounds.playCrash();
            }
            if (currentScore > this.bestScore) {
                this.bestScore = currentScore;
                this.bestScoreEl.textContent = this.bestScore;
                localStorage.setItem('snake_best_score', this.bestScore);
                this.bestScoreEl.parentElement.classList.add('new-record');
            }
        }
        this.lastGameOver = this.gameState.gameOver;
        this.bestScoreEl.parentElement.classList.toggle('new-record', currentScore >= this.bestScore && currentScore > 0);
    }

    createConfetti() {
        const colors = ['#f6e05e', '#f56565', '#4299e1', '#48bb78', '#ed64a6', '#9f7aea', '#ffffff', '#ffd700'];
        const types = ['circle', 'strip', 'square'];

        // Initial big bursts from 3 locations
        const locations = [
            { x: this.canvas.width / 2, y: this.canvas.height / 2 },
            { x: this.canvas.width / 4, y: this.canvas.height / 3 },
            { x: (this.canvas.width * 3) / 4, y: this.canvas.height / 3 }
        ];

        locations.forEach(loc => {
            for (let i = 0; i < 80; i++) {
                const angle = Math.random() * Math.PI * 2;
                const velocity = Math.random() * 12 + 5;
                this.confetti.push({
                    x: loc.x, y: loc.y,
                    vx: Math.cos(angle) * velocity,
                    vy: Math.sin(angle) * velocity - 3,
                    color: colors[Math.floor(Math.random() * colors.length)],
                    size: Math.random() * 5 + 3,
                    type: types[Math.floor(Math.random() * types.length)],
                    rotation: Math.random() * Math.PI * 2,
                    spin: (Math.random() - 0.5) * 0.2,
                    life: 1.0,
                    decay: Math.random() * 0.01 + 0.005 // Slower decay for longer duration
                });
            }
        });

        // Add a second wave after 500ms
        setTimeout(() => {
            if (!this.gameState?.gameOver) return;
            for (let i = 0; i < 100; i++) {
                this.confetti.push({
                    x: Math.random() * this.canvas.width, y: -20,
                    vx: (Math.random() - 0.5) * 4,
                    vy: Math.random() * 5 + 2,
                    color: colors[Math.floor(Math.random() * colors.length)],
                    size: Math.random() * 4 + 2,
                    type: types[Math.floor(Math.random() * types.length)],
                    rotation: Math.random() * Math.PI * 2,
                    spin: (Math.random() - 0.5) * 0.1,
                    life: 1.0,
                    decay: Math.random() * 0.008 + 0.004
                });
            }
        }, 800);
    }

    setupKeyboard() {
        document.addEventListener('keydown', (e) => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
            const key = e.key.toLowerCase();
            const actionMap = {
                'arrowup': 'up', 'w': 'up', 'arrowdown': 'down', 's': 'down',
                'arrowleft': 'left', 'arrowright': 'right', 'd': 'right',
                ' ': 'pause', 'p': 'pause', 'r': 'restart', 'q': 'quit', 'a': 'auto'
            };
            const action = actionMap[key];
            if (key === 'f' || key === 'enter') { this.fire(); return; }
            if (action) {
                e.preventDefault();
                this.ws.send(JSON.stringify({ action }));
                if (['up', 'down', 'left', 'right'].includes(action)) {
                    this.sounds.playMove();
                    this.triggerHaptic(15);
                }
            }
        });
    }

    setupMobileControls() {
        const buttons = { 'btn-up': 'up', 'btn-down': 'down', 'btn-left': 'left', 'btn-right': 'right', 'btn-pause': 'pause' };
        this.pressTimer = null;
        Object.entries(buttons).forEach(([id, action]) => {
            const btn = document.getElementById(id);
            if (!btn) return;
            const start = (e) => {
                e.preventDefault();
                this.ws.send(JSON.stringify({ action }));
                this.sounds.playMove();
                if (action !== 'pause') {
                    this.pressTimer = setInterval(() => this.ws.send(JSON.stringify({ action })), 80);
                }
            };
            const end = () => { if (this.pressTimer) { clearInterval(this.pressTimer); this.pressTimer = null; } };
            btn.addEventListener('touchstart', start); btn.addEventListener('touchend', end);
            btn.addEventListener('mousedown', start); btn.addEventListener('mouseup', end);
        });
        document.getElementById('btn-fire')?.addEventListener('click', () => this.fire());

        const handleStartRestart = () => {
            if (this.gameState?.gameOver) this.ws.send(JSON.stringify({ action: 'restart' }));
            else if (this.gameState && !this.gameState.started) this.ws.send(JSON.stringify({ action: 'start' }));
        };
        this.gameOverlay.addEventListener('click', handleStartRestart);
        this.canvas.addEventListener('click', handleStartRestart);
    }

    setupDifficulty() {
        ['low', 'mid', 'high'].forEach(d => {
            document.getElementById(`diff-${d}`)?.addEventListener('click', () => {
                if (!this.gameState?.started || this.gameState?.gameOver) {
                    this.ws.send(JSON.stringify({ action: `diff_${d}` }));
                } else {
                    this.showTempMessage("Can't change difficulty during game!");
                }
            });
        });

        // AI Toggle
        document.getElementById('ai-btn')?.addEventListener('click', () => {
            this.ws.send(JSON.stringify({ action: 'auto' }));
        });

        // Berserker Toggle
        const berserkerBtn = document.getElementById('berserker-toggle');
        berserkerBtn?.addEventListener('click', () => {
            console.log("Berserker toggle clicked");
            this.ws.send(JSON.stringify({ action: 'toggleBerserker' }));
        });
    }

    setupAutoPlay() { document.getElementById('btn-auto')?.addEventListener('click', () => this.ws.send(JSON.stringify({ action: 'auto' }))); }

    setupMode() {
        const setMode = (mode) => {
            this.ws.send(JSON.stringify({ action: `mode_${mode}` }));
            this.showTempMessage(`${mode.charAt(0).toUpperCase() + mode.slice(1)} Mode Activated`);
        };
        document.getElementById('mode-battle').onclick = () => setMode('battle');
        document.getElementById('mode-zen').onclick = () => setMode('zen');
    }

    showTempMessage(msg) {
        this.currentMessage = msg;
        this.messageStartTime = Date.now();
        if (this.messageTimeout) clearTimeout(this.messageTimeout);
        this.messageTimeout = setTimeout(() => {
            if (this.currentMessage === msg) this.currentMessage = '';
        }, 1500); // 1.5 seconds duration
    }

    updateConnectionStatus(status) {
        const el = document.getElementById('connectionStatus');
        if (!el) return;
        el.querySelector('.status-text').textContent = status === 'connected' ? 'Connected' : (status === 'disconnected' ? 'Disconnected' : 'Connecting...');
        el.classList.toggle('connected', status === 'connected');
    }
}

document.addEventListener('DOMContentLoaded', () => { new SnakeGameClient(); });
