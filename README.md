# 🐢 海龟汤 - 多人在线推理游戏

基于 Go + WebSocket 的多人在线海龟汤（Situation Puzzle）推理游戏，AI 担任主持人引导玩家进行推理。

## ✨ 功能特性

- 🤖 **AI 主持人**：集成 DeepSeek 大模型，自动生成海龟汤谜题并以"是/否/不重要"回答玩家提问
- 👥 **多人房间**：支持自定义房间 ID，多名玩家可同时加入同一房间进行推理
- 🔍 **联网搜索**：AI 可调用 DuckDuckGo 搜索工具，获取最新信息辅助回答
- 💾 **谜题库**：内置 SQLite 数据库存储谜题，支持随机抽取
- 💬 **实时通信**：基于 WebSocket 的实时消息推送，流畅的多人交互体验
- 🎨 **现代 UI**：深色主题，响应式设计，流畅的流式输出动画

## 🚀 快速开始

### 环境要求

- Go 1.27+
- DeepSeek API Key（[获取地址](https://platform.deepseek.com/)）

### 安装运行

```bash
# 克隆项目
git clone <repo-url>
cd go-situation-puzzle

# 配置 API Key
cp config/config_example.yaml config/config.yaml
# 编辑 config/config.yaml，将 your_deepseek_api_key 替换为你的真实 API Key

# 安装依赖
go mod tidy

# 运行
go run cmd/main.go
```

服务启动后访问 `http://localhost:8080` 即可进入游戏。

### 配置文件

在 `config/config.yaml` 中配置 DeepSeek API Key：

```yaml
deepseek:
  api_key: your_deepseek_api_key
```

⚠️ **注意**：`config/config.yaml` 已加入 `.gitignore`，请勿将真实 API Key 提交到版本库。如需分享配置模板，请使用 `config/config_example.yaml`。

## 🎮 游戏玩法

1. 打开浏览器访问游戏页面
2. 输入昵称和房间 ID（留空则自动创建房间）
3. 点击「加入游戏」，分享房间 ID 给好友
4. 点击「新谜题」，AI 主持人将生成一个海龟汤谜题（汤面）
5. 玩家输入问题向 AI 提问，AI 以"是/否/不重要"回答
6. 通过不断推理，还原故事真相（汤底）

## 📁 项目结构

```
go-situation-puzzle/
├── cmd/main.go            # 程序入口，初始化数据库、Agent、WebSocket 服务
├── config/
│   ├── config.go          # 配置加载（支持多路径查找 + 热重载）
│   ├── config.yaml        # 实际配置文件（已 gitignore）
│   └── config_example.yaml# 配置文件模板
├── agent/
│   ├── agent.go           # AI Agent 核心逻辑（海龟汤主持人）
│   ├── ddg.go             # DuckDuckGo 搜索工具
│   ├── puzzledb.go        # 谜题数据库工具
│   ├── session.go         # 会话管理（内存 + JSONL 持久化）
│   └── store.go           # 会话存储层
├── model/
│   └── model.go           # Puzzle 数据模型（GORM）
├── server/
│   └── server.go          # WebSocket 服务端 + HTTP 路由
├── html/
│   ├── index.html         # 前端页面
│   ├── app.js             # 前端游戏逻辑
│   └── style.css          # 前端样式
├── go.mod
└── go.sum
```

## 🛠 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go |
| AI 框架 | [CloudWeGo Eino](https://github.com/cloudwego/eino) |
| 大模型 | DeepSeek |
| 搜索工具 | DuckDuckGo |
| 数据库 | SQLite (GORM) |
| WebSocket | [coder/websocket](https://github.com/coder/websocket) |
| 配置管理 | Viper |
| 前端 | 原生 HTML/CSS/JS |

## 📄 License

查看 [LICENSE](LICENSE) 文件。