# TUI Surface 重构设计稿 v0

> **历史文档（已归档）**：本文描述旧 `tui/` 盒子模型实现；该目录已随 T4 删除，
> 现行 TUI 见 `clients/tui/docs/`。仅作设计背景保留。

状态：草案。范围限定在 TUI 层，不改动守护进程（daemon）。daemon 继续只提供现有 protobuf 控制接口。

## 1. 目标

把 TUI 的 UI 绘制与扩展从"宿主固定组件 + 服务端渲染整棵树"改成一个通用的 **Surface（面板/控件）模型**：

- 所有可见内容都是一个 Surface：终端 pane、terminal picker、新建 terminal 对话框、工作台侧栏、状态栏、命令面板，全部统一。
- Surface 内部画什么、怎么交互，宿主不关心；宿主只关心它的**状态**和最基本的**事件传递**（键盘、鼠标、resize、焦点）。
- Surface 可以在宿主进程内实现（内建），也可以在插件进程内实现（第三方扩展）。
- 用户不喜欢某个内建 UI（例如 terminal picker），可以挂一个自己的 Surface 替换它。

非目标（本阶段明确不做）：

- 不改 daemon 的任何行为、协议或代码。
- 不做客户端侧沙箱执行模型（WASM/JS）；本阶段采用进程外自绘。
- 不重写数据/终端/输入等与绘制无关的部分（尽量复用，见 §11）。

## 2. 核心模型

一句话：**daemon 是 producer，TUI 和插件都是 consumer，宿主是 compositor + router。**

| 角色 | 归属 | 职责 |
| --- | --- | --- |
| producer | daemon（含 daemon 侧插件） | 持有真相/权威状态，只经 protobuf 暴露读与写 |
| consumer | TUI 内建 pane、TUI 侧插件 | 订阅数据/事件，接收输入，产出绘制，发出命令 |
| host / compositor | TUI 进程 | 布局、焦点、z-order、合成、全局输入、剪贴板/IME、背压、安全边界 |

同一个插件包可以同时有 producer（daemon 侧，持有状态）和 consumer（TUI 侧，只订阅+渲染+发命令），但两者职责不混。

## 3. 两条平面

必须把"数据/命令"和"像素/输入"分开，它们是正交的。

| | 数据 / 命令面（现有，不改） | 渲染 / 输入面（新增） |
| --- | --- | --- |
| 通道 | stdio → SDK → **daemon** | TUI ↔ Surface 的**本地** IPC |
| 内容 | 注册、state read/write/watch、typed query/command、生命周期、挂载元数据 | rect、帧/差量、焦点、键盘鼠标、resize、cursor |
| 频率 | 低频 | 高频（目标 60fps） |
| 经过 daemon | 是 | **否** |
| 远程 | 支持（每个 endpoint 独立路由域） | 不适用（本地） |

规则：

> **数据和命令必须经过该资源所属 endpoint 的 daemon（本地或远程）；像素和输入只在本机 TUI ↔ Surface 之间。**

渲染面不能塞进控制协议：60fps 的帧既慢又会强制改 daemon。数据面不能绕过 daemon：否则丢失远程授权、租约与 epoch。

### 3.1 远程 endpoint 的数据流（保持现状）

以远程 Agent 为例：

1. 远程机器上的 Agent hook → 本地 socket → **远程 daemon**
2. 远程 daemon → 转发给**远程机器上安装的 daemon 侧插件**（该机器上的状态 owner）
3. daemon 侧插件写状态并广播（快照 + watch）
4. 本机 TUI 到**远程 daemon 有自己的已认证连接** → TUI 侧 consumer 订阅到变化
5. consumer 聚合各 endpoint 数据，渲染走**本机渲染面**

本地 daemon 不参与远程事件的汇聚，也不做 daemon 间中继。授权/租约绑定"到该 daemon 的认证连接"，所以这一步不能省。

### 3.2 consumer 到各 endpoint 的中继

本机 TUI 是 consumer 到各 endpoint 的中继：consumer 使用宿主发放的 lease，不自己拿凭据去连远程 daemon。保持该设计，避免把 client runtime 复制进每个 Surface。

## 4. Surface 契约

宿主只定义下面这几类消息，不认内部语义。

- **Attach / Detach**：`surface_id`、实例 id、初始 rect、一次性 token
- **Resize(rect)**：宿主拥有布局（workspace/tab/pane/floating/overlay 树沿用），Surface 只收最终矩形
- **DrawRequest / Frame | FrameDiff**：Surface 输出 cell grid 或差量
- **Input**：聚焦 Surface 收到原始事件，坐标相对自身
- **Focus(bool)**、**Visibility**、**Close**
- **Cursor{cell, shape}**（IME 需要）
- **Title**
- **RequestRedraw**（Surface 主动）

宿主侧接口（TUI 内，示意图）：

```go
type Surface interface {
    Resize(rect Rect)
    Draw(baseRevision uint64) Frame      // 返回差量；宿主负责合成与落屏
    Input(ev InputEvent)
    Focus(focused bool)
    Cursor() (cell Cell, ok bool)
    Close()
}
```

内建 Surface 直接实现该接口；插件 Surface 由本地 IPC 绑定到同样的接口。

## 5. Slot / contribution 模型

"用户可替换 terminal picker"要成立，光有 Surface 不够，还需要 contribution point。

- 宿主定义稳定槽位：`terminal.picker`、`terminal.create`、`workspace.sidebar`、`panel.header`、`statusbar` 等。
- 每个 slot = **默认实现 + 优先级 + 用户覆盖 + 冲突诊断 + 崩溃回退到默认**。
- 替换的是**交互 UX**，不是权限：Surface 必须返回**类型化结果**（例如"用户选择了 endpoint X / terminal Y"），真正的动作仍由宿主经数据面授权执行。Surface 永远不能自己创建/连接终端。

三类 Surface：

| 类别 | 例子 | 可否被插件覆盖 |
| --- | --- | --- |
| core-owned | 安全/配对/授权确认、copy mode、全局导航与快捷键、错误与破坏性操作确认 | 否 |
| overridable slot（有默认实现） | terminal picker、new-terminal、clipboard history、workbench navigator | 是 |
| plugin-only | 第三方侧栏、工具面板 | 本来就是 |

core-owned 是安全红线：不允许插件伪装成安全对话框或全局导航。

## 6. 绘制模型

- `Cell = {grapheme, fg, bg, attrs}`；`FrameDiff{base, next, dirty spans}`。
- 宿主全局双缓冲，把多个 Surface 按 z-order 合成，再与上一屏做一次全局 diff 落屏。
- Surface 永远接触不到终端转义序列；不可能绘制越权控制码。
- 宽字符/组合字由框架统一算宽度（runewidth/uniseg），Surface 不处理。
- 命中测试只在 **Surface 粒度**；Surface 内部命中自己处理。现有的按 node hit-region 机制随之退役。

## 7. 输入模型

- 全局快捷键、copy mode、overlay 捕获由宿主先拦截；剩余原始事件转发给聚焦 Surface。
- 事件：Key（code + mods + 可打印文本）、Mouse（相对坐标、按键、拖拽态）、Wheel、Paste（bracketed）、FocusIn/Out、Resize。
- Surface 声明 `input_mode`（`text` / `raw`），宿主据此决定是否吞掉可打印键。
- 输入是事件流，与数据流正交；不产生 RPC 往返。

## 8. 背压与流控（框架负责）

原则：**Surface 再慢也不能拖垮 TUI；控制帧永不丢，数据帧可合并。**

1. **每 Surface 信用窗口**：对"未确认帧字节数"和"未完成消息数"设上限，Surface 不得超额。
2. **单帧在飞 + 合并**：同一 Surface 同时只允许一个未 ack 帧；期间新绘制只保留最新（丢中间帧）；差量合并 dirty 区域。没有"每按键一次 RPC"。
3. **输入批量 + 有界队列**：按 tick 批量发送；队列满时鼠标移动/滚动可丢，**key/paste/resize/close 绝不丢**。
4. **时间片**：每帧对解码/应用 Surface 帧设总时间预算，超时顺延下一 tick，慢 Surface 不卡渲染循环。
5. **无响应降级**：超过 deadline 未 ack → 标 `unresponsive`、显示陈旧占位、暂停发输入；恢复后请求一次全量帧。
6. **resize 合并**：一 tick 内只发最终 rect。
7. **资源上限**：每 Surface 与全局的 cell 缓冲、fps、帧大小都有硬上限；超限降级为占位符而非 OOM。
8. 传输用非阻塞 unix socket + 每 Surface 单读单写 goroutine，写侧有界环形缓冲，溢出按上述策略处理。

## 9. 安全

- 渲染面：本地 unix socket + 每次启动一次性 token + peer uid 校验。
- 帧里只有 cell，宿主不执行 Surface 输出的任何控制序列。
- 边界：max rect、max cells、max fps、max frame size；超限降级。
- 插件仍是受信任的普通 OS 进程（capability 不是 OS 沙箱），与现状一致。

## 10. 与现有声明式 UI 共存

- 声明式 UI = **一种宿主渲染的 Surface**：宿主把现有 node tree 渲进同一个 cell grid，把输入翻译成类型化 interaction 事件，再走控制面。
- 自绘 Surface = 插件进程实现，走渲染面。
- 插件作者可选：声明式（便宜）或自绘（强大）。旧插件零改动。

## 11. 现有 TUI 代码复用评估（粗估）

TUI 约 12 万行（含测试）。非测试分布：`tui/app` 约 19.6k、`tui/render` 约 17.3k、`tui/state` 约 15.2k。

| 分类 | 内容 | 占比（粗估） |
| --- | --- | --- |
| 直接复用 | `state`（history/live/terminal_view/shell_*/endpoint_store）、`input`、`terminalhost`、`config`、`port`、`shortcut`+`action`、各 adapter | ~65–75% |
| 改造后复用 | `render` 的 canvas/vm/style/layout → 新合成器后端；`product_content_*`（picker/manager/navigator）→ core-owned Surface；shell/panel/header chrome | ~15–20% |
| 重写/新增 | `render/plugin.go`、`layout_hit_regions`、`app/plugin.go`、插件输入/事件 glue；以及新的 Surface 接口、渲染面 IPC、背压器 | ~10% |

结论：数据模型、输入解析、终端宿主、配置基本全留；被替换的主要是"插件那套宿主渲染 + 命中"，外加新写合成器与渲染面。

## 12. 迁移阶段

- **P0**：TUI 内引入 `Surface` 接口 + 本地 IPC 传输 + 背压器；用回显 Surface 压测，不接真实插件。
- **P1**：接一个自绘 demo（例如 herdr 侧栏），验证延迟、隔离、降级。
- **P2**：加入 slot/contribution + typed result + 覆盖/回退；把声明式渲染改造成宿主渲染的 Surface（控制面协议不变）。
- **P3**：把内建逐个迁成 slot 默认 Surface（terminal picker、new-terminal、navigator 等）；terminal picker 借新渲染层强化。

每阶段可独立测试、可回滚。

## 13. 北极星指标

- 聚焦 Surface 的按键到屏幕更新 **P99 < 16ms**。
- 单个 Surface 卡死不影响其他 pane。
- 接入一个新的自绘 Surface 代码量 **< 100 行**。

## 14. 开放问题

1. 渲染面 schema 放哪：新建 TUI 专属 proto（如 `proto/uipb`，daemon 不引用），还是复用 `apipb`？倾向新建。
2. 传输形态：新开 unix socket（推荐，stdio 留给控制面），还是复用一个额外 fd（`ExtraFiles`）？
3. 帧格式：cell grid vs styled runs；宽字符处理库选型。
4. 内建 Surface 是否也走同一接口（建议是），以便内建与插件同构。
5. 默认 fps 上限与各资源上限取值。
