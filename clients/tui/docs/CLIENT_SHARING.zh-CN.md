# 客户端连接层去重：tui2 ↔ client 共享（CLIENT_SHARING，最终态）

> 状态：M1–M5 已完成。tui2 不再持有任何 终端池协议/拨号实现；所有 终端池
> 连接（local-unix / tcp / ssh / direct / cloud）都经共享 `client/` 层，
> TUI 只做"共享会话 → runtime sources/组件"的薄适配。
> 相关：`ENDPOINTS.zh-CN.md`（endpoint 模型）、`REMOTE.zh-CN.md`（远程
> runbook）、`PROGRESS.zh-CN.md`（验收账本）。
> 约束：只改 `clients/tui/`；`client/`、`cmd/anytty`、`shell/`、`tui/` 未改动。

## 0. 结论（先行）

- 唯一连接真值：`client/endpoint`（registry，`endpoints.yaml`）+
  `client/runtime`（SessionOwner generation/winner、planner race、reconnect
  监督）+ `client/adapter/protocol`（ApplicationClient / ResourceStream）+
  route adapter（`client/adapter/{local,direct,ssh}`）。
- tui2 生产代码不再有：裸帧 wire client、zstd TCP 帧实现、direct/cloud/pion
  transport。`client.go`（761 行）与 `transport_tcp.go`（235 行）已删除，其
  等价实现只保留在测试（`wire_client_test.go`、`framed_transport_test.go`），
  作为 fake 终端池 harness。
- `clients/tui/endpoint` 生产代码 3642 → 2789 行（-853）；新增
  `shared_session.go`（343）/`shared_runtime.go`（255）/`session.go`（209）/
  `tcp_bridge.go`（113）。
- `tcp` 兼容语义保留：`tcp_bridge.go` 起一个进程内 unix socket，把字节透明
  转发到 HOST:PORT；连接本身仍由共享 local-unix route adapter 拨号（0 协议
  逻辑），因此 direct/cloud 之外不再有第二套 dialer。
- G1（client 层缺裸帧/stream 桥）：在 tui2 内的最小封装即
  `sharedClient`——以 `ApplicationClient` 的 command API 承载
  list/attach/input/resize/kill/事件，用 `ResourceStream` 承载 PTY 原始流；
  不改 `client/`。`clients/tui/endpoint/sessionConn` 是唯一拨号入口。
- M2：TUI host 启动时读共享 registry（`Options.LoadSharedRegistry`，
  main.go 开启）；picker 通过 host sources 自动列出 CLI 配对端点（含
  `kind=endpoint` 占位源与 health 角标）；tui2.json 同名字段仅作迁移。
- M3：`anytty endpoint add local ...` 写入 registry → TUI 冷启动列出/attach/
  输入/重启 → 离线端点可读报错不崩，均有 acceptance 断言。
- M3b（本轮）：隔离 XDG 的 dev/v2 用只读覆盖直接读老 XDG 的
  `endpoints.yaml`：`TUI2_ENDPOINTS=<path>` 或 `tui2 -endpoints <path>`
  （flag 优先，`:` 分隔多文件）；顺序 = 显式… → 默认
  `client/endpoint.DefaultPath()`，同名先到先得（显式优先），缺文件/坏格式
  只发可读 warning 且不崩、不写回，dev 自己的端点按不同 name 并存。
  runbook 见 `REMOTE.zh-CN.md` §4.0.1。

## 1. 共享层 API（tui2 实际使用）

| API | tui2 用法 |
| --- | --- |
| `client/endpoint.Load(DefaultPath)` | `shared_registry.go`/`shared_runtime.go` 只读加载（可经 `TUI2_ENDPOINTS`/`-endpoints` 追加显式文件，按 name 合并）；CLI 写、TUI 读 |
| `client/endpoint.AccessRoute` | `endpointFromConfig` 把 tui2 Config/兼容字段投影为 route（local-unix 承载 tcp 桥） |
| `client/runtime.SessionOwner` + `ClientRuntime.AcquireSession` | 每条连接一个 owner（generation/winner 唯一真值）；`sharedClient` 的 `Done/Err/close` 代理它 |
| `client/runtime.RoutePlanEnvironment` | `sharedRouteEnvironment` 按 route kind 与 credential store 可用性过滤 |
| `client/adapter/protocol.ApplicationClient` | `sharedClient` 的全部 command（`TerminalList/Create/Attach/Input/Resize/Kill/Remove/Restart/LiveScreenNext/Detach`） |
| `client/adapter/protocol.OpenResourceStream` | `startStream` 后按 `PTYOutput/SyncLost/Closed/StreamReady` 帧喂给既有 `RemotePTY.pump` |
| `client/adapter/local.Dialer` | local-unix（含 tcp 桥、ssh -L unix 转发） |
| `client/adapter/direct.Dialer` + `ssh.Dialer` + `cloud.Dialer` | direct-webrtc-tcp / ssh-webrtc-tcp / managed-webrtc（含 capability auth / Cloud enrollment），tui2 只注入 composition（controller 地址支持 `ANYTTY_CLOUD_CONTROLLER_*` 覆盖） | 
| `client/port` / `shared/remoteauth` | credential store 路径与 CLI 相同（`StateHome/anytty/remote-v2/credentials`），只读 |

## 2. tui2 适配层（最终文件）

| 文件 | 职责 |
| --- | --- |
| `session.go` | `sessionConn` 接口（唯一拨号 seam）、`attachment`/`stream` 解复用、`APIError` |
| `shared_session.go` | `sharedClient`：ApplicationClient/ResourceStream → sessionConn；连接丢失绝不伪装成终端退出（`streamEndError`）；close 释放 owner/bridge |
| `shared_runtime.go` | 每连接组装 SessionOwner+ClientRuntime；registry 优先的 plan snapshot；Config→Endpoint 迁移投影；credential/route environment |
| `tcp_bridge.go` | tcp 兼容入口的字节透明 unix↔TCP relay（无协议逻辑） |
| `shared_registry.go` | 只读共享 registry 投影（`RegistryPaths`/`LoadSharedEndpointConfigs`/`SharedConfigForEndpointIn`），多文件按 name 合并、显式优先、warning 可读 |
| `manager.go` | 每 endpoint supervisor（250ms..5s 退避）、终端清单、`sources`（含离线端点占位源）；`Options.Dial` 默认 `dialSharedSession` |
| `remotepty.go` | 快照权威 + rebind 语义不变，只依赖 `sessionConn` |
| `config.go` | 四种共享 mode + `ssh-webrtc-tcp` 识别；缺 route 参数在共享 planner/validator 给出可读错误 |

断线语义：共享 session `Done` → health=offline → 退避重连 → `rebindAll`
（重新 attach + `LiveScreenNext(0)` 快照重建）→ **再**发布 health=ok，保证
"connected" 之后第一键一定落在可用 attachment 上；`ErrStreamSyncLost` 仍触发
即时 rebind。

## 3. tui2 保留 / 删除清单（最终）

| 文件 | 处置 | 理由 |
| --- | --- | --- |
| `client.go`（761） | 删除（生产） | 裸帧 protocol client；测试侧别名保留在 `wire_client_test.go` |
| `transport_tcp.go`（235） | 删除（生产） | tui2 zstd 帧；tcp 改为 byte relay + 共享 local adapter；测试侧 `framed_transport_test.go` |
| `transport_direct.go`（620）/`transport_cloud.go`（244）/`transport_direct_pion.go`（67） | 已删除 | 复刻共享 direct/cloud |
| `manager.go`/`remotepty.go` | 保留 | supervisor 语义与组件管线是宿主职责；改为 `sessionConn` |
| `tcp_bridge.go` | 新增 | tcp 兼容入口，无协议逻辑 |
| 测试 | 保留+扩展 | 假 终端池（含 DeviceIdentity proof）覆盖 raw 与共享两条栈；`shared_stack_test.go` 覆盖共享重连 |

## 4. M2 registry 驱动

- `clients/tui/cmd/tui2`：`Options.LoadSharedRegistry`（main.go 置 true）→
  `LoadSharedEndpointConfigs()` 注册全部可表示端点；registry 损坏只发 notice。
- picker：终端池 终端来自 host sources（`<terminal> · <endpoint> · live`）；
  无终端的 终端池 endpoint 由 `Manager.Sources()` 发 `kind=endpoint` 占位源
  （label + health），选择占位源即 `terminal.create{endpoint}`，离线时状态行
  显示可读错误。
- `tui2.json` 的 `endpoints[]` 仍解析：同名时共享 registry 优先
  （`registerEndpoint` 先查 `SharedConfigForEndpointIn`），仅作迁移。
- 显式 registry 覆盖（M3b）：`TUI2_ENDPOINTS`（`os.PathListSeparator` 分隔）
  或宿主 `-endpoints`（flag 优先）；路径序即优先级，默认路径始终最后并入。
  同名 shadow、缺失文件、损坏文件都会进 notice；`SharedConfigForEndpoint`/
  `sharedPlanSnapshot` 在低优先级 registry 定义同名端点时继续可用，只有
  "任何 registry 都找不到且显式文件损坏" 才返回可读错误。
- `scripts/acceptance.sh` 与 `scripts/smoke.sh` 全量隔离 XDG（`$WORK/xdg/*`）
  并动态选择 signaling 端口，保证 registry/终端池 不外泄。

## 5. 缺口（本轮未清零，均已给可读错误）

- **Cloud composition 复制**：managed-webrtc connector 已装配，但 controller
  地址/环境变量解析与 `cmd/anytty` 各一份（默认值与语义相同）；后续应下沉
  `client/runtime`。Cloud enrollment/identity 管理仍在 CLI（tui2 只读
  credential store）。
- **`tcp` 无 registry kind**：CLI registry 仍无裸 HOST:PORT route kind；
  tui2 的 `tcp` 只作 tui2.json 兼容入口（经 tcp_bridge 走共享 local route）。
- **G3 composition 复制**：credential store 路径与 Cloud controller 环境变量
  在 tui2 与 `cmd/anytty` 各有一份（只读、默认值相同）；后续应下沉
  `client/runtime` composition。

## 6. 验收与守卫

- Go：`shared_stack_test.go`（真实共享栈 + framed fake 终端池：list/create/
  attach/input/resize/断线≠退出/快照重播/kill）、`client_sharing_test.go`
  （registry 优先、tcp 桥、`go list` 直接 import 守卫、
  `go list -deps` 共享适配器闭包、行数证据）。
- acceptance（283 → 298 → 313 项）：CLI `endpoint add` → registry 文件 → TUI
  picker 列出 → attach → `REG-OK` → `exit` 角标 → Ctrl-E 重启 → 离线端点
  角标/notice → 干净退出；M3b 新增 15 项：老 XDG registry 经
  `TUI2_ENDPOINTS` 直接列出/attach/输入、同名显式优先、dev registry 并存、
  md5 证明只读、missing/corrupt 只发出可读 warning 且宿主干净退出；tcp/ssh -L
  隧道回归全绿；守卫断言 tui2 直接 import `client/adapter/protocol` 且不再直接
  import `shared/transport/unix`。
