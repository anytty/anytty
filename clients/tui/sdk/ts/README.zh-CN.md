# TS/JS SDK（`@anytty/tui2-sdk`，Node ≥18，无构建步骤）

core + builder（widgets 暂缺；一致性要求由 core 覆盖）。运行时是普通
JavaScript，`index.d.ts` 提供完整 TypeScript 类型。

```js
const { App, Client, builder } = require('./clients/tui/sdk/ts');

class Demo extends App {
  onHello(hello) {
    this.client.commit(builder.text(`hello ${hello.view_id}`).build(), ['ctrl-p', '?']);
  }
}

new Client(process.stdin, process.stdout).run(new Demo());
```

一致性：

```bash
node clients/tui/sdk/ts/verify.js        # 经 tui2-sdk-verify --cmd 跑全套 fixtures
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "node clients/tui/sdk/ts/conformance.js"
```

被测程序契约见 `clients/tui/conformance/README.zh-CN.md` §2；API 清单见
`clients/tui/docs/SDK.zh-CN.md`。
