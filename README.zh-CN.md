<div align="center">
  <img src="docs/assets/logo.png" alt="AnyTTY Logo" width="120">
  <h1>AnyTTY</h1>
  <p><strong>让终端持续运行，随时随地接管现场</strong></p>
  <p>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-3d7ea6" alt="Apache-2.0 License"></a>
    <a href="https://github.com/anytty/anytty/releases"><img src="https://img.shields.io/github/v/release/anytty/anytty?include_prereleases&amp;sort=semver&amp;label=latest%20beta" alt="Latest Beta Release"></a>
    <a href="https://github.com/anytty/anytty/releases"><img src="https://img.shields.io/github/downloads/anytty/anytty/total?label=release%20downloads" alt="Total GitHub Release Downloads"></a>
    <a href="https://github.com/anytty/anytty/stargazers"><img src="https://img.shields.io/github/stars/anytty/anytty?style=flat&amp;label=stars" alt="GitHub Stars"></a>
    <a href="https://github.com/anytty/anytty/actions/workflows/ci.yml"><img src="https://github.com/anytty/anytty/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="docs/PACKAGE_MANAGERS.md"><img src="https://img.shields.io/badge/Homebrew-tap%20preparing-fbb040?logo=homebrew" alt="Homebrew Tap Preparing"></a>
  </p>
  <p><a href="README.md">English</a> · 简体中文</p>
</div>

AnyTTY 让终端会话持续运行在你自己的机器上，并可随时从键盘优先的 TUI、CLI 或手机 App 回到现场。同一条连接还提供实用的远程文件浏览能力，适合查看项目文件、日志与下载内容。

> **Beta：** `v0.0.1-beta.0` 已提供 macOS、Linux、Windows 和 Android 构建，可用于体验和评估；首个稳定版发布前，协议与配置仍可能变化。

<table>
  <tr>
    <td align="center"><img src="docs/assets/mac-tui.png" alt="macOS 上的 AnyTTY TUI 多终端工作台" width="520"></td>
    <td align="center"><img src="docs/assets/win-tui.png" alt="Windows 上的 AnyTTY TUI 多终端工作台" width="520"></td>
  </tr>
  <tr>
    <td align="center"><strong>macOS：分屏、浮动 Panel 与远程终端</strong></td>
    <td align="center"><strong>Windows：本地与远程终端统一管理</strong></td>
  </tr>
</table>

## 主要功能

- **Terminal 与 Panel 解耦。** `terminal` 是由 daemon 持续运行的任务，TUI 中的 `workspace` 和 `panel` 只是观察与操作它的工作界面。Panel 不拥有、也不绑定某个 terminal；通过 Terminal Picker，可以让当前 panel 快速切换到任意 terminal，而不改变已有的分屏、浮动窗或 workspace 布局。这与 pane 通常直接承载进程的 tmux、Zellij 等传统终端复用器不同。
- **Terminal 与消费者解耦。** TUI Panel、CLI、WebView 和手机 App 等交互消费者不会直接阻塞 terminal 的输出链路，而是按各自节奏读取最新画面快照，并通过独立 cursor 回溯历史。多个消费者同时连接时，terminal 不需要等待每个客户端完成网络传输或画面渲染；慢消费者只会让自己的视图暂时落后，重新读取最新快照即可追上，不会拖慢 terminal 中运行的程序，也不会影响其他消费者。
- **原生支持 Windows。** Windows 与 macOS、Linux 一样是正式支持的平台，CLI、TUI 和 daemon 均提供 Windows x64 与 ARM64 构建，不需要依赖 WSL 才能使用核心能力。
- **本地与远程终端属于同一个工作台。** 来自 Local、SSH、Direct 或 Cloud endpoint 的 terminal 都可以进入同一个 Terminal Picker，并像本地 terminal 一样查看、切换和操作。运行在远程工作站、服务器或构建机上的 AI Agent，也可以直接纳入本地工作流统一管理。
- **基于文件、理论上无限的历史记录。** 终端输出持续写入历史文件，实时界面只保留有界数据，因此内存不会随着累计历史长度线性增长，长期运行时仍可保持近似稳定。AnyTTY 不预设历史行数上限，实际容量只取决于可用磁盘空间；只要磁盘容量继续增长，历史记录就可以继续增长。即使程序意外退出、连接中断，或 AI Agent 挂机执行了很久，也能快速回溯完整上下文。
- **快速发现停止更新的终端。** 手机 App 的终端列表和 TUI 的 Terminal Picker 会集中展示终端状态与近期活动，便于快速定位长时间没有新输出的会话，进一步判断任务、服务或 Agent 是否仍在运行。
- **随时从不同设备接管。** 同一个 terminal 可以在 TUI、CLI、内置 WebView，以及支持 Android 与 iOS 的手机 App 之间切换，不需要迁移或重启进程。移动端针对触控和终端输入做了专门优化，提供 `Esc`、`Ctrl`、方向键、翻页键等扩展按键，适合离开电脑后临时查看和处理任务。
- **可选的 AnyTTY Cloud 内网连接。** 官方托管 Cloud 可以帮助位于 NAT 或内网中的 daemon 建立 P2P 连接；无法直连时自动使用 Relay 兜底，无需把终端服务直接开放到公网。
- **远程连接默认加密。** SSH、Direct 和 Cloud 路由都使用经过加密与身份验证的连接；配对权限绑定到具体客户端，并可随时从 daemon 撤销。Relay 只负责转发加密流量，不拥有终端或文件权限。
- **终端与文件使用同一条连接。** AnyTTY 不只传输终端画面，还提供文件浏览、上传、下载、重命名和选择能力。连接到内网或远程 endpoint 后，可以像处理终端一样处理该机器上经过授权的文件。
- **移动端文件管理与在线预览。** 手机 App 内置文件管理器，可浏览远程目录并预览常见文本、文档、图片和媒体格式，临时检查日志、配置、构建结果或下载内容时无需切换到其他工具。

### 移动端终端

<table>
  <tr>
    <td align="center"><img src="docs/assets/app_1.jpg" alt="AnyTTY 手机 App 的终端列表与活动状态" width="230"></td>
    <td align="center"><img src="docs/assets/app_6.jpg" alt="AnyTTY 手机 App 快速定位刚有输出的终端" width="230"></td>
    <td align="center"><img src="docs/assets/app_7.jpg" alt="AnyTTY 手机 App 使用扩展键盘接管远程终端" width="230"></td>
  </tr>
  <tr>
    <td align="center"><strong>多终端状态与近期活动</strong></td>
    <td align="center"><strong>快速定位刚有输出的终端</strong></td>
    <td align="center"><strong>通过扩展键盘直接接管</strong></td>
  </tr>
</table>

> **下载安装：** 手机 App 同时支持 Android 与 iOS。Android Beta APK 暂时可从 [GitHub Releases](https://github.com/anytty/anytty/releases/tag/v0.0.1-beta.0) 下载；iOS 当前还没有公开安装包，需要等待 Apple App Store 上架。Google Play 与 Apple App Store 的上架工作均在进行中。

### 连接与文件管理

<table>
  <tr>
    <td align="center"><img src="docs/assets/app_4.jpg" alt="AnyTTY 手机 App 的设备与 P2P 连接状态" width="260"></td>
    <td align="center"><img src="docs/assets/app_5.jpg" alt="AnyTTY 手机 App 的远程文件管理器" width="260"></td>
  </tr>
  <tr>
    <td align="center"><strong>Local、Cloud 与 P2P 路径</strong></td>
    <td align="center"><strong>远程文件管理</strong></td>
  </tr>
</table>

### 丰富的文件在线预览

<table>
  <tr>
    <td align="center"><img src="docs/assets/app_2.jpg" alt="AnyTTY 手机 App 在线预览远程文档" width="260"></td>
    <td align="center"><img src="docs/assets/app_3.jpg" alt="AnyTTY 手机 App 在线预览远程 3D 模型" width="260"></td>
  </tr>
  <tr>
    <td align="center"><strong>文档预览、搜索与缩放</strong></td>
    <td align="center"><strong>3D 模型预览与交互</strong></td>
  </tr>
</table>

### 平板多任务与终端应用

<table>
  <tr>
    <td align="center"><img src="docs/assets/pad.png" alt="平板上下分屏同时运行 Codex 与 OpenCode" width="360"></td>
    <td align="center"><img src="docs/assets/pad-ttt.png" alt="在平板终端中运行 TTT 编辑器" width="360"></td>
  </tr>
  <tr>
    <td align="center"><strong>上下分屏同时运行 Codex 与 OpenCode</strong></td>
    <td align="center"><strong>在终端内运行 TTT 编辑器</strong></td>
  </tr>
</table>

## 安装

### macOS 与 Linux

安装脚本会自动选择 x64 或 ARM64，下载对应 GitHub Release，校验 `SHA256SUMS`，并默认安装到 `~/.local/bin`。

```sh
curl -fsSL https://raw.githubusercontent.com/anytty/anytty/main/install.sh | sh
```

也可以指定版本和安装目录：

```sh
curl -fsSL https://raw.githubusercontent.com/anytty/anytty/main/install.sh | \
  sh -s -- --version v0.0.1-beta.0 --bin-dir "$HOME/bin"
```

### Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/anytty/anytty/main/install.ps1 -OutFile install.ps1
.\install.ps1
```

PowerShell 脚本会校验 SHA-256，安装到 `%LOCALAPPDATA%\Programs\AnyTTY\bin`，并把目录加入当前用户的 `PATH`。如不希望修改 `PATH`，可传入 `-NoModifyPath`。

你也可以直接从 [GitHub Releases](https://github.com/anytty/anytty/releases/tag/v0.0.1-beta.0) 下载各平台 CLI 压缩包和未签名 Android Beta APK。Homebrew、npm 与 WinGet 的包定义正在准备发布，当前状态见[包管理器发布说明](docs/PACKAGE_MANAGERS.md)。

## 快速开始

启动当前用户的 daemon，然后打开 TUI：

```sh
anytty daemon start
anytty
```

从 CLI 创建一个命名终端并立即连接：

```sh
anytty terminal create --name workspace --attach -- zsh
```

稍后查看或重新连接会话：

```sh
anytty terminal list
anytty attach workspace
```

执行 `anytty --help` 可查看完整命令列表，也可以继续阅读[中文快速开始](https://anytty.github.io/anytty/zh-CN/docs/quick-start.html)。

## 手机 App

从 Release 页面安装 Android Beta APK，然后在 App 中选择“添加设备”。在运行 daemon 的机器上使用 `anytty pair create` 生成短时配对二维码或文本 claim，再用 App 扫描或粘贴。配对完成后，设备页会列出运行中的终端，并可从终端工作目录打开文件浏览器。

手机 App 同时支持 Android 与 iOS。当前可以从 Release 页面安装 Android Beta APK；iOS 暂未提供公开安装包，需要等待 Apple App Store 上架。Google Play 和 Apple App Store 的正式商店版本都在上架过程中。

## 连接方式

| 路由 | 适用场景 | 是否需要 Cloud |
| --- | --- | --- |
| Local | 同一台机器上的 CLI 与 TUI | 否 |
| SSH | 已经可以通过 OpenSSH 访问的机器 | 否 |
| Direct | 自己管理的局域网或公网可达环境 | 否 |
| Cloud | 托管设备发现、P2P 协商与 Relay 兜底 | 是 |

Local、SSH 与 Direct 可以完全基于本仓库使用。官方 Cloud 路由是可选能力；它改变的是发现和传输方式，不改变 daemon 对终端与文件权限的最终控制。

## 从源码构建

CLI/TUI 开发需要 Go 1.26.5；共享 UI 与移动端使用 Node.js 24 及仓库中的 npm lockfile。

```sh
git clone https://github.com/anytty/anytty.git
cd anytty
npm ci
make build
```

二进制输出到 `.artifacts/bin/anytty`。主要检查命令为 `make test`、`make test-clients` 和 `npm run public:check`。Android 构建还需要 Java 21 与 Android SDK；iOS 构建需要 macOS 与 Xcode。

## 开源版本与 AnyTTY Cloud

AnyTTY 的 Apache-2.0 开源版本包含 CLI、TUI、daemon、共享 UI、Android 与 iOS 源码，以及 Local、SSH、Direct 等连接方式。你可以直接用它管理本地与远程终端、文件和长期任务，也可以自行构建各平台客户端。

在此基础上，AnyTTY 还额外提供官方托管的 Cloud 服务，负责设备发现、P2P 协商和无法直连时的 Relay 兜底，让位于 NAT 或内网中的终端也能更轻松地安全连接。Cloud 是即开即用的增强服务，终端与文件权限仍由用户自己的 daemon 控制。用户可见的信任模型见[安全边界](docs/zh-CN/SECURITY_BOUNDARY.md)。

## 项目入口

- [中文文档](https://anytty.github.io/anytty/zh-CN/)与[变更记录](docs/zh-CN/CHANGELOG.md)
- [贡献指南](docs/zh-CN/CONTRIBUTING.md)、[治理](docs/zh-CN/GOVERNANCE.md)与[行为准则](docs/zh-CN/CODE_OF_CONDUCT.md)
- [安全策略](docs/zh-CN/SECURITY.md)、[支持](docs/zh-CN/SUPPORT.md)与[私密漏洞报告](https://github.com/anytty/anytty/security/advisories/new)
- [Apache-2.0 许可证](LICENSE)、[NOTICE](NOTICE)、[第三方声明](THIRD_PARTY_NOTICES.txt)与[商标政策](docs/zh-CN/TRADEMARKS.md)

贡献须遵守 [DCO](DCO)。Apache-2.0 不授予 AnyTTY 名称和图标的商标权。
