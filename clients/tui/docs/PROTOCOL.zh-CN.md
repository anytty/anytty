# TUI 协议（v2 · 草案，先改这份再写代码）

> 面向"布局程序 ↔ 宿主运行时"的线协议。目标是**像写 web 一样写界面**：
> 组件有属性（声明）、方法（调用）、事件（回调）；宿主是唯一执行方。
> 状态：**草案**。与 `ARCHITECTURE.zh-CN.md` 的不变量一一对应。

> 载荷一律是 protobuf（字段名见下）；文中 JSON 代码块只是**字段形状示意**，
> 由 `anytty tui2 decode` 或调试日志输出，不是线格式。

> **绑定是可选的**：`clients/tui/proto/tui2.proto` 是**唯一真相**；`clients/tui/sdk`（Go 便利库，
> 不依赖 kernel/runtime/render/components）与 `clients/tui/examples/python-shell`（纯标准库
> Python 参考实现）都只是**可选绑定**，不参与协议定义。任何语言只要按本文收发帧即可
> 成为布局程序。

## 0. 传输与信封（二进制）

- 通道：stdin/stdout 双向，帧格式 `u32 长度(大端) | u8 类型 | protobuf 载荷`；stderr 只放日志。
- 帧类型是一级类型、编号固定（只能扩载荷，不能改编号）：

| 编号 | 类型 | 方向 | 载荷 |
|---|---|---|---|
| 1 | HELLO | 宿主→程序 | `view_id`/schema/epoch/视口/组件/方法/事件/features/限制 |
| 2 | VIEW | 程序→宿主 | `epoch`+`rev`+`keys`+盒子树（全量快照） |
| 3 | EVENT | 宿主→程序 | sources/key/paste/mouse/wheel/resize/notice/component/view_rejected |
| 4 | RESULT | 程序→宿主 | `request_id`+`epoch`+method+params |
| 5 | RESPONSE | 宿主→程序 | `request_id`+`epoch`+`ok`+`data?`/`error?` |

- claim 是 VIEW 的字段，与视图同帧原子生效（§6.5）；不存在独立 KEYS 帧。
- 解析边界（顺序写死）：先校验长度（0 或超过 `max_message_bytes` → 直接拒帧）→ 再分配缓冲区 →
  再解码 protobuf；超限判定先于解码失败判定。方向非法（程序发 HELLO/EVENT/RESPONSE，或宿主发
  VIEW/RESULT）一律按 decode error 处理并断开程序。
- 超限去向（按方向区分）：
  - 宿主→程序的帧超限：**合并/降级为 notice**，不得断开；这是唯一允许「降级为 notice」的方向。
  - 程序→宿主的 VIEW 超限（`max_nodes` 或 `max_message_bytes`）：宿主拒绝该帧并回
    `view_rejected{epoch,rev,reason}`（§3），同一 (`epoch`,`rev`) 只回一次；程序应缩小视图或降频，
    不得依赖逐帧 notice。
  - 程序→宿主的 RESULT 超限：回 `RESPONSE{ok:false,error:"oversize"}`；在途 RESULT 数超过
    `max_inflight_requests` 饱和时回 `RESPONSE{ok:false,error:"throttled"}`（不排队，§4）。
- 仅解码失败才断开程序，交由重启策略接管。
- 背压（宿主实现要求，不暴露新原语）：对 paste 与输出做每帧合并；`max_inflight_requests` 限制在途请求；
  输入解析与组件渲染设预算隔离，互不阻塞。
- 版本：`HELLO.schema` 决定语义版本；加可选字段不升版本，改/删语义必须升版本并写迁移。
- 调试：`anytty tui2 decode < log.bin` 把帧流解成可读 JSON；`ANYTTY_TUI2_LOG=1` 时宿主把双向帧
  解码为逐行 JSON 写入调试日志（仅调试用，不参与协议，也不是线格式）。
- 性能：全量快照 `rev` 单调；`rev` 作用域 = (`epoch`,`view`)，宿主按此丢弃过期帧（§0.5）。
  树很大时可后续加 delta（`rev_base`+补丁），但 v1 不做——先保证正确与可调试。

## 0.5 会话、view 身份与 epoch（重启后状态如何对齐）

- `HELLO.view_id`：宿主分配的连接标识；**v1 一个客户端连接 = 一个 view**。attach 的 fit/owner、
  copy 的目标剪贴板、history 游标都按该连接（view）寻址。
- `HELLO.epoch`（u64）：宿主维护，每次启动程序、以及每次重启程序后 +1；同一程序进程生命周期内不变。
- `VIEW`/`RESULT`/`RESPONSE` 必须携带 `epoch`；`rev` 作用域 = (`epoch`,`view`)，宿主按此丢弃过期帧。
- `request_id` 作用域 = (`epoch`, 连接)；跨 epoch 的同号 `request_id` 互不相关。
- 宿主收到新 epoch 的 HELLO 时**原子重置**：视图缓存、拖拽捕获、claim、focus、core overlay 全部清零，
  并**作废全部在途请求**——属于旧 epoch 的在途 RESULT 能回则回
  `RESPONSE{ok:false,error:"epoch reset"}`（连接已断/应答已发出则跳过），旧意图一律不得再执行。
- 程序重启后 `rev` 重新计数是合法的：**重启后的第一帧必须生效**。
  测试：kill 程序 → 宿主重启 → 程序发 `epoch=新, rev=1` 的 VIEW 不被丢弃、按它渲染。
  测试：确认框（core overlay）打开时 kill 布局程序 → 新 epoch 后确认框必须消失，旧确认意图不得执行。

## 1. 握手：hello（宿主→程序，启动与重启后各一次）

```json
{"type":"hello","schema":1,"view_id":"view:client-a:1","epoch":3,"cols":120,"rows":32,
 "components":["terminal"],
 "events":["key","paste","mouse","wheel","resize","sources","notice","component","view_rejected"],
 "methods":["terminal.attach","terminal.create","terminal.restart","terminal.kill","terminal.remove",
            "terminal.scroll","terminal.scrollEnd","terminal.copy","history.window","clipboard.read",
            "input.forward","system.quit","endpoint.sync"],
 "features":{"component":true,"state.save":false,"state.load":false},
 "limits":{"max_nodes":4096,"max_message_bytes":1048576,"max_paste_bytes":65536,
           "max_inflight_requests":64,"owner_lease_ttl_ms":15000}}
```

- `view_id` 由宿主分配；同一连接的所有 VIEW 共享该 view 的 `rev` 空间（§0.5）。
- `HELLO.methods` 必须由 §4 的权威方法表生成；不在 `methods` 里的方法，宿主拒绝并回
  `RESPONSE{ok:false,error}`（§4）。
- `limits`：`max_message_bytes` 为单帧上限；`max_paste_bytes` 为单块粘贴上限（§6.8）；
  `max_inflight_requests` 为未回 RESPONSE 的 RESULT 上限（§4）；`owner_lease_ttl_ms` 为 owner 租约 TTL（§5）。
- `features`：`true` = v1 可用；`false` = 预留、v1 不实现，程序必须容错。
  `component` = 组件语义事件；`state.save`/`state.load` = 宿主按程序 id 持久化不透明 blob（设置容量上限），
  **v1 不实现**（崩溃恢复见 §0.5 与 `SCENARIOS.zh-CN.md` §3）。
- 程序必须按 `components/methods/features` 判断能力，不得硬编码版本。
- 时间策略由程序自己实现（子进程自带时钟/定时器）；`timer/tick` 预留给沙箱化程序，v1 不实现。

## 2. 视图：view（程序→宿主）

```json
{"type":"view","epoch":3,"rev":7,
 "keys":{"claim":["ctrl-p","ctrl-f","?"],"all":false},
 "root":{"id":"root","flow":"col","children":[ ... ]}}
```

- `keys` 与视图**同帧原子生效**：`claim[]` 是程序要接收的按键，`all:true` 表示所有键都发程序；不存在独立 KEYS 帧。
- `rev` 作用域 = (`epoch`,`view`)；宿主按此丢弃过期帧；epoch 更新时缓存/claim/focus 已在 HELLO 处原子重置（§0.5）。
- VIEW 超限（`max_nodes`/`max_message_bytes`）被宿主拒绝并回 `view_rejected`（§0/§3）；
  同一 (`epoch`,`rev`) 只回一次，宿主不得逐帧发 notice。
- 程序侧业务对象（slot/pane/tab/workspace）不出现在 view 里；绑定表现为程序把某个盒子的 `content.self` 指向 source id。

节点字段（全部可选，未给出即默认）：

| 字段 | 类型 | 语义 |
|---|---|---|
| `id` | string | 命中测试/事件回传用；`core.*` 保留给宿主 |
| `size` | `{width,height,flex}` | 固定尺寸或弹性 |
| `pos` | `{x,y}` | 绝对摆放=overlay 子树，合成在组件之后 |
| `flow` | `row/col/stack` | 子盒子排布 |
| `visible` | bool | 隐藏但仍占位 |
| `cursor` | `{row,col,shape,visible}` | 程序光标（相对盒子 rect） |
| `content` | `{text}` / `{lines[]}` / `{self, props?}` | 文字 / 行 / **内容源（组件）引用**；`props` 为程序下发的组件属性 |
| `input` | `["key","paste","mouse","wheel"]` | 该盒子接收哪些输入 |
| `focused` | bool | 键盘焦点（宿主据此决定按键去向） |
| `style` | string | **显式样式串**（`fg:#RRGGBB;bg:#RRGGBB;bold;dim;reverse`）；空 = 宿主默认样式 |

- **组件属性由程序下发**：`content.props`（`map<string,string>`）是程序 → 组件的属性/样式通道。
  程序按组件约定的 key 下发（例如 terminal 的 `chrome.border`、`chrome.title`、`chrome.border_focus`、
  `chrome.border_dead`、`chrome.badge`，值为显式样式串），组件自行解释；**未识别的 key 一律忽略**。
  宿主只透传（kernel/runtime 原样带到 placement，宿主组件工厂原样转发），不解释、不做配色，
  组件缺省回落自身内置默认。
- `border`（wire field 6）**已删除**（`reserved 6`）：rect 是纯几何，内核不画边框也不做内缩。
  边框/标题/角标由**内容负责**（程序用文本行或 widget 自绘）或由**组件自绘**；组件通过自己的
  inset 声明"要占多少 chrome"，宿主只按 inset 算 PTY 尺寸与光标（§5 terminal、§9.6）。

## 3. 事件：events（宿主→程序）

| type | 负载 | 说明 |
|---|---|---|
| `key` | `id`,`key`,`char` | 含 `ctrl-p`/`page-up`/`enter` 等规范化键名；`id` 供 `input.forward` 引用 |
| `paste` | `id`,`text` | 粘贴文本块；宿主已按 §6.8 编码/分块，每块独立 `id`、严格保序；`id` 供 `input.forward` 引用 |
| `mouse` | `action`,`button`,`x`,`y`,`node` | `node` 为命中盒子 id |
| `wheel` | `delta`,`x`,`y`,`node` | |
| `resize` | `cols`,`rows` | 视口变化，程序须重排 |
| `sources` | `items[]` | 内容源清单（id/kind/title/attached/exited/health/resize_owner/owner_epoch/last_seen_ms） |
| `notice` | `level`,`message` | 宿主的**非请求类**信息（错误/信息）；请求类结果一律走 RESPONSE |
| `component` | `source`,`name`,`value` | 组件→程序的语义事件，宿主仅中转；`hello.features.component=true` 时可用（v1） |
| `view_rejected` | `epoch`,`rev`,`reason` | 程序 VIEW 被拒回执（超 `max_nodes`/`max_message_bytes`）；同一 (`epoch`,`rev`) 只回一次 |

- `wheel` 默认推给程序；仅"focused + 终端开 mouse tracking + 该面板声明 `wheel`"三者同时满足才透传 PTY（§6.5）。
- `sources` 的 owner 字段（`resize_owner`/`owner_epoch`）仅在 `owner_epoch` 变化时推送，避免通知风暴（§5）。
- `response` 不再是 event 的一种：它是独立帧类型（编号 5），且每个 RESULT 恰好触发一次（§4）。

## 4. 方法：result（程序→宿主，唯一副作用入口）

这是**唯一权威方法注册表**；`HELLO.methods` 必须由此表生成。

```json
{"type":"result","request_id":42,"epoch":3,"method":"terminal.scroll",
 "params":{"endpoint":"local","id":"terminal:local:main","delta":10}}
```

| method | params | 授权 | data | 说明 |
|---|---|---|---|---|
| `terminal.attach` | `endpoint`,`id`,`fit?:true`,`expected_owner_epoch?` | 免 | 无 | 绑定到**该连接（view）**；`fit:true`（默认）成为 resize owner；若当前存在**活跃** owner，必须带 `expected_owner_epoch` 做 CAS，缺失/不匹配回 `RESPONSE{ok:false,error:"owner conflict"}`；`fit:false` 只跟随 |
| `terminal.create` | `endpoint`,`argv?[]`,`cwd?`,`env?{}`,`title?`,`ephemeral?:false` | 免 | `{endpoint,id}` | 宿主分配 `id`，并在应答前直接完成 attach+fit，程序可立即绑定；`argv`/`cwd` 缺省用宿主全局默认（登录 shell），程序参数覆盖默认 |
| `terminal.restart` | `endpoint`,`id` | 免 | 无 | 重启同一 terminal id，不新建 |
| `terminal.kill` | `endpoint`,`id` | 宿主确认 | 无 | 终止进程 |
| `terminal.remove` | `endpoint`,`id` | 宿主确认 | 无 | 删除记录 |
| `terminal.scroll` | `endpoint`,`id`,`delta` | 免 | 历史窗口行 | 取数：`data.rows`（回看窗口）+ 游标信息 |
| `terminal.scrollEnd` | `endpoint`,`id` | 免 | 无 | 回到 live（动作） |
| `terminal.copy` | `endpoint`,`id`,`sel?:{mode:"char\|line\|block",start,end}` | 免 | 无 | 写入**该连接（view）**的剪贴板/OSC52；`sel` 缺省 = 可见区（v1 只实现缺省） |
| `history.window` | `endpoint`,`id`,`offset`,`rows` | 免 | 行 | 取数：`data.rows` |
| `clipboard.read` | — | 宿主确认 | text | 取数：`data.text` |
| `input.forward` | `event_id`,`source` | 免 | 无 | 只允许退回一个已收到的**完整原始** key/paste 块；文本不许程序自带字节（§6.6/§6.8） |
| `system.quit` | `cleanup_owned?:false` | 宿主确认 | 无 | `cleanup_owned:true` = 清算本程序创建的 ephemeral 终端；这是唯一的程序侧清算入口（另有用户确认路径），崩溃/重启不清算（§5） |
| `endpoint.sync` | `endpoint`,`kind?`,`socket?`,`address?`,`connect_mode?` | 免 | 无 | 注册一个配置 endpoint（M23/M27 append-only）：`kind=daemon` 时宿主后台连接并随后用 `sources` 发布终端池终端清单；`kind=command` 无需注册。语义见 `ENDPOINTS.zh-CN.md` §2.3 |

`MethodParams` 在 §4 参数之外新增 append-only 字段：`kind`(17)、`socket`(18)、
`connect_mode`(19)、`address`(20)，由 `terminal.create`/`terminal.attach`/`endpoint.sync`
携带 endpoint 连接元数据（`ENDPOINTS.zh-CN.md` §2.3；远程 tcp 见 `REMOTE.zh-CN.md`）。

规则：

- 每个 RESULT **总是**收到恰好一次 `RESPONSE{request_id,epoch,ok,data?,error?}`：异步、可乱序；
  `request_id` 作用域 = (`epoch`, 连接)。动作类 `data` 为空（`terminal.create` 例外，回 `{endpoint,id}`），
  取数类 `data` 见上表；`notice` 只用于非请求类的宿主信息。
- `RESPONSE{ok:true}` = 请求已被接受并生效；内容源状态的权威是 `sources`。两者到达顺序**不作保证**，
  程序必须按 source id 幂等对账，不得依赖"RESPONSE 先于 sources"之类的顺序（测试见 `SCENARIOS.zh-CN.md` §13）。
- 方法名不在 `HELLO.methods` 里 → 宿主拒绝并回 `RESPONSE{ok:false,error}`；参数非法 → 同样回 `RESPONSE{ok:false,error}`，不静默。
- 背压：在途（已发出、未回 RESPONSE）RESULT 数超过 `max_inflight_requests` → 立即回
  `RESPONSE{ok:false,error:"throttled"}`，**不排队**；单帧超过 `max_message_bytes` → 回 `"oversize"`（§0）。
- 宿主收到新 epoch 的 HELLO 时，旧 epoch 在途 RESULT 回收 `RESPONSE{ok:false,error:"epoch reset"}`（§0.5）。

## 5. 内容源 = builtin 组件（sources）

```json
{"id":"terminal:local:main","kind":"terminal","title":"main",
 "endpoint":"local","terminal_id":"main","attached":true,"exited":false,"exit_code":0,
 "health":"ok","resize_owner":"view:client-a:1","owner_epoch":7,"last_seen_ms":123456}
```

多客户端规则（位置透明、尺寸唯一）：

- `attach` 默认 `fit:true`：驱动显示方成为 **resize owner**（owner 标识 = 该连接（view）的 `view_id`）；
  `fit:false` 只跟随。存在**活跃** owner 时，`fit:true` 必须带 `expected_owner_epoch` 做 CAS，
  缺失/不匹配回 `RESPONSE{ok:false,error:"owner conflict"}`（§4）。
- owner 租约：TTL = `hello.limits.owner_lease_ttl_ms`（默认 15s）。续约触发点写死两条：
  ① owner 每次成功 resize 即续约；② 宿主每 5s 对活跃 owner 心跳续约。超时或断线后其他客户端可 CAS 接管
  （`expected_owner_epoch` 匹配 `owner_epoch` 才成功）。
- `owner_epoch` 随 owner 变更递增；`sources` 只在 `owner_epoch` 变化时推送 owner 字段（避免通知风暴）。
- 同一客户端内，一个终端同时只出现在一个 slot；跨客户端可多视图（各自 view，内容按各自 revision 推送）。
- 内容源的权威状态（attached/exited/health/owner）永远来自宿主；程序按 source id 幂等对账（§4）。

**ephemeral 清算（定案）**：`terminal.create{ephemeral:true}` 创建的终端**仅在**两种情形清算：
① 程序显式调用 `system.quit{cleanup_owned:true}`；② 用户通过宿主确认框确认清算。程序崩溃、
宿主重启程序（epoch 更替）、连接断开都**不清算**。测试：ephemeral → `kill -9` 程序 → 终端仍在。

组件契约（每个 builtin 组件在文档里必须给出同样三张表：属性/方法/事件）：

### terminal

- **属性**（程序在 view 里声明）：`content.self`、`focused`、`input`、`size/pos`、`content.props`；
  `style` 只作用于组件自身声明的内容样式。terminal 识别的 props（均为显式样式串，缺省回落内置默认）：
  `chrome.border`、`chrome.title`、`chrome.border_focus`、`chrome.border_dead`、`chrome.badge`；
  未识别的 key 忽略。
- **chrome 归属**：组件自绘边框/标题/焦点标记/角标（`[exited N]`、`[↑N]`），并**声明自己的 inset**
  （当前 = 1）。宿主不画边框：PTY winsize 与光标位置 = placement rect − 组件 inset（§6.3、§9.6）。
- **方法**（程序调 result）：见 §4 权威表（attach/create/restart/kill/remove/scroll/scrollEnd/copy）。
- **事件**（宿主推给程序）：`sources`（生命周期/owner）、`key/paste/mouse/wheel`、`component`、`notice`。
- **能力**：滚动回看（取数）、复制到该连接（view）剪贴板（`sel` 缺省可见区）、退出角标、尺寸对齐（owner + 租约）。
- **"嵌套"的唯一合法含义**：在终端里再跑一个复用器（普通进程），或另一客户端 attach 同一终端；
  宿主不提供组件内嵌，组件之间无直连（`ARCHITECTURE.zh-CN.md` §2.6）。

### （下一个组件在这里追加同样的三张表）

## 6. 焦点 / 鼠标 / 尺寸 / z 序（与架构不变量一一对应）

1. 一帧 ≤1 个 `focused`；索引非法时按键交程序，不静默进 PTY。
2. 鼠标只有在"命中 focused 且组件开启 mouse tracking"时才透传 PTY；终端类盒子的拖拽一律走 PTY（§6.7）。
3. 尺寸：程序只给盒子几何；runtime 由 placement 内容矩形推 PTY winsize，terminal 组件写入；驱动方是
   resize owner（CAS + 租约 TTL），镜像方 `fit:false` 跟随（§5）。
4. `pos` 子树最后合成；`core.*` overlay 永远最上、程序不可覆盖。
5. 程序模式（PANE/prompt 等）属于程序策略：程序必须在打开时清掉 `focused`。
6. 保留键：Ctrl-Q 永远归宿主；Ctrl-C **仅当 core overlay 打开时**归宿主，否则按未 claim 键处理（可透传 PTY）。

## 6.5 路由声明（claim）：随 VIEW 同帧生效

claim 是 VIEW 的字段，不存在独立 KEYS 帧，也不存在"先发 claim 后发 view"两步；
claim 与视图**原子生效**，程序切换模式时只需发一帧新 view（`rev` 递增）。

```json
{"type":"view","epoch":3,"rev":7,
 "keys":{"claim":["ctrl-p","ctrl-f","ctrl-t"],"all":false},
 "root":{ ... }}
```

路由优先级（全局唯一，与 `SCENARIOS.zh-CN.md` §7 同一张表）：

| 优先级 | 条件 | 去向 |
|---|---|---|
| 1 | Ctrl-Q（永远保留）；Ctrl-C **仅 core overlay 打开时**保留 | 宿主 |
| 2 | core overlay 打开 | 宿主独占（程序收不到任何输入） |
| 3 | `keys.all=true` | 程序（此时程序必须已清掉 `focused`） |
| 4 | `keys.claim` 命中 | 程序（业务快捷键） |
| 5 | 存在 `focused` 内容源 | 按其 `input`/kind 分派：终端（声明 `key`）→ 未 claim 的键写 PTY（含 Ctrl 组合）；非终端组件（声明 `key`）→ 组件。paste：focused 终端且声明 `paste` → 宿主按 §6.8 编码写 PTY，否则 → 程序。鼠标：仅"命中 focused 且终端开 mouse tracking"才透传（§6.2）。滚轮：默认 → 程序；仅"focused + 终端开 mouse tracking + 该面板声明 `wheel`"三者同时满足才透传 PTY |
| 6 | 无 `focused` 或以上都不满足 | 程序（必须处理或提示，禁止静默丢弃） |

- Ctrl-C 在 core overlay 关闭时按未 claim 键处理：focused 终端存在就透传 PTY（能中断前台进程），否则交程序。
- 组件不解析快捷键：要么收到原始输入，要么收不到。
- 程序要触发组件能力（滚动/复制/重启）时，仍走 `result` 方法，不走按键。

## 6.6 透传冲突：全局键 vs 终端内快捷键（双击透传）

Ctrl-F 既是我们声明的全局键，也可能是终端里 vim/emacs 的键。解决方式是**程序策略 + 宿主转发原语**：

```
[用户] Ctrl-F 第 1 次 → claim 命中 → 程序：执行全局动作（如开 picker），记 (key, now)
[用户] Ctrl-F 第 2 次（窗口内，如 300ms）→ 程序：判定为双击
        → result{method:"input.forward", params:{event_id:第2次事件, source:"terminal:local:main"}}
[宿主] 校验：该 event_id 存在、source 已 attached 且接受该事件类型（key/paste）→ 重新编码 → 写入该 PTY
[否则] 窗口超时/下一个非该键事件 → 正常处理全局动作
```

- 键/文本编码只存在于宿主：程序只说"把第 N 号事件退给某个内容源"，不碰字节
  （paste 转发只接受完整原始块，不许程序拼接/自带文本，§6.8）。
- 窗口计时用程序自己的时钟/定时器（v1 无宿主 timer 原语）。
- 同一原语可表达：前缀+字面键（`Ctrl-P` 后 `Ctrl-F` 转发）、双前缀（把 Ctrl-P 本身转发）、状态性放行。
- 校验失败（事件过期/目标不存在）→ `RESPONSE{ok:false,error}`，不静默。

## 6.7 拖拽捕获（隐式）

- **终端类盒子（开启 mouse tracking）的拖拽事件一律走 PTY，不参与宿主捕获**（如终端里的鼠标选择、tmux 分屏拖拽）。
- 宿主捕获只针对**非终端**且声明 `input:["mouse"]` 的盒子：`mouse press` 命中后，本次拖拽的后续
  `drag`/`release` **一律定向回该盒子 id**（即使指针移出其矩形），直到按键释放。
- 捕获在以下任一情况清除：按键 release、捕获节点从新 view 消失、程序重启（新 `epoch` 的 HELLO）。
- `mouse` 事件负载：`action`（press/drag/release/up）、`button`、`x`,`y`、`node`（捕获期间固定为按下时的节点）。
- 宿主按帧合并高频 `drag`（每帧只投递最新坐标）。

## 6.8 paste 编码与分块（宿主职责）

- 宿主负责全部字节级编码：按终端模式决定是否做 bracket 包裹（`ESC[200~…ESC[201~`）、CR/LF 归一、
  按 `hello.limits.max_paste_bytes` 分块。
- 每个块是独立事件（独立 `event_id`）且严格保序投递；禁止跨块合并，程序不得假设一次粘贴只对应一个事件。
- `input.forward` 只接受并写回**完整原始块**；程序不得拼接、截断或自带字节。
- 任何超限截断/丢弃都必须可观测（`notice` 或 `RESPONSE{ok:false,error}`），不得静默。
- 验收：多行粘贴不误执行（bracket 生效）；超长粘贴分块且顺序正确（`SCENARIOS.zh-CN.md` §13）。

## 7. 版本

- `schema` 当前 1；加可选字段不升版本，改语义/删字段必须升版本并在本文件记录迁移。
- 迁移（schema 1 内，v2 收口）：节点字段 `border`（wire field 6）**已删除并 reserved**——内核不再有
  border 概念，宿主不读该字段（旧程序发了也只是 protobuf 未知字段，被安全忽略）；边框/标题改由内容
  或组件自绘（§2、§5、§9.6）。新增可选字段 `content.props`（`map<string,string>`）作为
  程序 → 组件属性/样式通道（§2）。同一 schema 内 `style` 的语义从"主题 token"改为"显式样式串"。
- `epoch`/`view_id` 不属于 schema 版本：它们只标识"当前是哪个程序进程/哪条客户端连接的帧"，见 §0.5。

## 8. 术语表（四份文档统一）

| 术语 | 定义 |
|---|---|
| 盒子 box | 框架唯一图元；view 树节点，程序声明几何，宿主解算 rect/flex/hit |
| 内容源 source | 宿主管理的不透明内容单元（如一个终端），有 id/kind/health/lifecycle |
| 组件 component | 一类内容的能力实现（属性/方法/事件）；builtin 组件跑在宿主进程内 |
| slot | 程序侧业务对象，可绑定一个 source；宿主不认识 slot |
| workspace / tab | 程序侧业务对象；宿主不认识 |
| view | v1：一个客户端连接 = 一个 view；宿主在 HELLO 分配 `view_id`；多客户端 = 多 view |
| epoch | u64；宿主每次启动/重启程序 +1；新 epoch 的 HELLO 作废在途请求并重置状态 |
| rev | 程序内单调递增的 view 版本；作用域 = (`epoch`,`view`)，宿主据此丢弃过期帧 |
| claim | VIEW 内的按键路由声明 {claim[],all}，与视图同帧原子生效（不存在独立 KEYS 帧） |
| pane / 面板 | 非协议术语：只在程序侧业务语境使用；几何一律称"盒子" |

## 9. v1 语义补全（终审遗留项收口）

### 9.1 盒子字段默认值表（内核解算与命中的唯一依据）

| 字段 | 默认 | 语义（写死） |
|---|---|---|
| `id` | `""` | 空 id 不参与命中/事件，但正常渲染占位 |
| `visible` | `true` | `false` = 不渲染、不命中、**不占位**（等价 display:none） |
| `flow` | `col` | 仅对有子盒子的容器有意义 |
| `size` | `{0,0,0}` | 宽/高为 0 = 取内容固有尺寸；无固有尺寸则拉伸填满父容器；`flex` 只在同层分配时生效 |
| `pos` | `null` | 非 null = 脱离 flow 绝对定位（overlay 子树） |
| `cursor` | `null` | 无程序光标 |
| `content` | `null` | 空盒子（可命中，可作占位/拖拽区）；`content.props` 默认空 map，组件未识别 key 忽略 |
| `input` | `[]` | 默认不接受任何输入 |
| `focused` | `false` | 一帧内最多一个为 true |
| `style` | `""` | 显式样式串（`fg:#RRGGBB;bg:#RRGGBB;bold;dim;reverse`）；空 = 宿主默认样式；未知片段安全降级为不着色 |

**z 序（写死）**：同一父下按声明顺序，后声明者在上；`pos` 子树整体在常规流之上；core overlay 永远最上。

### 9.2 sources 快照语义（幂等对账的前提）

- `sources` **每次都是全量清单**：每项包含全部字段；不存在"缺字段=不变"的 delta 语义。
- `resize_owner`/`owner_epoch` 恒存在（无 owner 时为空串/0）；**仅当 owner_epoch 变化**才额外触发一次推送（防风暴），但任何 sources 推送都携带完整 owner 信息。
- 程序必须按 `id` 幂等对账：本地绑定只以最新快照为准，不依赖事件到达顺序。

### 9.3 三个小语义

- `terminal.copy` 缺省"可见区" = 发起时该 view 的当前屏幕内容（live 或回看窗口，按当时可见）。
- `terminal.remove` 对**运行中**终端一律拒绝：`RESPONSE{ok:false,error:"still running; use terminal.kill"}`。
- `input.forward` 的 `event_id` 有效期：宿主保留每个 view 最近 64 个输入事件；过期 → `RESPONSE{ok:false,error:"event expired"}`。

### 9.4 鼠标与捕获（与非终端组件统一）

- 终端类盒子（mouse tracking 开启且 focused）：鼠标/滚轮按 §6.5 第 5 行处理（滚轮需声明 `wheel`）。
- 非终端盒子声明 `input:["mouse"]`：宿主捕获该次拖拽（§6.7），点击/悬停命中后作为事件给程序。
- 两者都不会同时发生：捕获只在非终端盒子上建立。

### 9.5 滚动归属

v1 盒子**没有 `scroll` 字段**：回看 = 程序发起取数（`terminal.scroll`）+ 程序自行渲染；内核只负责 rect/flex/hit/overlay/cursor，不实现滚动。

### 9.6 内核实现补充语义（收口后）

- **rect 是纯几何**：内核没有 `border` 概念、不画边框、不做内缩；盒子的内容区 = rect 全量。
  固有尺寸 = 内容尺寸本身（内容 2×1 ⇒ 固有 2×1）；无内容或仅 `content.self` 时拉伸填满父容器。
- **chrome 归属**：边框/标题/角标由内容负责（程序用文本行或 `sdk/widgets` 自绘）或由组件自绘。
  组件声明自己的 **inset**（terminal 组件当前 inset=1），宿主只用 `rect − inset` 算 PTY winsize
  与光标偏移，不再假设固定的"W>=3 减 2"。
- **文本不换行**：超宽按 rect 宽截断、超高裁剪（换行/滚动不属于内核）。
- **样式归属**：`box.style` 是不透明样式串——程序发**显式样式**（`fg:#RRGGBB;bg:#RRGGBB;bold;dim;reverse`），
  宿主只做"显式样式 → SGR"翻译，不持有主题/色板；宿主内部组件可用内建 token
  （`accent` `border_focus`…），但那是宿主实现细节，程序不需要也不应依赖。程序不产生转义字节。
- **样式继承**已删除：`box.style` 为空就是宿主默认样式；没有 border 可继承。
- **根盒子**：始终等于视口 rect，自身 `size` 被忽略。

