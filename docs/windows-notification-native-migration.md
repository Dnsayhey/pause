# Windows 通知原生化改造记录

更新时间：2026-03-27

## 背景

当前 Windows 通知实现存在一条明显的技术债链路：

- 查询通知能力：通过 `powershell.exe` 调用 WinRT `ToastNotifier.Setting`
- 发送 WinRT toast：通过 `powershell.exe` 拼 XML 并调用 `ToastNotificationManager`
- 打开系统通知设置：通过 `powershell.exe` 执行 `Start-Process ms-settings:notifications`

这套方案功能上可用，但存在两个问题：

1. GUI 进程内拉起 `powershell.exe` 容易引发终端窗口闪烁，首次打开主界面时会被 `focus / visibilitychange` 刷新链路放大成循环弹窗。
2. Windows 通知能力与发送能力过度依赖外部脚本桥接，稳定性和可维护性都偏弱。

## 目标

本次改造的目标是把 Windows 通知相关能力全部收口为原生调用，不再依赖 PowerShell：

- `GetNotificationCapability()`：改为原生 WinRT 查询
- `ShowReminder()` 的 toast 主链路：改为原生 WinRT 发送
- `OpenNotificationSettings()`：改为原生 `ShellExecuteW`

保留项：

- AUMID 设置与注册逻辑保留
- balloon notification fallback 保留
- 前端交互策略本次不改，只修平台实现

## 里程碑

### 里程碑 0：建立执行记录

状态：已完成

结果：

- 新增本执行文档，作为本轮 Windows 通知原生化改造的唯一过程记录。

### 里程碑 1：去掉能力查询与打开设置中的 PowerShell

状态：进行中

计划：

- 新增 Windows 原生 URI 打开封装，替换 `OpenNotificationSettings()`
- 新增 Windows 原生 WinRT 通知能力查询封装，替换 `queryWindowsToastSetting()`
- 移除这两条链路中的 PowerShell 依赖
- 通过 Windows 目标编译验证相关包可编译

验收点：

- Windows 主界面首次打开不再因为通知能力查询拉起终端窗口
- `GetNotificationCapability()` 与 `OpenNotificationSettings()` 不再走 `powershell.exe`

### 里程碑 2：去掉发送通知中的 PowerShell

状态：未开始

计划：

- 新增原生 WinRT toast 构建与发送逻辑
- 保持现有 payload 语义不变
- WinRT toast 失败时继续 fallback 到 balloon notification

验收点：

- `ShowReminder()` 的 toast 主路径不再依赖 `powershell.exe`
- fallback 行为保持不变

### 里程碑 3：回归验证与文档收口

状态：未开始

计划：

- 整理最终实现结构
- 更新 `docs/notification-logic.md` 中 Windows 部分
- 做编译验证与必要测试

验收点：

- Windows 通知相关 PowerShell 依赖清理完成
- 文档与实现一致

## 当前实现边界

本次改造只处理 Windows 平台实现，不处理：

- 前端产品态文案调整
- reminder 保存/启用的交互策略调整
- macOS 通知实现
