# TUI 插件界面与交互

界面消息使用 `proto/apipb/plugin.proto`。插件首先向精确的 host 注册地址发送空 `PluginUiInit`，TUI 返回当前 workspace/tab/panel/floating 引用。请求和返回都经过所选 daemon，不能自行构造本地挂载事件。

## 本轮可用挂载

| owner | slot | 呈现 |
| --- | --- | --- |
| workspace | sidebar | 为终端预留空间的侧栏；窄屏聚焦时临时展开 |
| workspace | statusbar | 终端内容区下方独立状态行 |
| workspace | menu / overlay | 居中的声明式列表或按钮页面 |
| tab | header / menu / overlay / content | 标签页装饰、弹出页面或标签页内容 |
| panel | header / menu / content | 面板装饰、页面内容 |
| floating | header / menu / content | 浮窗装饰、页面内容 |

浮窗引用必须同时包含 workspace、tab 和 floating ID。切换标签隐藏其挂载；关闭容器会卸载从属挂载、撤销交互上下文，并经 daemon 通知插件。新增容器会触发新的 `PluginUiInit` 所有者清单。独立标签、面板和浮窗的创建仍由宿主现有布局命令完成；挂载 API 不隐式创建缺失 owner。

当前声明式组件支持 text、badge、progress、button、list、row、card、gap、column、table、tree，以及 form/input/select/checkbox。表格按文本行显示；树分支可以展开或折叠。`card` 是宿主统一渲染的可选中卡片，插件通过节点的 `description` 提供第二行元信息，通过 `style` / `selected_style` 声明 `primary`、`muted`、`success`、`warning`、`danger`、`info` 前景语义和 `surface`、`elevated`、`selected`、`transparent` 背景语义；`layout` 控制上下左右留白和卡片间距。宿主把这些语义映射到当前主题，插件不写 ANSI，也不依赖 `tui/render` 的实现。`gap` 只表达布局间隔，不进入键盘选中或鼠标命中列表。节点 ID 必须唯一，最多 4096 个节点、16 层。不接受 ANSI 或多行文本。当前尚未实现独立 PTY renderer；注册该组件明确返回 `UNSUPPORTED`，不能把空白页面当作支持。后续完整 SDK 的目标能力以设计稿为准。

`PluginUiInit.mount_revisions` 返回请求插件在当前 daemon 域内的权威挂载版本；缺失的挂载视为版本 0，不透露其他插件或其他 daemon 的挂载版本。插件须在宿主业务回复成功后推进版本，更新超时或冲突时重新 Init 对齐；daemon 收到消息不代表界面已经应用。`expected_revision` 必须对应当前挂载版本。更新保留按行 ID 定位的选中项，列表重新排序不会把操作目标换成其他 Agent。面板徽标中附有 terminal 引用的节点会按该面板当前绑定过滤。

## 键盘和鼠标

通过默认 Ctrl+G 系统菜单的 a/A（`plugins.focus_next` / `plugins.focus_previous`）循环进入插件挂载；这两个宿主动作也可配置到全局快捷键，不依赖鼠标或启动时抢焦点。

单击选择行，双击或 Enter 执行动作；上下键或 j/k 移动；/ 进入搜索；Escape 先退出搜索，再恢复终端焦点。点击宿主终端面板也会离开插件焦点。插件聚焦时不会把未处理的普通按键泄露给后台终端，宿主导航和复制入口保留。

表单中单击输入框或 Enter 开始编辑，支持光标左右/Home/End、删除和安全的单行粘贴；Escape 撤销当前编辑，Tab/Shift+Tab 切换控件。选择框支持鼠标循环、键盘左右/上下选择；复选框用单击或空格切换；按钮可单击或 Enter 提交。修改、取消、提交携带 Protobuf `values` 字段并经 daemon 送给插件。树分支的左右键与点击会发送 expand/collapse 语义事件。

清单动作的默认按键在注册时动态生效。解析顺序为聚焦挂载、panel/floating、tab、workspace、用户显式启用的 global；同层冲突禁用并显示诊断。global 默认按键不会自动启用。

可在现有 TUI 配置中覆盖或禁用插件快捷键，不替换核心快捷键配置：

```yaml
tui:
  plugin_shortcuts:
    org.anytty.agents/agents.filter: ctrl-alt-f
    org.anytty.agents/agents.open: enter
```

键名为 `插件 ID/动作 ID`；设置为 `""` 禁用。覆盖遵守动作自身的作用域。若动作声明 `global`，提供非空用户覆盖才会启用全局绑定。Escape 始终保留，不能覆盖。界面提示采用覆盖后的按键。

## 定向操作

交互捕获发起 TUI、mount、来源节点、目标面板及绑定版本，携带短期 context ID 经 daemon 发送。插件请求换绑时必须原样返回 context。TUI 校验来源插件、endpoint、上下文和目标版本，先异步 attach，再重新校验并提交；焦点变化不重定向，关闭或换绑竞争返回错误。重复使用已消费 context 不会重复 attach。错误会显示在插件页面的提示行。

持久化 workbench 作为启动恢复模板。其他 TUI 保存后，本实例记录可加载版本，保持自己的布局和绑定。保存冲突也不会自动加载并覆盖用户当前工作台；显式载入仍使用现有入口。

## 首个 Agent 插件的连接限制

各 endpoint 的 Agent 状态独立同步，但首版聚合列表及面板徽标固定通过首次完成 Init 的 control daemon 挂载。其他 endpoint 断线只使对应 Agent 状态过期；control daemon 断线会使整个聚合界面暂时过期并暂停交互，即使其他状态源仍在线。control 恢复后通过 Init 对齐版本并重新发布完整界面。首版不自动迁移挂载到另一 daemon；跨域迁移需要额外的宿主挂载组授权与旧上下文撤销机制，尚未实现。
