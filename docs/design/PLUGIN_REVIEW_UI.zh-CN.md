# 插件设计独立审核：TUI、挂载与交互

审核对象：`PLUGIN_SDK_V1.zh-CN.md` 修订 2 及后续修订。初审发现下列 4 项阻塞；复审结论：**PASS（设计）**。本审核未修改实现。

## 阻塞项

### UI-1：floating 的 owner 少了 tab 归属

设计 8.1 定义 `floating(workspace_id,floating_id)`，但代码 `tui/state/shell.go` 中 `TabState.Floatings` 明确由 tab 持有；`tui/app/workbench_storage.go` 的 `localFloatingKey` 也是 workspace/tab/floating 三元键。

修正：owner 必须是 `floating(workspace_id,tab_id,floating_id)`。声明关闭 tab 时关闭其 floating 挂载；切换 tab 时隐藏；浮窗折叠、隐藏、关闭分别定义。避免通过当前 active tab 解释异步浮窗消息。

### UI-2：挂载清单缺少自动实例化规则与合法矩阵

设计正确区分 owner 和 slot，但示例 `sidebar`、`panel.header` 的 auto_mount 没有说明对哪个 owner 实例化。statusbar/notification 等也没有明确是 workspace 作用域，还是需要新增 TUI 级 owner。两个 daemon 同时提供数据时，是创建两个列表，还是一个聚合列表，同样未落定。

修正：给出 owner × slot 合法矩阵，并补 `owner_selector` 或等价规则（例如每个 workspace 一个 sidebar，每个 terminal panel 一个 badge），唯一恢复键和重复注册语义。首插件明确一个 workspace 聚合列表，节点的资源引用各自带 daemon 身份；相应面板徽标按绑定资源过滤。宿主核心状态栏中的插件分段，不能与插件独立页面混用同一 owner/slot 解释。

### UI-3：布局隔离仍是建议和未来审批项，无法支撑双 TUI 验收

设计 12 节指出问题是正确的。但当前 `reduceWorkbenchStorageChanged` 自动生成 LoadRequest，LoadResult 会替换 `TerminalViews` 并产生 attach effects。单纯路由指定 TUI A 不会阻止存储 watch 将 A 的绑定传播到 B。

修正：本次实施明确选择实例内绑定覆盖层，或明确选择运行布局独立＋外部快照仅提示导入。写清普通手工换绑是否同样隔离、启动恢复和共享 profile 保存冲突如何处理。验收必须让 B 真正消费 A 触发的存储事件，再确认其绑定不变，不能只断言路由收件者。

### UI-4：首插件仍缺少可执行产品与 Hook 接入契约

只有通用 SDK 示例和一句 Codex/OpenCode 验收，尚缺首插件页面字段、筛选、动作与真实 Hook 事件对应关系。尤其不能以 Agent 完成通知推断完整 working/waiting 生命周期。

修正：添加首插件明确规格（endpoint、Agent 类型、项目/会话、状态、更新时间；全部/需关注过滤；鼠标点击、方向键、Enter、Escape；失联标 stale）。基于官方资料列出两个 Agent 的配置入口、实际事件、支持版本、载荷到状态映射、未知状态规则、独立安装/卸载及本地真实执行验收。只支持部分事件必须真实显示支持范围，不能靠扫描屏幕伪造等待状态。

## 非阻塞但实施必须覆盖

- 现有 `tui/shortcut/registry.go:SceneByName` 只接受内置场景；不要把插件动作硬编码到现有 catalog 冒充动态注册。注册、禁用、崩溃和 owner 关闭后，同步删除动作及命中目标。
- 快捷键作用域优先级已有设计，但应以同一解析器生成帮助、诊断和实际派发，避免显示能用而输入进终端。保留键优先级必须高于 focused component。
- 异步 click/select 应附带 mount、节点版本、固定目标绑定版本；鼠标滚轮和键盘选中应有一致选择行为。隐藏组件不收按键；Escape 恢复到此前有效内容面板。
- owner 关闭时只取消 mount 自己的订阅任务，不能取消同插件其他 mount 或聚合数据服务共享的订阅；引用计数或独立订阅均可。
- 浮窗窄到小于最小尺寸和小终端（例如 80×24）时必须可退出并能访问动作，不能阻挡全部终端输入。
- 声明式组件本地滚动/焦点与跨插件消息边界在文档中已清楚，符合“所有插件消息经 daemon”的要求。
- 多 endpoint 模型与当前 SessionOwner 所有权一致；终端身份必须保留稳定 daemon ID，出站 endpoint 别名只在本机解析。

## 复审标准

上述 4 项落实到主设计后可 PASS 设计。PASS 只表示允许按设计实施，不代表接口、挂载及真实 Hook 已实现或验证。


## 复审结果（2026-09-11）

**PASS：四项设计阻塞已解决，可以进入实现。**

- UI-1：8.1 已补全 floating 的 workspace/tab/floating 引用，8.2 明确切 tab 隐藏和关 tab 卸载。
- UI-2：8.2 已定义 owner_selector 和 owner×slot 矩阵；每 workspace 单个聚合列表、每 panel 按终端过滤徽标、控制路由域和行来源域分离。
- UI-3：12 节已确定启动恢复模板、运行实例独立、外部 watch 只提示导入，并将真实 watch 验证纳入本次实现。
- UI-4：8.2 已给出具体列表字段、稳定选择、键鼠操作；11.1 已列两个 Agent 的配置入口、事件映射、信任与配置隔离、错误和未知状态，以及事件夹具和实际触发验证。复审查看了所引官方 Hook/插件文档。

剩余实施细节不阻塞设计：恢复键应含插件/owner/slot/清单挂载 ID；自动挂载对重复注册须幂等；全局通知可采用 workspace overlay/notification 展示，新增 slot 必须经过矩阵校验。保存冲突不能重新载入并覆盖当前实例布局。

此 PASS 不证明功能已经存在。真实插件宿主、SDK 往返、所有挂载入口、动态快捷键注册、双 TUI 的实际 watch 隔离及两个 Agent Hook 的可执行验证，仍必须按主设计逐项验收。
