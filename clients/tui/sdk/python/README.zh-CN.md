# Python SDK（`tui2sdk`，纯标准库）

三层：`tui2sdk.wire`（protobuf/帧）、`tui2sdk.core`（Client/App：事件循环、
类型化事件、Emit/Commit）、`tui2sdk.builder`（盒子链式声明 + 文本度量）、
`tui2sdk.widgets`（与 Go `sdk/widgets` 对齐的 21 个模块：布局/chrome
（layout/basics/chrome）、内容（list/table/input/modal/richtext/scrollbar）、
交互（mouse/hover/contextmenu）、表单（form/validate/select/datepicker）、
图表（chart/progress）、format、tokens/theme/border；
按模块命名空间使用，如 `tui2sdk.widgets.list.List`。旧 `ChromeApp` 一族在
`tui2sdk.widgets.chrome`，顶层扁平名保持兼容）。

```python
import sys
sys.path.insert(0, "clients/tui/sdk/python")
from tui2sdk import App, Client, builder

class Demo(App):
    def on_hello(self, hello):
        self.client.commit(builder.text("hello %s" % hello["view_id"]).build(),
                           ["ctrl-p", "?"])

Client(sys.stdin.buffer, sys.stdout.buffer).run(Demo())
```

一致性（与官方 Go/TS 同一套 fixtures，14/14）：

```bash
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "python3 clients/tui/sdk/python/conformance.py"
```

完整 API 与"被测程序契约"见 `clients/tui/docs/SDK.zh-CN.md` 与
`clients/tui/conformance/README.zh-CN.md`。参考程序（v3 像素复刻）见
`clients/tui/examples/python-shell/v3ui.py`。
