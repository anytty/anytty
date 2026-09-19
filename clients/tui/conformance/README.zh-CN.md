# TUI v2 SDK 一致性套件（conformance）

目标：**任何语言（含 AI 生成的）SDK 都能自证合规**。套件把"宿主怎么发帧、
程序必须回什么"固定成语言中立的数据，官方 Go/Python/TS 与用户 SDK 用同一
标准跑：

```bash
go run ./clients/tui/cmd/tui2-sdk-verify                          # 官方 Go SDK（进程内）
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "python3 clients/tui/sdk/python/conformance.py"
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "node clients/tui/sdk/ts/conformance.js"
```

退出码 0 = 全部通过，1 = 有失败项；`--list` 列用例，`--only <substr>` 只跑
匹配用例，`--fixtures` 指定套件文件（默认 `clients/tui/conformance/fixtures.jsonl`）。

## 1. 套件资产

| 文件 | 作用 |
|---|---|
| `fixtures.jsonl` | 用例：每行一个 JSON 会话脚本 |
| `frames.jsonl` | 二进制帧样例（hex + 解码 JSON），可逐字节对照实现解码器 |
| `conformance.go` | 用例类型、runner、语义比对（宿主侧，权威实现） |
| `program.go` | **参考程序**：官方 Go SDK 实现的被测程序 |
| `command.go` | 把外部命令当被测程序的 Endpoint |
| `interop_test.go` | Go 测试：fixtures 自检 + 三语言官方 SDK 全过 |

## 2. 被测程序契约（candidate program）

`--cmd` 启动的进程就是**一致性程序**：stdin/stdout 双向跑 v2 帧，stderr
只放日志，stdin 干净 EOF 时以 0 退出。它必须只用被测 SDK 实现下面的行为
（官方三语言实现在 `program.go`、`clients/tui/sdk/python/tui2sdk/conformance.py`、
`clients/tui/sdk/ts/conformance.js`，可直接对照移植）：

1. **HELLO**：把 `view_id/epoch/cols/rows/schema` 记成一行状态；
   若有 `components`/`methods` 各加一行；然后：
   - `Emit("clipboard.read", {})`（无参数，回调记录响应）；
   - `Commit(view, claim=["ctrl-p","?"], all=false)`，`rev` 从 1 开始。
   - HELLO 到达即 **epoch 重置**：状态清空、`rev` 归 0、旧 `pending` 作废；
     `request_id` **单调递增、跨 epoch 不复用**。
2. **VIEW**：根盒子是一段文本，内容是上述状态行的 `\n` 拼接；runner 按
   树遍历拍平后逐行比对，不关心树形。
3. **EVENT**（按键名分派到类型化 handler），每类追加一行后 `Commit`
   （`rev` +1）：
   - key：`key key=<key> char=<char> pane=<0/1>`（`ctrl-p` 把 pane 置 1）；
     `ctrl-r` 再 `Emit("clipboard.read")` 并追加 `emit clipboard.read id=<n>`；
   - paste/mouse/wheel/resize/notice/component/view_rejected：一事件一行
     （字段见 fixtures 的期望文本）；
   - sources：`sources count=<n>` + 每项
     `source id=… kind=… endpoint=… terminal=… exited=<0/1>`；若含
     `kind="terminal"` 的项，对**第一项** `Emit("terminal.attach",
     {endpoint,id,fit:true})` 并追加 `emit terminal.attach id=<n>`。
4. **RESPONSE**：先触发该 `request_id` 的 per-request 回调（仅当
   `response.epoch == 当前 epoch`，否则视为过期、**不触发回调**），回调追加
   `cb id=<id> ok=<0/1> text=<data.text> endpoint=<data.endpoint>
   data_id=<data.id> rows=<len(data.rows)> err=<error>`；随后通用
   `on_response` 追加 `resp id=<id> epoch=<n> ok=<0/1> err=<error>`，
   `Commit` 一次（顺序固定：先 cb 后 resp）。
5. **退出**：stdin EOF -> exit 0。

## 3. fixtures 格式（JSON Lines）

一行一个会话；`steps` 里每步恰有 `send` / `expect` / `exit` 之一。

```json
{"name":"hello/parse-and-emit","desc":"...","steps":[
 {"send":{"hello":{"schema":1,"view_id":"view:t:1","epoch":1,"cols":80,"rows":24}}},
 {"expect":{"frames":[
   {"result":{"method":"clipboard.read","params":{},"request_id":"$r0","epoch":1}},
   {"view":{"epoch":1,"rev":1,"claim":["ctrl-p","?"],"all":false,
            "text":["hello view=view:t:1 epoch=1 cols=80 rows=24 schema=1","emit clipboard.read id=1"]}}]}},
 {"send":{"response":{"request_id":"$r0","epoch":1,"ok":true,"data":{"text":"CB"}}}},
 {"expect":{"frames":[{"view":{"rev":2,"text":["…","cb id=1 ok=1 text=CB endpoint= data_id= rows=0 err=","resp id=1 epoch=1 ok=1 err="]}}]}},
 {"exit":0}]}
```

- `send.hello/event/response` 是符号化的宿主帧，runner 用 protobuf 编码成
  真实二进制；`send.raw` 是完整帧 hex（连长度前缀），逐字节发送。
- `expect.frames` 按顺序语义比对：`view` 比 `epoch/rev/claim/all/text`
  （`text` 为拍平后的内容行），`result` 比 `method/params/epoch/request_id`；
  未写的字段不检查，`params:{}` 表示必须为空。
- `"$rN"` 绑定第 N 个被测程序发出的 RESULT 的 `request_id`，供后续
  `response.request_id` 或断言引用。
- 一个 fixture 一个全新进程；所有 fixture 共用同一份判据，官方 SDK 与
  用户 SDK 无差别。

用户 SDK 只需实现第 2 节的行为并让 `--cmd` 能启动它，即可用同一套
fixtures 拿到与官方 SDK 完全一致的结论。

## 4. 新增语言 SDK 的最小步骤

1. 读 `clients/tui/proto/tui2.proto`（唯一真相）+ `clients/tui/docs/PROTOCOL.zh-CN.md`；
2. 写编解码：先过 `frames.jsonl`（hex 解码成 JSON 完全一致），再过
   `fixtures.jsonl`；
3. 实现 core（事件循环 + `Emit`/`Commit`）与第 2 节的一致性程序；
4. `tui2-sdk-verify --cmd "<你的启动命令>"` 全绿即可发布；
5. 把启动命令写进你的 README，AI 生成的 SDK 同样适用。
