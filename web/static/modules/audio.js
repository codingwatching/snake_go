export class SoundManager {
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

    playWin() {
        if (!this.enabled || !this.ctx) return;
        // Major arpeggio for celebration
        const now = this.ctx.currentTime;
        const freqs = [523.25, 659.25, 783.99, 1046.50]; // C5, E5, G5, C6
        freqs.forEach((f, i) => {
            const osc = this.ctx.createOscillator();
            const gain = this.ctx.createGain();
            osc.type = 'triangle';
            osc.frequency.setValueAtTime(f, now + i * 0.1);
            gain.gain.setValueAtTime(0.1, now + i * 0.1);
            gain.gain.exponentialRampToValueAtTime(0.0001, now + i * 0.1 + 0.5);
            osc.connect(gain);
            gain.connect(this.ctx.destination);
            osc.start(now + i * 0.1);
            osc.stop(now + i * 0.1 + 0.5);
        });
    }

    playBoost(active) {
        if (active) {
            this.playTone(200, 'sine', 0.1, 0.03);
        }
    }

    playFire(variant = 'cannon') {
        if (!this.enabled || !this.ctx) return;

        switch (variant) {
            case 'cannon':
                this.playTone(150, 'square', 0.2, 0.2);
                setTimeout(() => this.playTone(80, 'sine', 0.5, 0.15), 30);
                break;
            case 'laser':
                const osc = this.ctx.createOscillator();
                const gain = this.ctx.createGain();
                osc.type = 'sawtooth';
                osc.frequency.setValueAtTime(800, this.ctx.currentTime);
                osc.frequency.exponentialRampToValueAtTime(100, this.ctx.currentTime + 0.2);
                gain.gain.setValueAtTime(0.1, this.ctx.currentTime);
                gain.gain.exponentialRampToValueAtTime(0.0001, this.ctx.currentTime + 0.2);
                osc.connect(gain);
                gain.connect(this.ctx.destination);
                osc.start();
                osc.stop(this.ctx.currentTime + 0.2);
                break;
            case 'pop':
                this.playTone(400, 'sine', 0.05, 0.2);
                setTimeout(() => this.playTone(200, 'square', 0.1, 0.1), 20);
                break;
            case 'pong':
                this.playTone(523.25, 'sine', 0.15, 0.15); // C5
                break;
        }
    }

    playExplosion() {
        if (!this.enabled || !this.ctx) return;
        this.playTone(100, 'sawtooth', 0.3, 0.2);
        this.playTone(60, 'square', 0.4, 0.1);
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();
        osc.type = 'triangle';
        osc.frequency.setValueAtTime(1000, this.ctx.currentTime);
        osc.frequency.exponentialRampToValueAtTime(10, this.ctx.currentTime + 0.15);
        gain.gain.setValueAtTime(0.1, this.ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.0001, this.ctx.currentTime + 0.15);
        osc.connect(gain);
        gain.connect(this.ctx.destination);
        osc.start();
        osc.stop(this.ctx.currentTime + 0.15);
    }

    playAIStun() {
        if (!this.enabled || !this.ctx) return;
        this.playTone(1200, 'sine', 0.1, 0.1);
        setTimeout(() => this.playTone(800, 'sine', 0.15, 0.05), 50);
        this.playTone(120, 'square', 0.5, 0.1);
    }

    playAIConsume() {
        if (!this.enabled || !this.ctx) return;
        this.playTone(220, 'triangle', 0.2, 0.08);
    }

    toggle() {
        this.enabled = !this.enabled;
        return this.enabled;
    }
}
