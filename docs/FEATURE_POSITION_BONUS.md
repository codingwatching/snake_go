# 🎯 位置难度奖励系统

## 功能概述

根据豆子的位置难度给予额外奖励分数，鼓励玩家挑战更困难的操作。

## 奖励规则

| 位置类型 | 奖励分数 | 图标 | 说明 |
|---------|---------|------|------|
| **角落** | +100 分 | 🏆 | 四个角落（最难吃到） |
| **靠边** | +30 分 | ⭐ | 贴墙但非角落 |
| **普通** | 0 分 | 豆子原色 | 中间区域 |

## 视觉效果

### 豆子图标
- 🏆 = 角落豆子（基础分 + 100）
- ⭐ = 靠边豆子（基础分 + 30）
- 🔴🟠🔵🟣 = 普通豆子（基础分）

### 恭喜消息
吃到特殊位置的豆子时，会显示3秒恭喜消息：
- **角落**: "🏆 恭喜！角落挑战 +100 分！"
- **靠边**: "⭐ 不错！靠边奖励 +30 分！"

## 计分示例

### 红色豆子（基础40分）
- 在角落: 40 + 100 = **140 分** 🏆
- 在边缘: 40 + 30 = **70 分** ⭐
- 在中间: 40 + 0 = **40 分** 🔴

### 紫色豆子（基础10分）
- 在角落: 10 + 100 = **110 分** 🏆
- 在边缘: 10 + 30 = **40 分** ⭐
- 在中间: 10 + 0 = **10 分** 🟣

## 实现细节

### 位置检测

**角落检测**（4个位置）:
```go
isTopLeft := x == 1 && y == 1
isTopRight := x == boardWidth-2 && y == 1
isBottomLeft := x == 1 && y == boardHeight-2
isBottomRight := x == boardWidth-2 && y == boardHeight-2
```

**边缘检测**（墙边但非角落）:
```go
isOnEdge := x == 1 || x == boardWidth-2 || y == 1 || y == boardHeight-2
```

### 核心方法

**文件**: `pkg/game/food.go`

```go
// 获取位置奖励
func (f *Food) GetPositionBonus(boardWidth, boardHeight int) int

// 获取总分（基础 + 位置奖励）
func (f *Food) GetTotalScore(boardWidth, boardHeight int) int

// 获取恭喜消息
func (f *Food) GetBonusMessage(boardWidth, boardHeight int) string

// 获取带位置标记的图标
func (f *Food) GetEmojiWithTimer(boardWidth, boardHeight int) string
```

### 消息系统

**文件**: `pkg/game/game.go`

```go
// 设置临时消息
func (g *Game) SetMessage(message string, duration time.Duration)

// 获取当前消息（如果还在显示期）
func (g *Game) GetMessage() string

// 检查是否有活跃消息
func (g *Game) HasActiveMessage() bool
```

## 测试验证

```bash
$ go test -v ./pkg/game/ -run TestPositionBonus
=== RUN   TestPositionBonus
    Corner {1 1}: bonus=100, message='🏆 恭喜！角落挑战 +100 分！'
    Corner {23 1}: bonus=100, message='🏆 恭喜！角落挑战 +100 分！'
    Corner {1 23}: bonus=100, message='🏆 恭喜！角落挑战 +100 分！'
    Corner {23 23}: bonus=100, message='🏆 恭喜！角落挑战 +100 分！'
    Edge {10 1}: bonus=30, message='⭐ 不错！靠边奖励 +30 分！'
    Edge {10 23}: bonus=30, message='⭐ 不错！靠边奖励 +30 分！'
    Edge {1 10}: bonus=30, message='⭐ 不错！靠边奖励 +30 分！'
    Edge {23 10}: bonus=30, message='⭐ 不错！靠边奖励 +30 分！'
    Normal {10 10}: bonus=0 (no message)
--- PASS: TestPositionBonus (0.00s)
```

## 游戏策略

### 风险 vs 回报
- **角落挑战**: 高风险高回报（撞墙概率高，但+100分）
- **靠边策略**: 中等风险（需要贴墙移动，+30分）
- **安全路线**: 低风险低回报（中间区域，无奖励）

### 最佳实践
1. 看到🏆要谨慎评估蛇的长度和位置
2. ⭐ 相对安全，值得一试
3. 结合豆子类型（红色角落 = 140分！）

## 受影响的文件

| 文件 | 修改内容 |
|------|---------|
| `pkg/game/types.go` | 添加消息系统字段 |
| `pkg/game/food.go` | 位置奖励计算和视觉标记 |
| `pkg/game/game.go` | 消息系统方法，更新计分逻辑 |
| `pkg/renderer/terminal.go` | 显示恭喜消息 |
| `pkg/game/bonus_test.go` | 完整测试套件 |

## 编译运行

```bash
# 重新编译
go build -o snake ./cmd/snake

# 运行游戏
./snake

# 现在试试吃角落的豆子，看看恭喜消息！🏆
```

---

**现在游戏更有挑战性和策略性了！** 🎮✨
