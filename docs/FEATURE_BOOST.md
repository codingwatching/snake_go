# 🚀 加速功能实现说明

## 功能概述

按住当前方向键可以触发3倍速度加速（boost）。

---

## 实现位置

加速功能主要在 **`cmd/snake/main.go`** 中实现。

---

## 核心机制

### 1. 配置参数

**文件**: `pkg/config/config.go`

```go
const (
    BaseTick             = 50 * time.Millisecond  // 基础tick间隔
    NormalTicksPerUpdate = 3                      // 正常速度：50ms * 3 = 150ms
    BoostTicksPerUpdate  = 1                      // 加速：50ms * 1 = 50ms (3倍速)
    BoostTimeout         = 150 * time.Millisecond // 加速超时时间
    BoostThreshold       = 2                      // 需要连续按几次才触发加速
    KeyRepeatWindow      = 200 * time.Millisecond // 连续按键的时间窗口
)
```

---

### 2. 加速状态追踪

**文件**: `cmd/snake/main.go`

```go
var (
    tickCount           = 0       // tick计数器
    boosting            = false   // 是否正在加速
    lastBoostKeyTime    time.Time // 上次加速按键时间
    lastDirKeyTime      time.Time // 上次按方向键的时间
    lastDirKeyDir       Point     // 上次按的方向
    consecutiveKeyCount = 0       // 连续按同方向键的次数
)
```

---

### 3. 加速检测逻辑

**文件**: `cmd/snake/main.go` (大约第50-70行)

```go
// 检查是否触发加速（需要连续快速按键）
checkBoostKey := func(inputDir Point) {
    now := time.Now()

    // 检查是否是连续按同方向键
    if inputDir == lastDirKeyDir && time.Since(lastDirKeyTime) < keyRepeatWindow {
        consecutiveKeyCount++  // 计数+1
    } else {
        // 方向变了或者间隔太长，重置计数
        consecutiveKeyCount = 1
    }

    lastDirKeyDir = inputDir
    lastDirKeyTime = now

    // 达到阈值后触发加速
    if consecutiveKeyCount >= boostThreshold && inputDir == game.Direction {
        boosting = true
        lastBoostKeyTime = now
    }
}
```

---

### 4. 输入处理

**文件**: `cmd/snake/main.go` (大约第80-110行)

```go
case inputEvent := <-inputChan:
    // 解析方向输入
    if inputDir, isValid := input.ParseDirection(inputEvent); isValid {
        dirChanged := g.SetDirection(inputDir)

        if dirChanged {
            // 方向改变了，重置加速
            consecutiveKeyCount = 1
            lastDirKeyDir = inputDir
            lastDirKeyTime = time.Now()
            boosting = false  // ❌ 停止加速
        } else {
            // 按下的是当前方向，检查是否触发加速
            checkBoostKey(inputDir)  // ✅ 可能触发加速
        }
    }
```

---

### 5. 加速超时检测

**文件**: `cmd/snake/main.go` (大约第115-120行)

```go
case <-ticker.C:
    // 检查加速是否超时
    if boosting && time.Since(lastBoostKeyTime) > config.BoostTimeout {
        boosting = false  // 超时，停止加速
    }
```

---

### 6. 根据加速调整更新频率

**文件**: `cmd/snake/main.go` (大约第120-130行)

```go
tickCount++

// 根据是否加速决定更新频率
ticksNeeded := config.NormalTicksPerUpdate  // 默认3
if boosting {
    ticksNeeded = config.BoostTicksPerUpdate  // 加速时为1
}

if tickCount >= ticksNeeded {
    tickCount = 0
    if !g.GameOver && !g.Paused {
        g.Update()  // 更新游戏状态
    }
    render.Render(g, boosting)  // 渲染，传递boosting状态
}
```

---

### 7. 视觉反馈

**文件**: `pkg/renderer/terminal.go` (大约第105-110行)

```go
// 根据是否加速显示不同的header
if boosting {
    r.buffer.WriteString(fmt.Sprintf(
        "  Score: %d  |  吃豆速度: %.2f 个/秒  |  已吃: %d 个  |  🚀 BOOST!\n\n",
        g.Score, g.GetEatingSpeed(), g.FoodEaten))
} else {
    r.buffer.WriteString(fmt.Sprintf(
        "  Score: %d  |  吃豆速度: %.2f 个/秒  |  已吃: %d 个\n\n",
        g.Score, g.GetEatingSpeed(), g.FoodEaten))
}
```

---

## 工作流程图

```
用户按方向键
    ↓
ParseDirection() 解析方向
    ↓
SetDirection() 设置方向
    ↓
方向改变？
├─ 是 → 重置加速状态 (boosting = false)
└─ 否 → checkBoostKey()
          ↓
      连续按键计数++
          ↓
      达到阈值(2次)？
      ├─ 是 → boosting = true ✅
      └─ 否 → 继续等待
          ↓
    Ticker触发
          ↓
    检查超时(150ms)
      ├─ 超时 → boosting = false
      └─ 未超时 → 保持加速
          ↓
    根据boosting决定更新频率
      ├─ 加速 → 每1个tick更新 (50ms)
      └─ 正常 → 每3个tick更新 (150ms)
          ↓
    Update() 更新游戏
          ↓
    Render() 渲染画面（显示🚀）
```

---

## 触发加速的条件

1. ✅ 连续快速按**同一个方向**键
2. ✅ 按键间隔 < 200ms (`KeyRepeatWindow`)
3. ✅ 连续次数 >= 2次 (`BoostThreshold`)
4. ✅ 按的方向必须是**当前移动方向**

---

## 加速效果

| 状态 | Tick间隔 | 更新频率 | 速度 |
|------|---------|---------|------|
| 正常 | 50ms × 3 = 150ms | 6.67次/秒 | 1x |
| 加速🚀 | 50ms × 1 = 50ms | 20次/秒 | **3x** |

---

## 停止加速的条件

1. ❌ 改变方向
2. ❌ 停止按键 > 150ms (`BoostTimeout`)
3. ❌ 游戏暂停
4. ❌ 游戏结束

---

## 相关文件

| 文件 | 作用 |
|------|------|
| `pkg/config/config.go` | 加速相关常量配置 |
| `cmd/snake/main.go` | 加速逻辑主实现 |
| `pkg/input/keyboard.go` | 键盘输入解析 |
| `pkg/renderer/terminal.go` | 加速视觉提示 |

---

## 如何修改加速参数

### 调整加速倍率

编辑 `pkg/config/config.go`:

```go
const (
    NormalTicksPerUpdate = 4  // 更慢：50ms * 4 = 200ms
    BoostTicksPerUpdate  = 1  // 加速：50ms * 1 = 50ms
    // 倍率 = 4 / 1 = 4x 加速
)
```

### 调整触发难度

```go
const (
    BoostThreshold  = 3   // 需要连续按3次才触发（更难）
    KeyRepeatWindow = 150 * time.Millisecond  // 更短窗口（更难）
)
```

### 调整加速持续时间

```go
const (
    BoostTimeout = 300 * time.Millisecond  // 停按后300ms才失效（更长）
)
```

---

## 总结

加速功能通过以下机制实现：

1. 🎮 **检测连续按键** - 追踪同方向按键
2. ⏱️ **时间窗口判断** - 200ms内的按键才算连续
3. 📊 **计数达阈值** - 2次以上触发加速
4. ⚡ **调整更新频率** - 3倍速度
5. ⏰ **超时自动停止** - 150ms不按就失效
6. 🎨 **视觉反馈** - 显示🚀图标

整个系统在 `cmd/snake/main.go` 的主游戏循环中实现！🚀
