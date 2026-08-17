<div align="center">
  <img src="docs/assets/logo.png" alt="AnyTTY Logo" width="120">
  <h1>AnyTTY</h1>
  <p><strong>Keep terminals running. Take control from anywhere.</strong></p>
  <p>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-3d7ea6" alt="Apache-2.0 License"></a>
    <a href="https://github.com/anytty/anytty/releases"><img src="https://img.shields.io/github/v/release/anytty/anytty?include_prereleases&amp;sort=semver&amp;label=latest%20beta" alt="Latest Beta Release"></a>
    <a href="https://github.com/anytty/anytty/releases"><img src="https://img.shields.io/github/downloads/anytty/anytty/total?label=release%20downloads" alt="Total GitHub Release Downloads"></a>
    <a href="https://github.com/anytty/anytty/stargazers"><img src="https://img.shields.io/github/stars/anytty/anytty?style=flat&amp;label=stars" alt="GitHub Stars"></a>
    <a href="https://github.com/anytty/anytty/actions/workflows/ci.yml"><img src="https://github.com/anytty/anytty/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="docs/PACKAGE_MANAGERS.md"><img src="https://img.shields.io/badge/Homebrew-tap%20preparing-fbb040?logo=homebrew" alt="Homebrew Tap Preparing"></a>
  </p>
  <p>English · <a href="README.zh-CN.md">简体中文</a></p>
</div>

AnyTTY keeps terminal sessions running on your own machines and lets you return at any time through its keyboard-first TUI, CLI, or mobile app. The same connection also provides practical remote file browsing for project files, logs, and downloads.

> **Beta:** `v0.0.1-beta.0` provides macOS, Linux, Windows, and Android builds for evaluation. Protocols and configuration may still change before the first stable release.

<table>
  <tr>
    <td align="center"><img src="docs/assets/mac-tui.png" alt="AnyTTY multi-terminal TUI workspace on macOS" width="520"></td>
    <td align="center"><img src="docs/assets/win-tui.png" alt="AnyTTY multi-terminal TUI workspace on Windows" width="520"></td>
  </tr>
  <tr>
    <td align="center"><strong>macOS: splits, floating panels, and remote terminals</strong></td>
    <td align="center"><strong>Windows: local and remote terminals in one workspace</strong></td>
  </tr>
</table>

## Key features

- **Terminals and panels are decoupled.** A `terminal` is a task kept alive by the daemon; TUI `workspaces` and `panels` are only views used to observe and operate it. A panel neither owns nor stays bound to a terminal. The Terminal Picker can switch the current panel to any terminal without changing the existing split, floating-panel, or workspace layout. This differs from traditional terminal multiplexers such as tmux and Zellij, where a pane normally hosts a process directly.
- **Terminals and consumers are decoupled.** Interactive consumers such as TUI panels, the CLI, WebView, and mobile apps do not block the terminal output path. Each reads the latest screen snapshot at its own pace and uses an independent cursor to revisit history. With multiple consumers connected, a terminal never waits for every client to finish network transfer or rendering. A slow consumer may temporarily fall behind, but it can catch up from a newer snapshot without slowing the program running in the terminal or affecting other consumers.
- **Native Windows support.** Windows is a first-class platform alongside macOS and Linux. The CLI, TUI, and daemon ship for Windows x64 and ARM64, with no WSL dependency for core functionality.
- **Local and remote terminals share one workspace.** Terminals from Local, SSH, Direct, or Cloud endpoints all appear in the same Terminal Picker and can be viewed, switched, and operated like local terminals. AI agents running on remote workstations, servers, or build machines can become part of the same local workflow.
- **File-backed, theoretically unlimited history.** Terminal output is continuously written to history files while the live view retains only bounded data, so memory use does not grow linearly with accumulated history and remains approximately stable during long-running sessions. AnyTTY sets no fixed history line limit; practical capacity depends only on available disk space. As long as disk capacity can keep growing, history can keep growing too. You can quickly recover full context after a crash, a disconnected client, or a long unattended AI agent run.
- **Find terminals that stopped updating.** The mobile terminal list and the TUI Terminal Picker show status and recent activity together, making it easy to spot sessions with no new output and determine whether a task, service, or agent is still running.
- **Take control from different devices.** The same terminal can move between the TUI, CLI, embedded WebView, and Android or iOS mobile apps without migrating or restarting the process. The mobile experience is optimized for touch and terminal input, with extra keys for `Esc`, `Ctrl`, arrows, paging, and other common actions.
- **Optional AnyTTY Cloud connectivity for private networks.** The managed Cloud service helps daemons behind NAT or private networks establish P2P connections, with automatic Relay fallback when a direct path is unavailable. You do not need to expose the terminal service directly to the public internet.
- **Remote connections are encrypted by default.** SSH, Direct, and Cloud routes use authenticated, encrypted connections. Pairing access is bound to a specific client and can be revoked from the daemon. A Relay forwards encrypted traffic without gaining terminal or file permissions.
- **Terminals and files share one connection.** AnyTTY carries more than terminal screens: it also supports file browsing, upload, download, rename, and selection. After connecting to a private-network or remote endpoint, you can work with authorized files on that machine just as you work with its terminals.
- **Mobile file management and rich online previews.** The mobile app includes a file manager that browses remote directories and previews common text, document, image, media, and 3D formats. Logs, configuration, build results, and downloads can be inspected without switching apps.

### Mobile terminals

<table>
  <tr>
    <td align="center"><img src="docs/assets/app_1.jpg" alt="AnyTTY mobile terminal list with status and recent activity" width="230"></td>
    <td align="center"><img src="docs/assets/app_6.jpg" alt="AnyTTY mobile app highlighting terminals with recent output" width="230"></td>
    <td align="center"><img src="docs/assets/app_7.jpg" alt="AnyTTY mobile app controlling a remote terminal with the extra-key bar" width="230"></td>
  </tr>
  <tr>
    <td align="center"><strong>Multi-terminal status and recent activity</strong></td>
    <td align="center"><strong>Quickly find terminals with fresh output</strong></td>
    <td align="center"><strong>Take control with the extra-key bar</strong></td>
  </tr>
</table>

> **Install the app:** The mobile app supports both Android and iOS. The Android Beta APK is temporarily available from [GitHub Releases](https://github.com/anytty/anytty/releases/tag/v0.0.1-beta.0). A public iOS build is not yet available and must wait for the Apple App Store release. Google Play and Apple App Store publishing are both in progress.

### Connections and file management

<table>
  <tr>
    <td align="center"><img src="docs/assets/app_4.jpg" alt="AnyTTY mobile device and P2P connection status" width="260"></td>
    <td align="center"><img src="docs/assets/app_5.jpg" alt="AnyTTY mobile remote file manager" width="260"></td>
  </tr>
  <tr>
    <td align="center"><strong>Local, Cloud, and P2P paths</strong></td>
    <td align="center"><strong>Remote file management</strong></td>
  </tr>
</table>

### Rich online file previews

<table>
  <tr>
    <td align="center"><img src="docs/assets/app_2.jpg" alt="AnyTTY mobile app previewing a remote document" width="260"></td>
    <td align="center"><img src="docs/assets/app_3.jpg" alt="AnyTTY mobile app previewing a remote 3D model" width="260"></td>
  </tr>
  <tr>
    <td align="center"><strong>Document preview, search, and zoom</strong></td>
    <td align="center"><strong>Interactive 3D model preview</strong></td>
  </tr>
</table>

### Tablet multitasking and terminal apps

<table>
  <tr>
    <td align="center"><img src="docs/assets/pad.png" alt="Codex and OpenCode running in a vertical split on a tablet" width="360"></td>
    <td align="center"><img src="docs/assets/pad-ttt.png" alt="The TTT editor running inside a terminal on a tablet" width="360"></td>
  </tr>
  <tr>
    <td align="center"><strong>Codex and OpenCode in a vertical split</strong></td>
    <td align="center"><strong>TTT editor running inside the terminal</strong></td>
  </tr>
</table>

## Install

### macOS and Linux

The installer selects x64 or ARM64, downloads the matching GitHub Release archive, verifies `SHA256SUMS`, and installs to `~/.local/bin` by default.

```sh
curl -fsSL https://raw.githubusercontent.com/anytty/anytty/main/install.sh | sh
```

Choose another version or installation directory when needed:

```sh
curl -fsSL https://raw.githubusercontent.com/anytty/anytty/main/install.sh | \
  sh -s -- --version v0.0.1-beta.0 --bin-dir "$HOME/bin"
```

### Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/anytty/anytty/main/install.ps1 -OutFile install.ps1
.\install.ps1
```

The PowerShell installer verifies SHA-256, installs to `%LOCALAPPDATA%\Programs\AnyTTY\bin`, and adds that directory to the current user's `PATH`. Pass `-NoModifyPath` to leave `PATH` unchanged.

You can also download CLI archives and the unsigned Android Beta APK directly from [GitHub Releases](https://github.com/anytty/anytty/releases/tag/v0.0.1-beta.0). Package definitions for Homebrew, npm, and WinGet are being prepared; see [package manager publishing](docs/PACKAGE_MANAGERS.md) for their current status.

## Quick start

Start the daemon for your user and open the TUI:

```sh
anytty daemon start
anytty
```

Create a named terminal from the CLI and attach immediately:

```sh
anytty terminal create --name workspace --attach -- zsh
```

List sessions or return to one later:

```sh
anytty terminal list
anytty attach workspace
```

Run `anytty --help` for the complete command list, or continue with the [quick-start guide](https://anytty.github.io/anytty/docs/quick-start.html).

## Mobile app

Install the Android Beta APK from the release page, then choose **Add device** in the app. On the machine running the daemon, use `anytty pair create` to generate a short-lived pairing QR code or text claim, then scan or paste it in the app. Once paired, the device page lists running terminals and opens the file browser from a terminal's working directory.

The mobile app supports Android and iOS. The Android Beta APK is currently available from the release page; iOS does not yet have a public install and is waiting for Apple App Store publication. Official Google Play and Apple App Store releases are both in progress.

## Connection options

| Route | Best for | Cloud required |
| --- | --- | --- |
| Local | CLI and TUI on the same machine | No |
| SSH | Machines you already reach through OpenSSH | No |
| Direct | Self-managed LAN or publicly reachable environments | No |
| Cloud | Managed device discovery, P2P negotiation, and Relay fallback | Yes |

Local, SSH, and Direct work entirely from this repository. The official Cloud route is optional: it changes discovery and transport, not the daemon's final authority over terminals and files.

## Build from source

CLI/TUI development requires Go 1.26.5. Shared UI and mobile builds use Node.js 24 and the checked-in npm lockfile.

```sh
git clone https://github.com/anytty/anytty.git
cd anytty
npm ci
make build
```

The binary is written to `.artifacts/bin/anytty`. Run the main checks with `make test`, `make test-clients`, and `npm run public:check`. Android builds additionally require Java 21 and the Android SDK; iOS builds require macOS and Xcode.

## Open source and AnyTTY Cloud

The Apache-2.0 version includes the CLI, TUI, daemon, shared UI, Android and iOS source, plus Local, SSH, and Direct connection options. You can use it directly to manage local and remote terminals, files, and long-running tasks, and build clients for each platform.

On top of that, AnyTTY provides an official managed Cloud service for device discovery, P2P negotiation, and Relay fallback when a direct connection is unavailable, making terminals behind NAT or private networks easier to reach securely. Cloud is an additional ready-to-use service; terminal and file permissions remain under the user's daemon. See the [security boundary](docs/SECURITY_BOUNDARY.md) for the user-visible trust model.

## Project links

- [Documentation](https://anytty.github.io/anytty/) and [changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md), [governance](GOVERNANCE.md), and [code of conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md), [support](SUPPORT.md), and [private vulnerability reporting](https://github.com/anytty/anytty/security/advisories/new)
- [Apache-2.0 license](LICENSE), [NOTICE](NOTICE), [third-party notices](THIRD_PARTY_NOTICES.txt), and [trademark policy](TRADEMARKS.md)

Contributions use the [Developer Certificate of Origin](DCO). The Apache License 2.0 does not grant trademark rights to the AnyTTY name or logo.
