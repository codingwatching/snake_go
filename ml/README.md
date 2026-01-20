# 🧠 Snake AI: Machine Learning Pipeline

This directory contains the Reinforcement Learning (RL) training pipeline for the Snake AI.

## 📁 Directory Structure

- `dataset.py`: Parses `.jsonl` game recordings and converts them into 6-channel tensors for training.
- `model.py`: Defines the 3-layer Convolutional Neural Network (CNN) architecture.
- `train.py`: The main training script using the **DQN (Deep Q-Learning)** algorithm.
- `requirements.txt`: Python dependencies required for training.
- `checkpoints/`: (Ignored by Git) Directory where `.pth` and `.onnx` models are saved.

## 🚀 How to Train

### 1. Collect Data
Play the game or let the heuristic AI play. Recordings are saved in the `records/` directory at the project root.
- Ensure the game server is configured to record (default behavior in recent versions).

### 2. Setup Python Environment
```bash
cd ml
pip install -r requirements.txt
```

### 3. Run Training
```bash
python train.py
```
This script will:
1. Scan the `records/` folder for transitions.
2. Train a DDQN-inspired model using the **Mean Squared Error** on Temporal Difference errors.
3. Export the final model to `checkpoints/snake_policy.onnx`.

### 4. Deploy to Go
The Go game server automatically looks for `ml/checkpoints/snake_policy.onnx` on startup. Just restart the server to apply the new "brain".

## 📊 Feature Grid Details (6 Channels)
The model "sees" the board as a 6-layered 25x25 grid:
1. **Channel 0**: AI Snake Head
2. **Channel 1**: AI Snake Body
3. **Channel 2**: Enemy Snake Head
4. **Channel 3**: Enemy Snake Body
5. **Channel 4**: Food Locations
6. **Channel 5**: Hazards (Walls, Obstacles, Fireballs)

## ⚖️ Reward Shaping
The AI learns to maximize the game score through these rewards:
- **Food**: +10 to +40 based on rarity.
- **Combat**: +50 for headshots, +20 for body hits.
- **Longevity**: +0.1 per tick.
- **Death**: -100 (The ultimate motivator).
