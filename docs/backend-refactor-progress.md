# 后端重构进度日志（Backend Refactor Progress）

最后更新：2026-03-25

关联文档：

- 计划文档：[backend-refactor-plan.md](/Users/yanlei/Projects/go/pause/docs/backend-refactor-plan.md)

## 1. 当前状态快照

- 分支：`codex/reminder-edit-unit-toggle`
- 工作区：代码工作区已稳定，本次新增文档待提交
- 最近一次提交：`feac4ee`
- 当前重构总体进度：约 `80%`（阶段 A/B/C/D 完成，阶段 E 进行中）

## 2. 已完成里程碑（按时间倒序）

### 2026-03-25

1. `feac4ee` `refactor(backend): move runtime engine into backend runtime layer`
- 将引擎从 `internal/core/service` 迁移到 `internal/backend/runtime/engine`。
- 修正 app / adapters / bootstrap 引用。

2. `2fd20b9` `refactor(backend): move settings store into backend storage layer`
- 将 settings 存储迁移到 `internal/backend/storage/settingsjson`。
- `core/settings` 收敛为纯模型与 normalize 逻辑。

3. `743c8e7` `refactor(service): depend on settings store interface`
- 引擎对 settings 存储改为接口依赖，降低耦合。

4. `27d5fb0` `refactor(backend): move history store into backend storage layer`
- 将 history 存储迁移到 `internal/backend/storage/historydb`。

5. `1a66bd2` `refactor(history): simplify store naming across backend`
- 命名收敛：`HistoryStore/OpenHistoryStore` -> `Store/OpenStore`。

6. `ea1a480` `refactor(backend): move settings adapter out of engine namespace`
- `internal/backend/adapters/engine` 重命名归位到 `adapters/settings`。

7. `a686e66` `refactor(app): isolate engine skip mode through adapter`
- 将 skip mode 通过 app adapter 隔离，减少 app 对底层类型泄漏。

8. `5420648` `refactor(app): depend on engine interface instead of concrete type`
- `App` 从依赖具体引擎改为依赖最小行为接口。

9. `3fc98b3` `test(reminder): cover ensure-defaults behavior in usecase`
- 补充 built-in reminder seed 行为测试。

10. `406cfc2` `refactor(history): alias analytics models to core analytics dto`
- 降低 analytics DTO 重复定义。

11. `0157884` `refactor(analytics): introduce dedicated API dto package`
- 建立更清晰的 analytics API DTO 边界。

12. `2c7a50e` `refactor(runtime): route break persistence through history adapter`
- break 落库通过 adapter 统一入口。

13. `9d44cb9` `refactor(reminder): move built-in seed creation into usecase`
- 默认提醒创建逻辑上移到 usecase。

14. `12afce5` `test(bootstrap): cover runtime wiring and empty path guard`
- 补充 runtime 组装测试覆盖。

15. `ffd1ba7` `refactor(bootstrap): add runtime assembler and slim app wiring`
- 强化 bootstrap 装配职责，减少 app 初始化负担。

16. `96aa999` `refactor(app): move app dependency contracts to dedicated types file`
- app 依赖契约抽离，便于持续解耦。

17. `a2db4a6` `refactor(bootstrap): centralize settings service wiring`
- settings service 组装统一收口。

18. `766d9bd` `refactor(reminder): seed built-ins via usecase and map domain errors`
- usecase 侧补齐错误映射与默认 seed 行为。

19. `957c05d` `refactor(app): extract runtime and notification handlers`
- 拆分 app 职责，降低单文件复杂度。

20. `94bafa9` `refactor(app): remove legacy settings fallback paths`
- 删除遗留 fallback 路径。

21. `5a679d5` `refactor(app): split app entrypoints by settings reminders analytics`
- app 入口按子域拆分。

22. `98455ea` `refactor(backend): route settings and startup ops through usecase`
- settings/startup 路径接入 usecase。

23. `d0bd817` `refactor(backend): route analytics queries through usecase slice`
- analytics 查询链路接入 usecase。

24. `9fdd07e` `refactor(backend): scaffold reminder slice with usecase and ports`
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

1. 收口跨层类型泄漏，统一 DTO 边界（尤其 app <-> backend）。
2. 评估 `core/scheduler + core/session + core/state` 与 `backend/runtime` 的边界归位方案。
3. 更新 README 的代码结构描述，确保与当前目录一致。

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
