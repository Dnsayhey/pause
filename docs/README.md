# Pause 文档索引

最后更新：2026-04-02

这份索引用来回答两个问题：

1. 现在应该先看哪份文档？
2. 哪些文档是当前规范，哪些只是规划或历史备注？

## 当前规范

这些文档描述的是当前实现和维护约束，默认应与代码保持同步。

- [架构与代码结构](./architecture.md)
  说明 `internal/app`、`internal/backend`、`internal/desktop`、`internal/platform` 的职责边界、依赖方向、Wails API 与运行链路。
- [打包、发版与更新源](./packaging.md)
  说明版本管理、桌面打包、GitHub Release、`stable.json` 生成与 GitHub Pages 部署流程。
- [通知能力与前端策略](./notification-logic.md)
  说明通知权限模型、运行时通知发送链路、前端三态产品态与平台差异。

## 设计备注 / 规划

这些文档不是“当前规范”，而是为后续迭代保留的设计草稿、迁移计划或背景说明。只有在相关迭代开始时才需要阅读。

- [History DB Migration Plan](./notes/historydb-migration-plan.md)
  记录 `historydb` 将来引入 schema migration 时的低风险方案，不代表当前已经实现。

## 建议阅读顺序

如果你是第一次接手这个仓库，推荐按这个顺序看：

1. 根目录 [README](../README.md)
2. [架构与代码结构](./architecture.md)
3. [打包、发版与更新源](./packaging.md)
4. 按需再看 [通知能力与前端策略](./notification-logic.md)

## 维护约定

- README 只保留项目入口、常用命令和文档入口。
- `docs/` 根目录优先放“当前规范”。
- 规划性、备忘性、迁移草稿类文档放到 `docs/notes/`。
- 如果代码行为已经变化，优先更新对应规范文档，而不是在别处补一份新的说明。
- 尽量让一份文档只回答一类问题，避免 README 和专题文档重复维护同一段细节。
