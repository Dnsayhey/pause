# Pause 文档索引

最后更新：2026-04-23

`docs/` 只放当前实现的说明和少量明确标记的设计备注。

## 当前文档

- [架构说明](./architecture.md)
  当前代码结构、分层职责、运行链路、数据边界和对外 API。

- [通知说明](./notification-logic.md)
  当前通知链路、前端预检规则、平台差异和日志。

- [打包说明](./packaging.md)
  当前打包、发版和更新源相关流程。

## 备注文档

- [History DB Migration Plan](./notes/historydb-migration-plan.md)
  未来可能使用的迁移方案，不代表当前已经实现。

## 建议阅读顺序

第一次接手仓库，建议按这个顺序看：

1. 根目录 [README](../README.md)
2. [架构说明](./architecture.md)
3. [通知说明](./notification-logic.md)
4. [打包说明](./packaging.md)

## 维护原则

- 文档只写当前实现
- 不保留无用历史说明
- 代码语义变化时直接改现有文档，不追加“补丁式说明”
- 主题太大就拆专题文档，不把所有背景都堆到一份文档里
