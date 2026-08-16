# AnyTTY 架构

本文描述当前 `master` 的稳定组件边界、连接模型、持久化和故障语义。它是架构总览，不记录整改批次、历史提交或未来功能清单。

## 1. 设计原则

1. daemon 是终端、文件、设备身份和客户端授权的最终所有者。
2. endpoint 是客户端本地保存的目标，route 是到达该目标的一种方式。
3. Local、SSH、Direct 和 Cloud 共用同一终端协议，不建立多套业务实现。
4. Controller 管理长期策略和注册真值，Edge 管理在线数据面的有界内存状态。
5. Controller 与 Edge 都不能替代 daemon 授予 terminal 或 file 权限。
6. 高频输出的主 PTY payload 必须受每 terminal 和全 daemon 的固定字节预算约束；附加订阅队列也必须独立有界。
7. 项目未发布，协议和开发数据直接升级；不保留旧格式兼容分支。

## 2. 仓库分层

```text
cmd/anytty                    CLI、TUI 和 daemon 进程组装
tui/                          TUI 状态、布局、渲染与配置
core/                         PTY、terminal、history、file 与 live screen
api_layer/                    core API 边界
client/                       endpoint、credential、session 与 route 选择
remote/                       Direct、SSH、WebRTC、DTLS/DataChannel
clients/ui/                   共享 React 产品界面
clients/mobile/               Android Capacitor 宿主与原生 Go bridge

cmd/anytty-cloud-controller   Controller 进程组装
cloud/controller/             账号、注册、策略、Directory、证书、用量
cmd/anytty-cloud-edge         Edge 进程组装
cloud/edge/                   Gateway、在线状态、Relay、Controller link
cloud/daemon/                 daemon 的 Cloud Agent runtime
cloud/web/                    公开网站、用户和运营控制台
proto/                        protobuf 源文件与生成物
```

## 3. 本地终端链路

```text
CLI / TUI / Mobile UI
        |
        | core protocol session
        v
daemon API -> TerminalProcess -> PTY -> shell / command
        |           |
        |           +-> bounded output buffer -> Live cursor
        |                                  \--> History cursor -> history store
        +-> file / access / endpoint services
```

daemon 由当前用户运行，默认使用用户级 socket。终端创建后拥有稳定 terminal ID 和独立 generation；终端退出会保留记录与历史，`kill` 停止进程，`rm` 删除已退出记录。

## 4. Endpoint 与 route

endpoint registry 是客户端本地配置。一个 endpoint 包含目标 daemon identity、展示信息、route 列表和选择策略。

| Route | 传输入口 | 信任锚 |
| --- | --- | --- |
| Local | 当前用户 daemon socket | 本机文件系统权限与 daemon identity |
| SSH | OpenSSH + 远端 loopback signaling/ICE-TCP | SSH host key pin + daemon identity |
| Direct | daemon 公开 signaling/ICE-TCP | 配对写入的 daemon identity 与 route credential |
| Cloud | Edge AgentGateway/ClientGateway | Controller 签名 grant、Edge CA、daemon/client proof |

route 选择只决定如何到达 daemon，不改变 CapabilityGrant 的权限。连接失败按结构化错误决定是否尝试下一条 route；授权、协议或终端错误不能伪装成网络失败触发宽泛 fallback。

## 5. 移动客户端

Android App 由 Capacitor WebView、共享 React UI 和原生 Go client bridge 组成。

- App 无账号、无登录、无自动发现。
- 用户只能扫描目标服务生成的一次性 pairing URI；App 不提供文本导入入口。
- endpoint registry 和客户端凭据保存在当前设备。
- 原生层拥有连接 session、secure credential、文件下载和 Android 生命周期协调。
- WebView 重载、App 前后台切换和系统网络变化通过 generation fence 使旧请求失效。
- 当前仓库包含 Android 工程；iOS 和桌面 GUI 只保留协议 product enum，不是已交付客户端。

## 6. 实时画面与背压

终端实时展示使用客户端基线驱动的 pull 模型：

1. 客户端用 `TerminalRef + observed_revision` 请求下一帧；protocol session 本身隔离不同客户端。
2. daemon 的短期 session baseline cache 能找到该 terminal revision 时，比较基线与最新画面并返回增量。
3. 基线缺失、过期、输出 gap 或尺寸不一致时返回全量画面。
4. 客户端把该 revision 选入唯一 renderer submission 后立即重挂 long-poll；渲染期间到达的结果只合并最新 damage。
5. 没有更新时请求等待；会话取消会终止等待，不保留孤立请求。

该模型避免按固定帧率持续推送，也避免每次都传完整屏幕。网络等待与物理渲染重叠，renderer 仍保持单写入。

每个 client baseline 只短期存在并自动清理；它不是持久历史，也不是全局帧 ring 的替代品。具体契约见 [docs/TERMINAL_DELIVERY.md](docs/TERMINAL_DELIVERY.md)。

## 7. 有界内存输出缓冲

每个 terminal generation 拥有一个共享 PTY payload buffer，Live 和 History 各自维护 cursor，同一主 payload 不复制两份。raw PTY stream 会为每个订阅者复制 chunk，并保留最多 16 个 chunk 的独立有界队列；这部分不计入 `resident_budget_bytes`。

默认预算：

| 参数 | 默认值 | 允许范围 |
| --- | ---: | ---: |
| 单 terminal `capacity_bytes` | 32 MiB | 64 KiB - 256 MiB |
| daemon `resident_budget_bytes` | 512 MiB | 64 KiB - 2 GiB |
| overflow | `block` | `block` / `drop` |

- `block`：缓冲满时停止继续读取/提交 PTY 输出，让上游自然减速。
- `drop`：不等待，淘汰旧 payload，并向每个受影响 consumer 发送有序 gap。
- gap 会切断 parser epoch；后续查询不能把 gap 两侧误当成连续画面。
- terminal 关闭、consumer 失败和 daemon shutdown 都会释放 resident budget。

因此主 PTY payload 受配置上限约束，raw-stream 内存受“订阅者数量 x 固定队列深度 x chunk 上限”约束；两者都不随累计输出总量增长，但当前实现不是与订阅者数量无关的严格每-terminal 常量。磁盘历史由独立大小和时间保留策略约束。

## 8. 历史、搜索与复制

history store 保存带时间信息的逻辑行和可重放终端内容。客户端用 token、generation、before/after cursor 和逻辑边界分页获取；协议没有数字 offset，也不一次加载全部历史。

- 进入历史模式时冻结 Live 末端和当前屏幕内位置。
- 首个历史页在 Live 画面后方 staging，避免进入模式时画面跳到错误末行。
- 向旧内容滚动时按当前列宽连续 prepend 页面。
- 新 Live 输出继续写历史，但冻结视口不会被推动。
- 返回逻辑尾部时客户端自动退出历史模式，并请求最新 Live 基线。
- 搜索返回逻辑行位置，复制保存 start/end range；确认复制时才分块物化文本。
- clipboard 是独立能力，不要求先进入复制模式。

共享 React UI 在 Android WebView 中交付，并使用同一套分页、搜索、复制和底部恢复逻辑。其普通浏览器构建当前只用于开发预览和测试，不是已发布的 Web terminal 产品。

## 9. 扫码配对

```text
daemon                  App / Client                    Edge (Cloud route)
  | create one-time offer    |                                  |
  |---------------- QR ----->|                                  |
  |                           | validate identity + route        |
  |                           |---- pairing admission ---------->|
  |<------------------------- live authorize -------------------|
  |<========== DTLS claim / capability exchange ===============>|
  |                           | verify daemon, save credential   |
```

pairing offer 可携带 Local 之外的 SSH、Direct 或 Cloud route hint。一次性 claim 有过期时间并在 daemon 内原子消费。Edge 和 Controller 不能根据二维码自行生成 terminal 权限；配对最终仍在客户端与 daemon 的端到端通道内完成。

稳定字段和失败语义见 [docs/PAIRING_PROTOCOL.md](docs/PAIRING_PROTOCOL.md)。

## 10. Cloud 拓扑

```text
                           account / enrollment / policy
Client ----------------------------------------------------> Controller
   |                                                            ^
   | cached locator, ClientGateway                               | mTLS EdgeControl v8
   v                                                            |
 Edge <----------------------------------------------------------+
   ^
   | persisted locator, AgentGateway
   |
daemon

Client <========== WebRTC + DTLS + DataChannel ==========> daemon
```

### Controller

Controller 使用 PostgreSQL 保存账号、daemon 状态、注册、套餐/用量、Edge desired config、证书档案和审计。它提供：

- 一次性 daemon enrollment 与 Edge bootstrap。
- Controller 签名的 binding、route grant 和 KeyBundle。
- EdgeControl v8 全量 snapshot、增量、daemon lifecycle、在线 Edge 重选、证书/config 控制和 EdgeIdentity 自动轮换。
- 仅在客户端可信 locator 缺失或明确不可达时使用的 Directory fallback。
- Relay reservation、renew 和 settlement 的商业真值。
- Cloud Web 的公开页面、JSON API 和运营入口。

### Edge

Edge 提供公网 AgentGateway、ClientGateway 和 TURN。它保存：

- 当前 Agent、client session、pending signaling、Relay allocation/group 的有界内存状态。
- Controller 下发的 daemon policy 内存投影。
- 可验证 desired config、managed public certificate、managed EdgeIdentity、binding KeyBundle cache。
- 未 ACK Relay reservation/settlement journal。

EdgeControl 断开后，Edge 清空 daemon policy 投影、关闭 Agent 并排空它仍跟踪的 Cloud session；新的受控准入 fail closed。Local、SSH 和 Direct 不依赖该控制链。Relay 不能离线创建或续租新的 authority。

### daemon enrollment

1. daemon 使用一次性 code 和 DeviceIdentity proof 访问 Controller。
2. Controller 选择 binding 目标 Edge，签发 binding 与 locator。
3. daemon 原子保存 Cloud enrollment record。
4. 后续启动直接连接记录中的 Edge，不在正常路径访问 Controller。
5. 已授权客户端优先使用 credential 中缓存的 locator。

AgentGateway 与 ClientGateway 均由 Edge 先发送一次性 challenge。proof 覆盖 challenge、Edge/boot/stream identity、双方 session/generation 和授权 envelope 摘要，避免捕获 Hello 在新连接重放。

## 11. Daemon 生命周期

Cloud daemon 的持久状态只有：

- `ACTIVE`：允许新的 Cloud client、pairing、P2P 和 Relay。
- `BLOCKED`：可恢复；Edge 先关闭准入，再排空现有 Cloud session/Relay，并通过保留的 Agent 连接通知 daemon。
- `DELETED`：终态；旧 binding 与 enrollment generation 永久拒绝，daemon 删除 Cloud enrollment 并断开 Agent。

Controller 提交状态后向所有 EdgeControl 广播 snapshot/delta。Edge 重连时必须先获取全量 snapshot；控制链断开时 Edge 丢弃该内存表并关闭 Agent。daemon 重连到 binding 指定的 Edge 时，该 Edge 使用当前 policy table 决定准入并下发目标状态。Controller 不持久化在线 Agent connection；在线 connection ownership 是 Edge 内存事实。

完整状态机见 [docs/CLOUD_DAEMON_LIFECYCLE.md](docs/CLOUD_DAEMON_LIFECYCLE.md)。

## 12. 信任与持久化

| 位置 | 持久数据 | 不应持久化 |
| --- | --- | --- |
| Controller PostgreSQL | 账号、daemon state、Edge config、证书元数据、用量、Relay reservation | terminal/file payload、私钥、在线连接对象 |
| Controller secret dir | signing key、证书私钥 | 普通 API 响应或日志 |
| Edge state dir | identity、公开 TLS、config cache、KeyBundle、Relay journal | Presence、session、terminal 数据 |
| daemon state | DeviceIdentity、AccessStore、Cloud enrollment、history | 客户端明文私钥、Controller session |
| client secure store | ClientAccessIdentity、CapabilityGrant、route credential/locator | Cloud 账号密码 |

任何一方都不得记录私钥、enrollment/claim token、完整 signed envelope、CapabilityGrant、TURN credential、SDP、ICE candidate、terminal 或 file payload。

## 13. 并发与资源边界

- 公网 gRPC 和协议消息有明确 byte limit，昂贵对象在完整校验后创建。
- Direct、AgentGateway、ClientGateway、Relay 和 terminal 请求均有并发/队列上限。
- 双向 stream 只有一个 writer，消息 sequence 严格单调。
- boot ID、connection ID、generation 和 revision 防止迟到清理影响新连接。
- state actor 线性化 Edge runtime mutation；关闭有 deadline，不无限等待 peer。
- Web 路由、Android native session 和文件传输使用 owner/generation fence 清理旧异步任务。
- 队列满时明确失败、block 或 drop，不允许无界 goroutine、slice、channel 或 JS promise 累积。

## 14. 故障语义

| 故障 | 行为 |
| --- | --- |
| 客户端丢失 live baseline | daemon 返回全量帧，建立新 baseline |
| terminal output buffer 满 | 按 `block` 减慢上游，或按 `drop` 产生 gap |
| History consumer 失败 | 错误可见，释放其 cursor；不能拖住 Live 内存 |
| Controller 停止 / EdgeControl 断开 | Edge 清空 policy、关闭 Agent 并排空 Edge 跟踪的 Cloud session；新准入 fail closed |
| Edge 重启且 Controller 暂时不可达 | KeyBundle 只能验证签名，不能恢复 daemon policy；Controller snapshot 恢复前所有托管 admission fail closed |
| binding Edge 不可达 | 已授权客户端只对明确 locator 网络/位置错误尝试 Directory fallback |
| App 进后台或 WebView 重载 | native generation 失效旧请求，前台恢复后按本地 registry 重建 session |
| daemon 被阻断 | Edge 拒绝新 Cloud 数据面并关闭现有 Cloud session，允许恢复 |
| daemon 被删除 | Cloud enrollment 清理；Local、SSH、Direct 和本地历史保留 |

## 15. 明确非目标

- App 账号登录、账号设备同步和自动发现。
- Controller 或 Edge 读取、代理或授权终端内容。
- daemon 自动跨 Edge 迁移。
- Web 浏览器终端产品；当前 Cloud Web 是公开信息和控制台。
- iOS 或桌面 GUI 的发布承诺。
- 开发期旧协议、旧 YAML 或旧 enrollment 数据兼容层。
- 通用事件总线、通用 actor 框架或为假设需求预留的扩展层。

## 16. 变更要求

跨边界变更按以下顺序完成：

```text
schema -> generated code -> store/runtime -> API mapping -> client/UI -> tests -> docs
```

至少保持 `go test ./...`、生成代码检查、相关 workspace test/typecheck/build 和对应平台测试通过。稳定行为必须进入本文件或 `docs/` 专题文档；一次性计划、审查报告和阶段状态不作为长期真值提交。
