# AnyTTY 目标架构：access 协议服务器 + 能力路由

> 状态：Phase 0–4 完成（2026-09-19）。每个阶段都保持 `go build ./...`、`go vet ./...`、
> `go test -count=1 -p 4 ./...` 全绿。
> 当前状态与缺口见 `docs/HANDOFF.zh-CN.md`；细节见 `access/docs/GATEWAY.zh-CN.md`。

## 实施进度（2026-09-19）

> 状态（T5 后）：Phase 0–4 + T1（provider 协议）/ T2 / T3（插件协议删除）/
> T4（CLI 默认入口切到 `clients/tui`：tui2 宿主 + tui2-shell，旧 `tui/` 删除）/
> T5（dead-code 清理与过期引用收口）完成。daemon 只在 `<canonical>.provider`
> 上服务 provider 协议；access 终结客户端 wire 与 file/storage/proxy/auth；
> 旧 daemon client-protocol 栈与 `api_layer` 已删除（Slice C）。§3 的
> `protocolserver` 抽包**不再需要**（daemon 客户端协议已整体删除）。

| 阶段 | 状态 | 说明 |
|---|---|---|
| Phase 0 骨架 | 完成 | `access/server`（session 循环 + 路由）、`access/provider/terminal` 契约、daemon provider（client engine）、tmux 占位；`protocolserver` 抽包按 §3 退路改为 access 新建 session 循环（daemon/core 协议测试直接读写未导出字段，原样迁移会破坏回归护栏）。 |
| Phase 1 终端路由 | 完成 | client→access→provider→daemon PTY 全链路（create/list/get/kill、attach 流桥接、input/resize、history/live、events）；`access/sessions` tracker 在 `access/server.ServeTransport` 挂钩，revoke 关闭客户端 transport 后 provider/附件/桥接随之释放。 |
| Phase 2 文件/转发 | **完成（access 侧）** | `access/storage`、`access/files`（metadata 全命令 + 可续传 transfer）、`access/proxy`（browser proxy）已迁移并在 `access/server` 本地终结；§11 路径安全/断点续传/自适应窗口/进度合并全部落地（含 e2e 与基准）。daemon/core 旧实现暂时保留，等 Phase 3 socket flip 后与直连客户端一起删除。 |
| Phase 3 入口切换 | **完成** | canonical socket 由 access 持有；daemon 移 `<canonical>.provider`；`anytty daemon start/stop/status` 双进程管理（独立日志）；daemon file/proxy/storage 实现与测试已删除（生成器缺失导致 port 方法保留 fail-closed stub）。 |
| Phase 4 收尾 | **完成** | `access/runtime/control` 与 `provider/daemonpipe` 已删除；`client_access.*`/`cloud.*` 由 `access/server` 的 `FamilyAuth` 直答 `access/runtime`；daemon 不再挂载 ClientAccessService/RemoteService；tmux 仍为接口占位。压缩/结构化进度事件因缺 codegen 显式跳过。 |

### Phase 2 交付细节（2026-09-19）

- **命令迁移**：`file.list/stat/preview/mkdir/rename/delete/copy/move`、
  `file.download_open/upload_open/transfer_cancel`、`browser.proxy.open` 全部在
  `access/server` 路由（`FamilyFile`/`FamilyProxy`），不再经过 daemon；
  `storage.*` 同样本地终结。Resource token 由 access 以同一不透明形状签发
  （file `[channel:2]+"ft"+hexID`，browser `[channel:2]+random`），provider token
  不暴露给客户端。
- **路径安全**：`access/files/path.go` 的 `Resolver.Resolve`（内容路径，完整解析
  符号链接）与 `ResolveParent`（目录项路径，保留最后组件）覆盖 list/stat/preview/
  mkdir/rename/delete/copy/move/upload/download；断链写目标也会被 root 约束拒绝。
  通过 `anytty-access --file-root` 配置允许根（缺省仍为 legacy 的“任意绝对路径”）。
- **断点续传**：`access/files/transfer.Store` 持久化在
  `$XDG_STATE_HOME/anytty/transfers/transfers.json`（0600、原子发布），记录绑定
  目标路径/大小/owner；resume token 是不透明 bearer secret；跨 session 与跨进程
  重启都能续传（临时文件领先于持久化 offset 时回退截断）。
- **自适应与进度**：`access/files/estimator.go` 从下载 ack 估计 RTT/吞吐，
  经 `transfer.Adapt` 选择窗口/分片，并且**只收紧不放大**历史默认值
  （1MiB 窗口 / 64KiB 分片），避免改变已上架客户端假设；窗口/分片通过既有
  `FileTransferHandle` 字段返回。`transfer.Coalescer` 负责服务端 250ms
  时间窗合并。**可选扩展已落地**：open 请求 `accept_compression` +
  `progress_interval_bytes`，handle 回 `content_encoding`/`progress_interval_bytes`，
  data frame 携带 per-frame `encoding`，ack/finish 携带结构化进度
  （`transferred_bytes/total_bytes/elapsed_millis`）；字段未设置时 proto3 零值
  不编码，旧客户端 payload 逐字节不变。当前压缩只支持 zstd。
- **验收证据**：`access/server` e2e（`TestFileCommandSurfaceThroughAccess`、
  `TestFileTransferResumeUploadAndDownload`、`TestBrowserProxyThroughAccess`）、
  `access/files` 单元/e2e、`access/proxy` 流控单测、基准
  `BenchmarkFileDownloadStreaming`（64MiB：85MB/s vs 等价
  `io.Copy+SHA-256` 基线 99MB/s ≈ 86%，满足 §11 ≥80% 目标）。

## 0. 一句话

客户端只连 **access**；access 终结 access wire，按能力路由：鉴权/配对/Cloud、
文件管理、端口转发由 access 自己处理；终端请求转给 **terminal provider**
（现在 = anyttyd，以后 = tmux/zellij 等）。daemon 降级为纯终端 provider，
只监听内部本地 socket。

## 1. 角色

| 组件 | 职责 |
|---|---|
| access（`anytty-access`） | 唯一客户端入口；协议服务器（Hello/请求/流/事件）；DeviceIdentity + AccessStore + 配对；文件服务；端口转发；终端路由；撤销/过期踢线 |
| daemon（`anyttyd`） | 终端 provider：PTY/进程/history/live/snapshot；只监听内部 owner-only socket；无身份、无账本、无文件、无网络出口 |
| 客户端（TUI/CLI/Flutter/Web） | access wire 不变；本地与远程都连 access |

## 2. 拓扑

```
客户端（TUI/CLI/Flutter/Web）
   │  access wire（Hello v7 + application protocol，不变）
   ▼
┌─ access（协议服务器 + 能力路由）─────────────────────────────────────┐
│ auth  : Hello / identity / client_access.* / cloud.* / 配对          │
│ files : file.* + 上传下载流（操作 access 主机文件系统）               │
│ proxy : webview 端口转发（从 access 主机拨号）                        │
│ store : storage.*（通用 KV，迁移中）                                  │
│ route : terminal.* / attach 流 / terminal 事件 ──▶ terminal provider  │
└───────────────┬──────────────────────────────────────────────────────┘
                │ provider 协议（owner-only 本地 socket）
                ▼
        ┌─ daemon provider（现在，= 现有 daemon 收窄为终端面）─┐
        │ PTY/进程 · history · live snapshot · terminal events  │
        └───────────────────────────────────────────────────────┘
        （以后：tmux provider —— 翻译成 tmux control mode）
```

### 2.1 socket 拓扑（本方案要拍板的迁移）

| socket | 归属 | 说明 |
|---|---|---|
| `$XDG_RUNTIME_DIR/anytty-v2-wire7.sock` | **access** | 客户端入口；owner-only；本地连接免鉴权（信任边界=0600） |
| `<client sock>.provider` | daemon | 终端 provider 协议；只有 access 会连 |
| `<client sock>.pair` | access | 本地 PairingExchange |
| `<client sock>.direct` | access | Direct listener 记录 |
| `<client sock>.control` | — | Phase 4 已删除（access 直答 client_access/cloud） |

生命周期：`anytty daemon start/stop` 同时管理 daemon 与 access（daemon 先起，
access 绑定客户端 socket）。日志分开：`anyttyd.log`、`anytty-access.log`。

## 3. 协议服务器

- access 使用 `access/server` 自己的 session 循环（frame/Hello/请求预算/stream
  registry/事件转发）；复用 `internal/protocol` 的 frame 与消息编码。
- daemon 侧不再有客户端协议服务器：provider server（`daemon/provider`）只服务
  access 的 owner-only `.provider` 连接。
- 原计划把 `daemon/core/protocol_service.go` 抽成 `protocolserver` 包供 access
  复用；T1–T4 完成后该路径已**作废**（daemon 协议栈整体删除，access session
  循环已独立稳定），不再实施。

## 4. 能力路由（access host）

按 command oneof / stream kind 路由，不做"整段转发"：

| 请求 | 处理 |
|---|---|
| Hello / identity / client_access.* / cloud.* | `access/runtime` 直答（原 `access/runtime/control` 已删除） |
| file.*（list/stat/preview/mkdir/rename/delete/move/copy） | `access/files` |
| 上传/下载流（`file_transfer` channel） | `access/files` 终结（窗口/续传在 access） |
| browser proxy open + 流 | `access/proxy`（从 access 主机拨号） |
| storage.* | `access/storage`（迁移 `daemon/core/storage.go`） |
| terminal create/list/get/kill | provider 调用 + 结果映射 |
| terminal attach + PTY 流 | provider 附件流原样桥接到客户端 channel |
| terminal input/resize/history/live | provider 调用 |
| events | 合并 terminal 事件（provider）与 storage 事件（access 本地） |
| path.defaults | provider（shell/cwd 属于终端主机） |

错误映射：access/provider 适配器输出 apipb typed envelope；provider 错误码
（400/403/404/409/412/429/503/500）映射为既有 `ApiErrorCode`，不新增客户端可见
错误码。

## 5. Terminal provider 接口

```go
// access/provider/terminal
type Provider interface {
    Create(ctx context.Context, spec TerminalSpec) (TerminalRef, error)
    List(ctx context.Context) ([]TerminalInfo, error)
    Get(ctx context.Context, id string) (TerminalInfo, error)
    Kill(ctx context.Context, id string) error
    Attach(ctx context.Context, request AttachRequest) (Attachment, error) // 含双向 frame 流
    Detach(ctx context.Context, attachment AttachmentRef) error
    Input(ctx context.Context, attachment AttachmentRef, data []byte) error
    Resize(ctx context.Context, attachment AttachmentRef, size Size) error
    History(ctx context.Context, id string, window HistoryWindow) (HistoryResult, error)
    LiveScreen(ctx context.Context, id string, revision uint64) (Snapshot, error)
    WatchEvents(ctx context.Context, filter EventFilter) (<-chan TerminalEvent, error)
    PathDefaults(ctx context.Context) (Defaults, error)
}
```

- **daemon provider（本期）**：access 用现成客户端引擎
  （`access/engine/adapter/protocol` + `clientruntime.ApplicationSession`）连
  `<sock>.provider`，把 provider 方法逐条映射；attach 流用 `OpenResourceStream` 桥接。
- **tmux provider（后置）**：实现同一接口（translate 到 tmux control mode/pane 流）；
  本期只落接口与错误占位，不实现。
- 连接策略：默认每个客户端会话复用一个 provider 会话；断线按现有 client engine
  重连策略；provider 不可用时返回 typed unavailable，不影响 auth/文件/转发。

## 6. 服务迁移

### 6.1 `access/files`
从 `daemon/core` 搬迁（同机语义不变，操作 access 主机文件系统）：
- `file_service.go`、`file_domain.go`、`file_roots_other.go`、`file_roots_windows.go`、
  `path_list.go`、`file_transfer.go`（含窗口/续传/temp/resume token）及对应测试；
- daemon 删除这些文件与命令实现；映射层随后续清理演进（`api_layer` 已于 Slice C
  删除；`api_mapping` 保留验证与 access 侧映射面）。

### 6.2 `access/proxy`
- `browser_proxy.go`、`browser_receive_window.go`、`browser_upload_queue.go` 及测试；
- 拨号出口在 access 主机；手机 webview 的 loopback 代理客户端
  （`access/engine/browserproxy`）不动。

### 6.3 `access/storage`
- `daemon/core/storage.go`（通用 KV + 事件）迁移；客户端命令 `storage.*` 不变。

### 6.4 删除/退役
- `access/runtime/control`（daemon→access 反向 RPC）在 access 直答后删除；
- daemon 的 `ClientAccessService/RemoteService` 挂载点删除；
- `access/provider/daemonpipe`（plan A 的认证后直通管道）退役，改由 provider 客户端替代；
- daemon 的 Direct/Cloud 相关残留（此前已删）确认清零。

## 7. 客户端与生命周期

- `resolveV3Socket` 语义不变：默认客户端 socket 由 access 提供；
- 本地连接仍免鉴权（owner-only 0600）；远程走 remoteauth；
- `anytty daemon start|stop` 同时起停两个进程；`daemon status` 汇总两者；
- TUI/CLI/Flutter access wire 一个字节不改；Flutter 无需发版；
- `--route`/`--listen`/`--allow`/`--pair-token` 全在 access 侧不变。

## 8. 历史实施阶段（归档；Phase 0–4 + T1–T5 均已完成）

> 以下是最初的分阶段计划，作为决策背景保留。实际执行中的偏差：
> `protocolserver` 抽包未做（daemon 客户端协议栈已整体删除，见 §3）；
> `api_layer` 随 Slice C 删除；旧 `tui/` 随 T4 删除。

### Phase 0：骨架与协议服务器
- 抽出 `protocolserver` + `SessionHost`，daemon 切过去（行为不变）；
- 新建 `access/server`（路由 host 骨架）、`access/provider/terminal` 接口；
- 新增 access 本地监听 `<client sock>.access`（临时开发口，Phase 3 换成 canonical）。

### Phase 1：终端路由打通
- access/server 接通 provider（daemon 客户端引擎），terminal 全命令 + attach 流 + 事件；
- e2e：client→access(.access)→provider→PTY 输入输出；kick/过期关会话；
- 旧路径（直连 daemon）保持可用。

### Phase 2：文件与转发搬入 access
- `access/files`、`access/proxy` 迁移；路由本地终结；
- 删除 daemon 对应实现与测试；文件/转发 e2e（本地 + Direct/Cloud 远程）。

### Phase 3：入口切换
- access 绑定 canonical 客户端 socket；daemon 移到 `.provider`；
- `anytty daemon start/stop` 双进程生命周期；日志/诊断更新；
- 全量 TUI/CLI/tmux smoke、Flutter 手测（wire 不变）。

### Phase 4：收尾
- 删除 control RPC 反向路径、daemonpipe、daemon 非终端代码；
- storage 迁移（若确认）；provider 错误面收口；
- tmux provider 接口占位与文档；
- 更新 `docs/HANDOFF.zh-CN.md`、`access/docs/GATEWAY.zh-CN.md`、`access/docs/ARCHITECTURE.zh-CN.md`。

## 9. 测试策略

- `protocolserver` 抽取后 daemon/core 原有协议测试必须原样通过（回归护栏）；
- access 路由单测：命令→handler 分派表、事件合并、流终结；
- provider 契约测试：同一套断言同时跑 daemon provider（必须绿）与 tmux provider（占位 skip）；
- e2e：本地 TUI/tmux smoke 经 access；Direct/Cloud 远程全链路；文件上传下载；
  webview 转发；revoke/TTL 踢线；
- Flutter 只做 wire 兼容性手测（不发版）。

## 10. 风险与决策

**风险**
1. protocolserver 抽取是最大工程，先抽后改，daemon 测试作回归护栏；
2. attach 流的 owner/resize epoch/raw PTY 语义必须由 provider 桥接完整保留；
3. 本地入口切到 access 后，access 不在时本地终端不可用（已接受）；
4. 多一跳的性能：access 只做 frame 转发与本地服务，不额外拷贝。

**已确认（2026-09-19）**
1. socket 拓扑：access 拿 canonical 客户端 socket，daemon 移 `.provider`；
2. `storage.*` 随文件/转发一起搬进 access；
3. `anytty daemon start` 同时拉起 daemon+access；
4. tmux provider 本期只留接口，不实现。

## 11. 文件传输优化（性能 + 体验，Phase 2 内完成）

目标：文件服务搬进 access 的同时把传输质量做上去。客户端 wire 只允许
**可选字段/新命令** 扩展，已上架 Flutter 端不能破坏。

1. **断点续传**
   - 跨会话/跨进程重启可续（access 侧持久化 transfer 记录，绑定目标路径+大小+owner）；
   - 续传 token 保持不透明 bearer secret；续传时校验目标文件大小/mtime 变化；
   - 取消后 token 失效；过期清理有界。
2. **性能**
   - 窗口/分片按通道类型自适应（局域网大窗口，高延迟链路小分片）；
   - 大文件边读边发，避免整文件驻留内存；下载/上传互不阻塞（session 级并发上限）；
   - 可选传输压缩：仅对声明可压缩的内容启用（新可选字段），默认关闭；
   - 目标：同机/局域网吞吐不低于裸 `io.Copy` 的 80%，内存占用与文件大小解耦。
3. **进度体验**
   - 结构化进度事件（已传字节/总量/速率/剩余时间），按时间窗合并推送（避免高频小帧）；
   - 断点/暂停/取消状态可恢复；客户端 CLI/Flutter 显示一致；
   - 进度事件必须可选，旧客户端忽略不报错。
4. **文件路径处理**
   - 统一规范化：绝对路径、`~`、相对路径、`.`/`..`、Windows 盘符与 UNC；
   - 符号链接安全：默认不跟随逃逸出允许根目录（`file_roots_*`），写入拒绝逃逸；
   - Unicode/UTF-8 与大小写敏感平台差异；长路径在 Windows 上的前缀处理；
   - 列表/预览/上传/下载/改名/删除使用同一套路径解析与错误分类。

验收：断点续传 e2e（中断→重启→续传成功且内容一致）、进度事件合并单测、
路径解析与逃逸防护单测、吞吐基准写入 `tui/render` 之外的 access 基准测试。
