# Starxo Plan

本目录现在只保留当前有效、可直接实施的计划，不再维护历史编号计划，也不再把实施计划散落到 `doc/` 目录。

## 当前权威方案

- [dynamic-tool-surface-eino](dynamic-tool-surface-eino.md)
  - 主题：基于 Eino 为 `starxo` 设计并实现 dynamic tool surface、`tool_search` 和 MCP resources。
  - 作用：当前唯一的实施计划入口。

## 目录约定

- `plan/` 只放当前有效的实施计划。
- `doc/src/` 放代码文件对应的技术说明，不与 `plan/` 重复承担变更方案职责。
- `doc/` 顶层放文档总览、研究和流程说明。
- 历史 plan 不在工作区长期保留；如需追溯，直接从 Git 历史查看。
- 变更流转以 `master` 为主干、`dev` 为开发缓冲分支；实施分支先进入 `dev`，再由 `dev` 统一进入 `master`。
