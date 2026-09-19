# 老 UI 像素复刻规格与缺口清单（LEGACY_PARITY）

> 目标：在 tui2 上复刻老默认 UI 的**逐格像素**与交互。参考来源：
> 老布局程序 `shell/main.go`（行号 `sh:N`）、老 surface 渲染器
> `tui/program/view.go`（`pv:N`）、老 terminal 组件
> `tui/program/components/terminal.go`（`tc:N`）、老主题
> `tui/render/style.go`（`st:N`）与 token→SGR 表
> `tui/render/result.go`（`rs:N`）。
> 实现：`clients/tui/examples/python-shell/legacy.py`（协议程序）+ `pb.py`（编解码）；
> 验证：`clients/tui/examples/python-shell/parity_test.py`（老公式独立重算 vs 程序 view）
> 与 `clients/tui/scripts/acceptance.sh`（真宿主机 tmux 抓屏 vs golden）。

## 0. 运行与验证

```sh
# 离线逐格对比（120x32 与 100x30，13 个场景），并刷新 golden
python3 clients/tui/examples/python-shell/parity_test.py

# 真宿主（交互复刻）
go build -o /tmp/tui2 ./clients/tui/cmd/tui2
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/legacy.py"

# 黑盒验收里的 legacy.py 段
bash clients/tui/scripts/acceptance.sh
```

## 1. 画布与布局公式

视口 `cols x rows` 由 HELLO/resize 给出。根盒子 `flow:stack`，一个 `flow:col`
主列（header 1 行 + body 弹性 + footer 1 行）+ `pos` 浮层（picker/help/prompt/
manager）+ toast（`sh:236-252`）。

### 1.1 header（`sh:166-185`，1 行）

| 顺序 | 文本 | 样式 token | 命中 id |
|---|---|---|---|
| 1 | `" WS main "`（固定字面量） | `header` | `ws` |
| 2 | 每个 tab：`" " + marker + (i+1) + " " + title + " "`，active 时 `marker="▎"`，否则 `" "` | active `accent`，否则默认 `foreground` | `tab:<i>` |
| 3 | 每个 tab：`"× "` | `muted` | `tabclose:<i>` |
| 4 | `" + "` | `muted` | `tabcreate` |

单 tab 120 列示例：`" WS main " + " ▎1 main " + "× " + " + "` → 行首 23 格有字。

### 1.2 body（`sh:187-226`）

- `body` 是 row：`[sidebar(30), panes 容器]`；sidebar 固定 30 宽（`sh:283`）。
- 水平分屏时 panes 容器 flow=row，分隔为 **1 格宽的 `"│"`**（muted，`sh:214`）。
  注意老 demo 的分隔节点没有高度，实际只画顶端 1 格；复刻保留该像素，同时把它
  做成整列命中盒以便拖拽（见 §3.4）。
- 竖直分屏（老 demo 的 body 恒为 row，`sh:187`；这是按老默认 TUI 语义补的扩展）：
  panes 容器 flow=col，分隔整行 `"─" * paneAreaWidth`（muted）。
- `paneWidths`（`sh:598-647`）：
  `avail = cols - sidebar(0/30) - (count-1)`；`out[i] = floor(avail*w[i]/Σw)`，
  每项至少 1，最后一项加余数 `avail-used`（仍至少 1）。
- 竖直方向用同一公式对 `avail = rows-2-(count-1)` 取 pane 高度。

### 1.3 sidebar（`sh:258-285`）

- 唯一子盒 `status`：宽 30、高 6（4 行内容 + 上下边框），边框与标题 token `muted`。
- 内容 4 行（`sh:272-277`，均为 muted）：
  `"tabs      " + N` / `"panes     " + N` / `"mode      " + mode` / world 行。
- world 行：老宿主由 `world` 事件给出 `"host panes %d · endpoints %d · focus %s"`
  （`sh:687`）；tui2 协议无该事件，程序由 `sources`（endpoint/attached）与
  `HELLO.view_id` 派生同形文案（见 §5-3）；首个 sources 事件之前显示
  `"host      (no world yet)"`（`sh:269-271`）。

### 1.4 pane（老源码公式推导）

- 本地 pane（无终端）：整框由程序自绘（内核无 border 概念）。
  - 标题串 `marker + pane.Title`：focused 为 `"▎ "`（含尾空格），否则 `"  "`（`sh:216-219`）。
  - 边框样式：focused `accent`，否则 `muted`（整框含标题同一 token，`sh:222`）。
  - 内容行 `window(lines, scroll) = lines[scroll:]`（`sh:436-444`），样式 `muted`，
    从内容区左上角起绘制，超出宽高被裁掉（`pv:422-442`）。
- 终端 pane（绑定 source）：边框/标题/角标由 terminal 组件自绘，程序用
  `content.props`（`chrome.*`）把颜色锁到老 token（组件规则 `tc:107-179`）：
  | 状态 | 边框/标题/角标 token |
  |---|---|
  | exited（优先，`tc:171-178`） | `warning`，标题追加 `" [exited N]"`（N=0 时为 `" [exited]"`，`tc:52-60`） |
  | focused | `accent`，标题前加 `"▎"`（无尾空格，`tc:119-124`） |
  | 其它 | `muted` |
  | 回看 | 标题追加 `" [↑N]"`（`tc:43-49`），样式随上面三态 |
- 组件仅在 `W>=3 && H>=3` 时画边框（`tc:82`，tui2 `DefaultInset` 同界）。

### 1.5 footer（`sh:465-474`）

- 单行 `" " + hints + padding + "ws:<tab> tabs:N panes:N "`；
  `padding = cols - displayWidth(left) - displayWidth(right)`，最小 1。
- hints 按层级（`sh:447-463`，字面量见 parity_test.py）：
  回看 `[PgUp/PgDn] SCROLL +N │ [y] COPY │ [esc] LIVE`；聚焦终端 exited
  `[Ctrl+E] RESTART │ [Ctrl+F] PICKER │ terminal exited`；PANE / RESIZE /
  GLOBAL / NORMAL 各一组。样式 token `footer`（= muted 前景，**不带 dim**）。
- **推荐 preset（M22）**：footer 切换到 v2 场景规格（`RECOMMENDED_CONFIG.zh-CN.md`
  §3.1），由 `legacy.py --footer-lines` 输出、`parity_test.py`
  `check_recommended_footer` 与 Go shell 共用
  `golden/recommended_footer_120x32.txt` 逐字符比对；默认（不带 preset）不受影响。

### 1.6 浮层（`sh:326-401`，均由程序自绘并 `pos` 居中）

| 浮层 | 尺寸 | 居中公式 | 内容 |
|---|---|---|---|
| picker | 56×12 | `center(w,h)`（`sh:416-434`） | 每项 `marker + item`（选中 `▸ ` accent，否则 `  ` muted）+ 提示行 |
| help | 64×(9+4)=64×13 | 同上 | `Help`(accent) + 9 行正文(foreground) + `esc close`(muted) |
| prompt | 56×(matches+5) | 同上 | `": " + prompt`(accent，行内光标) + matches(`▸ ` accent/muted) + 提示行(muted) |
| manager | 60×15 | 同上 | 13 行 `terminal-%03d   %-8s  local`(foreground)，`i%5==4 → exited` |
| toast | 1 行 | x=1, y=rows-2（`sh:404-413`） | `" " + toast + "  ·  any key dismiss"`(warning) |

`center`：`w=min(w,cols); h=min(h,rows); x=(cols-w)//2; y=(rows-h)//2`（负值取 0）。
picker 列表（`sh:476-495`）：`marker + " local  " + (title or terminal_id)`，
`✕` exited / `●` attached / `○` 其它；最后固定 `+ New terminal`。

### 1.7 边框与标题嵌入（`pv:422-442`, `pv:499-506`）

- 顶边 `"┌" + "─"*(W-2) + "┐"`；把 `" " + title + " "` 从**下标 1 起**覆盖；
  若 `len(label)+2 > W`（按 rune 数）则**不嵌标题**。
- 中间行 `"│" + 空格 + "│"`，内容文字覆盖在内容矩形（内缩 1）之上。
- 底边 `"└" + "─"*(W-2) + "┘"`。
- 浮层/本地 pane 的每一行由程序显式画出：两个边框格用边框 token，中间文字用
  自己的 token（等价于老的“先边框帧、后内容帧”覆盖顺序）。

## 2. 样式规格（token → 显式样式串）

老主题 `DefaultTheme`（`st:46-75`）：`HostFG/ChromeFG #dedbe6`、`Accent #a970ff`、
`Muted #b8b1c4`、`Warning #f0c45c`、`StatusFG #e7e2ef`、`StatusBG #08080d`。
token→SGR 见 `rs:427+`；`mixHostColor` 为四舍五入线性混色（`st:258-274`）。

| token | 老 SGR | legacy.py 显式样式串 |
|---|---|---|
| 默认/`foreground` | fg ChromeFG | `fg:#dedbe6` |
| `accent` | fg Accent + bold | `fg:#a970ff;bold` |
| `muted` | fg Muted + dim | `fg:#b8b1c4;dim` |
| `footer` | fg Muted | `fg:#b8b1c4` |
| `warning` | fg Warning | `fg:#f0c45c` |
| `header` | headerWorkspaceFG/BG + bold | `fg:#d4c0f4;bg:#3c2e55;bold` |

`header` 的混色推导（供核对）：
`headerChromeAltBG = mix(#08080d,#e7e2ef,.08) = #1a191f`；
`BG = mix(#1a191f,#a970ff,.24) = #3c2e55`；
`FG = mix(#e7e2ef,#a970ff,.30) = #d4c0f4`（`rs:595-601`）。

## 3. 交互规格（`legacy.py` 实现，均对照 `shell/main.go` 行号）

1. **tab**：点击 `tab:<i>` 切换（`sh:736-741`）、`tabclose:<i>` 仅当 >1 个 tab
   （`sh:883-891`）、`tabcreate` 新 tab（标题 `tab<seq>`，从 2 起，`sh:876-881`）、
   `shift? 数字键`：NORMAL/PANE 下 `1..9` 切 tab（`sh:1011-1014`）。
2. **pane focus**：点击 pane 盒（id=程序侧 `p1..`）切换 focus（`sh:769-777`）；
   `Tab` 循环（`sh:1008-1010`）。
3. **split**：`%` → 追加 `title+" copy"`、内容 `"split row"` 的 pane 并聚焦
   （`sh:893-903`）；`"` 同上但 flow=col（扩展）。权重各 1。
4. **drag**：press 命中 `divider:<i>` 开始捕获（tui2 隐式捕获：非终端 +
   `input:["mouse"]`，PROTOCOL §6.7），drag 用 `x/y` 绝对坐标；
   `left = clamp(x-sidebar, 2, avail-2)`，权重 `[left, avail-left]`（`sh:781-796`）；
   release 结束。竖直方向 `top = clamp(y-2, 2, avail-2)`。
5. **close**：`x` 删除聚焦 pane（>1 时；仅解绑式，终端仍在 sources/picker，
   `sh:905-914`）。
6. **picker**：`Ctrl-F`/点 `ws` 打开（`sh:742-744`）；`↑↓`/enter/点击行
   （`sh:1039-1049`, `sh:745-748`）；enter/click 对 exited 发 `terminal.restart`，
   对 live 发 `terminal.attach`，最后一项 `terminal.create`（`sh:798-841`）。
7. **prompt**：`:` 打开（`sh:1020-1024`），输入/退格/`↑↓`/enter 过滤并执行
   （`sh:1067-1089`, `sh:843-874`），行内光标位于 `": "+prompt` 之后
   （`sh:363-366`）。
8. **help**：`?` 打开，`?`/`esc` 关闭（`sh:1050-1053`）。
9. **esc 层级**：浮层 esc 先关浮层；PANE 回看态 esc=回 live；再 esc 退 PANE；
   NORMAL esc 清 toast/mode（`sh:945-950`, `sh:989-991`）。
10. **footer 随模式**：NORMAL/RESIZE/GLOBAL/PANE/回看/exited 六组（§1.5）。
11. **退出角标 / Ctrl-E**：exited 聚焦时角标由组件画；`Ctrl-E` 发
    `terminal.restart`（`sh:999-1007`）。
12. **回看**：PANE 下 `PgUp/PgDn` 对聚焦终端发 `terminal.scroll(±10)`
    （`sh:951-958`）；本地 pane 为 `scroll = clamp(scroll-delta, 0, len(lines))`
    （`sh:697-705`）；滚轮同理（终端 → scroll，本地 → 行窗口）。`y` 发
    `terminal.copy`（`sh:959-960`, `sh:552-563`）。
13. **toast**：`notice` → `"level: message"`；任意键清除（`sh:688-694`,
    `sh:917-919`）。

## 4. 像素验证矩阵

`parity_test.py` 对 120x32 与 100x30 各跑 14 个场景，逐格比较
（字符全比；样式只比非空格，空格样式不可见）：

| 场景 | 覆盖点 | golden |
|---|---|---|
| `cold_start_picker` | header/sidebar/footer/边框 + picker 居中 | ✅ |
| `single_pane` | 静态单 pane | ✅ |
| `split_row` | 44/45 分宽、1 格分隔、焦点标记、PANE footer | ✅ |
| `split_col` | 14/15 分高（120x32）、竖分隔 | ✅ |
| `help` | help 尺寸/文案/样式 | ✅ |
| `prompt_empty` / `prompt_filter` | prompt 高度 17/6、光标列 | ✅ |
| `tabs_two` | header 两 tab、active 样式 | ✅ |
| `picker_source` | `●`/`+ New terminal` 标记 | ✅ |
| `exited_badge` / `exited_zero` | `[exited 7]` / `[exited]`（零码） | ✅ |
| `scrollback` | `[↑10]` + SCROLL footer | ✅ |
| `sidebar_off` | Ctrl-W 后 pane 占满全宽 | ✅ |
| `toast` | 底部 toast 行 | ✅ |

真宿主对照：`acceptance.sh` 的 legacy 段把 tmux 抓屏按行与上述 golden 比较
（120x32），并用 `display-message` 断言硬件光标 `(35,8)`（空 prompt）与
`(37,14)`（输入 `he` 后）；acceptance.sh 共新增 28 项 legacy 断言。

## 5. 缺口清单（缺口 → 影响 → 补法）

| # | 缺口 | 影响 | 补法（状态） |
|---|---|---|---|
| 1 | 终端组件退出码 0 显示 `[exited 0]`，老组件显示 `[exited]`（`tc:52-60`） | exit 0 的退出角标像素不一致 | `components/terminal/render.go` `titleSegments` 改为 0 码裸 `[exited]`；`TestTitleTextBadges` 增加“零码+回看”组合用例（已修） |
| 2 | Python 参考绑定 `pb.py` 不编码 `Box.cursor`（proto field 7） | prompt `": "` 的程序光标无法复投 → 老 UI 缺输入光标 | `pb.encode_box` 增加 cursor（`visible` 省略即默认可见）；parity prompt 场景断言光标坐标、tmux 断言硬件光标（已修） |
| 3 | 协议无 `world` 事件（老宿主 `sh:686-687` 推送 panes/endpoints/focus） | sidebar 第 4 行拿不到宿主拓扑 | 程序由 `sources`（endpoint/attached）+ `HELLO.view_id` 派生同形文案（已定，见 §6-1） |
| 4 | 老 `workbench.command`（`:kill pane`，`sh:854`）不在 tui2 §4 方法表 | `:kill pane` 不可执行 | 映射为 `terminal.kill`（宿主确认后同路径）；差异记录（已定） |
| 5 | 老 demo 的 `"` 竖分不生效（body 恒 row），无竖向布局 | M4 的“竖分”场景无法成立 | `legacy.py` 按 tab.flow 实现竖堆叠 + 竖分隔（扩展）；`split_col` 场景逐格验证（已补） |
| 6 | 绑定 RESPONSE 可能先于 `sources` 快照到达 | 新绑定 pane 会退化成自绘本地 pane（不可聚焦、多一帧错误边框） | `legacy.py` 对“已绑定但快照未到”的 pane 直接按组件 chrome 渲染（标题回落短 id），焦点/按键立即生效（已修） |

审计结论（复刻中实际用到、tui2 已具备的能力，均非本轮缺口）：
程序光标 wire→合成（`runtime/compositor_test.go:122`）、overlay z 序与不透明
（`compositor.go:88-99`）、鼠标捕获与 drag 定向（`session.go:600-619`、
`cmd/clients/tui/host.go:380-405`）、历史取数（`terminal.go:430-451`）、显式样式
（`render/style.go:293-390`）、命中 id 规则（`kernel/frame.go:88-124`）。

## 6. 差异说明（架构差异，需用户拍板）

| # | 差异 | 老行为 | clients/tui/复刻 | 建议 |
|---|---|---|---|---|
| 1 | `world` 事件 | 宿主推 `host panes N · endpoints N · focus X` | 协议无此事件；legacy.py 由 sources+view_id 派生 | 如要真值（宿主 pane 数/focus），需新增 EVENT 字段；否则维持派生 |
| 2 | 全局键 `1..9` `:` `?` | 程序总收到事件，先于终端处理 | 聚焦终端且未 claim 的键进 PTY；这些入口只在无焦点/PANE/浮层下生效 | 保持 claim 语义（否则终端无法输入数字）；如需全局直通可改用双击 `input.forward` |
| 3 | 退出角标回看并存 | 老组件 `[exited N]` 后接 `[↑N]` | 同序；零码已按老组件改裸 `[exited]` | 已对齐 |
| 4 | 宽字符标题嵌入 | `embedTitle` 按 rune 数覆盖，宽字符会“多吃”格 | 复刻保留 rune 规则；组件路径用显示宽截断（`tc:122`） | 老行为本身不严谨；如要求显示宽一致需拍板 |
| 5 | 宿主默认样式 | 未写单元格 = 终端默认；显式文字才能量到色 | 一致（legacy.py 所有文字都带显式样式） | 无需动作 |
| 6 | RESIZE/GLOBAL 模式 | 与 NORMAL 同分支（`sh:982-1034`），仅有 footer 提示 | 复刻一致（无额外热键） | 无需动作 |
| 7 | `y` 复制反馈 | 老程序静默（`sh:552-563`） | legacy.py 在 RESPONSE ok 后弹 `copied visible screen` toast（可观察性） | 如要严格静默可删掉 `copy_visible` 回调 |

## 7. 已知不逐字节对齐的项

- 宿主把相邻同样式 run 合并、SGR 只发差量（例如 bold 继承），ANSI 字节流与老
  renderer 不逐字节相同；**可见 cell（字符+颜色/属性）一致**。
- tmux `capture-pane` 会去掉行尾空格，golden 与抓屏都按 `rstrip` 比较。
- picker/help/prompt 浮层内部空格样式：老的边框帧给空格带 muted/accent 前景，
  新合成器清成默认空格；空格无背景、前景不可见，逐格验证只比较非空格样式。
