# SDK 组件图鉴（WIDGETS）

本图鉴为 `clients/tui/sdk/widgets` 的每个组件给出「渲染图」：图由真实的宿主管线生成
（`runtime.NewSession` → `HandleView` → `ComposeFrame`），取的是内核/合成器排出的单元格
文本，所以看到的是字形与几何布局。颜色与样式 token 由终端主题在运行时解析，图上不体现，
同一张图在深色/浅色主题下一致。

- 重新生成：`go run ./clients/tui/cmd/tui2-widgetdoc`
- 检查漂移（提交前/CI）：`go run ./clients/tui/cmd/tui2-widgetdoc -check`
- 只打印不落盘：`go run ./clients/tui/cmd/tui2-widgetdoc -stdout`
- 图由生成器产出，不要手改；`clients/tui/cmd/tui2-widgetdoc/main_test.go` 会逐字节比对。

Go / Python / TS 三语言共享同一套组件与命名（命名风格随语言惯例），组件清单与用法见
`clients/tui/docs/SDK-GUIDE.zh-CN.md` §4.1。

## 布局 / 容器

### `SplitLayout`

按权重把空间分成若干 pane，返回每个 pane 与分隔条的 `Rect`；分隔条自己用 `Divider` 画，拖拽由程序处理。

```go
layout := widgets.SplitLayout{Orient: "row", Weights: []int{1, 2}}
panes, gaps := layout.Rects(36, 6)
for _, p := range panes {
    view.Child(widgets.Card{Title: "pane", Width: p.W, Height: p.H}.Build().Pos(p.X, p.Y))
}
```

```
┌─ pane 1 ┐│┌─ pane 2 ─────────────┐
│slot     │││slot                  │
│         │││                      │
│         │││                      │
│         │││                      │
└─────────┘│└──────────────────────┘
```

### `FloatingLayer`

程序侧浮窗：`Frame` 加 `Pos`，合成时压在常规流之上；`Collapsed` 只留标题行。

```go
float := widgets.FloatingLayer{Title: "floating", X: 5, Y: 2, Width: 18, Height: 4,
    Rows: []widgets.FrameRow{{Text: "drag to move"}, {Text: "click to collapse"}}}
view := sdk.Stack(base.Build(), float.Build()).Build()
```

```
┌─ workspace ──────────────┐
│pane 1                    │
│pane┌─ floating ─────┐    │
│pane│drag to move    │    │
│pane│click to collaps│    │
│pane└────────────────┘    │
│                          │
└──────────────────────────┘
```

### `Card`

带边框的文本框，可选水平/垂直居中；空槽位、对话框的积木，边框由 `Frame` 程序侧绘制。

```go
card := widgets.Card{Title: "empty slot", Lines: []string{"press ctrl-b"},
    Width: 22, Height: 6, Center: true}
view := sdk.Col(card.Build()).Build()
```

```
┌─ empty slot ───────┐
│                    │
│    press ctrl-b    │
│                    │
│                    │
└────────────────────┘
```

### `Frame / FrameRow`

程序侧带行标记的框：顶边嵌标题、左右竖线、底边；`FrameRow` 可带命中 `ID` 与 `Input`。

```go
frame := widgets.Frame{Title: "logs", Width: 26, Height: 6, Rows: []widgets.FrameRow{
    {Text: "alpha  ok"},
    {Text: "beta   running", ID: "log:beta", Input: []string{"mouse"}},
    {Text: "gamma  dead"},
}}
view := sdk.Col(frame.Build()).Build()
```

```
┌─ logs ─────────────────┐
│alpha  ok               │
│beta   running          │
│gamma  dead             │
│                        │
└────────────────────────┘
```

### `BorderBox`

自选边框字形的容器（`BorderNormal` / `BorderRounded` / `BorderThick` / `BorderDouble`），标题嵌上边框，内容放 `Child`。

```go
box := widgets.BorderBox{Title: "rounded", Width: 11, Height: 3,
    Border: widgets.BorderRounded()}
view := sdk.Col(box.Build()).Build()
```

```
┌ normal ─┐ ╭ rounded ╮
│         ││         │
└─────────┘╰─────────╯
┏ thick ━━┓ ╔ double ═╗
┃         ┃║         ║
┗━━━━━━━━━┛╚═════════╝
```

### `Divider`

一格宽/高的分隔条：横向 `─`、竖向 `│`（竖向按行构建，整列可见），作为 pane 间隙时可拖拽（ID/Input 由程序命中）。

```go
view := sdk.Col(
    widgets.Divider{Length: 20}.Build(),
    sdk.Row(panes[0], widgets.Divider{Vertical: true, Length: 3}.Build(), panes[1]),
).Build()
```

```
horizontal
────────────────────
vertical
L1│R1
L2│R2
L3│R3
```

### `Rect / Distribute`

纯几何：`Rect` 是矩形值，`Distribute` 按整数比例分格子（至少 1 格，余数给最后一个 pane）。

```go
sizes := widgets.Distribute(80, []int{1, 2, 1}) // → [20 40 20]
panes, gaps := widgets.SplitLayout{Weights: []int{1, 2}}.Rects(30, 6)
```

无需渲染图，`Distribute` 的输出即可说明：

```
Distribute(80, []int{1, 2, 1}) → [20 40 20]
Distribute(10, []int{1, 1, 1}) → [3 3 4]
Rects(30, 6) → panes (0,0 9x6) (10,0 20x6) / gaps (9,0 1x6)
```

## Chrome / 导航

### `TabBar / TabItem`

顶部标签条：左侧 workspace 段 + 每个标签一个可点击文本框（active 用 `[title]`）+ 可选 `+`。

```go
bar := widgets.TabBar{
    Left:  &widgets.Segment{Text: " ws "},
    Items: []widgets.TabItem{{ID: "tab:main", Title: "main", Active: true}},
    Plus:  true,
}
view := sdk.Col(bar.Build()).Build()
```

```
 ws [main] logs  +
 body
```

### `StatusBar / Segment`

一行状态栏：左/右两组 `Segment`，`Width > 0` 时右组贴右边，空间不够先丢尾段再截文本。

```go
bar := widgets.StatusBar{
    Left:  []widgets.Segment{{Text: "▸ connected", Style: "ok"}},
    Right: []widgets.Segment{{Text: "100x30"}},
    Width: 32,
}
view := sdk.Col(bar.Build()).Build()
```

```
 body
▸ connected │     100x30 │ 14:02
```

### `TitleBar`

一行标题条：左侧 `Segment` 组 + 右侧按钮组（`Button`），`Width` 右对齐并截断左组。

```go
bar := widgets.TitleBar{
    Left:    []widgets.Segment{{Text: "terminal:local:alpha"}},
    Buttons: []widgets.Button{{Text: "[x]", Hot: "x", ID: "close"}},
    Width:   32,
}
view := sdk.Col(bar.Build()).Build()
```

```
terminal:local:alpha         [x]
 body
```

### `Footer`

底部栏：可选场景 badge + 左侧按键组 + 右侧摘要段，`Width` 右对齐、空间不足先截左组。

```go
footer := widgets.Footer{
    Badge:    widgets.Segment{Text: "NORMAL"}, HasBadge: true,
    Groups:   []widgets.Segment{{Text: "ctrl-b 前缀"}},
    Right:    []widgets.Segment{{Text: "connected"}},
    Width:    56,
}
view := sdk.Col(footer.Build()).Build()
```

```
 body
NORMAL │ ctrl-b 前缀 │ ctrl-q 退出 │ 2 panes │ connected
```

### `KeyHint`

页脚左侧的按键提示：当前模式 + 一组键位，模式与键位分别用不同 token 渲染。

```go
hint := widgets.KeyHint{Mode: "PREFIX", Keys: []string{"q detach", "n new", "x close"}}
view := sdk.Col(hint.Build()).Build()
```

```
PREFIX │ q detach · n new · x close
```

### `Button`

可点击文本框：协议 `Input("mouse")` + 程序命中处理；`Hot` 字符单独成段方便标快捷键。

```go
button := widgets.Button{Text: "[ Connect ]", Hot: "C", ID: "connect"}
view := sdk.Row(button.Build()).Build()
```

```
[ Connect ]  [ Quit ]
```

### `Picker / PickerRow`

带边框的可选列表（选择器浮层）：行可带命中 `ID`，`Selected` 行加 `▸` 标记。

```go
picker := widgets.Picker{Title: "terminals", Width: 26, Height: 6,
    Rows: []widgets.PickerRow{
        {Text: "local:alpha", ID: "t1", Selected: true},
        {Text: "local:beta", ID: "t2"},
    }}
view := sdk.Col(picker.Build()).Build()
```

```
┌─ terminals ────────────┐
│▸ local:alpha           │
│  local:beta            │
│  dev:build             │
│  old:dead              │
└────────────────────────┘
```

### `Toast`

临时一行提示：程序把它放进下一帧、再下一帧移除即消失，没有宿主计时器。

```go
toast := widgets.Toast{Text: " copied 2 cells ", Style: "overlay", ID: "toast"}
view.Child(toast.Build().Pos(2, 3))
```

```
 body row 1
 body row 2

 copied 2 cells
```

## 内容

### `List`

窗口化单行列表：只渲染可见行，`Height` 含 Header/Footer，选中行加标记；`Follow` 让选中不越窗。

```go
list := widgets.List{
    Header: "terminals", Items: []string{"alpha", "beta", "gamma"},
    Height: 6, Selected: 2, Footer: "j/k move",
}
view := sdk.Col(list.Build()).Build()
```

```
terminals
  alpha
  beta
▸ gamma
  delta
j/k move
```

### `VirtualList`

`List` 的富行版本（`ListRow`）：逐行样式、`Disabled` 置灰，其余窗口化规则相同。

```go
list := widgets.VirtualList{Height: 6, Selected: 1, Rows: []widgets.ListRow{
    {Text: "alpha  connected", ID: "r0"},
    {Text: "gamma  exited", ID: "r2", Disabled: true},
}}
view := sdk.Col(list.Build()).Build()
```

```
sessions
  alpha  connected
▸ beta   connected
  gamma  exited
  delta  connected
enter attach
```

### `Table / Column`

表头 + 对齐/截断的单元格：`Column` 支持固定宽、`Flex` 弹性、对齐与 `Format`，CJK 截断安全。

```go
table := widgets.Table{Width: 40, Rule: true, Selected: 1, Columns: []widgets.Column{
    {Title: "session", Flex: 2},
    {Title: "status", Flex: 1},
    {Title: "uptime", Align: widgets.AlignRight},
}}
view := sdk.Col(table.Build()).Build()
```

```
session             status        uptime
────────────────────────────────────────
alpha               ok             2h10m
beta                ok               18m
gamma               dead              3d
```

### `TextInput`

单行编辑器状态 + 纯 `Build`：`HandleKey` 吃编辑键，焦点行渲染光标格（样式剥离后仍是该字符）。

```go
input := widgets.TextInput{Value: []rune("anytty"), Cursor: 3,
    Width: 24, Focused: true, Placeholder: "type a name"}
view := sdk.Col(input.Build()).Build()
```

```
anytty
type a session name
```

### `TextArea`

多行编辑器：光标是 rune 下标，`RowOffset`/`ColOffset` 控制可视窗口，只渲染可见行。

```go
area := widgets.TextArea{Value: []rune("line one\nline two"), Cursor: 8,
    Width: 24, Height: 4, Focused: true}
view := sdk.Col(area.Build()).Build()
```

```
line one
line two
line three
```

### `Modal`

程序侧对话框：`Frame` + `FloatingLayer`，可居中、可加遮罩；保留/移除由程序在下一帧决定。

```go
modal := widgets.Modal{Title: "close session", Width: 22, Height: 6,
    Center: true, Backdrop: true, ParentWidth: 34, ParentHeight: 10,
    Rows: []widgets.FrameRow{{Text: "Close alpha?"}, {Text: ""},
        {Text: "[ Enter ] yes"}, {Text: "[ Esc ]   no"}}}
view := modal.Build().Build()
```

```


┌─ close session ────┐
│Close alpha?        │
│                    │
│[ Enter ] yes       │
│[ Esc ]   no        │
└────────────────────┘
```

### `Menu / MenuItem`

键盘菜单浮层：热键/禁用/分隔线，`Move`/`Hotkey` 改选中，`Value` 给程序派发。

```go
menu := widgets.Menu{Title: "actions", Width: 20, X: 6, Y: 1, Selected: 1,
    Items: []widgets.MenuItem{
        {ID: "copy", Label: "Copy", Hotkey: "c"},
        {Separator: true},
        {ID: "kill", Label: "Kill", Disabled: true},
    }}
view := sdk.Stack(base.Build(), menu.Build()).Build()
```

```
┌─ pane ───────────────────┐
│conte┌─ actions ────────┐ │
│     │  Copy            │ │
│     │▸ Paste           │ │
│     │───               │ │
│     │  Close           │ │
│     │  Kill            │ │
└─────└──────────────────┘─┘
```

### `RichText / Span`

一行内多段样式：相邻同一样式自动合并，`WrapText`/`WrapSpans` 做 CJK 安全换行。

```go
line := widgets.RichText{Width: 30, Spans: []widgets.Span{
    widgets.Styled("status ", "muted"),
    widgets.Styled("ok", "success"),
    widgets.Styled(" · 2 panes", "accent"),
}}
view := sdk.Col(line.Build()).Build()
```

```
status ok · 2 panes · 14:02
```

### `Scrollbar`

按比例的滚动指示器：thumb 大小 = `Visible*Length/Total`，`OffsetAt` 把点击位置折成 offset；纵横两用。

```go
bar := widgets.Scrollbar{Total: 100, Visible: 20, Offset: 30, Height: 6}
view := sdk.Col(bar.Build()).Build()
offset := bar.OffsetAt(clickY)
```

```
vertical
│
┃
│
│
│
│
horizontal
───────━━━━─────────────
```

## 交互

### `ContextMenu`

锚定在点击处的 `Menu`：`Open` 记录点击并把菜单夹回父矩形内，`Build` 输出定位浮层；程序决定点击是否关闭。

```go
cm := widgets.ContextMenu{Menu: widgets.Menu{Title: "row actions", Width: 20,
    Items: []widgets.MenuItem{{ID: "open", Label: "Open", Hotkey: "o"}}},
    ParentWidth: 30, ParentHeight: 9, Margin: 1}
cm.Open(mx, my)
view := sdk.Stack(base.Build(), cm.Build()).Build()
```

```
┌─ table ────────────────────┐
│row 1                       │
│row 2   ┌─ row actions ────┐│
│row 3   │▸ Open            ││
│        │  Copy            ││
│        │───               ││
│        │  Delete          ││
│        └──────────────────┘│
└────────────────────────────┘
```

### `HitRegion / Hit`

命中测试：矩形 + id，`Hit` 返回第一个包含 (x,y) 的区域。

```go
regions := []widgets.HitRegion{{ID: "ok", X: 2, Y: 1, W: 8, H: 1}}
if region, ok := widgets.Hit(regions, mx, my); ok {
    dispatch(region.ID)
}
```

### `ListRegions / TableRegions`

把 `List`/`Table` 的可见行翻译成命中区域；`RowAt` 由 y 反查行号（越界返回 false）。

```go
regions := widgets.ListRegions(list, originX, originY, width)
regions = widgets.TableRegions(table, originX, originY)
row, ok := list.RowAt(my - originY)
```

### `ApplyWheel / OnWheel / TableScroll`

滚轮增量 → 新 offset：`ApplyWheel` 是纯函数，`List.OnWheel`/`VirtualList.OnWheel`/`TableScroll` 是现成接线。

```go
offset = widgets.ApplyWheel(offset, total, visible, delta)
list.OnWheel(ev) // 就地更新 Offset
offset = widgets.TableScroll(offset, total, visible, delta)
```

### `Drag / SelectRange`

拖拽选择：`Begin`/`Update`/`End` 得到归一化矩形；`SelectRange`/`ListSelectRange` 把 y0..y1 折成行区间。

```go
var drag widgets.Drag
drag.Begin(x, y, "left")
drag.Update(x2, y2)
rect, ok := drag.End()
lo, hi, ok := widgets.ListSelectRange(list, y0, y1)
```

### `ClickTracker`

单/双/三击判定：同一位置在双击窗口内连续点击返回 1/2/3，超时或移位重新计数。

```go
var clicks widgets.ClickTracker
n := clicks.Click(x, y, at) // 1 单击、2 双击、3 三击
```

### `Hover`

悬停追踪：`Update` 吃鼠标位置返回命中 id，`StyleFor(id)` 只给被悬停的 id 叠加 hover 样式。

```go
var hover widgets.Hover
id := hover.Update(mx, my)
box.Style(hover.StyleFor(id))
```

## 表单

### `Form / Field`

字段列 + 焦点导航 + 聚合校验：`Next`/`Prev` 跳过禁用/只读字段，`Values`/`Validate` 读值与查错。

```go
form := widgets.Form{Width: 32, Focus: 0, Fields: []widgets.Field{
    {ID: "name", Label: "Name", Required: true,
        Input: &widgets.TextInput{Value: []rune("anytty"), Width: 28}},
    {ID: "email", Label: "Email", Help: "used for alerts",
        Input: &widgets.TextInput{Placeholder: "you@example.com", Width: 28}},
}}
view := sdk.Col(form.Build()).Build()
```

```
Name *
anytty
Email
you@example.com
used for alerts
```

### `Validator`

校验器是 `func(string) string`（返回错误文案，空串通过），可直接挂到 `Field.Validate`。

```go
field.Validate = widgets.All(
    widgets.Required("required"),
    widgets.MinLen(3, "too short"),
)
msg := field.Validate(field.Text())
```

| 校验器 | 语义 |
|---|---|
| `Required(msg)` | 去空白后为空则失败 |
| `MinLen(n, msg)` / `MaxLen(n, msg)` | rune 长度下限 / 上限 |
| `Pattern(re, msg)` | 正则必须匹配 |
| `Email(msg)` | 邮箱形状 |
| `IntRange(min, max, msg)` | 整数字面量且落在闭区间 |
| `OneOf(values, msg)` | 必须是枚举之一 |
| `All(...)` | 全部通过；返回第一个错误 |
| `Custom(fn)` | 自定义 `func(string) string` |
| `DayValidator(min, max, msg)` | `YYYY-MM-DD` 且落在日期区间 |

### `Select / Option`

下拉选择状态机 + 纯 `Build`：`Open` 时 `Build` 输出下拉列表，支持 typeahead、禁用项与占位符。

```go
select := widgets.Select{Label: "profile", Width: 22, Value: "dev",
    Options: []widgets.Option{{Value: "dev", Label: "development"},
        {Value: "prod", Label: "production"}}}
select.Open = true // Build 输出下拉列表
view := sdk.Col(select.Build()).Build()
```

```
profile
development ▾
open:
  development
▸ production
  legacy
```

### `Calendar`

Mon-first 月历（6 行 × 7 列）：`Month`/`Selected`/`Today`/`Marked` 都由调用方给，不读挂钟。

```go
cal := widgets.Calendar{
    Month:    widgets.Date{Year: 2026, Month: 9, Day: 1},
    Selected: widgets.Date{Year: 2026, Month: 9, Day: 18},
    Today:    widgets.Date{Year: 2026, Month: 9, Day: 22},
}
view := sdk.Col(cal.Build()).Build()
```

```
September 2026
Mo Tu We Th Fr Sa Su
31  1  2  3  4  5  6
 7  8  9 10 11 12 13
14 15 16 17 18 19 20
21 22 23 24 25 26 27
28 29 30  1  2  3  4
 5  6  7  8  9 10 11
```

### `Date`

纯日期值（年/月/日）：`AddDays`/`AddMonths` 进位并夹住月末，`ParseDate` 拒绝被 time 静默归一化的日期。

```go
date, err := widgets.ParseDate("2026-09-18")
next := date.AddMonths(1)
leap := widgets.Date{Year: 2024, Month: 1, Day: 31}.AddMonths(1)
```

日期运算示例（生成时计算）：

```
Date{2026, 1, 31}.AddMonths(1) → 2026-02-28
Date{2024, 1, 31}.AddMonths(1) → 2024-02-29
Date{2026, 9, 18}.AddDays(7)  → 2026-09-25
ParseDate("2024-02-30")       → error（非法日期）
```

### `DateField`

`TextInput` 加日期校验：`NewDateField` 自动挂 `DayValidator`，`Date()` 读回解析后的日期。

```go
input := &widgets.TextInput{Placeholder: "YYYY-MM-DD", Width: 28}
field := widgets.NewDateField("due", "Due date", input)
field.Min = widgets.Date{Year: 2026, Month: 1, Day: 1}
date, ok := field.Date()
```

## 反馈

### `ProgressBar`

定宽进度条：`Value`/`Max` + 可选 label/百分比，glyph 可换；没有计时器，进度由程序推进。

```go
bar := widgets.ProgressBar{Label: "upload ", Value: 7, Max: 10, Width: 20}
view := sdk.Col(bar.Build()).Build()
```

```
upload ██████████████░░░░░░70%
```

### `Spinner`

动画的一帧：`Frame` 由程序自己的 tick 推进（宿主没有 timer），默认盲文帧序列可整体替换。

```go
spinner := widgets.Spinner{Frame: 3, Label: " starting"}
view := sdk.Col(spinner.Build()).Build()
```

```
⠋ starting
⠙ starting
⠹ starting
⠸ starting
```

### `Badge`

小号状态标签（状态、计数、按键标记）：一个文本框 + 样式 token。

```go
badge := widgets.Badge{Text: "● live", Style: "ok", ID: "live"}
view := sdk.Row(badge.Build()).Build()
```

```
● live beta 2 panes
```

### `Tags / Tag`

一行标签：`Tag` 可带命中 `ID` 与样式，`Separator` 决定间隔；点击派发同 Button/Picker。（没有独立的 `Tabs` 组件，标签页请用 `TabBar`）

```go
tags := widgets.Tags{Separator: " · ", Items: []widgets.Tag{
    {Text: "go", ID: "t1"},
    {Text: "python", ID: "t2"},
}}
view := sdk.Col(tags.Build()).Build()
```

```
go · python · ts
```

## 图表

### `Sparkline`

一行八分块迷你图：`Min`/`Max` 可选，缺省从数据推导；超宽序列按 `Width` 分桶平均。

```go
spark := widgets.Sparkline{Values: []float64{1, 3, 2, 5, 4, 8, 6, 9, 7, 4, 6, 8}, Width: 16}
view := sdk.Col(spark.Build()).Build()
```

```
cpu 12 samples
▁▃▂▅▄▇▅█▆▄▅▇
```

### `BarChart`

柱状图：纵向用八分块子格（`Height` 行）或横向整块；`Max <= 0` 自动量程，`Labels` 可选。

```go
chart := widgets.BarChart{Values: []float64{3, 7, 5, 9, 6, 2}, Height: 5,
    Labels: []string{"mo", "tu", "we", "th", "fr", "sa"}, Width: 26}
view := sdk.Col(chart.Build()).Build()
```

```
            ███
    ▇       ███ ▃
    ███ ▆   ███ ███
▅   ███ ███ ███ ███ ▁
███ ███ ███ ███ ███ ███
mo  tu  we  th  fr  sa
```

### `Heatmap`

矩阵热力图：值可以是锯齿行，缺格按最低档渲染；`RowLabels`/`ColLabels` 可选。

```go
heat := widgets.Heatmap{RowLabels: []string{"am", "pm", "night"}, Values: [][]float64{
    {0, 1, 3, 6, 9, 7, 4, 2, 1, 0},
    {2, 4, 8, 9, 7, 5, 3, 2, 1, 0},
}}
view := sdk.Col(heat.Build()).Build()
```

```
am     ·░▒█▓░··
pm    ·░▓█▓▒░··
night ··░░▒▒▒░░·
```

### `Meter / Gauge`

浮点值的标签条：复用 `ProgressBar` 的字形，`ShowValue` 显示 `值/上限`；`Gauge` 是别名。

```go
meter := widgets.Meter{Label: "disk ", Value: 72, Max: 100, Width: 16, ShowValue: true}
view := sdk.Col(meter.Build()).Build()
```

```
disk ████████████░░░░72/100
```

### `Legend`

一行图例：每个 `LegendItem` 是彩色 marker + label，与图表配合解释颜色含义。

```go
legend := widgets.Legend{Items: []widgets.LegendItem{
    {Label: "ok", Color: "ok"},
    {Label: "dead", Color: "danger"},
}}
view := sdk.Col(legend.Build()).Build()
```

```
■ ok ■ warning ■ dead
```

## 主题 / 非视觉

### `Theme`

设计 token 集合：`DarkTheme()` 映射宿主 token（随宿主主题），`LightTheme()` 固定浅色；`ThemedXxx(theme)` 预填各组件。

```go
theme := widgets.DarkTheme()
list := widgets.ThemedList(theme)
table := widgets.ThemedTable(theme)
style := widgets.WithBold("fg:#6d3bd4")
```

可用 token 常量（来自 `clients/tui/sdk/widgets/tokens.go`）：

| 常量 | 值 |
|---|---|
| `StyleDefault` | `default` |
| `StyleBackground` | `bg` |
| `StyleFG` | `fg` |
| `StyleForeground` | `foreground` |
| `StyleStrongForeground` | `strong-foreground` |
| `StyleMuted` | `muted` |
| `StyleAccent` | `accent` |
| `StyleAccentDim` | `accent_dim` |
| `StyleSuccess` | `success` |
| `StyleOK` | `ok` |
| `StyleWarning` | `warning` |
| `StyleDanger` | `danger` |
| `StyleInfo` | `info` |
| `StyleChrome` | `chrome` |
| `StyleChromeFocus` | `chrome_focus` |
| `StyleHeader` | `header` |
| `StyleTabActive` | `tab_active` |
| `StyleTabInactive` | `tab_inactive` |
| `StyleFooter` | `footer` |
| `StyleFooterAccent` | `footer-accent` |
| `StyleStatus` | `status` |
| `StyleOverlay` | `overlay` |
| `StyleBorder` | `border` |
| `StyleBorderFocus` | `border_focus` |
| `StyleBorderDead` | `border_dead` |
| `StyleActiveBorder` | `active-border` |
| `StyleInactiveBorder` | `inactive-border` |
| `StyleSelection` | `selection` |

### `Format*`

数字/文本格式化纯函数：输出固定、无 ANSI、无 CJK 宽度逻辑，图表与状态栏直接复用。

```go
text := widgets.FormatBytes(1536)          // 1.5 KB
count := widgets.FormatCount(1234567)     // 1,234,567
uptime := widgets.FormatDuration(elapsed) // 1m30s
```

| 调用 | 输出 |
|---|---|
| `FormatBytes(1536)` | `1.5 KB` |
| `FormatCount(1234567)` | `1,234,567` |
| `FormatDuration(90*time.Second)` | `1m30s` |
| `FormatPercent(0.125, 1)` | `12.5%` |
| `FormatFloat(3.14159, 2, 6)` | `  3.14` |
| `ScaleValue(1500)` | `1.5K` |
| `PadLeft("42", 5)` | `   42` |
| `PadRight("42", 5)` | `42   ` |

### `Terminal`

终端内容源：`sdk.Terminal(sourceID)` 绑定宿主终端，渲染由宿主的真实 PTY 决定。

```go
view := sdk.Col(
    sdk.Terminal("terminal:local:alpha"),
).Build()
```

图鉴跳过 `sdk.Terminal`：它需要宿主提供真实的终端源（PTY），离线渲染只会得到空白内容。

