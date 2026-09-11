# AnyTTY 插件 SDK：首版使用说明

本实现面向 TUI 与 endpoint daemon。设计与两个独立设计评审见 [设计稿](../design/PLUGIN_SDK_V1.zh-CN.md)、[协议评审](../design/PLUGIN_REVIEW_PROTOCOL.zh-CN.md)、[界面评审](../design/PLUGIN_REVIEW_UI.zh-CN.md)。评审通过不等同于运行验证；各项验证记录见 [验收记录](ACCEPTANCE.zh-CN.md)。

## 启动与安装

一个包可以同时声明 daemon 和 TUI 入口，两者都是独立子进程。daemon 入口随**启动该入口的新 daemon**运行，TUI 入口随新的 TUI 运行。配置不会重启现有 daemon，也不会把新插件代码注入旧版本 daemon。

```sh
go build -o .artifacts/bin/anytty-plugin-preview ./cmd/anytty
go build -o .artifacts/plugins/agents/anytty-agent-plugin ./cmd/anytty-agent-plugin
cp plugins/agents/anytty-plugin.toml .artifacts/plugins/agents/
.artifacts/bin/anytty-plugin-preview plugin install "$PWD/.artifacts/plugins/agents"
.artifacts/bin/anytty-plugin-preview plugin doctor
```

`install` 复制包到注册表旁边的独立目录，保留正在运行实例使用的旧文件；`link` 直接引用开发目录。`list` 查看注册表，`enable` / `disable` 切换后续实例是否加载，`uninstall` 移除注册项但不删除用户状态或正在使用的包文件。默认注册表为 XDG 配置目录中的 `anytty/plugins.yaml`，可通过 `ANYTTY_PLUGIN_REGISTRY` 或管理命令的 `--registry` 指定。读改写使用跨进程锁和原子发布。

日志位于注册表旁的 `plugin-logs/插件ID/`。`plugin logs ID` 读取 daemon 日志末尾 64 KiB；`--component tui-实例ID` 指定 TUI 日志。`doctor` 只检查配置与程序文件，不运行插件。

daemon 会轮询注册表并只重载发生变化的插件子进程，因此安装、启用、禁用或卸载不会重启 daemon 核心，也不会影响其他插件。TUI 插件清单在 TUI 实例启动时确定；新增 TUI 插件或改变该实例的 endpoint 集合时，需要新开 TUI 才会建立对应连接。远端 daemon 的插件需要在远端机器安装，客户端不会因为连接了远端而自动下载、安装或执行远端程序。

## 消息与身份

协议唯一来源是 `proto/apipb/plugin.proto`。Go SDK 位于 `plugin/sdk`，TypeScript SDK 位于 `plugin/sdk-ts`，不依赖私有服务仓库。stdio 使用有大小上限的 varint 长度前缀 Protobuf frame，stderr 用于日志；不使用 JSON-RPC。

每个 TUI 在各个连接的 endpoint daemon 分别注册。地址包含稳定 daemon ID、TUI 实例、插件及组件实例、注册 epoch。客户端 endpoint 别名仅用于选择连接；不能代替跨机器资源身份。秘密 source lease 绑定原认证会话，公开目标地址不包含 lease。

即使插件给当前 TUI 发消息，也按 `插件 → SDK bridge → 选定 daemon → TUI host` 流转；回复原路经过 daemon。Agent 状态通过每个 daemon 的原子 snapshot + watch 同步，再由 TUI 插件聚合。没有新增 daemon 之间的中继或集中远端状态管理。

`Send` 的成功只说明 daemon 接受投递。业务完成以对应 `PluginReply` 为准。桥接支持并发请求、取消、deadline 和有界缓冲；状态 revision 用于 CAS，队列超限要求重新建立快照订阅。delivery sequence 仅为诊断顺序，不是可恢复历史游标。

## 读、写与交互边界

- [读取 TUI](../../plugin/sdk/QUERY.md)：通过 `ui.read` 查询 owner、当前绑定、焦点和 generation，返回类型化 Protobuf 快照。不会把内部 reducer 或连接凭据暴露给插件。
- [挂载与交互](TUI_SDK.zh-CN.md)：workspace、tab、panel、floating 的合法 slot、声明式组件、键盘/鼠标、用户快捷键覆盖和容器生命周期。
- 换绑要求真实用户交互签发的短期 context，并检查原面板及绑定版本。查询当前焦点本身不授予修改权限；异步操作也不会跟随之后的焦点漂移。
- `state.read` / `state.write`、`events.subscribe`、`messages.send`、`ui.read`、`ui.mounts`、`ui.panes.bind`、`ui.notifications` 为对应宿主 API 能力。写状态仅开放给该插件的 daemon service。挂载还必须匹配清单声明的 ID 和 slot。
- 插件是受信任的普通 OS 进程，capability 不是 OS 沙箱。同用户插件仍可能访问文件和网络；只向已认证 full-access 会话开放插件协议，share / machine-events 受限会话不借此扩大权限。

独立 PTY 自绘 renderer、插件自行创建布局容器、终端屏幕/历史/选择文本读取及完整 transport provider 尚未开放。它们属于后续设计扩展，不以空白 renderer、无效成功回复或伪造内容代替。

## 第一个插件：Agents

[Agents 插件说明](../../plugins/agents/README.md) 包含构建、两种 Hook 安装和状态映射。每个 workspace 聚合已连接 endpoint 的 Agent，panel 徽标仅显示自身绑定终端的状态。键盘可通过现有 system 菜单的插件焦点动作进入列表，也可用鼠标；点击卡片或按 Enter 打开 Agent 终端，`f` 筛选需要处理的 Agent，Esc 返回终端。

聚合挂载的 control daemon 在创建时固定。该 daemon 断线时，列表暂时陈旧并停止交互；其他 endpoint 的订阅仍运行，control 恢复后通过权威挂载版本重新同步。首版不跨 daemon 自动接管同一个挂载，避免旧上下文被重新解释。普通非 control endpoint 断线只使对应 Agent 数据陈旧。

Codex 与 OpenCode 的外部 Hook 格式由各自产品定义；适配器读取后转成 Protobuf。Hook 需要 daemon 注入的终端 ID 和明确 socket，在 AnyTTY 外不报告。Codex 的 Hook 信任须由用户在 Codex 中审核，安装器不修改信任数据库。
