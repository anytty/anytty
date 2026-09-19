# TUI v2 外观与扩展指南（CUSTOMIZE）

> 适用版本：tui2-shell 0.2.0 / tui2 0.3.0。本文回答两个问题：**外观归谁管**、
> **想改主题 / 加按钮 / 加组件时改哪里**。协议与内核语义仍以
> `PROTOCOL.zh-CN.md`、`ARCHITECTURE.zh-CN.md` 为准，本文不重复。
> 默认外观 = 老版推荐配置（coralline-candy）：图标/颜色/键位映射表见
> `RECOMMENDED_CONFIG.zh-CN.md`。

## 0. 一句话结论

布局程序（`clients/tui/cmd/tui2-shell` 或你自己的程序）负责**所有外观决策**：
盒子结构、尺寸、间距、边框/标题、颜色，全部在程序侧决定。程序发**显式样式串**
（`fg:#RRGGBB;bg:#RRGGBB;bold;dim;reverse`），宿主只做"显式样式 → SGR"翻译，
**不持有主题/色板**，也不读配置文件。**程序不产生转义字节。**

内核（`clients/tui/kernel`）只解算盒子几何、命中测试与合成，**不画边框、不含内缩**：
chrome（边框/标题/角标/颜色）由内容负责——程序用文本行或 `sdk/widgets` 自绘，
内置组件（terminal）也自绘并声明自己的 inset。

框架与宿主没有任何配置文件：`tui2.json` 只是 **tui2-shell 程序的配置**
（主题、图标集、sidebar/gap/clock/keybindings/startup/endpoints）。

v2 里没有 `panel` 概念——面板已并入 slot，位置只由 `slot` 在 tab 内的比例决定。

## 1. 概念层级：workspace > tab > slot

| 概念 | 归属 | 说明 |
|---|---|---|
| `workspace` | 程序数据 | 逻辑工作区；单机默认 `local`，来自 view id |
| `tab` | 程序数据 | 一组 slot；`1..9` 直接切换，`+` / `Ctrl-T` 新建 |
| `slot` | 程序数据 | 一个可绑定终端的格子；绑定靠 `sources` 快照对账，解绑不杀 PTY |
| `box` | 协议视图 | slot 在 view 里的具体盒子；`sdk.Builder` 负责构造 |
| `terminal` | 宿主组件 | `box.content.self` 指向的内容源，边框/角标由组件自绘并声明 inset=1；颜色由程序经 `content.props` 下发 |

一个 slot 的完整生命周期：空的提示卡（`Card`，程序侧自绘边框）→ picker 绑定 source →
`sdk.Terminal(sourceID)` 盒子 → 退出角标 `[exited N]` → `Ctrl-E` 同 id 重启。

## 2. 主题在程序：显式样式

主题库在**程序侧**（`clients/tui/cmd/tui2-shell/theme.go`）：每个语义槽位（`chrome`
`tab_active` `selection` `border_focus` `muted` `accent` …）解析成一条**显式样式串**，
随 view 一起发给宿主；宿主只翻译成 SGR，不知道"主题"是什么。

```go
type Style struct { FG, BG string; Bold, Dim, Italic, Underline, Reverse bool }
render.ParseStyle("fg:#e6e2ec;bg:#0f1117;bold") // 显式样式 → Style
render.Style{FG: "#a78bfa", Bold: true}.String() // → "fg:#a78bfa;bold"
```

格式（`clients/tui/render/style.go`）：

```
fg:#RRGGBB;bg:#RRGGBB;bold;dim;italic;underline;reverse
```

- 片段顺序任意；未知/写坏的片段**安全降级**（忽略该片段，全空则不着色，不报错）。
- 宿主仍保留内建 token（`accent` `border_focus`…）供**自己内部**组件（terminal
  边框兜底、core overlay）使用；那是宿主实现细节，程序不需要也不应依赖。
- 想换配色：改 `theme.go` 里的 `recommended`/dark/light 表，或加一套新表并在
  `themeByName` 注册（`recommended` 是默认，即老 coralline-candy 推荐配色）。
- terminal 组件的边框/标题/角标颜色也由**程序**决定：shell 把当前主题解析成
  `content.props` 挂在终端槽上（`chrome.border`、`chrome.title`、`chrome.border_focus`、
  `chrome.border_dead`、`chrome.badge`，值都是显式样式串），`tui2.json` 的 `theme`
  切到 light 时这些颜色随之变化；组件只在程序没给 key 时回落内置默认。

### 2.1 示例：给终端槽换边框颜色

终端槽的 props 由 `theme.go` 的 `terminalChrome()` 生成，改主题表即可同时换掉
dark/light 两套边框色。只想单独改某个语义槽（例如让非活动边框更亮）：

```go
// clients/tui/cmd/tui2-shell/theme.go
func themeDark() map[string]render.Style {
    return map[string]render.Style{
        // …
        "border":       {FG: "#6b7089"}, // chrome.border（非活动/非退出）
        "border_focus": {FG: "#a78bfa", Bold: true}, // chrome.border_focus
        "border_dead":  {FG: "#a35d5d"}, // chrome.border_dead（已退出）
        "muted":        {FG: "#9a94a8", Dim: true}, // chrome.title
        "warning":      {FG: "#f0c05a"}, // chrome.badge（[exited N]/[↑N]）
    }
}
```

也可以不复用主题表，直接给某个终端槽下发任意显式样式（自己的程序里）：

```go
box := sdk.Terminal("terminal:local:main").
    Props(map[string]string{
        "chrome.border":       "fg:#565f89",
        "chrome.border_focus": "fg:#7aa2f7;bold",
        "chrome.badge":        "fg:#e0af68", // 缺省 key 回落组件内置默认
    })
```

宿主对这些 key 只透传不解释：`kernel → runtime → placement → 组件工厂` 原样送达；
组件不认识的 key 直接忽略，写坏的值安全降级为不着色。

### 2.2 图标集（`icons`，默认 = 推荐配置）

图标与颜色一样是**程序侧**决策：`tui2.json` 的 `icons` 选预置或逐名覆盖，
shell 把它们解析进 view，宿主只看见普通文本盒子。

```json
{ "icons": "recommended" }
{ "icons": "unicode" }
{ "icons": "ascii" }
{ "icons": { "preset": "recommended", "map": { "tab_new": "N", "slot_close": "C" } } }
```

| 预置 | 说明 |
|---|---|
| `recommended`（默认） | 老 coralline-candy 的 Nerd Font 码点：`󰙅 󰐕 󰅖 󰑐   `、模式徽标 `󰌌 󰱼 󰆏 󰋖` 等 |
| `unicode` | M20 之前的 v2 观感：`+`、`⟳ ⇔ ⇕ ✕`、无模式图标 |
| `ascii` | 无 Nerd Font 的安全回退：`ws`、`+`、`R`、`|`、`-`、`x` |

完整图标名/码点/用途表与"哪个界面用哪个名"见
`RECOMMENDED_CONFIG.zh-CN.md` §1。`map` 的 key 必须是已知图标名（写错是
config error，footer 显示 `config error:` 并回落默认）；value 是任意字符串，
空字符串表示隐藏该图标。覆盖成宽字符也可以，宽度由 `sdk.DisplayWidth`/render
按真实 cell 数处理。

## 3. 配置化（只有 shell 的配置文件）

`tui2.json` 是 **tui2-shell 程序自己的配置**，宿主不读任何配置文件。
默认路径 `$XDG_CONFIG_HOME/anytty/tui2.json`，可用环境变量
`ANYTTY_TUI2_CONFIG` 覆盖或 `tui2-shell --config <path>`。

宿主唯一的外部输入是**只读 registry 覆盖**（不写回、不配对）：
`TUI2_ENDPOINTS=<path>[:<path>...]` 或 `tui2 -endpoints <path>`（flag 优先），
显式文件按 name 合并到默认 `$XDG_CONFIG_HOME/anytty/endpoints.yaml` 之前。
dev 里直接看老 endpoint：

```bash
TUI2_ENDPOINTS=~/.config/anytty/endpoints.yaml bash clients/tui/scripts/run.sh
```

| 字段 | 取值 | 语义 |
|---|---|---|
| `theme` | `recommended`(默认) / `dark` / `light` / `coralline-candy`(别名) | 程序侧主题表（解析为显式样式） |
| `icons` | 预置字符串或 `{preset,map}` | 图标集；默认 `recommended`（§2.2） |
| `sidebar` | bool，默认 `true` | 左侧导航栏（Ctrl-W 可临时切换） |
| `gap` | `0` / `1`，默认 `1` | slot 间距与细分隔线 |
| `clock` | bool 或 `{enabled,format}` | 兼容字段：仍解析但不参与推荐 footer（M22 起右段用老 `*_summary`：workspace/floating/terminals） |
| `double_click_forward_ms` | 1..5000 | picker 双击向前窗口 |
| `keybindings` | 动作 → 按键 | 覆盖内置绑定（动作白名单固定） |
| `startup.auto_attach_first` | bool，默认 `true` | 冷启动空 view 是否自动接管第一个终端 |
| `startup.cwd` | string | 新建终端的工作目录 |
| `endpoints` | `{name,kind,label,argv,cwd,env,socket,address,connect_mode}[]` | endpoint：`command` 起本地 PTY，`daemon` 连接已有 anytty 终端池（§3.1、`ENDPOINTS.zh-CN.md`） |

```json
{
  "theme": "recommended",
  "icons": { "preset": "recommended" },
  "sidebar": true,
  "gap": 1,
  "clock": { "enabled": true, "format": "24h" },
  "double_click_forward_ms": 300,
  "keybindings": { "picker.open": "ctrl-g", "slot.close": "q" },
  "startup": { "auto_attach_first": true, "cwd": "" },
  "endpoints": [
    { "name": "remote", "kind": "command", "label": "dev box",
      "argv": ["ssh", "dev-box", "anytty", "attach"] }
  ]
}
```

可配置动作（白名单）：`pane.mode`、`pane.split_h`、`pane.split_v`、`picker.open`、
`tab.new`、`slot.close`、`scroll.copy`、`sidebar.toggle`、`terminal.restart`、
`prompt.open`、`help.open`。
按键写法：单字符（`x`）、`ctrl-a`..`ctrl-z`、`up/down/left/right`；
`esc/enter/tab/page-up/page-down/ctrl-q` 保留，不可绑定。
`pane.split_h`/`pane.split_v` 默认 `%`/`"`；想用老推荐的 `Ctrl-D`/`Ctrl-E`
分屏（并把重启挪到 `Ctrl-R`）：

```json
{ "keybindings": { "pane.split_h": "ctrl-d", "pane.split_v": "ctrl-e", "terminal.restart": "ctrl-r" } }
```

### 3.1 命令式 endpoint（远程/容器）

v1 的 endpoint 模型是"连接命令"：host 只接收
`terminal.create{endpoint, argv, cwd, env}`，用 argv 起一个本地 PTY，之后输入/
输出/resize/重启与本地终端走完全相同的协议路径。picker 在已有终端之后列出配置的
endpoint（`󰌷 label`），回车/单击即创建并绑定 `terminal:<name>:<id>`。

```json
{
  "endpoints": [
    { "name": "remote", "kind": "command", "label": "dev box",
      "argv": ["ssh", "-t", "dev-box", "anytty", "attach", "--socket", "/run/anytty.sock"],
      "env": { "TERM": "xterm-256color" } },
    { "name": "container", "kind": "command", "label": "docker web",
      "argv": ["docker", "exec", "-it", "web", "bash"] }
  ]
}
```

约束：`name` 不能为空且不能含 `:`（它会成为 source id 的一部分）；`command`
（缺省）必须有 `argv`。与老 终端池协议 endpoint（routes/连接状态/自动重连）的
对照见 `RECOMMENDED_CONFIG.zh-CN.md` §4/§5。

### 3.2 终端池 endpoint（连接已有 anytty 终端池）

`kind: "daemon"` 时 host 不再起本地进程，而是按 终端池协议连接 socket，
列出/attach/输入/resize/kill/remove 全部走真实 终端池；远程可先用
`ssh -L` 把远端 unix socket 转发到本地再填 `socket`。完整模型与调用序列见
`ENDPOINTS.zh-CN.md`。

```json
{
  "endpoints": [
    { "name": "dev", "kind": "daemon", "label": "dev-daemon",
      "socket": "/run/user/1000/anytty-v2-wire7-dev.sock" },
    { "name": "prod", "kind": "daemon", "label": "prod",
      "socket": "/tmp/anytty-prod.sock", "argv": ["tmux", "new", "-A", "-s", "main"] }
  ]
}
```

- picker 按 endpoint 分组：终端池 终端（带 `dev · live/exited/offline` 信息列）
  与配置行（`󰌷 dev-daemon`，信息列 `endpoint · daemon local-unix` 或
  `endpoint · daemon tcp`）。
- `argv/cwd/env` 可选：选中配置行新建终端时作为 终端池 侧 command/cwd/env；
  空 argv 用 终端池 默认 command。已有终端直接 attach。
- 断线时 footer 显示 `endpoint dev offline: …` notice，重连后自动重订阅并
  用 终端池 快照重建画面（不重放历史）；`terminal.kill` 走 终端池 kill，
  TUI 退出只 detach。
- `connect_mode` 缺省 `local-unix`；`tcp` 用 `address` 指定 `HOST:PORT`，
  对端是远端 终端池 socket 的透明隧道；`direct-webrtc-tcp` 不做，连接时给出
  可读错误（picker 仍展示配置行）。
- 终端池 endpoint 的权威配置是 CLI/TUI 共享的
  `~/.config/anytty/endpoints.yaml`（`client/endpoint.DefaultPath()`）：tui2.json
  同名 终端池 endpoint 会被共享 registry 覆盖（tui2 只读、不写、不配对）。
  去重边界见 `clients/tui/docs/CLIENT_SHARING.zh-CN.md`。

远程接入的三种最小可用方式（ssh 转发 unix socket、ssh/TCP 隧道、老命令式
`ssh host anytty ...`）与完整手动复测命令见 `REMOTE.zh-CN.md`：

```json
{
  "endpoints": [
    { "name": "server", "kind": "daemon", "label": "server",
      "socket": "/tmp/anytty-remote.sock" },
    { "name": "server-tcp", "kind": "daemon", "label": "server tcp",
      "connect_mode": "tcp", "address": "127.0.0.1:17777" }
  ]
}
```

配合 `ssh -N -L /tmp/anytty-remote.sock:/run/user/1000/anytty-v2-wire7.sock user@host`
（或 `ssh -N -L 127.0.0.1:17777:/run/user/1000/anytty-v2-wire7.sock user@host`）。

配置解析失败不崩：footer 显示一行 `config error: …`，程序用默认值继续运行。
`tui2-shell --print-default-config` 打印示例 JSON。
宿主的运行参数只有 `-shell/-version/-help`：`tui2 -shell /path/to/tui2-shell`。

## 4. 三种扩展路径

### 路径 A：只改外观（推荐起点）

改 `clients/tui/cmd/tui2-shell/view.go`（结构）与 `theme.go`（配色）：所有画面都从
`view()` 出来，只需换主题槽位或改盒子结构，不碰内核、协议与宿主。

```go
// 活动 tab 用高亮块，非活动用 muted（样式来自程序侧主题表）
row.Child(sdk.Text(label).ID(item.ID).Style(m.style("tab_active"))) // 单个盒子
// 选中行整行加 selection 背景（overlay 里一行一个盒子）
sdk.Text(text).Style(m.style("selection")).Width(inner).Height(1)
```

### 路径 B：用/扩 widget 库（`clients/tui/sdk/widgets`）

widget 全是**纯构建函数**：输入配置，输出 `*sdk.Builder`；不持有状态、
不发方法。点击由调用方命中后处理——框架没有"按钮"概念。

```go
bar := widgets.TabBar{
    Left:  &widgets.Segment{Text: " local ", Style: "chrome", ID: "workspace"},
    Items: []widgets.TabItem{{ID: "tab:0", Title: "1:1", Active: true}},
    Plus:  true, PlusID: "tab:new",
}.Build().ID("header").Height(1)

footer := widgets.StatusBar{
    Left:  []widgets.Segment{{Text: "NORMAL", Style: "status"}},
    Right: []widgets.Segment{{Text: "12:30", Style: "muted"}},
    Width: cols,
}.Build()
```

`Frame{ID,Title,Width,Height,Style,Rows}` 是程序侧自绘边框的底座（顶栏题字 + 每行
侧边条 + 底栏，全部是显式样式的文本行），`Card{Title,Lines,Width,Height,Center}` 用它
做空槽/对话框，`Divider{Vertical,Length}` 做分隔线，`Button{ID,Text,Hot}` 做可点文本
（`Input("mouse")` 是协议字段，命中后按 id 自己分发），`KeyHint{Mode,Keys}` 做按模式
变化的快捷键组。扩展方式就是加一个新的纯函数（保持"无状态 + 无方法调用"两条约束即可）。

### 路径 C：加宿主组件（需要实现协议）

如果内容不是文本/终端，而是全新的 `content.self` 组件（例如图片、表格），
需要在 `clients/tui/runtime` 注册组件并在宿主画 `render.Line`，协议 side 只引用
`self: "<sourceID>"`。这条路径动的是宿主实现，不影响布局程序的外观规则。

## 5. 加一个"按钮"的最短代码

```go
// 1) 程序：声明一个可命中的盒子（Button 只是"带 Input 的文本"）
paneBtn := widgets.Button{ID: "act:split", Text: "[ 分屏 ]", Hot: "s"}.Build()

// 2) 宿主会替你做命中测试并把 mouse 事件路由回程序（node = "act:split"）

// 3) 程序：在 mouse 处理里命中后执行动作
func (m *model) mouse(action, _, node string, _, _ int) []request {
    if node == "act:split" && action == "press" {
        m.split("row")
    }
    return nil
}
```

要点：按钮不是框架概念，是"盒子 + `input:["mouse"]` + 程序命中处理"。
热键同理：`Button.Hot` 只负责视觉强调，按键路由由 `keybindings` 决定。

### 5.1 完整示例：在槽标题栏加动作按钮

tui2-shell 的每个 slot 都有一条**程序自绘的 1 行标题栏**：左边 `▎标题`，右边
四个按钮（默认推荐图标 `󰑐   󰅖`，`icons: "unicode"` 时为
`⟳ ⇔ ⇕ ✕`）。整条标题栏是 slot 的 `pos` 子盒，压在 terminal 组件自绘的
边框题字上——**组件不感知按钮，协议/内核/宿主都不变**，slot 几何（以及 PTY 尺寸）
保持不变。下面是真实实现的骨架（`model.go` + `view.go` + `icons.go`）：

```go
// ---- model.go：按钮定义 + 节点 id（每个 slot 一套，id 带 slot id）----
const (
    slotButtonRestart = "restart"
    slotButtonSplitH  = "split-h"
    slotButtonSplitV  = "split-v"
    slotButtonClose   = "close"
)

// model.go：固定顺序；glyph 来自 tui2.json "icons"（默认推荐）
var slotButtons = []string{slotButtonRestart, slotButtonSplitH, slotButtonSplitV, slotButtonClose}
func (m *model) buttonIcons() []slotButton { /* iconSlotRestart/SplitH/SplitV/Close */ }

func slotButtonNode(slotID, action string) string { return "btn:" + slotID + ":" + action }

func parseButtonNode(node string) (slotID, action string, ok bool) {
    rest, found := strings.CutPrefix(node, "btn:")
    if !found {
        return "", "", false
    }
    slotID, action, found = strings.Cut(rest, ":")
    return slotID, action, found && slotID != "" && action != ""
}

// ---- view.go：标题栏是 slot 的 pos 子盒，合成在组件边框之上 ----
func (m *model) slotTitleBar(s *slot, width int) *sdk.Builder {
    row := sdk.Row().ID("title:"+s.id).Width(width).Height(1)
    // 左：▎标题 + 角标（[exited N]/[↑N]/pending …，各自主题色）；
    // 实际实现按 run 拆分：slotLabelRuns() → border_focus/border_dead/muted + warning。
    row.Child(sdk.Text("▎" + m.slotTitle(s)).Style(m.slotLabelStyle(s)).
        Width(width - 8).Height(1))
    // 右：四按钮，每个 1 格宽、间隔 1 格
    for i, button := range m.buttonIcons() {
        if i > 0 {
            row.Child(sdk.Text(" "))
        }
        node := slotButtonNode(s.id, button.action)
        row.Child(sdk.Text(button.icon).ID(node).
            Style(m.slotButtonStyle(s, button.action, node)). // button_hover/button_pressed/warning
            Width(1).Height(1).Input("mouse"))
    }
    return row.Pos(0, 0) // pos overlay：槽几何与 PTY 尺寸不变
}

// ---- model.go：host 只做命中与路由，程序按 id 分发 ----
func (m *model) buttonMouse(action, node string) []request {
    slotID, button, ok := parseButtonNode(node)
    if !ok {
        return nil
    }
    t, i := m.findSlot(slotID)
    if t == nil {
        return nil
    }
    if action == "release" { // 抬起清除按压高亮
        m.pressed = ""
        return nil
    }
    if action != "press" {
        return nil
    }
    m.active, t.focus = indexOfTab(m.tabs, t), i
    m.pressed = node // 本次 commit 用 button_pressed 重绘，重绘后由 commit 清除
    switch button {
    case slotButtonRestart:
        return m.restartSlot(t.slots[i]) // terminal.restart；s.pending="restart" 显示 …
    case slotButtonSplitH:
        m.split("row") // 等价 `%`
    case slotButtonSplitV:
        m.split("col") // 等价 `"`
    case slotButtonClose:
        return m.closeSlotAt(t, i) // 等价 `x`（解绑式关闭，PTY 不死）
    }
    return nil
}
```

要点：

- **按钮 = 盒子 + 程序动作**：每个图标是一个带 `input:["mouse"]` 的 1 格文本盒；
  host 命中后把 mouse 事件路由回程序（`node` 即盒子 id），程序自己分发。没有新增
  "按钮"概念、协议字段或内核语义。
- **按压/挂起**：press 置 `m.pressed`（该帧高亮），release 或下一次重绘
  （`program.commit`）清除；`restart` 发出后 `slot.pending` 在标题右侧显示 `…`，
  响应回调清除。无终端的 `⟳` 是 no-op + footer toast（`no terminal to restart`）。
- **键盘等价**（PANE 前缀模式）：`Ctrl-E` restart、`%` split-h、`"` split-v、
  `x` close；`restart`/`close` 可被 `keybindings` 覆盖（`%`/`"` 是固定 PANE 键）。
- **主题**（`theme.go`，dark/light 各一份显式样式）：`button`（非焦点槽）、
  `button_hover`（焦点槽；程序没有 hover 通道，用焦点作强调）、
  `button_pressed`（press 帧）；左侧标题用 `border_focus`/`border_dead`/`muted`，
  退出态 `⟳` 与角标用 `warning`。切换 `tui2.json` 的 `theme` 即整体换色。
- **props 不参与**：`content.props` 仍是程序 → **组件**的通道（`chrome.*`，§2.1）；
  标题栏按钮是程序自绘文本，只需要主题表，不新增任何 props key。

## 6. 用任意语言写布局程序（Python 示例）

布局程序只通过 stdin/stdout 的二进制帧协议与宿主通信（PROTOCOL §0），因此语言中立：
Go 不是必需的。仓库自带 `clients/tui/examples/python-shell/`（纯标准库，无 pip 依赖）作为参考证明：

```bash
go build -o /tmp/tui2 ./clients/tui/cmd/tui2
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/shell.py"
```

- **协议要点**：`u32 长度(大端) | u8 类型 | protobuf 载荷`；宿主→程序 HELLO(1)/EVENT(3)/
  RESPONSE(5)，程序→宿主 VIEW(2)/RESULT(4)。`VIEW` 是全量盒子快照 + `keys.claim`（与视图
  同帧原子生效），`RESULT` 是唯一副作用入口（`terminal.attach/create/...`），每个 RESULT
  恰好回一个 RESPONSE。`epoch` 由 HELLO 给出，`rev` 程序自增。
- **`pb.py` 的作用**：手写的极简 protobuf 编解码（varint/长度分隔），只覆盖协议用到的消息：
  解析 HELLO/EVENT/RESPONSE，编码 VIEW/RESULT/盒子树；字段号以 `.proto` 为准，没有代码生成器
  与运行时依赖。
- **`shell.py`**：约 600 行的状态机 + 盒子树构建，实现 tab/分屏/空槽卡片/footer/picker；
  用自己的 `Program.key_*` 分发按键、用 `build_view()` 产出盒子树。加一个新界面就是加一个
  `key_*` 分支和一个盒子函数——与 Go 程序的做法完全一致。
- **宿主侧只认命令行**：`-shell` 接收"命令 + 参数"（如上面的 `python3 ...`），
  命令不存在会在接管终端前报错退出；程序立即退出时宿主打印明确提示并按重启策略处理。

### 6.1 像素级兼容示例：`legacy.py`（老默认 UI 复刻）

`clients/tui/examples/python-shell/legacy.py` 是同一个协议写的**第二个** Python 程序，目标不是
"最小参考"而是**像素级复刻老默认 UI**（老布局程序 `shell/main.go`）：header
`WS main ▎1 main ×  +`、30 列 sidebar status 盒、带边框/标题/焦点 `▎`/退出角标的
pane、`│` 分段右对齐 footer、picker/help/prompt/manager 浮层、toast、以及
tab/pane/split/close/drag/picker/prompt/help/回看/复制/重启全套交互。

```bash
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/legacy.py"
python3 clients/tui/examples/python-shell/parity_test.py   # 老公式 vs 程序 view 逐格对比
```

它的价值是把"程序侧能自绘到什么程度"推到极限，同时充当框架能力探针：

- **内核无 border**：所有边框/标题/分隔都是程序用文本行自绘（老 `borderLines` +
  `embedTitle` 规则：标签从顶边下标 1 起覆盖，放不下则整条不嵌）；终端 pane 仍用
  组件自绘边框，颜色由 `chrome.*` props 锁定为老主题 token。
- **样式显式化**：老主题 token 全部换算成 `fg:#RRGGBB;...`（含 `mixHostColor` 混色的
  header 底色），见 `LEGACY_PARITY.zh-CN.md` §2。
- **程序光标**：prompt 的 `": "` 光标用 `box.cursor`（field 7）复投，`pb.py` 已支持。
- 与老 UI 的差异、以及复刻过程中发现并修补的框架缺口（如退出码 0 的
  `[exited]` 角标语义）逐条记录在 `LEGACY_PARITY.zh-CN.md` §5/§6。

想基于它做自己的"复古皮肤"：改 `legacy.py` 顶部的 `STYLE_*` 常量（或
`panel_rows`/`overlay_panel`），交互状态机不用动。也可以直接用推荐 preset：

```bash
printf '{"preset": "recommended"}' > /tmp/rec.json
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/legacy.py --config /tmp/rec.json"
# 或 ANYTTY_LEGACY_CONFIG=/tmp/rec.json
```

preset 换图标/颜色与 footer 场景规格，默认（不带 `-config`）仍逐像素复刻老默认 UI，
`parity_test.py` 同时检查默认 28 项、推荐 preset 与 8 行 footer golden（30/30）。

### 6.2 `v3ui.py`：老 v3 外观（surface framework 复刻）

`clients/tui/examples/python-shell/v3ui.py` 是第三个 Python 程序：复刻 git HEAD 的
老 v3（`tui/app` + `tui/render`）在 recommended/coralline-candy 下的界面，
目标样式是 `1.txt` 的真实抓屏——powerline workspace/tab 顶条、card 窗口框上的
`□ title ... ● x1 owner ─ <zoom/split/close> ─` 题字与按钮、`┃ Click to collapse`
提示行、以及 `󰌌 CTRL · P  PANE · … · G 󰒓 SYSTEM` + `󰙅 main 󰹙 0  11` 底栏。

```bash
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py"          # 真实使用
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py --demo"   # 1.txt 目标态
python3 clients/tui/examples/python-shell/v3_parity_test.py                   # 逐字符 parity
```

- **键位与场景**（recommended yaml）：`Ctrl-P` PANE（`x` close、`%`/`Ctrl-D` vsplit、
  `"`/`Ctrl-E` hsplit、`h/l` focus、`t` restart、`k` kill、`q` kill+close、`z` collapse）、
  `Ctrl-R` SIZE（`h/l/k/j` 调相邻 pane 比例）、`Ctrl-O` FLOAT（`n` 新建并自动建终端、
  `h/j/k/l`/方向键移动、`z/m` 折叠、`c` 居中、`x` 关闭、`1-9` 召唤）、`Ctrl-T` TAB
  （`c` 新建、`n/p` 切换、`1-9` 跳转、`x` 关闭）、`Ctrl-W` WORKSPACE、`Ctrl-F` picker
  （floating 场景下绑定到浮窗 pane）、`Ctrl-G` SYSTEM（`q` quit、`o` command、`?` help）、
  `⇧C` copy、`⇧H` clipboard、`⇧V` paste、`Ctrl-Alt-1..5` tab.jump；footer 随场景换 badge/键组。
- **真实分屏**：一个 card 内按 `flow`（row/col）+ `tab.weights` 比例切多个 pane，
  相邻 pane 之间是 1 cell 可拖拽分隔条（`divider:{tab}:{index}`，隐式鼠标捕获），
  `Ctrl-R` 的键盘 resize 与拖拽都会把比例写回 `weights`。split 出的新 pane 立刻
  `terminal.create`，每个 pane 是独立 terminal 组件，PTY 尺寸 = 自己的内容矩形；
  点 pane 内容聚焦，live 下按键只进聚焦 pane，`h/l` 或方向键切焦点。
- **浮窗**：`Ctrl-O n` 新建（老 v3 默认几何 + 级联），自动建终端并绑定；
  标题整行可拖拽移动；`z` 折叠只留标题行（动作组 `◎ ▾ 󰁌 󰅖` 仍在，可展开），
  `x` 关闭=解绑（终端留池）；浮窗画在普通帧之上，激活/拖拽/召唤会把窗口提到
  最高层；`Esc` 回 live 后输入进浮窗 PTY，点主 pane 内容焦点还给 pane。
- **手感**：点 tab 切换/关闭/新建，点 pane 内容聚焦，点 card 上的 zoom/split/close
  按钮执行动作，点 `Click to collapse` 折叠，拖 floating 标题移动，拖分隔条调比例。
- **样式**：全部是程序侧显式样式串（推荐 yaml 的 `#RRGGBB` 直接进
  `fg:/bg:/bold`）；内容区是 v2 terminal 组件，程序下发 `chrome.inset=0`
  （`CUSTOMIZE` §2 的 props 通道）让组件不重复画框，PTY 尺寸 = card 内容矩形。
- **规格/缺口/差异**：逐元素公式、v2 映射、4 个已补齐的框架缺口（`chrome.inset`、
  `ctrl-shift-*` key 名、`pos` overlay 内组件合成、重叠 `pos` overlay 的 placement
  归属）与仍存差异（120 列 footer 裁剪、内容占位）见 `V3_PARITY.zh-CN.md`；
  修改外观改 `v3ui.py` 顶部 `ST_*`/glyph 常量即可，状态机不用动。

## 7. 常见问题

- **为什么我改的颜色没生效？** 检查发出去的是不是**显式样式串**
  （`fg:#RRGGBB;...`）；未知/写坏的片段会被安全忽略（不上色），不会报错。
  程序侧改 `theme.go` 的主题表，宿主不参与配色。
- **边框是谁画的？** 内核不画。程序侧用 `widgets.Frame`/`Card` 自绘；
  terminal 组件自绘并声明 `inset=1`，宿主只用 `rect − inset` 算 PTY 尺寸。
  盒子的 `border` wire 字段已删除（`reserved 6`），terminal 的边框/标题/角标颜色
  一律走 `content.props`（§2.1）。
- **为什么 terminal 边框颜色没跟着主题变？** 只有程序不下发对应 prop 时组件才用
  内置默认色；shell 默认会为每个终端槽下发全套 `chrome.*`。若是自己的程序，
  在绑定盒子上加 `Props(...)`（上例）即可。
- **侧栏默认开着，怎么关？** 配置 `"sidebar": false`，或运行时 `Ctrl-W`。
- **改了 keybindings 但 footer 还显示旧键？** M22 起推荐 footer 是**场景规格表**
  （`footer.go`）：label 优先、按当前场景裁剪，label 内嵌的键是推荐配置的固定文案
  （老版同样如此）；`keybindings` 只改路由。若看到 `config error:` 说明配置没被读到：
  检查 `ANYTTY_TUI2_CONFIG` / `tui2-shell --config` 路径（宿主没有 `--config`）。
