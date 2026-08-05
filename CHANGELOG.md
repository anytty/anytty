# 变更记录

本文件记录尚未发布的用户可见变化。当前没有公开发布版本或稳定升级承诺；架构和稳定行为见 [ARCHITECTURE.md](ARCHITECTURE.md) 与 [文档索引](docs/README.md)。

## Unreleased

### Added

- 增加基于 Markdown 的英中双语官方文档站、GitHub Pages Actions、本地构建预览和未来 `anytty.com` 自定义域名构建参数。
- 增加 Apache-2.0 开源发布所需的社区治理文档、Issue/PR 模板、CODEOWNERS、Dependabot 和发布检查清单。
- 增加公开仓库 HTML/Markdown 链接、双语页面结构、敏感路径、私有 import 和潜在凭据文件自动门禁。
- 增加 Local、SSH、Direct 和 Cloud 多 route endpoint registry、测试与诊断命令。
- 增加一次性二维码配对；Android App 只通过扫码添加设备，不登录也不自动发现设备。
- 增加客户端基线驱动的终端 Full/Delta 拉取、历史分页、搜索、范围复制和 Live/History 连续切换。
- 增加 Cloud daemon `ACTIVE`、`BLOCKED`、`DELETED` 生命周期及 EdgeControl v6 收敛协议。
- 增加 Cloud Edge 候选测速、软偏好、网页/CLI 管理和无需重启 daemon 的在线重选；协议升级为 EdgeControl v7 与 AgentGateway v4。
- 增加 Cloud 公开文档页面、可搜索主题、响应式目录和真实产品说明。
- 增加完整项目 README、稳定专题文档、Cloud 部署模板和仓库文档索引。

### Changed

- PTY 输出改为每 terminal 单份有界 payload，并由 Live 与 History 独立 cursor 消费；溢出可配置为 `block` 或 `drop`。
- TUI 和移动端在提交当前 renderer 批次后立即重挂 long-poll，不使用固定帧率窗口；渲染期间合并最新 damage，不排队过时帧。
- App 前后台、WebView 重载和原生 session generation 变化会取消旧请求并从本地 endpoint registry 恢复。
- 历史模式冻结进入时的视觉锚点，滚动到最新位置自动返回 Live；大范围复制在确认时才物化文本。
- Controller 保存长期状态真值，Edge 只维护可由 snapshot 重建的在线策略和 Presence 内存投影。
- repository layout guard 改为检查当前稳定文档、错误路径和构建产物，不限制额外 Markdown。

### Security

- terminal 和 file 权限统一由 daemon 的 AccessStore 与 client-bound CapabilityGrant 校验。
- pairing claim、enrollment code 和 Edge bootstrap token 均为短期一次性凭据；前两者保留已消费记录以拒绝重放，只有 bootstrap token 在 Edge 注册后从配置移除。
- AgentGateway 与 ClientGateway 使用 Edge challenge-first proof，Cloud policy 缺失或失序时拒绝准入。
- Edge verification KeyBundle 和必要运行状态以原子私有文件持久化，日志不记录秘密或终端内容。

### Removed

- 删除账号自动发现移动设备的产品假设；Cloud 账号与 App endpoint registry 保持独立。
- 删除重复 TUI 配置模板、已完成整改计划、过期架构草案和旧开发工作流文档。
- 删除未发布旧协议、旧 YAML 和开发数据格式的兼容承诺。
