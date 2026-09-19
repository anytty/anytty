# 老 v3 TUI 像素复刻规格与缺口（V3_PARITY）

> 目标样式：`clients/tui/examples/python-shell/1.txt`（用户实际使用的界面抓屏）。
> 绘制真相：git HEAD 的 `tui/app` + `tui/render`（surface framework），样式/图标/键位来自
> `tui/docs/tui-v3.recommended.yaml`（profile `coralline-candy`）。
> 实现：`clients/tui/examples/python-shell/v3ui.py`（纯 stdlib，只依赖 `pb.py` 的 v2 线协议）。
> 验证：`clients/tui/examples/python-shell/v3_parity_test.py` + `golden/v3_*.txt`。

## 0. 样本测量

`1.txt` 实际是 **181 列 × 56 行**（`┌` 在第 0 列、`┐` 在第 180 列；y=0 顶条，
y=1..54 窗口，y=55 底栏）。任务要求的 golden 固定 **120×32**（与 legacy golden 同尺寸），
因此除宽度相关的补齐/裁剪外其余元素逐字符一致。底部 `󰙅 main 󰹙 0  11` 的 `11`
是终端池计数（terminals summary），不是 pane 数；1.txt 的内容区是 OpenCode 的 live 画面。

## 1. 元素规格（文本 → glyph → 样式 token → 位置/公式 → 交互）

样式全部来自 recommended yaml 的显式颜色；下表 `#RRGGBB` 即程序发出的
`fg:/bg:` 样式串（例如 `ST_TAB_BODY = "fg:#1b1230;bg:#f0abfc;bold"`），不依赖宿主主题。

### 1.1 顶部条（header，y=0，高 1）

| # | 元素 | 文本 | 样式（token→显式） | 位置/公式 | 交互 |
|---|---|---|---|---|---|
| 1 | workspace 左圆角 | `\ue0b6` | fg `#f0abfc` bg `#070611` | x=0 | workbench / help |
| 2 | workspace 主体 | ` 󰙅 {workspace} ` | fg `#1b1230` bg `#f0abfc` bold | 紧随 | 同上 |
| 3 | workspace 右箭头 | `\ue0b0` | fg `#f0abfc` bg `#3b2f63` | 紧随 | 同上 |
| 4 | active tab | `\ue0b0 ⎇ {title} · T{index} \ue0b0 󰅖 ` | 箭头 fg `#3b2f63`、主体 fg `#1b1230` bg `#f0abfc` bold、尾部 fg `#f0abfc`；close fg `#fb7185` | title `truncate 14`；index 从 1 起 | tab.select / tab.close |
| 5 | inactive tab | `\ue0b0 ⎇ {title} · T{index} \ue0b0 󰅖 ` | 箭头/尾 fg `#3b2f63`/`#261b44`、主体 fg `#c4b5fd` bg `#261b44` bold；close fg `#9ca3c9` | 同模板 | 同上 |
| 6 | 新建 tab | `\ue0b0 󰐕 \ue0b4` | fg `#3b2f63/#1b1230` bg `#fde68a`；尾 `#fde68a` bg `#070611` | 紧随 | tab.create |
| 7 | 右侧 notice | ` ! {notice} ` | warning `#fde68a` | 右贴边，其余用 bg `#070611` 填充 | 无 |

模板逐字来自 yaml `chrome.workspace_template` / `tab_template` / `tab_create_template`
（`\ue0b0`=U+E0B0、`\ue0b6`=U+E0B6、`\ue0b4`=U+E0B4；`⎇`=U+2387）。`index` 是
**tab 序号**；`title` 是 tab 标题（不是终端标题）。

### 1.2 窗口框（pane card，`panel_presentation: card`）

老 v3 每个 pane 是一张完整方框（`renderCardPanel` → `drawStyledPaneFrame` +
`renderPaneChromePrimitive`），推荐 profile 就是 card。框线 `┌┐└┘─│`，active frame fg
`#f0abfc`，非 active fg `#3b2f63`。复刻按 1.txt 的实际结构把**一个 tab 的 card
当作"窗口"**，card 的内容区再横向分成多个 pane 内容区（见 §3）。

顶部题字行（宽 `w`）：

- `innerLeft = 2`，`innerRight = w-1`。
- 动作组宽 `actionWidth = 2 + 3·n`（`\ue0b6` + 每项 ` glyph ` + `\ue0b4`），
  默认 `n=4`：`zoom 󰁌`、`split_vertical `、`split_horizontal `、`close 󰅖`；
  宽度不够时（`actionWidth > w-6`）只留 `close`（`≤ w-5`），再不够整组隐藏。
- `actionX = innerRight - actionWidth - 1`，`rightLimit = actionX - 1`（无动作时 = innerRight）。
- 左侧：lock ` □ `（3 cell；`CanLockSize` 时可点，锁定 `■`）→ 标题槽
  ` {terminal}@{endpoint} `（`pane_title_template`，`terminal` 截 18）→ 标题槽 advance
  吃掉剩余宽度，于是标题与右侧状态之间的缺口是**框线 `─`**。
- 右侧状态槽：state 固定宽 3 ` ● `（active→success `#86efac`，非 active→muted
  `#9ca3c9`；退出 `×` danger；折叠 `▾`）；attach 计数固定宽 4 `" x1 "`；owner 槽
  固定宽 ≥8 `" owner  "`（projected owner→success，pending→warning `owner?`，否则
  `follow`），owner 文本 fg `#f0abfc`。
- 181 宽样本顶部行：`┌` `─` ` □ ` ` anytty-surface@hs ` + 124×`─` +
  ` ● ` ` x1 ` ` owner  ` + `─` + `\ue0b6 󰁌      󰅖 \ue0b4` + `─┐`。动作项之间
  是**两个空格**（每项自带左右空格、separator 为空）。
- 动作组配色：active `\ue0b6` fg `#f0abfc`；item fg `#1b1230` bg `#f0abfc` bold；
  `\ue0b4` fg `#f0abfc`；非 active 换成 `#3b2f63` / `#c4b5fd` / `#3b2f63`。
- 交互：动作项各自命中（`pane:{id}:zoom/split-h/split-v/close`，命中区 = 含两侧空格的
  3 cell）；lock `pane:{id}:lock`；owner `pane:{id}:take-owner`；框/内容命中聚焦该 pane。

内容区 = rect 内缩 1。老 v3 的内容是 terminal 组件的 live 画面；复刻里由 **v2 terminal
组件**填充，并下发 `chrome.inset=0`（§4 缺口 A），组件不再重复画边框，PTY 尺寸 = 内容矩形。

### 1.3 pane 内容提示行（"Click to collapse"）

1.txt 的 `┃  Click to collapse` 是其终端内容（OpenCode）自己画的。复刻对**无终端的
占位 pane** 画同样的提示（内容第 1 行 `  ┃`、第 2 行 `  ┃  Click to collapse`），提示行
命中后切换该 pane 折叠（折叠后只留题字条状态槽的 `▾`，`z` 再展开）。提示行盒子宽度
取文本实际宽度，保证命中测试优先于整块内容盒子。

### 1.4 底部状态栏（footer，y=rows-1，高 1）

- 左：mode badge ` {mode_icon} {mode_label} `（live = ` 󰌌 CTRL `）。badge 样式取
  老 yaml `footer.modes` 的 per-scene token：live `footer-accent`、pane
  `footer-key-pane`、resize `footer-key-resize`、terminal-picker
  `footer-key-picker`、copy `footer-key-copy`；未列出的场景回落 `footer-accent`。
- 动作组：每个动作 = `" " + label`，之间 `" · "`（yaml `separator`；`key` 模板为空，
  label 内嵌键与图标）。live 顺序 = `P  PANE` `R 󰙖 SIZE` `O 󰹹 FLOAT` `T 󰓩 TAB`
  `W 󰙅 WORKSPACE` `F 󰱼 PICK` `⇧C 󰒉 SELECT` `⇧H 󰅌 CLIPBOARD` `⇧V 󰆏 PASTE` `G 󰒓 SYSTEM`。
- **每键不同颜色**（这是本轮修复的第二个问题）。老 renderer 的解析链是：
  1. yaml `shortcuts.actions.<id>.style` 显式 token；
  2. 无显式样式时 `shortcutActionStyle()` 给 `status-accent`（破坏性 id 给
     `status-warning`），`footerActionDisplayStyle()` 再把这两种状态色送进
     `footerActionKeyStyle(key, label)` 启发式（key 是 `ShortcutKeyDisplay()`
     的展示串，如 `^P`/`←`/`Esc`/`enter`）。
  复刻把这条链逐字实现为 `footer_action_style()` + `footer_fallback_style()`。
  coralline-candy 下各 token 的实际 RGB（`ansiForStyleToken()` +
  `mixHostColor()`，全部 bold 除注明外）：

  | token | 色值 | 说明 |
  |---|---|---|
  | `footer-accent` | `#f0abfc` bold | LIVE badge、右侧 float 计数、无匹配启发式 |
  | `footer-key-pane` | `#f0abfc` bold | P/PANE、PANE badge |
  | `footer-key-resize` | `#fde68a` bold | R/SIZE、HSPLIT、SIZE badge、`= 󰕕 BALANCE` 之外的 R 类 |
  | `footer-key-tab` | `#7dd3fc` bold | T/TAB、NEXT、SELECT/ATTACH |
  | `footer-key-workspace` | `#86efac` bold | W/WORKSPACE |
  | `footer-key-float` | mix(accent,info,.45)=`#bcbdfc` bold | O/FLOAT、DOWN/LOCK/LAYOUT |
  | `footer-key-copy` | mix(success,info,.35)=`#83e5c8` bold | ⇧H/CLIPBOARD、VSPLIT |
  | `footer-key-picker` | mix(danger,warning,.35)=`#fc9a87` bold | F/PICK、X/CLOSE、PICK badge |
  | `footer-key-global` | mix(accent,warning,.35)=`#f5c0d4` bold | G/SYSTEM、L→RIGHT |
  | `info` | `#7dd3fc` 常规 | ⇧V/PASTE（yaml 显式） |
  | `success` | `#86efac` 常规 | yaml 显式场景 |
  | `danger` / `danger-strong` | `#fb7185` / `#fb7185` bold | X CLOSE / Q KILL+CLOSE |

  NORMAL 场景 120 列的键组颜色（考古表，golden 逐块断言）：
  `P→pane`、`R→resize`、`O→float`、`T→tab`、`W→workspace`、`F→picker`、
  `⇧C SELECT→tab`（键展示 `^C` 无 R，label 先命中 T）、`G→global`；
  181 列补 `⇧H→copy`、`⇧V→info`。PANE：`X→danger`、`VSPLIT→copy`、
  `HSPLIT→resize`（label 自带 CTRL 命中 R）、`FOCUS→picker`、
  `Q→danger-strong`。PICKER：`ENDPOINT→float`、`SELECT/ATTACH→tab`、
  `ESC BACK→accent`。
- 右：summary ` 󰙅 {workspace}` + ` 󰹹 {float}` + ` {term_icon} {terminals}` + 尾空格。
- 宽度规则（shell_bar.go 逐条移植）：`width ≥ 120` 时右段预留全宽，
  动作按 `available = width - reserved` 用 `selectFooterActionTokens` 选择
  （超限时 tail = 最后一个带键动作收尾）；compose 先 `trimBarSegments`（丢
  priority 数字最大的段：尾空格 4 > terminals 4 > ws 2 > float 1）再 pad。
- 其他场景（pane/resize/tab/workspace/system/floating/picker/prompt/copy）的
  badge/动作按 recommended yaml 同名 scene 的 `show: true` 项逐条映射。

### 1.5 floating 窗口

老 v3 floating = `drawStyledBox` 全方框 + 标题槽（lock + `title@endpoint`，与 pane 相同
槽位公式）+ 右侧动作 `◎ ▾ ↗ 󰅖`（center/collapse/zoom/close）+ 内容 inset 1；
active fg `#f0abfc`，非 active fg `#3b2f63`。复刻（真实可用，非仅 chrome）：

- **新建**：`Ctrl-O` 进 floating 场景，`n` 新建。默认几何沿用老 v3 公式
  （`w = clamp(cols*4/5, min(64, cols-8), min(112, cols-4))`、
  `h = clamp(rows*3/4, min(18, rows-4), min(32, rows-2))`，居中 + 多窗按 4/1 级联）。
  宿主模式下新建后立刻 `terminal.create` 并把结果绑到浮窗 pane（先空亦可，picker
  `Ctrl-F` 在 floating 场景会把选中/新建的终端绑到**浮窗** pane，而不是主 pane）。
- **内容**：`content.self` + `chrome.inset=0`，pos = 浮窗内缩 1 的矩形，
  PTY 尺寸 = 浮窗内容矩形（宽-2 × 高-2），随拖移/resize 跟随。
- **拖移**：标题整行（lock/标题/右侧框线）都是 `float:{id}:title` 命中区，
  宿主按下即隐式捕获（`input:["mouse"]` 的非 terminal 盒子），drag 事件回程序
  按增量移动并 clamp 在 body 内；release 释放。
- **折叠/展开**：`z`/`m` 折叠后**只保留标题行**（动作组仍在，可点 `▾` 恢复），
  展开时提升到最上层并重新聚焦；折叠期间不发内容盒子（PTY 保留原尺寸）。
- **关闭**：`x` 关闭 = 解绑视图（终端留在池中，老 v3 `floating.close` 语义）；
  `q`/kill 才杀终端。
- **z 序**：浮窗全部画在普通帧之上；`self.floatings` 列表尾 = 最高层，
  激活/新建/拖拽/召唤都会把该窗移到列表尾；焦点浮窗永远最高。
- **焦点与输入**：`Esc`/`Ctrl-O` 回到 live 后 active 浮窗的 terminal 盒子
  `focused=true`，按键只进该 PTY；点主 pane 内容清除 active 浮窗并把焦点还给
  pane。`Ctrl-O 1..9` 召唤对应浮窗（展开 + 提升 + 聚焦）。
- 主 pane 的 `Ctrl-O n` 场景动作组（footer）为 `N/O/F/X/Z`，与老 yaml 一致。

### 1.6 分屏（递归切分树，真实 pane）

一个 tab = 一棵**递归切分树**：

- **树模型**：`Leaf(pane)` | `Split(orient, ratio, a, b)`；`orient="row"` 是左右
  并排（老 `panel.split_right`，键 `%`/`Ctrl-D`），`"col"` 是上下堆叠（老
  `panel.split_down`，键 `"`/`Ctrl-E`）。`ratio` 是 `a` 在可用尺寸中的占比
  （可用 = 该节点 rect − 1 格分隔条 − 两张 card 各 2 列/行边框）。
- **只切聚焦叶子**：split 把聚焦 `Leaf` 原地替换为 `Split(原叶子, 新叶子)`
  （row → 新叶子在右；col → 新叶子在下），兄弟节点与其它 Split 的 rect/ratio
  完全不动。于是“先左右分、再对右侧上下分”得到左 1 右 2 的任意矩形排布，
  而不是全局重排。
- **每个叶子一张完整 card**：每 leaf 自绘完整边框 `┌┐└┘─│` + 题字条
  （lock/标题/状态槽/`\ue0b6 󰁌      󰅖 \ue0b4` 动作组），几何/样式与 §1.2
  的单窗口 card 相同。聚焦 leaf 用 accent 边框/标题，其余用面板色；每个 leaf
  的内容区 = card rect 内缩 1，PTY 尺寸 = 内容矩形。
- **分隔条**：兄弟间恰好 1 格（row 画 `│`、col 画 `─`），命中 id
  `divider:{tab}:{split_seq}`；拖拽只把落点写回**该 Split 的 ratio**，不触碰
  其它节点；`Ctrl-R` SIZE 场景的 `h/l/k/j` 找聚焦 leaf 最近的同轴祖先 Split
  做 ±2；`r`/`=` 把所有 Split 的 ratio 重置为 0.5。外部 resize 后按各节点
  ratio 重新分配，比例保持。
- **关闭**：删除该 `Leaf`；父 `Split` 只剩一边时用兄弟子树替换父节点
  （逐级提升）。最后一个叶子关闭后回到 `empty` 占位叶子。
- **折叠**：leaf 折叠后从布局移除、空间归兄弟；全部折叠时只留聚焦 card 的
  题字条（`▾`）。再 `z` 恢复。
- **独立终端**：split 出的新 leaf 在宿主模式立刻 `terminal.create` 并绑定；
  每个 leaf 是独立 terminal 组件（独立 PTY、独立 `content.self`）。
- **焦点**：live 场景下恰有一个 leaf 的盒子 `focused=true`（浮窗 active 时
  焦点归浮窗）；点 leaf 的框/内容或 PANE 场景 `h/l`/方向键按树的中序切换；
  输入只进聚焦 leaf。

### 1.7 overlay

picker/help/prompt/clipboard 是居中方框（square box）overlay：标题嵌在顶边 x+2、
内容 inset 1、背景 `#261b44`，mode 切换 footer badge/动作组（picker badge `󰱼 PICK`）。
picker 行 `● {endpoint}  {label}` / `+ New terminal`，↑↓/单击选择、enter 绑定、esc 关闭。

## 2. v2 映射

| 老 v3 概念 | v2 表达 |
|---|---|
| 整屏 chrome（header/footer/card frame/title/actions） | 程序侧显式文本盒子 + `style` 显式样式串，`pos` 绝对定位 |
| pane/floating 内容 | `content.self = terminal:<endpoint>:<id>`，props 下发 `chrome.*`（本复刻用 `chrome.inset=0`） |
| 动作命中（pane 按钮/tab 按钮/footer/overlay 行） | 文本/内容盒子 `input:["mouse"]` + node id；宿主命中后 EVENT 回程序 |
| owner/attach 徽标数据 | `sources` 快照：`resize_owner`、`owner_epoch`、`attached`、`terminal_id`；本地绑定超时为 1 |
| tab 编号/计数、workspace 名 | 程序状态（v2 无 workspace 协议概念；复刻默认 `main`） |
| 终端池计数 | sources 快照里 kind=terminal 的条数 |
| 拖拽移动 | 鼠标 `drag` 的 x/y + 程序侧 implicit capture（node id 记忆） |
| 键位（`ctrl-shift-c/v/h`、`ctrl-alt-1..5`） | `keys.claim` + EVENT key 名；缺口 B（§4）补齐后精确匹配 |

## 3. 1.txt → 复刻状态映射（golden 状态）

- workspace = `main`；tabs = `auto-push`(T1, active) / `local`(T2)；
- active tab 是一棵 row Split 两 leaf：左 `anytty-surface@hs`（focused、projected
  owner，内容行含 `┃`/"Click to collapse"）+ 分隔条 + 右 `opencode@hs`
  （unfocused、follow），每个 leaf 各自一张完整 card（修复了“多 pane 共享一张
  外框”的旧偏差）；
- demo source 池 = 11 个终端（2 个已绑定，左 source 带 `resize_owner`）；
- footer = live，右段 ` 󰙅 main 󰹹 0  11`；
- 尺寸 120×32（golden）与 181×56（对照 1.txt 行）。

## 4. 缺口清单（M3 补齐的 Go 侧能力，全部已实现并有单测）

| # | 缺口 | 老 v3 需要 | v2 原状 | 补齐方式 |
|---|---|---|---|---|
| A | **terminal 组件无法关边框**：`content.props` 没有 inset 通道，`Inset()` 永远回落 1；card 复刻会出现"框中框"，PTY 也多缩 2 | card pane 只画自己的框，内容区就是 PTY 区 | `terminal.Props.Inset` 不可达 | 新增显式 prop `chrome.inset`（`PropInset`，整数，`0`=无边框）；`InsetSet` 区分"未设置"与显式 0；host placement 解析下发；单测覆盖 0/1/非法值/小盒退化（`TestInsetPropBorderlessTerminal`） |
| B | **`ctrl-shift-*` / `ctrl-alt-数字` 不可表达**：`keys.Name` 把 ctrl+shift+字母折叠成 `ctrl-字母`，ctrl+alt 也被丢弃 | `⇧C/⇧H/⇧V`（copy/clipboard/paste）与 `ctrl-alt+1..5`（tab.jump） | 无对应分支 | `keys.Name` 增加：ctrl+shift+ASCII 字母 → `ctrl-shift-<letter>`；ctrl+alt+数字 → `ctrl-alt-<digit>`；其余组合不变；单测覆盖 CSI-u 解析（`\x1b[118;6u` 等）与旧组合回归 |
| C | **`pos` 子树会擦掉内部组件**：合成器先画全局组件层再按包围盒**不透明**清 overlay，任何 `content.self` 盒子一旦 `pos` 化，终端就被自己的 overlay 清空（内容空白、只见框） | 程序用 pos 摆放组件及其 chrome | `blitOpaqueOverlay` 全局擦除 | `compositor.Compose` 改为：不属于 overlay 的 placement 照旧；属于某个 overlay 子树的 placement 在该子树内绘制（父节点 lines → 本节点组件 → 嵌套 overlay）；单测 `TestCompositorComponentInsideOverlaySurvives` / `TestCompositorComponentInsideNestedOverlay` |
| D | **重叠 `pos` overlay 的 placement 归属错误**：同级 pos 盒子重叠时（card 大小的大终端 + 其上的浮窗终端），placement 被归给**先声明**的外层 overlay，后声明的小 overlay 会用自己的不透明 fill 把它擦掉（浮窗内容空白、只剩框） | 浮窗终端画在主 pane 之上 | 首匹配分配（greedy first match） | `assignOverlayPlacements`：placement 归给**最后声明**（最上/最深）的包含它的 overlay；`Compose` 顶层与 `blitOverlayFrame` 嵌套层统一走该分配；单测 `TestCompositorPlacementFollowsTopmostOverlappingOverlay` |

其余 v3 能力（pane 标题条、按钮命中 id/action、workspace 条、编号计数、
owner/attach 数据、拖拽）确认 v2 已具备（文本盒子 + input + sources 快照 + drag），
无需框架改动；复刻的 pane collapse 是程序侧状态。

## 5. 仍存差异（对照行）

1. **footer 宽度裁剪**：1.txt 是 181 列，十个 live 动作全在；120 列下按
   `shell_bar.go` 规则只放下 `P/R/O/T/W/F/⇧C` + tail `G 󰒓 SYSTEM`，右段被 trim 到
   **只剩 ` 󰹹 0`**（ws/terminals 段按优先级丢失）。181 列渲染时与 1.txt 的
   header/frame/footer 逐字符一致（`v3_parity_test.py` 的 `capture_1txt` 用 1.txt
   本身当 oracle）。
2. **card-per-leaf 与 1.txt 的单框**：1.txt 的视觉是“一个窗口框 + 左右两块内容
   区”，而老 v3 card presentation 的每个 pane 本应各画一张 card。本轮按用户
   要求把分屏树改成每 leaf 独立完整 card（§1.6）；因此 `capture_1txt` 用
   **单 leaf** 的 3-tab 状态做 oracle（header/footer/单张 card 行仍逐字符对齐
   1.txt），交互分屏的 golden 另由 `left1right2`/`split_col`/`split_row` 公式
   断言。demo 初始状态（左 1 右 1）现在是两张 card。
3. **footer 颜色的启发式怪癖**：老 `footerActionKeyStyle()` 用的是
   `ShortcutKeyDisplay()` 展示键（`^C`、`←`、`Esc`、`enter`）而非配置 token，
   所以颜色是“文本命中”的结果：`⇧C SELECT` 命中 label 的 T → tab 蓝、
   `CTRL+E HSPLIT` 因 label 自带 CTRL 命中 R → resize 黄、picker 的
   `ENTER ATTACH` 命中 R → resize 黄、`ESC BACK` 无命中 → accent。复刻逐字
   保留这些怪癖（V3_PARITY 表格即 golden），未做“更合理”的语义改色。
4. **1.txt 的内容区**是 OpenCode live 画面（`┃`、会话标题、Context/Todo 面板）；
   复刻内容区由 v2 terminal 组件渲染真实 PTY，selftest/golden 用确定性占位文本，
   `--demo` 只在 chrome 上逐字符对齐。
5. **collapse 语义**：老 v3 floating `collapse` 是整体隐藏；复刻按既有任务要求
   改为**只留标题行**（可点动作组 `▾` 恢复），pane 提示行折叠后从布局移除
   （题字条状态槽显示 `▾`）。

## 6. 运行与验证

```bash
# 宿主 + v3 复刻（真实终端；冷启动 picker，Ctrl-F 绑定，Ctrl-Q 退出）
go build -o /tmp/tui2 ./clients/tui/cmd/tui2
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py"
# 目标态 demo（120×32 chrome 与 1.txt 对齐，用占位内容）
/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py --demo"
# 离线像素验证（独立公式实现 + 1.txt 行 oracle + 分屏树/左1右2 + footer 颜色）
python3 clients/tui/examples/python-shell/v3_parity_test.py            # 12 checks
python3 clients/tui/examples/python-shell/v3_parity_test.py --only left1right2
python3 clients/tui/examples/python-shell/v3_parity_test.py --only footer_colors
# 抓屏黑盒（acceptance.sh 内）：chrome 逐行 diff golden + 真实 PTY 分屏树/浮窗/颜色
bash clients/tui/scripts/acceptance.sh
```

手动复测分屏树（真终端，120×40）：

```
Enter 建终端 → Ctrl-P Ctrl-D 左右分（左 term-1 / 右 term-2，各一张 card）
  → Esc → Ctrl-P Ctrl-E 只切右侧叶子（右变 term-2/term-3 上下两张 card）
  → 三个 leaf 各自 stty size：左下 36x58 / 右上 17x57 / 右下 16x57
  → 拖右侧中间分隔条到 1-based row 27 → 右上 23x57、右下 10x57，左侧仍 36x58
  → tmux resize-window 100x30 → 26x48 / 16x47 / 7x47（比例保持），复原后回 10x57
  → Ctrl-P x 关闭右下 → 右侧恢复为单 leaf（36x57）
Ctrl-O n 新建浮窗（自动建终端）→ Esc 后直接输入 → Ctrl-O z 折叠只剩标题行
  → 再 z 展开 → 鼠标拖标题移动 → Ctrl-O x 关闭（终端留池）
外部 tmux resize-window 后所有 leaf/浮窗 PTY 尺寸跟随。
```
