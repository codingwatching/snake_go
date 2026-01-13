# 代码结构说明

## 📦 包架构 (Package Architecture)

项目采用清晰的包结构，遵循 Go 最佳实践：

```
snake_go/
├── cmd/
│   └── snake/
│       └── main.go           # 程序入口，游戏循环编排
├── pkg/
│   ├── game/                 # 核心游戏逻辑
│   │   ├── types.go         # 数据结构定义
│   │   ├── game.go          # 游戏状态管理
│   │   └── food.go          # 豆子相关逻辑
│   ├── renderer/             # 渲染层
│   │   └── terminal.go      # 终端渲染器
│   ├── input/                # 输入处理层
│   │   └── keyboard.go      # 键盘输入管理
│   └── config/               # 配置层
│       └── config.go        # 游戏常量和配置
└── internal/                 # 内部工具（未来使用）
```

## 🎯 包职责划分

### `cmd/snake` - 程序入口
- **职责**: 协调所有组件，运行主游戏循环
- **内容**:
  - 初始化各个子系统（游戏、渲染、输入）
  - 主游戏循环（ticker + select 模式）
  - 加速机制检测
  - 事件分发

### `pkg/game` - 核心游戏逻辑
- **职责**: 游戏状态管理和游戏规则
- **types.go**: 
  - `Point` - 坐标点
  - `FoodType` - 豆子类型枚举
  - `Food` - 豆子结构
  - `Game` - 游戏主结构
- **game.go**:
  - `NewGame()` - 创建新游戏
  - `Update()` - 游戏状态更新
  - `TrySpawnFood()` - 尝试生成豆子
  - `SetDirection()` - 设置方向（带验证）
  - `TogglePause()` - 暂停/继续
  - `GetEatingSpeed()` - 计算吃豆速度
- **food.go**:
  - `GetScore()` - 获取豆子分值
  - `GetDuration()` - 获取留存时间
  - `IsExpired()` - 检查是否过期
  - `GetEmoji()` - 获取图标
  - `GetTimerEmoji()` - 获取倒计时图标

### `pkg/renderer` - 渲染层
- **职责**: 将游戏状态渲染到终端
- **terminal.go**:
  - `NewTerminalRenderer()` - 创建渲染器
  - `Render()` - 渲染游戏画面
  - **性能优化**:
    - 预分配 board 数组
    - 使用 `strings.Builder` 缓冲输出
    - ANSI 转义码快速清屏（替代 `clear` 命令）
    - 单次 stdout 写入

### `pkg/input` - 输入处理层
- **职责**: 处理键盘输入，解析游戏指令
- **keyboard.go**:
  - `NewKeyboardHandler()` - 创建输入处理器
  - `Start()` / `Stop()` - 启动/停止监听
  - `ParseDirection()` - 解析方向键
  - `IsQuit()` / `IsRestart()` / `IsPause()` - 指令判断

### `pkg/config` - 配置层
- **职责**: 集中管理所有游戏常量
- **config.go**:
  - 游戏尺寸 (`Width`, `Height`)
  - 豆子生成参数 (`FoodSpawnInterval`, `MaxFoodsOnBoard`)
  - 速度和加速参数 (`BaseTick`, `BoostTicksPerUpdate`, etc.)
  - 渲染字符 (`CharWall`, `CharHead`, etc.)

## 🎮 游戏特性

### 多样化豆子系统
- 🔴 红色（40分，10秒）- 15%概率
- 🟠 橙色（30分，15秒）- 20%概率
- 🔵 蓝色（20分，18秒）- 25%概率
- 🟣 紫色（10分，20秒）- 35%概率

### 倒计时提示
豆子消失前5秒会显示圆圈数字倒计时（🔴⁵ → 🔴¹）

### 加速机制
连续快速按住当前方向键可触发3倍速度加速 🚀

### 实时统计
- 当前分数
- 吃豆速度（个/秒）
- 已吃豆子总数

## 🚀 性能优化

### 渲染优化
1. **ANSI 转义码清屏** - 替代 `exec.Command("clear")`，速度提升 10x
2. **strings.Builder 缓冲** - 单次写入 stdout，减少系统调用
3. **预分配 board** - 复用二维数组，减少 GC 压力
4. **消除重复代码** - 合并 `render()` 和 `renderWithBoost()`

### 内存优化
- 在 `NewTerminalRenderer()` 中预分配 board
- 复用 `strings.Builder`
- 避免每帧创建新的 map/slice

## 🔧 编译运行

```bash
# 开发运行
go run ./cmd/snake

# 编译当前平台
go build -o snake ./cmd/snake

# 多平台编译
./build.sh
```

## 📊 代码统计

| 包 | 主要功能 | 估计行数 |
|---------|----------|---------|
| `cmd/snake` | 主循环编排 | ~140 |
| `pkg/game` | 游戏逻辑 | ~300 |
| `pkg/renderer` | 渲染 | ~160 |
| `pkg/input` | 输入处理 | ~100 |
| `pkg/config` | 配置 | ~35 |

**总计：约 735 行代码**

## 🏗️ 架构优势

### 1. **关注点分离**
- 游戏逻辑与渲染完全解耦
- 输入处理独立于游戏逻辑
- 配置集中管理

### 2. **可测试性**
- 每个包可独立测试
- 易于编写单元测试
- 可以 mock 渲染器和输入

### 3. **可扩展性**
- 轻松添加新的渲染器（Web、GUI）
- 可以替换输入源（网络、AI）
- 易于添加新的游戏机制

### 4. **可维护性**
- 清晰的包边界
- 单一职责原则
- 易于理解和修改

## 🔮 未来扩展方向

### 已计划
- 高分持久化
- 难度等级选择
- 道具系统（护盾、时间冻结等）
- 障碍物
- 连击评分系统

### 架构支持
- 多渲染器支持（终端、Web、桌面 GUI）
- 多输入源（键盘、游戏手柄、网络）
- 插件系统
- 回放系统
