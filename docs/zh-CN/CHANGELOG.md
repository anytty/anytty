# 变更记录

[English](../../CHANGELOG.md) | 简体中文

本文件记录用户可见变化。当前版本仍为预发布版本，不提供稳定升级承诺；安全承诺见[安全边界](SECURITY_BOUNDARY.md)，使用说明见[文档索引](README.md)。

## Unreleased

## [0.0.1-beta.9] - 2026-09-02

### 变更

- macOS CLI Release 压缩包现在改为在 macOS 上构建，并使用明确的代码签名替代 Go linker 生成的临时签名。

### 修复

- macOS 安装器和自更新会在原子替换现有程序前，对候选可执行文件进行本机签名并严格校验，避免新安装的 CLI 被 Apple System Policy 直接终止。
- 更新检查现在支持使用 `GH_TOKEN`、`GITHUB_TOKEN` 或已登录 GitHub CLI 的凭据完成认证，避免匿名 GitHub API 请求更容易触发的共享 IP 限额。
- updater 现在会有界重试 GitHub 及 Release 资源 CDN 的瞬时连接失败、HTTP 429 和服务端错误。

## [0.0.1-beta.8] - 2026-09-02

### 新增

- 新增 Windows 安装器 CI 测试，覆盖 Release 压缩包安装、校验和验证、推荐配置安装以及已有配置保留。

### 变更

- 更新 Windows 安装命令，在默认 PowerShell 执行策略阻止下载脚本时也能运行，且不会修改用户的持久策略。

### 修复

- Windows daemon 探测遗留 AF_UNIX socket 时现在能够识别 Winsock `WSAECONNREFUSED`，并替换陈旧 socket，不再启动失败。

## [0.0.1-beta.7] - 2026-09-02

### 新增

- daemon 资源采样周期和每个 terminal 的保留采样点数现在可通过 YAML、环境变量及 CLI 配置。

### 变更

- TUI pane 进入 zoom 后会占满整个终端视口，不再显示 header、footer、pane 边框、溢出标记，也不再保留隐藏的 chrome 点击区域。
- terminal picker 现在默认显示运行中的 terminal，筛选顺序调整为 Running、Exited、All。
- 更新仓库文档中的原生移动端截图。

### 修复

- 资源采样改用有界、按时间排序的环形缓冲区；TUI 会保留 daemon 提供的完整历史窗口，不再截断为 64 个采样点。

### 安全

- 将 Browserslist 更新至 4.28.8，修复缓存无限增长和自定义统计数据不安全标准化问题。
- 将 gRPC-Go 更新至 1.83.1，防止碎片化 HTTP/2 DATA 帧导致堆内存耗尽。

## [0.0.1-beta.6] - 2026-09-01

### 新增

- 新增 Android 与 iOS 原生 Flutter 客户端，直接使用 AnyTTY Go Client Engine 与固定版本的 `libghostty-vt` 终端输入适配层。
- 手机端新增可搜索的 endpoint 与 terminal 选择器、原生终端历史和文件流程、可配置快捷键、触控花瓣菜单、外观设置与 route 管理。
- CLI 首次安装时自动写入推荐的 `coralline-candy` TUI 配置，同时保留用户已有配置。

### 变更

- 使用 Flutter 客户端替换 Capacitor/WebView 手机 App，同时保留 Android 包名 `com.anytty.app` 与原有正式签名发布路径。
- endpoint route 评分现在保留设备标签，Cloud 路径选择改为测量驱动，并自动完成客户端证书续期。
- Android 与 iOS 启动图标替换为 AnyTTY 正式图标，并补齐 Android 自适应与圆形图标。

### 修复

- 改进 endpoint 连接恢复，并统一 terminal 选择器与设备列表中的 endpoint 状态表达。
- 修正 Flutter 发布版本信息，确保 beta.6 Android 安装包可以按递增版本号从 beta.5 升级。
- 修正 iOS CI 构建顺序，在 Flutter 模拟器构建前生成所需的原生 XCFramework。

### 升级说明

- Flutter 客户端使用新的本地 registry 与安全凭据存储。从 Capacitor beta 升级的 Android 用户需要重新配对 endpoint；若现有 App 使用官方 release key 签名，则不需要先卸载。

## [0.0.1-beta.5] - 2026-08-26

### 新增

- 手机与 Web 终端列表新增运行中、已退出、全部和公开标签筛选。
- 将“结束运行中终端”和“移除已退出记录”拆成两个独立操作，分别确认并明确历史保留行为。
- Android 新增有界、脱敏的诊断包，仅在用户主动操作后生成并打开系统分享面板。
- Android 原生构建新增 ARMv7（`armeabi-v7a`），覆盖支持范围内的 32 位设备。
- tag 驱动的发布工作流新增签名的单 ABI APK、universal APK 和 Google Play AAB。

### 变更

- 前台恢复现在记录结构化阶段诊断；单个离线 endpoint 不再把其他健康 endpoint 的恢复变成全局 App 失败。
- 终端选择器新增结果计数，长列表会保持键盘选中项可见；位置标签只展示用户值，不再暴露协议占位键。
- 扩展内置 AnyTTY terminal skill，补充 endpoint 作用域的远程终端与文件操作，并收紧授权边界说明。
- Cloud operator 协议在数值字段之外新增稳定、可读的投影字段。

### 修复

- Android 系统 WebView 过旧时不再白屏，改为原生兼容提示页，显示当前 WebView 与 Chromium 101 最低要求。
- 原生 generation 恢复期间会隔离终端渲染和输入，避免重连时挂载旧 session。
- 修正 Android 原生恢复日志，导出的诊断信息保持结构化，且不包含 endpoint 标识或服务地址。

## [0.0.1-beta.4] - 2026-08-26

### Fixed

- Cloud daemon 准入现在会携带结构化的 daemon 名额受限详情，`anytty cloud status` 会展示可操作的 `quota_limited` 状态，不再无限停留在等待 Edge 就绪。

## [0.0.1-beta.3] - 2026-08-26

### Added

- 增加结构化 Cloud 订阅失败详情，让客户端可以区分订阅未生效、区域不可用、Relay 套餐受限和服务异常。

### Changed

- 桌面端、移动端、CLI 和 TUI 现在会展示订阅状态对应的恢复建议，不再统一显示为普通连接失败。
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
