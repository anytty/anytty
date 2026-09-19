# TUI v2 完整架构图（草案 · 一页看全）

> 配套：`ARCHITECTURE.zh-CN.md`（职责与不变量）、`PROTOCOL.zh-CN.md`（消息与组件契约）、`SCENARIOS.zh-CN.md`（场景与流转）。
> 内核语义唯一依据：`PROTOCOL.zh-CN.md` §9（默认值表/z 序/sources 快照）。
> 术语（四份文档统一）：**盒子 box**=框架唯一图元；**内容源 source**=宿主管理的不透明内容；**组件 component**=一类内容的能力（属性/方法/事件）；
> **slot/workspace/tab**=程序侧业务对象（宿主不认识）；**view**=一个客户端连接（宿主在 HELLO 分配 `view_id`）；**epoch**=宿主每次启动/重启程序 +1；
> **rev**=程序内视图版本（作用域 = `epoch`×`view`）；**claim**=VIEW 内的按键路由声明。完整表见文末附录。

## 图 1 · 全景：进程、职责与数据流

```
        ┌──────────────────────── 用户终端（TTY/emulator）────────────────────────┐
        │  键盘 · 鼠标 · 视口 resize                          屏幕（ANSI 帧）        │
        └───────────────▲────────────────────────────────────────────┬───────────┘
                        │ frame                                        │ bytes
┌───────────────────────┴────────────────────────────────────────────▼───────────┐
│ 宿主 runtime（唯一可信方）                                                        │
│   ① 输入解析：字节 → 规范化事件(key/paste/mouse/wheel/resize)                     │
│   ② 安全内核：保留键 · core overlay · destructive 授权 · core.* 命名             │
│   ③ 路由：claim 表 + 焦点  →  程序 或 组件（图 3）                                │
│   ④ 布局解算：盒子 → rect/flex/hit/overlay（纯函数）                              │
│   ⑤ 帧合成：程序帧 + 组件帧 + overlay 帧（z 序，图 5）                            │
│   ⑥ 连接：终端池 数据面、组件进程、协议连接                                        │
└───────┬────────────────────────────────────────────────▲─────────┬──────────────┘
        │ view / result（二进制帧）                       │ events   │ 数据面调用
        ▼                                                 │          ▼
┌──────────────────────────────┐                          │  ┌──────────────────────────┐
│ 布局程序（子进程，可重启）      │──────────────────────────┘  │ 终端池（终端生命的 owner） │
│  状态机：workspace/tab/slot/  │                             │  · PTY 与进程生命周期      │
│  focus/mode/overlay/ratio     │                             │  · history（回看窗口）     │
│  只做：声明 + 方法调用          │                             │  · clipboard（OSC52 出口） │
│  禁止：PTY/剪贴板/终端池/尺寸数字│                             │  · attachment/尺寸权属     │
└──────────────────────────────┘                             └────────────┬─────────────┘
                                                                           ▼
                                                              ┌────────────────────────┐
                                                              │ 真实终端进程 zsh/vim/…   │
                                                              └────────────────────────┘
```

**要点**：程序崩了宿主能重启它（重放 hello/sources）；终端池 崩了终端才真的死；宿主永远不认"tab/pane/picker"。

## 图 2 · 消息面（五种帧类型）

```
帧格式：二进制帧 u32 长度 | u8 类型 | protobuf 载荷（stdin/stdout，stderr=日志；decode 工具可读）
类型编号：1=HELLO(宿主→程序) 2=VIEW(程序→宿主) 3=EVENT(宿主→程序) 4=RESULT(程序→宿主) 5=RESPONSE(宿主→程序)

宿主 ──1 HELLO{view_id,epoch,schema,视口,组件,方法,features,limits(max_inflight_requests/owner_lease_ttl_ms/…)}──▶ 程序
程序 ──2 VIEW{epoch,rev,keys{claim,all},root}──▶ 宿主 ──解算/合成──▶ 屏幕（超限则回 view_rejected）
程序 ──4 RESULT{request_id,epoch,method,params}──▶ 宿主 ──授权──▶ 终端池/组件 执行副作用
宿主 ──5 RESPONSE{request_id,epoch,ok,data|error}──▶ 程序
宿主 ──3 EVENT(sources/key/paste/mouse/wheel/resize/notice/component/view_rejected)──▶ 程序
```

- 全量快照：每次状态变化发整棵 view（带 epoch+rev；rev 作用域 = epoch×view），宿主据此丢弃过期帧；程序重启后 rev 可回退，新 epoch 首帧必须生效。
- 解析顺序：长度校验（0/超限拒帧）→ 分配 → 解码；方向错误按 decode error 断开程序。
- claim 在 VIEW 里，与视图同帧原子生效；没有独立 KEYS 帧。
- 副作用唯一入口是 result，且每个 result 恰好回一次 `RESPONSE{request_id,epoch}`（ok:true=已生效；sources 是状态权威，顺序不作保证，按 source id 幂等对账）；没有隐式通道。
- 超限：宿主→程序帧降级 notice；VIEW 超限回 `view_rejected`（同一 rev 一次）；RESULT 饱和/超限回 `throttled`/`oversize`。

## 图 3 · 输入路由流水线（全局唯一优先级）

```
bytes → 解析
  │
  ├─① Ctrl-Q 总是保留；Ctrl-C 仅 core overlay 打开时保留 ──→ 宿主
  ├─② core overlay 打开? ────────────────是→ 宿主独占（程序收不到任何输入）
  ├─③ VIEW.keys.all=true? ───────────────是→ 程序
  ├─④ 命中 VIEW.keys.claim? ─────────────是→ 程序（业务快捷键）
  ├─⑤ 存在 focused 内容源?
  │      ├按键→ 按 input/kind：终端未 claim 键（含 Ctrl 组合）→ PTY；非终端组件 → 组件
  │      ├paste→ focused 终端（声明 paste）→ 宿主 bracket/CRLF/分块后写 PTY；否则给程序
  │      ├鼠标→ 仅"命中 focused 且终端开 mouse tracking"才透传；终端类盒子拖拽一律走 PTY
  │      └滚轮→ 默认给程序；仅"focused + 终端开 mouse tracking + 面板声明 wheel"三者齐备才透传 PTY
  └─⑥ 无 focused 或以上都不满足──────────────→ 程序（必须处理或提示，禁止静默丢弃）

冲突用原语解决：事件带 id；input.forward{event_id,source} 把某次 key/paste 事件退进终端（双击透传/前缀+字面键）。
```

## 图 4 · 组件模型（像写 web 一样用组件）

```
┌ terminal（builtin 组件）────────────────────────────────────────────┐
│ 属性（view 盒子字段）：content.self · content.props ·               │
│                        focused · input[key/paste/mouse/wheel] ·     │
│                        size/pos；props = chrome.* 显式样式          │
│ 方法（result）：attach · create · restart · kill · remove · scroll · │
│                scrollEnd · copy · history.window                   │
│ 事件（宿主推）：sources(attached/exited/health/owner) · key/paste/  │
│                mouse/wheel · component · notice                    │
└────────────────────────────────────────────────────────────────────┘
分工：能力=组件 · 策略=程序 · 管道=宿主
决策：需要特权(PTY/剪贴板/终端池)/独立进程/独占资源 → 做组件；
      纯展示与选择策略(picker/help/prompt) → 程序侧 SDK 库，不进协议。
```

## 图 4.5 · 组件间通信（无直连）

```
┌组件A┐ ──①语义事件{component,source,name,value}─▶ 宿主 ──▶ ┌程序┐（策略/协调者，v1 可用）
                                                        │
        ┌组件B┐ ◀──②声明式 props（view 盒子字段）────────┤
        └组件B┘ ◀──③命令式 result{method}─── 宿主授权 ────┘
   组件X ──▶ ④宿主共享能力（clipboard/history/终端池）     ✘ 组件A⇄组件B 直连（不存在）
                                                          ✘ 组件内嵌组件（宿主不支持；"嵌套"=终端内再跑复用器，或另一客户端 attach 同一终端）
```

兄弟组件交换数据 = LIFT STATE UP（组件上报事件 → 程序存状态 → 作为 props 下发）。
理由：单一授权/审计点、时序确定、组件互不信任。

## 图 5 · z 序与尺寸（两个最容易踩的静态契约）

```
z 序（从下往上合成）                     尺寸（三段接力）
┌ 程序常规帧（边框/文字/侧栏）            视口 cols×rows ──设备→宿主→程序（resize 事件）
├ 组件帧（terminal 的 live 画面）              程序：声明盒子几何 size/flex/pos
├ 程序 overlay 帧（pos 子树：picker/prompt）    宿主：解算内容矩形 = placement − 组件 inset
└ core overlay 帧（宿主确认框，永不被动摇）     组件：terminal 写 PTY winsize
                                          权属：驱动方=resize owner（attach 默认 fit:true，活跃 owner 在时需 expected_owner_epoch 做 CAS；租约 TTL 15s、每 5s 心跳续约）；镜像方 fit:false 跟随
```

## 图 6 · 生命周期归属（两种语义只是一个 switch）

```
终端池 拥有终端生命：  create → running → exited → kill/remove
程序   拥有 slot 生命： split → 空槽 → attach(绑定) → 解绑/关闭
                    ┌──────────────┴──────────────┐
        tmux 式（同生共死）              anytty 式（完全解绑）
   split→create 并绑定                    split→空槽，picker 绑定已有终端
   关槽→kill(+remove)                     关槽→只解绑，终端继续跑
   exited→关槽(可 remain-on-exit)         exited→保留角标 + Ctrl-E restart
   退出→显式 cleanup_owned:true 或用户确认才清算（崩溃/重启不清算）   退出→纯 detach
细节：create 回 {endpoint,id} 且已 attach+fit；在途 create/attach 返回按 source id 幂等对账；kill/remove 走宿主授权，程序不能绕开。
```

## 图 6.5 · 远程 endpoint（位置透明）

```
程序 ──result{request_id,method:"terminal.attach",params:{endpoint:"remote-1",id,fit:true}}──▶ 本地宿主
                                                         ├─ 连接管理器（鉴权/超时/退避重连/重订阅）
                                                         ├─ 归一化：远程事件 → 本地事件流
                                                         │    lifecycle → sources(health/resize_owner/owner_epoch)
                                                         │    屏幕 revision → 组件内容源
                                                         │    错误/超时 → notice + health
                                                         └─ 每个 result 恰好回一次 RESPONSE{request_id,epoch,ok,data|error}
程序可见：sources/notice/RESPONSE ——与本地终端完全同一套；不得按 endpoint 特判
```

## 图 7 · 程序状态机（业务全在程序）

```
CONNECT ──hello/sources──▶ 有终端? 自动绑定 → NORMAL
                          └─无终端──▶ EMPTY + PICKER ──enter/create──▶ NORMAL

NORMAL（终端持有键盘）
  Ctrl-P → PANE（清 focused；claim all=true）
             ├─ % / "   分屏 → 空槽（可点/可 Tab/可 picker 绑定）
             ├─ x       关当前槽（解绑或 kill，按图 6 策略）
             ├─ Tab     切槽       ├─ Ctrl-F  PICKER
             ├─ :       PROMPT     ├─ ?       HELP
             └─ esc     NORMAL
NORMAL/PANE ──滚轮/PgUp──▶ SCROLL（[↑N]；scroll→RESPONSE.data）──y→copy──esc→scrollEnd──▶ 原态
任意态 ──sources(exited)──▶ 该槽显示 [exited N]（Ctrl-E restart）
任意态 ──宿主判定 destructive──▶ CORE CONFIRM（宿主独占）──enter/esc──▶ 回原态
```

## 图 8 · 一次完整时序：分屏 → 绑定 → 输入 → 关闭

```
用户            程序                  宿主                     终端池/PTY
 │ Ctrl-P, %     │                     │                          │
 ├──────────────▶│ 追加空槽, focus=新槽  │                          │
 │               ├─view{epoch,rev,keys}▶│ 解算/合成                 │
 │ Ctrl-F        │                     │                          │
 ├──────────────▶│ overlay=picker, 清focused (keys.all=true)        │
  ├─↓/enter──────▶│                     │                          │
  │               ├─result{request_id,epoch,terminal.attach,fit:true}──▶ 授权(免)
  │               │                     ├────────attach───────────▶│
  │               │◀──RESPONSE{request_id,epoch,ok}+sources(attached)──┤
 │               ├─view(该槽绑定,focused)─▶│ 组件帧+overlay 合成      │
 │ 键入 "ls⏎"    │                     │                          │
 ├──────────────▶(未 claim 的键) ──原始字节──────────────────────▶│
 │               │                     │                          │
 │ Ctrl-F ×2     │                     │                          │
 ├──────────────▶│ input.forward{event_id,source}──▶ 重编码写 PTY   │
 │ 滚轮回看       │                     │                          │
 ├──────────────▶│ result{terminal.scroll,delta}──▶ RESPONSE{data.rows} │
 │ x（关槽）      │                     │                          │
 ├──────────────▶│ 解绑 / 或 result{kill/remove}(授权) 按策略选择     │
```

## 图 9 · 验收清单（黑盒，tmux 驱动）

1. 冷启动与绑定：无终端自动 picker；attach 后 1s 内可输入。
2. 分屏与焦点：`%`/`"` 新槽可点/可 Tab/可绑定；`esc` 退出 PANE 后输入进 PTY。
3. 透传：`Ctrl-F` 双击后终端里 `cat -v` 收到 `^F`。
4. 回看/复制：`[↑N]`、`y` 后剪贴板可验证、`esc` 回 live。
5. 授权/退出：`kill`/`quit` 弹宿主确认；`esc` 拒绝仍可用。
6. 生命周期：tmux 式关槽终端消失；anytty 式关槽终端仍在 picker、可重绑。
7. 崩溃：kill 布局程序 → 画面保留、自动重启、重绑后可用；确认框打开时 kill → 新 epoch 后确认框消失、旧意图不执行。
8. 尺寸：PTY 与槽内容区一致；拖拽分隔条后两槽与 PTY 同步变化。
9. 多客户端：一方关闭或 SIGKILL 后另一方在 TTL（15s，宿主每 5s 续约）内 CAS 接管尺寸，内容不互踩；活跃 owner 时不带 `expected_owner_epoch` 的 `fit:true` 回 `owner conflict`。
10. epoch/重启：程序重启后 `rev=1` 首帧生效；旧 claim/focus/捕获已清空；旧 epoch 在途 RESULT 收到 `epoch reset`。
11. 注册表一致：`HELLO.methods` 与 `PROTOCOL.zh-CN.md` §4 方法表一致；未知方法回 `RESPONSE{ok:false,error}`。
12. 背压：RESULT 饱和 → `throttled`（不排队）、单帧超限 → `oversize`；VIEW 超限 → `view_rejected` 且同一 rev 只回一次；宿主→程序帧超限降级 notice 不断开；非法 protobuf 才断开并进入重启策略。
13. 基准：10k 行/s 输出 + 60fps 拖拽 + 500 节点树，记录帧大小与 p99 延迟；饱和压力（程序刷 RESULT / 滚轮风暴 / 大输出）下 Ctrl-Q 的 p99 延迟须有上界（不被业务流量淹没）。
14. create/ephemeral：`terminal.create` 的 RESPONSE 直接带 `{endpoint,id}` 且已 attach+fit，可立即绑定；`create{ephemeral:true}` 后 `kill -9` 程序 → 终端仍在，显式 `cleanup_owned:true` 或用户确认才清算。
15. paste：多行粘贴不误执行（bracket 生效）；超长粘贴按 `max_paste_bytes` 分块且顺序正确、每块独立 event_id；截断有可观测告知。
16. RESPONSE/sources 顺序：制造 sources 先于/晚于 RESPONSE 到达两种顺序，程序行为一致（按 source id 幂等对账）。

## 附录 · 术语表（四份文档统一）

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
