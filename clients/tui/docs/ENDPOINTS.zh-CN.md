# TUI v2 Endpoint 连接策略（ENDPOINTS）

> 状态：M23 + M27（remote tcp / ssh 隧道）+ 连接层去重（M2）。
> 本文是 endpoint 模型、终端池协议调用序列与 v1 支持矩阵的规格。
> 相关文档：`CLIENT_SHARING.zh-CN.md`（tui2 ↔ client 共享层设计，强制）、
> `REMOTE.zh-CN.md`（远程接入考古与手动命令）、
> `PROTOCOL.zh-CN.md`（wire 方法表）、`CUSTOMIZE.zh-CN.md` §3.1
> （配置示例）、`RECOMMENDED_CONFIG.zh-CN.md` §4（老 v3 模型对照）、
> `PROGRESS.zh-CN.md` §2.10/§2.13（实现与验收）。

## 1. 考古：老 endpoint 模型

### 1.1 老 endpoints.yaml（v3，终端池 连接策略）

`git show HEAD:tui/docs/tui-v3.recommended.yaml` 只定义**外观与推荐键位**
（profile/theme/chrome/footer），不定义连接。真正的连接策略在
`~/.config/anytty/endpoints.yaml`（隔离开发环境由 `scripts/anytty-dev.sh
endpoints` 生成）：

```yaml
version: 3
default: local
endpoints:
  local:
    label: local-dev
    enabled: true
    connect_mode: auto          # auto / on_demand / manual
    routes:
      local:
        kind: local-unix
        enabled: true
        socket: "$ANYTTY_DEV_SOCKET"
  # 老 终端池 另支持（v2 未实现）：
  # direct: { kind: direct-webrtc-tcp, signaling_addresses: [...], ice_tcp_addresses: [...] }
  # cloud:  { kind: managed-webrtc, credential_ref: ..., relay_mode: auto }
```

要点：一个 endpoint = 一组按优先级排序的 **routes**，每条 route 有 kind 与
transport 参数；`connect_mode` 决定何时拨号（auto 冷启动、on_demand 首次使用、
manual 手动）；终端池 侧维护连接状态/租约/重连。`scripts/anytty-dev.sh` 的
`write_endpoints()` 固定写 `version: 3 / default: local / connect_mode: auto /
routes.local.kind: local-unix / socket: $ANYTTY_DEV_SOCKET`，所有 CLI/TUI 必须
显式 `--socket <dev socket>`，不碰生产配置。

### 1.2 终端池协议调用序列（wire v7，`proto/wire` + `proto/apipb`）

传输不是裸帧：本地 unix 走 `shared/transport/unix`（packet 分片 + zstd 压缩），
其上是 `proto/wire` 帧（`u16 channel | u8 type | u32 len | payload`），
control channel 0 承载 `Hello/Request/Response/Error/Event`。完整序列：

1. **连接/握手**：`DialContext(socket)` → `TypeHello{Version:7, Client}` →
   终端池 回 `TypeHello{Version:7, Server}`；版本不符或未握手就发请求会被 终端池
   拒绝并断开（`unsupported wire version` / `Hello is required`）。
2. **列终端**：`api.execute` + `TerminalListCommand` →
   `TerminalListResult{terminals[]}`（`TerminalInfo{ref,name,command,size,state,
   exit_code,attachment_count,...}`）。`TerminalDefaultsCommand` 提供默认
   command/cwd。
3. **attach（订阅 surface/revision）**：`TerminalAttachCommand{terminal,
   mode:COLLABORATOR, resize_policy:OWNER, surface_id, view_id, operation}` →
   `TerminalAttachResult{attachment{resource,terminal,operation,surface_id,view_id},
   size, resize_control}`。`ResourceHandle.opaque_token` 前 2 字节是 终端池 分配的
   stream channel；客户端在该 channel 发 `TypeBootstrapDone` 后 终端池 回
   `TypeStreamReady`，随后持续推送 `TypePTYOutput`（原始 PTY 字节）、
   `TypeSyncLost`（丢帧）、`TypeClosed{code}`（进程退出/流关闭）。
   屏幕快照走 `LiveScreenNextCommand{terminal, observed_revision:0}` →
   `NativeScreenResult{live_revision,size,full_replace,row_replacements,cursor,
   modes}`：revision 是 终端池 live screen 的权威版本，attach/重连后用它建立画面，
   之后 raw stream 只提供增量（**快照权威，不复播历史**）。
4. **输入**：`TerminalInputCommand{attachment, operation, data}`（PTY 原始字节，
   鼠标转义序列同样作为字节输入）。
5. **resize（owner CAS/epoch）**：`TerminalResizeCommand{attachment, operation,
   size, resize_policy:OWNER, take_ownership, expected_owner_epoch}`；首次 resize
   用 `take_ownership=true` 拿 owner，之后带 `expected_owner_epoch` 做 CAS；失败
   回 `ResizeControl{can_resize:false,reason:FOLLOWER/SIZE_LOCKED/OBSERVER,
   ownership{owner_surface_id,owner_view_id,epoch}}`，客户端必须给出可读错误而不是
   静默重试。
6. **kill/remove**：`TerminalKillCommand` 终止进程但保留记录与 history（槽显示
   `[exited N]`）；`TerminalRemoveCommand` 删除已退出记录。`TerminalRestartCommand`
   按 终端池 保存的 process spec 原地重启。
7. **detach/关闭**：客户端可先在 channel 发 `TypeClosed` 停流，再
   `TerminalDetachCommand{attachment, operation}` 释放 attachment；连接关闭时
   终端池 统一回收本连接的全部 attachment。退出 TUI 只 detach，不 kill。

所有 mutating command 必须携带与 `RequestContext.session` 一致的
`OperationStamp`（attach/input/resize/detach）；`RequestContext` 必须带
`request_id` 与 `api_version.major=1`。这些校验在 终端池 的 API Layer 是硬性的。

## 2. v2 映射：配置、picker、协议参数

### 2.1 配置字段（`tui2.json` `endpoints[]`）

| 字段 | 说明 |
|---|---|
| `name` | 协议 endpoint id，成为 source id `terminal:<endpoint>:<id>`；禁 `:` |
| `kind` | `command`（缺省，本地 PTY 跑 argv）或 `daemon`（连接已有终端池） |
| `label` | picker 展示名，缺省用 `name` |
| `address` | `tcp` 模式必填的 `HOST:PORT`：对端必须是 终端池 transport 的透明隧道（`ssh -L TCP→remote socket` 或 socat/`clients/tui/scripts/remote-bridge`） |
| `socket` | `daemon` + `connect_mode=local-unix` 必填的 unix socket 路径（也可以指向 `ssh -L` 转发的本地 socket） |
| `argv`/`cwd`/`env` | command endpoint 的进程参数；终端池 endpoint 下是"新建终端"的可选 command/cwd/env（空 argv = 终端池 默认 command） |
| `connect_mode` | `local-unix`（终端池 缺省）、`tcp`、`direct-webrtc-tcp`（不做）；未知值配置报错 |

`kind` 在配置层校验：command 必须有 argv；终端池 + local-unix 必须有 socket；
终端池 + tcp 必须有 address；webrtc 允许配置（picker 仍显示），连接时给可读错误。

### 2.2 picker 展示与分组

- shell 启动收到 HELLO 后，对每个 `kind=daemon` endpoint 发 `endpoint.sync`
  （host 注册并后台连接），终端池 终端清单随后作为 `sources` 到达。
- 配置了任意 终端池 endpoint 时 picker 按 endpoint 分组：组头一行
  `  <endpoint>`，组内是 source 行（信息列 `<endpoint> · live/exited/offline/
  connecting`）与配置 endpoint 行（`󰌷 <label>`，信息列 `endpoint · daemon
  <connect_mode>`）；`+ New terminal` 归入 `local` 组。没有 终端池 endpoint 时
  保持 v1 平铺像素（兼容规则）。
- 键盘/鼠标只落在可选项上，组头不可选，因此 v1 交互与既有验收不受分组影响。

### 2.3 协议参数（`MethodParams`，append-only 字段 17..20）

- `terminal.create{endpoint, argv?, cwd?, env?, kind?, socket?, address?, connect_mode?}`：
  终端池 endpoint 下 host 先 `TerminalCreate`（空 argv 用 `TerminalDefaults`），
  以 终端池 返回的 terminal id 作为 source id，再 attach。
- `terminal.attach{endpoint, id, fit?, kind?, socket?, address?, connect_mode?}`：
  host 用 kind/socket/address/connect_mode 注册 endpoint 后 attach 已有 终端池 终端。
- `endpoint.sync{endpoint, kind, socket, address, connect_mode}`：注册 + 后台连接 +
  发布清单；host 立即应答，sources 事件随后到达。
- `sources` 的 `health`：`ok`（已连接）、`connecting`、`offline`（断线/拨号失败）。
  断线只改 health 并发 `notice`，不把本地终端标成 exited。

### 2.4 host 侧行为

- 一个 endpoint 一个连接 supervisor：`health=connecting → ok`；
  连接失败/断开 → `offline` + `notice`（`endpoint <name> offline: ...`），
  指数退避重拨（250ms..5s）。
- 已 attach 的终端在重连后**重新 attach 同一 terminal id**、重新拉
  `LiveScreenNext(0)` 快照并继续 raw stream；本地 `Read` 在断线期间阻塞而不是
  EOF（终端不因断线被判定退出）。
- `terminal.kill` → 终端池 `TerminalKill`（槽显示 `[exited N]`）；
  `terminal.remove` → 终端池 `TerminalRemove`；`Ctrl-E` → 终端池
  `TerminalRestart` 后原地重 attach；TUI 退出只 detach。
- 终端池 终端与本地 PTY 共用 `runtime.Terminal`/组件/输入路由/scrollback，
  位置透明；resize 失败（owner CAS 冲突/尺寸锁）返回可读错误。

## 3. v1 支持矩阵

| 能力 | command | 终端池 local-unix | 终端池 tcp | direct-webrtc-tcp / managed-webrtc |
|---|---|---|---|---|
| 配置 + picker 展示 | ✅ | ✅（分组 + 状态） | ✅（分组 + 状态，`address` 必填） | 配置可写，连接报"不做" |
| list / attach / input | 本地 PTY | ✅ 真实 终端池 | ✅ 真实 终端池（`ssh -L`/桥） | ❌ |
| resize（owner CAS/epoch） | 本地 winsize | ✅ 可读错误 | ✅ 可读错误 | ❌ |
| kill / remove / restart | 本地进程 | ✅ | ✅ | ❌ |
| 断线重连 + health/notice | 不适用（本地） | ✅ 重订阅 + 快照权威 | ✅ 重订阅 + 快照权威 | ❌ |
| 远程转发（ssh / tcp） | argv 里跑 `ssh host anytty attach` | 经 `ssh -L local.sock:remote.sock` 可用 | 经 `ssh -L 127.0.0.1:PORT:remote.sock` 或 socat/`remote-bridge` 可用 | ❌ |

去重边界（`CLIENT_SHARING.zh-CN.md`）：tui2 只实现 `local-unix`/`tcp` 两种自持
transport；`direct-webrtc-tcp`/`managed-webrtc` 的唯一实现是 `client/adapter/*`
（共享层从 CLI 管理的 `endpoints.yaml` 读取），tui2 不再复制 dialer，遇到这两种
connect_mode 报可读错误并指向共享层。

不支持项明确标注：`direct-webrtc-tcp` 与 `managed-webrtc` = v2 不做（不引入
信令/中继/凭证）；远程接入的完整考古、配置示例与 P2 待办见 `REMOTE.zh-CN.md`。

## 4. 断线/错误路径（测试口径）

- 断线：health offline + notice；已 attach 终端画面冻结但 `Read` 不 EOF；
  重连后重新 attach + 快照重建，旧输出不重放。
- 拨号失败（socket 不存在/被删、tcp 拒绝/超时）：notice 含目标
  （socket 路径或 `HOST:PORT`）与原因；picker endpoint 行仍在。
- 不支持 connect_mode：`direct-webrtc-tcp is out of scope for v2`；
  tcp 缺 `address` 在配置层报 `address is required for connect_mode tcp`。
- attach 不存在的 terminal：`terminal pool error 6: NOT_FOUND: ...` 可读错误。
- resize 非 owner：`resize <id> denied: attachment is a resize follower
  (owner view ..., epoch N)`。
- 单测用进程内假 终端池（同协议最小服务端）覆盖 list/attach/input/resize/
  kill/断线重连/错误路径，并对 tcp 帧与 `shared/transport/unix` 做双向兼容
  验证：`go test -count=1 -race ./clients/tui/endpoint/`。
- 端到端用隔离 dev 终端池：`clients/tui/scripts/acceptance.sh` 的
  "daemon endpoint: local-unix / offline and unsupported modes /
  remote via tcp / ssh tunnels / ssh unix-socket forward" 段。
