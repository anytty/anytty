# 移动端网络与 WebView 恢复架构

状态：已接受，P0-P2 已实施。

实施记录（2026-08-29）：P0 的输入不重放与恢复状态传播、P1 的 renderer lease 和
attachment 资源隔离、P2 的窄 Go Endpoint Supervisor 均已落地。Android 与 iOS 的进程级
网络/生命周期信号由 native runtime owner 持有，全部已配置 endpoint 固定由 Go takeover；
WebView/TS 只维护 Demand 和 renderer binding lease，不再执行物理 session probe、invalidate
或拨号重试。真机 30 分钟 Doze/Wi-Fi 蜂窝切换仍必须作为发版验收执行，不能用单元测试替代。

本文定义 AnyTTY 移动端在 Android 前后台切换、WebView 冻结或重建、网络路径变化、
Go session 存活或失效时的所有权、恢复语义和用户体验。实现不得通过把底层状态逐层
穿透到 UI 来协调恢复。

## 1. 问题

移动端同时存在四种相互独立的存活状态：

- Android 应用进程是否存活。
- WebView renderer 是否运行、冻结或已被系统终止。
- JS 到本地 Go binding 的连接是否可用。
- Go 到远端 endpoint 的物理 session 是否仍可端到端通信。

Android 网络回调只能说明手机当前网络路径的状态，不能证明目标 endpoint 可达。
WebView 恢复也不能证明旧物理 session 仍然可用。反过来，WebView 冻结或 renderer
死亡时，Go session 可能仍然完全健康。

当前最容易产生首次恢复失败的路径包括：

- 旧 renderer 的 operation、stream、handle 或迟到事件进入新 renderer。
- 旧 renderer 的延迟 `active=false` 覆盖新 renderer 的 `active=true`。
- 网络变化与前台恢复并发，旧操作在超时后继续提交结果。
- 全局 runtime ready 与 endpoint ready 混为一谈：要么吞掉单 endpoint 失败，要么被一个离线
  endpoint 卡住整个 APP。

## 2. 目标与非目标

### 目标

- 临时网络和生命周期变化自动收敛，不要求用户点击重试。
- WebView 冻结时，连接维护不依赖 JS timer 或 JS 网络回调。
- renderer 重建不会污染新 renderer，也不会泄漏 binding 资源。
- 恢复期间终端输入不排队、不重放，文件写操作不被隐式重复执行。
- UI 只消费面向用户的结果，同一 surface 最多展示一个恢复状态。
- 返回、关闭、设置和查看已有内容始终可用。

### 非目标

- 不承诺 Android Doze、系统强杀进程或无网络时 TCP 永久在线。
- 不把整个 `NativeSessionManager` 逐行迁移到 Kotlin 或 Go。
- 不把 Android 网络状态当作 endpoint 连通性真值。
- 当前协议没有远端 session resume token，不宣称可以恢复已死亡的远端 session。
- 当前协议没有跨 session event cursor，不依赖 binding sequence 补齐事件。
- 在幂等协议完成前，不承诺终端 PTY 或任意文件 mutation 的 exactly-once 执行。

## 3. 核心不变量

1. 每个 endpoint 同一时刻最多有一个被发布的物理 session winner。
2. Android/iOS Host 不拥有 route、session generation 或重试策略。
3. renderer 只能操作属于自己 attachment 的 binding 资源。
4. renderer 失效时回收其全部资源，但不因 UI 消失而关闭仍有 Demand 的物理 session。
5. 旧 attachment、旧 control revision 和旧 attempt 的迟到结果不得改变新状态。
6. 网络恢复先探测旧 session；只有确认失效后才能建立并提交新 session。
7. 新 renderer 通过全量 snapshot 恢复，不依赖错过事件的 replay。
8. 已 dispatch 但未确认的终端输入状态是 `UNKNOWN`，绝不自动重放。
9. 自动恢复成功时静默恢复；暂时错误不展示“重试”按钮。

## 4. 分层所有权

| 层 | 拥有 | 不拥有 | 对外输出 |
| --- | --- | --- | --- |
| Native Host | process/runtime ID、Activity/Scene/WebView 生命周期、网络和电源 snapshot、FGS、用户 stop | route、session winner、业务重试 | 带 revision 的 host projection 和生命周期 hint |
| Go Endpoint Supervisor | endpoint Demand、物理 session、route winner、generation/stamp、探测、拨号、退避 | 页面和 UI 状态 | endpoint projection |
| Binding Renderer Lease | 当前 renderer 的 operation、session lease、stream、resource handle、event queue | endpoint 物理 session | attachment snapshot 和事件 |
| WebView/TS | 当前页面 Demand、页面订阅、AbortSignal、surface rehydrate | Android 生命周期和物理 session | 带 revision 的完整 Demand snapshot |
| UI | surface、影响范围、是否可恢复、可执行动作 | 网络层级和重连算法 | 单一用户可见状态 |

Go 继续复用现有 `SessionOwner` 作为物理 session 真值。新增的 Endpoint Supervisor 必须是
窄状态机，不能成为横跨 Kotlin、Go、TS 和 UI 的万能 RecoverySupervisor。

## 5. 标识与 fencing

- `runtimeId`：Go engine 每次进程级启动生成，进程死亡后必变。
- `attachmentId`：每个 WebView renderer/binding 代际唯一。
- `demandRevision`：同一 attachment 提交完整 Demand snapshot 的单调版本。
- `stopEpoch`：最近一次 native 用户 Stop 对应的持久单调 revision；renderer 漏事件或 resume 响应超时后仍可识别。
- `controlRevision`：endpoint desired state 每次变化时递增。
- `attemptId`：单次 probe/dial 唯一，只用于隔离迟到结果。
- `sessionStamp`：现有 Go endpoint session stamp；不在 Kotlin 创建第二套 session epoch。

所有异步提交至少校验与自己有关的 `{runtimeId, attachmentId, controlRevision, attemptId}`。
旧 attachment 的任何命令直接拒绝。Demand 使用完整替换：

```text
AttachRenderer() -> { runtimeId, attachmentId, demandRevision, projection }

ReplaceDemandSet(
  attachmentId,
  baseDemandRevision,
  endpointIds
) -> { demandRevision, projection }
```

每次 `ReplaceDemandSet` 都必须携带 `getSessionDemandLease` 由 native 发放的
`{attachmentId, baseDemandRevision, stopEpoch}`，native plugin 不得用当前 revision 替调用方补票。用户主动停止时，
native 必须先推进 `demandRevision`、清空 Demand 并锁存 `userStopped`，再通知 WebView。revision 拒绝
已经越过 bridge 的旧 payload；持久 stop gate 继续拒绝旧协调器用旧 lease 提交非空 Demand。

新的 workspace 或手动重连使用原子恢复接口：

```text
ResumeSessionDemand(intentId, baseStopEpoch)
  -> { attachmentId, demandRevision, stopEpoch, outcome: resumed | stopped }
```

同一个逻辑用户意图在超时和重试时必须复用同一个 `{intentId, baseStopEpoch}`。native 在当前 attachment
下维护有界的 intent 记录：首次调用只有在 `baseStopEpoch == currentStopEpoch` 时才原子解除 stop gate；若
CAS 不匹配，也必须先记录该 intent 并永久返回 `stopped`。已接受 intent 在同一 `stopEpoch` 的重复调用幂等
返回 `resumed`；若其间发生了新的 Stop，旧 `intentId` 只返回 `stopped`，绝不能再次开 gate。
attachment 轮换时清空该记录；非法 ID 或容量耗尽一律 fail closed。TS 还要拒绝低于当前
`stopEpoch/demandRevision` 的迟到结果，且每个意图在任何 await 前绑定本地 stop generation 和已缓存的
`stopEpoch`。收到 Stop 通知时 TS 立即刷新 native lease；该 generation 之后的新用户意图等待 refresh barrier
后绑定新 anchor，Stop 前已由 manager 调用 `createResumeIntent()` 绑定的 intent 不得借 barrier 升级。恢复只解
gate，不重放 Stop 前的 owner 图；只由触发恢复的 manager 重新提交自己的 `active=true`。TS 按 manager
owner 聚合 endpoint Demand，旧 manager 的迟到 `active=false` 只能释放自己的 owner，不能清掉新 manager
的需求。冷启动尚未观测到 `stopEpoch` 时也必须先等待首次 lease refresh，不能把未知基线默认为 `0`；用户在
首次 reconcile 仍在途时点击连接，仍应由同一个 intent 完成恢复。

## 6. 状态机

Renderer：

```text
DETACHED -> ATTACHING -> SNAPSHOTTING -> ACTIVE
     ^                                  |
     +---------- bridge failure --------+
```

Endpoint：

```text
NO_DEMAND
  -> VERIFYING       已有旧 winner
  -> CONNECTING      没有旧 winner

VERIFYING --成功----------------> REHYDRATING
VERIFYING --确认 transport 失效-> FENCED -> CONNECTING
CONNECTING --验证并 CAS 提交成功-> REHYDRATING
CONNECTING --瞬时失败------------> BACKOFF
CONNECTING --授权/配置永久失败---> BLOCKED
REHYDRATING --活动 surface 就绪--> READY
```

Native 网络变化只唤醒或延后状态机。READY 必须由端到端业务探测和活动 surface rehydrate
共同确认，不能由 `NetworkCapabilities`、WebSocket open 或 binding health 单独确认。

### Cloud 路径确认与回收

Cloud 的 ICE、远端认证和协议 Hello 成功后，路径仍然只是 provisional candidate，不能立刻发布或
向 Edge 确认。客户端必须在同一条 application session 上连续完成两次带当前 `sessionStamp` 的
`TerminalDefaults` 只读请求；两次都返回有效结果后，才发送 `PathDecision(CONFIRM)`，并等待精确匹配
transaction ID 的 ACK。Confirm 前必须重新读取最新 selected candidate pair，不能复用 DataChannel 刚 ready
时的旧路径快照。单次业务请求成功不能证明路径可复用，尤其不能覆盖“首个请求成功、后续请求冻结”的
Android WebView/VPN 组合。

`auto` 路由下，direct candidate 在 Confirm 前发生可恢复的 transport、Hello 或 application probe 失败时，
客户端先发送 `PathDecision(ABANDON)` 并等待精确 ACK，随后才允许在同一 endpoint generation 内尝试一次
Relay/TCP。显式 Direct/Relay 模式、认证拒绝、身份或权限错误、管理端关闭、调用方取消均不进入该自动
fallback。Confirm 只冻结路径决策，不得释放备用 Relay：非 trickle ICE 中迟到的 TURN candidate 可能在对端
表现为 `prflx`，客户端单边分类无法证明 TURN allocation 未被使用。已 Confirm 的 session 关闭时使用独立的
`SessionRelease` transaction，并等待 Edge 完成 Relay 和 Runtime 清理后的精确 ACK；EOF、普通 Done 或管理
关闭事件都不能冒充 ACK。Relay 并发占满属于可重试错误，由 Endpoint Supervisor 统一退避，不能要求用户
点击重连。

## 7. 生命周期

### 前台恢复

1. Native Host 重新采样网络和电源状态，并保证进程级 Go runtime 可用。
2. WebView attach 或确认当前 attachment 仍有效。
3. 对完整 Demand 做 reconciliation。
4. Go 优先探测当前 winner；失败后精确失效旧 stamp 并建立新 generation。
5. renderer 重新订阅资源并读取全量 snapshot。
6. 活动 surface 可用后发布 READY。

APP runtime ready 只表示 native runtime 可响应、旧 renderer lease 已被 fencing、完整 Demand
snapshot 已开始对账；它不表示所有 endpoint 已连接。每个 endpoint 独立保留
`waiting_network/reconnecting/failed/connected` 状态，业务操作必须等待自己的 endpoint READY，
失败也必须带 endpoint ID 记录。一个离线 endpoint 不得阻塞设备列表、设置或其他 endpoint。

### WebView 冻结

Host 和 Go 继续维护 Demand 和 endpoint。JS 恢复后先读取最新 projection/snapshot，不依赖冻结
期间错过的事件。若系统同时暂停整个进程，则恢复后按前台流程自动收敛。

### Renderer 死亡

立即撤销旧 renderer lease，取消 operation，关闭 stream，释放 resource handle，丢弃其 event
queue。若 Activity 已经 stopped，WebView 重建延迟到下一次 `onStart`。仍有 Demand 的 endpoint
session 不受影响。

### 进程死亡

视为冷启动。生成新 `runtimeId`，所有旧内存 handle 和 session 都失效。只恢复明确持久化的
用户意图和具备服务端 resume 语义的传输 journal。

### 用户主动停止

原子设置 `userStopped`、清空 Demand、取消退避和 pending attempt、失效 session、停止 FGS。
由此产生的 bridge close 不得触发自动 repair。等待 foreground/retry window 的旧 consumer 必须由独立
consumer generation 取消；底层 binding 关闭时，每个 renderer lease 必须自行释放 Demand owner，不能
依赖已丢失引用的 UI 再调用 `close()`。manager 主动 fencing binding generation 时，必须先同步终止该
raw session 发出的全部 wrapper lease、通知 close listener 并立即释放 owner，再静默关闭 raw session；
即使 cleanup command 尚未返回也不能延后释放。普通 UI lease 的资源清理最多保留 5 秒 ownership grace，
manager fencing 则立即释放。Stop All 的延迟收尾必须按 endpoint 检查用户意图代次：A 设备的新 workspace
不能阻止 B 设备完成 Stop，也不能让旧文件传输暂停在 A 已恢复后反向覆盖状态。新的 workspace 或手动重连
必须立即释放旧的共享 Demand 退避窗口并从 250ms 档重新开始；旧 15 秒 timer 不得吞掉新点击。

异步 `DisconnectEndpoint` 在 binding 接受请求时捕获 endpoint intent revision，取得 endpoint 锁后必须再次
校验；若期间已有新的 OpenSession 意图，旧 disconnect 直接作废，不能关闭新 winner。Demand 变为空时，
Endpoint Supervisor 必须登记并按精确 `sessionStamp` 退休 physical winner；即使 FGS 已按 canonical Demand
停止，worker 也会继续收敛并清除短暂残留的无人持有连接。普通 `OpenSession` 必须从 Go Supervisor 获取
已有 Demand 的 winner，不能绕开 stop gate 直接拨号；只有显式 `route_override` 的诊断调用允许直连。
手动 endpoint repair 只是加速 hint，renderer 最多等待 1.5 秒；native bridge 卡住时仍继续重新绑定并跟随
Go Supervisor。一次明确的手动重连意图可以跟随期间发生的 `networkChanged` / `foregroundResume` renderer
epoch，不能被普通生命周期更新吞掉；只有显式断开、更新的用户意图或 consumer cancellation 能取消它。

Android 的 Stop All 事件必须由 process owner 保留，直到 manager、文件传输和 native endpoint 清理全部
完成并返回精确 `stopEpoch` ACK。renderer 冻结、重建或 bridge 暂时失败时继续重放未 ACK 的 Stop；旧 ACK
不得清除更新的 Stop。主动停止统一映射为不可恢复的 `user_stopped`，不得被网络恢复逻辑重新分类为瞬时
断线或自动重新建立 Demand。

## 8. 终端与文件恢复语义

### 终端

- 通道不可用或输入交付不确定后立即拒绝后续输入。
- 不缓存、不重放普通字符、回车、控制键、鼠标输入或粘贴。
- 通道恢复后从 revision 0 获取完整 canonical screen，再继续增量更新。
- UI 明确显示“输入已暂停；恢复后请重新输入”。
- 后续协议可增加 `terminalIncarnation + writerId + writerEpoch + inputSeq + payloadHash`，
  但保证范围只能写成 terminal actor 存活期间的 exactly-once enqueue。

### 文件

- 已加载目录和文件预览保持可读，修改与传输暂停。
- 上传以服务端 authoritative offset 和 hash 恢复。
- 上传需要稳定 `uploadId`、状态查询和 completed tombstone，避免 rename 成功但结果丢失。
- mkdir、rename、delete、copy、move 需要 session-independent idempotency key、请求摘要和结果缓存。
- 协议完成前，不自动重试结果不确定的文件 mutation。

## 9. UI 规则

| Surface | 0 到 1.2 秒 | 超过 1.2 秒 | 需要用户动作 |
| --- | --- | --- | --- |
| 机器列表 | 不展示 | 顶部轻提示；列表仍可浏览 | 仅对应机器行展示动作 |
| 终端列表 | 保留缓存列表 | 内容内轻提示 | 无内容时显示动作 card |
| 活动终端 | 立即阻止输入，默认静默 | 内容区输入屏障和紧凑磨砂 card | 展示具体动作 |
| 文件管理器 | 保留缓存目录并只读 | 顶部轻提示 | 展示具体动作 |
| 已加载文件预览 | 保持可读 | 小型状态条 | Close 始终可用 |
| 无缓存内容 | 保持加载态 | 居中 card | 展示具体动作 |

补充规则：

- 机器列表不出现全局大遮罩。
- 标题、返回、关闭和设置始终位于遮罩之外。
- 磨砂只用于 card；终端背景轻微压暗但保持最后输出可读。
- 暂时状态统一使用“正在恢复”或“等待网络恢复”，没有“重试”。
- 成功后 150 到 200ms 淡出，不显示成功 toast。
- 只有授权失效、身份变化、无可用路径等永久错误提供具体动作。

## 10. 分阶段实施

### P0：立即止损

- 删除终端输入恢复队列和自动重放。
- 前台恢复传播 endpoint 失败，禁止 `allSettled` 后无条件 ready。
- 增加 `runtimeId/attachmentId/networkRevision/sessionStamp/attemptId/handleCount/eventQueueDepth` 日志。

### P1：Renderer 所有权

- 引入 renderer lease/attachment。
- operation、stream、handle 和 event queue 按 attachment 归属并整体清扫。
- 加入带 revision 的完整 Demand 对账。

### P2：窄 Go Endpoint Supervisor

- 冻结 JS 并切换 Wi-Fi/蜂窝验证进程级维护。
- Android/iOS 全量启用 Go 自动维护；未配置 endpoint 直接返回配置错误。
- 删除 TS transport fallback、endpoint 灰度和双重 probe/invalidate/retry。

P2 实现约束：

- Android/iOS 通过 C ABI 的 serialized Proto 控制面提交完整 Demand、网络 hint 和前台 revision；
  不通过 WebView bridge 驱动后台维护。
- Demand、foreground、network 和 endpoint repair 任一 supervisor C ABI 调用失败，都将当前 engine
  视为不可继续使用并触发有上限退避的进程级重建；新 engine 必须重放 canonical Demand 和最新 host
  signal，禁止在同一损坏 engine 上原地循环。重建必须先提交最新 host signal、再提交 Demand，离线状态
  不得在两次调用之间抢跑 dial。
- Android FGS ownership 直接跟随 process-owned canonical Demand。非空 Demand 在任何 fallible engine
  操作前取得 FGS ownership，异步 runtime 重建完整回放成功时再次确认；WebView 冻结后不得依赖 JS
  retry 补启动 FGS。旧 Stop All 异步完成时必须重新检查 canonical Demand；新 Demand 已出现时不得停止
  新一代 FGS ownership。
- Android platform pump 关闭最多等待 500ms，不能让不响应 interrupt 的平台任务永久占用 runtime owner
  锁。网络变化取消 Go exchange 后到达的未知、已取消或重复 platform response 必须幂等丢弃，不能把健康
  platform pump 误判为损坏并触发 runtime 重建。Android/iOS 的持久化操作必须由进程级 runtime generation
  与提交锁共同隔离：旧 generation 已开始的操作可以先完成，但新 generation 的读写必须排在其后，尚未进入
  提交区的旧操作必须取消，因此旧 runtime 不能在新 runtime 写入后反向覆盖 pairing 数据。
- 每个 demanded endpoint 只有一个 Go worker。worker 先取得并探测当前 `SessionOwner` winner；
  探测失败后只失效精确 `sessionStamp`，再执行 planner dial 和 application probe。
- `controlRevision` 变化会取消旧 attempt；迟到 probe/dial 不能发布 READY。瞬时失败执行 Go-owned
  backoff，授权、身份、订阅和配额等永久失败进入 BLOCKED。
- endpoint 固定使用 takeover。TS manager 不再 probe、invalidate 或安排物理连接重试，只重新取得 renderer
  binding lease 和页面资源。Go transport READY 早于 application probe 时不得穿透成 UI connected。
- Go fault injector 覆盖 probe failure、dial failure、backoff、永久错误和旧 winner 精确失效。
  真机 Doze 恢复使用 `scripts/android-network-recovery-smoke.sh`；30 分钟验收设置
  `ANYTTY_BACKGROUND_SECONDS=1800`。

### P3：资源级恢复协议

- terminal input fencing/idempotency。
- upload resume、completed tombstone 和 mutation idempotency。
- 如确实需要跨 session 事件补齐，再设计带保留水位的 event cursor；此前只使用全量 snapshot。

## 11. 验收

- renderer A 死亡后，A 的事件不能到达 B，全部 A handles 回到基线；循环 500 次无增长。
- 注入旧 attachment 的迟到 Demand，必须被拒绝且不能停止 FGS/session。
- WebView 冻结 30 分钟而 Go session 存活时，不重新拨号；恢复后画面收敛。
- 小米 Android 16 在“移动网络 + Clash VPN”下，迟到的 TURN candidate 即使表现为 `prflx`，Direct Confirm
  也不得提前释放备用 Relay；两轮预确认请求与 Supervisor 的确认后探测都必须成功并进入 READY。若预确认
  探测确实失败，则必须先收到 Abandon ACK，再在同一 generation 自动切到 Relay/TCP。两条路径全程都无需
  退回设备列表或点击重连。
- 网络回调风暴下，每个 control revision 最多一个 probe/dial。
- 活动 endpoint 失败时该 endpoint 不得发布 connected；APP runtime 和其他 endpoint 不受阻塞。
- 同一 foreground generation 的 Activity/Scene 回调和 JS ack 最多推进一次 foreground revision。
- 相同 native 网络 snapshot 不重复推进 network revision；旧 snapshot 不得覆盖新 snapshot。
- 恢复期间终端接受、排队、重放字节数全部为 0。
- 暂时恢复无需点击的成功率至少 99%，可见提示比例低于 0.5%。
- 机器列表大 card 曝光为 0，重复遮罩为 0。
- Back/Close 成功率 100%，目标响应时间低于 100ms。
