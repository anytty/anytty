# AnyTTY 系统架构与职责划分

> 状态：与当前工作区（branch `refactor/tui-surface`，含未提交的边界收口/pro通
> provider typed 化）一致。本文回答"谁负责什么、能力归谁、依赖方向是什么"；
> 协议细节见 `clients/tui/docs/PROTOCOL.zh-CN.md` 与 `access/docs/*`；
> 历史进展与缺口见 `docs/HANDOFF.zh-CN.md`；安全边界另见
> `docs/SECURITY_BOUNDARY.md`。

## 1. 设计目标与原则

1. **单一客户端入口**：所有客户端（CLI/TUI/Flutter/Web/第三方程序）只连
   **access**；access 终结客户端 wire，并按能力路由到本地服务或 terminal
   provider。
2. **能力 owner 唯一**：每一项能力只有一个所有者（§5 矩阵），跨 owner 只能通过
   明确契约（access wire、provider 协议、tui2 程序协议）交互。
3. **终端真值在 pool**：PTY/进程/history/live/events/path 由 pool 持有；pool 不
   持有身份、账本、文件、storage，也没有网络出口。
4. **客户端 wire 稳定**：access wire（Hello v7 + application protocol）只做
   字节透明或可选字段扩展；Flutter 不需要因为内部重构发版。
5. **信任边界显式**：本地 unix socket 是 owner-only（0600）免鉴权；走网络的连接
   必须经 remoteauth；relay 是字节透明通道，授权在网络层完成。
6. **角色隔离**：同一个二进制三种角色（默认前台 / pool run / access run），默认
   栈是两个独立进程、独立进程组，可分别升级与崩溃隔离。
7. **本地也走 access**：managed 终端一律经 access→provider→pool，即使在本机；
   宿主内 PTY 只保留给显式的 `kind: command` 端点（§6.9）。

## 2. 角色与进程

| 组件 | 进程/入口 | 职责 | 绝不做什么 |
|---|---|---|---|
| **access** | `anytty access run`（独立 `anytty-access` 需自行构建；发布归档不含） | 唯一客户端入口：Hello/请求/流/事件；identity/pairing/grant/revoke/TTL；文件服务；browser proxy；storage；Direct/Cloud/relay；把 terminal 命令路由给 provider；撤销/过期时关闭客户端 transport | 不持有 PTY/终端真值；不直接操作终端进程 |
| **pool** | `anytty pool run` | 纯 terminal provider：PTY/进程、history（linehist）、live/snapshot、terminal events、path defaults/list-directories、attachment registry | 不监听网络；不持有 identity/store/files/storage；不认识客户端 wire |
| **CLI** | 同一 binary 子命令 | 客户端 + 栈生命周期（`pool`/`access` start/stop/restart/status/logs）；endpoint registry/配对/凭据管理；offline history 维护前端；默认入口拉起 tui2 | 不把业务真值放进 CLI 状态（除主机布局与 registry） |
| **tui2 host** | `tui2`（由 `anytty` 前台拉起） | 合成器：TTY/输入/焦点/epoch、内建 terminal 组件、安全内核（core overlay/退出确认）、provider 会话与 sources、可选的 access 转发桥（access.call/stream） | 不认识业务页面（tab/pane/picker）；不决定布局 |
| **布局程序** | `tui2-shell` 或被替换的任意程序（Go/Py/TS SDK） | 界面策略：view 盒子树、页面/workspace/tab/slot、键位 claim、插件编排 | 不直接操作 PTY/终端池（通过 host 方法或 access.call） |
| **Flutter/Web** | 移动端/浏览器 | 远端客户端：access wire；本地 route 之外均经 access | wire 之外无特权 |
| **relay/gateway** | `access/gateway`（`--listen`） | 字节透明 TCP/unix 转发（不解析 frame） | 不做 access 能力/grant 鉴权（只有网络层 `--allow` + 可选 pair-token 预握手）；不理解业务 |

## 3. 拓扑与监听面

```
客户端（CLI / TUI / Flutter / Web / 第三方程序）
   │  access wire（Hello v7 + application protocol，字节稳定）
   ▼
access（canonical unix socket；能力 owner + 路由）
   │  provider 协议（providerv1，owner-only）
   ▼
pool（PTY / history / live / events）
```

监听面：

| listener | 归属 | 默认 | 说明 |
|---|---|---|---|
| canonical unix | access/server | `$XDG_RUNTIME_DIR/anytty-v3-wire7.sock` | 客户端入口；0600；本地免鉴权 |
| provider unix | pool/provider | `<canonical>.provider` | provider 协议；只有 access 发 provider 请求，CLI 仅做 Hello 探活 |
| pairing unix | access/runtime | `<canonical>.pair` | remoteauth v2 PairingExchange |
| Direct signaling / ICE-TCP | access/direct | `--route` 显式配置 | 网络入口，remoteauth |
| Cloud edge 连接 | access/cloud | 凭据/enrollment 驱动 | WebRTC DataChannel；edge 只做信令/relay |
| relay tcp/unix | access/gateway | `--listen` 显式配置 | 字节透明转发 |

- 本地 socket 名带代际（`v3-wire7`）：`v3` = access+pool 代际，`wire7` = 客户端
  wire 版本。生产路径都经 `resolveV3Socket`/`shared/runtimepath` 计算；少数脚本与
  测试仍硬编码同名（如 `scripts/test-release-artifacts.sh`、tmux 冒烟），改名前
  需要一起改。
- macOS 上超长路径由 `shared/transport/unix` 映射到 `/tmp/anytty-<hash>.sock`
  短路径并在原路径留 symlink。

## 4. 协议分层

### 4.1 access wire（客户端 ↔ access，Hello v7）

按 command family 路由，不做"整段转发"：

| family | 命令 | 处理 |
|---|---|---|
| Terminal | `terminal.*`、`history.*`、`live`、`path.*`、`event.subscribe`、`release_resource`、`cancel_operation` | provider（pool） |
| File | `file.*` + download/upload 流 | `access/files` 本地终结 |
| Proxy | `browser.proxy.open` + 流 | `access/proxy` 本地终结 |
| Storage | `storage.*` | `access/storage` 本地终结 |
| Auth | `client_access.*`、`cloud.*` | `access/runtime` 直答 |
| 未知 | — | fail closed（unsupported） |

握手：Hello v7（只携带 version/client/server）→ 身份/授权（本地免鉴权；远程
remoteauth + DTLS channel binding）→ 请求/流/事件。（schema/limits/features 是
tui2 程序协议 HELLO 的字段，见 §4.3；access wire 没有这些字段。）

resource token 的所有权分两类：
- **access 重签**：attachment/file/browser token 由 access 以同一不透明形状重新
  签发（file `channel+ft+id`、browser `channel+random`、attachment access 自己的
  channel），provider 原始 token 不出 access；
- **provider 原样透传**：terminal event 订阅 token 由 provider 生成并原样放进
  客户端可见的 `ResourceHandle`（release 时转回 provider）；history token 是
  `HistoryWindowResult.token` 等普通字段，同样原样透传，`HistoryRelease`/
  `HistoryCopy`/`HistorySearch` 时交回 provider。

> 命名约定：本文用 `client_access.identity`、`cloud.*` 表示命令族；生成/线上的
> 名字是下划线形式（如 `client_access_identity`、`remote_cloud_status`）。

### 4.2 provider 协议（access ↔ pool，providerv1）

- **typed facade**（`access/provider/terminal.Provider`）是 access 侧唯一契约：
  运输 `proto/provider/v1` DTO，方法覆盖 lifecycle/metadata/tags/attach/
  history/live/events/paths，并声明 `Capabilities`。
- **wire** 是 `providerv1`（`internal/providerproto` framing，Version=1）；
  `access/provider/pool` 把 typed 调用投影为该 wire，`pool/provider` 服务该 wire。
- apipb ↔ providerv1 的投影函数集中在消费方 `access/server/terminalmap`；
  个别调用点（如事件订阅命令装配）会直接构造 providerv1 消息。provider 契约与
  pool 实现都不 import 客户端 apipb。
- attach 的 bootstrap/ready/sync-lost 由适配器消化，调用方看到的是纯 PTY
  `Duplex`（`Receive` 输出字节、`Send` 输入字节、typed 关闭/缺口错误）。
- tmux provider 只落接口占位（`Capabilities` 全 false）；zellij 目前没有代码，只在包注释里作为未来目标提及。

### 4.3 tui2 程序协议（host ↔ 布局程序）

- 帧：`HELLO`(1) / `VIEW`(2) / `EVENT`(3) / `RESULT`(4) / `RESPONSE`(5) /
  `STREAM`(6，双向，append-only) / `VIEW_DELTA`(7，增量视图，append-only)；schema 1，
  epoch/rev 语义见 PROTOCOL。
- **增量视图**：HELLO `features["view_delta"]` 协商；`VIEW_DELTA` 用路径式补丁
  （set/replace/insert/remove/move，PROTOCOL §2.1）作用于缓存 `rev_base`，失败回
  `view_rejected{base_mismatch|path_invalid|max_nodes|oversize}`；宿主 COW 路径复制应用，
  新 epoch 首帧必须全量。SDK 自动 diff + 自动回退全量。
- 方法白名单（权威注册表 `clients/tui/runtime/methods.go`）：
  `terminal.attach/create/restart/kill/remove/scroll/scrollEnd/copy`、
  `history.window`、`clipboard.read`、`input.forward`、`system.quit`、
  `endpoint.sync`、`access.call`、`access.stream.open`、`access.stream.subscribe`。
- **access 能力桥**：`access.call` 把程序序列化的 access `CommandEnvelope`
  透明转发到指定 endpoint 的 ready 连接，返回 `ResultEnvelope`；`access.stream.open`
  把 access `ResourceHandle`（文件传输）绑定成双向 STREAM；
  `access.stream.subscribe` 把 `EventSubscribe` 命令绑定成事件流。
  host 不做家族过滤/确认（program 全权，单写者纪律自担）。
- 内建 `local-access`：registry 没有 local endpoint 时的兜底；正常情况下
  registry 的 `local`（`socket: auto`）解析为本地 canonical socket。

### 4.4 启动/恢复语义

- 布局程序崩溃/重启：host `epoch+1`，重放 `HELLO+sources`，程序重建布局，按
  source id 幂等对账；打开中的 access stream 在重启时统一关闭并释放资源。
- provider 断开：`access/server` 检测 `Done` → 释放该 provider 发布的全部流 →
  下一个请求 lazy 重拨；客户端 attach 断开按 wire 的 sync-lost/closed 语义处理。
- 客户端断线重连：TUI 的 endpoint manager 指数退避重连并 rebind（重新 attach +
  `LiveScreenNext(0)` 快照重建）后才发布 health=ok。

## 5. 能力与责任矩阵

| 能力 | owner | 实现位置 | 客户端入口 | 边界说明 |
|---|---|---|---|---|
| PTY/进程生命周期 | pool | `pool/core` | `terminal.*` → provider | 终端真值只此一份 |
| attach/输入/resize owner | pool | `pool/provider` + core | `terminal.attach/input/resize` | owner epoch + resize lock；单写者 |
| history（保留/搜索/copy） | pool | `pool/core/history/linehist` | `history.*` | 落盘 truth；CLI 只做 offline 维护前端 |
| live/snapshot/events | pool | `pool/core/live`、events | `live`、`event.subscribe` | provider 事件按订阅转发 |
| path list-directories | pool | `pool/core/path_*` | `path.list_directories` | 属于终端主机 |
| terminal defaults | pool | `pool/core` | `terminal.defaults` | 默认 shell/cwd |
| identity/DeviceIdentity | access | `access/runtime` | `client_access.identity` | 单 owner lock（state `remote-v2`） |
| 配对/grant/撤销/TTL | access | `access/runtime`、`access/sessions` | `.pair` + `client_access.*` | revoke/到期实时关 transport |
| remoteauth（Direct/SSH/Cloud） | access | `access/remote`、`access/engine/adapter/{direct,ssh,cloud}` | 各 route | DTLS binding；grant 校验 |
| 文件 metadata + 传输 | access | `access/files`（+`transfer`） | `file.*` + 流 | 路径安全/续传/压缩/进度 |
| browser proxy | access | `access/proxy` | `browser.proxy.open` + 流 | 从 access 主机拨号 |
| storage KV + 变更事件 | access | `access/storage` | `storage.*` | opaque KV |
| Direct listener/记录 | access | `access/direct` | — | `<sock>.direct` 记录 |
| Cloud agent | access | `access/cloud` | `cloud.*` | edge 信令 + P2P/relay |
| relay | access | `access/gateway` | — | 字节透明，无协议参与 |
| 页面/布局/插件编排 | 布局程序 | `clients/tui/cmd/tui2-shell` 或自写 | VIEW/RESULT | host 不认识业务词 |
| access 能力（程序侧） | 布局程序（经 host 桥） | `access.call`/`stream.*` | STREAM/RESULT | 透明转发；纪律自担 |
| endpoint registry | CLI | `~/.config/anytty/endpoints.yaml` | — | CLI 写、TUI 只读 |

## 6. 关键机制与不变量

1. **依赖方向**：`pool/**` 不 import 任何 `access/**`；`access/engine`（客户端
   引擎）不 import 服务端子包；TUI 不 import `pool/core`；`access/provider/*` 不
   import apipb。
2. **DTO 所有权**：auth/remote/file/storage → `access/contract`（access 能力）；
   terminal/history → `pool/core`（终端真值）。`api_mapping` 在 access 侧做
   apipb↔DTO 投影。
3. **token 形状**：attachment/file/browser 由 access 重新签发（file
   `channel+ft+id`、browser `channel+random`、attachment access 自己的 channel）；
   event 订阅与 history token 由 provider 生成并原样透传给客户端（release 时转回
   provider），不经过 access 重签。
4. **单写者**：同一终端只有一个 resize owner；类型化方法带 confirm 与 owner
   CAS；`access.call` 路径由程序自律（文档明示）。
5. **本地信任**：0600 + owner-only；不做 challenge。远程必须 remoteauth（身份
   proof + grant + DTLS binding）。
6. **失败面**：provider 不可用返回 typed unavailable，不影响 auth/file/proxy；
   未知命令 fail closed。
7. **生命周期**：`anytty pool start` 先起 pool 等 provider 就绪再起 access；
   `access restart` 不动终端；`pool restart --keep-access` 不动 access；
   auto-start 失败按"自己启动的 PID"精确回滚，不留半栈。
8. **记录**：runtime record（`<canonical>.pool.json`）由 pool owner 创建、CLI
   原子补写 access 身份；旧 `<sock>.daemon.json` 只读回退。
9. **显式例外**：TUI `kind: command` 端点在宿主本地 PTY 跑 argv（如 ssh 客户端），
   不经 access/pool；它不是 managed 终端。
10. **错误模型**：provider wire code → `terminal.CodedError`/typed 哨兵 →
   既有 `apipb.ApiErrorCode`（不新增客户端错误码）→ tui2 `RESPONSE.error` → CLI
   退出码；CLI 对用户可读错误有独立分类。
11. **增量视图布局复用**：`VIEW_DELTA` 经 COW 路径复制后，未改子树与上一帧共享
   同一 `*pb.Box` 指针；宿主据此复用已解算的 `kernel.Node` 与 layout 帧，但仅在
   该子树的**绝对 rect 不变**时复用（帧存绝对坐标），否则重新求解。复用以
   指针身份 + rect 相等为硬条件，正确性由"随机补丁序列 vs 全量重解"的等价性
   fuzz 保证；全量 VIEW 不参与复用。增量提交的簿记（`*pb.Box→*kernel.Node` 映射
   及其 prune、focus 复用、节点计数）也是 O(changed)：映射跨提交持久、按补丁
   分离出的指针剪枝并带容量兜底，`focus` 仅在补丁触及焦点相关字段时重走，
   `boxCount` 增量维护；layout 的 rect 索引也是树形的（复用子树按引用拼接，
   不每帧物化平坦 map）。同一帧多补丁若发生"先克隆、后分离该克隆"，无法精确
   剪枝（分离根不在映射中），此时按提交后的树重建一次映射；单补丁稳态不触发。

## 7. 包树与依赖

```
access/                     服务端（客户端入口与能力 owner）
  server/                   协议服务器 + family 路由 + terminalmap(apipb↔providerv1)
  files/ (+transfer)        文件服务/续传/窗口/压缩/进度/路径安全
  proxy/ storage/           本地终结能力
  runtime/                   identity/AccessStore/pairing/Cloud enrollment 记录
  localstate/                主机路径策略（CLI 与 access 共用）
  transport/                 客户端 transport（client/protocol/securetransport/ticket/webrtc）
  conformance/ sdk/          协议一致性；access SDK 目前只有 README/包声明（Go 直接 import access/engine）
  remote/ direct/ cloud/    remoteauth 会话、Direct listener 与 `<sock>.direct` 记录、Cloud agent
  sessions/ gateway/        会话登记/撤销；字节透明 relay
  provider/terminal/        provider 契约（typed + Capabilities）
  provider/pool/            pool 适配器（typed → providerv1 wire）
  contract/                 access-owned DTO（auth/remote/file/storage）
  engine/                   客户端引擎（Flutter/CLI/TUI 共用；不依赖服务端）
  accessrun/ cmd/           组合入口与独立二进制
pool/
  core/                     终端真值（PTY/history/live/events/path）
  provider/                 providerv1 服务端（wire）
clients/
  cli/                      客户端 + 生命周期 + registry/配对/维护前端；默认入口拉起 tui2
  tui/                      tui2 host + runtime/kernel/components + 布局 SDK（Go/Py/TS）+ shell
  flutter/ web/ ui/         移动端/Web/共享 UI
shared/                     transport/remoteauth/runtimepath/securefs/...（无业务 owner 语义）
api_mapping/                仅 access 侧使用的 apipb↔DTO 投影
proto/                      access wire/apipb、provider/v1、ui(tui2)、cloud
```

## 8. 客户端与 SDK

- **tui2 布局 SDK**（Go/Python/TS）：view/事件/方法 + STREAM + 增量视图
  （`VIEW_DELTA`，§4.3）；组件模型 `sdk/app`（Go）与组件库 `sdk/widgets`（Go/Python/TS 三语言对齐）。
  一致性由 `tui2-sdk-verify` + `clients/tui/conformance/fixtures.jsonl` 三语言
  自证（当前 14/14）。
- **access 能力**：经 host 统一桥接（`access.call`/`access.stream.*`），插件不
  需要自己实现 access wire；这是"统一 SDK（layout + access 两层）"的当前形态。
- **CLI**：`pool`/`access` 生命周期、`terminal`/`file`/`endpoint`/`pair`/`cloud`/
  `config`/`web` 等命令；本地与远程同一套 access wire。
- **Flutter/Web**：wire 未变，不需要随内部重构发版。

## 9. 配置、状态与日志

| 类别 | 路径 | owner |
|---|---|---|
| endpoint registry | `$XDG_CONFIG_HOME/anytty/endpoints.yaml` | CLI 写、TUI/CLI 读 |
| TUI/shell 配置 | `$XDG_CONFIG_HOME/anytty/tui2.json`（`ANYTTY_TUI2_CONFIG`） | 布局程序 |
| pool 配置 | `tui-v3.yaml` 的 `pool:` 段（`daemon:` 只读回退） | pool |
| identity/凭据/grant | `$XDG_STATE_HOME/anytty/remote-v2/{identity,access,credentials}` | access（identity/access）/CLI（credentials） |
| history | `$XDG_STATE_HOME/anytty/history-v2` | pool（store 语义）；CLI 仅 offline 维护前端 |
| 传输续传记录 | `$XDG_STATE_HOME/anytty/transfers` | access/files |
| 日志 | `anytty.log`（pool）、`anytty-access.log`（access）、`tui2.log`（host） | 各自进程 |
| runtime | canonical / `.provider` / `.pair` / `.direct` / `.pool.json` | access / pool / access / access / pool+CLI 补写 |

## 10. 测试与验收

- 单元/契约：`pool/core`（终端真值）、`access/files`（路径/续传/窗口）、
  `access/provider/pool`（typed facade e2e）、`clients/tui/{runtime,kernel,endpoint}`。
- e2e：`access/server`（terminal/file/storage/revoke）、`access/engine/adapter/
  {direct,ssh}`、`access/cloud`（真实 Pion DataChannel + mock AgentGateway）、
  `clients/cli`（tmux smoke/cloud/生命周期隔离）。
- 一致性：`tui2-sdk-verify` 三语言 14/14；`generate_application_api.go -check`
  与 `proto/ui` regen 幂等。
- 发布：`make test-release`（归档含 anytty/tui2/tui2-shell + 隔离栈生命周期）。
- 已知 flaky（基线可复现，非架构问题）：direct attach、gateway restart、
  tui/endpoint offline 窗口。

## 11. 已知缺口与演进方向

1. **tmux provider**：仅有 typed 契约占位（Capabilities 全 false），翻译层未实现；zellij 无代码。
2. **access 客户端 SDK**：`access/sdk` 目前只有 README 与包声明（Go 使用者直接
    import `access/engine`）；第三方若不经 TUI host 直连 access，需要自行实现
    framing/凭据/重连（当前推荐经 host 桥）。
3. **生产 Cloud 验证**：edge/controller 目前是 mock/本地，未做真实环境验证。
4. **移动端真机**：Android/iOS 构建与设备验证未做（wire 兼容）。
5. **PROXY/真实客户端 IP**：relay 路径未透传。
6. **多客户端共享/终端 tags/旧 TUI 细节**：tui2 parity 大项已闭环，剩余为体验项。
7. **自动恢复**：CLI 不内建 watchdog，交给系统服务管理器。
8. **provider 能力协商已就位**（Capabilities），但尚未接入路由时的 fail-fast 提示。

## 12. 设计决策记录（简表）

| 决策 | 理由 |
|---|---|
| pool 只做终端 provider | 收缩攻击面/职责面：无身份、无账本、无网络；文件/storage/proxy 都在 access |
| provider wire 独立 + typed facade | 客户端 wire 与内部 provider 解耦；typed 契约让 tmux 只做翻译；Capabilities 表达子集 |
| `access.call` 透明转发 | 程序用生成客户端即可获得 access 全能力，无需每语言重实现 wire；纪律由程序负责 |
| 本地一律走 access | 终端持久化/多客户端一致/统一升级路径；宿主 PTY 只留 `kind: command` 例外 |
| socket 代际命名 `v3-wire7` | 新旧栈（老单进程 daemon）天然隔离，升级不抢 socket；wire 版本仍在名字里 |
| 单二进制三角色 | 分发简单；进程隔离保证 access/pool 独立升级与崩溃隔离 |
| DTO 归 owner（access/contract vs pool/core） | 避免 access 依赖 pool 实现；映射单点化（terminalmap/api_mapping） |
