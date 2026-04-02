# Pause 通知能力与前端策略

最后更新：2026-04-02

本文档记录 Pause 当前通知能力的实现边界、前端交互策略与平台差异。

只记录当前实现，不记录历史排障过程和临时迁移方案。

## 当前结论

- 通知链路已统一为「运行时发送 + 前端启用前预检」。
- 前端产品态统一为三态：`pending / available / unavailable`（UI 文案：待授权 / 可以通知 / 通知关闭）。
- `CreateReminder / UpdateReminder` 不做权限申请；只在“新建或启用 notify 提醒”时预检。
- macOS 与 Windows 都支持：查询通知能力、请求权限（或能力刷新）、打开系统设置。
- Windows 使用原生 WinRT + ShellExecute 路径，通知能力查询/发送/设置跳转均走原生实现。
- 通知发送成败由 runtime 统一记日志，平台层只保留关键失败日志。

## 端到端链路

运行时行为：
- engine 每秒 tick。
- 调度命中后拆分 `rest` 与 `notify` 事件。
- `notify` 事件直接尝试发送系统通知，不进入休息会话。

关键代码：
- `internal/backend/runtime/engine/engine_tick.go`
- `internal/backend/runtime/engine/engine_notifications.go`
- `internal/platform/darwin/adapters.go`
- `internal/platform/windows/adapters.go`

发送日志：
- 成功：`reminder.notification_sent`
- 失败：`reminder.notification_err`

## 后端能力模型

后端抽象：
- `NotificationPermissionState`
- `NotificationCapability`
- `NotificationCapabilityProvider`

底层状态：
- `authorized`
- `not_determined`
- `denied`
- `restricted`
- `unknown`

能力字段：
- `canRequest`
- `canOpenSettings`
- `reason`

关键代码：
- `internal/backend/ports/runtime_dependencies.go`
- `internal/platform/api/platform.go`
- `internal/platform/fallbacks/notification_capability.go`
- `internal/platform/notification_capability_override.go`

## 前端产品态

前端统一暴露三态：
- `pending`（待授权）
- `available`（可以通知）
- `unavailable`（通知关闭 / 不可用）

映射规则：
- `authorized` -> `available`
- `not_determined` -> `pending`
- 其他状态（`denied / restricted / unknown`）-> `unavailable`

说明：
- Windows 通常不会产生 macOS 等价的 per-app 首次授权 `not_determined` 流程，属于平台差异；前端仍按同一三态模型呈现。

关键代码：
- `frontend/src/hooks/useSettings.ts`

## 前端交互策略

### 何时预检通知能力

只在“让 notify reminder 生效”的动作前预检：
- 新建 `notify` 类型 reminder
- 把已有 `notify` 从关闭切为开启
- 把已启用 reminder 从 `rest` 改为 `notify`

不会触发预检：
- 普通字段编辑（名称、间隔、时长等）
- 关闭 reminder
- 其他不导致 notify 生效的保存动作

### 三态下动作

当用户尝试新建或启用通知提醒：
- `available`：直接继续提交。
- `pending`：弹权限提示 + 发起 `RequestNotificationPermission()`，不依赖结果；本次提交中止。
- `unavailable`：弹“通知关闭”提示，并提供“打开系统设置” action；本次提交中止。

### 列表状态标签

当 reminder 满足：
- `reminderType=notify`
- `enabled=true`
- 当前通知状态不为 `available`

在标题侧显示状态标签：
- `pending` -> `待授权`
- `unavailable` -> `通知关闭`

点击标签行为：
- `pending`：再次发起授权请求并提示
- `unavailable`：提示并提供打开系统设置 action

关键代码：
- `frontend/src/pages/RemindersPage.tsx`
- `frontend/src/hooks/useSettings.ts`

### 刷新时机

当前刷新触发点：
- 提醒页初始化加载
- 应用窗口 `focus`
- 页面 `visibilitychange`（变为可见）
- 用户点击主界面任意位置（`pointerdown capture`）
- 授权请求返回 `authorized` 时立即更新本地状态

关键代码：
- `frontend/src/hooks/useSettings.ts`
- `frontend/src/App.tsx`

## 开发与调试

本地 `dev` 构建默认禁用通知相关能力。

行为说明：

- `go run -tags wails,dev .` 默认会把 `NotificationCapabilityProvider` 替换为禁用版 provider。
- 这样做是为了避免开发态与正式打包态在系统通知身份上的差异，尤其是 macOS 非 `.app` 进程导致的原生框架问题。
- 本地开发默认不验证真实通知行为；真实通知请在打包版中验收。
- 该开关在 `internal/platform` 组装层生效，不在前端做 dev 分支判断。
- 如需显式强制关闭通知能力，也可以设置：

```bash
PAUSE_DISABLE_NOTIFICATION_CAPABILITY=1 go run -tags wails,dev .
```

- 显式开关会继续把 `NotificationCapabilityProvider` 替换为禁用版 provider。
- 三个平台统一返回 `unknown / canRequest=false / canOpenSettings=false` 的降级能力对象。

## 平台实现

### macOS

基础能力（`UNUserNotificationCenter`）：
- 查询授权状态
- 请求授权
- 打开系统通知设置
- 发送通知前安装通知 delegate

行为要点：
- 未授权时发送会失败并返回错误，不做平台 fallback。
- `RequestNotificationPermission()` 为异步桥接到 Go 等待，带超时保护（180s）。
- `GetNotificationCapability()` 当前也使用异步桥接回传状态，避免主线程同步等待异步回调。
- 发送通知前的权限判断在 Go 层复用同一套状态查询结果，再进入原生发送路径。

关键代码：
- `internal/platform/darwin/adapters.go`
- `internal/platform/darwin/notification_user_cgo.go`
- `internal/platform/darwin/notification_user_callbacks_cgo.go`

### Windows

基础能力（原生实现）：
- WinRT `ToastNotifier.Setting` 查询通知能力
- WinRT toast 发送通知
- `ShellExecuteW` 打开 `ms-settings:notifications`

行为要点：
- `RequestNotificationPermission()` 直接返回当前能力（无 macOS 式授权弹窗）。
- 查询 `ToastNotifier.Setting` 若返回 `0x80070490`，按 `Enabled` 处理，避免首次安装误判。
- 安装器写入 `HKCU\\Software\\Classes\\AppUserModelId\\com.pause.app` 下的 `DisplayName/IconUri`，卸载时删除该键，保证通知图标元数据完整。
- 发送路径使用 WinRT toast。

关键代码：
- `internal/platform/windows/adapters.go`
- `internal/platform/windows/winrt_native.go`
- `scripts/windows-installer/project.nsi`

## 日志策略

统一原则：
- runtime 记录业务通知发送成功/失败。
- 平台层只记录关键失败与关键兼容分支（例如 Windows `0x80070490` 兜底）。
- 不保留临时排障型高频诊断日志。

代表日志：
- `reminder.notification_sent`
- `reminder.notification_err`
- `darwin.notification.* failed`
- `windows.notification.capability_lookup_failed`
- `windows.notification.settings_open_failed`

## 非目标与边界

- 不在提醒保存接口里绑定权限校验或权限申请。
- 不为平台差异强行做“假一致”状态机；统一产品态即可。
- 不引入冗余 fallback 掩盖真实失败；失败优先记录日志并显式暴露状态。
