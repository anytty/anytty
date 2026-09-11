# 插件设计独立评审：协议与路由

评审对象：`PLUGIN_SDK_V1.zh-CN.md` 修订 2。评审日期：2026-09-11。

结论：**PASS（修订后复审通过）。** 三处阻塞项已在主设计 §4.2、§6、§13 修正；新增同 daemon 多 endpoint 别名注册交接规则。总体架构可实施。这是设计审核，不是实现验收。

## 首轮问题与复审结果

以下问题现已解决；保留原审查理由作为实现检查依据。

1. **源凭据与目标地址分开。** §4.2 当前将注册租约写入目标地址，容易使实现把目标的发送凭据暴露给调用者。应规定 source lease 是绑定当前认证连接的秘密，只由桥/宿主附加；destination 仅含公开实例身份及 registration epoch。接收 daemon 从连接获取来源，检查 source lease；目标 epoch 只用于防止旧目标重放，不授予权限。Reply 必须匹配 daemon 保存的 pending request，不能把知道 request ID 当成回复权限。

2. **状态合并不得破坏游标语义。** §6 允许合并最新状态，但事件日志 sequence 要么完整递增投递，要么明确发送完整 snapshot/reset 与新的水位线。不能跳过 delta 后继续以连续事件流交付。首版最简单可靠策略是有界队列溢出立即 `RESYNC_REQUIRED`，重新建立原子快照 + watch。核心现有 `ApplicationEventSubscribe` 是瞬时通知流，并有编码失败跳过通知行为，不能直接宣称它已经提供可靠插件日志。

3. **明确现有认证范围如何限制插件 API。** 现有 `TransportScope` 包含 `AllowDaemon`、`TerminalID`、`MachineEventsOnly`、`PrincipalID`；`AcquireApplication` 对 capability 做白名单准入。新增插件 capability 需要单独 admission。首版可以只允许 daemon-owner/full-daemon 会话注册通用插件路由，受限 share/machine-events 会话返回 `PERMISSION_DENIED` 或 `UNSUPPORTED`；若支持受限会话，订阅和操作均必须逐资源继承原授权。不能仅凭“连接已认证”就开放全 Agent 列表或跨 TUI 控制。`PrincipalID` 必须按 daemon/认证域解释，本地的 `local` 不等于跨设备全局同一用户。

## 已核实可沿用的基础

- `client/runtime/contracts.go` 的 `ReadyPeerSessionEvidence` 已校验 daemon identity pin、鉴权完成和协议版本，可作为路由域身份依据。endpoint alias 和 RouteID 不能替代该身份。
- `client/runtime/application_session.go` 写入并校验 application session stamp；新增插件投递仍须经过相同 generation fence。
- `proto/apipb/application.proto` 的 Command/Result/Event oneof 可新增插件承载分支；不需要 JSON 或直接 TUI 调用捷径。
- `core/application_session_port.go` 的 session-owned subscription 与 lifetime cancellation 可复用资源管理模式，但可靠恢复日志是新增能力。
- `tui/app/workbench_storage.go` 收到外部变更后自动 LoadWorkbench。设计已经正确识别布局传播风险；实现验收必须通过真实 watch 路径证明 TUI A 的换绑不会改变 B。

## 实施时的非阻塞细化

- `via_endpoint_id` 只属于 SDK 本地选路参数，不放入跨客户端解释的 MountSpec wire 身份；在线用 routing daemon identity。
- 同一 TUI 通过两个 endpoint 别名连接同一 daemon 时，需要明确去重策略，避免注册第二条连接无意踢掉第一条。可按认证 daemon identity + TUI instance 选择一条活跃插件路由，其余连接仍保留原终端用途。
- pending request 的最终结果必须来自登记的目标连接/epoch；断线应清理 pending、路由、订阅和委托。
- 插件消息的 daemon 路由记录应有测试可观测钩子，不必默认输出敏感 payload。
- §12 和 §15 的“单独批准/尚未授权”是旧稿文字：用户已明确授权双 Agent 审核通过后实施，不应再次引入审批停顿。

## 最低协议测试

覆盖自发自收且断开 daemon 后禁止本地成功、两个 TUI 定向隔离、两个 daemon 同名资源隔离、伪造源与回复拒绝、目标重连 epoch 失效、状态快照与 watch 原子性、溢出 resync、scope 受限会话不能读取全量 Agent 数据、幂等键重复但参数变化拒绝。
