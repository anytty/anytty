# AnyTTY Access Gateway（GATEWAY.zh-CN）

> 范围：`access/`（协议服务器 + 能力路由 + runtime）与 `daemon/`（terminal provider）。
> 状态：Phase 0–4 完成（2026-09-19）。access 是唯一客户端入口与能力 owner；
> daemon 是纯终端 provider。access wire 保持不变。
> 总览见 `docs/HANDOFF.zh-CN.md`，目标设计见 `access/docs/ARCHITECTURE.zh-CN.md`。

## 0. 一句话

`access` 终结客户端 access wire 并按能力路由：鉴权/配对/Cloud、文件管理、
storage、端口转发在 access 本地执行；终端请求转给 terminal provider（当前 =
daemon，`<canonical>.provider`）。daemon 不认识远程会话、不持有身份/账本/文件，
也不再有网络出口。

## 1. 职责边界

### 1.1 daemon（`daemon/{core,remote,cloud,cmd/anyttyd}`）

- 只监听本地 unix socket `<canonical>.provider`（`0600`）；
  无网络 listener（`TestDaemonHasNoTCPListeners`）。
- 能力：PTY 字节流 + snapshot/revision + 无限历史 + path.defaults/
  path.list_directories + terminal events + attachment registry。
- `client_access.*` / `cloud.*` / `file.*` / `storage.*` / browser proxy 命令
  fail closed（unsupported）；daemon 不再挂载 ClientAccessService/RemoteService，
  也没有 control RPC 客户端。
- 旧的 file/storage/browser 实现与测试已删除；生成的 API mapping 因仓库缺失
  `scripts/generate_application_api.go` 未重生成，daemon port 上保留 fail-closed
  stub（见 HANDOFF §7）。

### 1.2 access（`access/`）

| 组件 | 作用 |
|---|---|
| `access/accessrun` | `anytty-access` / `anytty access run` 共享组合：runtime、pairing、Cloud、relay、Direct、协议服务器 |
| `access/server` | 协议服务器（Hello/请求预算/stream/事件）+ family 路由 |
| `access/provider/terminal` | terminal provider 契约（`Execute`/`OpenStream`/`Events`）；tmux 占位返回 unsupported |
| `access/provider/daemon` | daemon provider：owner-only dial + Hello v7，命令 correlation 原样透传 |
| `access/files` + `access/files/transfer` | 文件 metadata/传输、断点续传、自适应窗口、进度合并、路径安全 |
| `access/proxy` | browser proxy：access 主机拨号，双向流控 |
| `access/storage` | opaque KV + 变更广播 |
| `access/runtime` | DeviceIdentity + AccessStore（进程 owner lock）、pairing listener、Cloud、Direct 记录 |
| `access/direct` | Direct signaling/ICE-TCP listener、LAN discovery |
| `access/gateway` | tcp/unix 字节透明 relay（`--allow`/`--pair-token`） |
| `access/sessions` | grant 会话登记；revoke/过期关闭客户端 transport |
| `access/cmd/anytty-access` | 独立二进制入口（保持原有 flags + JSON config） |

监听面：

| listener | 归属 | 默认 | 说明 |
|---|---|---|---|
| canonical unix | `access/server` | `$XDG_RUNTIME_DIR/anytty-v2-wire7.sock` | 客户端入口；`0600`；本地免鉴权 |
| provider unix | `daemon/provider` | `<canonical>.provider` | provider 协议（[channel:2][type:1]），只有 access 连 |
| pairing unix | `access/runtime` | `<canonical>.pair` | remoteauth v2 PairingExchange |
| Direct signaling/ICE-TCP | `access/direct` | `--route` 显式配置 | 网络入口只由 access 打开 |
| relay tcp/unix | `access/gateway` | `--listen` 显式配置 | 字节透明，不解析 frame |

## 2. 数据路径

```
 客户端 ── access wire ──▶ access/server（Hello/请求/流/事件）
                              │ FamilyTerminal ──▶ provider ──▶ daemon（PTY/history/live）
                              │ FamilyFile/FamilyProxy/FamilyStorage ──▶ access 本地
                              │ FamilyAuth ──▶ access/runtime（identity/pair/cloud）
                              ▼
                        canonical unix（owner-only，0600）
```

- 本地连接信任边界是 `0600` socket，不做 identity challenge；
  远程（Direct/Cloud）必须先在 access 完成 remoteauth，再由
  `access/server.ServeTransport` 服务同一 protocol loop，并由
  `access/sessions` 登记 grant（撤销/过期关闭 transport）。
- relay 路径字节透明：access 不解析 frame；授权是网络层边界。relay 的本地
  转发目标是 canonical access socket（remote peer 说 access wire），不是
  `.provider`；daemon 只在 provider 协议上服务 access。

## 3. 能力路由（`access/server/router.go`）

| family | 命令 | 处理 |
|---|---|---|
| FamilyTerminal | terminal.*、history.*、live、events、path.* | provider（daemon） |
| FamilyFile | file.* + download/upload/cancel | `access/files`（含 §11 传输优化） |
| FamilyProxy | browser.proxy.open + 流 | `access/proxy` |
| FamilyStorage | storage.* | `access/storage`（事件按既有订阅 token 广播） |
| FamilyAuth | client_access.*、cloud.* | `access/runtime` 直答 |
| resource release | access-issued token | 本地回收 binding；provider resource 由对应命令释放 |

错误映射：access 不新增客户端可见错误码；provider 错误按既有 typed envelope
返回，本地 handler 使用 `INVALID_REQUEST`/`NOT_FOUND`/`CONFLICT`/
`RESOURCE_EXHAUSTED`/`UNAVAILABLE`/`CANCELLED`。

## 4. 授权与生命周期

- 本地：owner-only 0600 socket，免鉴权。
- 远程：remoteauth（grant/DeviceHello/DTLS binding）在 access 完成；daemon
  只看到 access 的内部连接。
- 撤销/过期：`access/sessions.Registry` 按 GrantID 登记活动客户端 transport，
  revoke/TTL 实时关闭；access session 结束会释放 provider 连接、附件与桥接。
- Provider token 从不出 access：attachment/file/browser resource 的
  channel/token 都由 access 重新签发（同一不透明形状）。

## 5. 运行

```bash
# 推荐：同一 CLI 管理两个进程（daemon 先起，access 后起；日志分开）
anytty --socket /tmp/anytty.sock daemon start
anytty --socket /tmp/anytty.sock daemon status --json
anytty --socket /tmp/anytty.sock daemon stop

# 前台/独立进程
anytty --socket /tmp/anytty.sock daemon run            # 绑定 /tmp/anytty.sock.provider
anytty --socket /tmp/anytty.sock access run            # 绑定 /tmp/anytty.sock（canonical）
anytty-access --socket /tmp/anytty.sock \
  --route 0.0.0.0:41120 --listen tcp:127.0.0.1:7331 --allow 127.0.0.1/32 \
  --file-root /home/user --log-file /tmp/anytty-access.log
```

配置文件（`anytty-access --config`，flag 覆盖文件）：

```json
{
  "socket": "/tmp/anytty.sock",
  "listen": ["tcp:127.0.0.1:7331"],
  "route": "0.0.0.0:41120",
  "allow": ["127.0.0.1/32"],
  "file_roots": ["/home/user"],
  "pair_token_file": "/home/user/.config/anytty/access-token"
}
```

日志只写文件：daemon 日志 `anyttyd.log`；access 日志
`ANYTTY_ACCESS_LOG_FILE` 或 `$XDG_STATE_HOME/anytty/anytty-access.log`（0600）。

## 6. 验收与证据

| 验收 | 测试 |
|---|---|
| terminal 全链路（client→access→provider→PTY） | `access/server/terminal_e2e_test.go` |
| 文件 metadata/传输/续传/browser proxy（access 本地） | `access/server/files_e2e_test.go`、`access/files/*_test.go`、`access/proxy/proxy_test.go` |
| storage 本地终结 + 事件 | `access/server/storage_e2e_test.go` |
| revoke/过期释放 provider 桥接 | `access/server/revoke_e2e_test.go` |
| family 路由表 | `access/server/router_internal_test.go` |
| daemon 无网络监听 | `TestDaemonHasNoTCPListeners` |
| Direct/SSH/Cloud remoteauth→access Core | `access/engine/adapter/{direct,ssh}/integration_test.go`、`clients/cli/cloud_edge_e2e_test.go` |
| 生命周期双进程 + 独立日志 | `clients/cli/daemon_lifecycle_test.go` |
| CLI/tmux smoke | `clients/cli` tmux 系列 |
| 文件传输吞吐基准 | `access/files.BenchmarkFileDownloadStreaming` |

## 7. 已知缺口

- daemon 的 API Layer PlatformController 已收窄为 history/live/event；
  file/storage/auth/browser capability 在 admission 直接拒绝（terminal-only），
  access 侧不受影响。api_mapping 的 DTO helper 保留给 access/clients。
- tmux/zellij provider 只有接口占位。
- 旧 `tui/`、deadcode 清理未做。

## 8. codegen

- `go run ./scripts/generate_application_api.go [-check]`：从
  `proto/access/apipb/application_commands.csv` 生成四个 `.gen.go`；
  `internal/appcodegen` 测试保证幂等。
- `scripts/generate-access-proto.sh apipb/file.proto wirepb/terminal.proto`：
  staging 到历史逻辑路径后 buf 生成，保留目标文件 protoc 版本行。
- 文件传输可选字段（compression/progress）已落地：默认关闭，旧客户端 payload
  逐字节不变。
