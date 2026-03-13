# Pause

Pause 是一个跨平台（macOS / Windows / Linux）的休息提醒应用。当前主线实现以 macOS 为主。

## 当前能力

### 提醒与会话

- 护眼提醒：默认 `20 分钟 -> 20 秒`（可调）。
- 站立提醒：默认 `1 小时 -> 5 分钟`（可调）。
- 同一分钟内的护眼/站立提醒会合并为一次休息，时长取较大值。
- 休息会话状态：`resting -> completed / skipped`。

### 强制休息策略

- macOS 使用原生多屏全屏遮罩进行强制休息。
- 不提供“关闭遮罩”的开关。
- 可配置是否允许休息中“跳过”。
- 遮罩期间会拦截 `Cmd+W` / `Cmd+H`，避免误关闭或隐藏。
- 当配置不允许跳过时，连续快速按两次 `Cmd+Q` 会显示紧急跳过按钮（仍不会直接退出应用）。
- 原生遮罩展示失败时，自动降级为系统通知提醒。
- 休息结束可播放提示音。

### 托盘与主界面

- 菜单栏常驻图标（可选显示倒计时）。
- Popover 快捷操作：暂停、恢复、立即休息、打开主界面、关于、退出。
- 主窗口点击关闭仅隐藏窗口，应用继续驻留在菜单栏运行。
- 主界面支持中英文与“跟随系统”语言。
- 主题设置支持 `auto / light / dark`（当前主要用于遮罩主题）。

### 持久化与启动项

- 配置文件默认路径：
  - macOS / Linux：`~/.pause/settings.json`
  - Windows：`%APPDATA%/Pause/settings.json`（不可用时回退到 `~/.pause/settings.json`）
- 配置文件损坏时自动回退默认值，并备份为 `settings.json.corrupt.<timestamp>.bak`。
- 支持开机启动：
  - macOS 13+：`SMAppService`
  - macOS 10.13 ~ 12：`SMLoginItemSetEnabled`（helper 方案）

## 配置结构

```json
{
  "globalEnabled": true,
  "eye": { "enabled": true, "intervalSec": 1200, "breakSec": 20 },
  "stand": { "enabled": true, "intervalSec": 3600, "breakSec": 300 },
  "enforcement": { "overlaySkipAllowed": true },
  "sound": { "enabled": true, "volume": 70 },
  "timer": { "mode": "idle_pause", "idlePauseThresholdSec": 300 },
  "ui": { "showTrayCountdown": true, "language": "auto", "theme": "auto" }
}
```

## Wails 绑定 API

- `GetSettings() -> Settings`
- `UpdateSettings(patch) -> Settings`
- `GetRuntimeState() -> RuntimeState`
- `GetLaunchAtLogin() -> bool`
- `SetLaunchAtLogin(enabled) -> bool`
- `Pause(mode, durationSec) -> RuntimeState`
- `Resume() -> RuntimeState`
- `SkipCurrentBreak() -> RuntimeState`
- `StartBreakNow() -> RuntimeState`

## 本地开发

### 1) 安装前端依赖

```bash
npm --prefix frontend install
```

### 2) 构建前端资源

```bash
npm --prefix frontend run build
```

### 3) 运行桌面版（Wails）

```bash
go run -tags wails,dev .
```

### 4) 运行无 UI 后端循环

```bash
go run .
```

### 5) 测试

```bash
go test ./...
go test -tags wails ./...
```

## 打包 DMG（macOS）

```bash
./scripts/build-dmg.sh
```

- 产物：`build/bin/Pause.dmg`
- 脚本会优先使用本机 `wails` 命令；如果未安装，会自动回退到：
  - `go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`

## 卸载（macOS）

```bash
./scripts/uninstall-pause.sh
```

会移除应用、用户数据、启动项与相关缓存/偏好设置。

## 平台状态

- macOS：主流程可用（托盘、遮罩、通知、声音、开机启动）。
- Windows / Linux：已有适配层，原生体验待补齐。
