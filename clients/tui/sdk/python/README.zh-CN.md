# Python SDK（`tui2sdk`，纯标准库）

三层：`tui2sdk.wire`（protobuf/帧）、`tui2sdk.core`（Client/App：事件循环、
类型化事件、Emit/Commit）、`tui2sdk.builder`（盒子链式声明 + 文本度量）、
`tui2sdk.widgets`（程序侧 chrome：card/tab 条/footer/picker/浮窗/分屏树/toast）。

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

一致性（与官方 Go/TS 同一套 fixtures，10/10）：

```bash
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "python3 clients/tui/sdk/python/conformance.py"
```

完整 API 与"被测程序契约"见 `clients/tui/docs/SDK.zh-CN.md` 与
`clients/tui/conformance/README.zh-CN.md`。参考程序（v3 像素复刻）见
`clients/tui/examples/python-shell/v3ui.py`。
