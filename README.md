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
- Windows 使用原生全屏置顶遮罩（第一版，覆盖虚拟桌面区域）。
- 不提供“关闭遮罩”的开关。
- 可配置是否允许休息中“跳过”。
- 遮罩期间会拦截 `Cmd+W` / `Cmd+H`，避免误关闭或隐藏。
- 当配置不允许跳过时，连续快速按两次 `Cmd+Q` 会显示紧急跳过按钮（仍不会直接退出应用）。
- 原生遮罩展示失败时，自动降级为系统通知提醒。
- 休息结束可播放提示音。

### 托盘与主界面

- 菜单栏常驻图标（可选显示倒计时）。
- Popover 快捷操作：暂停、恢复、立即休息、打开主界面、关于、退出。
- Windows 提供原生托盘菜单（右键）与主窗口唤起（左键/双击）。
- 主窗口点击关闭仅隐藏窗口，应用继续驻留在菜单栏运行。
- 主界面支持中英文与“跟随系统”语言。
- 主题设置支持 `auto / light / dark`（当前主要用于遮罩主题）。

### 持久化与启动项

- 配置文件默认路径：
  - macOS：`~/Library/Application Support/Pause/settings.json`
  - Linux：`$XDG_CONFIG_HOME/Pause/settings.json`（通常为 `~/.config/Pause/settings.json`）
  - Windows：`%AppData%/Pause/settings.json`
  - 当系统目录不可用时，回退到 `~/.pause/settings.json`
- 提醒配置与会话历史数据库默认路径：
  - macOS：`~/Library/Application Support/Pause/history.db`
  - Linux：`$XDG_CONFIG_HOME/Pause/history.db`（通常为 `~/.config/Pause/history.db`）
  - Windows：`%AppData%/Pause/history.db`
  - 当系统目录不可用时，回退到 `~/.pause/history.db`
- 日志文件默认路径（统一日志）：
  - macOS：`~/Library/Logs/Pause/app.log`
  - Linux：`$XDG_CACHE_HOME/Pause/logs/app.log`（通常为 `~/.cache/Pause/logs/app.log`）
  - Windows：`%LocalAppData%/Pause/logs/app.log`
  - 当系统目录不可用时，回退到 `~/.pause/logs/app.log`
- 日志级别默认 `info`，可通过 `PAUSE_LOG_LEVEL=debug|info|warn|error` 调整。
- 日志文件超过 `2MB` 会自动轮转为 `app.log.1`（仅保留 1 份备份）。
- 配置文件损坏时自动回退默认值，并备份为 `settings.json.corrupt.<timestamp>.bak`。
- `settings.json` 仅持久化非提醒设置；提醒规则（名称、开关、间隔、时长、投递方式）持久化在 `history.db` 的 `reminders` 表。
- 支持开机启动：
  - macOS 13+：`SMAppService`
  - macOS 10.13 ~ 12：`SMLoginItemSetEnabled`（helper 方案）
  - Windows：当前用户 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

## 配置结构（settings.json）

```json
{
  "globalEnabled": true,
  "enforcement": { "overlaySkipAllowed": true },
  "sound": { "enabled": true, "volume": 70 },
  "timer": { "mode": "idle_pause", "idlePauseThresholdSec": 300 },
  "ui": { "showTrayCountdown": true, "language": "auto", "theme": "auto" }
}
```

## 提醒结构（history.db / reminders）

提醒规则不写入 `settings.json`，而是通过 `GetReminders/UpdateReminders` 读写。运行时会由应用编排层组装为统一状态供调度器消费。

## Wails 绑定 API

- `GetSettings() -> Settings`
- `UpdateSettings(patch) -> Settings`
- `GetReminders() -> []ReminderConfig`
- `UpdateReminders(patches) -> []ReminderConfig`
- `GetRuntimeState() -> RuntimeState`
- `GetLaunchAtLogin() -> bool`
- `SetLaunchAtLogin(enabled) -> bool`
- `Pause() -> RuntimeState`
- `Resume() -> RuntimeState`
- `PauseReminder(reason) -> RuntimeState`
- `ResumeReminder(reason) -> RuntimeState`
- `SkipCurrentBreak() -> RuntimeState`
- `StartBreakNow() -> RuntimeState`
- `StartBreakNowForReason(reason) -> RuntimeState`
- `GetReminderWeeklyStats(weekStartSec, weekEndSec) -> WeeklyStats`

## 代码结构（当前）

- `main.go` / `main_wails.go`：根入口（Wails 构建根入口，含嵌入式前端资源）。
- `internal/app/`：应用编排层（App 生命周期、Wails 绑定、桌面交互逻辑）。
- `internal/core/`：纯业务核心（`config / scheduler / session / service`）。
- `internal/desktop/`：桌面能力抽象（状态栏、全屏遮罩、窗口行为）。
- `internal/desktop/macbridge/`：macOS 原生桥接实现（状态栏、全屏遮罩、窗口行为的 Objective-C 桥接）。
- `internal/platform/`：平台 facade 与接口定义。
- `internal/platform/darwin|windows|linux/`：平台能力实现（空闲检测、通知、开机启动、声音）。
- `internal/meta/bundle_id.txt`：应用 Bundle ID 单一来源（脚本与运行时均从这里派生，支持构建时覆盖）。
- `assets/branding/app-icon-1024.png`：应用图标源文件（打包时同步到 `build/appicon.png`）。
- `frontend/`：React 前端界面。

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
- Bundle ID 默认来源：`internal/meta/bundle_id.txt`（可通过环境变量 `APP_BUNDLE_ID` 临时覆盖）
- 图标默认来源：`assets/branding/app-icon-1024.png`（可通过环境变量 `APP_ICON_SOURCE` 临时覆盖）
- 脚本会优先使用本机 `wails` 命令；如果未安装，会自动回退到：
  - `go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`

## 打包 Windows 安装器（x64）

```bash
./scripts/build-windows-installer.sh
```

- 依赖：本机安装 `NSIS`（macOS 可用 `brew install nsis`）。
- 默认产物目录：`build/bin/windows-x64/`（与 macOS 产物分开存放）。
- 脚本会优先使用本机 `wails` 命令；如果未安装，会自动回退到：
  - `go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`
- 默认不加 `-clean`，避免清理掉其它平台已有产物；如需清理再打包：
  - `USE_CLEAN=1 ./scripts/build-windows-installer.sh`
- 可选参数示例：
  - `WINDOWS_WEBVIEW2=browser ./scripts/build-windows-installer.sh`
  - `WINDOWS_OUTPUT_DIR=build/bin/win-release ./scripts/build-windows-installer.sh`
- NSIS 模板默认来自 `scripts/windows-installer/project.nsi`（脚本会在构建前同步到 `build/windows/installer/project.nsi`）；该模板已包含把 `assets/branding/icon.ico` 安装到应用目录。

## 卸载（macOS）

```bash
./scripts/uninstall-pause.sh
```

会移除应用、用户数据、启动项与相关缓存/偏好设置。

## 平台状态

- macOS：主流程可用（托盘、遮罩、通知、声音、开机启动）。
- Windows：第一阶段可用（空闲检测、托盘菜单、通知、提示音、开机启动、原生遮罩第一版）。
  - 通知策略：默认使用原生 Win32 通知；若系统通知接口不可用，再回退到 PowerShell 通知兜底。
- Linux：已有适配层，原生体验待补齐。
