# 场景设计（v2 · 草案）：5 个真实场景 × 快捷键/页面/数据/形态流转

> 目标：任何新场景都能用同一套规则推导出来，而不是再打补丁。
> 字段默认值/sources 快照/捕获等底层语义见 `PROTOCOL.zh-CN.md` §9。
> 前置：`ARCHITECTURE.zh-CN.md`（分层与不变量）、`PROTOCOL.zh-CN.md`（消息与组件契约）。

## 0. 先定五条总规则（后面所有场景都由它们推导）

1. **状态只在程序**：宿主没有 tab/pane/picker/模式这些概念；宿主只有连接、内容源、授权、安全 overlay。
2. **页面 = 程序状态投影**：`view = f(程序状态, sources, 布局)`；切页面 = 程序状态迁移 + 新 view，不换进程。
3. **焦点唯一且显式**：一帧 ≤1 个 `focused` 内容源；程序进入任何"自己要用键盘"的状态（PANE/prompt/picker）必须先清掉 `focused`。
4. **副作用只走 result**：attach/create/restart/kill/remove/scroll/scrollEnd/copy/history.window/clipboard.read/quit 都是方法调用；每个 RESULT 恰好回一次 `RESPONSE{request_id,epoch}`（`terminal.create` 回 `{endpoint,id}`，其余动作类 data 为空），宿主负责执行与授权；`ok:true` = 已接受并生效，状态权威是 `sources`，两者到达顺序不作保证（按 source id 幂等对账）。
5. **输入优先级全局唯一**（见 §7）：任何输入只会命中一个去向，不允许"静默丢弃"。

## 1. 场景：冷启动 → 进入可用界面

**用户故事**：打开 anytty。若已有终端，直接显示并可输入；若一个都没有，弹 picker 让用户建一个。

| 维度 | 设计 |
|---|---|
| 快捷键 | 无终端时 picker 自动打开：`↑↓` 选择、`enter` 附加、`esc` 关闭；`+ New terminal` 建新的 |
| 页面 | `main`（tabs×panes 骨架） + `picker` overlay |
| 数据 | `hello`（能力/视口）→ `sources`（清单：id/kind/attached/exited）；程序不缓存权威数据，只投影 |
| 形态流转 | `CONNECT → (sources 到达) → 有终端? 自动绑定到 pane0 并 attach → NORMAL；无终端 → EMPTY+picker → create/attach → NORMAL` |

**约束**：`hello` 必须先于任何 `view`；程序在拿到视口尺寸前不得猜布局。

## 2. 场景：多 pane 分屏与焦点切换（最容易出 bug 的地方）

**用户故事**：`%` 横分、`"` 竖分、点击/Tab 切焦点、关掉 pane；新 pane 是空的，用 picker 给它挑终端。

| 维度 | 设计 |
|---|---|
| 快捷键 | `Ctrl-P` 进 PANE（程序拿键盘）、`%`/`"` 分屏、`Tab` 循环、`x` 关当前 pane、`Ctrl-F` picker、`esc` 退 PANE |
| 页面 | 仍是 `main`：body 内每个 pane 一个盒子；空 pane 显示"选择终端/Ctrl-F"的占位盒 |
| 数据 | pane 的**唯一真相**是程序侧的 `pane{id, rect 派生, terminal?: source_id}`；终端内容来自 `sources`；焦点只记 `focusPaneID` |
| 形态流转 | `NORMAL → Ctrl-P → PANE（清 focused）→ % → 新增空 pane、focusPaneID=新 → Ctrl-F → PICKER → enter → result(attach) → sources 回推 attached → 该 pane 绑定 → esc 退 PANE → NORMAL（聚焦 pane 的终端）` |

**硬约束（由总规则 3 推导）**：
- `focusPaneID` 必须指向存在的 pane；不存在 → 视为 `PANE`（不静默丢键）。
- 空 pane 也参与焦点/Tab；点击任何 pane 都切 `focusPaneID`（空 pane 同理，不要求它有终端）。
- 同一客户端内，一个终端同时只出现在一个 slot；跨客户端可多视图（各自 view），尺寸权属见 §6。

## 3. 场景：终端退出 / 程序崩溃回退

**用户故事**：shell 里敲 `exit`；或布局程序自己崩了。

| 维度 | 设计 |
|---|---|
| 快捷键 | 退出态 slot：`Ctrl-E` 重启、`x` 关 slot（走授权，见 §5） |
| 页面 | `main`；退出 slot 的组件角标 `[exited N]` + warning 边框；程序崩溃时宿主显示"最后一棵好树"+提示框 |
| 数据 | `sources` 事件携带 `exited/exit_code`（宿主权威）；程序只做展示与入口 |
| 形态流转 | `终端退出 → sources(exited) → 程序重绘该 slot → Ctrl-E → result(restart) → 宿主重启并重新 attach → sources(exited=false) → NORMAL`；`程序崩溃 → 宿主 RestartPolicy（退避重启）→ 新 epoch 的 HELLO + sources 重放 → 程序重建默认布局 → NORMAL` |

**约束**：

- 重启终端是"重新 attach 同一 terminal id"，不是新建。
- **崩溃恢复（v1 定案）**：宿主重启程序后重放 `HELLO+sources`，程序**重建默认布局**（不恢复崩溃前布局）；
  绑定按 source id 幂等对账（收到 `sources` 就重新绑定，不依赖一次性回调；新 epoch 的 HELLO 已让宿主清空旧缓存/claim/focus，
  关闭 core overlay，并作废旧 epoch 在途请求：能回则回 `RESPONSE{ok:false,error:"epoch reset"}`，旧意图不得执行）。
- 布局持久化预留为 `state.save{blob}` / `state.load`（`hello.features` 里标为预留，v1 不实现；
  宿主按程序 id 持久化不透明 blob，设置容量上限）。

## 4. 场景：回看（scrollback）与复制

**用户故事**：滚轮回看历史、`PgUp/PgDn`、`y` 复制可见内容到本客户端剪贴板、`esc` 回 live。

| 维度 | 设计 |
|---|---|
| 快捷键 | 滚轮 或 `PgUp/PgDn` 回看；`y` 复制；`esc` 回 live（先退出回看，再 `esc` 退 PANE） |
| 页面 | `main`；回看态由**程序**标记（slot.scrolled=true），组件标题显示 `[↑N]`，footer 显示回看组键位 |
| 数据 | 回看是**取数**：`terminal.scroll{endpoint,id,delta}` → `RESPONSE{data.rows}`（历史窗口行）；`terminal.scrollEnd` 是动作；历史游标是**每连接（view）独立**，程序负责记录 |
| 形态流转 | `NORMAL/PANE → 滚轮/PgUp → result(scroll, delta) → RESPONSE(data.rows) → 程序渲染该 slot → y → result(copy) → RESPONSE(ok) → esc → result(scrollEnd) → live` |

**约束**：回看是**组件能力**（terminal 组件实现窗口），程序只决定"什么时候请求"（策略）；
`terminal.copy{endpoint,id,sel?}` 写入**该连接（view）**的剪贴板/OSC52（`sel` 缺省 = 可见区，v1 只实现缺省），不影响其他 view。

## 5. 场景：危险动作、命令面板、退出

**用户故事**：`:` 打开命令面板执行命令；kill slot / 退出要宿主确认。

| 维度 | 设计 |
|---|---|
| 快捷键 | PANE 下 `:` 打开 prompt（输入过滤）；`enter` 执行、`esc` 关闭；宿主确认框：`enter` 允许 / `esc` 拒绝 |
| 页面 | `prompt` overlay（程序画）；确认框是 **core overlay**（宿主画，程序不可覆盖） |
| 数据 | 命令表是程序侧静态数据；执行 = `result`（如 `system.quit{cleanup_owned}`、`terminal.kill`、`terminal.remove`）；宿主把"需要授权"的方法升级为确认框 |
| 形态流转 | `PANE → : → PROMPT（清 focused）→ enter → 程序把命令映射成 result → 宿主判断是否需授权 → CORE CONFIRM（输入只归宿主）→ enter=allow → 执行并回 RESPONSE{ok}；esc=deny → RESPONSE{ok:false,error} → 程序提示` |

**约束**：授权判定在宿主（方法表 + 授权列）；程序不能伪造 `core.*` overlay，也收不到确认框期间的输入；
没有 `workbench.command` 这类宿主业务方法——命令表与语义完全在程序侧。

## 6. 场景：多客户端看同一个终端（fit / mirror）

**用户故事**：两台设备/两个界面同时看一个终端。

| 维度 | 设计 |
|---|---|
| 快捷键 | 同普通场景；非 owner 方多一个"我不是 owner"的提示（footer 或角标） |
| 页面 | 各自 `main`（各自 view）；同一个 `terminal:<endpoint>:<id>` 出现在两个界面 |
| 数据 | 内容流按各自 view 的 revision 推；`sources` 带 `resize_owner`（= 连接 view_id）与 `owner_epoch`，且仅在 `owner_epoch` 变化时推送 owner 字段；尺寸：**谁驱动显示谁是 resize owner**，`attach` 默认 `fit:true`，镜像方 `fit:false` 只跟随 |
| 形态流转 | `A attach(fit=true) → owner(owner_epoch=N)；B attach(fit=false) → follower；A 正常关闭 → B 发起接管（fit=true + expected_owner_epoch=N，CAS）→ owner_epoch=N+1，B 成为新 owner`；存在活跃 owner 时不带 `expected_owner_epoch` 的 `fit:true` 回 `owner conflict` |
| 失败路径 | A 被 SIGKILL（来不及释放）→ owner 租约停更 → **TTL（默认 15s；宿主每 5s 心跳续约）内** B 的 CAS 接管成功，随后 B 驱动 winsize |

**约束**：

- 同一客户端内，一个终端同时只出现在一个 slot；**跨客户端可多视图**（这是唯一允许的多视图来源）。
- PTY winsize 永远只有一个来源（owner）；程序用 `fit` 声明意图，数字仍由各 view 解算的内容矩形提供（非 owner 的矩形不参与）。
- 租约续约触发点：owner 每次成功 resize 即续约；宿主每 5s 对活跃 owner 心跳续约。TTL = `hello.limits.owner_lease_ttl_ms`（默认 15s）。
- owner 超时或断线 → 宿主允许他人 CAS 接管；`expected_owner_epoch` 缺失/不匹配则拒绝并回 `RESPONSE{ok:false,error:"owner conflict"}` + 最新 `sources`。
- `terminal.copy` 与回看游标都按该连接（view）独立：一个客户端复制/滚动不影响另一个 view 的画面与游标。

## 7. 输入去向优先级（唯一优先级表，扣死所有场景）

| 优先级 | 条件 | 去向 |
|---|---|---|
| 1 | Ctrl-Q（永远保留）；Ctrl-C **仅 core overlay 打开时**保留 | 宿主 |
| 2 | core overlay 打开 | 宿主独占（程序收不到任何输入） |
| 3 | `VIEW.keys.all=true` | 程序 |
| 4 | `VIEW.keys.claim` 命中 | 程序（业务快捷键） |
| 5 | 存在 `focused` 内容源 | 按键按其 `input`/kind 分派：终端（声明 `key`）→ 未 claim 键（含 Ctrl 组合）写 PTY；非终端组件（声明 `key`）→ 组件。paste：focused 终端且声明 `paste` → 宿主按 `PROTOCOL.zh-CN.md` §6.8 编码写 PTY，否则 → 程序。鼠标：仅"命中 focused 且终端开 mouse tracking"才透传，否则归程序 |
| 6 | 无 `focused` 或以上都不满足 | 程序（必须自行处理或提示，**不允许静默丢弃**） |

补充规则：

- Ctrl-C 在 core overlay 关闭时按**未 claim 键**处理：focused 终端存在就透传 PTY（可中断前台进程），否则交程序。
- 程序 overlay（picker/prompt）不是宿主可判定条件：程序打开 overlay 时同时发 `keys.all=true`（或把相关键放进 claim）+ 清 `focused`，规则 3/4 自然覆盖。
- 终端类盒子（开 mouse tracking）的 drag/release 一律走 PTY，不参与宿主捕获（§9 与 `PROTOCOL.zh-CN.md` §6.7）。
- 滚轮默认归程序；仅当"focused + 终端开启 mouse tracking + 该面板声明 `wheel`"三者同时满足才透传 PTY。
- 兜底禁止静默丢弃：程序收到就必须处理或显式提示。

**自洽性检查**：场景 2（点击切焦点）走 5 的鼠标分支"否则归程序"；场景 4（滚轮回看）在 PANE 下走 3、在 normal 下走滚轮规则（终端未开 mouse tracking 或未声明 `wheel` → 默认归程序）；场景 5（确认框）走 2。没有场景需要第 7 种规则。

## 8. 流程示例：Ctrl-F"任意时刻"呼出 picker

**为什么任意时刻都行**：Ctrl-F 写进程序的 claim（§7 第 4 行的"claim 内的键→程序"），
宿主不需要知道"picker"是什么；程序在任何状态收到它都能切到 picker 状态。
唯二例外：core overlay 打开（宿主独占）与 程序自身把它重定义（如 picker 内 Ctrl-F=换过滤源）——都是显式规则，不是意外。

**程序自绘 picker（推荐；picker 是程序策略，不是宿主组件）**

```
[用户] Ctrl-F
[runtime] 解析 → key{key:"ctrl-f"} → 安全闸门(否) → claim 命中 → 发给程序
[程序]  状态迁移：overlay=picker；focus 置空；rev++
[程序→宿主] view{epoch,rev,keys:{claim:[...],all:true}, 根:{...各 slot 盒子 focused=false..., pos:{picker 盒子+条目(来自 sources)}}}
[runtime] 解算 → 程序帧 + 组件 surface + overlay 帧 → 合成；无 focused 内容源 ⇒ 按键不进任何 PTY
[用户] ↑/↓ → 程序更新选中项 → 新 view
[用户] Enter → 程序：result{request_id,method:"terminal.attach", params:{endpoint,id,fit:true}}
[runtime] 授权(attach 免) → 执行 → RESPONSE{request_id,epoch,ok} + sources(attached=true)
[程序] overlay=""；本地 slot 记录 source id → 新 view{keys:{claim:[...],all:false}}（该 slot 的终端 focused=true）
[用户] 直接输入 → 未 claim 的键透传 PTY
```

要点：条目数据来自 `sources`（宿主权威）；选择/过滤/绑定策略全在程序；宿主只执行 attach。

**若把 picker 做成 builtin 组件**（当它需要独占资源/独立进程/跨程序复用时才值得）：

```
[程序] view 里放 content.self="picker:local:x" 的盒子，props{items:"sources",filter:"terminal"}，focused=true
[runtime] 按键路由给该组件（组件声明 input:["key"]，它是 focused 内容源）
[组件] 自己维护选中/过滤/渲染（组件帧）
[组件→宿主→程序] event{type:"component", source:"picker:local:x", name:"pick", value:"terminal:local:main"}
[程序] result{terminal.attach,...} 并关闭 overlay
```

代价：组件内焦点/选择语义与独立进程成本（"组件→程序事件"通道 v1 已提供：`component`）；收益：可复用。**决策规则**：
需要特权（PTY/剪贴板/终端池）或独立进程/独占资源 → 做组件；纯展示与选择策略 → 放程序侧 SDK 库，不走协议。



## 8.1 透传冲突：终端里的程序也想要 Ctrl-F

**策略**：双击透传（可配置成前缀+字面键，程序自己决定），机制全部走协议原语。

| 维度 | 设计 |
|---|---|
| 快捷键 | `Ctrl-F` 单击=开 picker；`Ctrl-F` 双击（<300ms）=把第二次按键透传进当前终端 |
| 页面 | 单击时 picker overlay；双击时 picker 立即关闭，画面回到原 pane |
| 数据 | 程序记 `(key, 本地时间戳)`；宿主事件带 `id`；转发只用 `input.forward{event_id,source}`（只写回完整原始块，`PROTOCOL.zh-CN.md` §6.8） |
| 形态流转 | `NORMAL → Ctrl-F(1) → picker（记时间戳）→ Ctrl-F(2) 且 <300ms → input.forward → 宿主写 PTY → 关 picker → NORMAL`；超时/异键 → 维持 picker |

**约束**：core overlay 打开时不参与（宿主独占）；转发支持 key 与 paste 两种事件（paste 只能转发完整原始块，`PROTOCOL.zh-CN.md` §6.8），目标必须已 attached 且声明对应 `input`；失败回 `RESPONSE{ok:false,error}`。

## 9. 场景：鼠标拖拽调整 slot 尺寸

| 维度 | 设计 |
|---|---|
| 快捷键 | 无需；鼠标 `press → drag → release`。键盘等价（可选）：PANE 下 `↑↓←→` 调相邻分隔 |
| 页面 | `main`；分隔条是 1 格**非终端**盒子（id `divider:N`，`input:["mouse"]`，muted 样式） |
| 数据 | 程序存 `slot.ratio`（比例）；拖拽中按 `Δ/(可用宽-1)` 调整两侧比例并重排；不落盘 |
| 形态流转 | `press(divider:2) → 宿主隐式捕获 → drag 帧合并投递 → 程序改 ratio → 新 view → release 结束` |

**约束**：

- 捕获只发生在**非终端**且声明 `input:["mouse"]` 的盒子上；终端类盒子（开 mouse tracking）的拖拽一律走 PTY。
- 捕获在 release、捕获节点从新 view 消失、程序重启（新 epoch）时清除；release 后程序与宿主各清一份拖拽状态。

## 10. 场景：workspace / tab / slot 三层建模（纯程序数据）

```
Workspace{id,name,tabs[],activeTab}
Tab{id,name,slots[],focusSlot}
Slot{id,ratio,source?}          ← source = attach 结果绑定；无 source 即空槽
```

| 维度 | 设计 |
|---|---|
| 快捷键 | tab：前缀模式下 `n`/`c`/`1..9`；workspace：前缀模式下 `w` 循环/新建；都可改成直接 claim（代价见下） |
| 页面 | header（workspace 名+tab 条）、body（当前 tab 的 slots）、sidebar（workspaces/tabs 列表）、overlay 同前 |
| 数据 | 三层全在程序；宿主只看到盒子+内容源+焦点；终端内容与生命周期来自 `sources` |
| 形态流转 | `Ctrl-T → Tab 追加、focusSlot=新空槽 → picker 绑定 source → sources(attached) → slot 渲染终端`；`切 tab/workspace = 只换程序状态，重发 view，不动 PTY` |

**恢复语义**：程序崩溃重启后按 `hello/sources` 重建**默认布局**（终端进程不受影响）；绑定按 source id 幂等对账。布局持久化预留为 `state.save/state.load`（v1 不实现；见 §3 与 `PROTOCOL.zh-CN.md` §1）。

**直接 claim 的代价**：`Alt+1..9` 这类会从终端手里抢走组合键，需程序明确取舍；前缀模式零代价但要两次按键。

## 11. 场景：终端与 slot 的生命周期绑不绑（tmux 式 vs anytty 式）

同一组宿主原语，两套程序策略。终端的生命属于 终端池，slot 的生命属于程序。

| 时机 | tmux 式（同生共死） | anytty 式（完全解绑） |
|---|---|---|
| split 新 slot | `create` → 宿主建 PTY+attach → 绑定 | 空 slot，等待 picker / `attach` 已有终端 |
| 关 slot | `kill`（+`remove`） | 只解绑（不发方法）；终端继续运行 |
| 终端 `exited` | 关 slot（可配置 `remain-on-exit` 保留角标） | slot 保留：`[exited N]` + `Ctrl-E` restart |
| 退出 TUI | 可选 `system.quit{cleanup_owned:true}` 清理本程序创建的终端 | 纯 detach，终端留在 终端池 |

**必须处理的三个细节**

1. 在途竞态：关闭 slot 时 create/attach 可能未返回 → 程序容忍迟到 `sources`（找不到 slot 即按策略忽略或补杀）；绑定以 source id 做幂等对账，不依赖一次性回调。
2. 归属标记（定案）：tmux 式清算用 `terminal.create{ephemeral:true}` 让宿主记归属。清算**仅**发生在：
   程序显式 `system.quit{cleanup_owned:true}`，或用户在确认框确认后；程序崩溃、宿主重启程序（epoch 更替）、断线都**不清算**。
3. 授权：`kill/remove` 是 destructive，走宿主确认框；"关 slot 即杀"要么确认、要么配置显式关闭确认，程序不能绕开。

**验收**：tmux 式——split 后 `v3 ls` 多一个终端，关 slot 后消失；anytty 式——关 slot 后终端仍在 picker 且可重新绑定，`exit` 后 slot 显示角标且 `Ctrl-E` 可恢复；ephemeral 式——`kill -9` 布局程序后终端仍在（新 epoch 可重新 attach）。

## 12. 场景：远程 endpoint（位置透明）

**原则**：协议里没有"远程"这个概念。远程事件由宿主连接层归一化成与本地完全相同的事件流；程序只按 `health` 展示，不按 endpoint 特判。

| 维度 | 设计 |
|---|---|
| 快捷键 | 与本地完全相同；picker 里远程终端多一个 endpoint 标签 |
| 页面 | `main`；离线时该 slot 显示灰化/重连角标（程序策略） |
| 数据 | sources 增加 `health:"ok\|degraded\|offline"`、`last_seen_ms`；屏幕以远端权威快照+revision 为准（不做断点续播） |
| 形态流转 | `terminal.attach(endpoint=remote-1) → 宿主鉴权/连接 → sources(attached,health=ok) → 绑定 → 输入进宿主有界队列 → 断线 → sources(offline)+notice → 宿主退避重连/重订阅 → sources(ok)` |

**方法语义（远程推着我们把错误模型定下来）**

- 所有 RESULT 都带 `request_id` 与 `epoch`，且**总是**收到一次 `RESPONSE{request_id,epoch,ok,data|error}`（异步、可乱序）；`ok:true` = 已接受并生效。
- 动作类（attach/restart/kill/remove/scrollEnd/copy/quit）：`RESPONSE.data` 为空；`terminal.create` 回 `{endpoint,id}`；生命周期结果仍通过 `sources` 推。
- 取数类（`terminal.scroll`、`history.window`、`clipboard.read`）：`RESPONSE.data` 带行/text；回看游标、复制目标都按该连接（view）独立。
- `sources` 是状态权威，与 RESPONSE 到达顺序不作保证：必须按 source id 幂等对账，不依赖顺序。
- 队列满/连接断：notice + health=degraded；**不静默丢输入**；RESULT 饱和回 `throttled`、超限回 `oversize`，VIEW 超限回 `view_rejected`（同 rev 一次）；只有宿主→程序帧超限才降级为 notice。

**约束**：程序永远拿不到 endpoint 凭据；宿主对所有 destructive 方法本地授权（远端还有自己一层）。

## 13. 每个场景的验收（黑盒，tmux）

1. 冷启动：无终端自动弹 picker；建终端后 1s 内可输入。
1.1 透传：`Ctrl-F` 双击后，终端里的 `cat` 能收到 `^F`（用 `cat -v` 验证）。
1.2 远程：断开 endpoint 后 slot 显示重连角标；恢复后自动重订阅、可继续输入。
2. 分屏：`%`/`"` 后新 pane 空且可点击、可 Tab、可 picker 绑定；`esc` 退出 PANE 后按键进 PTY。
3. 退出/崩溃：`exit` 后角标出现；`Ctrl-E` 重启可输入；kill 布局程序后画面保留并自动重启，重建默认布局；确认框打开时 kill → 新 epoch 后确认框消失、旧意图不执行。
4. 回看/复制：`[↑N]`、`y` 后剪贴板可验证、`esc` 回 live；回看取数走 `RESPONSE.data`，两个 view 的游标互不影响。
5. 授权/退出：`kill`/`quit` 弹宿主确认；`esc` 拒绝后回 `RESPONSE{ok:false,error}` 且界面仍可用；`:` 命令面板可见焦点光标。
6. 多客户端：A 关掉后 B 在 TTL（默认 15s、宿主每 5s 续约）内 CAS 接管尺寸；A 被 SIGKILL 同样在 TTL 内接管；两个界面内容同步、尺寸不互踩；活跃 owner 时不带 `expected_owner_epoch` 的 `fit:true` 回 `owner conflict`。

**协议与性能验收**

7. epoch/重启：kill 程序 → 宿主重启 → 程序 `rev=1` 的首帧生效；收到新 epoch HELLO 后旧缓存/claim/focus 已清空；旧 epoch 在途 RESULT 收到 `epoch reset`。
8. 文档-注册表一致性：`HELLO.methods` 与 `PROTOCOL.zh-CN.md` §4 方法表逐项一致；发一个不在 `methods` 里的方法 → `RESPONSE{ok:false,error}`。
9. 背压：RESULT 饱和（超过 `max_inflight_requests`）→ `RESPONSE{ok:false,error:"throttled"}`（不排队）；RESULT 单帧超限 → `oversize`；VIEW 超限 → `view_rejected` 且同一 rev 只收一次；宿主→程序帧超限 → 降级 notice 不断开；仅非法 protobuf 才断开并进入重启策略。
10. 基准：10k 行/s 输出 + 60fps 拖拽 + 500 节点树，记录帧大小与 p99 延迟；饱和压力（程序刷 RESULT / 滚轮风暴 / 大输出）下 Ctrl-Q 的 p99 延迟须有上界（不被业务流量淹没）。
11. create：`terminal.create` 的 RESPONSE 直接带 `{endpoint,id}` 且已完成 attach+fit，程序可立即绑定；缺省 argv/cwd 为宿主全局默认（登录 shell）。
12. ephemeral：`terminal.create{ephemeral:true}` → `kill -9` 程序 → 终端仍在；仅显式 `system.quit{cleanup_owned:true}` 或用户确认后清算。
13. paste：多行粘贴不误执行（bracket 生效）；超长粘贴按 `max_paste_bytes` 分块且顺序正确、每块独立 event_id；截断有可观测告知。
14. 租约：owner 每次成功 resize 续约、宿主每 5s 心跳续约；TTL 15s 到期后可 CAS 接管；`sources` 仅在 `owner_epoch` 变化时更新 owner 字段。
15. RESPONSE/sources 顺序：制造 sources 先于/晚于 RESPONSE 到达两种顺序，程序行为一致（按 source id 幂等对账）。

## 14. 术语表（四份文档统一）

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
