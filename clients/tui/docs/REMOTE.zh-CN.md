# TUI v2 连接远程 Terminal（REMOTE）

> 状态：M27 + 连接层去重（M1–M5，共享 client 层）+ M30（日志接管 + 路由
> 裁剪，见 §3.1）。目标：调 UI 时能像老版本
> 一样连接**远程服务器上的 daemon** 并 attach 终端；布局程序（UI）零改动，
> 连接全部由共享 `client/` 层负责，tui2 只剩薄适配。
> 相关文档：`CLIENT_SHARING.zh-CN.md`（共享层最终态）、
> `ENDPOINTS.zh-CN.md`（endpoint 模型与 daemon 协议序列）、
> `CUSTOMIZE.zh-CN.md` §3（配置示例）、`PROGRESS.zh-CN.md` §2.13/§2.16。

## 1. 考古：老版本怎么连远程

### 1.1 老模型

老 client/TUI 的连接真值在 `~/.config/anytty/endpoints.yaml`（`version: 3`）：

```yaml
version: 3
default: local
endpoints:
  local:
    label: local-dev
    connect_mode: auto          # auto / on_demand / manual
    routes:
      local: { kind: local-unix, enabled: true, socket: "$ANYTTY_DEV_SOCKET" }
  server:
    routes:
      direct:  { kind: direct-webrtc-tcp, signaling_addresses: [...], ice_tcp_addresses: [...] }
      tunnel:  { kind: ssh-webrtc-tcp, ssh_host: ..., remote_ice_tcp_address: ... }
      cloud:   { kind: managed-webrtc, credential_ref: ..., relay_mode: auto }
```

一个 endpoint = 多条按优先级排序的 **route**；route kind 决定 connector：

| route kind | 代码路径 | 传输/鉴权 |
|---|---|---|
| `local-unix` | `client/endpoint/registry.go` 常量、`client/adapter/local/dial.go` | `shared/transport/unix`（zstd + 分片帧）→ wire Hello |
| `direct-webrtc-tcp` | `client/adapter/direct/dial.go` | daemon embedded signaling（`cmd/anytty/v3_direct_daemon.go`，`net.Listen("tcp", signaling/ice)`）+ Pion ICE-TCP DataChannel + DTLS 指纹绑定 + `remoteauth` capability 握手 |
| `ssh-webrtc-tcp` | `client/adapter/ssh/dial.go` | Go `x/crypto/ssh` direct-tcpip 隧道到远程 ICE-TCP listener，再走 direct 同一套信令/鉴权 |
| `managed-webrtc` | `client/adapter/cloud/dial.go` | Cloud `directory`/`client_gateway` 发现+信令+relay（`proto/cloud/v1/*`），enrollment/edge/binding |

客户端选择与 generation 在 `client/runtime/endpoint_supervisor.go`；TUI 只读
registry 快照（`tui/adapter/clientruntime/endpoint_connection.go`），拨号完全由
runtime/connector 拥有。旧 TUI 的“远程”对布局程序同样是透明的。

### 1.2 对 v2 有用的结论

- **远端 daemon 的 wire 没有 TCP listener**：core-v2 只在 unix socket 上监听
  `shared/transport/unix`（`core/server.go` + `shared/runtimepath`，默认
  `$XDG_RUNTIME_DIR/anytty-v2-wire7.sock`）。`daemon --route HOST:PORT` 起的是
  WebRTC signaling/ICE-TCP，不是 wire 端口，协议与鉴权完全不同。
- **P0 就是老 `local-unix` + ssh**：ssh 端口转发把远端 unix socket 变成本地
  socket/TCP 端口，v2 现有 local-unix dialer 零改动即可用；旧 TUI 的
  “`ssh host anytty attach`”命令式路径在 v2 就是 `command` endpoint（本地 PTY
  里跑 ssh），也没有改动。
- **`tcp` 是新语义**：v2 的 `connect_mode: tcp` = 连接一个字节透明的
  HOST:PORT，对端终止在 daemon transport（`ssh -L TCP→unix` 或 socat/Go 桥）。
  它复用与 local-unix 完全相同的 zstd 分片帧，因此对端只需要是 daemon socket
  的透明转发，不引入信令/DTLS/cloud 依赖。
- **direct/managed WebRTC 由共享层接管（最终态）**：tui2 不自己实现信令/Pion/
  DTLS/cloud enrollment，也已在 M1 删除旧的本地复制；这些 route 一律由
  `client/adapter/{direct,ssh,cloud}` 拨号，TUI 只从 CLI 写入的 registry 读取
  route 并消费 ready session（见 §2、`CLIENT_SHARING.zh-CN.md`）。

## 2. v2 架构（共享 client 层）

```
tui2-shell (UI，零改动)
   │ endpoint.sync / terminal.create / terminal.attach
   ▼
tui2 host ── endpoint.Manager（每 endpoint supervisor：退避重连/health/notice）
   │
   ▼
clients/tui/endpoint.sessionConn ← 唯一拨号入口
   │
   ▼
sharedClient（tui2 适配层，无协议实现）
   ├─ client/runtime.SessionOwner/ClientRuntime（generation/planner race）
   ├─ client/adapter/protocol.ApplicationClient（command + ResourceStream）
   └─ route adapter：local-unix / tcp 桥(→local) / direct / ssh
   ▼
RemotePTY（clients/tui/pty.PTY）── 快照 LiveScreenNext(0) + PTYOutput 流
   ▼
runtime.Terminal / ANSI parser / 组件（位置透明）
```

- 拨号真值：`client/endpoint` registry（CLI 写）+ `client/runtime`
  planner/dialer；tui2 不再有 `client.go`/`transport_tcp.go` 生产实现。
- `tcp`：`clients/tui/endpoint/tcp_bridge.go` 只做字节透明 unix↔TCP relay，连接仍由
  共享 local-unix route adapter 完成（帧/压缩由 `shared/transport/unix` 负责）。
- 远程链路上的 reconnect、重订阅、快照重建、resize owner CAS、kill/restart/
  remove 全部复用共享 session 语义；health=ok 在 reattach 完成后发布，所以
  "connected" 之后第一键一定落在可用 attachment 上。

## 3. 支持矩阵

| 方式 | 状态 | 说明 |
|---|---|---|
| `command` + `ssh host anytty attach` / `ssh host sh` | ✅ 已有 | v1 本地 PTY 路径；老命令式远程用法原样可用 |
| `daemon` + `local-unix` | ✅ 已有 | 本机 daemon socket |
| `daemon` + `local-unix` + `ssh -L local.sock:remote.sock` | ✅ P0 | 零代码；ssh 负责鉴权/加密，wire 端到端仍是本机信任模型 |
| `daemon` + `tcp` + `ssh -L 127.0.0.1:PORT:remote.sock` | ✅ 新 | ssh 唯一需要的能力是 TCP→remote unix socket 转发 |
| `daemon` + `tcp` + socat/`clients/tui/scripts/remote-bridge` | ✅ 新 | 无 ssh 环境（同机模拟/自管隧道）；桥必须字节透明 |
| `daemon` + `direct-webrtc-tcp` | ✅（CLI registry） | 由共享 `client/adapter/direct` 拨号；端点必须由 CLI 配对写入 registry |
| `daemon` + `ssh-webrtc-tcp` | ✅（CLI registry） | 由共享 `client/adapter/ssh` 拨号；同上 |
| `daemon` + `managed-webrtc`（Cloud） | ✅（CLI registry） | 由共享 `client/adapter/cloud` 拨号（controller 默认值同 CLI，`ANYTTY_CLOUD_CONTROLLER_*` 可覆盖）；缺 credential/enrollment 时共享 planner 给可读 no-route 错误 |
| 连接超时/拒绝 | ✅ | `dial <target>: ... connection refused` 进 notice；health=offline |
| 断线退避重连 + 重订阅 + 快照重建 | ✅ | 250ms..5s 退避；重连后重 attach 同一 terminal 并重拉快照 |
| resize owner CAS（epoch） | ✅ | 远程链路同一实现；冲突返回可读错误 |

### 3.1 日志与路由策略（M30）

**日志文件**：tui2 宿主在接管终端之前就把 Go 标准 `log`（以及共享
`client/` 层的所有 `anytty connect` / `anytty network attempt` /
`anytty webrtc …` / `anytty cloud connect …` 诊断）重定向到文件：

- 路径优先级：`-log-file PATH` > `TUI2_LOG_FILE` >
  `$XDG_STATE_HOME/anytty/tui2.log`（默认 `~/.local/state/anytty/tui2.log`）。
- 文件 0600、追加写、目录自动创建；首行 `tui2 start version=… routes=…`
  记录本次实际生效的路由策略。
- host 自身的异常（布局程序崩溃、输入路由错误、endpoint notice）也只进
  该文件 + 程序 notice，**alt screen 上不会再出现任何非 TUI 输出**；
  布局程序的 stderr 本就不接终端。

```bash
tail -f "${XDG_STATE_HOME:-$HOME/.local/state}/anytty/tui2.log"
# 只看共享层连接诊断：
grep -E 'anytty (connect|network attempt|webrtc|cloud)' ~/.local/state/anytty/tui2.log
# 显式指定（测试/多开）：
TUI2_LOG_FILE=/tmp/tui2-a.log tui2 -shell tui2-shell
```

**路由策略**：默认只拨 tui2 支持的传输，避免老 registry 的
webrtc/cloud route 造成刷屏与 4 秒阻塞：

| route kind | 默认 | 开关 | 行为 |
|---|---|---|---|
| `local-unix`（含 `tcp` 桥、`command`） | ✅ 总是 | — | 正常连接 |
| `ssh-webrtc-tcp` | ⚠️ 共享 store 里有可用凭据时 | 默认在策略内 | 无凭据则不进 planner，端点 offline |
| `direct-webrtc-tcp` | ❌ | `TUI2_ROUTES=…,direct-webrtc-tcp`（别名 `direct`/`webrtc`） | opt-in 后由共享 direct adapter 拨号，诊断只进日志文件 |
| `managed-webrtc`（Cloud） | ❌ | `TUI2_ROUTES=…,managed-webrtc`（别名 `cloud`；需 Cloud 客户端/凭据可用） | 同上 |

- `--routes` 优先级高于 `TUI2_ROUTES`；值逗号分隔，`all` 开启全部，
  未知值启动即报错。例：`TUI2_ROUTES=local-unix,ssh,cloud tui2`。
- 老 registry 多 route 端点的投影降级顺序：`local-unix` > `ssh-webrtc-tcp`
  > `direct-webrtc-tcp` > `managed-webrtc`；能降级到 unix/tcp/ssh 的端点
  直接可用，只有 webrtc/cloud 的端点保持 `endpoint · offline` 并给一行
  可读 notice（planner 的 no-eligible-route / 缺凭据错误）。
- 默认策略下 webrtc/cloud 端点**不发起任何拨号**，因此没有
  `webrtc selected_pair` 噪音，也不占用 dial timeout；开启后失败诊断全部
  在日志文件里，终端只看到 TUI notice。

## 4. 手动复测命令（照抄连自己的服务器）

假设远端 daemon 用默认 socket（`$XDG_RUNTIME_DIR/anytty-v2-wire7.sock`，
通常是 `/run/user/1000/anytty-v2-wire7.sock`），本地 `tui2.json` 路径见
`$ANYTTY_TUI2_CONFIG` 或 `~/.config/anytty/tui2.json`。

### 4.0 CLI 配对/新增 → TUI 连接（主路径，照抄）

TUI host 启动时读取共享 registry（`~/.config/anytty/endpoints.yaml`）；
配对/管理永远由 CLI 负责，TUI 只读：

```bash
# 1) 建隧道（本机 daemon 直接跳过这步）：
ssh -N -L /tmp/anytty-remote.sock:/run/user/1000/anytty-v2-wire7.sock user@server &

# 2) CLI 写 registry（local-unix 例；direct/ssh 用 `endpoint add direct/ssh`，
#    pair 流程用 `anytty pair create|import`）：
anytty endpoint add local server --socket /tmp/anytty-remote.sock --label server
anytty endpoint list

# 3) 起 TUI：Ctrl-F 打开 picker → 出现 server 分组与其终端 → enter attach
tui2
```

- 选 daemon 终端行 = attach 已有终端；选 `󰌷 <label> endpoint · offline`
  占位行 = 在该端点新建终端，离线时状态行给出可读错误（不崩）。
- `tui2.json` 的同名 `endpoints[]` 仍解析，但共享 registry 优先（仅迁移）。
- direct/ssh/cloud 端点由共享 `client/adapter/{direct,ssh,cloud}` 拨号；
  cloud 的 enrollment/credential 管理仍在 CLI（见 `CLIENT_SHARING.zh-CN.md` §5）。

### 4.0.1 dev/v2 直接读老 registry（不重新配对，照抄）

dev/v2 跑在隔离 XDG（如 `~/.config/anytty-dev`）时，用只读覆盖直接看生产
registry；显式文件在前、按 name 合并（显式优先），dev 自己的 registry 仍然
可用，老文件**只读不写回**。也可用宿主参数 `tui2 -endpoints <path>`（优先于
环境变量；多个路径用 `:` 分隔）：

```bash
TUI2_ENDPOINTS=~/.config/anytty/endpoints.yaml bash clients/tui/scripts/run.sh
```

缺失/损坏的显式文件只会在状态行给出可读 warning，不崩；配对/管理仍归 CLI
（`anytty endpoint add ...`）。验收见 `CLIENT_SHARING.zh-CN.md` §6。

### 4.1 P0-A：ssh 转发 unix socket（零代码，推荐）

```bash
# 本地终端：把远端 daemon socket 拉到本机 /tmp/anytty-remote.sock
ssh -N -L /tmp/anytty-remote.sock:/run/user/1000/anytty-v2-wire7.sock user@server

# 另开一个终端：写配置并跑 TUI
cat >/tmp/tui2-remote.json <<'JSON'
{
  "endpoints": [
    { "name": "server", "kind": "daemon", "label": "server", "socket": "/tmp/anytty-remote.sock" }
  ]
}
JSON
tui2-shell --config /tmp/tui2-remote.json   # 或 $ANYTTY_TUI2_CONFIG=/tmp/tui2-remote.json tui2
```

在远端先 `anytty daemon start`（或 systemd 服务）并用 `anytty v3 new` 建终端，
本地 TUI 的 picker 会分组列出 `server` 端点与其终端，attach 后输入/回显/
resize/kill 全通。

### 4.2 P0-B：ssh 把远端 socket 暴露成本地 TCP 端口（新 `tcp` 模式）

```bash
ssh -N -L 127.0.0.1:17777:/run/user/1000/anytty-v2-wire7.sock user@server

cat >/tmp/tui2-tcp.json <<'JSON'
{
  "endpoints": [
    { "name": "server-tcp", "kind": "daemon", "label": "server", "connect_mode": "tcp", "address": "127.0.0.1:17777" }
  ]
}
JSON
tui2-shell --config /tmp/tui2-tcp.json
```

没有 ssh 或想让本机第二个 daemon 当“远程”时，用仓库自带桥：

```bash
go run ./clients/tui/scripts/remote-bridge -listen 127.0.0.1:17777 -unix /path/to/daemon.sock
# 或 socat TCP-LISTEN:17777,reuseaddr,fork UNIX-CONNECT:/path/to/daemon.sock
```

### 4.3 P0-C：老命令式 `ssh host anytty ...`

```json
{ "endpoints": [
  { "name": "shell", "kind": "command", "label": "server-shell", "argv": ["ssh", "user@server"] },
  { "name": "attach", "kind": "command", "label": "server-attach", "argv": ["ssh", "user@server", "anytty", "attach", "<id>"] }
] }
```

### 4.4 快速验证隧道是否可用

```bash
# 隧道自身：连接不报错即 listener 存在（daemon 会因没有 Hello 关掉它）
python3 -c 'import socket;s=socket.socket();s.settimeout(1);s.connect(("127.0.0.1",17777));print("tunnel ok")'
# 端到端：TUI picker 出现 endpoint，attach 后 `echo REMOTE-OK` 回显
```

## 5. 剩余缺口（均已给可读错误）

1. **Cloud composition 复制**：managed-webrtc connector 已装配并可拨号；
   controller 地址/环境变量解析与 `cmd/anytty` 各一份（只读、默认值相同），
   enrollment/identity 管理仍在 CLI。后续把 composition 下沉
   `client/runtime`（`CLIENT_SHARING.zh-CN.md` §5）。
2. **`tcp` 无 registry kind**：CLI registry 没有裸 HOST:PORT route；tui2 的
   `tcp` 仍是 tui2.json 兼容入口（内部走 tcp_bridge + 共享 local route）。
   需要 CLI 统一时应在 `client/endpoint` 增加 route kind 与共享 dialer。
3. **每 endpoint 超时/代理/known_hosts 等 ssh 细节**：交给用户 ssh_config，
   v2 不搬入内置 ssh 客户端到 tui2；registry 的 ssh-webrtc-tcp 由共享
   `client/adapter/ssh` 处理。
4. **cloud enrollment（`anytty cloud enroll/enable/edge`）与 ticket/identity
   管理**：独立于 terminal attach 的目标，未纳入本轮。
5. **WebRTC/Cloud 默认不拨号（策略，不是缺口）**：`direct-webrtc-tcp` /
   `managed-webrtc` 只有 `TUI2_ROUTES`/`-routes` 显式开启才进入 planner；
   默认策略只拨 `local-unix` 与凭据可用的 `ssh-webrtc-tcp`，老 registry 的
   webrtc/cloud 端点保持 offline + 一行 notice（见 §3.1）。
