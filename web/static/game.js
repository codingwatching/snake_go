// Snake Game - Web Version
// WebSocket connection and rendering

class SnakeGameClient {
    constructor() {
        this.canvas = document.getElementById('gameCanvas');
        this.ctx = this.canvas.getContext('2d');
        this.ws = null;
        this.gameState = null;

        // Canvas settings
        this.cellSize = 20;
        this.boardWidth = 25;
        this.boardHeight = 25;

        this.canvas.width = this.boardWidth * this.cellSize;
        this.canvas.height = this.boardHeight * this.cellSize;

        // UI elements
        this.scoreEl = document.getElementById('score');
        this.bestScoreEl = document.getElementById('bestScore');
        this.speedEl = document.getElementById('speed');
        this.eatenEl = document.getElementById('eaten');
        this.boostIndicator = document.getElementById('boostIndicator');
        this.messageDisplay = document.getElementById('messageDisplay');
        this.gameOverlay = document.getElementById('gameOverlay');
        this.overlayTitle = document.getElementById('overlayTitle');
        this.overlayMessage = document.getElementById('overlayMessage');
        this.connectionStatus = document.getElementById('connectionStatus');

        // High score persistence
        this.bestScore = parseInt(localStorage.getItem('snake_best_score')) || 0;
        this.bestScoreEl.textContent = this.bestScore;

        this.setupWebSocket();
        this.setupKeyboard();
        this.setupMobileControls();

        // State for UI optimization
        this.lastMessage = '';
        this.messageTimeout = null;
        this.lastGameOver = false;

        // Start animation loop once
        this.startAnimationLoop();
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

        Object.entries(buttons).forEach(([id, action]) => {
            const btn = document.getElementById(id);
            if (!btn) return;

            // Handle both touch and click
            const handleAction = (e) => {
                e.preventDefault();
                if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
                this.ws.send(JSON.stringify({ action }));
            };

            btn.addEventListener('touchstart', handleAction);
            btn.addEventListener('mousedown', handleAction);
        });

        // Restart support via overlay tap/touch
        const handleStartRestart = () => {
            if (this.gameState && this.gameState.gameOver) {
                this.ws.send(JSON.stringify({ action: 'restart' }));
            } else if (this.gameState && !this.gameState.started) {
                this.ws.send(JSON.stringify({ action: 'start' }));
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
            else if (key === 'arrowleft' || key === 'a') action = 'left';
            else if (key === 'arrowright' || key === 'd') action = 'right';
            else if (key === ' ' || key === 'p') action = 'pause';
            else if (key === 'r') action = 'restart';
            else if (key === 'q') action = 'quit';

            if (action) {
                e.preventDefault();
                this.ws.send(JSON.stringify({ action }));
            }
        });
    }

    updateUI() {
        if (!this.gameState) return;

        // 1. Update Current Stats
        const currentScore = this.gameState.score || 0;
        this.scoreEl.textContent = currentScore;
        this.speedEl.textContent = (this.gameState.eatingSpeed || 0).toFixed(2);
        this.eatenEl.textContent = this.gameState.foodEaten || 0;

        // 2. High Score Logic
        if (currentScore > this.bestScore) {
            this.bestScore = currentScore;
            this.bestScoreEl.textContent = this.bestScore;
            localStorage.setItem('snake_best_score', this.bestScore);
            this.bestScoreEl.parentElement.classList.add('new-record');
        } else {
            this.bestScoreEl.parentElement.classList.remove('new-record');
        }

        // 3. Game Over Special Message
        if (this.gameState.gameOver && !this.lastGameOver) {
            if (currentScore >= this.bestScore && currentScore > 0) {
                this.gameState.message = "🎊 AMAZING! NEW HIGH SCORE! 🎊";
            }
        }
        this.lastGameOver = this.gameState.gameOver;

        // 4. Boost indicator
        if (this.gameState.boosting) {
            this.boostIndicator.classList.add('active');
        } else {
            this.boostIndicator.classList.remove('active');
        }

        // 5. Message display optimization (same as before)
        const currentMsg = this.gameState.message || '';
        if (currentMsg && currentMsg !== this.lastMessage) {
            this.lastMessage = currentMsg;
            this.messageDisplay.textContent = currentMsg;
            this.messageDisplay.classList.add('show');

            if (this.messageTimeout) {
                clearTimeout(this.messageTimeout);
            }

            this.messageTimeout = setTimeout(() => {
                this.messageDisplay.classList.remove('show');
                this.lastMessage = '';
            }, 3000);
        } else if (!currentMsg) {
            this.messageDisplay.textContent = '';
            this.messageDisplay.classList.remove('show');
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
