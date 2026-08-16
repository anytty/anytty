# AnyTTY

[English](README.md) | 简体中文

AnyTTY 让终端会话持续运行在你自己的机器上，并可随时从键盘优先的 TUI、CLI 或手机 App 回到现场。同一条连接还提供实用的远程文件浏览能力，适合查看项目文件、日志与下载内容。

> **Beta：** `v0.0.1-beta.0` 已提供 macOS、Linux、Windows 和 Android 构建，可用于体验和评估；首个稳定版发布前，协议与配置仍可能变化。

![展示工作区导航与双终端分屏的 AnyTTY TUI](docs/assets/tui-workbench.png)

## 主要功能

- **让工作持续运行。** shell、构建任务、开发服务器或日志追踪只需启动一次；断开后重新连接，无需重启进程。
- **管理多个终端。** 使用工作区、标签页、分屏、浮动窗、搜索、缩放和键盘优先的终端管理器组织会话。
- **从手机操作终端。** 连接实时终端，并通过专用扩展键盘快速输入 `Esc`、`Ctrl`、方向键、翻页键等常用终端按键。
- **浏览远程文件。** 在移动端访问已授权路径，预览常见文档与媒体格式，并执行上传、下载、重命名和多选操作。
- **自由选择连接方式。** 可只在本机使用、复用 SSH、通过 WebRTC/ICE-TCP 直连，也可在网络不方便时使用可选的 AnyTTY Cloud 路由。
- **无需共享密码即可配对。** 通过短时二维码或粘贴 claim 添加设备，访问权限与客户端绑定，并可由 daemon 撤销。

<table>
  <tr>
    <td align="center"><img src="docs/assets/mobile-terminal.png" alt="带扩展键盘的 AnyTTY Android 远程终端" width="360"></td>
    <td align="center"><img src="docs/assets/mobile-files.png" alt="AnyTTY Android 远程文件浏览器" width="360"></td>
  </tr>
  <tr>
    <td align="center"><strong>实时终端</strong></td>
    <td align="center"><strong>远程文件</strong></td>
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

仓库同时包含 iOS 客户端源码。目前还没有经过应用商店签名的 Android 或 iOS 安装包。

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

## 开源边界

Apache-2.0 仓库包含 CLI、TUI、用户自己的 daemon、共享 UI、Android 与 iOS 源码、Local/SSH/Direct 路由、Cloud 客户端、公开协议以及端到端授权路径。

仓库不包含专有的 AnyTTY Cloud Controller、Edge 服务端实现、Cloud Web、计费、数据库迁移、生产部署与线上运营配置；这些托管服务端组件目前不作为可自托管 Cloud 栈提供。详细内容见[开源边界](docs/OPEN_SOURCE_BOUNDARY.md)、[安全架构](ARCHITECTURE.md)与[配对协议](docs/PAIRING_PROTOCOL.md)。

## 项目入口

- [中文文档](https://anytty.github.io/anytty/zh-CN/)与[变更记录](CHANGELOG.md)
- [贡献指南](docs/zh-CN/CONTRIBUTING.md)、[治理](docs/zh-CN/GOVERNANCE.md)与[行为准则](docs/zh-CN/CODE_OF_CONDUCT.md)
- [安全策略](docs/zh-CN/SECURITY.md)、[支持](docs/zh-CN/SUPPORT.md)与[私密漏洞报告](https://github.com/anytty/anytty/security/advisories/new)
- [Apache-2.0 许可证](LICENSE)、[NOTICE](NOTICE)、[第三方声明](THIRD_PARTY_NOTICES.txt)与[商标政策](docs/zh-CN/TRADEMARKS.md)

贡献须遵守 [DCO](DCO)。Apache-2.0 不授予 AnyTTY 名称和图标的商标权。
