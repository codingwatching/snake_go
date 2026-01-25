import { SoundManager } from './modules/audio.js';
import { GameRenderer } from './modules/renderer.js?v=2.3';

export class SnakeGameClient {
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
        this.userInfoBar = document.getElementById('userInfoBar');
        this.displayUsername = document.getElementById('displayUsername');

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
        this.currentUser = null;

        // Leaderboard state
        this.leaderboards = { scores: [], winRates: [] };
        this.currentLeaderboardTab = 'scores';

        // Matchmaking State
        this.isMatching = false;

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
        this.winRateEl = document.getElementById('winRate');
        this.gamesWonEl = document.getElementById('gamesWon');

        // Leaderboard Elements
        this.leaderboardList = document.getElementById('leaderboardList');

        // Auth Elements
        this.authOverlay = document.getElementById('authOverlay');
        this.authUsernameInput = document.getElementById('authUsername');
        this.authPasswordInput = document.getElementById('authPassword');
        this.authRememberInput = document.getElementById('authRemember');
        this.btnLogoutUser = document.getElementById('btnLogoutUser');
        this.btnLogin = document.getElementById('btnLogin');
        this.btnRegister = document.getElementById('btnRegister');
        this.authMessage = document.getElementById('authMessage');
        this.cancelMatchBtn = document.getElementById('cancelMatchBtn');

        if (this.btnLogoutUser) this.btnLogoutUser.onclick = () => this.handleLogout();

        if (this.cancelMatchBtn) {
            this.cancelMatchBtn.onclick = (e) => {
                e.stopPropagation(); // Prevent starting game via overlay click
                if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                    this.ws.send(JSON.stringify({ action: 'cancel_match' }));
                    this.isMatching = false;
                    this.updateOverlay();
                }
            };
        }

        this.autoModeSelect = document.getElementById('auto-mode');

        // Leaderboard Tabs
        this.tabScore = document.getElementById('tabScore');
        this.tabWinRate = document.getElementById('tabWinRate');
        if (this.tabScore) this.tabScore.onclick = () => this.switchLeaderboardTab('scores');
        if (this.tabWinRate) this.tabWinRate.onclick = () => this.switchLeaderboardTab('winRates');

        // Allow Enter key to trigger login
        const handleEnter = (e) => {
            if (e.key === 'Enter') {
                this.handleAuth('login');
            }
        };
        if (this.authUsernameInput) this.authUsernameInput.onkeydown = handleEnter;
        if (this.authPasswordInput) this.authPasswordInput.onkeydown = handleEnter;

        // Auto-fill username if remembered
        this.checkAutoLogin();
    }

    checkAutoLogin() {
        const saved = localStorage.getItem('snake_auth');
        if (saved) {
            try {
                const creds = JSON.parse(atob(saved));
                if (this.authUsernameInput) this.authUsernameInput.value = creds.username;
                if (this.authPasswordInput) this.authPasswordInput.value = creds.password;
            } catch (e) {
                localStorage.removeItem('snake_auth');
            }
        }
    }

    handleLogout() {
        if (!confirm("Are you sure you want to logout?")) return;

        localStorage.removeItem('snake_auth');
        this.currentUser = null;
        this.authUsernameInput.value = '';
        this.authPasswordInput.value = '';

        // Hide user info bar
        if (this.userInfoBar) this.userInfoBar.classList.add('hidden');

        // Show auth overlay
        this.authOverlay.classList.remove('hidden');
        this.authTitle.textContent = 'SNAKE LOGIN';
        this.authMessage.textContent = 'Logged out successfully';
        this.authMessage.className = 'success';

        // Reset stats UI
        if (this.winRateEl) this.winRateEl.textContent = '0%';
        if (this.gamesWonEl) this.gamesWonEl.textContent = '0/0';

        this.showTempMessage("Logged out");
    }

    switchLeaderboardTab(tab) {
        this.currentLeaderboardTab = tab;
        if (this.tabScore) this.tabScore.classList.toggle('active', tab === 'scores');
        if (this.tabWinRate) this.tabWinRate.classList.toggle('active', tab === 'winRates');
        this.renderLeaderboard();
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
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
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
        const scoreDuration = 1200; // 1.2 seconds total
        this.floatingScores = this.floatingScores.filter(s => now - s.startTime < scoreDuration);
        this.floatingScores.forEach(s => {
            const age = now - s.startTime;
            // Float up faster at start, slower later
            s.y -= 0.8;
        });

        // 3. Update and Filter Confetti
        this.confetti.forEach(p => {
            p.x += p.vx;
            p.y += p.vy;
            p.vy += 0.15; // Gravity
            p.vx *= 0.99; // Air resistance
            p.rotation += p.spin;
            p.life -= p.decay;
        });
        this.confetti = this.confetti.filter(p => p.life > 0);
    }

    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

        this.ws.onopen = () => {
            this.updateConnectionStatus('connected');
            // Attempt auto-login if credentials exist
            const saved = localStorage.getItem('snake_auth');
            if (saved) {
                try {
                    const creds = JSON.parse(atob(saved));
                    this.authMessage.textContent = "Auto-logging in...";
                    this.authMessage.className = "success";
                    this.ws.send(JSON.stringify({
                        action: 'login',
                        username: creds.username,
                        password: creds.password
                    }));
                } catch (e) {
                    localStorage.removeItem('snake_auth');
                }
            }
        };
        this.ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (msg.type === 'config') {
                this.handleConfig(msg.config);
            } else if (msg.type === 'state') {
                this.gameState = msg.state;
                if (msg.user) {
                    this.currentUser = msg.user;
                    this.updateUserStatsUI();
                }
                if (msg.leaderboard) {
                    console.log("🏆 Leaderboard update received via state message:", msg.leaderboard);
                    this.leaderboards.scores = msg.leaderboard;
                    if (msg.win_rates) this.leaderboards.winRates = msg.win_rates;
                    this.renderLeaderboard();
                }
                this.updateUI();
            } else if (msg.type === 'leaderboard') {
                this.leaderboards.scores = msg.leaderboard || [];
                this.leaderboards.winRates = msg.win_rates || [];
                this.renderLeaderboard();
            } else if (msg.type === 'auth_success') {
                this.onAuthSuccess(msg);
            } else if (msg.type === 'auth_error') {
                this.onAuthError(msg.error);
            } else if (msg.type === 'update_counts') {
                this.updatePlayerCount(msg.sessionCount);
            }
        };
        this.ws.onerror = () => {
            this.updateConnectionStatus('disconnected');
            this.isMatching = false;
            this.updateOverlay();
        };
        this.ws.onclose = () => {
            this.updateConnectionStatus('disconnected');
            this.isMatching = false;
            this.updateOverlay();
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
        } else if (this.gameState.mode === 'pvp') {
            this.aiStatEl.style.display = 'flex';
            this.timerEl.style.display = 'flex';
            this.aiStatEl.querySelector('.stat-label').textContent = this.gameState.p2Name || 'Player 2';
            this.scoreEl.previousElementSibling.textContent = this.gameState.p1Name || 'Player 1';
            this.aiScoreEl.textContent = this.gameState.aiScore || 0;
        } else {
            this.aiStatEl.style.display = 'flex';
            this.timerEl.style.display = 'flex';
            this.aiStatEl.querySelector('.stat-label').textContent = 'AI Score';
            this.scoreEl.previousElementSibling.textContent = 'Score';
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
        ['battle', 'zen', 'pvp'].forEach(m => {
            const btn = document.getElementById(`mode-${m}`);
            if (btn) {
                btn.classList.toggle('active', currentMode === m && !this.isMatching);
                if (m === 'pvp') {
                    btn.classList.toggle('searching', this.isMatching);
                }
            }
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

    handleAuth(type) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            this.onAuthError("Connecting to server...");
            return;
        }
        const username = this.authUsernameInput.value.trim();
        const password = this.authPasswordInput.value;
        if (!username || !password) {
            this.onAuthError("Missing username or password");
            return;
        }
        this.ws.send(JSON.stringify({
            action: type,
            username: username,
            password: password
        }));
    }

    onAuthSuccess(msg) {
        if (msg.user) {
            this.currentUser = msg.user;
            this.authOverlay.classList.add('hidden');
            this.showTempMessage(`Welcome, ${this.currentUser.username}!`);

            // Sync best score and stats from server
            if (this.currentUser.best_score > this.bestScore) {
                this.bestScore = this.currentUser.best_score;
                this.bestScoreEl.textContent = this.bestScore;
                localStorage.setItem('snake_best_score', this.bestScore);
            }
            this.updateUserStatsUI();

            // Save credentials if Remember Me is checked
            if (this.authRememberInput && this.authRememberInput.checked) {
                const creds = {
                    username: this.currentUser.username,
                    password: this.authPasswordInput.value
                };
                localStorage.setItem('snake_auth', btoa(JSON.stringify(creds)));
            } else {
                localStorage.removeItem('snake_auth');
            }
        } else if (msg.success) {
            this.authMessage.textContent = msg.success;
            this.authMessage.className = 'success';
        }
    }

    onAuthError(error) {
        this.authMessage.textContent = error;
        this.authMessage.className = 'error';
        // Clear saved credentials if auto-login fails
        if (error.toLowerCase().includes("not found") || error.toLowerCase().includes("invalid")) {
            localStorage.removeItem('snake_auth');
        }
    }

    updateUserStatsUI() {
        if (!this.currentUser) return;
        const total = this.currentUser.total_games || 0;
        const wins = this.currentUser.total_wins || 0;
        const rate = total > 0 ? Math.round((wins / total) * 100) : 0;

        if (this.winRateEl) this.winRateEl.textContent = `${rate}%`;
        if (this.gamesWonEl) this.gamesWonEl.textContent = `${wins}/${total}`;

        // Update User Info Bar
        if (this.userInfoBar && this.displayUsername && this.currentUser) {
            this.userInfoBar.classList.remove('hidden');
            this.displayUsername.textContent = this.currentUser.username;
        }
    }

    updateOverlay() {
        if (!this.gameState) return;

        const isGameOver = this.gameState.gameOver;
        const isStarted = this.gameState.started;
        const isPaused = this.gameState.paused;
        const isImportant = this.messageType === 'important';

        // 1. Reset special elements
        this.cancelMatchBtn.classList.add('hidden');
        this.overlayTitle.classList.remove('searching-pulse', 'important-message');

        if (this.isMatching) {
            this.gameOverlay.style.display = 'flex';
            this.overlayTitle.textContent = 'SEARCHING...';
            this.overlayTitle.classList.add('searching-pulse');
            this.overlayTitle.style.color = '#f6e05e';
            this.overlayMessage.textContent = 'Waiting for an opponent to join';
            this.cancelMatchBtn.classList.remove('hidden');
        } else if (isImportant && this.currentMessage) {
            // Synchronized Countdown / Match Found state
            // Only show overlay if game is paused or just starting
            // If the game is already moving, we don't want to block the view
            if ((isPaused || !isStarted || this.currentMessage.includes("FOUND")) && !this.currentMessage.includes("GO")) {
                this.gameOverlay.style.display = 'flex';
                this.overlayTitle.textContent = this.currentMessage;
                this.overlayTitle.classList.add('important-message');
                this.overlayTitle.style.color = '#fff';
                this.overlayMessage.textContent = this.gameState.mode === 'pvp' ? 'GET READY!' : '';
            } else {
                this.gameOverlay.style.display = 'none';
            }
        } else if (isGameOver) {
            this.gameOverlay.style.display = 'flex';
            if (this.gameState.winner) {
                if (this.gameState.mode === 'pvp') {
                    // PVP specific winner display
                    if (this.gameState.winner === 'player') {
                        const winnerName = this.gameState.p1Name || 'PLAYER 1';
                        this.overlayTitle.textContent = `🏆 ${winnerName.toUpperCase()} WINS!`;
                        this.overlayTitle.style.color = '#f6e05e';
                    } else if (this.gameState.winner === 'ai') {
                        const winnerName = this.gameState.p2Name || 'PLAYER 2';
                        this.overlayTitle.textContent = `🏆 ${winnerName.toUpperCase()} WINS!`;
                        this.overlayTitle.style.color = '#f6e05e';
                    } else {
                        this.overlayTitle.textContent = '🤝 DRAW!';
                        this.overlayTitle.style.color = '#4299e1';
                    }
                } else {
                    // Standard vs AI display
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
                }
                this.overlayMessage.innerHTML = `<span class="tap-hint">Tap or Press 'R' to Restart</span>`;
            } else {
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

    renderLeaderboard() {
        if (!this.leaderboardList) return;
        const type = this.currentLeaderboardTab;
        const entries = this.leaderboards[type] || [];

        if (type === 'scores') {
            this.leaderboardList.innerHTML = entries.map((entry, index) => {
                const dateObj = new Date(entry.date);
                const timeStr = dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
                const dateStr = dateObj.toLocaleDateString();
                const isTop1 = index === 0 ? 'top-1' : '';
                return `
                    <div class="leaderboard-item ${isTop1}">
                        <span class="rank">#${index + 1}</span>
                        <span class="player-name">${this.escapeHTML(entry.name)}</span>
                        <span class="player-score">${entry.score}</span>
                        <span class="player-meta">${entry.mode}<br>${dateStr} ${timeStr}</span>
                    </div>
                `;
            }).join('') || '<p class="loading-text">No scores yet. Be the first!</p>';
        } else {
            // Win Rate Leaderboard
            this.leaderboardList.innerHTML = entries.map((entry, index) => {
                const isTop1 = index === 0 ? 'top-1' : '';
                return `
                    <div class="leaderboard-item ${isTop1}">
                        <span class="rank">#${index + 1}</span>
                        <span class="player-name">${this.escapeHTML(entry.name)}</span>
                        <span class="player-score">${entry.win_rate.toFixed(1)}%</span>
                        <span class="player-meta">${entry.total_wins} / ${entry.total_games} Wins</span>
                    </div>
                `;
            }).join('') || '<p class="loading-text">No data yet.</p>';
        }
    }

    updateLeaderboardUI(entries) {
        // Legacy method replaced by renderLeaderboard
    }

    escapeHTML(str) {
        const p = document.createElement('p');
        p.textContent = str;
        return p.innerHTML;
    }

    processEvents() {
        if (this.gameState.scoreEvents) {
            this.gameState.scoreEvents.forEach(ev => {
                this.floatingScores.push({
                    x: ev.pos.x * this.cellSize + this.cellSize / 2,
                    y: ev.pos.y * this.cellSize,
                    text: ev.label,
                    startTime: Date.now(),
                    duration: 1200,
                    color: ev.label.includes('HEADSHOT') ? '#f6e05e' : (ev.label.includes('HIT') ? '#fc8181' : '#63b3ed')
                });
            });
        }

        if (this.gameState.hitPoints) {
            this.gameState.hitPoints.forEach(hp => {
                this.explosions.push({ x: hp.x, y: hp.y, startTime: Date.now(), duration: 500 });
            });
            if (this.gameState.hitPoints.length > 0) {
                this.sounds.playExplosion();
                this.triggerHaptic(30);
            }
        }

        if (this.gameState.message && this.gameState.message !== this.lastMessage) {
            this.messageType = this.gameState.messageType || 'normal';
            this.showTempMessage(this.gameState.message);
            this.lastMessage = this.gameState.message;

            // If match found or starting, clear searching state
            if (this.messageType === 'important' && (this.gameState.message.includes("MATCH FOUND") || this.gameState.message.includes("STARTING"))) {
                this.isMatching = false;
            }
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
                    decay: Math.random() * 0.01 + 0.005
                });
            }
        });

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
            if (e.ctrlKey || e.metaKey || e.altKey) return;

            // Ignore game controls if typing in an input field
            if (e.target.tagName === 'INPUT') return;

            const key = e.key.toLowerCase();
            const actionMap = {
                'arrowup': 'up', 'w': 'up', 'arrowdown': 'down', 's': 'down',
                'arrowleft': 'left', 'arrowright': 'right', 'd': 'right',
                ' ': 'pause', 'r': 'restart', 'q': 'quit', 'p': 'auto'
            };
            const action = actionMap[key];
            if (key === 'f' || key === 'enter') { this.fire(); return; }
            if (action) {
                e.preventDefault();
                const msg = { action };
                if (action === 'auto') {
                    msg.mode = this.autoModeSelect ? this.autoModeSelect.value : 'heuristic';
                }
                this.ws.send(JSON.stringify(msg));
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
                if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
                this.ws.send(JSON.stringify({ action }));
                this.sounds.playMove();
                if (action !== 'pause') {
                    this.pressTimer = setInterval(() => {
                        if (this.ws.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ action }));
                    }, 80);
                }
            };
            const end = () => { if (this.pressTimer) { clearInterval(this.pressTimer); this.pressTimer = null; } };
            btn.addEventListener('touchstart', start); btn.addEventListener('touchend', end);
            btn.addEventListener('mousedown', start); btn.addEventListener('mouseup', end);
        });
        document.getElementById('btn-fire')?.addEventListener('click', () => this.fire());

        const handleStartRestart = () => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
            if (this.isMatching) return; // Prevent clicking while matching
            if (this.gameState?.gameOver) this.ws.send(JSON.stringify({ action: 'restart' }));
            else if (this.gameState && !this.gameState.started) this.ws.send(JSON.stringify({ action: 'start' }));
        };
        this.gameOverlay.addEventListener('click', handleStartRestart);
        this.canvas.addEventListener('click', handleStartRestart);
    }

    setupDifficulty() {
        ['low', 'mid', 'high'].forEach(d => {
            document.getElementById(`diff-${d}`)?.addEventListener('click', () => {
                if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
                if (!this.gameState?.started || this.gameState?.gameOver) {
                    this.ws.send(JSON.stringify({ action: `diff_${d}` }));
                } else {
                    this.showTempMessage("Can't change difficulty during game!");
                }
            });
        });


        const berserkerBtn = document.getElementById('berserker-toggle');
        berserkerBtn?.addEventListener('click', () => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ action: 'toggleBerserker' }));
        });
    }

    setupAutoPlay() {
        // Redundant with setupDifficulty but keeping for completeness
        document.getElementById('btn-auto')?.addEventListener('click', () => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                const mode = this.autoModeSelect ? this.autoModeSelect.value : 'heuristic';
                this.ws.send(JSON.stringify({ action: 'auto', mode: mode }));
            }
        });
    }

    setupMode() {
        const setMode = (mode) => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify({ action: `mode_${mode}` }));
                this.showTempMessage(`${mode.charAt(0).toUpperCase() + mode.slice(1)} Mode Activated`);
            }
        };
        document.getElementById('mode-battle').onclick = () => setMode('battle');
        document.getElementById('mode-zen').onclick = () => setMode('zen');
        document.getElementById('mode-pvp').onclick = () => {
            if (!this.currentUser) {
                this.showTempMessage("Please login for P2P Battle!");
                this.authOverlay.classList.remove('hidden');
                return;
            }
            if (this.isMatching) return;

            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.isMatching = true;
                this.ws.send(JSON.stringify({ action: 'find_match' }));
                this.showTempMessage("Searching for opponent...");
                this.updateOverlay(); // Update UI immediately
            }
        };
    }

    showTempMessage(msg) {
        this.currentMessage = msg;
        this.messageStartTime = Date.now();
        if (this.messageTimeout) clearTimeout(this.messageTimeout);
        this.messageTimeout = setTimeout(() => {
            if (this.currentMessage === msg) this.currentMessage = '';
        }, 1000);
    }

    updateConnectionStatus(status) {
        const el = document.getElementById('connectionStatus');
        if (!el) return;
        el.querySelector('.status-text').textContent = status === 'connected' ? 'Connected' : (status === 'disconnected' ? 'Disconnected' : 'Connecting...');
        el.classList.toggle('connected', status === 'connected');
    }

    updatePlayerCount(count) {
        const el = document.getElementById('totalPlayers');
        if (el) {
            el.textContent = `👥 ${count} Player${count !== 1 ? 's' : ''}`;
        }
    }
}

if (!window.DISABLE_GAME_INIT) {
    document.addEventListener('DOMContentLoaded', () => { new SnakeGameClient(); });
}
