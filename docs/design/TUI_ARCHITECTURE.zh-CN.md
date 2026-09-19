# AnyTTY TUI 架构（盒子模型）

> **历史文档（已归档）**：本文描述旧 `tui/` 盒子模型实现；该目录已随 T4 删除，
> 现行 TUI 见 `clients/tui/docs/`。仅作设计背景保留。

> 状态：现行实现（refactor/tui-surface 完成后）。本文描述分层、数据流、红线与缺口清单。
> 相关文档：`TUI_PROGRAM_MODEL.zh-CN.md`（盒子协议设计稿）、`PROGRAM_DEV.zh-CN.md`（开发指南）。

## 定位

宿主是一个**盒子模型 TUI 框架**：程序产出盒子树（rect/flow/content），宿主解算、合成、
路由输入，并把数据面动作交给 owning endpoint 的 daemon。三层职责：

| 层 | 归属 | 回答的问题 |
|---|---|---|
| 框架内核 | `tui/program` + `tui/program/host` 协议 + 安全内核 | 盒子怎么解算、帧怎么合成、哪些东西程序不能绕过 |
| 组件 | `tui/program/components` | 某个引用（如真实终端）怎么渲染、有什么能力 |
| 布局程序 | `shell/`（可替换的外部进程） | 谁摆在哪、多大、焦点给谁、键位怎么定 |

原则：**能力属于组件，策略属于布局程序，管道属于 runtime，框架内核保持无业务。**
普通终端程序（bash/vim/htop…）零接入：宿主给 PTY，terminal 组件显示 cell。

## 架构图

```
                        ┌──────────────────────────────┐
                        │  真实终端 / tmux              │
                        └──────────────┬───────────────┘
                     ANSI 帧 ↑          │          ↓ 键盘/鼠标/resize
┌───────────────────────────────────────▼──────────────────────────────────────┐
│ anytty 宿主进程                                                               │
│                                                                              │
│  ┌─────────────────┐        ┌──────────────────────┐       ┌───────────────┐ │
│  │ tui/terminalhost│        │ tui/program/runtime  │       │ tui/program/  │ │
│  │                 │        │                      │       │ host          │ │
│  │ · raw mode      │───────▶│ · 事件循环            │◀─────▶│               │ │
│  │ · 输入解析       │        │ · 终端状态/焦点       │       │ · NDJSON 协议 │ │
│  │ · FrameSink diff│◀───────│ · overlay / 意图执行  │       │ · 退避重启    │ │
│  │ · resize 信号    │        │ · 数据面调用          │       │ · 最后一棵树  │ │
│  └─────────────────┘        └───────┬──────────────┘       │ · 保留键/id   │ │
│                                     │                      └───────┬───────┘ │
│  ┌──────────────────────────────────▼───────────────────┐          │         │
│  │ tui/program（框架内核，业务无关）                       │          │ stdin/  │
│  │ · 盒子树解算 flow/flex/pos/裁剪/图层/合成              │          │ stdout  │
│  ├──────────────────────────────────────────────────────┤          │         │
│  │ tui/program/components（组件层）                      │          ▼         │
│  │ · 引用解析 registry · terminal 组件（边框/高亮/光标）  │   ┌──────────────┐ │
│  ├──────────────────────────────────────────────────────┤   │ shell/       │ │
│  │ tui/render（最小渲染集）                              │   │ 布局程序      │ │
│  │ · canvas/surface/line/style/ANSI · LiveSurfaceLines  │   │ 盒子树+结果   │ │
│  └──────────────────────────────────┬───────────────────┘   └──────────────┘ │
└─────────────────────────────────────┼────────────────────────────────────────┘
                                      │ tui/port + adapters（数据面契约）
                                      ▼
                      ┌────────────────────────────────┐
                      │ endpoint daemon (core)          │
                      │ PTY / history / lifecycle       │
                      └───────────────┬────────────────┘
                                      │ PTY
                      ┌───────────────▼────────────────┐
                      │ 普通终端程序（bash/vim/…）      │
                      │ 零接入，宿主给 PTY              │
                      └────────────────────────────────┘
```

## 数据流

```
① 输入   TTY → terminalhost 解析 → runtime
          ├ 保留键（Ctrl-Q/C）→ 宿主 overlay（程序收不到）
          ├ 聚焦 terminal 的 key/paste → EncodeTerminalInput → port.SendInput → PTY
          └ 其余 → ProgramEvent → host.Send → shell
② 帧     shell stdout 盒子树 → host（StripReservedIDs）→ runtime.render
          → 框架解算 + 组件渲染 + CoreOverlayFrames → render → FrameSink → TTY
③ live   本帧 LiveTargets(endpoint,terminal,revision) → port.LiveScreenNext → surface → 下一帧
④ 意图   shell typed result → runtime 白名单：直执行(attach/create) 或 overlay 授权(kill/quit)
⑤ 数据面 runtime → port → owning endpoint daemon（lease/epoch 语义）
```

## 红线

1. 宿主不认识业务语义：list/picker/分屏只能由布局程序用盒子搭。
2. 程序不可覆盖安全内核：overlay 后合成、保留 id/键、破坏性意图必须授权。
3. 数据和命令只经 owning endpoint daemon；像素/输入只在本机。
4. 普通终端程序零接入；只有布局程序说盒子协议。

## 缺口与路线

| 优先级 | 项 | 归属 | 说明 |
|---|---|---|---|
| P0 | 终端退出检测 | runtime | 接 `WatchTerminalEvents`；退出态进入终端状态与组件视图 |
| P0 | 退出后动作 restart/remove | program 入口 + runtime 执行 | typed result `terminal.restart` / `terminal.remove`（remove 走授权） |
| P0 | 鼠标透传 | component 上报 + runtime 路由 | 终端开了 mouse tracking 时把鼠标字节发 PTY |
| P0 | 点击聚焦 | program 策略 | shell 里点 terminal pane 切换焦点 |
| P0 | 输入编码补全 | 框架（`tui/program/host`） | PgUp/PgDn、Home/End、Delete/Insert、F1-F12、Alt 组合 |
| P1 | program 光标协议 | 框架 + 组件 | 布局程序自己的 overlay（picker/prompt 输入框）声明光标 |
| P1 | 能力握手 + 协议文档 | 框架 | schema/feature 协商，第三方可照着写 |
| P1 | 第三方组件 provider | 组件 | 程序注册自定义引用渲染器 |
| P1 | TUIConfig 接线或删除 | cmd | 快捷键/主题配置目前加载但不生效 |
| P2 | 回看/复制 | 组件 + runtime | 需要恢复精简 history 端口与 clipboard 写端口 |
| P2 | deadcode 清理、E2E 自动化、性能基线 | 工程 | 旧快捷键系统等 unreachable 代码 |

## 非目标

- 不在宿主内置 list/tree/select 等业务 widget（组件只做通用渲染原语）。
- 不恢复声明式插件 UI；第三方 UI = 写布局程序或注册引用。
- workbench/工作区持久化不再是宿主概念，由布局程序自行决定（可经 typed result 扩展）。