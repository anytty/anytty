# TUI v2 半小时教程：用 `dev` 循环写第一个布局程序

> 目标：30 分钟内跑起一个官方起点模板 → 改出一个自己的按钮/面板 →
> 用仓库自带的验证工具自测。
> 所有命令在**仓库根目录**执行。需要 Go（宿主与 Go 模板）、`python3`
> （Python 模板）、`node`（TS 模板，可选）。
>
> 布局程序的唯一契约是 `docs/PROTOCOL.zh-CN.md` 的二进制帧协议；SDK 是可选
> 绑定，目录与三层 API 见 `SDK.zh-CN.md`。本教程全程不要求理解协议细节。

## 0. 一分钟检查环境（1 分钟）

```sh
go version          # 期望 go1.26.x（go.work 已包含本仓库）
python3 --version   # Python 3.8+
node --version      # 可选；没有就跳过 TS 模板
```

找不到 `go`？用仓库自带工具链：

```sh
export PATH="$HOME/.local/share/go-toolchains/go1.26.7/bin:$PATH"
export GOTOOLCHAIN=local
```

## 1. `dev.sh` 一键开发循环（2 分钟）

`clients/tui/scripts/dev.sh` = 构建宿主 + 以 `-dev` 启动 + 文件保存即热重载。
脚本开头注释就是全部用法；直接跑 Go 模板：

```sh
bash clients/tui/scripts/dev.sh clients/tui/templates/go
```

**预期效果**（120×36 终端）：

```
 main [main] +
  [空槽]                          [空槽]
  Ctrl-F 选择终端                 Ctrl-F 选择终端
...
Ctrl-F picker · Ctrl-T tab · click focus · Esc quit        tab 1/1
```

- 冷启动没有终端 → 弹出 picker；按 `Enter` 选 `+ New terminal` → 槽里出现
  终端，footer 显示 `bound term-1`。
- `Ctrl-F` 再开 picker（列出已有 terminals）；`Ctrl-T` 加 tab；
  点击槽切换焦点；`Esc` 或 `Ctrl-Q` 退出（宿主确认框，`Enter` 允许）。
- **不用退出**：改 `clients/tui/templates/go/main.go` 里任一字符串并保存，
  宿主会重启程序（画面保留最后一棵树），footer 显示 `reloaded <file>`。

`dev.sh` 选模板的规则：

| 目标 | 命令 | 说明 |
|---|---|---|
| `shell`（默认） | `bash clients/tui/scripts/dev.sh` | 默认 Go 布局程序 `tui2-shell` |
| 目录/`.go` | `bash clients/tui/scripts/dev.sh clients/tui/templates/go` | `go build` 后运行 |
| `.py` | `bash clients/tui/scripts/dev.sh clients/tui/templates/python` | `python3` 运行 |
| `.js` | `bash clients/tui/scripts/dev.sh clients/tui/templates/ts` | `node` 运行 |
| 自己的文件 | `bash clients/tui/scripts/dev.sh /tmp/my.py` | 自动按扩展名选择运行时 |

宿主的额外 flag 放在 `--` 之后，例如：

```sh
bash clients/tui/scripts/dev.sh ts -- -routes local-unix
```

## 2. `dev` 模式到底给什么（3 分钟）

| flag | 行为 |
|---|---|
| `-dev` | 开发模式：默认帧日志、stderr 捕获、崩溃/`view_rejected`/协议错误 notice |
| `-protocol-log <file>` | 双向帧解码成 JSON Lines（时间戳 + 方向 + protobuf→JSON） |
| `-watch <path>[,...]` | 轮询文件/目录；变化 → 重启布局程序并提示 `reloaded <path>` |
| `-reload-on-save` | 等价于自动对 `-shell` 的文件参数做 `-watch` |

帧日志默认路径：`$XDG_STATE_HOME/anytty/tui2-dev.log`
（`$XDG_STATE_HOME` 缺省是 `~/.local/state`）。实时看协议：

```sh
tail -f "${XDG_STATE_HOME:-$HOME/.local/state}/anytty/tui2-dev.log"
# {"ts":"...","dir":"host->program","type":"HELLO","bytes":390,"payload":{...}}
# {"ts":"...","dir":"program->host","type":"VIEW","bytes":812,"payload":{...}}
```

日志只写文件，**终端永远不打印日志**（与 `-log-file`/`TUI2_LOG_FILE` 的宿主
诊断日志一致；后者默认 `.../anytty/tui2.log`）。

程序报错在 dev 模式可见：stderr 末尾几行 + 崩溃原因 + 重启倒计时会作为 notice
发给布局程序（模板把它显示在 footer）；同时写日志文件：

```
layout program exited immediately (21ms after start); restarting in 250ms; stderr: Traceback ...
```

## 3. 三语言模板对照（5 分钟）

三个模板行为一致（tab 条、两个槽、焦点/点击、footer 提示、picker、
`Esc`/`Ctrl-Q` 退出），都是 100–200 行，用各自 SDK：

```sh
bash clients/tui/scripts/dev.sh clients/tui/templates/go
bash clients/tui/scripts/dev.sh clients/tui/templates/python
bash clients/tui/scripts/dev.sh clients/tui/templates/ts
```

| 语言 | 入口 | SDK | 源码里的"改哪里" |
|---|---|---|---|
| Go | `templates/go/main.go` | `clients/tui/sdk`（core/builder/widgets） | `header/footer` · `key` · `pick` · `view` |
| Python | `templates/python/program.py` | `clients/tui/sdk/python`（`tui2sdk`） | `header/footer` · `key` · `pick` · `view` |
| TS/JS | `templates/ts/program.js` | `clients/tui/sdk/ts` | `header/footer` · `key` · `pick` · `view` |

把模板复制出仓库后，Python 需要告诉它 SDK 在哪：

```sh
cp clients/tui/templates/python/program.py /tmp/my.py
TUI2_PYTHON_SDK="$PWD/clients/tui/sdk/python" \
  bash clients/tui/scripts/dev.sh /tmp/my.py
```

TS 模板用相对路径 `require('../../sdk/ts')`，复制走时把 `require` 改成
SDK 目录的绝对路径（或 `npm link`/`NODE_PATH`）。

## 4. 动手改：标题、按钮、面板（10 分钟）

以 Go 模板为例（Python/TS 是同一结构，按注释找同名方法）。

### 4.1 改标题（1 分钟）

`templates/go/main.go` 的 `header()`：

```go
header := sdk.Row(sdk.Text(" main ").Style("tab_inactive")).ID("header").Height(1)
```

把 `" main "` 改成 `" my-app "`，保存。**预期**：footer 出现
`reloaded ...`，标题立即变化——这就是热重载。

### 4.2 加一个可点击按钮（5 分钟）

在 `view()` 的 header 里追加一个盒子（放在 `header.Child(" + ")` 之前）：

```go
header.Child(sdk.Text(" [ping] ").ID("btn:ping").Style("tab_active").Input("mouse"))
```

在 `mouse()` 的 `switch` 里加一个分支：

```go
case node == "btn:ping":
    a.status = "pong"
```

保存。**预期**：header 出现 `[ping]`，鼠标点一下 footer 显示 `pong`。
要点：**可点击 = 稳定 id + `input:["mouse"]` + `mouse()` 里处理节点**，
三件套缺一不可。

### 4.3 加一个面板（4 分钟）

在 `view()` 里给根布局追加一行（footer 之前）：

```go
panel := sdk.Row(sdk.Text(" side panel ").Style("selection")).ID("panel").Height(1)
root := sdk.Col(header, body, panel, footer)
```

或做一个右侧栏：把 `body` 包成 `sdk.Row(body, panel.Flex(0).Width(24))`。
保存后画面立即重排；终端 PTY 尺寸会自动跟随新的内容矩形。

### 4.4 Python / TS 的同一处修改

- Python：`header()` 里 `row.child(builder.text(" [ping] ").id("btn:ping")...)`，
  `on_mouse` 里加 `elif node == "btn:ping": self.status = "pong"`。
- TS：`header()` 里 `row.child(builder.text(' [ping] ').id('btn:ping')...)`，
  `onMouse` 里加 `else if (node === 'btn:ping') this.status = 'pong'`。

## 5. 自测：三套工具（5 分钟）

```sh
# 1) 语言/模板能编译、能加载
go build ./clients/tui/templates/go
node --check clients/tui/templates/ts/program.js
python3 -m py_compile clients/tui/templates/python/program.py

# 2) SDK 一致性（官方 Go/Python/TS 三语言，10 个协议 fixtures）
bash clients/tui/scripts/sdk-verify.sh
# 期望：三行 sdk-verify: 10/10 fixtures passed + "all bindings passed"

# 3) 冒烟（10 项）+ 全量黑盒验收（含模板/dev 段）
bash clients/tui/scripts/smoke.sh
bash clients/tui/scripts/acceptance.sh
# 期望：冒烟 10 passed；验收 0 failed

# 4) 测试驱动本身（两种驱动同一批断言 + 无 tmux 自动回退）
bash clients/tui/scripts/driver-parity.sh
# 期望：32 passed, 0 failed（pty/tmux 冒烟各 10/10、关键类别各 13/13、
#        无 tmux PATH 下自动回退 10/10、一致性 diff 为空）
```

验收脚本里与本教程直接相关的新增断言（都可单独复现）：

- `template/go: go build`、三语言各自 `tab bar/footer/picker/bound term-1/退出`；
- `dev: frame log decodes both directions`（`-protocol-log` 文件里同时有
  `"dir":"host->program"` 与 `"dir":"program->host"`）；
- `dev: saving the watched file hot-reloads (chrome changes)`（改文件后画面变化）；
- `dev: crash notice carries the reason and restart countdown` +
  `DEV-CRASH-MARKER`（崩溃程序重启后 notice 可见，同时写默认帧日志）。

### 5.1 测试驱动：tmux 或内置 PTY（二选一）

`smoke.sh`/`acceptance.sh` 共用 `scripts/libdriver.sh` 驱动层（`spawn`/`send_keys`/
`capture`/`capture_osc52`/`resize`/`kill`）：

| 环境 | 驱动 | 说明 |
|---|---|---|
| 装有 tmux（默认） | `tmux` | 隔离 socket/XDG，不碰你的配置；行为与历史验收完全一致 |
| 无 tmux | `pty` | 自动回退内置 Go/PTY harness `cmd/tui2-harness`，跑同一批断言 |
| `TUI2_TEST_DRIVER=tmux\|pty` | 强制 | 本地/CI 都可显式选择；`auto` 为默认 |
| `TUI2_HARNESS_BIN=<path>` | pty | 指定预编译 harness，无 go 也能跑 |

pty 驱动覆盖 attach/输入/split/picker/回看复制（OSC52）/resize/退出等全部关键
类别，抓屏/原始 SGR 抓屏/光标/剪贴板与 tmux 同语义；无 tmux 时不可用的断言会
打印 `SKIP` 和原因，不会静默跳过。目前仅一条：`policy: pruned endpoint has a
one-line notice`（布局程序只有一个瞬时 notice 槽，后续端点 notice 可能先覆盖
它；同内容的持久日志断言仍执行）。两者通过数对比与同用例一致性抽样见
`scripts/driver-parity.sh`。脚本会自动重置 SIGHUP（GNU `env --default-signal`），
所以 `nohup` 下也能稳定重跑——否则被 daemon 杀掉的 `sh` 会继承 SIG_IGN 并让
kill 用例误报。

## 6. 常见问题

| 现象 | 原因/解决 |
|---|---|
| `ModuleNotFoundError: tui2sdk` | 复制出了仓库；设 `TUI2_PYTHON_SDK=<repo>/clients/tui/sdk/python` |
| 画面空白、footer 有 `Traceback` | 程序启动即崩溃；dev 模式 notice 里有 stderr 末尾行，或看 `tui2-dev.log` |
| 保存后没重载 | `-watch` 的路径不对：`dev.sh` 目录目标会监视整个目录；`-watch` 支持文件或目录、可重复/逗号分隔 |
| 日志文件在哪 | 帧日志 `.../anytty/tui2-dev.log`；宿主诊断 `.../anytty/tui2.log`（`-log-file`/`TUI2_LOG_FILE` 可改） |
| 点了没反应 | 盒子缺 `id`、缺 `input("mouse")` 或 `mouse()` 没处理该 id |
| 终端里出现日志行 | 不可能：日志走文件；若看到请报 bug（验收有抓屏断言） |

## 7. 下一步

- 协议细节：`PROTOCOL.zh-CN.md`；组件/props：`SDK.zh-CN.md` §5；
- 完整参考实现：`cmd/tui2-shell`（默认布局）、`examples/python-shell/`
  （纯协议参考 + 老 UI 像素复刻）；
- 用自己的语言写 SDK：`SDK.zh-CN.md` §4 的 AI 生成流程 +
  `conformance/README.zh-CN.md` 的被测程序契约；
- 验收覆盖范围与里程碑：`PROGRESS.zh-CN.md`、`SCENARIOS.zh-CN.md`。
