# Pause 通知说明

最后更新：2026-04-23

本文档只描述当前实现。

## 1. 当前模型

Pause 的通知链路是：

- runtime 负责决定什么时候发通知
- platform 负责执行单次系统通知发送
- frontend 负责在启用 notify reminder 前做能力预检

通知只用于 `notify` 类型 reminder。

`rest` 类型 reminder 会进入 break session；`notify` 类型 reminder 只发系统通知，不进入休息会话。

## 2. 运行时行为

运行时流程：

1. engine 每秒 tick
2. 调度器命中后生成事件
3. runtime 把事件拆成 `rest` 和 `notify`
4. `notify` 事件直接发送系统通知

当前约束：

- 通知任务绑定到 engine 的运行时 `ctx`
- `Engine.Stop()` 会取消运行时上下文
- `Engine.Stop()` 会等待已发起通知任务结束
- 通知发送有轻量并发上限
- 超限通知会被丢弃并写日志

关键代码：

- `internal/backend/runtime/engine/engine_tick.go`
- `internal/backend/runtime/engine/engine_notifications.go`
- `internal/platform/darwin/adapters.go`
- `internal/platform/windows/adapters.go`

## 3. 后端能力模型

后端抽象：

- `NotificationPermissionState`
- `NotificationCapability`
- `NotificationCapabilityProvider`
- `Notifier`

状态枚举：

- `authorized`
- `not_determined`
- `denied`
- `restricted`
- `unknown`

能力字段：

- `canRequest`
- `canOpenSettings`
- `reason`

## 4. 前端产品态

前端统一映射成三态：

- `pending`
- `available`
- `unavailable`

映射规则：

- `authorized` -> `available`
- `not_determined` -> `pending`
- `denied / restricted / unknown` -> `unavailable`

这层三态是产品模型，不要求和不同平台的原生状态机一一对应。

## 5. 前端交互规则

只在“会让 notify reminder 生效”的动作前预检通知能力：

- 新建 `notify` reminder
- 把已有 `notify` reminder 从关闭切为开启
- 把已启用 reminder 从 `rest` 改成 `notify`

不会触发预检：

- 普通字段编辑
- 关闭 reminder
- 与 notify 生效无关的保存动作

三态下行为：

- `available`
  直接提交
- `pending`
  发起 `RequestNotificationPermission()`，本次保存中止
- `unavailable`
  提示用户通知不可用，并提供打开系统设置入口，本次保存中止

列表标签：

- `notify + enabled + 非 available`
  会显示状态标签
- `pending`
  显示“待授权”
- `unavailable`
  显示“通知关闭”

## 6. 刷新时机

前端当前会在这些时机刷新通知能力：

- 提醒页初始化
- 应用窗口 focus
- 页面 `visibilitychange` 回到可见
- 用户点击主界面
- 授权请求返回 `authorized`

## 7. 平台实现

### macOS

能力：

- 查询授权状态
- 请求授权
- 打开系统通知设置
- 发送通知

实现：

- 使用 `UNUserNotificationCenter`
- 发送前安装 notification delegate
- 未授权时发送直接失败，不做 fallback

### Windows

能力：

- 查询通知能力
- 发送 toast
- 打开系统通知设置

实现：

- 使用 WinRT toast
- 通知设置通过 `ms-settings:notifications`
- `RequestNotificationPermission()` 只返回当前能力，不弹系统授权对话框

## 8. 开发态约束

本地 `dev` 构建默认禁用通知能力。

原因：

- 开发态进程身份和正式打包态不同
- 尤其在 macOS 下，非 `.app` 进程的通知行为不稳定

开发态建议：

- 不在 `go run -tags wails,dev .` 下验收真实通知
- 真实通知行为用打包产物验证

显式禁用方式：

```bash
PAUSE_DISABLE_NOTIFICATION_CAPABILITY=1 go run -tags wails,dev .
```

## 9. 日志

runtime 负责业务日志，平台层只保留关键失败日志。

当前相关日志：

- `reminder.notification_sent`
- `reminder.notification_err`
- `reminder.notification_dropped`
- `darwin.notification.* failed`
- `windows.notification.capability_lookup_failed`
- `windows.notification.settings_open_failed`

## 10. 边界

- 提醒保存接口不直接申请权限
- 不为平台差异强行伪造一致状态机
- runtime 负责通知任务生命周期，platform 只负责单次发送
