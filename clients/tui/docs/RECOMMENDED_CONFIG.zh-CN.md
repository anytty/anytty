# 老版推荐 TUI 配置 → v2 映射规格（RECOMMENDED_CONFIG）

> 本文是 M21 的考古与映射记录：老版"推荐 TUI 配置"（coralline-candy）逐项拆解，
> 以及它在 tui2 的落地方式（默认主题/图标、键位、endpoint 模型）。
> 配置怎么写见 `CUSTOMIZE.zh-CN.md`；协议与场景以 `PROTOCOL.zh-CN.md`、
> `SCENARIOS.zh-CN.md` 为准。

## 0. 考古结论：老推荐配置在哪、模型是什么

| 来源 | 内容 |
|---|---|
| `tui/docs/tui-v3.recommended.yaml`（925 行） | **唯一权威的推荐配置**：`profile: coralline-candy`，含 theme/图标/模板/footer modes/shortcuts 全部场景 |
| `README*.md` §安装 | 安装脚本首次安装把它写到 `~/.config/anytty/tui-v3.yaml`（已有配置不覆盖） |
| 本机 `~/.config/anytty/tui-v3.yaml` | 与仓库文件逐行一致（仅注释行差异），确认它就是用户实际生效的推荐配置 |
| `tui/docs/tui-v3.example.yaml:268-271` | 默认配置里 Nerd Font 是 opt-in：注释写 `tab_create_icon: "󰐕"`，默认值 `"+"` |
| `tui/config/config.go` `Default()` | 非推荐默认：`profile default`、Palette `host`、`tab_create_icon "+"`、状态 glyph `○●◐○×!?`、`DimInactivePanels 0.5` |
| `tui/state/config.go` `DefaultTUIPickerEndpointStatusConfig()` | endpoint 状态 glyph：unknown/idle/disabled `○`、connected `●`、connecting `◐`、offline `×`、reconnect `!`、unregistered `?` |
| `tui/render/glyphs.go` `defaultPaneChromeGlyphs` | 非推荐 pane 字形：`×`/`↗`/`│`/`─`/`■`… |
| `~/.config/anytty/endpoints.yaml` + `scripts/anytty-dev.sh:write_endpoints` | 老 endpoint 模型 = **daemon 协议**：`version: 3`、`default`、`endpoints.<id>{label,label_source,enabled,connect_mode,routes}`，route kind 有 `local-unix`/`managed-webrtc`/`direct-webrtc-tcp`（字段 socket/credential_ref/target_device_id/signaling_addresses/ice_tcp_addresses/relay_mode…）；dev 脚本写的是 `local-unix` + socket 路径 |

**模型结论**：老推荐配置 = 一套**表现层 profile**（主题色 + Nerd Font 图标 + 模板 +
场景键位标签），挂在**daemon 协议 endpoint 模型**（routes/连接策略）之上。
tui2 的 v1 里：表现层全量可映射（见 §1/§2），endpoint 先用"命令式 endpoint"落地
（见 §4），daemon 协议差异显式记录。

## 1. 图标清单（codepoint + 用途 + v2 字段）

老推荐图标全部是单宽 PUA/Unicode；v2 的 `tui2.json` `icons` 预设
`recommended`（默认）逐名对应（实现 `clients/tui/config/icons.go`）：

| v2 图标名 | 码点 | 老推荐用途 | v2 出现位置 |
|---|---|---|---|
| `workspace` | U+F0645 `󰙅` | `workspace_template` 的 workspace 图标；footer `workspace_summary` | header workspace 槽、侧栏标题、footer/状态栏 |
| `tab_marker` | U+2387 `⎇` | `tab_template` 的活动/非活动 tab 前缀 | tab 条：`[⎇ 1:1]` / ` ⎇ 2:2 ` |
| `tab_new` | U+F0415 `󰐕` | `tab_create_icon` | tab 条 `+` 盒、picker `+ New terminal` |
| `tab_close` | U+F0156 `󰅖` | pane glyph `close`（老 tab 关闭同款） | 保留名（v2 tab 无关闭盒，slot 关闭用 `slot_close`） |
| `slot_marker` | U+258E `▎` | 老 pane 标题焦点标记 | 槽标题栏左端 |
| `slot_restart` | U+F0450 `󰑐` | 老 `panel.reconnect`/`RESET` 图标 | 槽标题栏 `restart` 按钮、footer restart 提示 |
| `slot_split_h` | U+EB56 `` | 老 `split_vertical`（左右分屏） | 槽标题栏 `split-h` 按钮 |
| `slot_split_v` | U+EB57 `` | 老 `split_horizontal`（上下分屏） | 槽标题栏 `split-v` 按钮 |
| `slot_close` | U+F0156 `󰅖` | 老 pane glyph `close` | 槽标题栏 `close` 按钮 |
| `mode_normal` | U+F030C `󰌌` | footer mode badge `live` | footer/状态栏 `󰌌 NORMAL` |
| `mode_pane` | U+EBEB `` | `menu.panel`/pane mode 图标 | footer/侧栏 `PANE` |
| `mode_scroll` | U+F018F `󰆏` | copy mode `copy` 图标 | `SCROLL` 提示、footer copy |
| `mode_picker` | U+F0C7C `󰱼` | picker 图标 | picker 浮层标题/提示 |
| `mode_prompt` | U+F4B5 `` | `menu.prompt`（command） | prompt 浮层标题 |
| `mode_help` | U+F02D6 `󰋖` | `menu.help` | help 浮层标题 |
| `summary_workspace` | U+F0645 `󰙅` | `workspace_summary` | 状态栏 workspace 段 |
| `summary_floating` | U+F0E59 `󰹙` | `floating_summary` | footer 右段 floating 计数（M22） |
| `summary_tab` | U+F04E9 `󰓩` | `tabs_summary`、tab 图标 | 状态栏 `󰓩 tab 1/1`、侧栏 tabs |
| `summary_slot` | U+EBEB `` | `panes_summary` | 状态栏 ` slot 1/1`、侧栏 slot |
| `summary_terminals` | U+F489 `` | `terminals_summary` | footer 右段 terminal 计数（M22） |
| `endpoint` | U+F0337 `󰌷` | `menu.connections`/endpoint 状态图标 | picker endpoint 行 |
| `focus` | U+F0734 `󰜴` | 老 panel `H/L 󰜴 FOCUS` label 图标 | footer PANE `TAB 󰜴 FOCUS`（M22） |
| `page_up` | U+F005D `󰁝` | 老 copy `PGUP 󰁝 OLDER` label 图标 | footer SCROLL `PGUP 󰁝 OLDER`（M22） |
| `page_down` | U+F0045 `󰁅` | 老 copy `PGDN 󰁅 NEWER` label 图标 | footer SCROLL `PGDN 󰁅 NEWER`（M22） |
| `attach` | U+F02FA `󰋺` | 老 terminal_picker `ENTER 󰋺 ATTACH` | footer picker（M22） |
| `run` | U+F0627 `󰘧` | 老 prompt `ENTER 󰘧 RUN` | footer prompt（M22） |

覆盖方式（preset 或 per-name map，见 CUSTOMIZE §3/§3.1）：

```json
{ "icons": "recommended" }
{ "icons": { "preset": "unicode", "map": { "tab_new": "N" } } }
```

预置：`recommended`（默认，上表）、`unicode`（旧 v2 字形 `+ / ⟳ ⇔ ⇕ ✕` 等，等于
M20 之前的观感）、`ascii`（无 Nerd Font 终端：`ws / + / R / | / - / x`）。

## 2. 颜色（主题）与模板

老推荐 `theme`（`palette: builtin` + 全量 token）：

| 老 token | 值 | v2 theme 槽（`recommended`） |
|---|---|---|
| `primary` | `#f0abfc` | `accent`、`border_focus`、`icon`、`tab_active`/`chrome_focus` 底色 |
| `secondary` | `#3b2f63` | `border`、`selection` 底、`button_pressed` 对比 |
| `foreground` | `#f8f4ff` | `fg`/`default`/`status`/`overlay` 文字 |
| `background` | `#070611` | `bg`、`chrome` 底 |
| `muted` | `#9ca3c9` | `muted`、`tab_inactive` 文字（另见 `accent_dim #c4b5fd`） |
| `success` | `#86efac` | `ok` |
| `warning` | `#fde68a` | `warning`、`button_pressed` 底 |
| `danger` | `#fb7185` | `danger`、`border_dead` |
| `info` | `#7dd3fc` | `info`、`endpoint` |
| `border.panel` / `inactive` | `#3b2f63` | `border` |
| `border.active` | `#f0abfc` | `border_focus` |
| `border.muted` | `#17132a` | 无独立槽（`status` 底色 `#17132a`） |
| `surface.chrome_bg` | `#070611` | `chrome` 底 |
| `surface.status_bg` | `#17132a` | `status` 底 |
| `surface.overlay_bg` | `#261b44` | `overlay` 底、`tab_inactive` 底 |
| `surface.toast_bg` | `#3b2f63` | `selection` 底 |
| tab 模板内联色 `#1b1230` | — | 活动 tab/按钮的深色文字 |
| `dim_inactive_panels` / `inactive_panel_dim_amount` | `true` / `0.5` | **无等价项**（v2 组件不调暗失焦内容） |

老模板 → v2 结构映射：

| 老模板 | v2 |
|---|---|
| `tab_create_template` + `{{create_icon}}` | TabBar `PlusText`（图标 + 1 格） |
| `workspace_template` `{{workspace}}` | header workspace 槽（图标 + 名称） |
| `tab_template` `⎇ {{title}} · T{{index}}` | tab 条 `[⎇ N:name]`（无 `· Tn` 后缀） |
| `pane_title_template` `{{terminal}}@{{endpoint}}` | 槽标题保留 terminal 标题（endpoint 在 picker 信息列） |
| footer `mode_badge` `{{mode_icon}} {{mode_label}}` | footer 左段 badge（M22 规格表 §3.1） |
| footer `action` `{{key}} {{icon}} {{label}}` | footer 左段 `" ·  " + label`（label 内嵌 key/icon，老 key 模板为空） |
| footer 右侧 summaries | footer 右段 `summary_workspace`/`summary_floating`/`summary_terminals`（仅 NORMAL/EXITED） |
| `picker.endpoint_status` glyphs `○●◐×!?` | 保留为老默认语义；v2 picker 目前以 `localhost · live/exited` 文案 + endpoint 名表达 |

## 3. 键位映射（推荐配置 ↔ v2 action）

老推荐全局：`Ctrl-P` panel、`Ctrl-R` resize、`Ctrl-O` float、`Ctrl-T` tab、
`Ctrl-W` workspace、`Ctrl-F` picker、`Ctrl-G` system、`Ctrl-Shift-C/H/V` copy/paste；
panel：`Ctrl-D` split-right、`Ctrl-E` split-down、`x` close、`q` kill+close、
`h/l` focus、`z` zoom；tab：`n/p/r/x`；system：`h/f/p/e/w/o/?/q`。

| 老推荐键 | v2 action | v2 默认键 | 差异 |
|---|---|---|---|
| `Ctrl-P` menu.panel | `pane.mode` | `ctrl-p` | ✅ 一致 |
| `Ctrl-F` picker | `picker.open` | `ctrl-f` | ✅ |
| `Ctrl-T` menu.tab | `tab.new` | `ctrl-t` | ✅（v2 新 tab 自动开 picker） |
| `Ctrl-W` menu.workspace | `sidebar.toggle` | `ctrl-w` | ≈ 语义最近（v2 无 workspace 菜单） |
| `x` panel.close | `slot.close` | `x` | ✅（解绑式关闭） |
| `y` copy_selection | `scroll.copy` | `y` | ✅ |
| `:` / `o` open_prompt | `prompt.open` | `:` | 老推荐是 `o`；v2 保留 `:` |
| `?` help | `help.open` | `?` | ✅ |
| `Ctrl-D` split_right | `pane.split_h` | `%` | **可改**；见下 |
| `Ctrl-E` split_down | `pane.split_v` | `"` | **可改**；默认 `Ctrl-E` 在 v2 是 restart |
| `r` panel.reconnect | `terminal.restart` | `ctrl-e` | 老 panel restart 是 `t`；`r` 是 reconnect |
| `Ctrl-R/O/G`、`Ctrl-Shift-C/H/V` | — | — | **v2 无对应能力**（resize 模式/floating/system 菜单/clipboard 历史） |
| `1...9` / `tab 场景 n/p` | 数字直切 tab | `1..9` | v2 无 next/prev action |
| pane `h/l` focus、`z` zoom | — | `Tab` 循环 | 无 zoom 概念 |

推荐风格键位覆盖（与老推荐 panel 场景对齐；`terminal.restart` 挪到 `r` 腾出
`Ctrl-E`，也是老 reconnect 键）：

```json
{
  "keybindings": {
    "pane.split_h": "ctrl-d",
    "pane.split_v": "ctrl-e",
    "terminal.restart": "ctrl-r"
  }
}
```

默认值保持 M20 老默认 UI 的键位（`%`/`"`/`Ctrl-E`）不变，避免破坏既有习惯与验收；
`pane.split_h`/`pane.split_v` 是新增的**可绑定 action**（白名单见 CUSTOMIZE §3），
不配置时与老默认完全一致。

### 3.1 v2 footer 规格表（M22：键位按场景裁剪，逐字符 golden）

老推荐配置的键位是**按 scene 分组**的，footer 只渲染当前场景的 key group。v2 的
scene 是 NORMAL / PANE / picker / prompt / help / scroll / exited；每个场景只取老
yaml 中**v2 已实现**的键位，未实现的场景动作不显示、不映射。

显示规则（与老渲染器 `tui/render/*` 的 recommended 行为一致）：

- **键名 ≠ 标签**：label 优先；无 label 时用规范化键名（`SPACE`、`PGUP/PGDN`、
  `CTRL-P`、`TAB`、`ESC`）。协议键名（`space`）与动作名（`split-h`、
  `resize.layout_toggle`）永远不出现在 footer。
- 左段：`" " + badge + " "`，然后每个键位 `" · " + " " + <label>`；label 按老
  yaml 内嵌 key/icon（`P  PANE`、`X 󰅖 CLOSE`），`Ctrl` 提升到 badge。
- 右段（仅 NORMAL/EXITED）：老 `*_summary` 模板，三段
  `summary_workspace`/`summary_floating`/`summary_terminals`（workspace/0/terminal 数）。
- 填充：`pad = cols - width(left) - width(right)`，最小 1（老 `footerText` 公式）。
- 瞬时提示（`status`/toast，如 `bound remote:term-1`）在 v2 里前置到右段
  （老版是 body 底部单独的 toast 行）；golden 场景不带提示。

| v2 场景 | badge（老 yaml `modes`） | 左段键位（老 label/icon/顺序，已实现子集） | 右段 |
|---|---|---|---|
| NORMAL | `󰌌 CTRL`（live） | `P  PANE` · `T 󰓩 TAB` · `W 󰙅 WORKSPACE` · `F 󰱼 PICK`（global 顺序 p/t/w/f） | `summary_workspace local` · `summary_floating 0` · `summary_terminals N` |
| PANE | ` PANE`（pane） | `X 󰅖 CLOSE` · `%  VSPLIT` · `"  HSPLIT` · `TAB 󰜴 FOCUS` · `ESC BACK` | — |
| picker | `󰱼 PICK`（terminal-picker） | `↑/↓ SELECT` · `ENTER 󰋺 ATTACH` · `ESC BACK` | — |
| prompt | `PROMPT`（老 fallback 大写） | `ENTER 󰘧 RUN`（prompt scene） | — |
| help | `HELP`（老 fallback 大写） | —（help scene 全 `show:false`） | — |
| scroll | `󰆏 COPY`（copy） | `PGUP 󰁝 OLDER` · `PGDN 󰁅 NEWER` · `Y 󰆏 COPY` · `ESC LIVE` | — |
| exited | `󰌌 CTRL`（live badge；老无独立 exited 模式） | `E 󰑐 RESTART` · `F 󰱼 PICK` | `summary_workspace local` · `summary_floating 0` · `summary_terminals N` |

- **不支持因此不显示**的老场景动作：`ctrl-r` SIZE、`ctrl-o` FLOAT、`ctrl-g` SYSTEM、
  `⇧C/H/V` copy/clipboard、panel 的 `d/r/z/h/l/q`、resize 整场景（`SPACE 󰘶 LAYOUT`、
  `= 󰕕 BALANCE`、`CTRL+←/→/↑/↓ PAN`）、floating/zoom、tab `n/p`、terminal_picker 的
  endpoint/status/tags/split/rename/kill/remove。
- **键位替换**：v2 用实际绑定 `%`/`"`/`TAB`/`ESC` 替换老推荐的 `CTRL+D`/`CTRL+E`/
  `H/L`；label 的 icon/文案取老推荐（默认键位仍是老默认 UI 的 `%`/`"`，推荐键位覆盖
  `pane.split_h: ctrl-d` 等仍可用）。重启同理：`E 󰑐 RESTART`（老 `CTRL+R 󰑐 RESTART`）。
- **颜色**：老 `footer-key-*` token→SGR 的实现随老程序移除；v2 按语义映射到推荐色板
  （`footer-key-pane/picker`→accent、`resize/global`→warning、`tab/workspace`→accent_dim、
  `float`→info、`copy`→ok）。逐字符 golden 只比对字符与空格。
- **golden**：`clients/tui/examples/python-shell/golden/recommended_footer_120x32.txt`
  （NORMAL / NORMAL_2PANE / PANE / PICKER / PROMPT / HELP / SCROLL / EXITED 共 8 行），
  Go shell 从**最终合成帧**取 footer 行（`footer_golden_test.go`），legacy.py
  `--footer-lines` 输出同一格式，两侧由同一规格表驱动逐字符比对（parity 30/30）。

## 4. Endpoint 模型（老 daemon 协议 → v2 连接策略）

老模型（`endpoints.yaml` v3，daemon 协议）：

```yaml
version: 3
default: local
endpoints:
  local:
    label: local-dev
    enabled: true
    connect_mode: auto            # auto / on_demand / manual
    routes:
      local:  { kind: local-unix, socket: /run/... }
      cloud:  { kind: managed-webrtc, credential_ref: ..., relay_mode: auto }
      direct: { kind: direct-webrtc-tcp, signaling_addresses: [...], ice_tcp_addresses: [...] }
```

v2（`tui2.json` `endpoints`，两种 kind）：

```json
{
  "endpoints": [
    { "name": "remote", "kind": "command", "label": "dev box",
      "argv": ["ssh", "dev-box", "anytty", "attach", "--socket", "/run/anytty.sock"],
      "cwd": "", "env": {} },
    { "name": "dev", "kind": "daemon", "label": "dev-daemon",
      "socket": "/run/user/1000/anytty-v2-wire7-dev.sock" }
  ]
}
```

| 老字段 | v2 字段 | 说明 |
|---|---|---|
| endpoint id（map key） | `name` | 成为协议 source id `terminal:<endpoint>:<id>`；禁 `:` |
| `label` | `label` | picker 展示名，缺省用 `name` |
| `routes.local.kind=local-unix` + `socket` | `kind: "daemon"` + `socket` | host 按 daemon 协议连接（list/attach/input/resize/kill/remove/重连），picker 分组显示 endpoint 与其终端 |
| `connect_mode`（auto/on_demand/manual） | `connect_mode` | `local-unix`/`tcp` 已实现（tcp 用 `address: HOST:PORT` 指向 `ssh -L`/桥）；`direct-webrtc-tcp`/`managed-webrtc` 不做（配置可写，连接报可读错误） |
| 命令式等价用法 | `kind: "command"` + `argv` | 起本地 PTY；远程 daemon 也可在 argv 里跑 `ssh host anytty attach ...` |
| `enabled` | 无 | 列入配置即用；daemon 断线自动退避重连并给 health/notice |
| `env`/cwd（daemon 侧） | `env`/`cwd` | daemon endpoint 下作为新建终端的 spec（协议字段 5/6/7） |

行为：picker 在已知终端之后、`+ New terminal` 之前列出配置 endpoint
（`󰌷 label` + `endpoint · command <argv[0]>` 或 `endpoint · daemon local-unix`）；
配置了 daemon endpoint 时按 endpoint 分组（组头 `  <endpoint>`，daemon 终端带
`<endpoint> · live/exited/offline`）。daemon endpoint 选中即
`terminal.create{endpoint,kind,socket,...}`；已有 daemon 终端单击即
`terminal.attach`，之后与本地终端同一条组件/输入/scrollback 路径。

完整调用序列、health 语义、支持矩阵与断线/错误路径见
`ENDPOINTS.zh-CN.md`。

## 5. 默认行为差异与无法等价项（≤4 条）

1. **失焦调暗**：老 `dim_inactive_panels/inactive_panel_dim_amount` 在 v2 没有
   等价（组件不调暗内容）；用 `border`/`muted` 颜色表达层级。
2. **图标宽度**：老推荐全是单宽 PUA；v2 按单宽布局（`sdk.DisplayWidth`），
   覆盖成宽字符时由 render 截断处理，按钮位置/点击坐标会随宽度变化。
3. **键位能力缺口**：resize 模式、floating、system 菜单、clipboard 历史、
   zoom、tab next/prev 在 v2 不存在对应 action；footer 按场景只显示已实现键位
   （§3.1），这些老场景动作不映射也不显示。老 `footer-key-*` token→SGR 的实现随
   老程序移除，v2 用推荐色板按语义近似。
4. **endpoint 是命令不是连接策略**：v1 无 routes/credentials/relay/自动重连；
   远程场景通过 argv（`ssh host anytty attach …`）达成，状态图标不反映真实连接态。

## 6. legacy.py 的可选 preset

老 UI 像素复刻程序 `clients/tui/examples/python-shell/legacy.py` 也支持同一套表现：

```bash
# 默认：逐像素老默认 UI（parity 28 项不变）
python3 clients/tui/examples/python-shell/legacy.py

# 推荐 preset（-config 或 ANYTTY_LEGACY_CONFIG）
python3 clients/tui/examples/python-shell/legacy.py --selftest --config rec.json
# rec.json：{"preset": "recommended"}（可附 {"icons": {...}, "colors": {...}} 单点覆盖）

/tmp/tui2 -shell "python3 clients/tui/examples/python-shell/legacy.py --config rec.json"
```

preset 换掉 workspace/新建/关闭 glyph 与六个 legacy 主题 token，并把 footer 切到
§3.1 的推荐场景规格（workspace 名 `local`）；`icons`/`colors` 键名见
`RECOMMENDED_ICONS`/`RECOMMENDED_STYLES`（legacy.py 顶部）。`--footer-lines`
打印 8 个场景的 footer 行（`<SCENE>|<line>|`），与 Go shell 共用同一 golden：

```bash
python3 clients/tui/examples/python-shell/legacy.py --footer-lines --config rec.json
python3 clients/tui/examples/python-shell/parity_test.py   # 30/30，含 8 行逐字符比对
```
