# TUI 可编程界面模型（Program + 纯展示树）设计稿 v0

> **历史文档（已归档）**：本文描述旧 `tui/` 盒子模型实现；该目录已随 T4 删除，
> 现行 TUI 见 `clients/tui/docs/`。仅作设计背景保留。

状态：与现状并行验证的草案。核心目标是让"界面怎么展示"和"怎么响应"彻底分开：
宿主是通用展示树 reconciler，界面逻辑由外部程序（任意语言）负责。

## 1. 目标

- 宿主不定义 workspace / sidebar / header / footer / keymap 等业务抽象。
- 界面 = 一棵**纯展示 JSON 树**；响应逻辑 = 外部程序。
- 外部程序可以用任意语言、以子进程形式加载；效率靠"程序不在像素热路径 + 共享内存"保证。
- 现有 TUI 退化为"编译进宿主的默认程序"，可被用户程序替换。

非目标（本阶段）：

- 不改 daemon 控制协议。
- 不做 WASM 沙箱（后续可选，接口保持一致）。

## 2. 核心模型

```
        events(IPC)                 view(desired, shm/JSON)
host  ─────────────►  Program  ─────────────►  host
host  ──  globals / effective (shm, 只读广播) ──►  Program / 组件 / 插件
host  ──  content(每 Surface 旁路通道) ───────►  合成器
```

- **Host / Kernel**：布局解算、合成（z-order）、命中测试、事件源、数据面代理、安全内核。
- **Program**：维护状态与交互（快捷键、切页、popup 时机、抽象），产出展示树。
- **Panel**：`terminal`、`surface`（内置面板/插件 Surface）、远端 endpoint 面板，都是**不透明叶子**；
  树只管它们放哪、多大，内部动作/数据面/lease/授权与树无关。

## 3. 展示树（Program 产出，宿主消费）

节点字段：`id / type / style / text / flex / width / height / min / max / visible / children / props`。
原语：`col / row / stack / box / text / float / terminal / surface / list`（list 后续）。
内容叶子：`terminal{target,title,lines?}`、`surface{surface_id,renderer}`。

```json
{ "rev": 12, "root": { "type": "col", "children": [
  { "type": "text", "style": "header", "text": "WS main" },
  { "type": "row", "flex": 1, "children": [
    { "id": "agents", "type": "surface", "width": 30, "props": {"surface_id": "agents.navigator"} },
    { "id": "term",   "type": "terminal", "flex": 1, "props": {"title": "main"} } ]},
  { "type": "text", "style": "footer", "text": "[Ctrl+P] PANE" } ]}}
```

规则：

- 尺寸：固定 `width/height` 优先；`flex` 分摊剩余；未声明尺寸的容器默认填满，文本取固有尺寸。
- `float` 在父 rect 内按固定宽高居中（后续支持锚点）。
- 展示树**不含** `on/action/keymap/scene`。
- 高频内容（terminal cell / 插件自绘）**不进树**，走旁路通道。

## 4. 事件（host → Program）

NDJSON 或 protobuf：

```json
{"type":"key","key":"ctrl-p","char":""}
{"type":"mouse","node":"term","x":12,"y":3,"button":"left"}
{"type":"resize","cols":120,"rows":30}
{"type":"focus","node":"term"}
{"type":"data","node":"term","kind":"screen"}
{"type":"lifecycle","event":"program.ready"}
```

宿主用上一帧布局做命中测试，把事件关联到 `node` id，不解释语义。

## 5. 共享内存（进程外零拷贝；无 shm 能力时退回 IPC）

| 区域 | 写者 | 读者 | 内容 |
| --- | --- | --- | --- |
| `view` | Program | Host | desired 展示树（双缓冲 + 原子 rev + seqlock） |
| `effective` | Host | Program/组件/插件 | 解算后的 `node→rect`、焦点、命中表、viewport、状态 |
| `globals` | Host（只读广播） | 所有 | 主题、能力、endpoint 摘要、时钟等共享上下文 |
| `events`（可选 ring） | Host | 多读者 | 事件日志，避免每读者一条 IPC |

一致性：双缓冲 + 版本号 + 通知（eventfd/futex）；读者校验版本避免撕裂。
`content`（cell 帧）不在共享内存里，走每个 Surface 自己的渲染面通道 + 背压。

## 6. 面板与内容

- `terminal`：绑定某 endpoint/terminal；屏幕订阅、输入发送、resize 归属、lease/epoch 全在面板内部。
- `surface`：内置面板或插件 Surface。`renderer=declarative`（宿主渲染声明式子节点）或 `self`（不透明自绘，走渲染面）。
- **远端 endpoint**：只是一个内置面板实现；树不感知远程、不参与授权。

## 7. 安全内核

不可编程、不进共享内存、Program 不可覆盖：配对/授权确认、破坏性确认、全局逃生/退出。

## 8. 加载与生命周期

- 默认程序：编译进宿主，保证永远可用（= 现有 TUI）。
- 用户程序：`--program <path>` 或配置；子进程优先，WASM 内嵌后续可选，接口一致。
- 崩溃隔离：程序退出时宿主保留最后一棵好树，或回退默认程序。
- 热重载：新程序产出新 view 后切换；旧程序不影响宿主。

## 9. 与现有代码的关系

- 保留并复用：`tui/program`（盒子模型 + host）、`tui/render`（合成 + PanelFrames/PluginFrames 内容源）、`terminalhost`、数据面。
- 已删除（被盒子模型取代的早期探索）：`tui/surface/**`（contract/ipc/manager/slot/echo）、`proto/uipb`、`cmd/anytty-echo-surface`、`cmd/anytty-panel-surface`、`cmd/anytty-surface-harness` 及 `ANYTTY_SURFACE_*` 开关。需要自绘插件协议时再按新模型重新引入。
- `tui/program`：展示树 + 布局解算 + 帧生成（已实现原型）。

## 10. 原型验证（已完成）

- `tui/program`：展示树、布局解算、`col/row/stack/box/text/float/terminal/surface`。
- `cmd/anytty-shell-program`：外部 Go 程序复刻现有 TUI 主要界面与交互（tabs/panes/sidebar/footer、Ctrl-P/R/G、Ctrl-F picker、Ctrl-T/W、Tab、打字）。
- `cmd/anytty-program-harness`：宿主最小闭环（spawn + 事件转发 + 展示树合成）。
- tmux 验证：header/sidebar/panes/footer 渲染；picker float；选择附着改变 pane；sidebar 切换；模式变化；打字追加。全部行为在外部程序，JSON 无响应逻辑。

## 11. 待定决策

1. `--program` 接入真实 TUI 的方式：全新子命令，还是主 TUI 的 opt-in 开关（默认程序不变）。
2. 共享内存优先级：先 `view`，还是 `view + globals` 一起。
3. 事件/视图传输：先 NDJSON（易调试），还是直接上 protobuf + shm。
4. `list` 节点的投影语义（`each/template`）是否本阶段就做。
5. 默认程序的迁移粒度：一次性替换，还是保留旧渲染作为 fallback 一段时间。

## 12. 分阶段计划

- P0：文档 + `tui/program` 原型（已完成）。
- P1：`--program` 接入主 TUI（默认程序不变、可回退）。
- P2：共享内存 `view`（双缓冲 + 通知）；`globals`/`effective`。
- P3：默认程序迁移（现有 TUI → 展示树），`slot` 下放。
- P4：WASM 内嵌（同接口），多语言示例。

## 13. 风险

- 若 props 需要大量计算，会把 JSON 变成一门语言 → 明确划界：树只放数据与简单动作，计算交给程序/宿主 effect。
- 共享内存一致性（seqlock/版本）与多写者冲突 → 单一写者 + 版本提交。
- 程序在热路径（每帧布局）→ 默认不在；需要时用 in-proc/WASM。

## 14. 评审修正：硬伤清单（P0 必须先修）

1. **入站帧尺寸未校验，可 OOM**：`ipc.Server` 收到 Frame 直接按 Surface 给的 `Rect.W/H` 分配。→ 入站即按 `Limits.MaxCols/MaxRows/MaxCells` 校验/拒绝，校验 `len(Cells) <= W*H`、`Revision` 不回退。**已修**。
2. **可靠输入可能被静默丢弃**：写队列满时 `send` 丢消息。→ `Pump` 发送失败把未发出的输入重新入队重试；`send` 成功后记入信用窗口。**已修**。
3. **同 id 重新挂载泄漏/误删**：`Attach/Launch` 覆盖 entry 不关旧实例；旧进程退出按 id 删新 entry。→ 先关旧 entry；`removeByProcess` 按 server 指针匹配；entry 持 ctx，Close 取消 + Interrupt + 超时 Kill。**已修**。
4. **展示树节点 id 不唯一**（原型每个 pane 都是 `"pane"`）→ 强制稳定唯一 id，否则 id diff/命中不可用。**已修（程序侧）**。
5. **展示树无 schema 版本** → `program.View.Version` + `SchemaVersion`。**已修**。

仍待修（P1）：
- 单一布局/命中/焦点权威（见 §15）。
- 安全内核强制落点（见 §17）。
- world state 命名空间/一致根/分片（见 §16）。
- 事件面补齐 `mouse/focus/data/lifecycle`；非法输出上报；崩溃回退默认程序。

## 15. desired / effective 与 focus 三义

- **desired（Program 写）**：展示树（结构/尺寸/可见性意图）。
- **effective（Host 写，只读广播）**：解算后的 `node→rect`、命中表、viewport、焦点、状态。Program 用它做坐标换算与自绘对齐；鼠标/焦点由 Host 按 effective 命中到 node id。
- 布局权威只有一个：**Program 表达意图，Host 唯一解算**；删除 host 侧硬编码 rect。

`focus` 三义必须拆分：

| 名称 | 归属 | 语义 |
| --- | --- | --- |
| `core.focus` | Host 权威 | 终端/面板焦点；高频，放 effective 热区 + 通知，不做可 watch 的持久集合 |
| `ActiveContext` | Host 生成 | UI 交互上下文只读快照（已存在） |
| `FocusedMountID` | Host 内部 | 插件 UI 焦点调度，不对外发布为权威 |

插件聚焦不产生动作（panel 自处理）符合该模型：focus 只是 Host 写的读模型，不授予写权限。

## 16. World State / Collections / Schema

- 三样分开：**view（展示）/ world state（数据）/ events（变化）**。全局数据不进展示树。
- **权威存储**：daemon core + daemon 侧插件状态库；**Host 物化**为本地 read model；Program/插件只读，不直连 daemon 取凭据。
- **信封固定**：`collection / owner / schema / version / revision`；**核心集合固定 schema**（`core.endpoints`、`core.terminals`、`core.focus`、`core.summary`、`core.theme`），**插件集合动态**（`plugin.<id>.<name>`，payload 自描述、opaque）。
- 现有差距（需扩展，不能照搬）：
  - 状态库 key 固定 `pluginId/collection`、只允许自身写 → 扩成 `owner/collection`，保留 `core` 命名空间，ACL 在 **daemon 侧**按注册身份裁决。
  - 无跨集合一致根 → 增加 `world_root_rev` / `committed_at_root`，读侧先读 manifest 再读集合；承认 join 只能最终一致。
  - 未知集合静默返回空 → 改明确 `NOT_FOUND`。
  - 单值 64KiB 上限 → per-collection 配额；`TerminalInfo` 重字段不进 world state。
  - 多 daemon 无全局 revision → 按 `daemon_id` 分片 + provenance；Host 生成物化 join（如 `core.agent_cards`）。
  - resync 不带 generation → 带 `(daemon_id, boot_epoch, world_root_rev)`，generation 变化整体重建该分片。
- schema 策略：信封固定 + 核心固定 + 插件动态；提供 schema descriptor/fingerprint；未知即忽略。

## 17. 安全内核强制点（不可编程）

必须由 Host 在合成与事件路由层强制，Program/树不可覆盖：

1. **保留 overlay 层**：配对/授权确认、破坏性确认由 Host 渲染在最上层，z-order 不能被树内 `float/box` 覆盖。
2. **保留键先拦截**：退出/强制逃生、copy mode、全局导航；Program 不能声明吞掉这些键。
3. **保留 id 命名空间**：`core.*` 由 Host 所有，树里出现保留 id 一律忽略/拒绝。
4. **类型化结果仍走授权**：`contract.Result` 只是数据，真实动作由 Host 经数据面 lease 执行。

## 18. 威胁模型（必须写死）

- 插件/Program 是**受信任的普通 OS 进程**（同 uid），不是沙箱。
- **同 uid 进程可直连本机 daemon 拿 `LocalOwner` 全权**，所以 Host/lease 不是本地能力边界；远程数据面靠凭据保护。
- 因此：不可信 Program **不下发 daemon socket/lease**；只给它类型化意图与 `effective`/`globals` 只读视图；如需真隔离，必须 OS 级沙箱（后续）。
- shm：单写者 + 读者只读映射 + 数据分级；lease/token/远程会话内容绝不入 shm。

## 19. 更新后的落地顺序

1. **P0 修复**（进行中）：帧校验、可靠输入、entry 生命周期、唯一 id、schema 版本。**已落地**。
2. **P1**：desired/effective + 唯一命中；安全内核落代码；事件面补齐；崩溃回退默认程序；`--program` 接主 TUI（默认不变）。
3. **P2**：world state（owner/ACL + world_root_rev + 分片 + 物化 join）；shm（先 view 三缓冲/不可变快照 + 通知，再 effective/globals）。
4. **P3**：默认 TUI 迁移为编译进宿主的默认 Program；`slot` 下放为默认程序内部概念；WASM 可选。

