# 移动端网络与 WebView 恢复架构

状态：已接受，分阶段实施。

实施记录（2026-08-20）：P0 的输入不重放和恢复失败传播已落地；P1 的 renderer
lease、attachment 资源隔离、Android revision Demand 对账已落地并完成 review。P2 的窄 Go
Endpoint Supervisor、Android 全量 Demand/host signal 控制面、按 endpoint shadow/takeover 灰度、
TS fallback 和故障注入测试已经落地。真机 30 分钟 Doze/Wi-Fi 蜂窝切换仍必须作为发版验收执行，
不能用单元测试替代。

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
- 单个 endpoint 恢复失败被 `Promise.allSettled` 吞掉，APP 仍发布 ready。

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
2. Android 不拥有 route、session generation 或重试策略。
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
| Android Host | process/runtime ID、Activity/WebView 生命周期、网络和电源 snapshot、FGS、用户 stop | route、session winner、业务重试 | 带 revision 的 host projection 和生命周期 hint |
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

这避免旧 renderer 的迟到 `active=false` 清掉新 renderer 的需求。

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

Android 网络变化只唤醒或延后状态机。READY 必须由端到端业务探测和活动 surface rehydrate
共同确认，不能由 `NetworkCapabilities`、WebSocket open 或 binding health 单独确认。

## 7. 生命周期

### 前台恢复

1. Android 重新采样网络和电源状态。
2. WebView attach 或确认当前 attachment 仍有效。
3. 对完整 Demand 做 reconciliation。
4. Go 优先探测当前 winner；失败后精确失效旧 stamp 并建立新 generation。
5. renderer 重新订阅资源并读取全量 snapshot。
6. 活动 surface 可用后发布 READY。

前台 transaction 中任一有 Demand 的 endpoint 失败，都不能发布 APP ready。并行工作可以全部
完成后聚合失败，但不得吞掉失败。

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
由此产生的 bridge close 不得触发自动 repair。

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

- 先以 shadow mode 记录 probe/dial/backoff 决策。
- 冻结 JS 并切换 Wi-Fi/蜂窝验证需求。
- 按 endpoint 灰度启用 Go 自动维护，保留 TS fallback 一个版本。

P2 实现约束：

- Android 通过 ABI 5 的 serialized Proto 控制面提交完整 Demand、网络 hint 和前台 revision；
  不通过 WebView bridge 驱动后台维护。
- 每个 demanded endpoint 只有一个 Go worker。worker 先取得并探测当前 `SessionOwner` winner；
  探测失败后只失效精确 `sessionStamp`，再执行 planner dial 和 application probe。
- `controlRevision` 变化会取消旧 attempt；迟到 probe/dial 不能发布 READY。瞬时失败执行 Go-owned
  backoff，授权、身份、订阅和配额等永久失败进入 BLOCKED。
- shadow endpoint 只记录 `would_probe/would_dial`，不执行网络动作；其 TS manager 保持原恢复逻辑。
- takeover endpoint 的 TS manager 不再 probe、invalidate 或安排网络重试，只重新取得 renderer
  binding lease 和页面资源。Go transport READY 早于 application probe 时不得穿透成 UI connected。
- `ANYTTY_ENDPOINT_SUPERVISOR_PERCENT` 控制 Android 构建的确定性 endpoint 分桶，范围 0 到 100；
  默认 100。降到 0 可在不改协议的情况下完整回退到 shadow + TS。
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
- 网络回调风暴下，每个 control revision 最多一个 probe/dial。
- 活动 endpoint 失败时 APP 不得发布 ready。
- 恢复期间终端接受、排队、重放字节数全部为 0。
- 暂时恢复无需点击的成功率至少 99%，可见提示比例低于 0.5%。
- 机器列表大 card 曝光为 0，重复遮罩为 0。
- Back/Close 成功率 100%，目标响应时间低于 100ms。
