# access conformance（wire 快照）

目标：把已上架 Flutter App 依赖的 wire 行为固化成 fixtures，任何访问产品线
重构都必须通过同一组快照，避免悄悄改变已发布客户端的兼容性。

- `fixtures/`：wire 快照（当前为空实现，先冻结目录与契约）。
- 快照来源：`proto/access/*` 与 `access/transport/*` 的现有 wire 行为。
- 本目录同时承载仓库布局守卫测试（daemon/access/layout 依赖方向）。
