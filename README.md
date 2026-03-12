# Pause (v1)

Pause 是一个跨平台（Win/macOS/Linux）休息提醒应用，当前以 macOS 版本为主完成度最高。

v1 采用「全屏不可操作遮罩」作为强制休息机制，不做系统锁屏。

## 当前能力

### 核心提醒

- 护眼提醒：默认 `20 分钟 -> 20 秒`（可自定义）。
- 站立提醒：默认 `1 小时 -> 5 分钟`（可自定义）。
- 冲突合并：同一分钟内触发的护眼/站立提醒会合并为一次休息，时长取最大值。
- 休息会话状态机：`resting -> completed / skipped`。

### 强制休息

- macOS 原生多屏全屏遮罩（每个屏幕都覆盖）。
- 支持淡入淡出。
- 可配置是否允许“跳过”。
- 休息结束可播放提示音。

### 交互与托盘

- 菜单栏常驻图标（可选显示倒计时）。
- Popover 快捷操作：暂停、恢复、立即休息、打开主界面、更多菜单（关于/退出）。
- 主界面中英双语（支持“跟随系统”）。

### 配置与持久化

- 本地 JSON 配置持久化（默认路径：`~/.pause/settings.json`）。
- 支持开机启动（macOS LaunchAgents）。
- 计时模式：`idle_pause` / `real_time`。

## 已实现但当前 UI 隐藏的能力

以下能力后端已支持，当前主界面暂未展示入口：

- 全局提醒总开关 `globalEnabled`
  - 关闭后停止计时和触发提醒；恢复后重建调度。
  - 可通过 `UpdateSettings({ globalEnabled: false })` 调用。
- 自定义临时暂停时长
  - `Pause(mode="temporary", durationSec)` 支持任意秒数。
  - `durationSec=0` 时默认回退为 `15 分钟`。
  - 当前 UI 已隐藏“暂停 30 分钟”入口，但 status/action 和后端能力仍在。

## App API（Wails Bindings）

- `GetSettings() -> Settings`
- `UpdateSettings(patch) -> Settings`
- `GetRuntimeState() -> RuntimeState`
- `Pause(mode, durationSec) -> RuntimeState`
- `Resume() -> RuntimeState`
- `SkipCurrentBreak() -> RuntimeState`
- `StartBreakNow() -> RuntimeState`

## 配置结构（v1）

```json
{
  "globalEnabled": true,
  "eye": { "enabled": true, "intervalSec": 1200, "breakSec": 20 },
  "stand": { "enabled": true, "intervalSec": 3600, "breakSec": 300 },
  "enforcement": { "overlayEnabled": true, "overlaySkipAllowed": true },
  "sound": { "enabled": true, "volume": 70 },
  "timer": { "mode": "idle_pause", "idlePauseThresholdSec": 300 },
  "ui": { "showTrayCountdown": true, "language": "auto" },
  "startup": { "launchAtLogin": false }
}
```

## 本地开发

### 1) 安装前端依赖

```bash
npm --prefix frontend install
```

### 2) 构建前端静态资源

```bash
npm --prefix frontend run build
```

### 3) 运行 Wails 版本（macOS 主流程）

```bash
go run -tags wails,dev .
```

### 4) 运行无 UI 后端循环（调度/状态调试）

```bash
go run .
```

### 5) 测试

```bash
go test -tags wails ./...
```

## 平台状态

- macOS：已实现主要能力（托盘、遮罩、通知、提示音、开机启动、空闲检测）。
- Windows/Linux：当前为适配层占位，待补齐原生实现。

## 备注

- `main_wails.go` 为 Wails 入口（`wails` build tag）。
- 主窗口关闭后应用继续驻留在状态栏运行。
