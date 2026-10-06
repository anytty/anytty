# herdr — anytty 版终端工作区管理器（SDK 示例）

`herdr` 把 pool 里的真终端组织成 **Workspace → Tab → Pane**：左侧 sidebar 一眼看出
哪个 pane 要处理，右侧是终端面板，顶部是 tab 条（右端是紧凑连接指示），右下角是
仅在消息存在时出现的瞬时 toast 浮层。它只使用官方 Go SDK（`clients/tui/sdk`、
`sdk/app`、`sdk/widgets`）：宿主拥有终端，`herdr` 拥有全部盒子，不碰协议字节。

**产品与交互契约是 [`SPEC.zh-CN.md`](SPEC.zh-CN.md)**（本文件只做导读；两者冲突时以
SPEC 为准）。它对齐 herdr.dev，但明确不做 agent 状态猜测（blocked/working/done）、
session/CLI socket、worktree/git、通知声音、多行拖选。

## 数据模型与状态

```
workspaces[] → tabs[] → panes
  Workspace: 名字 + tabs + active tab
  Tab:       名字 + split 轴（row/col）+ 权重（百分比，和 100）+ focus + panes
  Pane:      id（w1:t1:p1）+ 名字 + 绑定的终端 source id
```

Pane 状态**全部来自 `sources` 快照**，不猜进程：

| 状态 | 条件 |
|---|---|
| `running` | source 存在、未退出、`health == "ok"` |
| `exited(N)` | source 存在且 `exited`，显示退出码 |
| `offline` | source 存在但 endpoint `health != "ok"`，pane 不可交互 |
| `gone` | 绑定还在，但快照里没有该 source；`enter` 可新建终端重新绑定 |
| `empty` | 未绑定；`enter` 通过 `terminal.create`（endpoint `local`）建一个并绑定 |

附加标记：`you`（`attached && resize_owner == view_id`）、`other`（别的 owner）、
`attached`（运行中已连接）。Workspace 的 sidebar 状态点 = 该 workspace 所有 pane 的
最严重状态：`offline > gone > exited > running`（SPEC 只定义了前三档，`gone` 与
`offline` 同属"不可用"层，见 SPEC §2）。

## Sidebar（Agents / Spaces）

- `Agents`：**所有 workspace** 的 pane，按机器（绑定 source 的 endpoint）分组；多于
  一台机器时每组显示一行机器标题，只有一台时省略。每个 pane 两行：第 1 行
  `状态点 · workspace 名 · tab 名`，第 2 行 pane 名（重命名 → source 标题 →
  terminal id，附 `you/other/attached` 等标记）。空 pane 与 gone 的 pane 归入
  `(unbound)` 组。行 id `herdr.agent:<w>:<t>:<p>`，点击聚焦。
- `Spaces`：每个 workspace 一行（状态点 + 名字，tab 名 muted 追加）；行 id
  `herdr.space:<w>`，点击激活。当前 workspace 的行用 `selection` 样式，聚焦 pane 的
  Agents 行带 `▸` 标记。
- 两个面板都可滚动（鼠标滚轮）；tab（或 mode）条之下没有常驻 status/footer 行，
  toast 只是右下角浮层。

## 模式

| 模式 | 进入 | 行为 | 退出 |
|---|---|---|---|
| terminal（默认） | — | 除 claim 的键外全部进 pane | — |
| prefix | `ctrl+b` | 等一个动作键；未知键忽略并退出 | 动作后 / `esc` |
| navigate | `prefix w` | `↑/↓` 选 workspace（sidebar 高亮跟随），`h/j/k/l` 切 pane，`enter` 激活 | `enter` / `esc` / `ctrl+b` |
| resize | `prefix r` | `h/j/k/l` 每次 5% 移动分割线（边界 5%–95%） | `enter` / `esc` / `ctrl+b` |
| scroll | `prefix [` | `j/k`、`PageUp/PageDown` → `terminal.scroll`；`y` 复制可见区（`terminal.copy`）；回到 live 自动退出 | `q` / `esc`（先 `scrollEnd`） |

mode bar 在 prefix/navigate/resize/scroll 时**替换 tab 行**；navigate 时 Spaces 的选中
行用 `selection` 样式。

## 键位

prefix（默认 `ctrl+b`）：

| 动作 | 键 | 动作 | 键 |
|---|---|---|---|
| 左右分屏 | `v` | 上下分屏 | `-` |
| 切 pane 焦点 | `h/j/k/l` | 交换 pane | `H/J/K/L` |
| 循环 pane | `tab` / `shift+tab` | 关 pane | `x` |
| zoom | `z` | resize 模式 | `r` |
| 新建 tab | `c` | 关 tab | `X` |
| 上/下 tab | `p` / `n` | 切 tab 1..9 | `1`..`9` |
| 新建 workspace | `N` | 关 workspace | `D` |
| 重命名 workspace/tab/pane | `W` / `T` / `P` | navigate | `w` |
| sidebar 折叠/展开 | `b` | 帮助 | `?` |
| scroll 模式 | `[` | **detach（退出）** | `q` |

terminal 模式直接 chord：`ctrl+alt+h/j/k/l` 切 pane、`ctrl+alt+c` 新 tab、
`ctrl+alt+d` 分屏（左右）、`ctrl+alt+z` zoom。

**按键归属（claim）**：焦点在**运行中的终端 pane** 时，`herdr` 只 claim `ctrl+b` 与
上述 `ctrl+alt+*`，其余按键（含普通字符）全部进 PTY；焦点在 sidebar/空 pane/退出或
离线 pane，或处于 prefix/navigate/resize/scroll/浮层时，claim 全部按键。这是
`keys.all=true` 时必须清掉 `focused` 的协议要求，因此程序模式下面板边框的焦点色会消失。

## 鼠标

- **左键**：sidebar 的 Spaces 行 → 激活 workspace；Agents 行 / tab → 切 tab 并聚焦
  pane；pane 内容 → 聚焦该 pane（点进终端后按键直接进 PTY）。
- **右键**：pane / tab / sidebar 行弹 `ContextMenu`（split right / split down /
  new tab / new workspace / rename / close pane / close tab / close workspace）。
- **拖拽分割线**：press 命中分割线盒子（`Input("mouse")` + 宿主隐式捕获，
  PROTOCOL §6.7）后按坐标差调权重，release 结束并保存布局。
- **滚轮**：滚动 sidebar 的 Agents / Spaces 面板；pane 内容不拦滚轮（终端 mouse
  tracking 时宿主透传 PTY）。

## 持久化 / detach / attach

布局（workspaces/tabs/panes/名字/轴/权重/focus + sidebar 折叠）序列化成 JSON，写
`access.call` storage：`AppId "herdr"`、scope PRIVATE、key `layout`；结构性变更后
异步保存（失败只 toast）。启动时先读 `layout` 恢复结构，再按 `sources` 快照按
source id 重新绑定 pane：绑定不上的显示 `gone`；快照里存在但还没本地 attach 的
（pool 终端）自动 `terminal.attach` 回来。兼容读取旧 key `pinned`，作为初始 pane 的
绑定提示。

**首次运行 / `layout` 不存在**（storage Get 返回 `NOT_FOUND` envelope，或宿主
RESPONSE 以 `not found` 文本拒绝调用）视为"没有保存布局"：静默保留默认 workspace，
不弹错误 toast，只有其他读取失败才 toast。

**detach = 退出 TUI**：`prefix q` 发 `system.quit`，宿主弹确认；pool 里的终端继续跑。
重新启动 `herdr` 会恢复布局并重新绑定。

## 降级

| 情况 | 表现 |
|---|---|
| 没有 pool/access | sidebar 空态（`no agents`）+ `layout save failed: …` toast；程序照常可用 |
| 首次运行没有 `layout` | 静默使用默认 workspace，不弹 toast |
| 没有任何 pane 绑定 | pane 空态 `empty pane · enter creates a terminal` |
| endpoint offline | 对应 pane 显示 `endpoint offline`，不可交互，布局仍在 |
| `terminal.create/attach/scroll/copy` 被拒 | RESPONSE 的 `error` 原文进 toast |
| attach 遇到别的 resize owner | 带 `expected_owner_epoch` CAS；冲突时降级 `fit:false` 跟随 |
| `view_rejected` / 传输错误 | 只 toast，不退出 |

## 运行

最省事（构建 + 全套测试 + 直接进 herdr；独立 state 目录，不动机器上的旧 daemon）：

```bash
bash clients/tui/scripts/quickstart.sh --run
# 退出后清理：bash clients/tui/scripts/quickstart.sh --stop
```

手动：

```bash
# 1) 编译到 /tmp（仓库内不留构建产物）
go build -o /tmp/herdr ./clients/tui/examples/herdr
go build -o /tmp/anytty ./cmd/anytty

# 2) 让 anytty 用 herdr 当布局程序（anytty 会先确保本地 pool+access 栈）
XDG_STATE_HOME=$HOME/.local/state/anytty-v3-herdr TUI2_SHELL=/tmp/herdr /tmp/anytty
```

开发循环（自动构建宿主、热重载、帧日志）：

```bash
bash clients/tui/scripts/dev.sh clients/tui/examples/herdr
```

测试：

```bash
TMPDIR=/tmp go test -count=1 ./clients/tui/examples/herdr/...
TMPDIR=/tmp go test -count=1 -race ./clients/tui/examples/herdr/...
```

## 与 SPEC 的已知差异（实现说明）

- 新建 workspace/tab 直接以默认名创建（`main`/`shell`），命名走 `W/T` 重命名浮层；
  SPEC §3 的"新建输入浮层"未做（SPEC §5 键位表本身也是直接创建）。
- 单 tab 的 split 轴是平的（一个 tab 一个轴），不做嵌套分屏；`v`/`-` 在单 pane
  时设定轴，多 pane 后保持现有轴。
- pane 边框/标题由程序自绘（`chrome.inset=0` 关掉终端组件边框），因此标题里的
  `pane N ·` 前缀由程序提供；终端组件自身的 `chrome.title` 颜色通道未使用。
- `zoom` 是视图状态（全屏当前 pane），不写入布局。
