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

状态：已完成

结果：

- 新增 `internal/platform/windows/winrt_native.go`
- `GetNotificationCapability()` 已改为原生 WinRT 查询，不再通过 PowerShell 读取 `ToastNotifier.Setting`
- `OpenNotificationSettings()` 已改为原生 `ShellExecuteW` 打开 `ms-settings:notifications`
- adapter 层新增可注入入口，便于后续测试与替换
- 原有 PowerShell toast 发送链路暂时保留到里程碑 2 再统一清理

验收点：

- Windows 主界面首次打开不再因为通知能力查询拉起终端窗口
- `GetNotificationCapability()` 与 `OpenNotificationSettings()` 不再走 `powershell.exe`

验证记录：

- `GOOS=windows GOARCH=amd64 go test -c ./internal/platform/windows`
- `GOCACHE=$(pwd)/.cache/go-build GOOS=windows GOARCH=amd64 go build ./...`

备注：

- 当前代码中仍保留 PowerShell 相关 helper，是为了支撑里程碑 2 之前的 toast 发送主链路；它们已经不再被通知能力查询和系统设置跳转调用。

### 里程碑 2：去掉发送通知中的 PowerShell

状态：已完成

结果：

- `ShowReminder()` 的 WinRT toast 主路径已改为原生调用
- toast XML 改为在 Go 内构建并通过 `Windows.Data.Xml.Dom.XmlDocument` 加载
- `ToastNotificationManager` / `ToastNotification` 已改为直接通过 WinRT activation factory 调用
- 原有 balloon notification fallback 完全保留
- `internal/platform/windows` 中通知相关的 PowerShell helper 已清理完成

验收点：

- `ShowReminder()` 的 toast 主路径不再依赖 `powershell.exe`
- fallback 行为保持不变

验证记录：

- `rg -n "powershell|runPowerShell|escapePowerShell|Start-Process|ToastNotificationManager\\]::CreateToastNotifier|New-Object Windows.Data.Xml.Dom.XmlDocument" internal/platform/windows -g '*.go'`
- `GOCACHE=$(pwd)/.cache/go-build GOOS=windows GOARCH=amd64 go test -c ./internal/platform/windows`
- `GOCACHE=$(pwd)/.cache/go-build GOOS=windows GOARCH=amd64 go build ./...`

### 里程碑 3：回归验证与文档收口

状态：已完成

结果：

- 已更新 `docs/notification-logic.md` 中的 Windows 实现描述
- 已补充 Windows adapter 测试，覆盖能力映射、系统设置跳转委托、toast XML 转义
- 已完成最后一轮 Windows 目标编译验证

验收点：

- Windows 通知相关 PowerShell 依赖清理完成
- 文档与实现一致

验证记录：

- `GOCACHE=$(pwd)/.cache/go-build GOOS=windows GOARCH=amd64 go test -c ./internal/platform/windows`
- `GOCACHE=$(pwd)/.cache/go-build GOOS=windows GOARCH=amd64 go build ./...`

最终结论：

- Windows 通知相关三条链路已经全部脱离 PowerShell：
  - 通知能力查询
  - toast 发送
  - 系统通知设置跳转
- 当前仍保留的降级路径只有 balloon notification fallback，这属于产品保底方案，不再属于脚本桥接。

## 当前实现边界

本次改造只处理 Windows 平台实现，不处理：

- 前端产品态文案调整
- reminder 保存/启用的交互策略调整
- macOS 通知实现
