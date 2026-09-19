# 编写 anytty Program（盒子模型）开发说明

> **历史文档（已归档）**：本文描述旧 `tui/` 盒子模型实现；该目录已随 T4 删除，
> 现行 TUI 见 `clients/tui/docs/`。仅作设计背景保留。

状态：原型阶段。宿主只认识**盒子（rect/div）**：布局、裁剪、z-order、命中、合成。
界面结构与交互逻辑由外部程序负责；list/tree/input/select/terminal 等一律是程序用盒子
搭出来的组件，宿主不认。

## 1. 模型

- **宿主**：解算盒子布局、命中测试、合成到终端；只认识「盒子 + 纯绘制 + 输入路由」。
- **程序**：读事件（NDJSON stdin）、写展示树（JSON stdout）；状态、快捷键、切页、弹窗全在程序里。
- **内容旁路**：盒子可用 `content.self` 绑定一个不透明内容源（终端/插件自绘），内容走渲染面，不进展示树。

## 2. 快速开始

```sh
bash scripts/anytty-dev.sh prog
bash scripts/anytty-dev.sh prog --program examples/program-hello.py
bash scripts/anytty-dev.sh prog --trace /tmp/ev.ndjson --dump-effective /tmp/eff.json
```

## 3. 协议

### 宿主 → 程序（stdin，每行一个 JSON 事件）

```json
{"type":"resize","cols":120,"rows":30}
{"type":"key","key":"ctrl-f","char":""}
{"type":"mouse","node":"agents","button":"left","action":"press","x":10,"y":5}
{"type":"wheel","node":"p1","delta":1,"x":40,"y":10}
```

- `key`：`ctrl-*`、`tab`、`enter`、`esc`、`up/down/left/right`、`backspace`、`digit`(char)、`char`(char)。
- `mouse`：宿主已命中到 `node`；`action` ∈ `press|release|drag`。
- 宿主保留键（退出等）由宿主自己处理，不下发。

### 程序 → 宿主（stdout，每行一个 JSON 视图）

```json
{"version":1,"rev":42,"root":{ "flow":"col", "children":[ ... ] }}
```

## 4. 盒子（唯一节点类型）

```json
{
  "id": "pane-1",
  "size": { "flex": 1, "min": 10 },
  "pos":  { "x": 10, "y": 4 },          // 可选：绝对坐标，脱离父 flow
  "flow": "row",                         // row | col(默认) | stack
  "visible": true,
  "content": { "self": "agents.navigator" },  // 或 { "text": "..." } / { "lines": ["a","b"] }
  "border":  { "title": "main", "style": "accent" },  // 纯绘制
  "input":   ["key", "mouse"],
  "focused": true,
  "children": [ /* 更多盒子 */ ]
}
```

- 尺寸：固定 `width/height` 优先；`flex` 分摊剩余；纯文本内容有固有尺寸；其余默认填满。
- `flow`：`row/col` 线性排布，`stack` 叠放（z-order 即顺序）；`pos` 的盒子不参与 flow。
- `content` 三选一：`self`（不透明源）/ `text` / `lines`；宿主只画，不解释。
- `border`：纯绘制边框 + 标题，不产生行为。边框上的按钮 = 子盒子声明 `input`，行为由程序处理。
- `style`：`header/footer/accent/muted/warning/success`；未知会被忽略。
- 完整 schema：`docs/design/schema/display-tree.schema.json`。

## 5. 命中与输入转发

- 鼠标/滚轮：宿主用上一帧 `node→rect` 命中到 `node` id，再发给程序。
- 若命中的盒子声明了 `input` 且其 `content.self` 绑定了内容源，宿主**直接把输入转发给该内容源**（相对坐标），不再发给程序。

```json
{"id":"agents","content":{"self":"agents.navigator"},"input":["mouse","wheel"]}
```

## 6. 内容源（content.self）

程序只决定盒子位置/大小与内容源 id；宿主负责：
- 拉起/绑定对应 Surface 进程（本机渲染面 + 背压），把它自绘的帧合成到该盒子；
- 不解释它是终端、插件还是内置物。

## 7. 调试

- `--trace <file>`：宿主→程序事件 NDJSON。
- `--dump-effective <file>`：退出时输出 `node→rect`。
- `--log <file>`：程序 stderr 落文件（默认 `$TMPDIR/anytty-program.log`）。
- 非法视图/版本不符：记日志并保留上一棵好树。

## 8. 示例

- Go：`cmd/anytty-shell-program`（tabs/panes/sidebar/footer、模式、picker、命令面板、mouse、分屏拖拽、content.self）。
- Python：`examples/program-hello.py`。

## 9. 当前限制与后续

- 没有共享内存（走 NDJSON）；world state / collections / 类型化结果未实现。
- 安全内核、desired/effective 等设计见 `docs/design/TUI_PROGRAM_MODEL.zh-CN.md`。
