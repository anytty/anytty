# TUI SDK 使用指南（写给写插件/布局程序的人）

> 这份文档是"怎么用"，不是"协议规范"。协议细节见
> `clients/tui/docs/PROTOCOL.zh-CN.md`，SDK 分层 API 见
> `clients/tui/docs/SDK.zh-CN.md`（组件模型见本文 §4），半小时教程见
> `clients/tui/docs/TUTORIAL.zh-CN.md`。
> 一句话：**你写一个普通程序，负责画界面（和用 access 能力）；tui2 负责
> 终端渲染、输入与连接**。managed 终端的 PTY 真值在 pool（access→pool），
> 只有 `kind: command` 端点的 argv 在宿主本地跑。

## 0. 你能做什么

- **换掉整个界面**：写自己的布局程序（shell），决定页面、tab、侧边栏、浮窗。
- **做一个侧边栏/面板**：读终端列表和状态，自己画。
- **做一个数据插件**：通过 `access.call` 用文件、storage、配对等能力。
- **订阅事件**：终端创建/退出等事件实时推到你的程序里。
- 分发不需要你的源码：Go 是单个可执行文件；Python/TS 需要随程序带上/安装 SDK
  （详见 §7）。

## 1. 五分钟跑起来

```bash
# 1) 构建宿主和默认布局程序（或直接用 release 归档里的三件套）
export PATH="$HOME/.local/share/go-toolchains/go1.26.7/bin:$PATH"
go build -o /tmp/tui2 ./clients/tui/cmd/tui2

# 2) 从一个模板开始（Go / Python / TS 三选一；Go 目标放在本仓库 module 内）
go build -o /tmp/my-shell-bin ./clients/tui/templates/go
#   Python/TS 可以复制到任意目录：Python 设 TUI2_PYTHON_SDK 指向 SDK；
#   TS 改一下 require 的相对/绝对路径（或把 clients/tui/sdk/ts 一起复制过去）

# 3) 开发循环：自动构建宿主、热重载、帧日志、崩溃提示
bash clients/tui/scripts/dev.sh clients/tui/templates/go
#   或指定语言：bash clients/tui/scripts/dev.sh python
#   或加宿主参数（不要加 --，dev.sh 直接转发）：bash clients/tui/scripts/dev.sh python -routes local-unix
```

装到 `anytty` 里（让默认入口用你的 shell）：

```bash
TUI2_SHELL=/tmp/my-shell-bin anytty
# 注意：anytty 默认入口会先把本地栈拉起来（pool+access），再启动你的程序。
# 前提：tui2 能在 TUI2_BIN、anytty 同目录或 PATH 找到（release 归档就是同目录布局）。
```

也可以直接指定宿主：`/tmp/tui2 -shell "/tmp/my-shell-bin"`。

**完整示例**：`clients/tui/examples/herdr/`（自绘终端管理器：侧栏列表 + 终端面板 +
分屏/帮助/toast，演示 `app.Memo`+`sdk.Raw`、Keymap/SetKeys 焦点切换、`access.call`
storage pin、以及无栈时的降级）。一键构建 + 全套测试 + 直接运行：
`bash clients/tui/scripts/quickstart.sh --run`（`--fast` 只跑 Go+conformance，
`--stop` 清理隔离栈）。

## 2. 心智模型（一图）

```
你的程序（布局程序）
  │  VIEW：声明界面上有什么盒子、谁有焦点      ▲  EVENT：键鼠、终端列表、notice
  ▼                                          │
tui2 宿主（合成器）
  │  解算盒子 → 画到屏幕；把按键送到对应盒子   │
  ▼
终端设备 + 内建 terminal 组件（内容源）
```

- **你的程序拥有页面**：tab、侧边栏、picker、浮窗都是你自己画的普通盒子/文本。
- **宿主拥有渲染和终端**：它不认识 "tab" 这个词；它只看到盒子、焦点、按键。
- **终端是"内容源"（source）**：每个终端在宿主里有一个 id（如
  `terminal:local:t1`）；你把一个盒子绑到 source 上，宿主就帮你把 PTY 画进去。

## 3. 最小程序（Go，逐段讲）

```go
package main

import (
    "os"
    "github.com/anytty/anytty/clients/tui/sdk"
    pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func main() {
    var client *sdk.Client
    client = sdk.New(os.Stdin, os.Stdout, sdk.Handlers{
        // 1) 收到 HELLO：拿到视口大小，画第一帧
        Hello: func(h *pb.Hello) { _ = client.Commit(view(), sdk.Keys{Claim: []string{"ctrl-q"}}) },
        // 2) 终端列表/状态变化（source 增删、exited、health）
        Sources: func(items []*pb.Source) { _ = client.Commit(view(), sdk.Keys{}) },
        // 3) 按键（先被 claim 命中，或按焦点规则投递）
        Key: func(k *pb.KeyEvent) { /* ... 改状态再 Commit ... */ },
    })
    _ = client.Loop() // 读到 EOF（宿主退出）返回
}

func view() *pb.Box {
    return sdk.Col(
        sdk.Text(" my tui ").Style("tab_active"),
        sdk.Terminal("terminal:local:t1").Flex(1).Focused(true).Input("key", "paste", "wheel"),
        sdk.Text(" Ctrl-Q quit ").Style("muted"),
    ).Build()
}
```

要点：
- `Commit(root, keys)` 每次提交**完整界面**；收到事件改状态后再 Commit。
- `Claim` 是"这些键归我处理"；`sdk.Keys{All: true}` 表示全要（常用于 picker/输入框）。
- 盒子只是几何声明（`Flex/Width/Height/Pos`），不用写坐标。
- 程序变复杂就换 §4 的组件模型：同样的 VIEW/EVENT，状态与提交交给框架。

## 4. 组件模型（`sdk/app`，推荐写法）

手写 `Handlers` 也能用；但大多数程序要的是同一套循环：收事件 → 改状态 →
重画 → 提交，并且**一批事件只提交一帧**。`sdk/app` 就是这套循环，外加键位
映射和焦点栈。

```go
package main

import (
    "errors"
    "fmt"
    "os"

    "github.com/anytty/anytty/clients/tui/sdk"
    "github.com/anytty/anytty/clients/tui/sdk/app"
    "github.com/anytty/anytty/clients/tui/sdk/widgets"
    pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type model struct {
    items []string
    sel   int
}

func (m *model) Init() app.Cmd { return app.None } // 首次 HELLO / 宿主重连后各跑一次

func (m *model) Update(msg app.Msg) app.Cmd {
    switch v := msg.(type) {
    case app.SourcesMsg:
        m.items = m.items[:0]
        for _, s := range v.Items {
            if s.GetKind() == "terminal" {
                m.items = append(m.items, "terminal:"+s.GetEndpoint()+":"+s.GetTerminalId())
            }
        }
    case app.KeyMsg:
        switch v.Key.GetKey() {
        case "up", "k":
            if m.sel > 0 {
                m.sel--
            }
        case "down", "j":
            if m.sel+1 < len(m.items) {
                m.sel++
            }
        case "ctrl-q", "esc":
            return app.Quit()
        }
    }
    return app.None
}

func (m *model) View() *pb.Box {
    list := widgets.List{Items: m.items, Height: 8, Selected: m.sel}
    return sdk.Col(
        sdk.Text(" terminals ").Style("tab_active"),
        list.Build(),
        sdk.Text(" j/k 移动  ctrl-q 退出 ").Style("muted"),
    ).Build()
}

func main() {
    client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{}) // Run 会接管 handlers
    err := app.Run(client, &model{}) // 不要自己再调 client.Loop()
    if err != nil && !errors.Is(err, app.ErrQuit) {
        fmt.Fprintln(os.Stderr, "my-shell:", err)
        os.Exit(1)
    }
}
```

要点：
- **`Update` 不碰 Commit**：`Run` 在每个批次末尾用 `View()` 提交一次；N 个事件
  到达也只提交一帧。实现 `Dirty() bool`（`app.Cleaner`）可在状态没变时跳过提交；
  HELLO、`SetKeys` 批次会强制提交。
- **`Cmd` 是异步副作用**：`app.Tick`（本地计时器，宿主没有 timer）、
  `app.Emit`（包 `access.call`，把响应解成 Msg）、`app.SendStream`、
  `app.SetKeys`、`app.Batch`。Cmd 在 goroutine 里跑，返回值作为 Msg 回到
  `Update`。
- **键位映射**：`app.NewKeymap(app.Binding{Key: "ctrl-q", Msg: yourMsg{}})` 绑定到
  你自己的消息类型，在 `Update` 里 `return app.Quit()`；带 `Context` 的绑定可用
  `SetContext` 切换（弹窗/输入模式）。不配 Keymap 时所有按键以 `KeyMsg` 直达
  `Update`。
- **焦点栈**：`app.Focus` 决定 `ComponentMsg` 投给谁（如当前输入框），
  自己 `Push/Pop` 维护。
- **重连**：宿主重启会换 epoch：`Run` 恢复 Keys 并清空焦点、调 `Reset(epoch)`
  （可选 `app.Resetter`）、重跑 `Init`；旧 epoch 的 Cmd 结果被丢弃。
- 消息类型：`HelloMsg/SourcesMsg/KeyMsg/PasteMsg/MouseMsg/WheelMsg/ResizeMsg/
  NoticeMsg/ComponentMsg/ViewRejectedMsg/ResponseMsg/StreamMsg/TickMsg/ErrorMsg`。

### 4.1 内置组件（`sdk/widgets`）

带 `Build()` 的组件都是纯结构体：改自己的字段 → `View` 里重新 `Build()`；
不持有定时器或 goroutine，`Build()` 返回 `*sdk.Builder`，可直接当子节点，也可
继续 `.Flex(1).ID(...)` 后再用。纯逻辑辅助（`Hit`/`ApplyWheel`/`Validator`/
`Format*`/`Style*` 等）没有 `Build()`，直接在程序里调用。

> 每个组件的渲染图鉴见 clients/tui/docs/WIDGETS.zh-CN.md（由 go run ./clients/tui/cmd/tui2-widgetdoc 生成）。

**布局 / 容器**

| 组件 | 用途 |
|---|---|
| `SplitLayout` / `FloatingLayer` | 分栏（`Rects()` 算矩形，自己摆内容）、浮层堆叠 |
| `Card` / `Frame` / `Divider` | 卡片 / 带行标记的框 / 分隔线 |
| `BorderBox` + `BorderNormal/Rounded/Thick/Double` | 自选边框字形，标题嵌上边框 |
| `Rect` / `Distribute` | 几何矩形、按比例分配空间 |

**导航 / 外框（chrome）**

| 组件 | 用途 |
|---|---|
| `TabBar` / `StatusBar` / `TitleBar` / `Footer` / `KeyHint` | 标签页、状态栏、标题、页脚、按键提示 |
| `Button` / `Picker` / `Toast` | 按钮、选择器、临时提示 |
| `ProgressBar` / `Spinner` / `Badge` / `Tags` | 进度、加载帧（自己推进 Frame）、状态标签 |

**内容 / 交互**

| 组件 | 用途 |
|---|---|
| `List` / `VirtualList` | 长列表窗口化（只渲染可见行）、选择、滚动、`Follow` 跟随 |
| `Table` | 列定义（宽/弹性/对齐/Format）、表头、CJK 截断、选中行 |
| `TextInput` / `TextArea` | 编辑、光标、密码遮罩、最大长度、多行滚动；`HandleKey` 吃编辑键 |
| `Modal` / `Menu` | 浮层（可选遮罩）；菜单项热键/禁用/分隔线，值由你命中后自行派发 |
| `RichText` / `Span` | 一行内多段样式；`WrapText/WrapSpans` CJK 安全换行；`ParseEmphasis` 支持 `**bold**`/`_em_` |
| `Scrollbar` | 按比例 thumb、点击定位，纵横两用 |

**鼠标 / 表单 / 日期**

| 组件 | 用途 |
|---|---|
| `HitRegion` / `Hit` / `ListRegions` / `TableRegions` | 命中测试；`List/Table.RowAt` 反查行 |
| `ApplyWheel` / `List.OnWheel` / `VirtualList.OnWheel` / `TableScroll` | 滚轮 → offset |
| `Drag` / `ClickTracker` / `Hover` / `ContextMenu` | 拖拽选择、单/双/三击、悬停样式、右键菜单 |
| `Form` / `Field` / `Validator` | 焦点导航、`Values/Validate`；`Required/MinLen/Pattern/Email/IntRange/OneOf/All/Custom` |
| `Select` / `Option` | 下拉选择、typeahead、禁用项、`BuildDropdown()` |
| `Date` / `Calendar` / `DateField` | 日期运算、Mon-first 月历、日期校验 |

**图表 / 格式化 / 主题**

| 组件 | 用途 |
|---|---|
| `Sparkline` / `BarChart` / `Heatmap` / `Meter` / `Legend` | 文本图表（八分块/柱状/热力/仪表/图例），自动量程、锯齿行补齐 |
| `FormatBytes/Count/Duration/Percent/Float` / `ScaleValue` / `PadLeft/PadRight` | 数字格式化 |
| `Theme` + `DarkTheme()/LightTheme()` + `ThemedXxx(theme)` | 语义化样式 token |
| `StyleMuted` 等 `Style*` 常量 + `WithBold/Underline/Reverse/Italic/Fg/Bg` | 样式 token 常量与组合（避免手写字符串） |

> 三语言都有组件库，命名按各自惯例：Go `widgets.List`、Python
> `tui2sdk.widgets.list.List`（模块命名空间；旧 `ChromeApp` 一族在
> `widgets.chrome`）、TS `sdk.widgets.list.List`（`src/widgets/*.js` + `.d.ts`）。

### 4.2 性能（SDK 层）

- 编解码走池化缓冲：Commit/Emit/Stream 稳态 **0 alloc**（Commit 按
  100/1000/10000 节点树计，数据见 `clients/tui/sdk/bench/RESULTS.md`）。
- **增量视图自动启用**：宿主在 HELLO 广告 `features["view_delta"]` 时，`app.Program`
  自动对上一帧做 diff、发 `VIEW_DELTA`（路径式补丁，协议 §2.1），补丁不比全量
  小或无法表达时自动回退全量 VIEW。10k 节点改一个叶子：帧 **86.5 KB → 40 B**
  （0.046%）。调试可设 `Program.ForceFullView=true`。
- 宿主侧先做过一轮**分配优化**：10k 节点全量 VIEW 从 **17.7 ms / 37 MB /
  269k allocs 降到 ~3.5 ms / 9.7 MB / 30k allocs**（全量路径）。
- 增量帧另外做 **COW 路径复制 + 未改子树 layout 复用**（rect 不变才复用）、
  **O(changed) 提交簿记**（node 映射/prune、focus、节点计数）和**树形 rect 索引**
  （复用子树按引用拼接，不再每帧物化 10k 条平坦 map）：
  同样的 1 叶子 delta 从最初 **3.5 ms / 9.3 MB / 30k allocs** 降到
  **~1.24 ms / 4.6 MB / ~215 allocs**（端到端约 2.7×，分配 -99%；数字随机器
  负载有波动，以 `RESULTS.md` 的中位数为准）。
  （数据与命令见 `clients/tui/sdk/bench/RESULTS.md`；复用正确性由随机 patch
  序列 + 全量重解的等价性 fuzz 保证，含映射有界/计数不漂移断言。）
- **大数据量程序用 `app.Memo` 让 diff 变 O(changed)**：`View()` 里对不变的大子树用
  `memo.Box(key, build)` 返回同一指针，再用 `sdk.Raw(...)` 组合进 builder 树，
  diff 的指针快路径直接跳过（10k 节点改一个叶子：diff ~7 µs vs ~0.6–0.8 ms，帧仍
  ~44 B）。`app.Program.Memo` 非 nil 时框架每批自动 `BeginFrame/EndFrame` 并回收
  未用 key；返回的 box 必须视为不可变（详见 §5.5）。
- 大列表用 `List/VirtualList`，构建成本只与可见行数相关。
- 自定义 reader 可开 `Decoder.ReuseBuffer(true)`：解码返回的 payload 只在下一次
  Decode 前有效；默认仍是原来的每帧分配行为。
- 协议兼容：不支持 `view_delta` 的旧宿主上，程序始终发全量 VIEW，线格式未变。

## 5. 常用配方

### 5.1 列出终端并绑定

`sources` 事件里每个终端是一个 `*pb.Source`（`GetKind()=="terminal"`、
`GetEndpoint()`、`GetTerminalId()`、`GetExited()`）。把某一项绑到盒子：

```go
sdk.Terminal("terminal:" + src.GetEndpoint() + ":" + src.GetTerminalId())
```

新建一个终端（走 access/pool，持久、其他客户端可见）：

```go
client.Emit("terminal.create", &pb.MethodParams{Endpoint: "local"}, func(r *pb.Response) {
    if !r.GetOk() { return } // r.GetError() 是可读原因
    id := "terminal:" + r.GetData().GetEndpoint() + ":" + r.GetData().GetId()
    // 把 id 放进你的状态，下一帧绑到盒子上
})
```

### 5.2 用 access 的能力（`access.call`）

`access.call` 是"透明转发"：你用 access 的 proto 生成客户端，序列化一条命令，
宿主替你发到指定 endpoint 的连接上，把结果原样带回来。

```go
import (
    "github.com/anytty/anytty/proto/access/apipb"
    gproto "google.golang.org/protobuf/proto"
)

cmd, _ := gproto.Marshal(&apipb.CommandEnvelope{
    Command: &apipb.CommandEnvelope_StorageGet{StorageGet: &apipb.StorageGetCommand{
        Key: &apipb.StorageKey{AppId: "herdr", Scope: apipb.StorageScope_STORAGE_SCOPE_PRIVATE, Key: "pinned"},
    }},
})
client.Emit("access.call", &pb.MethodParams{Endpoint: "local", AccessCommand: cmd}, func(r *pb.Response) {
    if !r.GetOk() { /* 宿主/连接错误 */ return }
    var result apipb.ResultEnvelope
    if err := gproto.Unmarshal(r.GetData().GetAccessResult(), &result); err != nil { return }
    if apiErr := result.GetError(); apiErr != nil { /* access 侧拒绝，看 Message */ return }
    // result.GetStorageGet()... 就是你要的数据
})
```

endpoint 名字从哪来？常见是 `local`（本机 access）或 registry 里的远端
endpoint id（可在 `sources` 里看到）。`access.call` 不弹确认、不做过滤——**你的
程序是全权的**，删文件/撤销授权这类操作会直接生效。

### 5.3 订阅事件（终端创建/退出等）

```go
// Stream 回调要在创建客户端时就带上（它接管后续所有 STREAM 帧）：
var client *sdk.Client
client = sdk.New(os.Stdin, os.Stdout, sdk.Handlers{
    Hello: func(h *pb.Hello) {
        subscribe, _ := gproto.Marshal(&apipb.CommandEnvelope{
            Command: &apipb.CommandEnvelope_EventSubscribe{EventSubscribe: &apipb.EventSubscribeCommand{
                Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE},
            }},
        })
        client.Emit("access.stream.subscribe", &pb.MethodParams{
            Endpoint: "local", StreamId: 2, AccessCommand: subscribe,
        }, nil)
    },
    Stream: func(f *pb.StreamFrame) {
        if f.GetKind() != "data" { return }
        var env apipb.EventEnvelope
        if gproto.Unmarshal(f.GetPayload(), &env) != nil { return }
        if lc := env.GetTerminalLifecycle(); lc != nil {
            // lc.GetTerminal() = 终端快照（state/name/...）
        }
    },
})
_ = client.Loop()
```

### 5.4 下载文件（流式）

两步：`access.call` 打开 → `access.stream.open` 挂流；之后用 STREAM 帧收数据，
按窗口回 ack。

```go
// 第一步：打开（拿 resource handle）
open, _ := gproto.Marshal(&apipb.CommandEnvelope{
    Command: &apipb.CommandEnvelope_FileDownloadOpen{FileDownloadOpen: &apipb.FileDownloadOpenCommand{Path: "/tmp/a.bin"}},
})
client.Emit("access.call", &pb.MethodParams{Endpoint: "local", AccessCommand: open}, func(r *pb.Response) {
    var result apipb.ResultEnvelope
    _ = gproto.Unmarshal(r.GetData().GetAccessResult(), &result)
    handle := result.GetFileTransferOpen().GetTransfer()      // 含 window/chunk 与 resource
    resourceBytes, _ := gproto.Marshal(handle.GetResource())
    // 第二步：把 resource 绑定成流（stream_id 由你分配）
    client.Emit("access.stream.open", &pb.MethodParams{
        Endpoint: "local", StreamId: 1, AccessResource: resourceBytes,
    }, nil)
})

// 第三步：收数据/回 ack。wirepb 是公开 proto，用 proto.Unmarshal 即可；
// 在 Handlers.Stream 里维护 received/acked 两个变量：
//   kind=="data" && wire_type==wire.TypeFileData → 解 wirepb.FileTransferData{Offset,Data}
//   累计到 windowBytes 后回 ack（delta credit）：
//     ack, _ := gproto.Marshal(&wirepb.FileTransferAck{
//         Offset: int64(received), WindowBytes: int64(received) - acked,
//     })
//     acked = int64(received)
//     _ = client.SendStream(&pb.StreamFrame{
//         StreamId: 1, Kind: "data", WireType: uint32(wire.TypeFileAck), Payload: ack,
//     })
//   wire_type==wire.TypeFileFinish → 解 wirepb.FileTransferFinish{Size,Sha256} 校验
//   kind=="close"/"error" → 结束
// 需要的 import（文件头）：
//   "github.com/anytty/anytty/proto/access/wire"
//   "github.com/anytty/anytty/proto/access/wirepb"
```

上传同理：`FileUploadOpen` + 发 `wire.TypeFileData`，收 `wire.TypeFileAck`，
最后发 `wire.TypeFileFinish`（size+sha256）。

### 5.5 大界面：用 `app.Memo` 减少 diff 成本

`app.Program` 默认对上一帧做全树 diff；树很大时（>1000 节点）diff 本身会接近
全量开销。把不常变的大子树 memo 起来即可让 diff 只走变化路径：

```go
package main

import (
    "fmt"
    "os"

    "github.com/anytty/anytty/clients/tui/sdk"
    "github.com/anytty/anytty/clients/tui/sdk/app"
    pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type model struct {
    memo    *app.Memo
    terms   []string
    termRev int
}

func (m *model) Init() app.Cmd              { return app.None }
func (m *model) Update(msg app.Msg) app.Cmd { return app.None }

// termList builds the (expensive) list subtree.
func (m *model) termList() *pb.Box {
    rows := make([]*sdk.Builder, 0, len(m.terms))
    for _, t := range m.terms {
        rows = append(rows, sdk.Text(" "+t))
    }
    return sdk.Col(rows...).Build()
}

func (m *model) View() *pb.Box {
    // 只在 m.termRev 变化时重建终端列表；否则复用同一棵 box 树（同一指针）。
    list := m.memo.Box(fmt.Sprintf("terms:%d", m.termRev), m.termList)
    return sdk.Col(
        sdk.Text(" terminals ").Style("tab_active"),
        sdk.Raw(list), // Raw 保留指针：diff 的指针快路径会整棵跳过它
        sdk.Text(fmt.Sprintf(" %d 个终端 ", len(m.terms))).Style("muted"),
    ).Build()
}

func main() {
    m := &model{memo: &app.Memo{}}
    client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{})
    _ = (&app.Program{Client: client, Model: m, Memo: m.memo}).Run()
}
```

要点：
- key 必须可比（string 或可比较 struct）；key 变化才重建，未用到的 key 在批次
  结束时自动回收（内存有界）。
- **memo 返回的 `*pb.Box` 视为不可变**：不要再改它的字段，否则指针复用会跳过
  本该发出的 diff（框架契约，和 diff 的指针快路径同一假设）。
- 只对"大且不常变"的子树用；小节点直接新建更简单。

## 6. Python / TS 速览

| 能力 | Go | Python | TS |
|---|---|---|---|
| 创建客户端 | `sdk.New(..., sdk.Handlers{...})` | `Client(sys.stdin.buffer, sys.stdout.buffer).run(App())` | `new Client(process.stdin, process.stdout).run(new App())` |
| 声明界面 | `sdk.Box()/Text()/Terminal()` | `builder.box()/text()/terminal()` | `builder.box()/text()/terminal()` |
| 提交 | `client.Commit(root, keys)` | `app.client.commit(root, claim)` | `this.client.commit(root, claim)` |
| 调方法 | `client.Emit(method, params, cb)` | `app.emit(method, params, cb)` | `this.client.emit(method, params, cb)` |
| 收流帧 | `Handlers.Stream` | `App.on_stream(frame)` | `App.onStream(frame)` |
| 发流帧 | `client.SendStream(frame)` | `app.send_stream(frame)` | `this.client.sendStream(frame)` |

模板分别在 `clients/tui/templates/{go,python,ts}`；三语言的一致性由同一套
fixtures 验证（见 §8）。

## 7. 打包与分发

- Go：编译产物是**一个可执行文件**，可直接分发（SDK 静态链接进去）。
- Python/TS：发布时要把对应的 SDK 一起带上，或让用户安装/设置路径
  （`TUI2_PYTHON_SDK=/path/to/clients/tui/sdk/python`；TS 用相对/绝对路径 require）。
- 运行方式二选一：
  - `TUI2_SHELL=/path/my-shell anytty`（推荐，CLI 会先确保本地栈）；
  - `tui2 -shell "/path/my-shell --flag"`（自己做宿主管理）。
- 程序以**当前用户权限**运行，能读写文件/网络；宿主不会沙箱它（这是设计：
  你自己写的程序自己负责）。

## 8. 自检与调试

```bash
# 协议一致性：你的程序要能通过这套 fixtures（任何语言都能自证）
go run ./clients/tui/cmd/tui2-sdk-verify --fixtures clients/tui/conformance/fixtures.jsonl --cmd "/path/my-shell"

# 开发循环（热重载 + 帧日志 + 崩溃提示）
bash clients/tui/scripts/dev.sh /path/my-shell

# 看双向帧（JSON Lines，排障神器）
/tmp/tui2 -protocol-log /tmp/frames.jsonl -shell "/path/my-shell"
```

常见报错：
- RESPONSE 的 `error` 形如 `access.call: endpoint "x" is not registered`：
  endpoint 名不对（用 `local` 或 registry 里的 id）。
- `missing params.xxx`：没填必填参数（看 `PROTOCOL` §4 的 params 列）。
- 请求到了 access 但被拒绝：`access.call` 的 `access_result` 里是
  `ApiError{code: API_ERROR_CODE_*, message}`，看 `message` 排查。
- 流终止：STREAM 帧 `kind="error"`（ack/window 非法、超 4MiB 队列、资源被释放
  等；缺 ack 只是暂停下载，不会报错）。

## 9. 常见坑（先看这里）

1. **stream_id 由你自己分配**（非 0、关闭前不要复用）。
2. **`wire_type` 原样透传**：它是 access 的帧类型；不要自己发明。
3. **单写者纪律**：同一个终端的 attach/resize/input 只能有一个驱动方；如果你
   用 `access.call` 绕过类型化方法 attach，就等于自己接管了 owner 责任。
4. **不要阻塞事件循环**：`Handlers` 回调里做重活会卡住整个界面；开 goroutine。
5. **sources 是权威快照**：按 source id 幂等对账，不要假设事件顺序。
6. **程序崩溃会被宿主重启**（epoch 变化、布局重建）；需要持久状态就自己落盘，
   或用 access storage。
