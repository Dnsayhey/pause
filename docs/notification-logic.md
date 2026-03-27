# 通知逻辑现状与当前约定

更新时间：2026-03-27

## 文档定位

这份文档是 Pause 当前通知相关实现的现状说明，也是后续继续调整通知体验时的主参考文档。

本文档记录四类内容：
- 当前提醒通知的发送链路
- 当前跨平台通知能力模型
- 当前前端交互与提示策略
- 2026-03-27 macOS 首次授权异常退出问题的修复结论

说明：
- `docs/macos-notification-discussion.md` 现在只保留历史讨论背景。
- 以本文档为准，不再以旧的“实施方案清单”作为当前状态描述。

## 当前整体结论

截至 2026-03-27，通知相关改动已经形成当前稳定方案：
- 后端提供统一的通知能力查询、请求权限、打开系统设置接口。
- 前端把不同平台的底层状态收敛成三态：`待授权 / 可以通知 / 无法通知`。
- 不在 reminder 的保存接口里做通知能力校验，只在“让 notify reminder 生效”的动作上做前端预检。
- macOS 首次授权在用户点击“拒绝”或进入系统通知设置后，不会再导致 Pause 退出。
- 为排查首次授权异常退出而临时加入的诊断日志已移除，当前只保留必要的失败日志。

## 1. 提醒通知发送链路

运行时行为：
- engine 每秒 tick 一次。
- scheduler 命中后会把事件拆成 `rest` 和 `notify` 两类。
- `notify` 类型不会进入休息会话，而是直接尝试发送系统通知。

当前链路：
- `internal/backend/runtime/engine/engine_tick.go`
- `internal/backend/runtime/engine/engine_notifications.go`
- `internal/platform/darwin/adapters.go`
- `internal/platform/windows/adapters.go`

当前行为：
- 通知标题统一走“有效语言”结果。
- 中文标题为“提醒”。
- 英文标题为“Reminder”。
- 通知内容为命中的提醒名称，多个提醒用 ` · ` 拼接。
- `ShowReminder` 只负责发送，不再承担首次授权申请。

## 2. 当前通知能力模型

后端已经引入跨平台能力抽象：
- `NotificationPermissionState`
- `NotificationCapability`
- `NotificationCapabilityProvider`

底层状态字段：
- `authorized`
- `not_determined`
- `denied`
- `restricted`
- `unknown`

辅助字段：
- `canRequest`
- `canOpenSettings`
- `reason`

相关代码：
- `internal/backend/ports/runtime_dependencies.go`
- `internal/platform/api/platform.go`

### 前端产品层三态

前端不会直接把所有底层状态逐个暴露给用户，而是映射成三态：
- `待授权`
- `可以通知`
- `无法通知`

当前映射规则：
- `authorized` -> `可以通知`
- `not_determined` -> `待授权`
- `denied / restricted / unknown` -> `无法通知`

说明：
- 底层仍保留更细的 `reason / canRequest / canOpenSettings`，供前端决定提示文案和 action。
- Windows 没有与 macOS 完全等价的 per-app 首次授权状态，因此三态是更合适的产品抽象。

## 3. 当前 App 层接口

当前 App 层已暴露：
- `GetNotificationCapability()`
- `RequestNotificationPermission()`
- `OpenNotificationSettings()`

职责划分：
- `GetNotificationCapability`：查询当前通知状态
- `RequestNotificationPermission`：仅在需要时主动请求权限
- `OpenNotificationSettings`：跳转系统通知设置

相关代码：
- `internal/app/app_api_notification_capability.go`
- `internal/app/app_api_types.go`
- `frontend/src/api.ts`

## 4. 当前前端策略

### 4.1 什么时候检查通知状态

当前前端不会在任意 reminder 编辑时检查通知能力，只在“让 notify reminder 生效”的动作上检查。

会触发预检的动作：
- 新建 `notify` 类型 reminder
- 把已有 `notify` reminder 从关闭切到开启
- 把一个已启用 reminder 从 `rest` 改成 `notify` 时

不会触发预检的动作：
- 编辑名称
- 编辑间隔
- 编辑休息时长
- 编辑普通 `rest` reminder
- 关闭 reminder
- 仅保存一个当前还未生效的 notify reminder 配置

说明：
- `CreateReminder` / `UpdateReminder` 本身不做通知权限申请。
- reminder 配置的保存与“当前能不能通知”已经解耦。

### 4.2 三态下的前端行为

当用户尝试新建或启用一个通知型 reminder：

- 若状态为“可以通知”
  - 直接继续当前动作。

- 若状态为“待授权”
  - 先提示“请先授予通知权限”。
  - 然后调用 `RequestNotificationPermission()`。
  - 不等待授权结果。
  - 当前动作直接停止，不继续提交。
  - 用户完成授权后，等待状态刷新，再重新执行新建或启用动作。

- 若状态为“无法通知”
  - 提示“通知权限被关闭”。
  - 提供“打开系统设置”入口。
  - 当前动作不继续提交。

相关代码：
- `frontend/src/hooks/useSettings.ts`
- `frontend/src/App.tsx`

### 4.3 提醒卡片上的异常标识

当 reminder 同时满足：
- `reminderType=notify`
- `enabled=true`
- 当前通知状态不是“可以通知”

则在 reminder 标题右侧显示一个小状态标签，用来提示“这个提醒当前无法正常通知”。

当前行为：
- `待授权` 显示为 `待授权`
- `无法通知` 显示为 `通知关闭`
- 标签文案与 `待授权 / 无法通知` 使用不同 tooltip 文案。
- 点击标签时：
  - `待授权` 会再次发起授权请求，并弹出通知权限相关 toast
  - `通知关闭` 会弹出带“打开系统设置” action 的 toast

相关代码：
- `frontend/src/components/ReminderCard.tsx`
- `frontend/src/pages/RemindersPage.tsx`

### 4.4 当前提示呈现方式

通知相关提示目前以 toast 为主：
- 位置：主界面顶部居中。
- 普通通知提示默认 5 秒自动收起。
- 手动关闭后，同 key 的后续提示仍可再次出现。
- 带 action 的 toast 会把提示文本和 action 保持在同一行，action 以普通下划线文本展示。

当前通知相关 toast：
- `notification-prompt`：通知权限相关提示，默认 5 秒自动收起。
- `runtime-error`：运行时错误 toast，当前仍为持久展示，直到显式清除。

说明：
- 引导用户重新加载应用的启动级错误，仍保留 `InlineError` 兜底，不属于通知权限提示链路本身。

相关代码：
- `frontend/src/components/ui/ToastProvider.tsx`
- `frontend/src/App.tsx`
- `frontend/src/components/ui/InlineError.tsx`

### 4.5 前端刷新通知状态的时机

当前会在以下时机刷新通知状态：
- 提醒页首次加载时
- 应用窗口重新获得焦点时
- 页面重新可见时
- `RequestNotificationPermission()` 明确返回 `authorized` 时
- 用户点击主界面任意位置时

## 5. 当前 macOS 实现

macOS 当前基于 `UNUserNotificationCenter`。

已实现能力：
- 查询授权状态
- 请求授权
- 打开系统通知设置
- 安装 `UNUserNotificationCenterDelegate`
- 接管普通通知点击回调，但回调中不主动做窗口展示/激活动作

当前行为要点：
- 只有授权允许时才真正发送业务通知。
- 请求权限与真正发通知已经拆开。
- `UNUserNotificationCenterDelegate` 当前只在真正发送通知前安装，不再在“请求权限”时安装。

相关代码：
- `internal/platform/darwin/notification_user_cgo.go`
- `internal/platform/darwin/notification_user_callbacks_cgo.go`
- `internal/platform/darwin/notification_user_stub.go`
- `internal/platform/darwin/adapters.go`

### 5.1 2026-03-27 macOS 首次授权异常退出问题

已修复问题：
- 首次安装后启用 notify reminder
- 系统弹出通知权限授权
- 用户点击“拒绝”或进入系统通知设置后
- Pause 会直接退出

这次修复后的结论：
- 根因不在前端，也不在 `app.Quit()` / `Shutdown()` 链路。
- 问题集中在旧的“同步 cgo + semaphore 等待授权结果”桥接实现。
- 当 macOS 在 deny 场景下返回授权结果时，旧实现会导致进程以 `exit(2)` 退出。
- 现在已经改为“Objective-C 异步回调 -> Go channel 等待”的桥接方式，问题消失。

### 5.2 修复后的关键观察

修复后，从 `app.log` 和 macOS unified log 可以观察到：
- `RequestNotificationPermission()` 可以完整走完：
  - `started`
  - `callback`
  - `completed`
- deny 场景下最终会稳定收敛到：
  - `permission_state=denied`
  - `can_request=false`
  - `can_open_settings=true`
- 打开系统通知设置后，Pause 进程不会退出。
- 用户后来重新允许通知后，状态会正常刷新为 `authorized`。

一个需要记录的实现细节：
- macOS unified log 在用户点击“拒绝”时仍会出现：
  - `Requested authorization [ didGrant: 0 hasError: 1 hasCompletionHandler: 1 ]`
- 这不应再被当作“异常崩溃信号”。
- 当前实现已经把 deny 场景下的这类返回正规化为 `denied` 状态处理。

### 5.3 当前日志策略

此前为排查 macOS 首次授权异常退出，曾临时保留一批通知诊断日志。

当前状态：
- 这批 `started/completed/callback` 级别的诊断日志已经移除。
- 当前只保留必要的失败日志，便于在真正出错时定位问题。

## 6. 当前 Windows 实现

Windows 当前基于 `ToastNotifier.Setting` 获取通知能力，并保留原有 toast 发送路径。

已实现能力：
- 查询通知能力
- 打开系统通知设置 `ms-settings:notifications`
- `RequestNotificationPermission()` 返回当前能力，不触发类似 macOS 的首次授权弹窗

说明：
- Windows 仍然更接近“当前可通知能力查询”，而不是 macOS 风格的 per-app 首次授权状态机。

相关代码：
- `internal/platform/windows/adapters.go`

## 7. 当前构建层处理

macOS 构建脚本当前会在产物 `Info.plist` 中写入：
- `CFBundleIdentifier`
- `LSUIElement=true`

目的：
- 让菜单栏应用在构建产物层面保持 agent app 形态
- 避免 Dock 表现完全依赖运行时策略兜底

相关代码：
- `scripts/build-dmg.sh`

## 8. 当前有效语言来源

当前 App 和 runtime engine 已统一使用平台层语言探测结果作为“有效语言”来源，避免 UI 和通知标题在 `auto` 场景下分叉。

相关代码：
- `internal/platform/preferred_language_darwin_wails.go`
- `internal/platform/preferred_language_windows_wails.go`
- `internal/platform/preferred_language_default.go`
- `internal/app/app_locale.go`
- `internal/backend/runtime/engine/engine_locale.go`

## 9. 当前已知问题与后续观察点

截至 2026-03-27，之前“首次授权后应用退出”的问题已修复。

当前仍值得继续观察的点：
- 普通业务通知点击后是否还需要产品层动作
- macOS 不同系统版本下 deny 场景返回值是否一致
- Windows 是否还要补充更细的“被系统压制但并非完全关闭”状态说明

当前没有阻塞本轮交付的通知级故障。
