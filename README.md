# Pause

Pause 是一个跨平台（macOS / Windows / Linux）的休息提醒应用。当前 macOS 与 Windows 主流程已基本对齐，Linux 提供适配层并持续补齐原生体验。

## 后端重构跟踪

- 重构计划：`docs/backend-refactor-plan.md`
- 进度日志：`docs/backend-refactor-progress.md`

## 当前能力

### 提醒与会话

- 护眼提醒：默认 `20 分钟 -> 20 秒`（可调）。
- 站立提醒：默认 `1 小时 -> 5 分钟`（可调）。
- 同一分钟内的护眼/站立提醒会合并为一次休息，时长取较大值。
- 休息会话状态：`resting -> completed / skipped`。

### 强制休息策略

- macOS 使用原生多屏全屏遮罩进行强制休息。
- Windows 使用原生全屏置顶遮罩（覆盖虚拟桌面区域）。
- 不提供“关闭遮罩”的开关。
- 可配置是否允许休息中“跳过”。
- 遮罩期间会拦截常见关闭/隐藏快捷键，避免误关闭或隐藏（macOS 重点覆盖 `Cmd+W` / `Cmd+H`）。
- 当配置不允许跳过时，macOS 下连续快速按两次 `Cmd+Q` 会显示紧急跳过按钮（仍不会直接退出应用）。
- 原生遮罩展示失败时，自动降级为系统通知提醒。
- 休息结束可播放提示音。

### 托盘与主界面

- 状态栏 / 托盘常驻图标。
- macOS 使用 Popover 进行快捷操作（暂停、恢复、立即休息、打开主界面等）。
- Windows 使用原生托盘菜单（右键）与主窗口唤起（左键/双击），不使用 Popover。
- 主窗口点击关闭仅隐藏窗口，应用继续驻留在菜单栏运行。
- 主界面为三页结构：`提醒 / 分析 / 设置`。
- 分析页提供趋势图、类型分布图、小时热力图（ECharts），支持按时间范围与指标切换。
- 设置页集中管理语言、主题、开机启动、遮罩可跳过、空闲暂停、提示音等选项。
- 主界面支持中英文与“跟随系统”语言。
- 主题设置支持 `auto / light / dark`，并作用于主界面与休息遮罩。
- 平台外观差异：macOS Popover 与 Windows 原生菜单当前跟随系统外观，不跟随 Pause 主题单独切换。

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
  "timer": { "mode": "idle_pause", "idlePauseThresholdSec": 60 },
  "ui": { "showTrayCountdown": true, "language": "auto", "theme": "auto" }
}
```

## 提醒结构（history.db / reminders）

提醒规则不写入 `settings.json`，而是通过 `GetReminders/UpdateReminder` 读写。运行时会由应用编排层组装为统一状态供调度器消费。

## Wails 绑定 API

- `GetSettings() -> Settings`
- `UpdateSettings(patch) -> Settings`
- `GetReminders() -> []ReminderConfig`
- `UpdateReminder(patch) -> []ReminderConfig`
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
- `GetAnalyticsWeeklyStats(fromSec, toSec) -> AnalyticsWeeklyStats`
- `GetAnalyticsSummary(fromSec, toSec) -> AnalyticsSummary`
- `GetAnalyticsTrendByDay(fromSec, toSec) -> AnalyticsTrend`
- `GetAnalyticsBreakTypeDistribution(fromSec, toSec) -> AnalyticsBreakTypeDistribution`
- `GetAnalyticsHourlyHeatmap(fromSec, toSec, metric) -> AnalyticsHourlyHeatmap`

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

## 版本管理

版本号单一来源：`VERSION`

- 校验版本一致性：

```bash
./scripts/check-version-sync.sh
```

- 一次性更新版本（同步 `VERSION` + `wails.json` + `frontend/package.json` + `frontend/package-lock.json`）：

```bash
./scripts/bump-version.sh 0.1.1
```

## 打包与发布（macOS / Windows）

完整规范见：`docs/packaging.md`

### 1) 打包 macOS DMG

```bash
./scripts/build-dmg.sh
```

- 默认产物：
  - `build/bin/macos-arm64/Pause-v<version>-macos-arm64.dmg`
  - `build/bin/macos-x64/Pause-v<version>-macos-x64.dmg`
- 常用参数：
  - `./scripts/build-dmg.sh --version 0.1.0 --no-clean`
  - `./scripts/build-dmg.sh --platform darwin/arm64 --bundle-id com.pause.app --codesign "-" --output-dir build/bin/macos-arm64`
  - `./scripts/build-dmg.sh --platform darwin/amd64 --bundle-id com.pause.app --codesign "-" --output-dir build/bin/macos-x64`
- 查看全部参数：`./scripts/build-dmg.sh --help`

### 2) 打包 Windows 安装器

```bash
./scripts/build-windows-installer.sh
```

- 依赖：本机安装 `NSIS`（macOS 可用 `brew install nsis`）。
- 默认产物目录：`build/bin/windows-x64/`
- 默认安装包文件名：`Pause-v<version>-windows-x64-setup.exe`
- 常用参数：
  - `./scripts/build-windows-installer.sh --platform windows/amd64 --webview2 browser`
  - `./scripts/build-windows-installer.sh --output-dir build/bin/win-release --clean`
- 查看全部参数：`./scripts/build-windows-installer.sh --help`

### 3) 生成统一发布清单与校验和

```bash
./scripts/generate-release-manifest.sh --version 0.1.0 --channel stable
```

- 默认扫描目录：`build/bin`
- 默认输出目录：`build/bin/release`
- 生成文件：
  - `build/bin/release/release-manifest.txt`
  - `build/bin/release/SHA256SUMS`

## 完全清理

macOS：

```bash
./scripts/cleanup/macos/cleanup-pause.sh
```

Windows（PowerShell）：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\cleanup\windows\cleanup-pause.ps1 -DryRun
powershell -ExecutionPolicy Bypass -File .\scripts\cleanup\windows\cleanup-pause.ps1
```

说明：
- 清理脚本已统一放在 `scripts/cleanup/` 下。

## 平台状态

- macOS：主流程可用（托盘、遮罩、通知、声音、开机启动）。
- Windows：主流程可用（空闲检测、托盘菜单、通知、提示音、开机启动、原生遮罩）。
  - 与 macOS 的主要差异：状态栏交互形态不同（Windows 为原生托盘菜单，macOS 为 Popover）。
  - 通知策略：默认使用原生 Win32 通知；若系统通知接口不可用，再回退到 PowerShell 通知兜底。
- Linux：已有适配层，原生体验待补齐。
