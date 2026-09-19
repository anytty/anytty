# 盒子协议（宿主 ↔ 布局程序）

> **历史文档（已归档）**：本文描述旧 `tui/` 盒子模型实现；该目录已随 T4 删除，
> 现行 TUI 见 `clients/tui/docs/`。仅作设计背景保留。

> 布局程序是宿主拉起的子进程：stdin 收事件（NDJSON），stdout 写展示树（NDJSON）。
> 只有布局程序需要这个协议；普通终端程序零接入（宿主给 PTY，terminal 组件显示）。

## 消息总览

| 方向 | 消息 | 说明 |
|---|---|---|
| 宿主 → 程序 | `hello` | 能力握手（schema/组件/事件/结果/features） |
| 宿主 → 程序 | `sources` | 可引用的内容源元数据（kind/title/endpoint/terminal_id/attached/exited） |
| 宿主 → 程序 | `world` | 只读世界状态最小集（panes/endpoints/focused） |
| 宿主 → 程序 | `resize` | 视口尺寸变化 |
| 宿主 → 程序 | `key` / `paste` / `mouse` / `wheel` | 未被宿主/终端消费的输入 |
| 宿主 → 程序 | `notice` | 宿主侧通知（info/warning/error，只读） |
| 程序 → 宿主 | `view` | 整棵盒子树（每次提交一整棵） |
| 程序 → 宿主 | `result` | 类型化意图（宿主白名单执行/授权） |

## hello（能力握手）

程序启动（含重启）后宿主立即发送一次：

```json
{"type":"hello","schema":1,
 "components":["terminal"],
 "events":["hello","sources","world","notice","resize","key","paste","mouse","wheel"],
 "results":["terminal.attach","terminal.create","terminal.restart","terminal.remove","workbench.command","system.quit"],
 "features":["node-cursor","mouse-passthrough","terminal-lifecycle","terminal-restart"]}
```

程序据此做 feature 检测；不认识的 feature 应降级而不是崩溃。

## view（盒子树）

```json
{"version":1,"rev":7,"root":{"flow":"col","children":[
  {"id":"header","size":{"height":1},"content":{"text":" WS main ","style":"header"}},
  {"id":"body","size":{"flex":1},"flow":"row","children":[
    {"id":"sidebar","size":{"width":30},"children":[]},
    {"id":"term","size":{"flex":1},"content":{"self":"terminal:local:main"},
     "input":["key","paste","mouse"],"focused":true}
  ]}
]}}
```

Node 字段：

| 字段 | 取值 | 说明 |
|---|---|---|
| `id` | string | 命中/焦点/`content.self` 元数据用；`core.*` 前缀保留给宿主 |
| `size` | `{width,height,flex,min,max}` | 固定尺寸优先，flex 分摊剩余 |
| `pos` | `{x,y}` | 脱离 flow 的绝对摆放（相对父内容区） |
| `flow` | `col`(默认)/`row`/`stack` | 子盒子排布 |
| `visible` | bool | 默认 true |
| `content` | `{self,text,lines,style}` | 三选一：不透明引用 / 文本 / 多行 |
| `border` | `{title,style}` | 纯绘制边框 |
| `cursor` | `{row,col,visible,shape}` | **程序光标**：相对该盒子内容区；宿主优先展示它（block/bar） |
| `input` | `["key","mouse","wheel"]` | 该盒子接收的输入类型 |
| `focused` | bool | 是否持有键盘焦点（宿主据此把按键送给终端 PTY 或程序） |
| `children` | []Node | 子盒子 |

## 事件负载

| 事件 | 字段 |
|---|---|
| `resize` | `cols` `rows` |
| `key` | `key`（`char`/`enter`/`up`/`ctrl-x`/`digit`…）`char` |
| `paste` | `text` |
| `mouse` | `node`（命中 id）`button`（left/middle/right/none）`action`（press/drag/release）`x` `y` |
| `wheel` | `node` `delta`（上滚 +1）`x` `y` |
| `notice` | `level`（info/warning/error）`message` |

## result（类型化意图）

```json
{"type":"result","rev":8,"kind":"terminal.create","params":{"endpoint":"local","title":"logs"}}
```

| kind | params | 授权 | 动作 |
|---|---|---|---|
| `terminal.attach` | endpoint/terminal | 无需 | 数据面 attach 并显示 |
| `terminal.create` | endpoint/title/command/workdir | 无需 | daemon 创建并自动 attach |
| `terminal.restart` | endpoint/terminal | 无需 | 重启已退出终端并重新 attach |
| `terminal.remove` | endpoint/terminal | **core overlay** | 删除已退出终端记录 |
| `terminal.scroll` | endpoint/terminal/delta | 无需 | 回看历史（delta>0 往更旧） |
| `terminal.scrollEnd` | endpoint/terminal | 无需 | 回到 live 并释放 history window |
| `terminal.copy` | endpoint/terminal | 无需 | 把当前可见内容写进本机剪贴板（OSC52） |
| `workbench.command` | action/target_id | kill/delete 需要 | 宿主终止对应终端 |
| `system.quit` | — | **core overlay** | 退出 anytty |

## 内容源（sources）

```json
{"type":"sources","items":[{"id":"terminal:local:main","kind":"terminal","title":"main",
  "terminal":true,"endpoint":"local","terminal_id":"main","attached":true,"exited":false,"exit_code":0}]}
```

- `id` 即 `content.self` 的引用值；只有已 attach 的终端建议放进主布局，未 attach 的适合放 picker。
- `exited/exit_code` 来自 owning daemon 的 lifecycle，程序据此提供重启入口。

## builtin 组件模型（属性 / 方法 / 事件）

组件 = 一个 `content.self` 内容源 + 属性（盒子字段）+ 方法（result）+ 事件（宿主推送）。
布局程序像写 web 一样声明组件；**能力属于组件，策略属于布局程序，管道属于宿主**。

### terminal（builtin）

| 属性（盒子字段） | 含义 |
|---|---|
| `content.self` | 内容源 id：`terminal:<endpoint>:<terminal_id>` |
| `border.title` | 标题（组件自带退出/回看角标） |
| `focused` | 键盘焦点；宿主只在 `focused=true` 时把按键送 PTY |
| `input` | 该盒子接收的输入：`key` / `paste` / `mouse` / `wheel` |
| `size` / `pos` | 布局程序负责的几何（宿主按此换算 PTY 尺寸） |

| 方法（result） | 授权 |
|---|---|
| `terminal.attach` / `terminal.create` | 无需 |
| `terminal.restart` / `terminal.remove` | 重启无需；删除需宿主确认 |
| `terminal.scroll` / `terminal.scrollEnd` / `terminal.copy` | 无需（回看/复制是组件能力） |

事件：`hello`（能力）、`sources`（含 `attached/exited/exit_code`）、`key/paste/wheel/mouse`、`notice`。

### 宿主契约（不变量）

1. **焦点**：一帧内最多一个 `focused=true` 的内容源；程序给出的索引必须有效，否则宿主把按键交给程序（不会静默进 PTY）。
2. **键盘**：Ctrl 组合（保留键除外）与 `focused` 终端的按键都归程序路由；`mode/overlay` 打开时程序必须自己让出 `focused`。
3. **鼠标**：只有"命中的面板 = `focused=true` 且该终端 live modes 开启 mouse tracking"才透传 PTY；其余鼠标事件一律给程序做命中测试。
4. **尺寸**：把 PTY 对齐到 pane 的一方就是 resize owner（CAS 接管）；follow/split 不改变权属。
5. **z 序**：`pos != null` 的子树是 overlay，宿主必须在所有组件 surface 之后合成；`core.*` overlay 永远最后。

## 版本

- `view.version` 与 `hello.schema` 当前为 `1`；新增可选字段不升版本，删除/改语义必须升版本并写迁移说明。
- 保留键（Ctrl-Q/Ctrl-C）与 `core.*` id 永远由宿主所有，程序收不到/不能占用。
