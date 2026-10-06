# herdr（anytty 版）产品与交互规格

> 对齐对象：herdr.dev（terminal workspace manager）。本文是 anytty 上重写版的
> **唯一契约**：界面、模式、键位、鼠标、状态、持久化都以本文为准；与上游的差异
> 见 §11。

## 1. 定位与非目标

- 定位：把 pool 里的真终端组织成 **Workspace → Tab → Pane**，sidebar 一眼看出
  哪个 pane 要处理；鼠标优先，键盘提供 prefix/navigate 两套习惯（tmux 风格）。
- 非目标（v1 明确不做）：agent 进程检测/屏幕启发（不假装 blocked/working）、
  session/CLI socket API、worktree/git、通知与声音、terminal-title token、
  复制模式的多行选择（只做滚动回看 + 复制可见区）。

## 2. 数据模型（只用宿主真实可得的数据）

```
pool（宿主，持久）          程序侧模型（本程序）
终端 source（terminal:x:y）  Pane  ── 绑定一个 source id
                             Tab   ── 一组 Pane + 布局（split 方向/权重/焦点）
                             Workspace ── 一组 Tab + 名字 + 激活 Tab
```

- Pane 状态（全部来自 `sources` 快照，不吃不存在的字段）：
  `running`（存在且未退出）/ `exited(N)` / `offline`（endpoint `health != "ok"`）；
  附加标记：`you`（`attached && resize_owner == hello.view_id`）、`other`（别的
  resize owner）、`attached`（running 且无 owner 时显示；有 owner 时用
  `you`/`other` 表达）。
- **rollup**：Workspace 状态 = 其 pane 的最严重状态：
  `offline > gone > exited > running`（越靠前越严重；gone 表示绑定还在但源已不存在）；
  sidebar 的状态点即该 rollup。
- 没有"未读/done"：拿不到输出事件；不显示假状态。
- 源消失（`d`/外部删除）：Pane 保留绑定但标记 `gone`；用户可在该 pane 上 `enter`
  重新选择/新建。

## 3. 界面（桌面布局）

```
┌───────────────┬──────────────────────────────────────────────────────┐
│ main          │  1:main │ 2:logs │ +                ● connected        │ ← tab 条
│ ┌ Agents ───┐ │ ┌─ w1:t1 main ────────────────────────────────────┐  │
│ │ prod      │ │ │ ┌─ pane 1 · tui2-a ───┬─ pane 2 · tui2-b ─────┐ │  │
│ │  ● main · │ │ │ │                     ║                      │ │  │
│ │    main   │ │ │ │                     ║ ← 分割线可拖拽        │ │  │
│ │    tui2-a │ │ │ └─────────────────────┴───────────────────────┘ │  │
│ │  ● main · │ │ └─────────────────────────────────────────────────┘  │
│ │    main   │ │                                                      │
│ │    tui2-b │ │   mode bar（prefix/navigate/resize/scroll 时替换 tab 行）
│ └───────────┘ │   toast：仅在有消息时右下角浮层（无消息时不占行）    │
│ ┌ Spaces ───┐ │                                                      │
│ │ ● main    │ │                                                      │
│ └───────────┘ │                                                      │
└───────────────┴──────────────────────────────────────────────────────┘
```

- **Sidebar**（左，默认宽 26；`prefix+b` 折叠/展开）：
  - `Agents`：**所有 workspace** 的 pane，按机器（绑定 source 的 endpoint）分组；
    多于一台机器时每组一行机器标题，只有一台时不画标题。每个 pane 两行：
    第 1 行 `状态点 · workspace 名 · tab 名`，第 2 行 pane 名（用户重命名 → source
    标题 → terminal id，附 `you/other/attached` 与 exit/health 标记）。空 pane 与
    源消失（gone）的 pane 归入 `(unbound)` 组；行 id `herdr.agent:<w>:<t>:<p>`。
  - `Spaces`：每个 workspace 一行（状态点 + 名字，tab 名以 muted 追加）；行 id
    `herdr.space:<w>`。
  - 当前 workspace 的 Agents 行与激活的 Spaces 行用 `selection` 样式；聚焦 pane 的
    Agents 行带 `▸` 标记；点击行 = 聚焦 pane / 激活 workspace；鼠标滚轮滚动各面板。
- **Tab 条**：`序号:名字`，未激活用 `tab_inactive`，激活 `tab_active`；`prefix+z`
  zoom 时显示 `ZOOM` 标记；右侧只保留紧凑连接指示（`● endpoint` / `● connected`，
  未连接为 `○ waiting`）
- **Pane 边框**：仅分割时画（`widgets.BorderBox`/`SplitLayout` + 顶部标题
  `pane N · 终端标题`，激活 pane 用 `StyleBorderFocus`）；单 pane 不画外框
- **Mode bar**：prefix/navigate/resize/scroll 激活时显示在 tab 行位置，内容形如
  `PREFIX  v split · - split · c tab · w navigate · z zoom · ? help · q detach`
- **无常驻 status / footer 行**：tab（或 mode）条之下就是 body；尺寸、模式等不再
  常驻显示
- **Toast**：仅在消息存在时作为右下角浮层出现（`Pos` overlay，宽度 = 文本宽 +
  padding，超出屏幕即裁剪）；没有消息时不保留任何行
- **Overlay**：右键菜单（ContextMenu）、帮助浮层（Modal）、重命名输入（Modal +
  TextInput；workspace/tab/pane 分别用 `W`/`T`/`P`）；新建 workspace/tab 直接使用
  默认名，改名走重命名浮层

## 4. 模式

| 模式 | 进入 | 行为 | 退出 |
|---|---|---|---|
| terminal（默认） | — | 除本程序 claim 的键外全部进 pane | — |
| prefix | `ctrl+b` | 等一个动作键；未知键忽略并退出 prefix | 动作后 / `esc` |
| navigate | `prefix+w` | ↑/↓ 选 workspace（sidebar 高亮跟随）；`h/j/k/l` 切 pane；`enter` 激活 | `enter`/`esc`/`prefix` |
| resize | `prefix+r` | `h/j/k/l` 调整当前分割权重（每次 5%，边界 5%–95%）；`enter`/`esc` 退出 | 同上 |
| scroll | `prefix+[` | `j/k`、`PageUp/PageDown` → `terminal.scroll`；`y` 复制可见区（`terminal.copy`）；回到 live 即退出 | `q`/`esc`（先 `scrollEnd`） |

模式指示：mode bar 顶部一行显示模式名；navigate 时 Spaces 的选中行用
`StyleSelection`。

## 5. 键位（prefix 模式，默认对齐 herdr）

| 动作 | 键 | 动作 | 键 |
|---|---|---|---|
| 分屏（左右） | `v` | 分屏（上下） | `-` |
| 切 pane 焦点 | `h/j/k/l` | 交换 pane | `H/J/K/L`（shift） |
| 循环 pane | `tab` / `shift+tab` | 关 pane | `x` |
| zoom pane | `z` | resize 模式 | `r` |
| 新建 tab | `c` | 关 tab | `X`（shift+x） |
| 上/下 tab | `p` / `n` | 切 tab 1..9 | `1`..`9` |
| 新建 workspace | `N`（shift+n） | 关 workspace | `D`（shift+d） |
| 重命名 workspace | `W` | 重命名 tab | `T` |  ← 文本输入浮层 |
| navigate | `w` | 帮助 | `?` |
| 折叠/展开 sidebar | `b` | **detach（退出）** | `q` |
| scroll 模式 | `[` | 重命名 pane | `P` |

- terminal 模式直接 chord（可选，默认绑定）：`ctrl+alt+h/j/k/l` 切 pane、
  `ctrl+alt+c` 新 tab、`ctrl+alt+d` 分屏、`ctrl+alt+z` zoom
- 在 pane 聚焦时，除 `ctrl+b`、`ctrl+alt+*` 外都进 PTY（tmux 习惯：普通字符不抢）
- `q` 发 `system.quit`（宿主会弹 "Quit tui2?" 确认）；终端里要退出用 `ctrl+b q`

## 6. 鼠标（v1）

- **左键点击**：sidebar 行 → 激活 workspace / 聚焦 pane；tab → 切换；pane 内容 →
  聚焦该 pane（点进终端后按键直接进 PTY）
- **拖拽分割线**：press 在分割线盒子（`Input("mouse")` + 隐式捕获）→ `drag` 按
  x 差调整权重 → `release` 结束；只改布局，不重排其他几何
- **右键**：pane/tab/sidebar 行上弹 ContextMenu：`split right`/`split down`/
  `new tab`/`rename`/`close pane`/`close tab`/`new workspace`
- **滚轮**：sidebar 滚动列表；pane 内容默认给程序（本程序对 pane 只做焦点，不拦
  滚轮；终端 mouse tracking 时宿主透传）
- 不做：拖选复制、双链、拖拽 tab 重排

## 7. 持久化与 detach/attach

- 布局（workspaces/tabs/分屏权重/焦点/名字 + sidebar 折叠）序列化成 JSON，写
  `access.call` storage：`AppId "herdr"`、scope PRIVATE、key `layout`；结构性变更
  后异步保存（失败只 toast，不阻塞）
- 启动：先读 `layout` 恢复结构，再按 `sources` 快照重新绑定 pane（按 source id）；
  绑定不上的 pane 显示 `gone` + 空态提示
- **首次运行 / `layout` 不存在 = 没有保存布局**：storage Get 返回
  `ApiError NOT_FOUND` envelope，或宿主 RESPONSE 以 `not found` 错误文本拒绝调用，
  都按"无保存布局"处理——静默保留默认 workspace，不弹错误 toast，直接打开保存闸门；
  只有其他读取失败才 toast
- **detach = 退出 TUI**（`prefix+q` → 宿主确认）：pool 里的终端继续跑；
  重新 `herdr` 即 attach 回来（布局恢复 + 重新绑定）
- 旧 key `pinned` 兼容读取：升级时把 `pinned` 当作初始 pane 的绑定提示

## 8. 降级

- 没有 pool/access：sidebar 空态（`no agents`）+ 空 pane 提示 `enter creates a terminal`；
  storage 读取失败（非 not found）只 toast，首次运行没有 `layout` 静默
- endpoint offline：对应 pane 显示 `offline`，pane 不可交互但布局仍在
- 终端创建失败/被拒：toast 显示宿主返回的原因（不吞错误）

## 9. 测试与验收

- 单测（`model_test.go`）：workspace/tab/pane 增删改、split 权重与 resize 边界、
  prefix/navigate/scroll 模式状态机、键位表全覆盖、布局序列化/反序列化、sources
  对账与 rollup、gone/offline 状态、Agents（机器分组/两行行/`(unbound)`）与 Spaces
  面板、toast 右下浮层与无常驻 status/footer 行、首次运行无 `layout` 静默
- e2e（`e2e_test.go`，内存宿主）：HELLO+sources → `prefix v` 分屏 → `prefix c`
  新 tab → `prefix q` detach 的 RESULT → 重命名输入流 → storage 保存/恢复往返
- PTY 冒烟（手动/脚本）：`bash clients/tui/scripts/quickstart.sh --run` +
  harness `sendhex` 发 SGR 鼠标序列验证点击/拖拽/右键
- 全量门禁：`go test ./clients/tui/...`、`gofmt`/`go vet`、conformance 14/14×3

## 10. 与上游 herdr 的差异（明确保留）

| 上游 | 本版 | 原因 |
|---|---|---|
| agent 状态 blocked/working/done + 检测 | 无（只有 running/exited/offline） | 宿主无前台进程/屏幕 manifest |
| server/client detach、session attach | detach = 退出，重启恢复布局 | 宿主无 client 概念；pool 保证终端不丢 |
| worktree/git、notifications、sounds、CLI/socket API | 无 | 非核心 |
| copy mode 多行选择/搜索 | 仅滚动回看 + 复制可见区 | 需要程序侧选择模型 |
| 拖选复制、拖拽 tab、zoom 内的 pane 布局 | 无 / zoom=全屏当前 pane | 鼠标拖选需宿主选择模型 |

## 11. 键位与上游对照（备忘）

上游默认 prefix `ctrl+b`、detach `prefix+q`、帮助 `prefix+?`、navigate `prefix+w`、
resize `prefix+r`、copy `prefix+[`、分屏 `prefix+v`/`prefix+minus`、tab
`prefix+c`/`p`/`n`/`1..9`/`shift+x`、workspace `prefix+shift+n`/`prefix+w`/
`shift+d`、sidebar `prefix+b`、zoom `prefix+z`。本版只保留上表；未列出的上游键位
（如 `shift+g` worktree、`prefix+e` 编辑滚动、`prefix+s` 设置、`prefix+o` 通知）
按 §1 非目标处理。
