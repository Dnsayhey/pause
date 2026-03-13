# TODO: 主界面偶发自动拉起（macOS）

更新时间：2026-03-13
负责人：yanlei / codex
状态：进行中

## 问题描述

在状态栏 `Pause Item` 和 `Popover` 操作过程中，主界面会偶发被自动拉起。  
该行为并非每次必现，且存在“有时没有明显业务动作日志”的情况。

## 已确认现象

- 触发异常时，不一定出现 `open_window` 行为。
- 部分日志片段显示：
  - `statusbar.event status_item_toggle_open`
  - `window.event did_change_occlusion_state visible=true/false`
  - 偶发 `window.event did_become_key`
- 异常片段中未看到：
  - `statusbar.callback action=open_window`
  - `window.show activate source=status_bar_open`

结论：当前证据指向“不是显式点击 Open Pause 触发”，更可能是系统焦点/窗口层级变化导致。

## 已完成改动（用于定位）

1. 状态栏埋点增强
- `statusbar.callback`：记录 action id + action name
- `statusbar.event`：记录 toggle/popover/app active 事件

2. 窗口行为埋点增强
- `window.show`/`window.hide` source 级日志
- 原生窗口通知：
  - `did_become_key` / `did_resign_key`
  - `did_become_main` / `did_resign_main`
  - `did_deminiaturize` / `did_miniaturize`
  - `did_expose` / `did_change_occlusion_state`
  - `app_did_become_active` / `app_did_resign_active`
- `snapshot_changed`：仅在窗口状态变化时记录（visible/key/main）

3. Popover 相关调整
- 打开 Popover 时不强制激活 app。
- Popover 打开时临时隐藏状态栏 tooltip，关闭后恢复。

4. 窗口识别增强
- 日志新增 `popover=true/false` 标记，用于区分 Popover 窗口与主窗口事件。
- 主窗口解析时优先排除 Popover 类型窗口，减少误判。

## 当前假设

1. 事件混入假设
- 之前窗口通知可能混入 Popover 窗口事件，导致误以为主窗口被操作。

2. 系统焦点链路假设
- 某些交互路径触发 app active/focus 切换，可能间接影响主窗口可见性。

## 复现与采样方法

```bash
launchctl setenv PAUSE_DEBUG_LOG 1
tail -f ~/.pause/trace.log | rg "statusbar|desktop.action|window\\.show|window\\.event|open_window|snapshot_changed"
```

建议在以下序列下采样：
- 仅点击状态栏图标开/关 Popover
- 在 Popover 内点击 pause/resume/zzZ
- 打开过一次主界面后重复上述动作

## 下一步

1. 先确认 `popover` 标记后的新日志是否仍出现“主窗口 visible=true 且无 show 调用”。
2. 若确认存在，补一层“窗口对象地址/标题/className”日志，精确定位来源窗口。
3. 若来源为系统 focus 链路，增加主窗口展示白名单（仅 `open_window` 或明确业务场景允许拉起）。
4. 回归测试用例补充：
- 打开 Popover 不应导致主窗口显示
- Popover pause/resume/zzZ 不应拉起主窗口
- 仅 Open Pause 动作允许激活并显示主窗口
