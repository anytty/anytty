# 交接报告（新 context 从这里开始）

> 状态时间：目标架构 Phase 0–4 + T1–T5 全部完成（access = 唯一客户端入口与
> 能力 owner；daemon = 纯 terminal provider，只在 `.provider` 上说 provider
> 协议；CLI 默认入口 = `clients/tui` 的 tui2 宿主 + tui2-shell；旧 daemon
> client-protocol 栈、`api_layer` 与旧 `tui/` 已删除）。门禁命令都能重跑；
> 最终缺口见 §7。

## 0. 一句话总览

```
客户端（TUI/CLI/Flutter/Web）
   │  access wire（Hello v7 + application protocol，不变）
   ▼
access（canonical $XDG_RUNTIME_DIR/anytty-v2-wire7.sock）
   │  auth/files/proxy/store 本地终结；terminal 路由 provider
   ▼
daemon（<canonical>.provider，owner-only unix）
   │  PTY/进程 · history · live snapshot · terminal events
   └─（以后：tmux/zellij 翻译型 provider；本期只有占位接口）
```

- **access 是唯一客户端入口**：终结 access wire、DeviceIdentity/AccessStore、
  配对、Direct/Cloud、remoteauth、文件服务、browser 转发、storage、撤销/过期踢线。
- **daemon 是纯终端 provider**：只监听内部 owner-only socket，无身份、无账本、
  无文件、无 storage、无网络出口；`client_access.*` / `cloud.*` / `file.*` /
  `storage.*` / browser proxy 在 daemon 侧 fail closed。
- 客户端 wire 一个字节未改；Flutter 无需发版。

## 1. 目标架构（已落地）

### 1.1 access（`access/`）

| 组件 | 作用 |
|---|---|
| `access/server` | 协议服务器：Hello/请求预算/stream registry/事件转发 + command family 路由 |
| `access/server.FamilyTerminal` | terminal command/attach 流/事件 → terminal provider |
| `access/server.FamilyFile` | file.* + 传输在 `access/files` 本地终结 |
| `access/server.FamilyProxy` | browser proxy 从 access 主机拨号（`access/proxy`） |
| `access/server.FamilyStorage` | `storage.*` 在 `access/storage` 本地终结 |
| `access/server.FamilyAuth` | `client_access.*` / `cloud.*` 直答 `access/runtime` |
| `access/files` | 文件 metadata + 可续传 transfer（§11 优化） |
| `access/proxy` | browser/webview TCP 转发（接收窗口/上传队列流控） |
| `access/storage` | opaque KV + 变更广播 |
| `access/accessrun` | anytty-access / `anytty access run` 的共享组合入口 |
| `access/provider/terminal` | terminal provider 契约；tmux 占位返回 unsupported |
| `access/provider/daemon` | provider 适配器：apipb 命令 → providerv1 typed 调用，附件流 bootstrap、terminal event 投影、provider 错误码映射 |
| `access/runtime` | DeviceIdentity + AccessStore + pairing + Cloud + Direct 记录 |
| `access/direct` / `access/gateway` | Direct listener / 字节透明 relay |
| `access/sessions` | grant 会话登记，撤销/过期实时关闭客户端 transport |

### 1.2 daemon（`daemon/{core,remote,cloud,cmd/anyttyd}`）

- 只监听 `<canonical>.provider`（`0600`，本地 owner 等价）；
  `TestDaemonHasNoTCPListeners` 仍成立。
- 保留：PTY/进程、history、live/snapshot、terminal events、path.defaults/
  path.list_directories、attachment registry。
- 已删除实现：file_service/file_transfer/storage/browser proxy 及其测试；
  `client_access.*`/`cloud.*`/`file.*`/`storage.*`/browser 命令 fail closed。
- 已退出的挂载：`WithClientAccessService`/`WithRemoteService`、control RPC 客户端、
  `TransportScope`、插件子系统（更早轮次）。

### 1.3 socket 拓扑（最终）

| socket | 归属 | 说明 |
|---|---|---|
| `<canonical>`（默认 `$XDG_RUNTIME_DIR/anytty-v2-wire7.sock`） | access | 客户端入口；owner-only；本地免鉴权 |
| `<canonical>.provider` | daemon | terminal provider，只有 access 连 |
| `<canonical>.pair` | access | remoteauth v2 PairingExchange |
| `<canonical>.direct` | access | Direct listener 记录 |
| `<canonical>.control` | — | 已删除（不再有 daemon→access 反向 RPC） |

## 2. 生命周期与 CLI

- `anytty daemon start`：先起 daemon（provider 协议，绑定 `.provider`），
  就绪探测走 provider Hello，再起 access（canonical）；写入同一 runtime record 的
  daemon + access PID；失败回滚，避免半栈。
- `anytty daemon stop`：先停 access，再停 daemon；status 同时报告两者。
- 日志分开：`anyttyd.log`（daemon）与 `anytty-access.log`（access）。
- `anytty access run|status|logs`：前台运行/查看 access；
  `anytty-access` 独立二进制保持 `--listen/--route/--socket/--allow/--pair-token/
  --log-file/--config/--access-socket/--provider-socket/--file-root/--transfer-dir`。
- CLI 客户端 auto-start 会成对拉起 daemon+access，并等待 provider 就绪后才暴露
  canonical 入口。

## 3. 文件传输优化（§11）状态

| 项 | 状态 |
|---|---|
| 统一路径解析（绝对/相对/`~`/`.`/`..`/Windows UNC/大小写） | 完成（`access/files/path.go`；`--file-root` 配置根） |
| 符号链接/断链逃逸防护（读写都拒绝逃出 roots） | 完成 + 单测 |
| 跨 session/进程断点续传（持久化记录、owner 绑定、mtime/size 校验） | 完成（`access/files/transfer`，0600 原子发布） |
| 自适应窗口/分片（RTT/带宽估计，只收紧不放大历史默认） | 完成（`estimator.go` + `transfer.Adapt`） |
| 流式传输（不整文件驻留内存） | 完成 |
| 进度合并（服务端 250ms 时间窗，结构化日志） | 完成（`transfer.Coalescer`） |
| 客户端可见结构化进度 | 完成（可选字段：`FileTransferAck.transferred_bytes/total_bytes/elapsed_millis`，`progress_interval_bytes` 请求启用；未启用时 payload 与旧版逐字节一致） |
| 传输压缩协商 | 完成（可选字段：open 请求 `accept_compression`，`FileTransferHandle.content_encoding`，data frame `encoding`；当前支持 zstd，未协商时 identity） |
| 吞吐基准 | `BenchmarkFileDownloadStreaming`：64MiB 85MB/s vs 等价 `io.Copy+SHA-256` 99MB/s ≈ 86%（目标 ≥80%） |

## 4. 目录结构（当前）

```
anytty/
├─ access/{accessrun,server,files,proxy,storage,provider/{terminal,daemon},
│          runtime,direct,gateway,localstate,sessions,engine,...,cmd/anytty-access}
├─ daemon/{core,remote,cloud,cmd/anyttyd}
├─ clients/{tui,cli,flutter,web,ui}
├─ api_mapping/（仍供 Proto DTO/映射；api_layer 已随 Slice C 删除）
├─ proto/{access/*,ui/*,cloud/v1}
├─ access/docs/{ARCHITECTURE.zh-CN.md,GATEWAY.zh-CN.md}
└─ docs/HANDOFF.zh-CN.md
```

## 5. 常用命令

单二进制 / 角色隔离：`anytty` 同一二进制以两个独立进程运行 daemon（`.provider`）与 access
（canonical socket）。`anytty access start|stop|restart` 只动 access；`anytty daemon restart
--keep-access` 只动 daemon；access 崩溃不影响终端。详见 `access/docs/ARCHITECTURE.zh-CN.md` §9。

```bash
go build ./... && go vet ./...
ANYTTY_ALLOW_NESTED=1 go test -count=1 -p 4 ./...
go test -count=1 -race -p 2 ./access/... ./daemon/... ./clients/cli/...

# codegen（幂等护栏：-check 不写盘，只验证无漂移）
go run ./scripts/generate_application_api.go
go run ./scripts/generate_application_api.go -check
scripts/generate-access-proto.sh apipb/file.proto wirepb/terminal.proto

# 开发：同一二进制管理两进程
go run ./cmd/anytty --socket /tmp/anytty.sock daemon start
go run ./cmd/anytty --socket /tmp/anytty.sock access run --route 0.0.0.0:41120

# 或沿用独立二进制（--socket 现在是 canonical base）
go run ./daemon/cmd/anyttyd --socket /tmp/anytty.sock        # 实际绑定 /tmp/anytty.sock.provider
go run ./access/cmd/anytty-access --socket /tmp/anytty.sock --route 0.0.0.0:41120
```

## 6. 硬约束（别踩）

- access wire 只允许字节透明或**可选字段**扩展；Hello v7/pairing/Direct/Cloud
  握手不改。
- daemon 永不暴露公网、不持有 identity/store/files/storage；access 是唯一网络
  入口与授权 owner。
- 本地 unix 是 owner-only 信任边界，不做 challenge；走网络的必须经 remoteauth。
- resource token 由 access 以同一不透明形状签发；provider token 不出 access。
- 测试与日志不得污染终端（日志进文件）。

## 7. codegen 与已知缺口

**T3 client-protocol cleanup（本轮完成）**

- 删除 `proto/access/apipb/plugin.proto` + `plugin.pb.go`、`application.proto` 的
  plugin oneof（field 130）、`application_commands.csv` 的 plugin 行，以及
  `ApiCapability` 的 `API_CAPABILITY_PLUGIN`（改为 `reserved 16`）；
  `api_mapping` 的 plugin capability 分支和 daemon 的
  `ApplicationCapabilityPlugin` 一并删除。
- `application.pb.go`/`common.pb.go` 经
  `scripts/generate-access-proto.sh` 重新生成；四个 `.gen.go` 由
  `scripts/generate_application_api.go` 重新生成并通过 `-check`；public API
  descriptor baseline 已更新。
- 其余 command/field 均仍被 clients/cli、access/engine、access/server 或
  Flutter Dart source 使用，未删除。

**T1 Slice A1（provider 协议 + daemon provider server）**

- 新增 `proto/provider/v1/provider.proto`（provider-native，无 apipb 依赖）：
  Hello/ProtocolError、terminal create/list/get/restart/kill/remove/
  set-metadata/set-tags、path defaults/list-directories、typed error；
  由 `scripts/generate-access-proto.sh provider/v1/provider.proto` 生成 Go。
- 新增 `internal/providerproto`（[channel:2][type:1] framing 复用 access wire
  编码常量与 provider 控制帧类型）与单测。
- 新增 `daemon/provider`：provider server（Hello/请求预算/typed 错误映射）+
  同步 Client；只调用 `core.Server`/`Terminal` 的终端服务，新增最小 core hook
  `Server.TerminalDefaultsSnapshot`/`Server.ListPathDirectories`。
- additive：旧 daemon client 协议与 access 路径完全未动；provider server 仅在
  测试中监听独立 socket。
- **Slice A2（本轮完成）**：provider proto 增加 typed attach/detach/input/
  resize/resize-lock（attachment token、resize ownership/epoch/policy/size-lock）；
  `daemon/provider` 增加 server 级 attachment 仲裁 registry、session 级 channel/
  token registry、raw PTY 桥接（复用 wire stream frame 类型与 closed/sync-lost
  编码）、Client 流支持（Stream/BootstrapDone/Close）与 typed 方法；
  core 新增 `Terminal.SubscribeRawPTY` 导出 hook。测试覆盖 attach→ready→input→
  PTY 输出→resize→detach 与所有权错误。
- **Slice A3（本轮完成）**：provider proto 增加 provider-native history
  （window/copy/release/search/backlog）、live-screen-next（含 baseline）、
  event subscription（subscribe/release + terminal lifecycle events）；daemon
  provider 迁入 history token ownership、live baseline 缓存、event 订阅与
  terminal event fan-out；core 新增 `Server.NextLiveScreenWithBaseline` +
  `NativeScreenBaseline` 导出 hook。Client 增加对应方法、stream/事件订阅。
  测试覆盖 history token 生命周期（stale/二次 release）、live delta、事件
  fan-out 与 release 后不再投递。
- **Slice B（本轮完成）**：access 全量切换到 provider 协议。
  `access/provider/daemon` 从旧的 v7 client engine 改写为 providerv1 适配器
  （terminal command create/list/get/restart/kill/remove/set-metadata/set-tags/
  attach/detach/input/resize/resize-lock、path defaults/list-directories、
  history/live 全族、event subscription/release；附件 token 前 2 字节 channel，
  OpenStream 解析 channel → BootstrapDone → await Ready → 转发 PTY 帧；
  provider 错误码 → apipb ApiErrorCode 映射；storage-only 订阅用合成 token，
  不向 provider 订阅）。`clients/cli` daemon 改为 `core.Server.Start(ctx)` +
  `daemon/provider.Server` 监听 `.provider`；daemon start/status 健康探针改为
  provider Hello；accessrun 的 relay 目标从 `.provider` 改指 canonical access
  （remote 客户端仍说 access wire）。新增 `access/provider/daemon` 适配器测试
  （命令分发/错误映射/事件投影/storage-only release），access e2e、clients/cli
  smokes 全绿。
- **Slice C（本轮完成）**：删除旧 daemon client-protocol 栈。`daemon/core` 删除
  `protocol_service.go`/`application_port.go`/`application_session_port.go`/
  `application_framing.go`/`raw_pty_protocol.go`/`protocol_live_screen_baseline.go`/
  `tracked_transport.go`/`client_access_service.go`/`remote_service.go`
  （约 2.9k 行）；`core.Server` 不再有 `ListenAndServe`/`ServeTransport`/
  `TransportLifecycleObserver`/`WithApplicationExecutorFactory`/
  `WithProtocolSessionLimits`，只保留 `Start(ctx)`（history retention +
  listening 事件）与 `Shutdown`。`api_layer` 整包删除，`internal/appcodegen`
  去掉对应生成目标；`testkit/daemon.go` 旧 daemon 测试 helper 删除。
- access 归属类型下沉：新增 `access/contract`（ClientAccess*/Remote* DTO 与
  interface）与 `access/sessions` 的 RemoteSessionInfo context helpers；
  `access/server` 不再 import `daemon/core`/`daemon/remote`，`api_mapping`、
  `access/runtime`、`clients/cli` 改用 access-owned 类型，行为不变。
- 覆盖迁移：旧协议测试删除约 2.8k 行；语义仍存在的部分迁到 provider：
  `daemon/provider/ownership_test.go`（owner transfer/epoch fence、promotion、
  ownerless release、observer 规则、persisted size lock、view count、attachment
  projection replay）、`history_live_test.go`（history token 隔离/回滚/release
  stale、confirmed baseline delta 与 session 隔离、跨 journal window 桥接）、
  `baseline_internal_test.go`（baseline promotion/pin/expiry/entry bound）。
  request budget/channel allocator/duplicate-in-flight/executor panic 等属于已删除
  的 client-facing protocol 机制，随栈删除；access 保留每连接 64 in-flight 上限。
  core shutdown 测试保留并发 waiter 与 deadline 覆盖，listener/transport/executor
  专项删除；CLI/Gateway/TUI 集成测试改为 access+provider 拓扑。
- **T4（本轮完成）**：CLI 默认入口切到新 TUI。`anytty`（根命令）与 `anytty
  attach <id>` 加载 endpoint registry 后启动 `tui2` 宿主 + `tui2-shell` 布局
  程序：`TUI2_BIN`/`TUI2_SHELL`（或同目录/PATH）定位二进制，必要时自动拉起
  daemon+access；显式 `--socket` 通过临时 registry override 注入；`--config`
  转发给 tui2-shell（新 JSON 配置，默认 `$XDG_CONFIG_HOME/anytty/tui2.json`，
  `ANYTTY_TUI2_CONFIG` 可覆盖）；`--log-file` 传给宿主；`attach <id>` 以
  `-attach` 绑定目标（缺失时退回 picker）。`anytty config` 迁移到 tui2 JSON
  （get/set/unset/show/validate/paths）。daemon/history 的 `--config` 仍是
  `tui-v3.yaml` 的 `daemon:` 段（换用新解析器，格式/环境变量/校验不变）。
- 删除旧 `tui/` 整树、`shell/`、`cmd/anytty-program-harness`、
  `cmd/anytty-keyprobe`；CLI tmux smokes 重写为新入口黑盒（picker、创建/输入、
  ANSI+Unicode、resize→`stty size`、多轮稳定性），artifacts 仍写入临时目录。
- 用户可见差异：`--config` 对 TUI 入口现在是 tui2 JSON（旧 `tui-v3.yaml`
  的 `tui:` 段不再被 CLI 解析；shell 会以一行 notice 回退默认值）；`attach <id>`
  在新 TUI 工作区中绑定目标，不再有旧 TUI 的 pane 级 UI；release/install 仍随包
  安装 `tui-v3.yaml`（现为 daemon 配置模板）。
- **T5（本轮完成，最终清理）**：`deadcode`（经镜像 `golang.org/x/tools/cmd/deadcode@latest`）
  对 `./cmd/anytty`（默认/dev tag）、`./daemon/cmd/anyttyd`、`./access/cmd/anytty-access`、
  `./clients/tui/cmd/tui2{,-shell}`、`./access/engine/binding/cabi/androidlib` 做可达性
  扫描；删除定义后无任何调用的迁移残留（localstate/accessrun 旧 socket helper、
  securetransport server-TLS 集群、wire history payload codec、providerproto 事件编码、
  provider 死 helper、CLI socket shim 等）。Flutter 绑定可达的 engine supervisor/
  platform broker 与测试可达的 api_mapping 映射面保留为有意 API。
- 过期引用收口：仓库内 `api_layer`/`daemonpipe`/`runtime/control`/旧 `tui/`/删除的
  dev 工具引用只在历史说明中出现；`workflow.json` 标记为历史迁移日志。
- 后续：无（T1–T5 全部完成）。

**codegen（已恢复）**

- `internal/appcodegen` + `scripts/generate_application_api.go` 从
  `proto/access/apipb/application_commands.csv` 重建四个 `.gen.go`；生成结果与
  checked-in 文件逐字节一致，`internal/appcodegen` 测试与
  `go run ./scripts/generate_application_api.go -check` 是幂等护栏。
- `scripts/generate-access-proto.sh` 负责 access proto 的 buf 生成（staging 逻辑
  路径、按目标文件保留 protoc 版本行、mirror 安装 buf/protoc-gen-go）。

**已知缺口（最终）**

1. **protocolserver 抽包不再需要**：daemon 客户端协议栈已删除，access 使用
   自己的 `access/server` session 循环；原计划作废（见 ARCHITECTURE §3）。
2. **tmux provider**：只有接口与 `ErrUnsupported` 占位，未实现翻译层。
3. **客户端压缩/结构化进度可选字段**：协议与 access 已支持；Flutter/旧客户端
   尚未全部启用（wire 兼容，启用即可）。
4. **api_mapping 的 corev2-typed 映射面保留**：验证器与 access/provider 适配器
   在用；部分 mapping helper 仅测试与未来 API 使用，未删除。
5. **deadcode 之外的动态可达面**：Flutter cgo 绑定与反射路径不在静态扫描内，
   相关 engine binding 符号按“Flutter 可达”保留。
