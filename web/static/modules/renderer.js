export class GameRenderer {
    constructor(canvas, ctx, cellSize) {
        this.canvas = canvas;
        this.ctx = ctx;
        this.cellSize = cellSize;
    }

    render(gameState, boardWidth, boardHeight, explosions, confetti, floatingScores, currentMessage, messageStartTime, messageType = 'normal', clientUsername = null) {
        // Clear canvas
        this.ctx.fillStyle = '#1a1a2e';
        this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);

        if (!gameState) {
            this.ctx.fillStyle = '#fff';
            this.ctx.font = '20px Inter, sans-serif';
            this.ctx.textAlign = 'center';
            this.ctx.fillText('Connecting to server...', this.canvas.width / 2, this.canvas.height / 2);
            return;
        }

        // ... (Walls, Obstacles, Foods drawing unchanged) ...

        // Draw walls
        this.ctx.fillStyle = '#4a5568';
        for (let x = 0; x < boardWidth; x++) {
            this.drawCell(x, 0);
            this.drawCell(x, boardHeight - 1);
        }
        for (let y = 0; y < boardHeight; y++) {
            this.drawCell(0, y);
            this.drawCell(boardWidth - 1, y);
        }

        // Draw obstacles
        if (gameState.obstacles) {
            this.ctx.fillStyle = '#cbd5e0';
            this.ctx.shadowBlur = 8;
            this.ctx.shadowColor = 'rgba(203, 213, 224, 0.5)';
            gameState.obstacles.forEach(obs => {
                obs.points.forEach(p => this.drawCell(p.x, p.y));
            });
            this.ctx.shadowBlur = 0;
        }

        // Draw props
        if (gameState.props) {
            this.drawProps(gameState.props);
        }

        // Draw foods
        if (gameState.foods) {
            const isActive = gameState.started && !gameState.paused && !gameState.gameOver;
            gameState.foods.forEach(food => this.drawFood(food, isActive));
        }

        // Draw player snake
        if (gameState.snake) {
            const isLocal = clientUsername && gameState.p1Name === clientUsername;
            const isPlayerStunned = gameState.playerStunned;
            gameState.snake.forEach((segment, index) => {
                if (index === 0) {
                    this.ctx.fillStyle = isPlayerStunned ? '#a0aec0' : '#48bb78';
                    this.drawCell(segment.x, segment.y);
                    this.drawEyes(segment.x, segment.y, false, isPlayerStunned);
                    if (isLocal) this.drawYouIndicator(segment.x, segment.y, '#48bb78');
                    this.drawActiveEffects(segment.x, segment.y, gameState.p1Effects || []);
                } else {
                    this.ctx.fillStyle = isPlayerStunned ? '#cbd5e0' : '#68d391';
                    this.drawCell(segment.x, segment.y);
                }
            });
        }

        // Draw AI snake
        if (gameState.aiSnake) {
            const isLocal = clientUsername && gameState.p2Name === clientUsername;
            const isStunned = gameState.aiStunned;
            gameState.aiSnake.forEach((segment, index) => {
                if (index === 0) {
                    this.ctx.fillStyle = isStunned ? '#718096' : '#9f7aea';
                    this.drawCell(segment.x, segment.y);
                    this.drawEyes(segment.x, segment.y, true, isStunned);
                    if (isLocal) this.drawYouIndicator(segment.x, segment.y, '#9f7aea');
                    this.drawActiveEffects(segment.x, segment.y, gameState.p2Effects || []);
                } else {
                    this.ctx.fillStyle = isStunned ? '#a0aec0' : '#b794f4';
                    this.drawCell(segment.x, segment.y);
                }
            });
        }

        // Draw fireballs
        if (gameState.fireballs) {
            gameState.fireballs.forEach(fb => this.drawFireball(fb));
        }

        // Draw effects
        explosions.forEach(exp => this.drawExplosion(exp));
        this.drawConfetti(confetti);
        this.drawFloatingScores(floatingScores);

        // Crash point
        if (gameState.gameOver && gameState.crashPoint) {
            this.ctx.font = `${this.cellSize}px sans-serif`;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'middle';
            this.ctx.fillText('💥',
                gameState.crashPoint.x * this.cellSize + this.cellSize / 2,
                gameState.crashPoint.y * this.cellSize + this.cellSize / 2
            );
        }

        // UI Message
        this.drawCanvasMessage(currentMessage, messageStartTime, messageType);
    }

    drawYouIndicator(x, y, color) {
        const bounce = Math.sin(Date.now() * 0.01) * 3;
        const centerX = x * this.cellSize + this.cellSize / 2;
        const topY = y * this.cellSize - 15 + bounce;

        this.ctx.save();
        this.ctx.fillStyle = color;
        this.ctx.font = 'bold 12px Inter, sans-serif';
        this.ctx.textAlign = 'center';
        this.ctx.shadowBlur = 10;
        this.ctx.shadowColor = color;

        // Background bubble
        this.ctx.globalAlpha = 0.8;
        this.ctx.fillText("YOU", centerX, topY);

        // Triangle pointer
        this.ctx.beginPath();
        this.ctx.moveTo(centerX - 4, topY + 4);
        this.ctx.lineTo(centerX + 4, topY + 4);
        this.ctx.lineTo(centerX, topY + 10);
        this.ctx.fill();
        this.ctx.restore();
    }

    drawCell(x, y) {
        this.ctx.fillRect(x * this.cellSize + 1, y * this.cellSize + 1, this.cellSize - 2, this.cellSize - 2);
    }

    drawProps(props) {
        props.forEach(prop => {
            if (!prop.pos) return;
            const centerX = prop.pos.x * this.cellSize + this.cellSize / 2;
            const centerY = prop.pos.y * this.cellSize + this.cellSize / 2;

            // Pulsing glow
            const pulse = Math.sin(Date.now() * 0.005) * 5 + 10;
            this.ctx.save();
            this.ctx.lineWidth = 2;
            this.ctx.shadowBlur = pulse;

            let emoji = "🎁";
            if (prop.type === 0) { emoji = "🛡️"; this.ctx.shadowColor = "#4299e1"; }
            else if (prop.type === 1) { emoji = "🌀"; this.ctx.shadowColor = "#9f7aea"; }
            else if (prop.type === 2) { emoji = "✂️"; this.ctx.shadowColor = "#f56565"; }
            else if (prop.type === 3) { emoji = "🧲"; this.ctx.shadowColor = "#48bb78"; }
            else if (prop.type === 4) { emoji = "👑"; this.ctx.shadowColor = "#ecc94b"; } // Big Chest (Yellow/Gold)
            else if (prop.type === 5) { emoji = "💰"; this.ctx.shadowColor = "#f6e05e"; } // Small Chest (Light Yellow)
            else if (prop.type === 6) { emoji = "⚡"; this.ctx.shadowColor = "#ffeb3b"; } // Rapid Fire (Yellow)
            else if (prop.type === 7) { emoji = "🌟"; this.ctx.shadowColor = "#e0aaff"; } // Scatter Shot (Light Purple)

            this.ctx.font = `${this.cellSize * 0.8}px sans-serif`;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'middle';
            this.ctx.fillText(emoji, centerX, centerY);
            this.ctx.restore();
        });
    }

    drawActiveEffects(x, y, effects) {
        if (!effects || effects.length === 0) return;

        const centerX = x * this.cellSize + this.cellSize / 2;
        const centerY = y * this.cellSize + this.cellSize / 2;

        effects.forEach((eff, i) => {
            this.ctx.save();
            this.ctx.beginPath();
            this.ctx.arc(centerX, centerY, this.cellSize * (0.6 + i * 0.2), 0, Math.PI * 2);
            this.ctx.lineWidth = 2;
            this.ctx.globalAlpha = 0.4 + Math.sin(Date.now() * 0.01 + i) * 0.2;

            if (eff.type === "SHIELD") this.ctx.strokeStyle = "#4299e1";
            else if (eff.type === "TIMEWARP") this.ctx.strokeStyle = "#9f7aea";
            else if (eff.type === "MAGNET") this.ctx.strokeStyle = "#48bb78";
            else if (eff.type === "RAPIDFIRE") this.ctx.strokeStyle = "#ffeb3b";
            else if (eff.type === "SCATTER") this.ctx.strokeStyle = "#e0aaff";

            this.ctx.stroke();
            this.ctx.restore();
        });
    }

    drawEyes(x, y, isAI, isStunned = false) {
        const centerX = x * this.cellSize + this.cellSize / 2;
        const centerY = y * this.cellSize + this.cellSize / 2;

        if (isAI) {
            this.ctx.fillStyle = isStunned ? '#4a5568' : '#fff';
            this.ctx.beginPath();
            this.ctx.arc(centerX - 4, centerY - 4, 3, 0, Math.PI * 2);
            this.ctx.arc(centerX + 4, centerY - 4, 3, 0, Math.PI * 2);
            this.ctx.fill();

            if (isStunned) {
                this.drawXEyes(centerX, centerY);
            } else {
                this.ctx.fillStyle = '#e53e3e';
                this.ctx.beginPath();
                this.ctx.arc(centerX - 4, centerY - 4, 1, 0, Math.PI * 2);
                this.ctx.arc(centerX + 4, centerY - 4, 1, 0, Math.PI * 2);
                this.ctx.fill();
            }
        } else {
            this.ctx.fillStyle = isStunned ? '#4a5568' : '#000';
            this.ctx.beginPath();
            this.ctx.arc(centerX - 4, centerY - 4, 2, 0, Math.PI * 2);
            this.ctx.arc(centerX + 4, centerY - 4, 2, 0, Math.PI * 2);
            this.ctx.fill();

            if (isStunned) {
                this.drawXEyes(centerX, centerY, 4);
            }
        }
    }

    drawXEyes(centerX, centerY, size = 6) {
        this.ctx.strokeStyle = '#fff';
        this.ctx.lineWidth = 1;
        this.ctx.beginPath();
        // Left eye X
        this.ctx.moveTo(centerX - 4 - size / 2, centerY - 4 - size / 2); this.ctx.lineTo(centerX - 4 + size / 2, centerY - 4 + size / 2);
        this.ctx.moveTo(centerX - 4 + size / 2, centerY - 4 - size / 2); this.ctx.lineTo(centerX - 4 - size / 2, centerY - 4 + size / 2);
        // Right eye X
        this.ctx.moveTo(centerX + 4 - size / 2, centerY - 4 - size / 2); this.ctx.lineTo(centerX + 4 + size / 2, centerY - 4 + size / 2);
        this.ctx.moveTo(centerX + 4 + size / 2, centerY - 4 - size / 2); this.ctx.lineTo(centerX + 4 - size / 2, centerY - 4 + size / 2);
        this.ctx.stroke();
    }

    drawFood(food, isActive = true) {
        this.ctx.save();
        const centerX = food.pos.x * this.cellSize + this.cellSize / 2;
        const centerY = food.pos.y * this.cellSize + this.cellSize / 2;
        const emojis = { 0: '🟣', 1: '🔵', 2: '🟠', 3: '🔴' };

        let scale = 1.0;
        if (isActive && food.remainingSeconds > 0 && food.remainingSeconds <= 5) {
            const frequency = 5 - (food.remainingSeconds - 1) * 0.5;
            scale = 1 + 0.15 * Math.sin(Date.now() * 0.005 * frequency);
        }

        this.ctx.font = `${(this.cellSize - 4) * scale}px sans-serif`;
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';
        this.ctx.fillText(emojis[food.foodType] || '🟣', centerX, centerY);

        if (food.remainingSeconds > 0 && food.remainingSeconds <= 5) {
            const radius = this.cellSize / 2 - 1;
            const startAngle = -Math.PI / 2;
            const progress = food.remainingSeconds / 5;
            const endAngle = startAngle + (progress * Math.PI * 2);

            this.ctx.beginPath();
            this.ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.15)';
            this.ctx.lineWidth = 2;
            this.ctx.stroke();

            this.ctx.beginPath();
            this.ctx.arc(centerX, centerY, radius, startAngle, endAngle);
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.8)';
            this.ctx.lineWidth = 2;
            this.ctx.lineCap = 'round';
            this.ctx.shadowBlur = 6;
            this.ctx.shadowColor = '#fff';
            this.ctx.stroke();
        }
        this.ctx.restore();
    }

    drawFireball(fb) {
        const centerX = fb.pos.x * this.cellSize + this.cellSize / 2;
        const centerY = fb.pos.y * this.cellSize + this.cellSize / 2;
        this.ctx.save();
        this.ctx.shadowBlur = 15;
        this.ctx.shadowColor = '#ff4d00';
        this.ctx.fillStyle = '#ff6600';
        this.ctx.beginPath();
        this.ctx.arc(centerX, centerY, this.cellSize / 2.5, 0, Math.PI * 2);
        this.ctx.fill();
        this.ctx.shadowBlur = 4;
        this.ctx.fillStyle = '#ffcc00';
        this.ctx.beginPath();
        this.ctx.arc(centerX, centerY, this.cellSize / 5, 0, Math.PI * 2);
        this.ctx.fill();
        this.ctx.restore();
    }

    drawExplosion(exp) {
        const centerX = exp.x * this.cellSize + this.cellSize / 2;
        const centerY = exp.y * this.cellSize + this.cellSize / 2;
        const progress = (Date.now() - exp.startTime) / exp.duration;
        this.ctx.save();
        const radius = (this.cellSize * 1.5) * progress;
        this.ctx.globalAlpha = Math.max(0, 1 - progress);
        const grad = this.ctx.createRadialGradient(centerX, centerY, 0, centerX, centerY, radius);
        grad.addColorStop(0, '#fff');
        grad.addColorStop(0.3, '#ff0');
        grad.addColorStop(0.7, '#f40');
        grad.addColorStop(1, 'transparent');
        this.ctx.fillStyle = grad;
        this.ctx.beginPath();
        this.ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
        this.ctx.fill();
        this.ctx.restore();
    }

    drawConfetti(confetti) {
        confetti.forEach(p => {
            this.ctx.save();
            this.ctx.globalAlpha = Math.max(0, p.life);
            this.ctx.fillStyle = p.color;
            this.ctx.translate(p.x, p.y);
            this.ctx.rotate(p.rotation);

            if (p.type === 'strip') {
                this.ctx.fillRect(-p.size, -p.size / 3, p.size * 2, p.size / 1.5);
            } else if (p.type === 'square') {
                this.ctx.fillRect(-p.size, -p.size, p.size * 2, p.size * 2);
            } else {
                this.ctx.beginPath();
                this.ctx.arc(0, 0, p.size, 0, Math.PI * 2);
                this.ctx.fill();
            }
            this.ctx.restore();
        });
    }

    drawFloatingScores(scores) {
        if (!scores || scores.length === 0) return;
        const now = Date.now();

        this.ctx.save();
        // Reset shadow to avoid bleed from other renderers
        this.ctx.shadowBlur = 0;
        this.ctx.shadowColor = 'transparent';

        scores.forEach(s => {
            const age = now - s.startTime;
            const progress = age / s.duration;
            if (progress >= 1) return;

            // Calculate alpha: stay solid for 20%, then fade linearly to 0
            let alpha = 1.0;
            const fadeStart = 0.2;
            if (progress > fadeStart) {
                alpha = 1.0 - (progress - fadeStart) / (1.0 - fadeStart);
            }

            // Critical: Absolute clamping
            alpha = Math.max(0, Math.min(1, alpha));

            // Hard Cutoff: Lowered threshold for more noticeable fade
            if (alpha < 0.05) return;

            this.ctx.save();
            this.ctx.globalAlpha = alpha;

            this.ctx.font = 'bold 13px Inter, sans-serif';
            const textWidth = this.ctx.measureText(s.text).width;
            const paddingH = 8, paddingV = 4;
            const w = textWidth + (paddingH * 2), h = 14 + (paddingV * 2);
            const x = s.x - w / 2;
            const y = s.y - 10 - paddingV;

            // Draw background pill
            this.ctx.fillStyle = 'rgba(0, 0, 0, 0.6)'; // Solid dark background
            this.ctx.beginPath();
            this.ctx.roundRect(x, y, w, h, 10);
            this.ctx.fill();

            // Border
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
            this.ctx.lineWidth = 1;
            this.ctx.stroke();

            // Text
            this.ctx.fillStyle = s.color;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'middle';
            this.ctx.fillText(s.text, s.x, y + h / 2);

            this.ctx.restore();
        });

        this.ctx.restore();
    }

    drawCanvasMessage(msg, startTime, messageType = 'normal') {
        if (!msg) return;
        const age = Date.now() - startTime;

        // Calculate display duration based on message type
        const maxDuration = messageType === 'permanent' ? Infinity :
            (messageType === 'bonus' ? 600 : 700);
        const fadeStart = maxDuration - 300;

        // Check if message has expired
        if (age > maxDuration) return;

        // Calculate fade animation
        let displayAlpha = 1.0, yOffset = 0;
        if (messageType !== 'permanent' && age > fadeStart) {
            const fadeProgress = (age - fadeStart) / 500;
            displayAlpha = 1.0 - fadeProgress;
            yOffset = -fadeProgress * 30;
        }

        const x = this.canvas.width / 2;
        const y = messageType === 'bonus' ? this.canvas.height / 5 : this.canvas.height / 5;

        this.ctx.save();
        this.ctx.globalAlpha = Math.max(0, displayAlpha);

        // Different font sizes based on message type
        const fontSize = messageType === 'bonus' ? 16 : (messageType === 'permanent' ? 28 : 18);
        this.ctx.font = `bold ${fontSize}px Inter, sans-serif`;

        // Handle multi-line text
        const lines = msg.split('\\n');
        const lineHeight = fontSize * 1.4;

        // Measure all lines to get max width
        let maxWidth = 0;
        lines.forEach(line => {
            const width = this.ctx.measureText(line).width;
            if (width > maxWidth) maxWidth = width;
        });

        const rectWidth = maxWidth + (messageType === 'bonus' ? 30 : 35);
        const rectHeight = (messageType === 'bonus' ? 36 : 42) + (lines.length - 1) * lineHeight;

        this.ctx.fillStyle = messageType === 'bonus' ? 'rgba(0, 0, 0, 0.65)' : 'rgba(0, 0, 0, 0.8)';
        this.ctx.shadowBlur = messageType === 'bonus' ? 10 : 15;
        this.ctx.shadowColor = 'rgba(0,0,0,0.5)';
        this.ctx.beginPath();
        this.ctx.roundRect(x - rectWidth / 2, y + yOffset - rectHeight / 2, rectWidth, rectHeight, messageType === 'bonus' ? 18 : 27);
        this.ctx.fill();
        this.ctx.strokeStyle = messageType === 'bonus' ? 'rgba(255, 255, 255, 0.3)' : 'rgba(255, 255, 255, 0.5)';
        this.ctx.lineWidth = messageType === 'bonus' ? 1 : 2;
        this.ctx.stroke();

        this.ctx.fillStyle = '#ffffff';
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';

        // Draw each line
        const startY = y + yOffset - ((lines.length - 1) * lineHeight) / 2;
        lines.forEach((line, index) => {
            this.ctx.fillText(line, x, startY + index * lineHeight);
        });

        this.ctx.restore();
    }
}
