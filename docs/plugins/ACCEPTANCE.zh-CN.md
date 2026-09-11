# 插件首版验收记录

日期：2026-09-11。范围为公共仓库 `anytty` 中的 endpoint daemon / TUI 插件 SDK 与首个 Agents 插件。设计经过 [协议评审](../design/PLUGIN_REVIEW_PROTOCOL.zh-CN.md) 和 [界面评审](../design/PLUGIN_REVIEW_UI.zh-CN.md) 后实施，后续实现验证另列如下。

## 可验收能力

- 生成式 Go / TypeScript Protobuf SDK；所有插件请求、回复、UI 更新与交互均经过选定 endpoint daemon，包含当前 TUI 发给自身的消息。
- 每个 endpoint 独立注册、稳定 daemon 身份、会话 lease、目标 epoch、定向及 scoped broadcast；远端状态由 TUI 插件通过各连接聚合。
- daemon / TUI 独立插件进程，清单、能力声明、安装/开发链接、启停注册、检查和日志。安装只影响后续实例。
- workspace / tab / panel / floating owner 与合法 slot，声明式 UI、键盘和鼠标、用户快捷键覆盖、类型化 UI 查询和带交互上下文校验的终端换绑。
- Agents 列表、终端状态徽标、跨 endpoint 选择终端；Codex 与 OpenCode Hook 适配、状态持久化、断线陈旧标识与重新同步。

具体 API、挂载位置与边界见 [SDK 使用说明](README.zh-CN.md)、[TUI SDK](TUI_SDK.zh-CN.md) 和 [Agents 说明](../../plugins/agents/README.md)。

## 验证方法与证据范围

- 最终 `env -u ANYTTY make test` 全仓检查通过。重点集成测试启动两个独立临时 UNIX socket daemon、两个独立 TUI 插件进程，并调用真实 Hook CLI，验证跨 endpoint 状态聚合和消息只发给原 TUI。
- TypeScript SDK 七项测试通过；SDK 测试覆盖真实 Node → Go bridge → 独立 daemon，以及取消、deadline、满载、有界帧；registry 测试覆盖并发更新与包快照安装。
- TUI 测试将真实 Agents `BuildMount` 输出交给实际 reducer / renderer，覆盖本地和远端同名终端的徽标隔离、键盘鼠标、owner 关闭、绑定冲突和恢复版本确认。
- 首轮真实 CLI / PTY 联调发现内部 terminal surface ID 含路径字符、不能直接充当插件日志名；现在插件采用独立 UUID 路由身份，并有 runtime 构造回归测试。
- 真实 CLI / PTY 还发现 daemon 重写点击 context ID，导致原 TUI 无法核对换绑回复；已统一为宿主生成、daemon 校验并登记同一个 ID，重复 ID 不可覆盖已有授权。TUI 保留完整上下文和绑定版本校验。
- 真实鼠标事件发现 SGR 的 1-based 坐标被直接用于 0-based 布局命中，导致点击偏移一行；插件命中现在复用宿主既有坐标归一化，并由实际渲染区域驱动回归测试。
- 客户端 Protobuf 生成产物同步后，UI / Web 共 616 项测试、类型检查和构建通过；Flutter 431 项测试通过，最终生成产物静态分析通过；本地 Web 嵌入包同步并通过一致性检查。
- OpenCode 1.18.20：用独立配置与本地测试服务器真实加载安装后的插件，创建/删除 session 验证事件，不调用模型。
- Codex 0.154.0：真实 app-server 通过进程级配置覆盖解析八项 Hook，保持 untrusted。此项验证解析，不宣称已经在用户配置中启用或运行 trusted Hook；实际启用仍需用户在 Codex `/hooks` 中审核。
- 公共发布文件集合的边界与 Markdown 链接检查通过。直接在现有工作目录运行同一检查，会命中原有未跟踪 `cloud/v1` 空目录和忽略的 Ghostty 包缓存断链；没有删除用户缓存来掩盖这些检查结果。

真实 PTY 最终复验：`Ctrl+G` → `a` → 下箭头 → Enter 从 Codex 切到 OpenCode，徽标更新为 working；鼠标点击 Codex 可见行正确选中，双击切回 Codex，徽标更新为 blocked。通过 `Ctrl+G` → `q` 正常退出并清理演示专属 daemon。

## 环境与首版限制

没有停止、重启或替换用户当前正在使用的 daemon，没有写入用户真实 Codex / OpenCode Hook 配置。临时集成测试与演示只清理自己创建的进程和 socket。

首版不包含独立 PTY 自绘 renderer、插件创建布局容器、终端文本/历史读取、热安装或 TUI 启动后动态扩展插件 endpoint 集合。聚合 UI 的 control daemon 固定；该 daemon 离线时界面暂停交互，恢复后同步权威版本，不做跨 daemon 自动接管。这些是明确保留的后续能力。

## 预览产物

仓库 `.artifacts/` 中提供原生 macOS 预览 CLI、Agents 插件包、TypeScript SDK 包和 `plugin-review/run-isolated-demo.sh`。演示脚本使用新建临时配置、socket 和测试 daemon，注入明确标记的合成 Hook 事件，不启动模型；通过 `Ctrl+G` 再按 `q` 退出。用户当前 daemon 运行旧代码，预览产物不会自动替换它。
