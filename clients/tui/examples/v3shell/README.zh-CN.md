# tui2-v3shell：老 v3 TUI 的 Go 复刻程序

`clients/tui/examples/v3shell` 是 **master（`main`）分支老 v3 TUI（surface framework）**
的独立 Go 复刻程序：用当前分支的 tui2 架构（`clients/tui/sdk` + `sdk/app`）实现，
样式取老版默认生效配置 `tui/docs/tui-v3.recommended.yaml`（profile
`coralline-candy`）——powerline 顶条、card 窗口题字与 `󰁌  󰖖  󰖗  󰅖` 动作组、
`● x1 owner` 状态槽、footer 场景键组与每键颜色、floating、递归切分树、overlay。
分屏几何与原版一致：**相邻 card 用自己的边框相接，没有独立分隔条**；terminal
picker 与原版一样**按机器（endpoint）分区**（endpoint tabs + 每个 endpoint 的
终端列表）。

像素规格与缺口见 `clients/tui/docs/V3_PARITY.zh-CN.md`。行为以 `main` 的原版为准；
Go 布局使用自己的 screen golden，footer 与 `1.txt` 行 oracle 继续共享参考校验。

## 1. 构建与运行

一键脚本（推荐先跑这个；`run`/`demo` 会先确保本地 pool+access 已启动，
否则 registry 里的 `local` 端点要走拨号预算（默认 3s+5s）才能创建终端）：

```sh
# 直接启动复刻程序（隔离 XDG，不动你的真实环境；--isolated 可省）
bash clients/tui/scripts/v3shell.sh run

# 直接看 1.txt 目标态（demo 状态，不需要任何终端）
bash clients/tui/scripts/v3shell.sh demo

# 离线自检：构建 + go test（9 项 golden）+ selftest 与 golden 逐行 diff + footer 8 场景
bash clients/tui/scripts/v3shell.sh check

# tmux 黑盒：chrome 逐行 golden、PANE footer、折叠提示、zoom、CSI-u、
# tab 新建、真实 PTY 建终端/分屏/关闭/退出（无 tmux 自动 SKIP）
bash clients/tui/scripts/v3shell.sh test          # --keep 保留会话供手动检查
```

手动构建与运行：

```sh
# 构建
go build -o /tmp/tui2-v3shell ./clients/tui/examples/v3shell
go build -o /tmp/tui2 ./clients/tui/cmd/tui2

# 直接用宿主运行（-shell 接受路径或"命令 + 参数"）
/tmp/tui2 -shell /tmp/tui2-v3shell

# 作为 CLI 默认 TUI 的布局程序（anytty 会读取 TUI2_SHELL）
TUI2_SHELL=/tmp/tui2-v3shell anytty
```

离线模式（不需要宿主，供测试与像素对比）：

```sh
/tmp/tui2-v3shell -selftest              # 光栅化 1.txt demo 状态（120x32）
/tmp/tui2-v3shell -selftest -cols 181 -rows 56
/tmp/tui2-v3shell -footer-lines          # 打印每个场景的 footer 行
/tmp/tui2-v3shell -version
```

## 2. 操作（老 v3 recommended 场景表）

> 底部快捷键栏是**精选子集**（与 Python 参考 golden 逐字节一致）。RESIZE 底栏把 `H/L/K/J` 合并为一个 token，腾出空间显示内容布局键 `0/$/^/B ALIGN`、`m/|/_ CENTER`（`A/S/W/D PAN` 在 ≥132 列显示）。每个 scene 的**完整键位**在 `?` help overlay 里（老版 `buildHelpContent` 枚举全部 binding，含 footer 里 `show:false` 的 `resize.pan_*`/`align_*`/`center*` 与 `copy.*`）。`?` 在 PANE/SYSTEM/RESIZE 场景打开 help（LIVE 场景下 `?` 按老版行为进 PTY）。

全局（live，聚焦终端时只有这些键被程序接管，其余键进 PTY）：

| 键 | 动作 |
|---|---|
| `Ctrl-P` | PANE 场景（`x`/`w` 关闭、`X` kill、`R`/`t` 重启、`Ctrl-D` 左右分、`Ctrl-E` 上下分、`h/l` 焦点、`z` zoom、`k` kill、`q` kill+close、`s` 锁尺寸、`b` 平衡、`a` 取 owner） |
| `Ctrl-R` | RESIZE 场景（`h/l/k/j` ±2、`H/L/K/J` ±6（对齐老版 bias 步进）、`space` 切换**切分方向**（见 §4 偏差说明）、`M` 切换内容布局模式 auto→fit→center、`m` 内容居中（双轴）、`r` 重置内容布局+切分、`=`/`b` 平衡、`s` 锁尺寸、`a` 取 owner；`0`/`$` 左/右对齐、`^`/`B` 上/下对齐、`\|`/`x` 水平居中、`_`/`y` 垂直居中、`A`/`S`/`W`/`D` 与 `shift+方向`（以及 `ctrl+方向`/`alt-HJKL` 别名）平移（X ∓2、Y ∓1）。align/center/pan 是 per-view 内容布局：把终端的权威 extent 按老版 `applyContentLayoutToExtent`/`alignedContentOrigin` 公式**在 pane 内容区内平移**（内容区始终全幅、PTY 尺寸不变），extent 外由 terminal 组件用 `·` 占位、四边越界由边框画 `◂ ▸ ▴ ▾`，并弹出 `terminal.layout` 状态 toast；布局随 pane 持久化（见 §4）。**owner 的 pan 同样生效**（只有 align/center 在 extent 等于 pane 时才是 no-op）。 |
| `Ctrl-O` | FLOAT 场景（`n` 新建空 panel，panel 内 `↑/↓` 选择 CTA、`enter` 执行；`z`/`m` 折叠只留标题行、`c` 居中、`x` 关闭、`1-9` 召唤、`h/j/k/l` 移动、`,`/`.`/`;`/`/` 与 `H/L/K/J` 缩放、`v` 全部折叠、`=` 最大化、`f` picker、`a` 取 owner） |
| `Ctrl-T` | TAB 场景（`c` 新建、`n/l/]` 与 `p/h/[` 前后、`1-9` 跳转、`x` 关闭、`X`/`k` kill+关闭、`r` 重命名） |
| `Ctrl-W` | WORKSPACE 场景（`c` 新建、`n/p` 前后、`x` 删除、`r` 重命名） |
| `Ctrl-F` | Terminal Picker（`←/→` 切换机器/endpoint 分区、`Shift+←/→` 循环 Running→Exited→All、直接输入搜索（大小写不敏感**子序列**匹配，命中标题/ID/状态/tag/`xN`/尺寸；中文名还支持拼音全拼与首字母，如 `suoping`/`sp` 命中「锁屏」，命中处高亮）、`Ctrl-T` 打开标签复选列表（`↑/↓` 选择、`space` 勾选、`Ctrl-T`/`esc` 返回）、`↑/↓` 选择、`enter` 绑定、`tab` 分屏绑定、`ctrl-k` kill、`ctrl-x` remove、`esc` 返回；首行是 `+ New terminal`；选中后弹出 Create Terminal 表单（name/command/server/workdir/tags），`Tab` 切换字段、`Enter` 提交、`Esc` 取消；overlay 高度上限 24） |
| `Ctrl-G` | SYSTEM 场景（`h` 顶条开关、`f` footer 开关、`p`/`m`/`t` picker、`o`/`:` 命令行、`e` connections overlay（`↑/↓` 选择、`t`/`enter` test、`r` reconnect）、`l` shortcut lock、`T`/`c`/`x` 关闭 toast、`?` help、`q` 退出） |
| `Ctrl-Shift-C` | COPY 场景（选区与搜索，见下） |
| `Ctrl-Shift-H` | Clipboard overlay |
| COPY 场景 | `h/l`/`←/→` 移动列，`j/k`/滚轮移动光标（到边缘才滚视图），`PgUp/PgDn` 步长为视口行数-2，`u/d` 半页，`g` 最老，`G` 回 live（再按入口键 `Ctrl-Shift-C` 也可退出；**老版 copy 场景没有 `esc` 绑定**，`esc` 不退出）。`space`/鼠标左键标记，`y` 复制并保留 copy，`enter` 复制并退出；无标记时滚回底部自动退出。`/` 编辑查询（带查询时打开会把光标放到末尾并保留原查询），**搜索栏替换底栏 footer 行**（不是 panel 最下方）：左侧 `⌕ [MODE] query`，右侧状态徽标（`N/M`、`no match`、错误）+ 窄屏隐藏的按键提示，编辑时在查询处显示反显光标；查询过长时围绕光标开窗滚动（同老版 `searchBarPresentation`）。`tab` 仅在搜索栏可见时循环 text→glob→regex，输入时高亮已加载窗口中的匹配，`Enter`/`n`/`N` 调用 `terminal.search` 导航并环绕（`n`/`N`/回车从**当前匹配之后**继续，与老版 `beginCopyModeSearch` 一致）。选区使用 ansi:8/ansi:3，复制经 `terminal.copy{sel}` 写 OSC52。`Ctrl-Shift-C` 重进时先释放快照，再读取最新窗口。历史来源与边界见下文。 |
| `Ctrl-Shift-V` | 粘贴（见 §4 差异） |
| `Ctrl-Alt-1..5` | 直跳 tab |
| `Ctrl-Q` | 退出（宿主确认） |

鼠标：点击 tab/新建、点击 card 按钮（zoom/split/close/lock/take-owner）、
拖拽两张 card 相接的边框调整比例（命中区就是相邻边框的 1 格，只写回该 Split）、
浮窗标题拖移、点击浮层行（含 picker 的 endpoint tab）、滚轮回看
（终端走 `terminal.scroll`，本地占位 pane 走行窗口）。**回看（COPY）会话按 pane 保存**（对照老版 `CopyModeByView`）：只有当前
聚焦 pane 的会话拥有输入，点击其它 pane 只是把输入交给它（滚轮/键盘不再被
回看场景吞掉），原 pane 的回看位置保留，切回时恢复 COPY 场景；滚回底部
（无选区）自动退出回看。空 pane 与原版一样**不画任何内部提示**。**panel 边框在拥有 copy 会话时整体变黄**（老版 `history-border`：warning+bold），标题与动作字形仍保持 accent 色；冻结的回看窗口被裁切时，边框上画 legacy `overflow_*` 提示字形（`◂ ▸ ▴ ▾`，`overflow_style #9ca3c9`：顶/底边框用 `▴/▾`，行宽超出行内容区时右边框用 `▸`），窗口比内容区短的行用 `extent_placeholder` 暗点 `·`（`#3b2f63`）补满，避免露出下层 pane。
owner 槽颜色对齐 legacy `terminalChromeVMFromBinding`：本视图拥有 resize 时 `owner` 用 success 绿色，`owner?` 等待态用 warning，`follow` 跟随态用 muted。

## 3. 实现结构

| 文件 | 内容 |
|---|---|
| `main.go` | 入口、`-demo`/`-selftest`/`-footer-lines` 离线模式 |
| `theme.go` | coralline-candy 色板/字形/场景表/footer 颜色解析链（yaml → 启发式） |
| `model.go` | 状态机：split 树、tab/workspace、floating、overlay、键鼠路由、`terminal.*`/`system.quit` 调用 |
| `view.go` | 视图树（header/card/floating/overlay/toast/footer），全部显式 `pos` + 显式样式 |
| `raster.go` | 离线光栅化器（golden 逐字符对比用） |
| `*_test.go` | 与 Python golden 同源的 9 项像素校验 + 交互行为测试 |

输入模型与老 UI 一致：**modal 场景接管全部按键**（`keys.all`），live 且聚焦
终端时只接管全局 chord，普通输入进 PTY。

## 4. 与老 v3 的差异与待完成项

| 项 | 老 v3 | 本复刻 |
|---|---|---|
| overlay 细节 | picker 有 endpoint tabs + toolbar（搜索/状态筛选）+ tags；clipboard 有持久历史 | picker 已按 endpoint 分区并用 endpoint label 作 tab 名，默认 Running，`Shift+←/→` 循环状态，搜索为子序列 + 拼音（全拼/首字母），`Ctrl-T` 打开标签复选列表；工具栏为「搜索左 / 状态+Tags 右」；尺寸与活跃度来自 `sources.cols/rows/last_output_ms`；tag 数据来自 `sources.tags`；clipboard history 通过 host 的持久 store 提供 list/delete/paste，overlay 仍可继续补齐完整老版视觉细节 |
| 空 panel 生命周期 | 新 panel 先显示未连接状态与 Attach/Create/Manager/Close 动作，`↑/↓` 选择、`enter` 执行（点击同样可用），选择后才绑定 terminal | 已实现：分屏、tab、浮窗创建空 panel；CTA 高亮只在聚焦 panel 上；Close 只关闭 panel，不隐式创建或 kill terminal |
| workspace/tab 持久化 | host storage（workbench store） | 通过 `access.call` 的 storage API 持久化到 `AppId=v3shell`/PRIVATE/`workbench` 键（版本化 JSON：workspaces/tabs/panes/split 树/focus/header/footer/per-pane 内容布局）。HELLO 时读取并恢复，结构变更后合并写回（`workbenchReady`/`savePending` 去抖）；demo/离线运行不落盘。老的 workbench store 无 typed 方法，storage 分区是等价的可移植实现 |
| terminal rename / detach / reconnect / shortcut lock / connections | daemon/宿主能力 | rename/detach/reconnect、shortcut lock 已接入 host 方法；connections 由 `Ctrl-G e` 的 overlay 呈现（`endpoint.list`/`endpoint.test`/`endpoint.reconnect`）。**插件概念已移除**：本复刻的 shell 本身即“插件”，不再有宿主 plugin runtime |
| copy 选择 | copy 会话按 pane/view 保存，支持持久历史查询 | shell 保存每 pane 的交互状态；内建 terminal 对象持有冻结 token、分页、搜索、选区复制和释放。不同 terminal 的请求独立排队；两个 pane 绑定同一 source 时仍共享该 terminal 的回看视口。宿主 API 支持 char/line/block，shell 使用标记流选区；跨出当前视口的完整选区仍需独立逻辑锚点支持 |
| paste（⇧V） | 系统剪贴板写入聚焦终端 | 通过 `clipboard.paste` 由 host 读取系统剪贴板并注入聚焦 PTY；历史条目通过 `clipboard_id` 选择 |
| resize align/center/pan | 完整几何操作 | 已按 `state.TerminalViewLayout` + `render.applyContentLayoutToExtent`/`alignedContentOrigin` 完整实现：per-pane 的 `mode`（auto/fit/center）、`alignX`/`alignY`（start/center/end/base）与 `panX`/`panY`，`0/$/^/B` 对齐、`m`/`\|`/`_`/`x`/`y` 居中、`A/S/W/D`+`shift+方向` 平移（X ∓2、Y ∓1）、`r` 重置，并弹 `terminal.layout` toast。布局随 pane 进入版本化 JSON（缺字段默认 auto/start/start/0/0）。**实现方式**：终端 box 始终声明为完整 pane 内容区（PTY 尺寸/owner resize 路径完全不动），布局只通过组件 prop `content.offset`(=`x,y` 平移量)/`content.size`(=`cols,rows` 权威 extent footprint) 传给 terminal 组件；组件把屏幕 1:1 画在偏移处、footprint 外用 `chrome.placeholder`（`·`）填充，四边越界仍由 pane 边框画 `◂ ▸ ▴ ▾`。因此**owner 的 pan 也可见**（align/center 在 extent==pane 时自然为 no-op），且大于 pane 的 extent 平移同样生效。**偏差**：老版 `space`=`resize.layout_toggle`（内容布局模式循环），但本复刻既有测试/几何把 `space` 钉在**切分方向**切换（`TestResizeModeChangesNearestSplit`、`TestLayoutToggleOnlyFocusedSplit`），故 `space` 保留切分方向，内容模式循环改绑到 `M`（auto→fit→center→auto） |
| 分屏分隔条 | 没有独立分隔条：相邻 card 边框相接，拖拽命中区是相接边框的 1 格 | 一致（Python 参考 `v3ui.py` 保留旧的 1 格分隔条近似，Go 复刻按原版） |
| 空 pane 提示 | 不画内部提示（1.txt 的 `┃ Click to collapse` 是终端内容） | 一致（不画提示；pane 折叠不存在） |
| zoom | `panel.toggle_zoom`（pane 占满 body，图标变 `↙`） | 已实现 |

footer 的 badge/键组/颜色/裁剪规则与 recommended yaml 逐字符对齐（见
`v3_footer_120x32.txt` / `v3_footer_colors_*.txt`）。

## 5. 验证

```sh
go test -count=1 ./clients/tui/examples/v3shell/     # 9 项 golden + 交互行为
python3 clients/tui/examples/python-shell/v3_parity_test.py   # Python 侧同一批 golden
```

golden 覆盖：
- Go 自有回归 golden（`testdata/golden/`，`go test -update` 可重生成）：demo
  （120x32/181x56）、demo_tab2、float 折叠；
- 公式断言钉几何：相邻 card 边框相接（无分隔条）、row/col 分屏 rect、
  left1right2 三 leaf rect、拖拽只写回本 Split、zoom 占满 body；
- 与 Python 参考共享的 golden：`1.txt` 行 oracle、footer 8 场景文本、footer
  每键颜色（NORMAL ≥7/9 色、PANE 5 色、PICKER 4 色）、card 动作按钮；
- 交互行为：claim/modal、分屏树、拖拽、zoom、浮窗、工作区、prompt、回看/复制
  （选区/搜索/OSC52 复制）、重启/kill、picker 分区/绑定、回看跨 pane 隔离与恢复。
- 宿主能力（配套改动）：`runtime.Terminal.CopyWindow`（char/line/block 选区提取）、
  `terminal.copy{sel}`、`history.window`/`terminal.scroll` 返回实际 offset
  （`MethodData.offset`）、terminal 组件的 copy 覆盖层 props
  （`copy.cursor`/`copy.sel`/`copy.match`/`copy.match_current` + 显式样式，
  显式样式串新增 `fg:ansi:N`/`bg:ansi:N`/`idx:N` 调色板写法）。

持久历史由 `runtime.Terminal.HistoryWindow/HistoryScroll/Search/HistoryCopy/HistoryRelease`
封装。RemotePTY 提供绑定当前连接的 backend；token 不会在重连后自动重放。
provider 返回逻辑行，terminal 按 pane 列宽展开 grapheme、保留宽字符与软折行坐标。
分页只保留尾页、当前页及视口坐标，深度由 provider 的保留策略决定，**不受本地
4096 行 ANSI 缓存限制**。本地 command PTY 没有 provider 时仍使用有限 parser 历史。

历史操作通过每个 terminal 的有界 FIFO 异步执行，网络等待不占用协议读循环或
渲染锁；关闭 terminal 会取消在途操作并释放 token。`CompleteForEpoch` 防止旧程序
应答命中新 epoch 复用的 request ID。create/restart 等其它操作移出会话锁并不等于
它们也已从协议读循环异步分离。

拖拽分屏边框时，宿主对每个 terminal 的 PTY resize 做 latest-only 合并：一次拖拽
每帧只产生一个请求，飞行中的请求不会被排成逐帧阻塞往返，返回后只应用最新尺寸
（对照 main 的 resize coalescing），因此拖拽保持跟手。

宿主帧循环是**事件驱动**的：程序提交 VIEW/VIEW_DELTA、PTY 产生输出、endpoint
健康或清单变化、notice 入队、quit 都会立即唤醒重绘，不再靠 16ms ticker 轮询。
分屏/移动等操作因此是帧级响应（实测分屏约一帧内可见）。lone `Esc`/`Alt` 仍用
一次性超时定时器处理，不引入轮询。

针对性验证：

```sh
TMPDIR=/tmp go test -race -count=1 -timeout 60s ./clients/tui/endpoint -run TestTerminalObjectReadsPersistentHistoryBeforeAttach
go test -race -timeout 60s ./clients/tui/runtime ./clients/tui/history ./clients/tui/cmd/tui2 ./clients/tui/examples/v3shell
```

真实 pool/access 测试先输出 6200 多行再 attach，验证最早历史、三种搜索、搜索后的
选区复制，以及宽字符跨行的坐标；宿主回归验证一个 terminal 等待历史时，另一 terminal
与 VIEW 提交继续完成。这些检查覆盖本轮能力，不代表原版所有交互均已对齐。
