# access SDK 薄壳

访问产品线的官方 SDK 采用薄壳策略：Go 直接复用 `access/engine` 的公开包，
Python / TypeScript 只实现同一 wire 的编解码与最小会话封装。

- 语言范围：Go、Python、TypeScript。
- 不做 Dart：Flutter 已通过 FFI 直接依赖 `access/engine/binding`，不重复包装。
- wire 快照：见 `access/conformance/`。

当前目录先固定边界与文档；Python / TypeScript 薄壳随对应迁移任务落地，
不改变既有 CLI、terminal pool 与 Flutter 的行为。
