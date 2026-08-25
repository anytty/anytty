# 变更记录

[English](../../CHANGELOG.md) | 简体中文

本文件记录用户可见变化。当前版本仍为预发布版本，不提供稳定升级承诺；安全承诺见[安全边界](SECURITY_BOUNDARY.md)，使用说明见[文档索引](README.md)。

## Unreleased

### Changed

- 产品官网与面向用户的文档统一由独立的 `anytty/anytty-site` 仓库维护，文档链接统一指向 `anytty.com`。

### Removed

- 从公开源码仓库删除重复的 Astro 官网、网站 npm workspace 和 GitHub Pages 发布流程。

## [0.0.1-beta.2] - 2026-08-25

### Added

- 新增面向大宽屏的本地 Web 工作台，支持标签页、递归多分屏、拖拽布局预览和 `Ctrl+F` 终端选择器。
- 为通过外部反向代理开放的 Local Web 增加可选密码保护，包括 Argon2id 密码校验、登录限流和有界浏览器会话。

### Changed

- Local Web 改用同源 bridge，使 HTTPS 反向代理可以访问本地 daemon，同时不在浏览器里开放机器添加、注册或配对入口。
- Web 标签页会保留各自的分屏布局；桌面端和移动端拖放终端时只切分目标 pane。

### Security

- 带密码的 Local Web 公网入口必须使用安全来源；未启用密码时仍严格限制为回环地址访问。

## [0.0.1-beta.1] - 2026-08-24

### Added

- 新增 `anytty web`，按需启动仅监听回环地址的本地浏览器入口，复用终端和文件共享 UI，且不重启 daemon 或终端池。
- 新增 `anytty update`，从公开的 `anytty/anytty` GitHub Releases 检查并安装经过校验的归档，且不重启正在运行的 daemon。
- 在公开源码仓库新增 tag 驱动的发布自动化，统一生成 CLI 归档、未签名 Android APK、checksum、构建信息和 provenance attestation。

### Changed

- 后续安装脚本与 Release 资产统一以 `anytty/anytty` 为唯一来源；原 `anytty-site` Release 仅作为旧链接兼容入口保留。

## [0.0.1-beta.0] - 2026-08-17

### Added

- 增加基于 Markdown 的英中双语官方文档站、GitHub Pages Actions、本地构建预览和未来 `anytty.com` 自定义域名构建参数。
- 增加 Apache-2.0 开源发布所需的社区治理文档、Issue/PR 模板、CODEOWNERS、Dependabot 和发布检查清单。
- 增加公开仓库 HTML/Markdown 链接、双语页面结构、敏感路径、私有 import 和潜在凭据文件自动门禁。
- 增加 Local、SSH、Direct 和 Cloud 多 route endpoint registry、测试与诊断命令。
- 增加一次性二维码配对；Android App 只通过扫码添加设备，不登录也不自动发现设备。
- 增加客户端基线驱动的终端 Full/Delta 拉取、历史分页、搜索、范围复制和 Live/History 连续切换。
- 增加可选的 AnyTTY Cloud 连接，支持托管设备发现、P2P 协商、Relay 兜底，以及无需重启 daemon 的连接路径刷新。
- 增加 Cloud 公开文档页面、可搜索主题、响应式目录和真实产品说明。
- 增加完整项目 README、稳定专题文档和仓库文档索引。

### Changed

- PTY 输出改为每 terminal 单份有界 payload，并由 Live 与 History 独立 cursor 消费；溢出可配置为 `block` 或 `drop`。
- TUI 和移动端在提交当前 renderer 批次后立即重挂 long-poll，不使用固定帧率窗口；渲染期间合并最新 damage，不排队过时帧。
- App 前后台、WebView 重载和原生 session generation 变化会取消旧请求并从本地 endpoint registry 恢复。
- 历史模式冻结进入时的视觉锚点，滚动到最新位置自动返回 Live；大范围复制在确认时才物化文本。
- Cloud 路由只改变发现与传输方式，终端和文件权限仍由 daemon 控制。
- repository layout guard 改为检查当前稳定文档、错误路径和构建产物，不限制额外 Markdown。

### Security

- terminal 和 file 权限统一由 daemon 的 AccessStore 与 client-bound CapabilityGrant 校验。
- pairing claim 是短期一次性凭据，访问授权与客户端身份绑定并由 daemon 执行。
- 远程连接验证 daemon 身份；身份、认证或授权失败时拒绝连接，不会通过切换路径降低安全要求。
- 凭据使用安全存储并以原子方式更新，日志不记录秘密、终端内容或文件内容。

### Removed

- 删除账号自动发现移动设备的产品假设；Cloud 账号与 App endpoint registry 保持独立。
- 删除重复 TUI 配置模板、已完成整改计划、过期架构草案和旧开发工作流文档。
- 删除未发布旧协议、旧 YAML 和开发数据格式的兼容承诺。
