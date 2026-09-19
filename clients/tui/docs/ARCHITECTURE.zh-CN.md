# TUI 架构（v2 · 草案，先改这份再写代码）

> 本文只回答"谁负责什么、谁不能碰什么、数据怎么流"。协议细节见 `PROTOCOL.zh-CN.md`。
> v1 语义细节（字段默认值、sources 快照、鼠标捕获、滚动归属）见 `PROTOCOL.zh-CN.md` §9。
> 状态：**草案**。任何一层没写清职责，就不要开工。

## 1. 分层图（三个角色 + 一条数据面）

```
┌────────────────────────────────────────────────────────────────────┐
│ 布局程序 layout program（用户写的，跑在子进程里，通过二进制帧收发协议）│
│  职责：决定"界面上有什么、摆在哪、谁有焦点"                            │
│  产物：view（盒子树 + claim） + result（方法调用）                     │
│  禁止：直接调 PTY/剪贴板/daemon；不得发"数字尺寸命令"（只能改盒子几何）  │
└───────────────▲───────────────────────────────────┬────────────────┘
   events / RESPONSE（hello/sources/key/paste/mouse/wheel/notice/component/view_rejected）
                                                      │ view / result
┌───────────────┴───────────────────────────────────▼────────────────┐
│ 宿主运行时 runtime（唯一可信方，代码在 clients/tui/runtime）                  │
│  职责：进程与协议、事件路由、帧合成、安全内核、生命周期、尺寸权属          │
│  产物：给终端设备的 ANSI 帧 + 给 PTY 的字节                            │
│  禁止：业务策略（不知道"tab/pane/picker"是什么）                        │
└───────▲───────────────────────┬───────────────────────▲────────────┘
        │ frame                 │ bytes                  │ attach/resize/...
┌───────┴───────────┐  ┌────────┴─────────┐  ┌──────────┴───────────┐
│ 终端设备（TTY）     │  │ PTY / daemon      │  │ builtin 组件（进程）   │
│ 用户键盘/鼠标/resize│  │ 真实 shell 进程    │  │ terminal / 其它内容源 │
└───────────────────┘  └──────────────────┘  └──────────────────────┘
```

数据流两个方向，互不绕路：

- **上→下**：`view`（声明界面）/ `result`（调用方法）→ runtime 解算/授权/执行/合成帧。
- **下→上**：`events`（键盘鼠标、终端生命周期、内容源变化、能力握手）→ 程序改状态再产出新 `view`。

## 2. 三层职责表（越界即 bug）

| 层 | 拥有 | 绝不拥有 |
|---|---|---|
| 内核 kernel（盒子模型/解算/命中/合成） | rect、flex、hit、overlay z 序、光标 | 边框/chrome（rect 纯几何）、进程、PTY、业务概念 |
| 运行时 runtime | 协议、路由、授权、帧合成、PTY 连接、尺寸权属、生命周期 | 想不出界面长什么样 |
| 组件 component（builtin，宿主侧） | 一类内容的能力：渲染、输入、方法实现（如 terminal 的滚动/复制） | 决定自己摆在哪、什么时候出现 |
| 布局程序 layout program | 有哪些盒子/组件、几何、焦点、模式（tab/pane/picker 等业务词） | 直接执行有副作用的动作；直接提供 PTY 尺寸数字 |

## 2.5 尺寸：三个概念，三段接力（最容易混，先说清）

| 尺寸 | 谁产生 | 谁消费 |
|---|---|---|
| 视口 viewport（cols×rows） | 用户终端；runtime 从 SIGWINCH 感知 | 布局程序（`resize` 事件后重排，只能响应不能设置） |
| 盒子几何 box rect | **布局程序**（view 的 `size/pos/flex`） | runtime 解算 |
| chrome inset | **组件自己**（如 terminal 声明 inset=1 并自绘边框/标题/角标） | runtime 用它算内容矩形 |
| PTY winsize | **runtime 解算出的内容矩形**（placement rect − 组件 inset） | terminal 组件写入自己 PTY |

```
程序声明盒子几何(策略) → runtime 解算 rect(管道) → rect − 组件声明的 inset → PTY winsize(能力)
```

- 程序**不得**发 `resize(cols,rows)` 这类数字命令；改尺寸的唯一方式是改盒子几何。
- 内核 rect 不含边框：边框/标题/颜色是内容或组件的事，组件通过 inset 把 chrome 占用告诉 runtime。
- 多客户端时一个终端只有一个 winsize：谁"驱动显示"谁是 **resize owner**（CAS；租约 TTL 默认 15s、宿主每 5s 心跳续约，
  见 `PROTOCOL.zh-CN.md` §5）；`attach` 默认 `fit:true`（驱动方成为 owner），镜像方 `fit:false` 只跟随；
  存在**活跃** owner 时 `fit:true` 必须带 `expected_owner_epoch` 做 CAS，缺失/不匹配回 `owner conflict`。
  fit/mirror 是**策略**（程序用方法声明意图），数字仍来自解算结果。

## 2.6 组件间通信：不直连，程序是唯一协调者

```
允许（三条路径）：
  组件A ──event{type:"component",source,name,value}──▶ 宿主 ──▶ 程序    ① 语义事件上报
  程序 ──view 里组件的 props（title/focused/items…）──▶ 宿主 ──▶ 组件B  ② 声明式下发
  程序 ──result{method:"terminal.scroll",…}──▶ 宿主授权 ──▶ 组件B       ③ 命令式调用能力
  组件X ──▶ 宿主共享能力（clipboard/history/daemon 生命周期）           ④ 共享资源只经宿主

禁止：
  组件A ⇄ 组件B 直连（无地址/无句柄/无通道）
  宿主不提供组件内嵌：组件是不透明叶子。
  "嵌套"的合法含义只有两个：在终端里再跑一个复用器（普通进程），或另一客户端 attach 同一终端。
```

- 兄弟组件的数据交换 = **LIFT STATE UP**：谁产生的状态由程序收集（事件），再作为 props 下发给需要的组件。
- 理由：单一授权/审计点、时序确定、组件互不信任、程序崩了宿主仍能重放 sources 恢复。
- 高吞吐数据流（如 A 的输出喂 B）也走程序转发；确有需要时再在宿主加受控 `stream` 原语，不开直连后门。

## 3. 不变量（写成测试，违反就红）

1. **单一真相**：程序侧每个业务对象（pane 等）只有一个模型，不并存两套。
2. **焦点**：一帧内 ≤1 个 `focused` 内容源；索引必须有效；mode/overlay 打开时程序自己让出焦点。
3. **键盘**：Ctrl-Q 永远归宿主；Ctrl-C **仅当 core overlay 打开时**归宿主，否则按未 claim 键处理（可透传 PTY 中断前台进程）。
   聚焦内容源的按键按其 `input`/kind 分派：终端（声明 `key`）的未 claim 键（含 Ctrl 组合）写 PTY；非终端组件（声明 `key`）给组件；
   无 focused 时交给程序（不静默进 PTY）。claim 是 view 的字段，见 `PROTOCOL.zh-CN.md` §6.5。
4. **鼠标**：命中 focused 且终端开启 mouse tracking → 透传 PTY；**终端类盒子（开 mouse tracking）的拖拽一律走 PTY，不参与宿主捕获**；
   宿主捕获只针对非终端且声明 `input:["mouse"]` 的盒子；其余一律给程序命中测试。
4.1 **滚轮**：默认归程序；仅当"focused + 终端开启 mouse tracking + 该面板声明 `wheel`"三者同时满足才透传 PTY。
5. **尺寸权属**：程序只给盒子几何；PTY winsize = placement rect − 组件声明的 inset，由 terminal 组件写入；谁驱动显示谁是 owner
   （CAS + 租约 TTL 15s、宿主每 5s 心跳续约；存在活跃 owner 时 `fit:true` 必须带 `expected_owner_epoch`），镜像方跟随。
6. **z 序**：`pos` 子树 = overlay，合成在所有组件之后；`core.*` overlay 永远最后。
7. **能力边界**：组件只实现"能力"，开关策略在程序；宿主只做"管道"和授权。
   组件之间不得直连，也不得内嵌组件；跨组件协作一律经程序协调（§2.6）。
8. **协议版本**：只加可选字段不升版本；改语义必须升版本 + 迁移说明。

## 4. 老代码里已经验证过的结论（保留，别再踩）

- 盒子模型够用：rect/flex 由宿主解算，程序只声明；rect 是纯几何、不含边框（chrome 内容/组件自绘，
  组件用 inset 声明占用）；回看不是内核能力（协议 §9.5）。
- overlay 必须是显式的 `pos` 子树，z 序写进契约（否则会被组件帧盖住）。
- 终端尺寸只有 resize owner 说得算；`attach` 的默认值必须是驱动方（`fit:true`），默认 follower 会造成"盒子内容区与实际 PTY 尺寸不符"。
- 鼠标透传必须同时满足"聚焦 + 终端开启了 mouse tracking"，否则点击永远进不了程序；终端类盒子的拖拽直接走 PTY，宿主捕获只给非终端盒子。
- core overlay（确认框）必须宿主合成且不可被程序覆盖/隐藏。
- 内容源生命周期（attached/exited）由宿主推给程序，程序只做展示与重启入口。

## 5. 已定案（全部闭环）

**已定案**

1. **组件进程模型**：builtin 组件（terminal/header/footer 等）跑在**宿主进程内**（宿主实现+渲染）；
   插件类组件（如 agents.navigator）是**独立进程**（需要隔离/第三方扩展）。
2. **时间**：布局程序是普通子进程，自己读时钟、自己定时；**v1 不引入宿主 timer 原语**
   （双击窗口=记录两次事件的本地时间戳）。仅当将来沙箱化程序（WASM 等）才加 `timer/tick`。
3. **错误模型（统一请求-应答）**：每个 `result{request_id,epoch,method,params}` 总是收到恰好一次
   `RESPONSE{request_id,epoch,ok,data|error}`（异步、可乱序）；`request_id` 作用域 = (`epoch`, 连接)；
   `ok:true` = 请求已被接受并生效（`terminal.create` 的 data 为 `{endpoint,id}`，其余动作类为空），取数类 data 为行/文本；
   `notice` 只用于非请求类的宿主信息。`sources` 是生命周期与内容源状态的权威，且与 RESPONSE 的到达顺序
   不作保证——程序必须按 source id 幂等对账。

4. **布局程序完全信任、无沙箱**：程序由用户自己编写/运行，对本机有完整权限，自行负责。
   安全内核（保留键/core overlay/授权框）**不是沙箱**：它保证程序卡死/挂掉时全局逃生仍可用，
   并给 destructive 动作一个统一确认点；程序不得绕过，但也不被限制文件/环境/网络访问。
5. **协议承载：二进制帧**（`u32 长度 | u8 类型 | protobuf 载荷`，复用仓库现有 protobuf 工具链）；
   配 `tui2 decode` 把帧解成 JSON 供调试。全量 view 快照为主，后续可选增量（`rev_base`+delta）。
6. **会话与恢复（v1 定案）**：宿主维护 `epoch`（每次启动/重启程序 +1）与 `view_id`（一个客户端连接 = 一个 view）；
   新 epoch 的 HELLO 到达时宿主原子重置缓存/捕获/claim/focus 并关闭 core overlay，同时作废旧 epoch 全部在途请求
   （能回则回 `RESPONSE{ok:false,error:"epoch reset"}`，旧意图不得执行）；程序重启后从 `rev=1` 重发也生效
   （`rev` 作用域 = (`epoch`,`view`)）。崩溃恢复 = 宿主重放 `HELLO+sources`，程序**重建默认布局**，绑定按 source id 幂等对账。
   `state.save/state.load`（宿主按程序 id 持久化不透明 blob，设容量上限）在 `hello.features` 里预留、v1 不实现。
   超限去向按方向区分：宿主→程序帧合并/降级为 notice（不断开）；VIEW 超限被拒并回 `view_rejected`（同一 rev 一次）；
   RESULT 饱和/超限回 `RESPONSE{error:"throttled"|"oversize"}`；仅解码失败才断开并进入重启策略。

## 6. 术语表（四份文档统一）

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
