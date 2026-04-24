# Pause 架构说明

最后更新：2026-04-24

本文档只描述当前实现。

## 1. 总览

Pause 目前由四层组成：

- `internal/app`
  对外接口层。负责 Wails 绑定、DTO 映射、App 生命周期、桌面壳编排。
- `internal/backend`
  核心业务层。负责领域模型、用例、运行时状态机、存储、组装。
- `internal/desktop`
  桌面 UI 外壳。负责状态栏、遮罩、窗口控制和原生桌面桥接。
- `internal/platform`
  系统能力层。负责空闲检测、锁屏检测、通知、提示音、开机启动。

依赖方向：

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

约束：

- `app` 不直接访问 `storage`
- `desktop` 不依赖 `backend`
- `platform` 不依赖 `app` / `desktop`
- `usecase` 不依赖具体平台和具体存储实现

## 2. 目录职责

### `internal/app`

职责：

- Wails 暴露 API
- 对外 DTO 与 DTO 映射
- App 启动、关闭、平台装饰逻辑
- 桌面壳编排

不负责：

- 直接实现业务规则
- 直接操作数据库或配置文件
- 直接实现 runtime 状态机

主要文件分组：

- `app_api_*`：对外 API
- `app_desktop_*`：状态栏、遮罩、桌面交互
- `app_platform_*`：平台主题、语言、平台特定装饰
- `app_run_*`：Wails / headless 入口辅助

### `internal/backend`

当前结构：

```text
internal/backend/
  adapters/
  bootstrap/
  domain/
  ports/
  runtime/
  storage/
  usecase/
```

职责：

- `domain/`
  领域模型和纯规则，无 IO。
- `usecase/`
  用例编排，通过 `ports` 依赖外部能力。
- `runtime/`
  调度器、运行时状态、break session 状态机。
- `storage/`
  `settingsjson`、`historydb` 等持久化实现。
- `adapters/`
  把 storage 或其他实现接到 `ports`。
- `bootstrap/`
  组装 runtime、usecase、platform、storage。

### `internal/desktop`

职责：

- 状态栏 / 托盘控制
- break overlay 控制
- 主窗口显示隐藏
- 原生桌面事件桥接

`desktop` 只管“怎么显示”和“怎么回调”，不做业务决策。

### `internal/platform`

职责：

- `IdleProvider`
- `LockStateProvider`
- `Notifier`
- `NotificationCapabilityProvider`
- `SoundPlayer`
- `StartupManager`

`platform.NewAdapters()` 返回一组 backend 可直接消费的系统能力实现。

## 3. Backend 关键边界

### Settings

settings 现在拆成三类职责：

- `SettingsStoreRepository`
  只负责 `settings.json` 的读取和更新。
- `PlatformSettingsSyncer`
  只负责首次安装时的平台同步动作。
- `StartupManager`
  只负责开机启动状态的读取与设置。

这意味着：

- `settings.json` 只保存应用设置
- 开机启动不存入 `settings.json`
- `GetLaunchAtLogin / SetLaunchAtLogin` 直接反映系统真实状态

### Reminder

Reminder 的约束在 `domain/reminder`：

- 名称不能为空
- 名称不能带前后空格
- `intervalSec > 0`
- `breakSec > 0`
- `reminderType` 只能是 `rest` 或 `notify`
- `CreateInput` 默认 `enabled = true`

storage 层保留防御式校验，但领域规则以 domain 为准。

### Runtime

`runtime/engine` 负责：

- 每秒 tick
- 基于提醒规则推进调度
- 维护 `globalEnabled`
- 管理 break session
- 记录 break history
- 发送通知

当前实现约束：

- break session 结束态只有 `completed` 和 `skipped`
- break history 在锁内完成状态收口与写入参数快照，在锁外提交到 `history.db`

runtime 生命周期：

- `Engine.Start(ctx)` 启动 tick 循环
- `Engine.Stop()` 取消运行时上下文并等待后台任务收敛
- `Runtime.Close()` 会先 `Engine.Stop()` 再关闭 `history.db`
- `app.Shutdown()` 统一走 `Runtime.Close()`

### Notification

通知由 runtime 直接发送：

- 调度命中后把事件拆成 `rest` 和 `notify`
- `notify` 不进入休息会话，直接尝试发系统通知
- 通知发送绑定到 engine 的运行时 `ctx`
- runtime 关闭时会取消并等待已发起的通知任务
- 通知发送有轻量并发上限，超限任务直接丢弃并记日志

## 4. 启动链路

```text
main / main_wails
  -> app.NewApp()
     -> backend/bootstrap.NewRuntime()
        -> settingsjson.OpenStore()
        -> historydb.OpenStore()
        -> platform.NewAdapters()
        -> runtime / usecase 组装
```

首次安装行为：

- `settingsjson.Store.WasCreated()` 为 `true` 时
- 会做一次内置提醒初始化
- 会通过 `PlatformSettingsSyncer` 做一次平台设置同步

## 5. 数据持久化

### `settings.json`

当前结构：

```json
{
  "enforcement": { "overlaySkipAllowed": true },
  "sound": { "enabled": true },
  "timer": { "mode": "idle_pause", "idlePauseThresholdSec": 60 },
  "ui": { "showTrayCountdown": true, "language": "auto", "theme": "auto" }
}
```

说明：

- 不保存 reminder
- 不保存 `globalEnabled`
- 不保存开机启动状态

### `history.db`

当前承担两类数据：

- `reminders`
- `break_sessions` / `break_session_reminders`

其中 `break_sessions.status` 当前只使用：

- `completed`
- `skipped`

analytics 查询也全部基于 `history.db`。

## 6. 对外 API

`internal/app` 当前暴露的主要 API：

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

这些返回值都经过 `internal/app` DTO 映射，前端不应依赖 backend 内部结构。

## 7. 平台差异

### macOS

- 通知：`UNUserNotificationCenter`
- 开机启动：`SMAppService` / Login Items
- 提示音：`afplay`

### Windows

- 通知：WinRT toast
- 开机启动：`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- 通知设置跳转：`ms-settings:notifications`

## 8. 维护原则

- 文档只写当前实现
- 代码改动后优先同步本文档
- 如果某个主题需要展开，放到单独专题文档，不在这里堆历史背景
