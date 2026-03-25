# 后端重构进度日志（Backend Refactor Progress）

最后更新：2026-03-25（晚间，迭代追加）

关联文档：

- 计划文档：[backend-refactor-plan.md](/Users/yanlei/Projects/go/pause/docs/backend-refactor-plan.md)

## 1. 当前状态快照

- 分支：`codex/reminder-edit-unit-toggle`
- 工作区：存在待提交改动（settings 链路解耦 + analytics alias 清理）
- 最近一次提交：`d444b57`
- 当前重构总体进度：约 `96%`（阶段 A/B/C/D 完成，阶段 E 收尾中）

## 2. 已完成里程碑（按时间倒序）

### 2026-03-25

1. `WIP` `refactor(settings): decouple settings usecase from runtime engine`
- `settings` 仓储由“转发到 engine”改为直接依赖 `settingsjson store + startup manager`，移除反向依赖链路。
- `runtime/engine` 删除 `SyncPlatformSettings/GetLaunchAtLogin/SetLaunchAtLogin`，职责收敛到纯运行时逻辑。
- 新增 `adapters/settings` 单测覆盖首次启动同步与开机启动状态读写行为。

2. `WIP` `refactor(analytics): remove historydb alias passthrough`
- 删除 `storage/historydb/analytics_types_alias.go`，`historydb` 直接返回 `domain/analytics` 类型。
- `adapters/history/analytics_repo.go` 去除无意义 DTO 二次映射，直接透传查询结果。
- 验证：`go test ./...` 通过，`npm --prefix frontend run build` 通过。

3. `WIP` `refactor(platform): tighten ports boundary between app/backend and platform`
- `app` 与 `bootstrap` 中通知能力类型从 `platform.*` 收敛到 `backend/ports.Notifier`，上层不再依赖平台门面类型。
- `platform/platform.go` 删除无用的 `Idle/Lock/Sound/Startup/Noop` 转发别名，仅保留 `Adapters` 聚合类型。
- `platform/platform.go` 已删除；`platform/factory_*` 直接返回 `platform/api.Adapters`，平台包收敛为纯工厂入口。
- `platform/api` 删除无用接口别名声明，统一引用 `backend/ports` 契约，仅保留 `Adapters` 聚合与 `Noop` 能力实现。
- 明确 `platform` 角色为“能力提供工厂”，`desktop` 角色为“桌面壳交互层”。

4. `WIP` `refactor(runtime): split engine file by responsibility`
- 将 `runtime/engine/engine.go` 从单文件拆分为：
  - `engine.go`（结构/构造）
  - `engine_tick.go`（tick 主循环与通知）
  - `engine_commands.go`（命令入口与 runtime state）
  - `engine_history.go`（break 历史落库辅助）
- 目标是“按职责阅读”，不是行为变更；现有测试与构建均通过。

5. `WIP` `refactor(reminder-domain): remove legacy alias type names`
- 删除 `backend/domain/reminder` 中 `ReminderConfig/ReminderPatch/ReminderCreateInput` 兼容别名，统一回归 `Reminder/Patch/CreateInput`。
- app/runtime/scheduler 侧引用已同步切换，减少“同义词类型”造成的认知负担。

6. `WIP` `refactor(reminder-sync): move runtime snapshot sync into reminder usecase`
- 新增 `ports.ReminderRuntimeSink`，由 runtime engine 实现 `ApplyReminderSnapshot`。
- `usecase/reminder.Service` 内建同步能力（`SetRuntimeSink` + `Sync`），CRUD/EnsureDefaults 后自动推送 runtime 快照。
- `bootstrap.NewRuntime` 统一装配并执行首次 `Sync`；`app` 层删除手动 `SetReminderConfigs` 同步代码。

1. `WIP` `refactor(domain): consolidate core models into backend domain`
- `internal/core` 目录已从代码层移除，`analytics/reminder/settings` 类型统一收口到 `internal/backend/domain/*`。
- app 与 platform 引用切换到 `backend/domain`，移除 analytics 旧转换层。
- settings 领域模型文件迁移到 `backend/domain/settings`，并清理残留 `core*` 命名，降低阅读噪音。
- 验证：`go test ./...` 通过，`npm --prefix frontend run build` 通过。

1. `7e79da5` `refactor(runtime): remove empty analytics alias and split engine helpers`
- 删除无实际价值的空别名端口：`ports/analytics_query_repo.go`。
- 将 `runtime/engine/engine.go` 的纯辅助函数拆分到 `engine_helpers.go`，`engine.go` 从 993 行降到 777 行。

2. `fa8ae46` `refactor(ports): consolidate runtime dependency ports into single file`
- 将 runtime 依赖端口从多个小文件合并为 `ports/runtime_dependencies.go`。
- 保持端口边界不变，同时减少目录碎片与认知跳转成本。

3. `5bd01e9` `refactor(backend): route engine io dependencies through ports`
- 新增并补齐运行时 IO 端口：`break/idle/lock/notifier/sound/startup/clock`。
- `runtime/engine` 由直接依赖 `platform` 接口改为依赖 `backend/ports`。

4. `4a8b4ef` `refactor(runtime): move scheduler session state into backend runtime layer`
- 将 `scheduler/session/state` 从 `internal/core` 迁移到 `internal/backend/runtime`。
- app 与 engine 引用路径同步切换，行为保持不变。

4. `8272731` `docs(backend): add refactor plan and progress tracking`
- 新增重构计划与进度文档，建立后续每次迭代记录规范。

5. `feac4ee` `refactor(backend): move runtime engine into backend runtime layer`
- 将引擎从 `internal/core/service` 迁移到 `internal/backend/runtime/engine`。
- 修正 app / adapters / bootstrap 引用。

6. `2fd20b9` `refactor(backend): move settings store into backend storage layer`
- 将 settings 存储迁移到 `internal/backend/storage/settingsjson`。
- `core/settings` 收敛为纯模型与 normalize 逻辑。

7. `743c8e7` `refactor(service): depend on settings store interface`
- 引擎对 settings 存储改为接口依赖，降低耦合。

8. `27d5fb0` `refactor(backend): move history store into backend storage layer`
- 将 history 存储迁移到 `internal/backend/storage/historydb`。

9. `1a66bd2` `refactor(history): simplify store naming across backend`
- 命名收敛：`HistoryStore/OpenHistoryStore` -> `Store/OpenStore`。

10. `ea1a480` `refactor(backend): move settings adapter out of engine namespace`
- `internal/backend/adapters/engine` 重命名归位到 `adapters/settings`。

11. `a686e66` `refactor(app): isolate engine skip mode through adapter`
- 将 skip mode 通过 app adapter 隔离，减少 app 对底层类型泄漏。

12. `5420648` `refactor(app): depend on engine interface instead of concrete type`
- `App` 从依赖具体引擎改为依赖最小行为接口。

13. `3fc98b3` `test(reminder): cover ensure-defaults behavior in usecase`
- 补充 built-in reminder seed 行为测试。

14. `406cfc2` `refactor(history): alias analytics models to core analytics dto`
- 降低 analytics DTO 重复定义。

15. `0157884` `refactor(analytics): introduce dedicated API dto package`
- 建立更清晰的 analytics API DTO 边界。

16. `2c7a50e` `refactor(runtime): route break persistence through history adapter`
- break 落库通过 adapter 统一入口。

17. `9d44cb9` `refactor(reminder): move built-in seed creation into usecase`
- 默认提醒创建逻辑上移到 usecase。

18. `12afce5` `test(bootstrap): cover runtime wiring and empty path guard`
- 补充 runtime 组装测试覆盖。

19. `ffd1ba7` `refactor(bootstrap): add runtime assembler and slim app wiring`
- 强化 bootstrap 装配职责，减少 app 初始化负担。

20. `96aa999` `refactor(app): move app dependency contracts to dedicated types file`
- app 依赖契约抽离，便于持续解耦。

21. `a2db4a6` `refactor(bootstrap): centralize settings service wiring`
- settings service 组装统一收口。

22. `766d9bd` `refactor(reminder): seed built-ins via usecase and map domain errors`
- usecase 侧补齐错误映射与默认 seed 行为。

23. `957c05d` `refactor(app): extract runtime and notification handlers`
- 拆分 app 职责，降低单文件复杂度。

24. `94bafa9` `refactor(app): remove legacy settings fallback paths`
- 删除遗留 fallback 路径。

25. `5a679d5` `refactor(app): split app entrypoints by settings reminders analytics`
- app 入口按子域拆分。

26. `98455ea` `refactor(backend): route settings and startup ops through usecase`
- settings/startup 路径接入 usecase。

27. `d0bd817` `refactor(backend): route analytics queries through usecase slice`
- analytics 查询链路接入 usecase。

28. `9fdd07e` `refactor(backend): scaffold reminder slice with usecase and ports`
- reminder 分层骨架搭建完成。

## 3. 当前目录状态（重构后）

```text
internal/backend/
  adapters/
    history/
    settings/
  bootstrap/
  domain/
    analytics/
    reminder/
    settings/
  ports/
  runtime/
    engine/
    scheduler/
    session/
    state/
  storage/
    historydb/
    settingsjson/
  usecase/
    analytics/
    reminder/
    settings/
```

## 4. 进行中与下一步

### 正在进行

- 阶段 E：结构收尾与一致性治理。

### 下一步候选（按优先级）

1. 继续收口跨层类型泄漏，统一 DTO 边界（尤其 app <-> backend）。
2. 拆分 runtime engine 的命令处理职责（按 pause/resume/break/settings 分文件），继续压缩超大文件。
3. 评估并消除剩余“仅命名转发”的 type alias，保留真正有语义价值的边界类型。

## 5. 每次更新模板（后续追加到本文件）

请按以下模板在文件末尾新增一节，标题格式为 `## YYYY-MM-DD / Iteration N`：

```md
## 2026-03-26 / Iteration N

### 目标
- 本次要解决的问题（1-3 条）

### 变更
- 核心改动点
- 关键文件
- 删除了哪些旧路径或旧接口

### 验证
- `go test ./...`：通过/失败
- `npm --prefix frontend run build`：通过/失败

### 风险与备注
- 潜在风险
- 偏离计划说明（如果有）

### 下一步
- 下一迭代的最小闭环目标
```
