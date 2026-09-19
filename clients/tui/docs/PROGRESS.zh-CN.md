# TUI v2 进度台账（PROGRESS）

> 规则：本文只记**状态、验收方式与队列**；协议/架构以 `PROTOCOL.zh-CN.md`、
> `ARCHITECTURE.zh-CN.md`、`SCENARIOS.zh-CN.md` 为准，正文不在本文重复。
> 最近更新：2026-09-18（M33 测试驱动去 tmux 强依赖：`scripts/libdriver.sh`
> 统一原语（spawn/send_keys/capture/capture_raw/cursor/OSC52/resize/kill），
> 默认 tmux、自动回退新 `cmd/tui2-harness`（clients/tui/pty + render/ansi，
> 固定 cols/rows、SGR 原始抓屏、OSC52 解码、进程组 kill）；smoke/acceptance
> 同一批断言双驱动，`TUI2_TEST_DRIVER=tmux|pty` 强制，新增
> `scripts/driver-parity.sh`（≥6 项：无 tmux PATH 下冒烟真跑、关键类别、
> resize、OSC52、双驱动一致性抽样）；tmux 374/374 不变。详见 §2.20 与
> `TUTORIAL.zh-CN.md` §5.1。）
> M32 TUI 开发体验：M1 定位 acceptance 两个
> killed-terminal 失败的根因是**运行环境**——父进程忽略 SIGHUP（nohup）把
> SIG_IGN 传进隔离 daemon 的交互 `sh`，daemon 的 SIGHUP 进程组 kill 打不死它；
> 脚本现在用 `env --default-signal=HUP` 自愈重执行，前台/nohup 均稳定
> 374/374。M2 宿主新增 `-dev`/`-protocol-log`/`-watch`/`-reload-on-save`：
> 双向帧解码 JSON Lines（时间戳+方向，默认 `$XDG_STATE_HOME/anytty/
> tui2-dev.log`，终端零日志）、stderr 环形缓冲、崩溃原因+倒计时+stderr
> notice、保存热重载保留最后一棵好树 + `reloaded <file>`；修掉热重载触发的
> 假崩溃重启（sessionEnd 世代校验）。M3 三语言起点模板
> `clients/tui/templates/{go,python,ts}`（各 100–200 行：tab 条/两槽/焦点
> 点击/footer/picker/Esc·Ctrl-Q）+ `scripts/dev.sh` 一键循环 +
> `docs/TUTORIAL.zh-CN.md` 半小时教程；acceptance 新增 39 项，总
> **374/374**；详见 §2.19。）
> M30 日志接管 + 路由裁剪：宿主启动最早期把 Go 标准
> logger 重定向到 `$XDG_STATE_HOME/anytty/tui2.log`（`--log-file`/
> `TUI2_LOG_FILE` 可覆盖），共享层 `anytty connect/network attempt/webrtc`
> 与 host 异常全部进文件 + notice，alt screen 不再出现任何日志；默认只拨
> `local-unix` 与“凭据可用”的 `ssh-webrtc-tcp`，老 registry 的
> direct/cloud route 不尝试、不阻塞 4 秒，降级顺序 local > ssh > direct >
> managed，纯 webrtc/cloud 端点 offline + 一行 notice；`TUI2_ROUTES`/
> `-routes` 显式开启 direct/cloud；acceptance 335/335（新增 22 项：抓屏无
> 日志、日志文件诊断行、offline notice、降级 attach、启动失败/退出无污染），
> parity 31/31、v3 parity 12/12、冒烟 10/10；说明见 `REMOTE.zh-CN.md` §3.1。
> M29 dev 直读老 registry：宿主新增只读 registry 覆盖
> `TUI2_ENDPOINTS`/`-endpoints`，显式文件在前、按 name 合并（显式优先），
> 缺文件/坏格式只发可读 warning 不崩，dev registry 按不同 name 并存，老
> registry 永不写回；dev picker 直接列出/attach/输入老 XDG 的 endpoint 终端，
> acceptance 313/313（新增 15 项）；runbook 见 `REMOTE.zh-CN.md` §4.0.1。
> M28 连接层去重收官：tui2 删除自有裸帧 wire client 与
> zstd tcp 帧生产实现，全部 daemon 连接（local-unix/tcp/direct/ssh/cloud）经
> `client/` 共享层；host 启动读 CLI registry，picker 列出配对端点/离线角标；
> CLI→registry→TUI attach/输入/重启端到端 + 15 项新验收 + 守卫，acceptance
> 298/298；详见 §2.16 与 `CLIENT_SHARING.zh-CN.md`。
> M26 v3 复刻分屏树 + footer 每键颜色：`v3ui.py` 的分屏
> 从“flat panes + 全局 flow”改成**递归切分树** `Leaf | Split(orient, ratio, a, b)`，
> split 只替换聚焦叶子，每 leaf 一张完整 card、兄弟间 1 格可拖拽分隔条且只写回
> 该 Split 的 ratio；关闭提升兄弟、外部 resize 比例保持；footer 按老
> `shortcutActionStyle`→`footerActionKeyStyle` 解析链给 badge/每键/badge/右段
> 各自显式颜色（coralline-candy token→RGB 考古表见 V3_PARITY §1.4）；v3 parity
> 12/12（新增 `left1right2` 逐字符 golden + `footer_colors` 三场景逐块断言/异色
> 计数）、acceptance 249/249（新增 23 项：左1右2 三 leaf stty/拖拽隔离/关闭恢复/
> footer 颜色块/外部 resize 比例），parity 31/31、冒烟 10/10。
> M25 v3ui 分屏/浮窗真实可用：`v3ui.py` 的 split 从
> “只改 chrome”补成真实多 pane 布局（`flow` + `weights` + 1 cell 可拖拽分隔条、
> 每 pane 独立 terminal/PTY、焦点与输入隔离、split 自动 create）；floating 补成
> 真实 pos 窗口（新建自动建终端/绑定、标题整行拖拽、折叠只留标题行、关闭解绑、
> z 序提升、浮窗/主 pane 焦点切换）；框架补第 4 个缺口——重叠 `pos` overlay 的
> placement 归属改为最后声明者（否则浮窗终端被自身 overlay 擦空）；验收 226/226
> （新增 31 项）、v3 parity 10/10（新增 split_col/split_row/float chrome + golden）、
> parity 31/31、冒烟 10/10；规格/缺口见 `V3_PARITY.zh-CN.md`。
> M27 连接远程 terminal：考古老 v3 endpoints.yaml 的
> `local-unix`/`direct-webrtc-tcp`/`ssh-webrtc-tcp`/`managed-webrtc` 四类 route
> 与 daemon 监听（wire 只在 unix socket；`--route` 是 WebRTC signaling/ICE），
> v2 落地 Dialer 抽象（`local-unix`/`tcp`）与 `tcp` 帧传输（与
> `shared/transport/unix` 字节兼容，单测双向验证），`MethodParams.address`(20)
> 打通 shell/host；P0 隧道（`ssh -L local.sock:remote.sock` 与
> `ssh -L TCP→remote.sock`）与老命令式 `ssh host anytty ...` 全通，断线
> offline/notice、退避重连、快照重建、resize owner CAS 在 tcp 链路复用；
> 新增第二隔离 daemon + 真实 ssh 隧道端到端验收（acceptance 283/283，新增 34 项），
> 规格/考古/手动命令见 `REMOTE.zh-CN.md`。
> M24 老 v3 TUI 像素复刻：新增纯 stdlib
> `clients/tui/examples/python-shell/v3ui.py`，按 recommended/coralline-candy 复刻
> powerline 顶条、card 窗口题字/按钮、`┃ Click to collapse` 提示、footer
> `󰌌 …` 场景键组与 floating/picker；离线 `v3_parity_test.py` 用独立公式 +
> `1.txt` 行 oracle 做 7 项逐字符校验并落 golden；同时补齐三个框架缺口
> （`chrome.inset` 无边框 terminal、`ctrl-shift-*`/`ctrl-alt-数字` key 名、
> `pos` overlay 内组件的 z 序合成）；验收 195/195（新增 15 项）、parity 31/31、
> v3 parity 7/7、冒烟 10/10；规格/缺口见 `V3_PARITY.zh-CN.md`。
> M23 daemon 连接策略 endpoint：新增 `clients/tui/endpoint`
> （wire7 + `shared/transport/unix` 的 local-unix 客户端：list/create/attach/
> 输入/resize owner CAS/kill/remove/restart/detach，断线退避重连 + health/notice，
> 重连后重 attach 并用 `LiveScreenNext(0)` 快照重建画面不复播），host 把 daemon
> 终端映射为同形 `runtime.Terminal`（位置透明），picker 按 endpoint 分组，
> `MethodParams` 新增 `kind/socket/connect_mode`(17..19) 与 `endpoint.sync`；
> 验收 180/180（新增 28 项，真实隔离 dev daemon）、parity 31/31（新增分组 picker
> 兼容检查）、冒烟 10/10；规格/考古/支持矩阵见 `ENDPOINTS.zh-CN.md`。
> M22 footer 场景化：根因是 v2 把键名/动作名当标签渲染（
> （`split-h`/`split-v` 之类，等价于老 resize 场景会显示 `spacebar`）且未按场景
> 裁剪；新增 `cmd/tui2-shell/footer.go` 单一规格表（label 优先、规范化键名、
> 只取当前场景已实现键位），推荐 badge/label/icon/顺序对齐老 yaml，新增
> `golden/recommended_footer_120x32.txt`（8 场景）由 Go 最终帧与 legacy.py
> `--footer-lines` 两侧逐字符比对；验收 152/152（新增 5 项），parity 30/30，
> 冒烟 10/10；规格表与差异见 `RECOMMENDED_CONFIG.zh-CN.md` §3.1。
> M21 老推荐 TUI 配置落地：默认主题/图标 = 老
> coralline-candy 推荐配置（Nerd Font 单宽码点，`icons` preset/map 可配），
> keybindings 新增 `pane.split_h`/`pane.split_v` 并把推荐键位映射表写进
> `RECOMMENDED_CONFIG.zh-CN.md`，`endpoints`（name/kind/argv/cwd/env）命令式
> endpoint 接入 picker + `terminal.create`，`legacy.py` 支持 `-config` 推荐
> preset；验收 147/147（新增 14 项），parity 29/29，冒烟 10/10。
> M20 老 UI 像素复刻：纯标准库 Python 程序
> `clients/tui/examples/python-shell/legacy.py` 复刻 `shell/main.go` 的静态像素与交互，
> `parity_test.py` 用老公式独立重算并与程序 view 逐格对比（14 场景 × 120x32/100x30
> 全过，golden 落盘），验收新增 28 项 legacy 断言（133/133），并修补复刻中发现的
> 退出角标零码语义；视觉规格与缺口清单见 `LEGACY_PARITY.zh-CN.md`；冒烟 10/10。
> M19 布局程序与框架零依赖：`clients/tui/sdk` 不再 import
> kernel/runtime/render/components（依赖守卫测试 `sdk/deps_test.go`），宿主
> `-shell` 支持"命令+参数"并诊断立即退出，新增纯标准库 Python 参考程序
> `clients/tui/examples/python-shell/`；验收 105/105（Python 新增 14 项，见 §2.6），
> 冒烟 10/10。
> M18 外部尺寸跟随 + 鼠标点击补全：宿主 SIGWINCH →
> `resize` 事件 → 程序重排（`Session.Resize` 去重）；picker 单击即绑定、
> 侧栏 tab 行可点；验收 91/91（新增 10 项，见 §2.5），冒烟 10/10。
> M17 槽标题栏动作按钮：程序自绘 1 行标题栏 + `⟳ ⇔ ⇕ ✕`
> 四个盒子按钮，press/pending 状态、键盘等价与文档；验收 81/81，冒烟 10/10。
> M16 组件 props 契约：程序下发 chrome 样式，terminal 边框随
> shell 主题变化；删除 `proto.Border` 残渣；M15 架构收口：rect 纯几何、显式样式、
> 程序侧主题、widget 自绘 chrome；M14 美化与配置化）。

## 1. v1 完成定义（"可日常使用"验收清单）

同时满足以下全部条目，v1 才算完成；每项都要能现场演示，不以"代码存在"代替。

- [x] 终端 attach / 输入 / 输出：聚焦终端可打字、程序输出实时上屏、resize 跟随内容矩形
      （tmux 冒烟 `echo hi` 回显 + `Host` 假 PTY 尺寸断言；resize 跟随后续黑盒）
- [x] split / 关槽：`%`/`"` 双槽空占位、空槽点击/Tab/绑定、关槽解绑式可重绑
      （验收 §2 + §9 拖拽；单测覆盖）
- [x] tab：`Ctrl-T` 新 tab、`1..9` 切换、后台不丢状态（验收 §10）
- [x] picker：全局键弹出选择器，可取消/确认（冷启动 picker + `+ New terminal` + 重绑已有终端）
- [x] prompt：`:` 命令行提示与输入回显（`help`/`quit`/`kill`，验收 §5）
- [x] 回看 / 复制：`PgUp`/滚轮 `[↑N]`、`y` 复制经 OSC52 落 tmux buffer、`esc` 回 live（验收 §4）
- [x] 退出：`system.quit` 与宿主确认；`esc` 拒绝仍可用、`enter` 干净退出（验收 §13.5）
- [x] 重启：`exit 7` 角标 `[exited 7]` + `Ctrl-E` 同 id 重启可输入；`:kill` 授权后同路径（验收 §3/§5）
- [x] 崩溃恢复：kill -9 → 画面保留 → 新 epoch 重启 → 默认布局重建 + 终端仍可用；
      确认框打开时 kill → 新 epoch 清 overlay、旧意图不执行（验收 §3/图 9.7）
- [x] 快捷键与鼠标：claim 键不落 PTY、未 claim 键落 PTY、双击 `input.forward`（`cat -v` 见 `^F`）、
      滚轮三条件（无 tracking 归程序出 `[↑N]`；tracking+焦点+声明透传见 `^[[<64;`）、
      分隔条横/纵拖拽 + `stty size` 同步（验收 §2b/§9）
- [x] 单机本地：endpoint 只含 local，无远程依赖即可完成以上全部（本轮全部在本地完成）

## 2. 里程碑清单

| 里程碑 | 状态 | 如何验证（一行） |
|---|---|---|
| M0 规格三件套（协议/架构/场景，术语统一） | 已完成 | 通读 `clients/tui/docs/*.zh-CN.md`，四份文档术语表一致 |
| M1 内核 kernel（盒子/解算/命中/合成/光标） | 已完成 | `go test -count=1 -race ./clients/tui/kernel/` |
| M2 协议帧与 protobuf（HELLO/VIEW/EVENT/RESULT/RESPONSE） | 已完成 | `go test -count=1 -race ./clients/tui/proto/` |
| M3 渲染与 ANSI 解析（diff 帧、SGR、模式位） | 已完成 | `go test -count=1 -race ./clients/tui/render/...` |
| M4 terminal 组件（渲染/滚窗/复制/角标） | 已完成 | `go test -count=1 -race ./clients/tui/components/terminal/` |
| M5 PTY（真实 shell、resize、关闭幂等） | 已完成 | `go test -count=1 -race ./clients/tui/pty/` |
| M6 runtime 会话（epoch/rev、方法表、sources、路由、z 序合成） | 已完成 | `go test -count=1 -race ./clients/tui/runtime/` |
| M7 输入编码 `runtime/keys` + forward 历史（64/视图） | 已完成 | `go test -count=1 -race ./clients/tui/runtime/keys/` |
| M8 输出失效通知（PTY 输出 → revision++ → 无输入也重绘） | 已完成 | `go test -count=1 -race -run TestScreenRevision ./clients/tui/runtime/` |
| M9 路由→PTY/程序接线（Input、input.forward、claim/all） | 已完成 | `go test -count=1 -race -run 'TestInput\|TestForward' ./clients/tui/runtime/` |
| M10 布局程序 `cmd/tui2-shell`（默认布局 + tab/pane/picker/prompt/help/滚动状态机） | 已完成（本轮） | `go test -count=1 -race ./clients/tui/cmd/tui2-shell/` + 冒烟 |
| M11 宿主入口 `cmd/tui2`（raw mode、SIGWINCH、帧循环、core overlay、创建/重启接线） | 已完成（本轮） | `go test -count=1 -race ./clients/tui/cmd/clients/tui/` |
| M12 tmux 黑盒验收（SCENARIOS §13 单机项） | 已完成（78/78 通过） | `bash clients/tui/scripts/acceptance.sh`；缺口：租约/多客户端、背压、基准、文档注册表 |
| M13 v1 日常可用验收 | 进行中（第 1 节仅剩上条缺口） | 按第 1 节清单打勾，全部通过 |
| M14 美化与配置化（主题 token / widget 库 / 配置文件） | 已完成 | `go test ./clients/tui/...` + `bash clients/tui/scripts/acceptance.sh`（78/78）；配置见 `CUSTOMIZE.zh-CN.md` |
| M15 架构收口（rect 纯几何 / 显式样式 / 程序主题 / widget chrome） | 已完成（本轮） | `gofmt -l tui2 && go vet ./clients/tui/... && go test -count=1 -race ./clients/tui/... && bash clients/tui/scripts/smoke.sh && bash clients/tui/scripts/acceptance.sh`（78/78） |
| M16 组件 props 契约（程序下发 chrome 样式 + `proto.Border` 残渣清理） | 已完成（本轮） | `go test -count=1 -race ./clients/tui/...`；shell 主题 props 断言见 `cmd/tui2-shell/view_test.go`，端到端见 `cmd/clients/tui/host_test.go`；验收 78/78 |
| M17 槽标题栏动作按钮（`⟳ ⇔ ⇕ ✕`，程序自绘 + 命中分发） | 已完成（本轮） | `go test -count=1 -race ./clients/tui/...` + `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（81/81，新增 3 项见 §2.4） |
| M18 外部 resize 跟随 + 鼠标点击补全（SIGWINCH→`resize`、picker 单击绑定、侧栏点选） | 已完成 | `go test -count=1 -race ./clients/tui/...` + `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（91/91，新增 10 项见 §2.5） |
| M19 布局程序与框架零依赖（SDK 去 kernel/runtime/render/components、`-shell` 命令+参数、纯标准库 Python 参考程序） | 已完成 | `go test -count=1 -race ./clients/tui/sdk/`（依赖守卫）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（105/105，Python 新增 14 项见 §14） |
| M20 老 UI 像素复刻（`legacy.py` 静态像素 + 交互、parity 逐格 golden、缺口修补） | 已完成 | `python3 clients/tui/examples/python-shell/parity_test.py`（14 场景 × 2 视口）+ `go test -count=1 -race ./clients/tui/...` + `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（133/133，legacy 新增 28 项见 §2.7）；规格/缺口见 `LEGACY_PARITY.zh-CN.md` |
| M21 老推荐 TUI 配置落地（默认主题/图标 = coralline-candy、推荐键位映射、命令式 endpoint、legacy preset） | 已完成 | `go test -count=1 -race ./clients/tui/...` + `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（147/147，新增 14 项见 §2.8）+ `python3 clients/tui/examples/python-shell/parity_test.py`（29/29）；映射表见 `RECOMMENDED_CONFIG.zh-CN.md` |
| M22 footer 场景化（键名≠标签、按场景裁剪、推荐规格逐字符 golden） | 已完成 | `go test -count=1 -race ./clients/tui/...`（含 `TestRecommendedFooterGolden`）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（152/152，新增 5 项见 §2.9）+ `python3 clients/tui/examples/python-shell/parity_test.py`（30/30，含 8 行 footer 逐字符比对）；规格表见 `RECOMMENDED_CONFIG.zh-CN.md` §3.1 |
| M23 daemon 连接策略 endpoint（`clients/tui/endpoint` 真实 local-unix 客户端 + host 位置透明接线 + picker 分组 + 断线重连/health/notice + legacy.py 兼容） | 已完成 | `go test -count=1 -race ./clients/tui/...`（含 endpoint 假 daemon 套件）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（180/180，新增 28 项见 §2.10）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31，新增分组 picker 检查）；规格/考古/矩阵见 `ENDPOINTS.zh-CN.md` |
| M24 老 v3 TUI 像素复刻（`v3ui.py`：powerline 顶条 + card 题字/动作按钮 + 折叠提示 + footer/floating/picker/场景键位；`v3_parity_test.py` 公式 parity + `1.txt` 行 oracle；框架缺口 `chrome.inset`、`ctrl-shift-*` 键名、`pos` overlay 内组件合成） | 已完成 | `gofmt -l tui2` 空 + `go build/vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...` + `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（195/195，新增 15 项见 §2.11）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（7/7）；规格/缺口/差异见 `V3_PARITY.zh-CN.md` |
| M25 v3ui 分屏/浮窗真实可用（多 pane 布局 + 可拖拽分隔条 + 每 pane 独立 PTY + split 自动 create；floating 新建绑定/拖移/折叠只留标题/关闭解绑/z 序/焦点切换；框架缺口 D：重叠 pos overlay 的 placement 归属） | 已完成（本轮） | `gofmt -l tui2` 空 + `go build/vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...`（含 `TestCompositorPlacementFollowsTopmostOverlappingOverlay`）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（226/226，新增 31 项见 §2.12）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（10/10，新增 split/float chrome）；规格/缺口见 `V3_PARITY.zh-CN.md` |
| M26 v3 复刻分屏树 + footer 每键颜色（`Leaf|Split` 递归树只切聚焦叶子、每 leaf 独立完整 card、1 格可拖拽分隔条写回单节点 ratio、关闭提升兄弟、resize 比例保持；footer badge/每键/badge/右段按老 style 解析链显式着色） | 已完成（本轮） | `gofmt -l tui2` 空 + `go build/vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...` + `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（249/249，左1右2 三 leaf stty/拖拽隔离/关闭恢复/footer 颜色/外部 resize 比例）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（12/12，新增 left1right2 + footer_colors）；规格/色表/差异见 `V3_PARITY.zh-CN.md` §1.4/§1.6/§5 |
| M27 连接远程 terminal（Dialer 抽象 `local-unix`/`tcp`、tcp zstd 帧与 shared/unix 字节兼容、`MethodParams.address`(20)、P0 `ssh -L`/命令式 ssh、断线重连/快照/owner CAS 复用、第二 daemon + 真实 ssh 隧道验收、`REMOTE.zh-CN.md`） | 已完成（本轮） | `gofmt -l tui2` 空 + `go build/vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...`（endpoint tcp 帧兼容/生命周期/错误可读）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（283/283，新增 34 项：第二隔离 daemon + ssh tcp/unix 隧道 + Go 桥回退 + 老 ssh 命令式 + 断线重连）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（12/12）；考古/矩阵/手动命令见 `REMOTE.zh-CN.md` |
| M28 连接层去重收官（tui2 只读 registry + 共享 `client/` 拨号：`sharedClient`/`sessionConn`、tcp 桥、host 启动加载 endpoints.yaml + picker 端点占位/角标；删除 tui2 裸帧 client 与 zstd tcp 帧生产实现；CLI→registry→TUI attach 验收 + 守卫） | 已完成（本轮） | `gofmt -l tui2` 空 + `go build ./...` + `go vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...`（`shared_stack_test.go` 真实共享栈重连；`client_sharing_test.go` go-list 守卫/行数）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（298/298，新增 15 项：CLI registry→TUI 列出/attach/输入/exit/Ctrl-E/离线角标 + 守卫；既有 tcp/ssh 隧道回归）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（12/12）；最终态见 `CLIENT_SHARING.zh-CN.md` |
| M29 dev 直读老 registry（`TUI2_ENDPOINTS`/`-endpoints` 只读多文件合并：显式优先、缺/坏文件可读 warning、dev registry 并存、老 registry 不写回；dev picker 直接列出/attach/输入老 XDG 端点终端；不改 `client/`） | 已完成 | `gofmt -l tui2` 空 + `go build ./...` + `go vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...`（`registry_override_test.go` 合并/优先级/缺坏文件/只读 + host 环境变量接线）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（313/313，新增 15 项：老 XDG 直接列出/attach/输入 + 同名显式优先 + dev 并存 + 只读 md5 + missing/corrupt 可读 warning 不崩）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（12/12）；runbook 见 `REMOTE.zh-CN.md` §4.0.1 |
| M30 日志接管 + 路由裁剪（宿主 `--log-file`/`TUI2_LOG_FILE` → `$XDG_STATE_HOME/anytty/tui2.log` 重定向 Go 标准 logger 与共享层；host 诊断走文件+notice；默认只拨 local-unix + 凭据可用的 ssh，direct/cloud 由 `TUI2_ROUTES`/`-routes` opt-in；老 registry 降级 local>ssh>direct>managed，纯 webrtc/cloud offline+notice，不刷屏不阻塞） | 已完成（本轮） | `gofmt -l tui2` 空 + `go build ./...` + `go vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...`（`logging_test.go` 重定向/默认路径/flag 优先级；`policy_test.go` 解析/裁剪/降级投影；host 诊断只进 Logf）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（335/335，新增 22 项：抓屏无日志、默认/opt-in 日志文件诊断行、offline notice、降级 attach、启动失败与退出无污染）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（12/12）；策略与查日志见 `REMOTE.zh-CN.md` §3.1 |
| M31 SDK 三层 + 一致性套件（core/builder/widgets；Go/Python/TS 三绑定；`tui2-sdk-verify` + fixtures；`docs/SDK.zh-CN.md`） | 已完成 | `go test -count=1 -race ./clients/tui/...`（三语言 conformance 测试）+ `go run ./clients/tui/cmd/tui2-sdk-verify` 与 `--cmd` Python/TS（各 10/10）；详见 §2.18 |
| M32 TUI 开发体验（M1 acceptance SIGHUP 环境根因修复；M2 `-dev` 热重载/帧日志/报错可见；M3 三语言起点模板 + TUTORIAL） | 已完成 | `gofmt -l tui2` 空 + `go build ./...` + `go vet ./clients/tui/...` + `go test -count=1 -race ./clients/tui/...`（`dev_test.go` 帧日志/方向/时间戳、watcher、stderr、崩溃 notice、reload notice）+ `bash clients/tui/scripts/smoke.sh`（10/10）+ `bash clients/tui/scripts/acceptance.sh`（374/374，新增 39 项：三模板 SDK 加载/关键 chrome/建终端/退出 + dev 帧日志双向解码/终端无日志/热重载/崩溃可见 + 默认帧日志；nohup 与前台各一次）+ `python3 clients/tui/examples/python-shell/parity_test.py`（31/31）+ `python3 clients/tui/examples/python-shell/v3_parity_test.py`（12/12）；教程见 `TUTORIAL.zh-CN.md` |
| M33 测试驱动去 tmux 强依赖（共享驱动层 `scripts/libdriver.sh`：spawn/send_keys/capture/capture_raw/cursor/OSC52/resize/kill；tmux 首选、无 tmux 自动回退内置 Go/PTY `cmd/tui2-harness`；`TUI2_TEST_DRIVER=tmux\|pty` 强制；smoke/acceptance 同断言双驱动 + pty 能力 SKIP 说明；新增 `scripts/driver-parity.sh` 与 harness 单测） | 已完成（本轮） | `go build ./...` + `go vet ./...` + `go test -count=1 -race ./clients/tui/...`（`testing/harness` 键映射/SGR 抓屏/OSC52 分片/真实 PTY 往返/进程组管理）+ `bash clients/tui/scripts/smoke.sh`（10/10，tmux 与 pty）+ `bash clients/tui/scripts/acceptance.sh`（tmux 374/374；无 tmux 见 §2.20 对比）+ `bash clients/tui/scripts/driver-parity.sh`（无 tmux PATH 冒烟、关键类别 13/13、resize/OSC52、一致性抽样）+ `sdk-verify` 三语言 + parity 31/31 + v3 parity 12/12；驱动选择见 `TUTORIAL.zh-CN.md` §5.1 |

### 2.1 M14 新增（外观与配置，内核语义不变）

- **M1 主题与语义 token**：`clients/tui/render/style.go` 提供
  `bg fg muted accent accent_dim warning danger ok chrome chrome_focus
  tab_active tab_inactive status border border_focus border_dead selection`
  以及 `ThemeDark()`（默认，避免纯黑纯白）/`ThemeLight()`/`ThemeByName()`；
  `ANSIToken/RawSGR` 路径不受影响，新增 golden 测试（render/style_test.go）。
- **M2 视觉改版**（全在布局程序）：tab 条活动块 `tab_active` + `+` 新建、
  槽边框 `border_focus`/`border`/`border_dead`（终端组件改用语义 token）、
  槽间 1 格 gutter（`gap=0` 可关）、空槽居中提示卡、footer 左快捷键组右状态
  （`│` 分段 + 时钟）、picker/prompt/help 统一浮层视觉（`selection` 选中行、
  picker 增 endpoint/state 信息列）、可选左侧栏（默认开，Ctrl-W 切换）。
- **M3 配置化**：`clients/tui/config` 解析共享配置；`tui2-shell --config`（默认
  `$XDG_CONFIG_HOME/anytty/tui2.json`，`ANYTTY_TUI2_CONFIG` 可覆盖）、
  `--print-default-config`；字段 `theme/sidebar/gap/clock/
  double_click_forward_ms/keybindings/startup`；解析失败降级为默认值并在
  footer 显示一行 `config error:`。主题由 shell 消费（M15 起宿主不读配置文件）。
- **M4 widget 库**：`clients/tui/sdk/widgets` 提供 `TabBar/StatusBar/Card/Button/
  KeyHint`，全部是"配置 → `*sdk.Builder`"的纯函数，不持状态、不发方法；
  点击由调用方命中后处理。
- **协议与内核**：`box.style`（field 12）为**追加字段**，用于文本样式；
  内核仅在 `box.style` 非空时改变文本颜色，默认值/命中/z 序与 v1 完全一致。

### 2.2 M15 架构收口（chrome 走出内核，样式走出宿主）

- **rect 纯几何**：删除内核 `Border` 字段、边框渲染与内容内缩；盒子内容区 = rect 全量，
  固有尺寸 = 内容尺寸；相关测试改为纯几何断言（边框渲染测试迁到组件/程序侧）。
- **显式样式**：`clients/tui/render` 支持 `fg:#RRGGBB;bg:#RRGGBB;bold;dim;italic;underline;reverse`
  的主题无关样式串，直接翻译成 SGR；未知片段安全降级（忽略、不着色、不报错）；
  内建 token 仅供宿主内部组件（terminal 边框 / core overlay）使用。golden 测试覆盖。
- **组件 chrome**：terminal 组件声明 `Inset=1` 并自绘边框/标题/角标（视觉不变）；
  runtime 与 `cmd/tui2` 用 `rect − 组件 inset` 算 PTY winsize 与光标偏移，
  不再假设 "W>=3 减 2"。
- **程序主题与配置**：`tui2-shell` 内置 dark/light 主题表（→ 显式样式），
  `tui2.json` 的 `theme` 由 shell 消费；宿主不再读 theme、不再有配置文件，
  `cmd/tui2` 只留 `-shell/-version/-help`。
- **widget chrome**：`sdk/widgets` 新增 `Frame`（程序侧自绘边框：题字顶栏 + 侧边条 + 底栏）
  与 `Divider`；`Card` 改由 `Frame` 实现；shell 的 tab 条/侧栏/卡片/浮层/footer/分隔线
  全部走 widget 或自绘文本行，不依赖内核 border。
- **测试**：kernel/render/widgets/shell 断言同步更新；`go test -count=1 -race ./clients/tui/...`
  全绿，smoke 10/10，验收 78/78（未减少用例）。

### 2.3 M16 组件 props 契约（外观决策彻底归程序）

- **协议**：`content` 新增可选字段 `props`（`map<string,string>`，wire field 4）——程序 →
  组件的属性/样式通道；`proto.Border` 消息与 `box.border`（field 6）删除并 `reserved`，
  生成代码（`clients/tui/protobuf/tui2.pb.go`）与 `clients/tui/proto/tui2.proto` 同步重生成。
- **透传**：`kernel.Content.Props` 原样承载；runtime `toNode` 拷贝进解算树，新增
  `Session.BoxProps(id)` 供宿主取用，`runtime.Placement.Props` 带上组件收到的 props；
  宿主组件工厂只转发不解释（宿主仍然没有色板/主题配置）。
- **terminal 组件**：识别 `chrome.border` / `chrome.title` / `chrome.border_focus` /
  `chrome.border_dead` / `chrome.badge`（显式样式串）；按
  focused→border_focus、exited→border_dead、否则 border 选色，标题用 chrome.title、
  角标（`[exited N]`/`[↑N]`）用 chrome.badge；未识别 key 忽略，缺省回落内置默认
  （无 props 的渲染与 M15 golden 逐字节一致）。
- **shell 主题**：`theme.terminalChrome()` 把 dark/light 表解析成上述 props 挂到每个
  终端槽；`tui2.json` 切 `theme` 后终端边框/标题/角标颜色随之变化。
- **测试**：terminal golden/覆盖（有 props 用 props、无 props 用默认）、runtime props
  透传与拷贝、sdk `Builder.Props` 快照、shell 视图断言、host 端到端断言（显式 SGR 上屏）。

### 2.4 M17 槽标题栏动作按钮（按钮=盒子+程序动作，不新增框架概念）

- **视图**：每个 slot 的标题栏是程序自绘的 1 行 `row`（`slotTitleBar`），作为 slot 的
  `pos` 子盒压在 terminal 组件自绘的边框题字上（组件不感知，slot 几何与 PTY 尺寸不变）：
  左侧 `▎标题`（focused→`border_focus`、exited→`border_dead`、否则 `muted`；角标
  `[exited N]`/`[↑N]` 用 `warning`），右侧 `⟳ ⇔ ⇕ ✕` 四个 1 格按钮（间隔 1 格）。
  主题新增 `button`/`button_hover`（焦点槽）/`button_pressed`（press 帧）；退出态
  `⟳` 用 `warning` 提示可重启。
- **动作与状态**：mouse press 命中 `btn:<slot>:<action>` → 焦点切到该槽并执行
  `⟳`=terminal.restart（无终端 no-op+toast）、`⇔`/`⇕`=等价 `%`/`"` 的 split、
  `✕`=等价 `x` 的解绑式关闭；press 置 `pressed`（release 或重绘清除），restart 挂起时
  `slot.pending` 在标题右侧显示 `…`，响应回调清除。
- **键盘等价**：PANE 下复用现有绑定（`Ctrl-E` restart、`%`/`"` split、`x` close），
  未新增按键；文档见 `CUSTOMIZE.zh-CN.md` §5.1（完整代码 + 主题/props 说明）。
- **测试**：视图树断言每个 slot 含 4 个按钮 id 与主题样式、退出态 `⟳` 为 `warning`；
  分发表驱动 4 个按钮 → 正确动作/方法参数；press/release/pending 生命周期；
  无终端 restart no-op+toast；程序侧 press 只高亮一帧（commit 后清除）。
- **验收**：新增 3 项（图标可见；`✕` 解绑后 picker 可重绑；`exit 7` 后点 `⟳` 恢复可输入），
  总 81/81。

### 2.5 M18 外部尺寸跟随与鼠标点击补全（两处根因）

- **Bug 1（外部 resize 不跟）根因**：`runtime.Session.Resize` 只重解旧 VIEW、重置
  diff 基线，从未把视口变化作为协议 `resize` 事件发给布局程序；shell 的
  `onResize`/`model.resize` 因此永远不触发，旧 VIEW 里显式声明的槽宽高（按旧视口
  算出）让 placement rect 与 PTY winsize 钉在旧尺寸。修复：`Session.Resize` 在尺寸
  真正变化时更新视口/合成器、重解旧树并发 `resize{cols,rows}`（重复 SIGWINCH 去重）；
  其余链路本已正确（程序重排 → 新 VIEW → runtime 解算 → `rect − 组件 inset` →
  `Terminal.Resize` → `stty size` 跟随）。单测 `TestSessionResizeEmitsResizeEvent`
  （runtime）与端到端 `TestHostResizeFollowsProgramReflow`（cmd/tui2）锁定该链路。
- **Bug 2（点击大多无效）根因**：程序侧 `model.mouse` 只覆盖 `tab:*`/`slot-*`/
  `btn:*`/`divider:*`/`title:*`：picker 行 `pick:<i>` 只移动选中索引、不执行绑定
  （键鼠两套语义不等价）；侧栏 tab 行既没有稳定 id、也没有 `input:["mouse"]`
  声明，宿主命中后落到没有处理器的父盒子上。修复：`pick:<i>` 的 press 直接走与
  enter 相同的 `m.pick`（一次点击完成选择 + 绑定）；侧栏 tab 行改为
  `side:tab:<i>` + `input:["mouse"]`，press 调 `selectTab`；可观察反馈沿用
  `selection` 主题（侧栏选中行）、`border_focus`（槽聚焦标题/边框）与
  `binding`/`bound <id>` 状态行。
- **测试**：`TestPickerMouseClickBinds`、`TestSidebarMouseClickSwitchesTab`（shell）、
  `TestHostResizeFollowsProgramReflow`（host）、`TestSessionResizeEmitsResizeEvent`
  （runtime）；验收新增 10 项（外部 resize 3、槽点击聚焦高亮 2、picker 单击绑定与
  焦点 3、侧栏行可点与切换 2），总 91/91；smoke 10/10。

### 2.6 M19 布局程序与框架零依赖（协议是唯一契约，Python 参考实现）

- **根因**：`clients/tui/sdk`（含 widgets）借用 `kernel.DisplayWidth/Truncate`，Go 程序
  `cmd/tui2-shell` 还 import `render`/`components/terminal` 取样式类型与 props key，
  于是"布局程序"必须链接宿主内部包，协议之外多了一条隐性依赖。
- **M1 SDK 解耦**：`sdk/width.go` 自带显示宽度表；widgets 与 Go shell 全部改用
  `sdk.DisplayWidth/Truncate`；shell 主题改为本地 `style` 类型 + `chrome.*` 字面量
  key。新增 `sdk/deps_test.go`：解析 `go list -deps ./clients/tui/sdk/...` 断言只含
  stdlib+proto+protobuf；并断言 `cmd/tui2-shell` 不含 kernel/runtime/render/components。
- **M2 宿主**：`-shell` 支持"命令+参数"（引号/转义切分，`resolveShell`）；命令不存在
  在接管终端前报错退出；程序立即退出（<1s）打印明确提示并按既有重启策略处理。
  单测 `main_test.go`（切分/解析/错误）与 `TestHostRestartsImmediatelyExitingProgram`。
- **M3 Python 参考程序**：`clients/tui/examples/python-shell/{pb.py,shell.py,README.md}`，
  纯标准库：`pb.py` 手写 varint/长度分隔编解码；`shell.py` 实现 tab 条 + 分屏 +
  空槽卡片 + footer + picker，`terminal.attach/create` 走 RESULT 并解析 RESPONSE。
- **M5 验收**：acceptance.sh 新增 Python shell 段 14 项（tab/footer、创建与输入回显、
  picker 重绑已有终端、`%`/`x`、单键 picker、Ctrl-Q 退出），总 105/105。
- **协议/文档**：`PROTOCOL.zh-CN.md` 注明 `.proto` 是唯一真相、Go SDK 与 Python 示例
  都是可选绑定；`CUSTOMIZE.zh-CN.md` §6 给出运行命令与扩展方式。

### 2.7 M20 老 UI 像素复刻（legacy.py + parity golden + 缺口修补）

- **静态像素（M1）**：`clients/tui/examples/python-shell/legacy.py`（纯标准库，复用 `pb.py`）
  按 `LEGACY_PARITY.zh-CN.md` 的逐行规格复刻老默认 UI：header
  `WS main ▎1 main ×  +`、30 列 sidebar status 盒（tabs/panes/mode/world 派生行）、
  pane 自绘边框 + `embedTitle` 标题嵌入 + 焦点 `▎ `/`▎` 标记、`│` 分段右对齐
  footer、picker/help/prompt/manager 浮层与 toast。终端 pane 的组件 chrome 用
  `content.props`（`chrome.*`）锁定为老 token（exited→warning、focused→accent、
  否则 muted）。
- **交互（M2）**：tab 点击/切换/新建/关闭、pane 点击聚焦与 Tab 循环、`%`/`"`
  split、`x` 关槽、分隔条拖拽（`input:["mouse"]` 隐式捕获 + 绝对坐标改权重）、
  picker（↑↓/enter/点击/重启 exited）、prompt（`: ` 前缀 + 过滤 + 光标复投）、
  help（`?` 开关）、esc 层级、footer 六态提示、退出角标 + `Ctrl-E`、回看
  `[↑N]`/`y` 复制/`esc` live、toast、本地 pane 滚轮行窗口。
- **框架/协议缺口（M3）**：
  1. 终端组件退出码 0 显示 `[exited 0]`，老组件是 `[exited]` →
     `components/terminal/render.go` 按老规则修（单测 `TestTitleTextBadges`
     增加零码与“零码+回看”组合）；
  2. Python 参考绑定 `pb.py` 不编码 `Box.cursor` → 补 field 7 编码（parity 断言
     prompt 光标坐标、验收断言 tmux 硬件光标 (35,8)/(37,14)）；
  3. 协议无 `world` 事件 → 程序由 `sources` + `HELLO.view_id` 派生同形文案（记录）；
  4. 老 `workbench.command` 不在 §4 → `:kill pane` 映射 `terminal.kill`（记录）；
  5. 老 demo `"` 竖分不生效 → `legacy.py` 按 tab.flow 补竖堆叠（记录）。
  审计确认已具备（非缺口）：程序光标合成、overlay z 序/不透明、鼠标捕获与
  drag、历史取数、显式样式、命中 id 规则。
- **像素验证（M4）**：`clients/tui/examples/python-shell/parity_test.py` 用**独立**的
  老公式渲染器（header/sidebar/paneWidths/window/footerText/center + 边框嵌入 +
  老 terminal 组件 chrome）对 `legacy.py` 的 view（按 tui2 内核语义光栅化）
  逐格比较；14 场景 × 120x32/100x30，字符全比、非空格比样式；通过后刷新
  `golden/*.txt`。
- **验收（M5）**：acceptance.sh 新增 legacy 段 28 项：离线 per-cell parity、冷启动
  picker/单 pane/横向 split 三张全屏 golden 逐行 diff、header/sidebar/footer
  右对齐、picker 文案、prompt 光标 (35,8)/(37,14)、绑定后焦点标记与输入、
  `[exited 7]`、`Ctrl-E` 重启、零码 `[exited]` 且无 `[exited 0]`、`[↑10]` +
  SCROLL footer、`y` 复制 toast、`esc` 回 live、退出。总 **133/133**。
- **文档（M6）**：`LEGACY_PARITY.zh-CN.md`（视觉规格/差距/缺口清单）、
  `CUSTOMIZE.zh-CN.md` §6.1、本页。

### 2.8 M21 老推荐 TUI 配置（coralline-candy → v2）

- **考古（M0）**：老推荐配置的权威来源是 `tui/docs/tui-v3.recommended.yaml`
  （profile `coralline-candy`，安装脚本写到 `~/.config/anytty/tui-v3.yaml`）；
  默认值对照 `tui/config/config.go`、`tui/state/config.go`
  （`DefaultTUIPickerEndpointStatusConfig`）、`tui/render/glyphs.go`
  （`defaultPaneChromeGlyphs`）；endpoint 模型来自 `~/.config/anytty/endpoints.yaml`
  （v3：`label/connect_mode/routes`，route kind `local-unix`/`managed-webrtc`/
  `direct-webrtc-tcp`）。逐项结论与码点表见 `RECOMMENDED_CONFIG.zh-CN.md`。
- **M1 图标与视觉**：`clients/tui/config/icons.go` 定义 `icons`（preset 字符串或
  `{preset,map}` 对象）与 `recommended`/`unicode`/`ascii` 三套表；
  `config.DefaultTheme = "recommended"`、`themeByName` 新增 coralline-candy
  色板（primary `#f0abfc`、bg `#070611`、surfaces `#17132a/#261b44/#3b2f63`…）；
  shell 的 header（`󰙅`、`⎇`、`󰐕`）、槽标题栏（`󰑐   󰅖`）、footer
  （`󰌌 NORMAL`、动作图标）、状态栏（`󰓩 tab 1/1`、` slot 1/1`）、侧栏/浮层标题全部
  走 `m.icon(...)`。单测 `recommended_test.go` 断言 view 树 + 由 runtime 解算并
  合成的**最终帧字节**含码点（`TestRecommendedIconsInViewTree`/
  `TestRecommendedFinalFrameCarriesIcons`）；`icons` preset/map 有配置测试。
- **M2 键位**：白名单新增 `pane.split_h`/`pane.split_v`（默认 `%`/`"`，与老默认
  一致），推荐 panel 键位（`Ctrl-D`/`Ctrl-E` 分屏 + `Ctrl-R` 重启）作为一行
  `keybindings` 覆盖写进映射表；重叠默认（`Ctrl-P/F/T/W`、`x`、`y`、`:`、`?`）
  已对齐，v2 无对应能力的键（resize/floating/copy 模式）逐条记录。
- **M3 endpoint**：`tui2.json` 新增 `endpoints`
  （`name/kind/label/argv/cwd/env`，v1 仅 `command`）；picker 在终端与
  `+ New terminal` 之间列出 endpoint（`󰌷 label`、`endpoint · command <argv0>`），
  选中即 `terminal.create{endpoint,argv,cwd,env}`（协议字段 5/6/7 早已具备，host
  `gateHandler.create` 直通 PTY），绑定 `terminal:<name>:<id>`；单测覆盖参数与绑定，
  tmux 验收用假命令 `sh -lc 'echo REMOTE-READY; exec cat'` 断言画面与输入回显。
  与老 daemon 协议（routes/credentials/连接状态）的差异与后续见文档 §4/§5。
- **M4 legacy.py preset**：图标/颜色抽成模块级 `GLYPH_*`/`STYLE_*`，新增
  `-config`/`--config`/`ANYTTY_LEGACY_CONFIG`（JSON `{"preset":"recommended"}`，
  可 `icons`/`colors` 单点覆盖）；默认不带配置时逐像素不变，`parity_test.py`
  新增 `check_preset`（29/29）。
- **M5 验收**：新增 14 项——推荐 workspace/tab/状态图标与 picker 模式徽标、
  endpoint 列出与图标、选中绑定 `remote:term-1`、`REMOTE-READY`、输入回显、
  模式徽标回到 live、`legacy.py` preset 的 `󰙅`/`󰐕` 与退出，总 **147/147**；
  既有 133 项仅按新图标更新 tab/按钮正则，无删除。
- **M6 文档**：`RECOMMENDED_CONFIG.zh-CN.md`（考古、码点/颜色/键位/endpoint
  映射、差异清单）、`CUSTOMIZE.zh-CN.md` §2.2/§3/§3.1/§6.1、本页。

### 2.9 M22 footer 场景化（键名≠标签；按场景裁剪；推荐规格 golden）

- **根因（一句话）**：v2 footer 把**键名/动作名当标签**渲染（PANE 显示
  `split-h`/`split-v`，等价于 resize 场景会把 `space` 显示成 `spacebar`），
  且没有按场景裁剪——老推荐配置的键位是 scene 分组的，footer 必须只取当前
  场景中 v2 已实现的键位。
- **M1 复现/考古**：用推荐配置跑 Go shell 抓屏，viewport 树与最终帧都定位到
  `split-h`/`split-v` 字样；在 HEAD 上写了临时 oracle 测试（`/tmp` 工作树，
  未入库），用**老渲染器真实代码**（`tui/render` 的
  `footerLeftSegments`/`appendFooterActionSegments`/`composeFooterBarLine` +
  `tui/config.Parse` 读 `tui-v3.recommended.yaml`）产出每个场景的旧 footer
  行，得到规格表：badge 带前后空格、动作 `" · " + " " + label`、label 内嵌
  key/icon、右段 ` 󰙅 ws 󰹙 0 󰒉 N `、`pad = cols - left - right`（最小 1）。
  规格表落进 `RECOMMENDED_CONFIG.zh-CN.md` §3.1。
- **M2 修复**：新增 `clients/tui/cmd/tui2-shell/footer.go` 作为唯一规格表：
  `keyDisplay()`（`SPACE`/`PGUP`/`CTRL-P`/`TAB`/`ESC`，绝不输出协议名或动作名）、
  `footerSceneFor()`（NORMAL/PANE/picker/prompt/help/scroll/exited）、
  `ctrlKey()`（Ctrl 提升进 badge，老推荐约定）；`view.go` 的 footer 改为
  badge + 场景键位 + 右段摘要 + 填充，删掉 `split-h`/`keyLabel`/重复 badge 的
  旧实现。未实现的场景动作（resize/system/floating/zoom/tab n-p、panel 的
  d/r/z/h/l/q 等）不映射也不显示。新增 `focus/page_up/page_down/attach/run/
  summary_floating` 图标名与 `footer-key-*` 主题槽（颜色按语义映射，见文档差异）。
- **M3 像素验证**：`golden/recommended_footer_120x32.txt` 8 行
  （NORMAL/NORMAL_2PANE/PANE/PICKER/PROMPT/HELP/SCROLL/EXITED）；
  `footer_golden_test.go` 从**最终合成帧**取 footer 行逐字符比对并断言无
  `spacebar|split-h|split-v|layout_toggle|resize.` 等字样；legacy.py
  `--footer-lines` 输出同一格式，`parity_test.py` 新增 `check_recommended_footer`
  比对同一 golden（30/30）。
- **M4 验收**：新增 5 项——推荐 picker/NORMAL/PANE footer 逐行、footer 无原始
  键名/动作名、legacy.py preset footer；`x 关闭槽` 与 scroll/live 断言改为新
  文案（`1/1 term-1`、`PGUP 󰁝 OLDER`、`󰌌 CTRL`），总 **152/152**。
- **M5 文档**：`RECOMMENDED_CONFIG.zh-CN.md` §3.1（规格表 + 不支持动作清单 +
  替换/颜色差异）、§1/§2/§5/§6 更新，本页。
- **教训**：**键名 ≠ 标签**（label 优先，fallback 必须是规范化显示名）；**场景化
  键位必须按场景裁剪**——footer 是"当前场景可执行动作"的投影，不是全量绑定表；
  跨实现一致性要靠**同一份规格表 + 同一 golden**（Go 与 legacy.py 各自渲染、
  同一文件比对），否则两边会各自漂移。

### 2.10 M23 daemon 连接策略 endpoint（真实 local-unix 客户端）

- **M1 考古/规格**：`clients/tui/docs/ENDPOINTS.zh-CN.md`——老 `endpoints.yaml`
  v3（label/connect_mode/routes local-unix/direct-webrtc-tcp/managed-webrtc）与
  `scripts/anytty-dev.sh` 写入口径；daemon wire7 调用序列（Hello v7 →
  `api.execute` list → attach/`BootstrapDone`/`StreamReady`/`PTYOutput` →
  input → resize owner CAS/epoch → kill/remove/restart → detach/关闭）；
  v2 字段/picker/协议参数/sources health 映射与 v1 支持矩阵（command ✅、
  local-unix ✅、tcp 下一步、webrtc 不做）。
- **M2 `clients/tui/endpoint`**：`Config`/`Manager`（每 endpoint 一个 supervisor：
  拨号失败/断线 → `offline` + notice，250ms..5s 退避重连，重连后重新 attach
  全部 RemotePTY）；`client`（wire7 control 相关 + channel 解复用；请求带
  `RequestContext` 与 `OperationStamp`；attach 的 `ResourceHandle.opaque_token`
  前两字节解出 stream channel）；`RemotePTY` 满足 `clients/tui/pty.PTY`，把 daemon
  raw stream 喂给同一 ANSI parser/组件；attach/重连先拉
  `LiveScreenNext(observed=0)` 全量快照渲染成 ANSI 注入读流（快照权威、
  不复播），resize 走 owner CAS 并把冲突转成可读错误，Close=kill、TUI 退出=
  detach 不杀 daemon 终端。假 daemon（同协议最小服务端）覆盖 list/attach/
  input/resize/kill/断线重连/不支持 connect_mode/owner 冲突/快照渲染。
- **M2 接线**：`pty.Config` 带 `Endpoint/ID`；host `NewPTY` 按 endpoint kind
  分发，`gateHandler` 处理 `endpoint.sync`、daemon `terminal.create`
  （daemon 分配 id）、`terminal.restart`（daemon restart 原地重启）、
  `terminal.remove`（确认后 daemon remove）；sources 合并 daemon 清单
  （attached=false 可被 picker 绑定），health 随连接状态，notice 经宿主
  `Session.SendNotice` 转发。
- **M2 协议**：`MethodParams` append-only 字段 `kind/socket/connect_mode`
  （17..19）；方法表追加 `endpoint.sync`；`tui2.pb.go` 由
  `clients/tui/protobuf/regen`（无 protoc：从已编译 descriptor 追加字段后跑
  protoc-gen-go 插件）重生成，proto/生成代码/`pb.py` 同步。
- **M2 shell**：`config.Endpoint` 支持 daemon（socket/address/connect_mode，
  校验 kind/argv/socket）；HELLO 后对 daemon endpoint 发 `endpoint.sync`；
  create/attach 带 kind/socket/connect_mode；配置了 daemon endpoint 时 picker
  按 endpoint 分组（组头不可选，键盘/鼠标语义不变；否则保持 v1 平铺像素）。
- **M3 验收（真实 daemon）**：`scripts/anytty-dev.sh` 隔离 dev daemon +
  `cli v3 new`，TUI 里 picker 分组/endpoint 标签 → attach（快照回显）→
  `echo EP-DAEMON-OK` → `tmux resize-window` 后 `stty size` 跟随（32 118 ⇄
  26 98）→ `:kill` 确认后 `[exited N]` 且 `v3 ls` 为 exited → Ctrl-E daemon
  restart → 本地 command endpoint 与 daemon endpoint 同屏互不干扰；
  offline socket 与 `connect_mode: tcp` 各自发可读 notice 且不阻塞退出。
  新增 28 项，总 **180/180**。
- **M4 legacy.py**：`endpoints` 配置（command/daemon）与 picker 分组；
  默认无 endpoint 时逐像素不变（parity 场景仍 30/30），新增
  `check_endpoints` 分组兼容检查，parity **31/31**。
- **M5 文档**：`ENDPOINTS.zh-CN.md`（新增）、`CUSTOMIZE.zh-CN.md`
  §3.1/§3.2、`RECOMMENDED_CONFIG.zh-CN.md` §4、`PROTOCOL.zh-CN.md`
  方法表/字段、本页。

### 2.11 M24 老 v3 TUI 像素复刻（`v3ui.py` + `v3_parity_test.py`）

- **M1 考古/规格**：`V3_PARITY.zh-CN.md`——从 git HEAD `tui/render`
  （`shell_bar.go`/`panel_chrome.go`/`panel_terminal_chrome.go`/`pane_chrome_actions.go`/
  `header_tab_template.go`/`box_glyph.go`）逐条抄出 header 模板、pane card
  `paneChromeTopSlots` 几何、`● x1 owner` 槽位、动作组 `\ue0b6 … \ue0b4`、footer
  选择/裁剪算法与 recommended yaml 的 glyph/颜色/键位；`1.txt` 实测 181×56
  （任务指定 golden 120×32）。
- **M2 `v3ui.py`**（纯 stdlib + `pb.py`）：`pos` 绝对布局的整套 chrome；
  recommended 模板（workspace/tab/create/顶条）逐字符；card 窗口 + 内容区多 pane；
  pane 提示行/折叠；footer 场景表（live/pane/resize/tab/workspace/system/floating/
  picker/prompt/copy）；floating（新建/居中/折叠/拖拽）；picker/prompt/help/clipboard；
  `Ctrl-Q` 走宿主确认；真实终端内容走 v2 terminal 组件（`chrome.inset=0`）。
- **M3 Go 缺口**：(A) `terminal.Props` 增加 `chrome.inset`（`PropInset`+`InsetSet`，
  host placement 解析下发）——card 内组件不再画框，PTY = 完整内容矩形；
  (B) `keys.Name` 保留 `ctrl-shift-<letter>` 与 `ctrl-alt-<digit>`（CSI-u
  enhanced keyboard）——`⇧C/⇧H/⇧V` 与 `ctrl-alt-1..5` 可 claim；
  (C) `runtime/compositor`：落在 `pos` 子树里的组件 placement 改为在该子树内
  合成（父 lines → 本节点组件 → 嵌套 overlay），不再被 overlay 的不透明清屏擦掉。
  三处均有单测（components/terminal、runtime/keys、runtime）。
- **M4 像素验证**：`v3_parity_test.py` 用独立公式实现对比 `v3ui` 的 selftest 光栅
  （120×32/181×56 chrome 全行、card 边框逐行、footer 逐字符），并用 `1.txt`
  本体作 3-tab oracle（header/frame/footer 行逐字符）；写出
  `golden/v3_ui_120x32.txt`、`v3_ui_181x56.txt`、`v3_ui_demo_tab2_*.txt` 与
  `v3_footer_120x32.txt`（8 场景）。
- **M5 验收**：acceptance 新增 15 项（golden 逐行 3 项、parity 1 项、pane 聚焦 2 项、
  split+折叠提示 3 项、PANE footer 1 项、Esc live 1 项、`⇧V` CSI-u 1 项、
  顶条新建 tab 1 项、Ctrl-Q 退出 2 项），总 **195/195**。
- **M6 文档**：`V3_PARITY.zh-CN.md`（新增）、`CUSTOMIZE.zh-CN.md` §6.2、本页。

### 2.12 M25 v3ui 分屏/浮窗真实可用（`v3ui.py` + compositor 缺口 D）

- **复现**：tmux 181×56 真宿主下，`Ctrl-P Ctrl-E`/`%` 后新 pane 无终端、布局只有
  一份 card chrome；浮窗能画框但 `terminal.create` 绑不到浮窗 pane（`pane_by_id`
  只搜 tab），且 `Ctrl-O n` 后无法输入；`z` 折叠是整体消失；拖拽只有标题文字可抓。
- **分屏**：`tab.weights` + `distribute_extents` 成为唯一布局真相；
  `pane_rects` 输出 pane 矩形 + 1 cell `Divider`（row `│`/col `─`，
  `divider:{tab}:{index}` + `input:["mouse"]`）；`drag_resize` 用 drag 坐标换算
  相邻两 pane 尺寸并写回 weights，`Ctrl-R` 的 `h/l/k/j` 走同一套 resize；
  `split_pane` 把源权重对半、`mode="pane"`（保持老验收），宿主下自动
  `terminal.create` 并绑定；`sub_pane_nodes` 只在 live 且无 active 浮窗时
  `focused=true`，保证输入只进聚焦 pane。
- **浮窗**：`pane_by_id` 纳入 floating pane；`new_floating` 用老 v3 几何 + 级联并
  自动建终端；`floating_runs` 的整行框线都是 title 命中区（可拖）；
  `toggle_floating_collapse` 折叠后只画标题行（`visible` 不再过滤）；
  `raise_floating` 把窗口移到列表尾实现 z 序；`bind_pane` 命中浮窗时自动
  展开/提升/聚焦；`Esc` 回 live 后浮窗 terminal 获得焦点，点主 pane 清除
  active 浮窗；`clamp_floatings` 在外部 resize 时收敛几何。
- **框架缺口 D**：`runtime/compositor` 的 placement→overlay 分配从“首个包含者”
  改为“最后声明（最上/最深）包含者”（`assignOverlayPlacements`，顶层与嵌套
  统一），修复浮窗终端落在 card 大 overlay 内、被浮窗自身 fill 擦空的问题；
  单测 `TestCompositorPlacementFollowsTopmostOverlappingOverlay`。
- **验证**：`v3_parity_test.py` 新增 `split_col`/`split_row`/`float_chrome`
  （独立公式逐字符 + 4 个新 golden，共 10 checks）；acceptance 新增 31 项：
  单 pane PTY `36x118`、上下分屏 `17/18x118`、分隔条拖拽后 `27/8`、
  互不串扰、左右分屏 `36x58/59`、行分隔拖拽 `78/39`、浮窗自动绑定 `28x94`、
  浮窗/主 pane 焦点切换、标题拖拽、折叠只留标题/展开恢复、外部 resize 后
  `26x94` 与 `26x64/33`，总 **226/226**。
- **文档**：`V3_PARITY.zh-CN.md` §1.5/§1.6/§4-D/§5/§6、`CUSTOMIZE.zh-CN.md` §6.2、本页。

### 2.13 M26 v3 复刻分屏树 + footer 每键颜色

- **复现**：120×40 tmux 下 `Ctrl-P %` 后再对右侧 `"` 分屏，三个 pane 变**全局上下
  重排**（flat `flow` + weights，一个 card 内切）；`capture-pane -e` 抓 footer 只有
  一个 muted `38;2;156;163;201`，所有键组同色。
- **分屏树**：`v3ui.py` 引入 `Leaf(pane)`/`Split(orient, ratio, a, b)` + tab.root；
  `split_pane` 只把聚焦（或指定）`Leaf` 替换成 `Split(原, 新)`，`Tab.rebuild()`
  维护中序 `panes`/`focus`；`layout_node` 递归算 rect：`a` 占
  `round(avail*ratio)`（avail = 节点尺寸 − 1 分隔 − 4 边框），row→左右、col→上下；
  每 leaf 走 `card_nodes` 画完整 card，分隔条 id `divider:{tab}:{split_seq}` 只写回
  本 Split ratio；`close_pane` 递归删除并提升兄弟；`resize_focused` 找最近同轴祖先；
  `reset_tab_splits` 把所有 ratio 归 0.5；折叠/全部折叠从布局移除。
- **footer 颜色**：显式 `footer-key-*` token 常量（`mixHostColor` 同式取整）+ yaml
  `shortcuts.actions` 覆盖表 + `footer_fallback_style()` 启发式，精确复刻
  `shortcutActionStyle`→`footerActionDisplayStyle`→`footerActionKeyStyle` 链；
  badge 用 yaml `footer.modes` per-scene style（pane/resize/picker/copy 各有色）。
- **验证**：`v3_parity_test.py` 新增 `left1right2`（120/181 两视口，leaf rect +
  三 card 题字/边框逐字符 + golden）与 `footer_colors`（LIVE/PANE/PICKER 逐非空格
  cell 比对 + 异色计数 7/5/4 + `v3_footer_colors_*.txt` golden），共 **12/12**；
  acceptance 新增 23 项：左 1 右 2 三 leaf `36x58 / 17x57 / 16x57`、拖右侧分隔
  `23/10` 且左侧不动、footer NORMAL ≥7 色/PANE ≥5 色/PICK 徽标色、外部 resize
  `26x48 / 16x47 / 7x47`、复原 `10x57`、关闭右下提升为单 leaf `36x57`，总
  **249/249**。旧的 v3pty 断言语义不变、数值按 card-per-leaf 几何更新
  （如 `17/18`→`17/16`、`78/39`→`77/38`、`26x33`→`26x31`）。
- **文档**：`V3_PARITY.zh-CN.md` §1.4 色表/§1.6 树规格/§3/§5/§6、本页。

### 2.14 M27 连接远程 terminal（tcp / ssh 隧道）

- **M1 考古**：`REMOTE.zh-CN.md`——老 `endpoints.yaml` 的 route kind
  （`client/endpoint/registry.go`、`client/adapter/{local,direct,ssh,cloud}/dial.go`、
  `client/runtime/endpoint_supervisor.go`）与远端监听事实：core-v2 wire 只监听
  unix socket（`core/server.go` + `shared/runtimepath`），`daemon --route` 是
  WebRTC signaling/ICE（`cmd/anytty/v3_direct_daemon.go`），没有 wire TCP 端口；
  鉴权在 direct/cloud 走 remoteauth/DTLS+Cloud，sshd 隧道则交给 ssh。
- **M2 Dialer + tcp**：`endpoint.Options.Dial` 已是注入点，新增默认
  `dialTransport` 按 `connect_mode` 选 `local-unix`/`tcp`；
  `clients/tui/endpoint/transport_tcp.go` 实现与 `shared/transport/unix` 字节兼容的
  zstd 分片帧（相同 packet kind/64KiB 分片/zstd window），`MethodParams` 新增
  append-only `address`(20)，host/shell/`pb.py`/`legacy.py` 全链路打通；tcp 缺
  address 在配置层报可读错误，拨号错误带目标（`dial HOST:PORT: ...`）。
- **M3 P0 隧道**：`ssh -N -L local.sock:remote.sock`（local-unix，零代码）与
  `ssh -N -L 127.0.0.1:PORT:remote.sock`（tcp）都在验收里用真实
  `ssh localhost` 验证；无 ssh 环境回退 `clients/tui/scripts/remote-bridge`（Go 写的
  字节透明 TCP→unix 代理，验收里注明 tunnel kind）。老命令式路径
  `command` endpoint + `ssh host ...` 仍有独立验收。
- **M4 健壮性/UI**：断线 → health=offline + notice、250ms..5s 退避重连、重 attach +
  快照重建、resize owner CAS 冲突可读；picker 信息列从 18 格放宽到 21 格以容纳
  `endpoint · daemon tcp`（唯一程序侧小补，其余 UI 零改动）。
- **M5 验收**：启第二隔离 daemon（独立 socket/XDG/direct 端口 `:0`）→ 隧道暴露 →
  TUI picker 分组 → attach → `echo REMOTE-OK` → resize `32 118 ⇄ 26 98` →
  kill/Ctrl-E → 隧道断开 offline/重开 connected → ssh 命令式端点；
  acceptance 新增 34 项，总 **283/283**。
- **M6 单测**：tcp 帧与 shared/unix 双向兼容（shared listener + TCP 桥）、tcp
  endpoint 全生命周期（attach/input/resize/断线重连/快照重播/kill）、拨号拒绝与
  配置错误可读；`go test -count=1 -race ./clients/tui/...`。
- **M7 文档**：`REMOTE.zh-CN.md`（新增）、`ENDPOINTS.zh-CN.md`
  §2.1/§2.3/§3/§4、`CUSTOMIZE.zh-CN.md` §3.2、本页。

### 2.15 连接层去重（CLIENT_SHARING，M1+M2 最小步）

- **M1 审计/设计**：`clients/tui/docs/CLIENT_SHARING.zh-CN.md`——共享层 API
  （`client/endpoint` registry、`client/runtime` session/supervisor、
  `client/adapter/*` dialer、`client/port`）、tui2 保留/删除清单、适配器接口
  设计与 4 项 client 缺口（G1 `ReadyPeerSession`→裸帧/`sessionConn` 桥、
  G2 registry 无 `tcp` kind、G3 composition 在 `cmd/anytty`、
  G4 supervisor attach 钩子）。本轮不改 `client/`。
- **M2 去重**：删除 `clients/tui/endpoint/transport_direct.go`（620）、
  `transport_cloud.go`（244）、`transport_direct_pion.go`（67），
  `direct-webrtc-tcp`/`managed-webrtc` 不再有 tui2 本地实现，统一报可读错误
  指向共享层；修复 `Config` 不可比较导致的构建失败。生产代码
  **3642 → 2835 行（-807）**；包总行数 4952 → 4409（含 +268 新测试）。
- **M3 配置**：新增 `clients/tui/endpoint/shared_registry.go`，daemon endpoint 注册时
  读 `client/endpoint.DefaultPath()`（`endpoints.yaml`），同名条目以共享层为准
  覆盖 tui2.json 兼容字段；只读、不写、不配对；文件缺失不注入默认 local。
- **M4 防复发**：`client_sharing_test.go` 用
  `go list -deps ./clients/tui/endpoint ./clients/tui/cmd/tui2` 断言依赖闭包不含
  pion/ssh/directsignal/remote-webrtc/remote-client/cloud-client/datachannel。
- **M5 验收**：新增 7 项 Go 验收（共享路径、共享 registry 读取、缺失文件不注入
  默认、可表示 route 列表、direct/cloud 可读错误、依赖守卫、行数证据），
  `acceptance.sh 283/283` 保持；完整 G1 桥接与共享 direct/cloud 实拨列为剩余步骤。

### 2.16 M28 连接层去重收官（CLI registry + 共享 client 拨号）

- **M1 桥接**：`clients/tui/endpoint/sessionConn` 是唯一 daemon 拨号 seam；
  `shared_session.go` 的 `sharedClient` 用 `client/runtime`（SessionOwner
  generation/planner race）+ `client/adapter/protocol`（ApplicationClient
  command + `ResourceStream`）实现 list/create/attach/input/resize/kill/
  remove/restart/liveScreen/detach；`RemotePTY`/组件零改动。断线的
  `io.EOF`/cancel 一律转成 transient 错误（不再伪装成终端退出），health=ok
  在 `rebindAll` 完成后发布，保证重连后第一键可用。
- **删除重复实现**：生产 `client.go`（761）/`transport_tcp.go`（235）删除，
  等价代码只留在测试（`wire_client_test.go`/`framed_transport_test.go`）；
  `tcp` 由 `tcp_bridge.go` 字节透明 relay 到本地 unix socket，再走共享
  local-unix route adapter。生产 3642 → 2789 行（-853）。
- **M2 registry 驱动**：`Options.LoadSharedRegistry`（main.go 开启）在 host
  启动时注册 `endpoints.yaml` 的全部可表示端点；`Manager.Sources()` 为无终端
  daemon 端点发 `kind=endpoint` 占位源（label + health）；shell picker 显示
  角标、选择即 `terminal.create{endpoint}`；同名 tui2.json 字段仅作迁移。
- **M3 端到端**：acceptance 新增 CLI `endpoint add local` → registry 文件 →
  TUI 冷启动列出分组/终端 → attach → `REG-OK` → `exit` 角标 → Ctrl-E 重启 →
  离线端点 `endpoint · offline` + 可读 notice → 干净退出；smoke/acceptance
  全量隔离 XDG 并动态分配 signaling 端口，避免读取开发机 registry。
- **M5 守卫**：`TestTui2HasNoRawProtocolDialer`（`go list` 直接 import：必须
  有 `client/adapter/protocol|local`、不得有 `internal/protocol`/
  `proto/wirepb`/`shared/transport/unix`/zstd/ssh）、
  `TestTui2DependencyClosureUsesSharedAdapters`（`go list -deps`）、
  `TestTui2EndpointDeduplicationEvidence`（行数与文件清单）、
  `TestSharedStackLocalUnixLifecycle`（真实共享栈 + 带 DeviceIdentity proof
  的 framed fake daemon：断线重连≠终端退出/快照重播/resize/kill）。
- **M4 文档**：`CLIENT_SHARING.zh-CN.md` 改写为最终态；`REMOTE.zh-CN.md`
  新增 §4.0 CLI 配对 runbook 并更新矩阵/缺口；本页。
- **剩余**：cloud connector 已装配（`client/adapter/cloud` + controller 默认
  值同 CLI），但 composition/环境变量解析与 `cmd/anytty` 各一份；CLI registry
  仍无 `tcp` route kind；`cmd/anytty` 云端 enrollment 管理未变。

### 2.17 M30 日志接管 + 路由裁剪（用户反馈：进 TUI 被日志刷屏）

- **根因确认**：共享 `client/` 层（`client/runtime`、`client/adapter/direct|
  ssh|cloud`、`shared/netpath`、`shared/connecttrace`）用 Go 标准 `log`
  默认写 stderr；TUI 进入 alt screen 后 stderr 直通终端，`anytty connect
  trace_id=…`、`anytty webrtc selected_pair …` 等把 picker 画面冲掉。老
  registry 的 endpoint 带 direct/cloud route 时，tui2 的 route planner 还会
  真去拨号（signaling/ICE/Cloud），失败重试刷屏且卡数秒。
- **M1 日志接管**：宿主 `main()` 在 `resolveShell` 之前
  `setupLogging`：`-log-file` > `$TUI2_LOG_FILE` >
  `$XDG_STATE_HOME/anytty/tui2.log`（0600、追加、自动建目录），
  `log.SetOutput` 一次覆盖共享层；host 自身异常（布局程序崩溃/输入路由
  错误/endpoint notice）改走 `Options.Logf`（默认同标准 logger）+ 程序
  notice，`writeOut` 只保留帧/alt screen/clipboard。日志首行
  `tui2 start version=… routes=…` 便于确认策略。
- **M2 路由裁剪**：`policy.go` 定义可拨 route 集合（默认
  `local-unix, ssh-webrtc-tcp`），`sharedRouteEnvironment` 只把策略内的 kind
  交给共享 planner；ssh 需 endpoint 凭据在共享 store 可解析才进入 planner。
  direct/managed 默认不进入 planner → 不发 signaling/Cloud、不占 4 秒，
  endpoint 保持 `endpoint · offline` + 一行可读 notice；`TUI2_ROUTES`/
  `-routes`（`all` 或别名 local/tcp/ssh/direct/cloud）显式开启。
  `configFromSharedEndpoint` 投影顺序改为 local > ssh > direct > managed，
  老 registry 多 route 端点自动降级到可拨 route；纯 webrtc/cloud 端点仍列出。
- **M3 验收**：acceptance 新增 22 项（335/335）：真实 CLI 写出的
  local+webrtc/webrtc-only/cloud-only 老 registry；默认策略下混端点在 picker
  以 live 列出并 attach/输入，两个纯 webrtc/cloud 端点 offline + notice；
  抓屏断言无 `trace_id=`/`anytty connect|cloud|network`/`webrtc selected`；
  默认日志文件含 `tui2 start` 与 offline notice；`TUI2_ROUTES` opt-in 后
  共享层 `anytty connect` 行只出现在日志文件；启动失败（坏 -shell）与退出
  后终端都不出现日志行。
- **M4 文档**：`REMOTE.zh-CN.md` §3.1（日志路径/如何 tail/route 支持矩阵/
  开关），本页。

### 2.18 M31 SDK 三层 + 一致性套件（"像写 web 一样写界面"的地基）

- **目标**：把"布局程序 SDK"做成可验证的公共产品——core（帧/事件/Emit/Commit）
  / builder（盒子链式声明）/ widgets（程序侧 chrome）三层，任何语言（含 AI
  生成的）用同一套 fixtures 自证合规。
- **M1 fixtures + verify（地基）**：新增 `clients/tui/conformance/`：
  `fixtures.jsonl` **10 个会话用例**（hello 解析与 view/keys 提交、9 类事件
  分发、result→response 关联、request_id 单调、epoch 重置 + 过期 RESPONSE
  不触发回调、view_rejected、raw 二进制 HELLO 帧）；`frames.jsonl` + `frames/*.bin`
  3 个二进制帧样例（hex + 解码 JSON + 实体文件，可跨语言逐字节对照）；
  语言中立 runner（宿主帧用
  protobuf 编码、程序帧语义比对、`$rN` 绑定 request_id）；被测程序契约见
  `conformance/README.zh-CN.md` §2。新增 CLI
  `tui2 cmd/tui2-sdk-verify`：`--cmd "python3 …"` 逐项判定、非零退出码，
  官方 Go 与用户 SDK 同一标准；Go 测试 `TestOfficialGoSDKPasses`、
  `TestPythonSDKConformance`、`TestTSSDKConformance`、帧样例测试全过。
- **M2 Go 分层**：`clients/tui/sdk/core`（协议客户端）+ `clients/tui/sdk/builder`
  （盒子/度量）+ `clients/tui/sdk/widgets`（chrome）；`clients/tui/sdk` 保留同名再导出，
  `cmd/tui2-shell` 行为不变。补齐 widgets：`TitleBar`、`Footer`、
  `Picker`、`SplitLayout`(+`Distribute`)、`FloatingLayer`、`Toast`；
  删除 shell 本地 `distribute`（改用 `widgets.Distribute`）。RESPONSE 关联
  修正为按 `(epoch, request_id)`：过期 epoch 的响应不再误触新回调。
- **M3 Python SDK + v3ui 重写（关键验证）**：`clients/tui/sdk/python/tui2sdk/`
  （wire/core/builder/widgets/conformance）；`examples/python-shell/v3ui.py`
  **2541 → 467 行**（≤500，不含 SDK），v3 像素与交互（分屏树/浮窗/拖拽/
  footer 每键颜色/回看/快捷键）不减；`pb.py` 变 SDK shim，`shell.py`/
  `legacy.py` 零改动。`v3_parity_test.py 12/12`、`parity_test.py 31/31`。
- **M4 TS SDK（尽量）**：`clients/tui/sdk/ts/`（wire/core/builder + `index.d.ts` +
  `conformance.js` + `verify.js`），Node 直接跑帧协议，无构建步骤；
  `sdk-verify --cmd "node …"` 10/10。
- **M5 文档**：`docs/SDK.zh-CN.md`（三层 API 清单 Go/Python/TS、AI 生成新
  语言 SDK 的步骤、内容源/组件作者接法）、`conformance/README.zh-CN.md`；
  本页。三语言一致性：Go/Python/TS 各 **10/10**。
- **门槛**：`gofmt` 空、`go build ./...`、`go vet ./clients/tui/...`、
  `go test -count=1 -race ./clients/tui/...` 全绿；smoke 10/10；acceptance 335/335
  （v3/legacy 像素与交互用例全过）；`parity_test.py 31/31`、
  `v3_parity_test.py 12/12`。

### 2.19 M32 TUI 开发体验（acceptance 环境根因 + dev 循环 + 三语言起点模板）

- **M1 acceptance 两个失败 = 环境而非回归**。失败项固定为
  `daemon: cli v3 ls reports the killed terminal as exited` 与
  `remote: killed remote terminal shows the exit badge`；对照实验：同一棵树
  前台跑 `335 passed, 0 failed`，`nohup` 下跑 `333 passed, 2 failed`
  （TUI 侧因本地 `Close()` 已把 slot 标成 `[exited 0]`，daemon `v3 ls` 仍是
  `running`）。根因：daemon 的 `Kill()` 给终端进程组发 **SIGHUP**；`nohup`/
  忽略 SIGHUP 的父进程把 `SIG_IGN` 继承进隔离 daemon 的交互 `sh`，shell 按
  语义存活，于是 kill 用例必然误报（`daemon/core/process_platform_unix.go`）。
  修复（测试环境假设，不改 daemon 语义、不减用例、不放宽断言）：
  `acceptance.sh` 在最早处检测 GNU `env --default-signal=HUP` 可用后
  `exec` 自愈重执行（`TUI2_ACCEPTANCE_HUP_RESET` 防重入；macOS 无该选项时
  保留原前台运行 caveat）。修复前后：nohup 333/2 → 374/374 ×2。
- **M2 `-dev` 开发循环（`cmd/tui2/dev.go`）**：
  - `-protocol-log <file>`：`tapWriter`/`tapReader` 在**不改变字节流**的前提
    下截帧，`wire.UnmarshalPayload` + `protojson` 解码为 JSON Lines
    （`ts`/`dir=host->program|program->host`/`type`/`payload`）；
    `-dev` 未给路径时默认 `$XDG_STATE_HOME/anytty/tui2-dev.log`；终端零日志。
  - `-watch <path>[,...]`（可重复/逗号分隔，文件或目录，200ms 轮询，
    mtime+size+小文件 FNV 内容哈希防止同尺寸同时间戳漏检）与
    `-reload-on-save`（从 `-shell` argv 推导）；文件变化 → `reloaded <path>`
    notice + 复用 `RestartPolicy` 重启，新 program 提交前保留最后一棵好树。
  - 报错可见：`cmd.Stderr` 环形缓冲（默认 64 行，notice 取末 3 行）；
    崩溃 diagnostic 在 dev 下补 `restarting in <delay>` 与 `stderr: ...`；
    宿主看到编码出的 `view_rejected` EVENT 与协议解码错误时发 warning
    notice（同错误去重）。
  - 非 dev 路径零行为变化（`Options.Dev == nil` 时不包 tap、不建 watcher、
    诊断文案原样）；`gofmt`/`go vet`/`-race` 单测（`dev_test.go`）。
  - 修一处热重载并发缺陷：主动重启后旧 session 的 `Serve()` 错误会再次触发
    重启并显示假崩溃；`sessionDone` 改为带 session 世代的 `sessionEnd`，
    非当前 session 的错误直接忽略，`reloadProgram` 后重新 `serve`。
  - 第二处：主动停止程序时旧 stdout 管道关闭会返回 `fs.ErrClosed`
    （"file already closed"），不是 EOF——若不归一化会被新加的协议错误
    notice 误报并覆盖 `reloaded` 提示；`tapReader` 把
    `fs.ErrClosed`/`io.ErrClosedPipe` 归一为干净 EOF（单测
    `TestTapReaderClosedPipeEndsCleanly`）。
- **M3 三语言起点模板 + `dev.sh` + 教程**：
  - `clients/tui/templates/{go,python,ts}/`：各 100–200 行，行为一致——tab 条
    （Ctrl-T/点击）、两个终端槽（focus/click）、footer 键位提示、picker
    （列出 sources，Enter attach 或 `+ New terminal` create）、`Esc`/`Ctrl-Q`
    退出；分别使用 `clients/tui/sdk`、`tui2sdk`、`clients/tui/sdk/ts`，
    代码内标"改哪里"。
  - `clients/tui/scripts/dev.sh [--log f] [go|python|ts|<file|dir>] [-- ...]`：
    构建宿主 + `-dev` + 自动 `-watch` 目标（`.go` 先 build 成临时二进制）。
  - `clients/tui/docs/TUTORIAL.zh-CN.md`：半小时教程（环境检查 → dev.sh 跑
    模板 → dev flag 表）→ 改标题/加点击按钮/加面板 → 三套自测命令与预期
    输出 → FAQ；SDK 文档新增 §6。
- **验收**：acceptance 新增 39 项（模板 Go build + 三语言各 7 项关键
  chrome/建终端/退出 + dev 11 项 + 崩溃 6 项），总 **374/374**（nohup 与前台
  各跑一遍）；`go test -count=1 -race ./clients/tui/...` 全绿；smoke 10/10；
  SDK verify 三语言 10/10；parity 31/31、v3 parity 12/12。

### 2.20 M33 测试驱动去 tmux 强依赖

- **驱动层 `scripts/libdriver.sh`**：统一原语 `driver_spawn(name,cols,rows,cmd)`、
  `driver_send`（tmux 键名：Enter/Escape/Tab/方向/PageUp/`C-x`）、
  `driver_send_literal`、`driver_send_bytes`（逐字节 hex）、
  `driver_capture`/`driver_capture_raw`（纯文本 / SGR 原始行）、
  `driver_resize`、`driver_cursor`（0 基 `x,y`）、`driver_osc52`（解码文本）、
  `driver_kill`/`driver_kill_all`/`driver_shutdown`、
  `driver_supports <capability>`。另有 `tm` 兼容适配器，acceptance 既有的
  `tm new-session/send-keys -l|-H/capture-pane -pt|-pet/resize-window/
  display-message/show-buffer/set-option/kill-session` 调用原样工作。
- **后端**：`tmux`（首选；`auto` 时 `command -v tmux` 命中即用，隔离
  `-L $SOCK -f /dev/null`）与 `pty`（新 `clients/tui/cmd/tui2-harness`）。
  `TUI2_TEST_DRIVER=tmux|pty` 强制，`TUI2_HARNESS_BIN` 指定预编译 harness；
  pty 下 `set-option`（set-clipboard/history-limit）为隐式 no-op。
- **pty harness**（`clients/tui/testing/harness` + `cmd/tui2-harness`）：
  `pty.New` 固定 cols/rows 起 `bash -c`，`render/ansi` Parser 按宽度解析成
  Screen；`capture` 纯文本行（右裁剪）、`capture_raw` 逐属性 SGR（粗体与
  真彩分开输出，便于 `focusseq` 类断言）、`cursor`、`resize`（先扩网格再
  `TIOCSWINSZ`，内核补 SIGWINCH）、OSC52（`ESC]52;sel;b64`，BEL/ST、跨读
  分片、取最后一次）、`alive`/`EXITED <code>` 回读退出码，kill 用
  `clients/tui/pty` 的会话 leader PID 对进程组 `SIGKILL`（新增 `PID()`）。
  serve 协议逐行请求/响应，诊断只走 stderr，不污染抓屏。
- **接线与证据**：smoke/acceptance 探测驱动后跑同一批断言；新增
  `scripts/driver-parity.sh`（一次跑完 32/32）：tmux/pty 冒烟各 10/10、
  无 tmux PATH 屏蔽下自动回退 pty 冒烟 10/10、关键类别（picker/attach/
  输入/split/回看复制/OSC52/resize/退出）tmux 与 pty 各 13/13、冷启动与
  绑定态抓屏规范化后逐行 diff 一致。pty 下唯一 SKIP：
  `policy: pruned endpoint has a one-line notice`（单一瞬时 notice 槽会被
  后续端点 notice 覆盖，抓屏时机竞争；同内容的持久日志断言仍执行），故
  pty 全量 acceptance 为 373 PASS + 1 SKIP、0 FAIL，tmux 保持 374/374。
  终端日志策略不变（两种驱动抓屏均无日志行）。
- **门槛**：`go build ./...`、`go vet ./...`、`go test -count=1 -race
  ./clients/tui/...`（15 包，新增 harness 单测）、`smoke.sh` 10/10（两驱动）、
  `acceptance.sh` tmux 374/374 / pty 373 PASS + 1 SKIP、`sdk-verify` 三语言
  各 10/10、parity 31/31、v3 parity 12/12。

## 3. 本轮新增：如何运行（精确命令）

```sh
# 全量测试（门槛）
go build ./clients/tui/...
go test -count=1 -race ./clients/tui/...

# 构建两个可执行（单机本地）
go build -o /tmp/anytty-tui2-shell ./clients/tui/cmd/tui2-shell
go build -o /tmp/anytty-tui2       ./clients/tui/cmd/tui2

# 跑起来（需要真实 TTY；Ctrl-Q 两次确认退出）
/tmp/anytty-tui2 -shell /tmp/anytty-tui2-shell

# 版本 / 帮助（无需 TTY；宿主 flags：-shell/-endpoints/-log-file/-routes/-version）
/tmp/anytty-tui2 -version
/tmp/anytty-tui2 -help

# 日志（M30）：默认 $XDG_STATE_HOME/anytty/tui2.log，或 -log-file/TUI2_LOG_FILE
tail -f "${XDG_STATE_HOME:-$HOME/.local/state}/anytty/tui2.log"
# 路由开关（M30）：默认 local-unix + 凭据可用的 ssh；direct/cloud 需显式开启
TUI2_ROUTES=local-unix,direct-webrtc-tcp /tmp/anytty-tui2 -shell /tmp/anytty-tui2-shell

# 配置示例 / 自定义路径（只有 shell 读配置；宿主不读配置文件）
/tmp/anytty-tui2-shell --print-default-config
/tmp/anytty-tui2-shell --config ~/anytty/tui2.json

# SDK 一致性（M31；任何语言 SDK 同一标准；官方三语言）
go run ./clients/tui/cmd/tui2-sdk-verify
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "python3 clients/tui/sdk/python/conformance.py"
go run ./clients/tui/cmd/tui2-sdk-verify --cmd "node clients/tui/sdk/ts/conformance.js"

# tmux 黑盒冒烟（可 CI 重跑；无 tmux/go 时 SKIP，退出码 0）
bash clients/tui/scripts/smoke.sh

# tmux 黑盒全量验收（SCENARIOS §13 单机可验证项 + 推荐配置 + 两个像素复刻，
# 374 项断言：Go shell + Python shell + legacy 复刻 + 推荐配置 + v3 复刻
#   + v3 切分树（左1右2 三 leaf PTY/拖拽隔离/关闭提升/footer 每键颜色）
#   + remote（第二隔离 daemon + ssh tcp/unix 隧道 + Go 桥回退）
#   + log/route（老 registry 含 webrtc/cloud：抓屏无日志、日志文件诊断、
#     offline notice、降级 attach、TUI2_ROUTES opt-in、启动失败无污染）
#   + templates/dev（三语言模板 SDK 加载/关键 chrome/建终端/退出、dev 帧日志
#     双向解码、终端无日志、保存热重载、崩溃原因+倒计时+stderr notice））
bash clients/tui/scripts/acceptance.sh

# dev 开发循环（M32，细节见 TUTORIAL.zh-CN.md）：
bash clients/tui/scripts/dev.sh                                    # 默认 tui2-shell
bash clients/tui/scripts/dev.sh clients/tui/templates/go
bash clients/tui/scripts/dev.sh clients/tui/templates/python
bash clients/tui/scripts/dev.sh clients/tui/templates/ts -- -routes local-unix
# 手动等价：宿主 flags -dev / -protocol-log <file> / -watch <path>... / -reload-on-save
# 帧日志默认：tail -f "${XDG_STATE_HOME:-$HOME/.local/state}/anytty/tui2-dev.log"

# 远程 daemon（详见 docs/REMOTE.zh-CN.md）：
# P0-A  ssh -N -L /tmp/anytty-remote.sock:/run/user/1000/anytty-v2-wire7.sock user@server
#      配置 {"name":"server","kind":"daemon","socket":"/tmp/anytty-remote.sock"}
# P0-B  ssh -N -L 127.0.0.1:17777:/run/user/1000/anytty-v2-wire7.sock user@server
#      配置 {"name":"server","kind":"daemon","connect_mode":"tcp","address":"127.0.0.1:17777"}
# 无 ssh 时用仓库桥：go run ./clients/tui/scripts/remote-bridge -listen 127.0.0.1:17777 -unix <sock>
# P0-C  配置 {"name":"attach","kind":"command","argv":["ssh","user@server","anytty","attach","<id>"]}

# 推荐配置：图标 preset/map 与命令式 endpoint（tui2.json）
#   {"theme": "recommended", "icons": {"preset": "recommended", "map": {"tab_new": "N"}},
#    "endpoints": [{"name": "remote", "argv": ["ssh", "host", "anytty", "attach"]}]}
# 推荐键位覆盖：{"keybindings": {"pane.split_h": "ctrl-d", "pane.split_v": "ctrl-e",
#                                "terminal.restart": "ctrl-r"}}

# 任意语言：宿主 -shell 接受"命令 + 参数"（示例为纯标准库 Python 程序）
/tmp/anytty-tui2 -shell "python3 clients/tui/examples/python-shell/shell.py"

# 老 UI 像素复刻（规格见 LEGACY_PARITY.zh-CN.md）
/tmp/anytty-tui2 -shell "python3 clients/tui/examples/python-shell/legacy.py"
python3 clients/tui/examples/python-shell/parity_test.py

# 老 v3 TUI 像素复刻（规格/缺口见 V3_PARITY.zh-CN.md）
/tmp/anytty-tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py"
/tmp/anytty-tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py --demo"
python3 clients/tui/examples/python-shell/v3_parity_test.py

# v3 切分树/浮窗（120×40 tmux）：Enter 建终端 → Ctrl-P Ctrl-D 左右分（58/57）
# → Ctrl-P Ctrl-E 只切右侧 → 三 leaf 36x58/17x57/16x57 → 拖右侧分隔条 23/10
# （左侧不动）→ resize 100x30 保持比例 26x48/16x47/7x47 → Ctrl-P x 关闭右下恢复
# 单 leaf 36x57；footer NORMAL/PANE/PICK 每键颜色见 V3_PARITY §1.4。
# Ctrl-O n 浮窗（自动建终端 28x94）→ z 折叠只留标题 → 标题拖拽 → x 关闭。
/tmp/anytty-tui2 -shell "python3 clients/tui/examples/python-shell/v3ui.py"

# 老推荐配置效果（默认已开启；映射与码点见 RECOMMENDED_CONFIG.zh-CN.md）
/tmp/anytty-tui2-shell --print-default-config
printf '{"preset": "recommended"}' > /tmp/legacy-rec.json
/tmp/anytty-tui2 -shell "python3 clients/tui/examples/python-shell/legacy.py --config /tmp/legacy-rec.json"

# footer 场景规格 golden（M22）：两侧同一文件逐字符比对
go test -count=1 -run TestRecommendedFooterGolden ./clients/tui/cmd/tui2-shell/
python3 clients/tui/examples/python-shell/legacy.py --footer-lines --config /tmp/legacy-rec.json
```

常用按键（细节以 `?` help overlay 与 `SCENARIOS.zh-CN.md` 为准）：
NORMAL 下 `Ctrl-P` 进 PANE、`Ctrl-F` picker、`Ctrl-T` 新 tab、
`Ctrl-W` 侧栏（默认开，再按隐藏）、`Ctrl-E` 重启退出终端、`PgUp/PgDn` 回看；
PANE 下 `%`/`"` 分屏、`x` 关槽（只解绑）、`Tab` 切槽、`1..9` 切 tab、
`:` 命令、`?` 帮助；回看态 `y` 复制、`Esc` 回 live；`Ctrl-Q` 宿主确认退出。
每个槽标题栏还有鼠标动作按钮（默认推荐图标 `󰑐` restart、`󰅖` close 等，
`icons: "unicode"` 时是 `⟳ ⇔ ⇕ ✕`，分别等价 `Ctrl-E`、`%`、`"`、`x`，
实现见 `CUSTOMIZE.zh-CN.md` §5.1）。
以上除保留键外均可用配置 `keybindings` 覆盖（白名单见 `CUSTOMIZE.zh-CN.md`）。
默认外观（主题/图标）即老版推荐配置；`icons`/`endpoints`/推荐键位见
`RECOMMENDED_CONFIG.zh-CN.md` 与 `CUSTOMIZE.zh-CN.md` §2.2/§3.1。

## 4. 下一步队列（优先级从上到下）

1. **多客户端（§6 租约/CAS）**：同机第二 view 的 `fit:false` 跟随、TTL 15s 接管、
   `owner conflict`；当前 `cmd/tui2` 只起一个 view，需要宿主支持多连接才有黑盒入口。
2. **背压/基准（§13.9/§13.10）**：`throttled|oversize|view_rejected` 的落盘断言与
   10k 行/s + Ctrl-Q p99 延迟；已有 runtime 单测覆盖错误码，缺端到端。
3. **注册表一致性（§13.8）**：`HELLO.methods` 与 PROTOCOL §4 表逐项扫描的自动断言。
4. **`terminal.create{ephemeral:true}` 清算路径**：`cleanup_owned:true` 的端到端用例
   （当前 `:quit` 固定 cleanup=false 的解绑式语义）。
5. **连接层 G1 桥接（`CLIENT_SHARING.zh-CN.md` §5）**：给 `clients/tui/endpoint` 加
   `sessionConn` 小接口，把 manager/remotepty 从裸帧 client 迁到共享
   `protocoladapter.ApplicationClient` + `ResourceStream`，随后 direct/cloud 由
   共享 dialer 实拨（不再需要 WebRTC/云依赖进 tui2）；不阻塞 P0 ssh 隧道用法。
6. **v1 清单收口（M13）**：上面 4 条完成后，第 1 节全勾。
   （M15 遗留项已由 M16 清空：terminal 边框颜色经 `content.props` 随 shell 主题变化；
   协议有程序 → 组件属性通道；`proto.Border` 已删除并 `reserved`。）

## 5. 已知坑（踩过或必须显式处理）

- **事件 id 只属于投递给程序的 key/paste**：PTY 直通的输入不离手、无 id；`input.forward`
  过期回 `RESPONSE{ok:false,error:"event expired"}`；历史随 epoch reset 清空。
- **OnOutput 必须接线**：`TerminalHandler.OnOutput` → `Session.MarkOutput`（`cmd/tui2` 已接），
  否则程序输出不会触发重绘。宿主帧循环用 16ms tick + `FrameBytes` diff，无变化不写字节。
- **鼠标/滚轮条件要实时探测**：路由时重探 mouse tracking；宿主负责命中测试与隐式捕获
  （非终端且声明 `input:["mouse"]` 的盒子），终端类盒子拖拽一律走 PTY。
- **Esc 歧义**：裸 Esc 在 parser 缓冲，超时（50ms）后由 host `Flush` 提交，否则序列前缀会被吞。
- **`terminal.create` 现为真创建**：`cmd/tui2` 的 gate 分配 `term-N` 并直接 attach+fit；
  `terminal.restart` 同 id 重建 PTY；`terminal.kill/remove/system.quit` 走 core overlay 确认。
- **换行/样式**：内核文本不换行、按 rect 截断；内核无 border（`box.border` 已删除并
  `reserved 6`），边框/标题/颜色由内容（widget 文本行）或组件自绘；terminal 组件自绘并
  声明 inset=1，宿主按 `rect − inset` 算 PTY 尺寸与光标。
- **props 只是程序 → 组件的透传通道**：`content.props` 的 key 语义归组件（terminal 认
  `chrome.*`），宿主 kernel/runtime/host 一律原样搬运、不校验不解释；未识别 key 忽略，
  无 props 时组件回落内置默认，所以新增 key 不需要改协议或宿主。
- **单机范围**：endpoint 只有 local；远程由连接层归一化成同形事件流，程序不特判 endpoint。
  远程接入（M27）不改这条边界：`daemon` endpoint 经 local-unix/tcp（含 `ssh -L`
  隧道）归一化成同一 `runtime.Terminal`，picker 只多出 endpoint 分组与 health。
- **退出码要自己传**：`pty` 的 wait goroutine 记录 `ExitCode()`（信号 = 128+sig），
  `sources.exit_code` 与变更检测必须带上它，否则 `[exited N]` 只剩 `[exited]`。
- **copy 必须落 OSC52**：`cmd/tui2` 已接 `Clipboard` → `\x1b]52;c;<b64>\x07`；
  tmux 验证需 `set-clipboard on`（默认 `external` 只转发不建 buffer）。
- **程序 `pos` overlay 要不透明**：合成时先把 overlay 子树包围盒清成默认空格，
  否则底层分隔条/文本会从 picker/prompt/help 的空白处透出来。
- **bracket paste 只在前台是 ZLE 时打开**：shell 跑前台命令时会 `2004l`，长粘贴分块测试
  要自己 `printf '\033[?2004h'`；粘贴尾块的 `201~` 会占一行 canonical 行缓冲，
  `cat` 结束要连按两次 `C-d`（第一次冲标记、第二次 EOF）。
- **焦点/状态文案**：绑定已有 source 的 `bound` 提示要用短 id（`shortSourceID`），
  否则与 `terminal.create` 路径显示不一致（`bound terminal:local:term-2`）。
- **resize 是发给程序的信号，不是宿主能替它做的布局**：宿主只改视口/合成器并发
  `resize` 事件；少了这一帧，显式几何的 VIEW 会把 placement rect 和 PTY 钉死在
  旧尺寸（M18 Bug 1）。
- **点击 = 稳定 id + `input:["mouse"]` + dispatch 三件套**：只改选中高亮不执行业务
  动作等于没实现（picker 行），没有 id/input 的装饰行宿主永远命中不到（侧栏行）；
  新增可点行必须同时补 view 断言、mouse 单测与抓屏验收（M18 Bug 2）。
- **教训**：键盘能做的事，鼠标入口必须有等价路径，并且两条入口由同一组行为用例
  覆盖——"已实现"只算到 dispatch 为止，落不落业务要按可观察结果断言。
