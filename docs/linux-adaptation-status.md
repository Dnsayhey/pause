# Linux 适配现状与梳理（Pause）

更新时间：2026-03-16

## 1. 目标与范围

当前 Linux 方向按“能力探测 + 分级降级”推进，不按发行版硬编码分支。  
首批落地范围：

- `notifier`
- `startup`
- `sound`
- `idle`
- `overlay`（V1：X11 可用，Wayland 明确降级）

## 2. 环境认知（先按 DE + Session，不按 distro 名）

运行时关键判断维度：

- 桌面环境：`XDG_CURRENT_DESKTOP` / `DESKTOP_SESSION`
- 会话类型：`XDG_SESSION_TYPE`（`wayland` / `x11`）
- 显示与会话能力：`DISPLAY` / `WAYLAND_DISPLAY` / `DBUS_SESSION_BUS_ADDRESS`

结论：

- GNOME 不是只等于 X11 或 Wayland；它可以跑在两者上。
- Ubuntu Desktop 上，GNOME 默认通常是 Wayland（X11 可选会话）。
- Linux 托盘能力取决于 DE/扩展实现，不能假设“所有环境都等价于 macOS/Windows”。

## 3. 发行版与默认桌面（用于测试分层，不用于硬编码）

- Ubuntu Desktop：GNOME（默认 Wayland，X11 可选）
- Kubuntu：KDE Plasma
- Xubuntu：Xfce
- Fedora Workstation：GNOME（近年趋势为 GNOME Wayland-only）
- Debian GNOME：支持 Wayland/X11（默认与硬件/配置相关）
- Pop!_OS（24.04 线）：COSMIC（需视为独立平台能力）

注：`Ubuntu 26.04 LTS` 在 2026-03-16 时点尚未正式 GA，当前只做前置兼容准备，不做版本特判。

## 4. 当前已开发落地（代码级）

### 4.1 Linux adapters（notifier/startup/sound/idle）

文件：`internal/platform/linux/adapters.go`

- `Notifier`
  - 优先 `notify-send`
  - 回退 `dbus-send`
  - 带超时与失败日志
- `StartupManager`
  - 写入 `~/.config/autostart/*.desktop`
  - 支持启停和状态读取
- `SoundPlayer`
  - 优先级：`canberra-gtk-play` -> `paplay` -> `aplay`
  - 声音文件路径自动探测与回退
- `IdleProvider`
  - 由 noop 改为真实探测
  - 后端优先级：`xprintidle` -> `xssstate` -> `gdbus org.gnome.Mutter.IdleMonitor`
  - 采样缓存、超时、失败回退
- 环境/能力日志
  - `linux.env ...`
  - `linux.capabilities ...`
  - `linux.wayland_limited ...`
  - `linux.idle_probe_backend ...`

测试：`internal/platform/linux/adapters_test.go`

- 覆盖 desktop entry、通知回退、声音后端优先级、idle 解析与后端选择、环境探测等。

### 4.2 Linux overlay V1（Wails）

文件：`internal/desktop/overlay_linux_wails.go`

- Wayland：显式禁用 native overlay，回退前端 overlay 路径
- X11：native-like 外部后端优先级
  - `yad` -> `zenity` -> `xmessage`
- 支持 skip 回调（按退出码判定）
- 通过 countdown bucket 节流重启，减少每秒闪烁

测试：`internal/desktop/overlay_linux_wails_test.go`

关联 build tag 调整：

- `internal/desktop/overlay_stub_wails.go`  
  从 `wails && !darwin && !windows` 改为 `wails && !darwin && !windows && !linux`

### 4.3 close 行为兼容修正

文件：`internal/app/app_before_close_wails.go` + `internal/desktop/background_hide_*.go`

- 新增 `SupportsBackgroundHideOnClose()` 能力门控
- macOS / Windows 返回 `true`
- 默认（含 Linux 当前）返回 `false`
- 避免 Linux 上“关闭后隐藏但无法恢复”的死状态

## 5. 当前限制与已知降级

- Wayland 下无法保证 macOS/Windows 等价的“强制遮罩”能力（协议安全模型限制）
- Linux 托盘/状态栏能力高度依赖 DE 与宿主支持
- 当前 Linux `statusbar` 仍是 noop（尚未实现 tray + menu 控制器）

## 6. 下一步建议（按优先级）

1. 落地 `statusbar_linux_wails.go`（tray + menu，先对齐 Windows 交互）
2. 托盘可用后，评估 Linux 是否可条件性支持 `BackgroundHideOnClose`
3. 增加运行时 capability 快照日志，便于用户反馈时定位
4. 制作测试矩阵（GNOME Wayland、GNOME X11、KDE Wayland、Xfce X11）
5. 打包侧补齐依赖提示（`yad/zenity/xmessage/notify-send` 等）

## 7. 测试与验证状态

- 已通过本地 `go test ./...`（历史 darwin 警告不影响本次 Linux 工作）
- 已进行 Linux 交叉编译级检查（包含 `-tags wails` 场景）
- 由于宿主是 macOS，未直接执行 Linux 二进制（`exec format error` 预期）

