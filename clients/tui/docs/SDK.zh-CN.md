# TUI v2 SDK（三层 API + 一致性验证）

> 协议唯一真相是 `clients/tui/proto/tui2.proto` + `docs/PROTOCOL.zh-CN.md`；SDK 只是
> **可选绑定**。任何语言按协议收发帧都能成为布局程序；`tui2-sdk-verify` +
> `clients/tui/conformance/fixtures.jsonl` 让每个 SDK（含 AI 生成的）自证合规。

## 0. 三层结构与目录

| 层 | 职责 | Go | Python | TS/JS |
|---|---|---|---|---|
| 1 core | 帧编解码、事件循环、类型化事件、`Emit`/`Commit`、hello/epoch、日志钩子 | `clients/tui/sdk/core`（`sdk` 包兼容再导出） | `tui2sdk.wire` + `tui2sdk.core` | `sdk/ts/src/wire.js` + `src/core.js` |
| 2 builder | box/row/col/stack/text/terminal/divider + 链式属性 | `clients/tui/sdk/builder`（`sdk` 兼容再导出） | `tui2sdk.builder` | `sdk/ts/src/builder.js` |
| 3 widgets（可选） | 程序侧 chrome：Card/Frame/TitleBar/TabBar/Footer/StatusBar/Picker/SplitLayout/FloatingLayer/Button/KeyHint/Toast/Toast | `clients/tui/sdk/widgets` | `tui2sdk.widgets` | 暂无（core+builder 已够一致性要求） |

目录：

```
clients/tui/sdk/            Go：facade（向后兼容）+ core/ + builder/ + widgets/
clients/tui/sdk/python/     Python：tui2sdk/{wire,core,builder,widgets,conformance}.py
clients/tui/sdk/ts/         TS/JS：index.js + index.d.ts + src/ + conformance.js + verify.js
clients/tui/conformance/    fixtures.jsonl + frames.jsonl + runner + 参考程序
clients/tui/cmd/tui2-sdk-verify/  一致性 CLI
clients/tui/templates/      三语言起点模板 go/ + python/ + ts/（§6）
clients/tui/docs/TUTORIAL.zh-CN.md  半小时上手教程
clients/tui/scripts/dev.sh  一键 dev 循环（构建宿主 + -dev + -watch）
```

## 1. core API

### Go `clients/tui/sdk/core`（`clients/tui/sdk` 同名再导出）

```go
c := core.New(os.Stdin, os.Stdout, core.Handlers{
    Hello: func(*pb.Hello), Sources: func([]*pb.Source), Key: func(*pb.KeyEvent),
    Paste: func(*pb.PasteEvent), Mouse: func(*pb.MouseEvent), Wheel: func(*pb.WheelEvent),
    Resize: func(cols, rows int), Notice: func(level, msg string),
    Component: func(*pb.ComponentEvent), ViewRejected: func(epoch, rev uint64, reason string),
    Response: func(*pb.Response),          // 每个 RESPONSE 都到；特定回调另算
})
c.Loop()                                    // 读到 EOF 返回 nil
c.Commit(root *pb.Box, core.Keys{Claim: []string{"ctrl-p"}})
c.Emit("terminal.attach", &pb.MethodParams{Endpoint: "local", Id: "t1"}, func(r *pb.Response) {})
c.Hello() / c.Epoch() / c.Rev() / c.Pending()
```

- 每个 `Emit` 分配新 `request_id`（连接内单调不复用）；RESPONSE 先按
  `request_id` 触发该请求回调、再进通用 `Response`；`response.epoch != 当前
  epoch` 时不触发回调（request_id 作用域 = (epoch, connection)，§0.5）。
- HELLO 原子重置：`rev=0`、pending 清空、epoch 更新；`Commit` 自动带 epoch+rev。
- 错误：`ErrNoHello` / `ErrNilRoot`。

### Python `tui2sdk`

```python
from tui2sdk import Client, App, builder

class My(App):
    def on_hello(self, hello):                       # 类型化事件；override 任意
        self.client.commit(builder.text("hi").build(), ["ctrl-p"])
    def on_key(self, event): ...                     # on_paste/mouse/wheel/resize/
    def on_response(self, response): ...             # sources/notice/component/view_rejected

Client(sys.stdin.buffer, sys.stdout.buffer, log=hook).run(My())   # EOF -> 0
self.emit("clipboard.read", None, on_response)       # App 上的便捷方法
```

`wire`：`decode_hello/decode_event/decode_response`、`encode_view/encode_result`、
`frame/read_frame`、`HELLO..RESPONSE`、`WireError`。

### TS/JS `clients/tui/sdk/ts`

```js
const { App, Client, builder } = require('./clients/tui/sdk/ts');
class My extends App {
  onHello(hello) { this.client.commit(builder.text('hi').build(), ['ctrl-p']); }
  onKey(event) { ... }
}
new Client(process.stdin, process.stdout).run(new My());  // Promise<exitCode>
```

`index.d.ts` 给出完整类型；无构建步骤（Node ≥18）。

## 2. builder API

| 语言 | 入口 | 链式属性 |
|---|---|---|
| Go | `sdk.Box()/Row()/Col()/Stack()/Text()/Terminal()/Divider()` | `ID/Size/Width/Height/Flex/Pos/Style/Focused/Input/Cursor/Content/Lines/Self/Props/Visible/Flow/Child/Build` |
| Python | `builder.box()/row()/col()/stack()/text()/terminal()/divider()` | `id/size/width/height/flex/pos/style/focused/input/cursor/content/lines/self_ref/props/visible/child/build` |
| TS | `builder.box()/row()/col()/stack()/text()/terminal()/divider()` | `id/size/width/height/flex/pos/style/focused/input/cursor/content/lines/selfRef/props/visible/child/build` |

文本度量三家同名：`RuneWidth/DisplayWidth/Truncate`（Go）、
`display_width/truncate/center_pad/clamp`（Python）、
`displayWidth/truncate/centerPad/clamp`（TS）。零值保持协议默认（§9.1）。

## 3. widgets API

Go `clients/tui/sdk/widgets`：`TabBar`、`StatusBar`、`Frame`(+`FrameRow`)、
`Divider`、`Card`、`Button`、`KeyHint`、`TitleBar`、`Footer`、`Picker`(+`PickerRow`)、
`SplitLayout`(+`Rect`/`Distribute`)、`FloatingLayer`、`Toast`；样式全部用
token 名（宿主主题解析），命中/按键由程序在 hit-test 后处理。

Python `tui2sdk.widgets`：`Theme` + `ChromeApp`（组合以上全部原语，含
`pane_runs`/`card_nodes`/`header_runs`/`footer_runs`/`overlay_rows`/
`floating_runs`/`toast_nodes`/`screen`）与模型 `Pane/Leaf/Split/Tab/Floating`。
参考实现见 `clients/tui/examples/python-shell/v3ui.py`（467 行，主题/场景/演示数据）。

## 4. 一致性验证（任何语言）

```bash
# 官方三语言
go run ./clients/tui/cmd/tui2-sdk-verify
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "python3 clients/tui/sdk/python/conformance.py"
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "node clients/tui/sdk/ts/conformance.js"

# 你的 SDK
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "<你的启动命令>"
```

- 用例格式与"被测程序契约"见 `clients/tui/conformance/README.zh-CN.md`；
- `frames.jsonl` 是二进制帧样例（hex + 解码 JSON），写解码器先过它；
- fixtures 由 runner 用 protobuf 编码宿主帧、语义比对程序帧，不绑定字节序
  或语言内部结构；官方 SDK 与用户 SDK 同一标准、逐项判定、非零退出码。

### 用 AI 生成新语言 SDK

1. 把 `clients/tui/proto/tui2.proto`、`PROTOCOL.zh-CN.md` 与
   `conformance/README.zh-CN.md` 第 2 节（被测程序契约）交给 AI；
2. 要求先实现 codec + core + 契约程序，命名对齐官方 SDK（便于对照移植）；
3. `tui2-sdk-verify --cmd "…"` 迭代到全绿；把命令写进该 SDK 的 README；
4. fixtures 只增不减：新行为先加用例，再让所有 SDK 跟上。

## 5. 内容源（组件）作者如何接

布局程序不直接画终端：它把一个盒子绑定到内容源（PROTOCOL §5）：

```json
{"id":"pane-1","size":[58,36,0],
 "content":{"self":"terminal:local:t1","props":{"chrome.border":"fg:#3b2f63"}},
 "input":["key","paste","wheel"],"focused":true}
```

- `content.self` = `sources` 事件里的 source id；`props` 是程序 → 组件的
  属性/样式通道（terminal 认 `chrome.border/title/border_focus/border_dead/
  badge`，未识别 key 忽略）；组件声明自己的 inset（terminal 当前 1），
  宿主按 `rect − inset` 算 PTY winsize/光标。
- 组件 → 程序的语义事件走 `component{source,name,value}`（宿主只中转）；
  能力调用（attach/create/restart/kill/scroll/copy…）走 RESULT 方法表。
- 生命周期/owner 以 `sources` 全量快照为准，程序设计成按 id 幂等对账
  （§9.2）；示例见 `SHELL/legacy/v3ui` 三个程序与 `ENDPOINTS.zh-CN.md`。

## 6. 起点模板与 dev 循环

三语言起点模板在 `clients/tui/templates/{go,python,ts}/`（100–200 行，行为
一致：tab 条、两个槽、焦点/点击、footer 键位提示、picker、`Esc`/`Ctrl-Q`
退出；源码内以"改哪里"注释标出扩展点）：

```bash
go build ./clients/tui/templates/go                       # Go 模板（SDK-linked）
python3 -m py_compile clients/tui/templates/python/program.py
node --check clients/tui/templates/ts/program.js
```

用 dev 循环跑模板（热重载 + 帧日志 + 报错可见）：

```bash
bash clients/tui/scripts/dev.sh clients/tui/templates/go       # 或 python / ts
bash clients/tui/scripts/dev.sh clients/tui/templates/python -- -routes local-unix
```

宿主 `-dev` 家族（非 dev 模式行为不变，验收有抓屏断言）：

| flag | 行为 |
|---|---|
| `-dev` | 默认帧日志 + stderr 捕获 + 崩溃/`view_rejected`/协议错误 notice |
| `-protocol-log <file>` | 双向帧逐条解码为 JSON Lines（`ts`/`dir`/`type`/`payload`） |
| `-watch <path>[,...]` | 轮询文件/目录，变化即重启布局程序并提示 `reloaded <path>` |
| `-reload-on-save` | 自动监视 `-shell` 的文件参数 |

帧日志默认 `$XDG_STATE_HOME/anytty/tui2-dev.log`；终端不打印日志。给模板加
按钮/面板、自测（`sdk-verify.sh`/`acceptance.sh`）与排错见
`clients/tui/docs/TUTORIAL.zh-CN.md`。

黑盒自测的驱动层在 `scripts/libdriver.sh`：有 tmux 用 tmux，没有则自动回退
内置 Go/PTY harness `clients/tui/cmd/tui2-harness`（`spawn`/`send_keys`/
`capture`/`capture_osc52`/`resize`/`kill` 原语，屏幕解析复用
`render/ansi`）。`TUI2_TEST_DRIVER=tmux|pty` 可强制；两种驱动跑同一批
smoke/acceptance 断言，无 tmux 时的覆盖范围/差异与对比表见
`TUTORIAL.zh-CN.md` §5.1 和 `scripts/driver-parity.sh`。

给模板加"可点击"三件套：稳定 `id` + `input:["mouse"]` + 程序在 mouse 事件里
处理该 `id`（缺一不可，SDK 没有隐式按钮语义）。
