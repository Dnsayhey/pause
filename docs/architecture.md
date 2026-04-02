# Pause 架构与结构说明

最后更新：2026-04-02

本文档是 Pause 当前代码结构、职责边界、运行链路与维护约束的单一说明入口。  
README 只保留项目入口与常用命令；涉及结构、架构、数据边界、Wails API、平台职责时，以本文档为准。

## 1. 当前架构目标

Pause 当前采用“接口层 + 后端分层 + 桌面壳 + 平台能力”的结构：

- `internal/app`：对外接口层与应用编排层
- `internal/backend`：业务用例、运行时、存储与组装根
- `internal/desktop`：桌面 UI 外壳能力
- `internal/platform`：系统能力提供层

核心目标不是追求教科书式目录命名，而是保证下面三点长期成立：

1. 前端和宿主只看到稳定 API，不感知 backend 内部模型调整。
2. 业务规则和运行时状态机集中在 backend，不散落到 app / desktop / platform。
3. 原生桌面能力与系统能力分开演进，避免修改 UI 行为时牵动平台底层实现。

## 2. 顶层结构

```text
.
├── frontend/                 # React 前端
├── docs/                     # 架构、打包、重构文档
├── internal/
│   ├── app/                  # Wails 绑定、App 生命周期、DTO、桌面编排
│   ├── backend/              # 核心后端分层
│   ├── desktop/              # 状态栏、遮罩、窗口行为
│   ├── platform/             # idle/lock/notifier/notification capability/sound/startup 等系统能力
│   ├── entry/desktop/        # 桌面入口装配
│   ├── logx/                 # 日志
│   ├── meta/                 # 版本、Bundle ID 等元信息
│   └── paths/                # 配置/日志等路径解析
├── main.go                   # Headless 入口
└── main_wails.go             # Wails 入口
```

## 3. 分层职责

### 3.1 `internal/app`

`app` 是宿主可见的后端接口层，负责：

- `App` 生命周期
- Wails 绑定 API
- app 自有 DTO 与 DTO 映射
- 桌面壳行为编排
- 平台主题/语言与 runtime state 的最终装饰

`app` 不负责：

- 直接实现业务规则
- 直接访问存储
- 直接依赖具体 runtime engine 实现
- 直接实现状态栏/遮罩原生细节

当前文件分组约定：

- `app_api_*`：Wails 绑定 API、对外 DTO、DTO <-> backend 转换
- `app_desktop_*`：桌面控制器、状态栏、overlay 编排
- `app_platform_*`：主题/语言探测、平台信息与平台特定 runtime state 装饰
- `app_lifecycle_*`：窗口关闭、退出等宿主生命周期钩子
- `app_run_*`：Wails / headless 启动入口

### 3.2 `internal/backend`

`backend` 是核心业务层，当前分为：

```text
internal/backend/
  adapters/     # ports 的实现适配器
  bootstrap/    # 组装根
  domain/       # 领域模型与无 IO 规则
  ports/        # usecase/runtime 依赖契约
  runtime/      # 引擎、调度器、会话与运行态
  storage/      # 持久化
  usecase/      # 用例服务
```

职责解释：

- `domain/`
  - 唯一领域模型来源
  - 不做 IO
- `usecase/`
  - 业务流程编排
  - 通过 `ports` 依赖外部能力
- `runtime/`
  - 时间推进、提醒调度、break session 状态机
  - 不定义对外 API 形状
- `storage/`
  - `historydb`、`settingsjson` 等具体持久化实现
- `adapters/`
  - 把 `storage` 或其他实现接到 `ports`
- `bootstrap/`
  - 把 platform、storage、usecase、runtime 装配成可运行对象

### 3.3 `internal/desktop`

`desktop` 是桌面 UI 外壳层，负责：

- 状态栏 / 托盘控制器
- break overlay 控制器
- 主窗口显示/隐藏/行为配置
- 各平台原生桌面桥接

`desktop` 不负责：

- 业务决策
- 提醒排序、下一次休息选择、文案本地化决策
- 运行时状态计算

可以把它理解为：

- `app` 决定“显示什么、事件发生后调用什么”
- `desktop` 决定“怎么显示、怎么把原生事件回调回来”

### 3.4 `internal/platform`

`platform` 是系统能力提供层，负责：

- 空闲检测
- 锁屏状态检测
- 系统通知
- 提示音
- 开机启动

它当前通过 `NewAdapters()` 返回一组 backend 可消费的能力适配器：

- `IdleProvider`
- `LockStateProvider`
- `Notifier`
- `NotificationCapabilityProvider`
- `SoundPlayer`
- `StartupManager`

`platform` 不负责：

- 桌面 UI
- Wails API
- 业务编排

## 4. 关键依赖方向

当前推荐依赖方向如下：

```text
frontend
  -> app
     -> backend/bootstrap
        -> backend/usecase/runtime/storage/adapters

app
  -> desktop

backend/bootstrap
  -> platform
```

更具体地说：

1. `app` 可以依赖 `backend/bootstrap`、`backend/domain`、`desktop`
2. `app` 不应直接依赖 `backend/storage/*`
3. `app` 不应直接依赖 `backend/runtime/engine` 具体实现
4. `desktop` 不应依赖 `backend`
5. `platform` 不应依赖 `app` 或 `desktop`
6. `backend/usecase` 不应依赖具体 `storage/*` 与 `platform/*`

## 5. 运行链路

### 5.1 启动链路

```text
main / main_wails
  -> app.NewApp()
     -> backend/bootstrap.NewRuntime()
        -> settingsjson.OpenStore()
        -> historydb.OpenStore()
        -> platform.NewAdapters()
        -> usecase / runtime 组装
```

### 5.2 Wails API 链路

```text
frontend
  -> window.go.app.App.*
     -> internal/app/app_api_*.go
        -> backend/usecase / runtime
        -> app 自有 DTO 返回前端
```

### 5.3 桌面状态栏 / 遮罩链路

```text
runtime state
  -> app_desktop_* 计算视图与动作
     -> desktop.StatusBarController / BreakOverlayController
        -> 平台原生桌面桥接
```

## 6. 数据与持久化

### 6.1 `settings.json`

保存非提醒设置，当前结构：

```json
{
  "enforcement": { "overlaySkipAllowed": true },
  "sound": { "enabled": true, "volume": 70 },
  "timer": { "mode": "idle_pause", "idlePauseThresholdSec": 60 },
  "ui": { "showTrayCountdown": true, "language": "auto", "theme": "auto" }
}
```

说明：

- 全局调度开关不再持久化到 `settings.json`。
- `Pause()/Resume()` 仅影响运行时计时推进（暂停时冻结进度，恢复后继续）。
- 运行时开关状态通过 `RuntimeState.globalEnabled` 返回给前端/状态栏展示。

### 6.2 `history.db`

当前承担两类数据：

- `reminders`：提醒规则
- `break_sessions` / `break_session_reminders`：休息会话历史

当前阶段尚未引入 schema migration 系统。  
这意味着：

1. 现阶段默认把 schema 视为稳定结构。
2. 真正需要修改既有表结构时，再在那次迭代中引入最小可用 migration 方案。

## 7. 当前对外 API（Wails 绑定）

`internal/app` 当前对前端暴露的主要 API：

- `GetSettings() -> Settings`
- `UpdateSettings(patch) -> Settings`
- `GetReminders() -> []ReminderConfig`
- `CreateReminder(input) -> []ReminderConfig`
- `UpdateReminder(patch) -> []ReminderConfig`
- `DeleteReminder(reminderID) -> []ReminderConfig`
- `GetRuntimeState() -> RuntimeState`
- `Pause() -> RuntimeState`
- `Resume() -> RuntimeState`
- `PauseReminder(reminderID) -> RuntimeState`
- `ResumeReminder(reminderID) -> RuntimeState`
- `SkipCurrentBreak() -> RuntimeState`
- `StartBreakNow() -> RuntimeState`
- `StartBreakNowForReason(reminderID) -> RuntimeState`
- `GetLaunchAtLogin() -> bool`
- `SetLaunchAtLogin(enabled) -> bool`
- `GetNotificationCapability() -> NotificationCapability`
- `RequestNotificationPermission() -> NotificationCapability`
- `OpenNotificationSettings() -> void`
- `GetPlatformInfo() -> PlatformInfo`
- `GetAnalyticsWeeklyStats(fromSec, toSec) -> AnalyticsWeeklyStats`
- `GetAnalyticsSummary(fromSec, toSec) -> AnalyticsSummary`
- `GetAnalyticsTrendByDay(fromSec, toSec) -> AnalyticsTrend`
- `GetAnalyticsBreakTypeDistribution(fromSec, toSec) -> AnalyticsBreakTypeDistribution`

说明：

- 这些返回值已经切换为 `internal/app` 自有 DTO
- frontend 不应依赖 backend 内部类型结构

## 8. Build Tags 与平台差异

当前 `app` / `desktop` 中有较多按平台与 Wails 编译条件划分的文件：

- `*wails.go`
- `*darwin_wails.go`
- `*windows_wails.go`
- `*default.go`

这样做的原因是：

1. 避免在运行时分支里塞过多平台判断
2. 让平台特定逻辑显式可见
3. 降低无关平台改动互相影响的风险

## 9. 当前已知边界约定

### 9.1 `app` 与 `desktop`

这是当前最容易让新人困惑的一条边界。  
明确规则如下：

- `app` 负责 desktop view-model 与 desktop event orchestration
- `desktop` 负责 native desktop implementation

所以：

- “状态栏应该显示什么”在 `app`
- “状态栏怎么显示到系统 UI”在 `desktop`

### 9.2 `app` 与 `platform`

当前已经明确：

- `app` 不直接拿系统能力
- `platform` 通过 `bootstrap` 接入 `backend`

这条边界比早期已经清晰很多，应继续保持。

## 10. 文档分工

- 文档总索引：`docs/README.md`
- 架构与代码结构：`docs/architecture.md`
- 打包、发版与更新源：`docs/packaging.md`
- 通知能力与前端策略：`docs/notification-logic.md`
- 规划与迁移备注：`docs/notes/`

README 不再承载详细结构说明，只保留项目入口、文档索引与常用开发命令。
