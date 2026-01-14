export class GameRenderer {
    constructor(canvas, ctx, cellSize) {
        this.canvas = canvas;
        this.ctx = ctx;
        this.cellSize = cellSize;
    }

    render(gameState, boardWidth, boardHeight, explosions, confetti, floatingScores, currentMessage, messageStartTime, messageType = 'normal') {
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

        // Draw foods
        if (gameState.foods) {
            const isActive = gameState.started && !gameState.paused && !gameState.gameOver;
            gameState.foods.forEach(food => this.drawFood(food, isActive));
        }

        // Draw player snake
        if (gameState.snake) {
            gameState.snake.forEach((segment, index) => {
                if (index === 0) {
                    this.ctx.fillStyle = '#48bb78';
                    this.drawCell(segment.x, segment.y);
                    this.drawEyes(segment.x, segment.y, false);
                } else {
                    this.ctx.fillStyle = '#68d391';
                    this.drawCell(segment.x, segment.y);
                }
            });
        }

        // Draw AI snake
        if (gameState.aiSnake) {
            const isStunned = gameState.aiStunned;
            gameState.aiSnake.forEach((segment, index) => {
                if (index === 0) {
                    this.ctx.fillStyle = isStunned ? '#718096' : '#9f7aea';
                    this.drawCell(segment.x, segment.y);
                    this.drawEyes(segment.x, segment.y, true, isStunned);
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

    drawCell(x, y) {
        this.ctx.fillRect(x * this.cellSize + 1, y * this.cellSize + 1, this.cellSize - 2, this.cellSize - 2);
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
                this.ctx.strokeStyle = '#fff';
                this.ctx.lineWidth = 1;
                this.ctx.beginPath();
                this.ctx.moveTo(centerX - 6, centerY - 6); this.ctx.lineTo(centerX - 2, centerY - 2);
                this.ctx.moveTo(centerX - 2, centerY - 6); this.ctx.lineTo(centerX - 6, centerY - 2);
                this.ctx.moveTo(centerX + 2, centerY - 6); this.ctx.lineTo(centerX + 6, centerY - 2);
                this.ctx.moveTo(centerX + 6, centerY - 6); this.ctx.lineTo(centerX + 2, centerY - 2);
                this.ctx.stroke();
            } else {
                this.ctx.fillStyle = '#e53e3e';
                this.ctx.beginPath();
                this.ctx.arc(centerX - 4, centerY - 4, 1, 0, Math.PI * 2);
                this.ctx.arc(centerX + 4, centerY - 4, 1, 0, Math.PI * 2);
                this.ctx.fill();
            }
        } else {
            this.ctx.fillStyle = '#000';
            this.ctx.beginPath();
            this.ctx.arc(centerX - 4, centerY - 4, 2, 0, Math.PI * 2);
            this.ctx.arc(centerX + 4, centerY - 4, 2, 0, Math.PI * 2);
            this.ctx.fill();
        }
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
        this.ctx.globalAlpha = 1 - progress;
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
            this.ctx.globalAlpha = p.life;
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
        scores.forEach(s => {
            this.ctx.globalAlpha = s.life;
            const paddingH = 8, paddingV = 4;
            this.ctx.font = 'bold 13px Inter, sans-serif';
            const textWidth = this.ctx.measureText(s.text).width;

            this.ctx.fillStyle = 'rgba(255, 255, 255, 0.1)';
            const x = s.x - (textWidth / 2) - paddingH;
            const y = s.y - 10 - paddingV;
            const w = textWidth + (paddingH * 2), h = 14 + (paddingV * 2);

            this.ctx.beginPath();
            this.ctx.roundRect(x, y, w, h, 10);
            this.ctx.fill();
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
            this.ctx.stroke();

            this.ctx.fillStyle = s.color;
            this.ctx.textAlign = 'center';
            this.ctx.fillText(s.text, s.x, s.y);
        });
        this.ctx.globalAlpha = 1.0;
    }

    drawCanvasMessage(msg, startTime, messageType = 'normal') {
        if (!msg) return;
        const age = Date.now() - startTime;

        // Calculate display duration based on message type
        const maxDuration = messageType === 'permanent' ? Infinity :
            (messageType === 'bonus' ? 800 : 1000);
        const fadeStart = maxDuration - 500;

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
        this.ctx.globalAlpha = displayAlpha;

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
