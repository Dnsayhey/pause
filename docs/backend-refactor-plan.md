# 后端重构计划（Backend Refactor Plan）

最后更新：2026-03-25  
负责人：`codex/reminder-edit-unit-toggle` 当前重构线

## 1. 背景与问题陈述

本轮重构的出发点来自以下持续痛点：

- 历史实现中存在“同一职责跨多层重复表达”的问题，例如 reminder 与 analytics 类型在不同层重复定义且语义漂移。
- `app` 与底层运行时实现耦合过深，替换实现或局部重写时改动面过大。
- 持久化实现曾散落在 `internal/core`，包名与职责边界不清晰，影响可读性和迁移效率。
- 接口历史包袱较多，删除成本高于重写成本，导致“越兼容越复杂”。

## 2. 本轮重构目标

### 2.1 总目标

建立清晰、可演进、可替换的后端分层：

- `app` 只做编排与交互绑定；
- `backend` 承担用例、端口、适配器、运行时与存储实现；
- `core` 收敛为纯领域模型与无 IO 的业务规则。

### 2.2 约束原则

- 不做历史兼容迁移，允许破坏性重构。
- 优先删除冗余接口和过时行为，再补回最小闭环。
- 每次改动可小步提交、可独立回滚。
- 每一步必须通过 `go test ./...` 与 `npm --prefix frontend run build`。

## 3. 架构目标（Target Architecture）

```text
internal/
  app/                      # 应用编排层（Wails API、生命周期、桌面交互）
  backend/
    domain/                 # 领域对象（analytics/reminder/settings）
    ports/                  # 用例依赖的抽象端口
    usecase/                # 用例服务（业务流程）
    adapters/               # 端口适配器（history/settings 等）
    runtime/                # 运行时引擎与调度编排
    storage/                # 持久化实现（historydb/settingsjson）
    bootstrap/              # 组装根（容器、运行时装配）
  core/
    analytics/              # API/DTO 基础类型
    reminder/               # reminder 纯类型
    scheduler/ session/     # 纯业务逻辑
    settings/               # settings 结构与 normalize 逻辑
    state/                  # runtime state 结构
```

## 4. 分阶段计划与验收

### 阶段 A：提醒与历史存储简化（已完成）

目标：

- 统一 reminders 主键语义（int64）。
- 删除冗余流程，保留最小状态机语义（`canceled` 保留）。
- 移除过时接口与补丁式兼容逻辑。

验收：

- reminder CRUD 走同一主路径；
- 历史记录行为可由测试覆盖验证；
- 无遗留 string 主键路径。

### 阶段 B：分层骨架搭建（已完成）

目标：

- 建立 `backend/domain + ports + usecase + adapters + bootstrap` 结构。
- settings / reminder / analytics 通过 usecase 统一出入口。

验收：

- 上层不再直接操作底层存储细节；
- bootstrap 具备统一装配入口；
- 三条主线（settings/reminder/analytics）均可独立测试。

### 阶段 C：app 解耦与接口收口（已完成）

目标：

- `App` 对 runtime 引擎从“具体类型依赖”改为“最小接口依赖”。
- SkipMode 等行为通过 adapter/本地类型隔离。

验收：

- `internal/app` 不依赖 runtime 具体实现细节；
- 变更引擎实现时 app 层改动最小化。

### 阶段 D：存储与运行时边界归位（已完成）

目标：

- `history store` 从 `core` 迁至 `backend/storage/historydb`。
- `settings store` 从 `core` 迁至 `backend/storage/settingsjson`。
- runtime engine 从 `core/service` 迁至 `backend/runtime/engine`。

验收：

- `core` 不再承担 IO 存储职责；
- import 路径与包名与职责一致；
- 全量测试通过。

### 阶段 E：结构收尾与一致性治理（进行中）

目标：

- 继续收敛跨层类型泄漏，统一 API DTO 边界。
- 评估并收口 `core/scheduler + core/session + core/state` 与 runtime 的边界。
- 建立稳定的“重构日志 + 验收标准”机制，避免后续回弹。

验收：

- 每次迭代都有可追溯记录；
- 目录职责可被新人在 10 分钟内理解；
- 重构分支可平滑并回主线。

## 5. 风险与应对

- 风险：大规模 rename 造成隐藏引用遗漏。  
  应对：统一使用 `rg` 全局检索 + 全量测试。

- 风险：小步提交过多导致回顾困难。  
  应对：在进度文档按阶段汇总“里程碑提交”。

- 风险：边改边删导致上下文遗失。  
  应对：每次提交前记录“动机/范围/验证/后续”。

## 6. 执行规范（后续每次必须遵守）

- 每完成一个可提交单元，更新一次 [backend-refactor-progress.md](/Users/yanlei/Projects/go/pause/docs/backend-refactor-progress.md)。
- 记录必须包含：
  - 日期
  - 目标
  - 变更范围（文件或目录）
  - 验证结果（测试/构建命令）
  - 风险与下一步
- 若出现偏离本计划的重构动作，先在进度文档写明“偏离原因”再执行。

