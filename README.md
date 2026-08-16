# AnyTTY

AnyTTY is an open-source remote terminal system for reaching terminals and files on your own machines. One `anytty` daemon owns the terminal processes, history, file access, device identity, and client authorization; the CLI/TUI and mobile apps reach it over Local, SSH, Direct WebRTC, or AnyTTY Cloud routes.

> **Project status:** pre-release and under active development. Beta artifacts are available for evaluation, but there are no stable releases, compatibility guarantees, or production-readiness claims. Expect protocols and configuration to change before the first stable release.

<p align="center">
  <img src="docs/assets/android-pairing.png" alt="AnyTTY Android device screen with the Scan service QR action" width="300">
</p>

## What is included

- The `anytty` CLI, TUI, and per-user daemon.
- Shared React terminal and file UI, plus Android and iOS app source.
- Local socket, OpenSSH, Direct WebRTC/ICE-TCP, and AnyTTY Cloud client routes.
- QR pairing, client-bound capability grants, end-to-end authentication, and public protobuf protocols.
- The Cloud daemon agent and the client support required to connect to the official AnyTTY Cloud.

Terminal sessions provide live output, retained history, attach/detach lifecycle, resize and input handling. File operations are authorized by the daemon and exposed through the same endpoint model. See the [architecture](ARCHITECTURE.md) and [terminal delivery model](docs/TERMINAL_DELIVERY.md).

## Connection routes

| Route | Transport | Cloud required | Intended use |
| --- | --- | --- | --- |
| Local | Current-user socket | No | Same-machine CLI and TUI |
| SSH | OpenSSH tunnel plus loopback signaling/ICE-TCP | No | Machines already reachable over SSH |
| Direct | Pinned daemon identity plus WebRTC/ICE-TCP | No | Self-managed network reachability |
| Cloud | Edge signaling, WebRTC P2P when possible, Relay when needed | Yes | Reachability through the official managed service |

Every route terminates at the daemon. Choosing Cloud changes discovery and transport, not who may grant terminal or file access.

## Architecture and security boundary

```text
CLI / TUI / Mobile
        |
        | Local, SSH, Direct, or Cloud transport
        v
  user-owned daemon  ----> PTY / shell
        |                    terminal history
        +------------------> authorized file roots

Cloud route only:
client <-> managed Edge (signaling / relay) <-> daemon Cloud agent
```

The daemon is the final authority for device identity, terminal access, file access, and revocation. Managed Cloud services handle account policy, daemon registration, signaling, route selection, and relay transport; they cannot mint daemon terminal or file capabilities. Pairing claims are one-time and client-bound. Read [ARCHITECTURE.md](ARCHITECTURE.md), the [pairing protocol](docs/PAIRING_PROTOCOL.md), and the [open-source boundary](docs/OPEN_SOURCE_BOUNDARY.md) before deploying or modifying a trust boundary.

## Build from source

CLI/TUI development requires Go 1.26.5. Shared UI and mobile builds use Node.js 24 and the checked-in npm lockfile.

```sh
git clone https://github.com/anytty/anytty.git
cd anytty
npm ci
make build
```

The binary is written to `.artifacts/bin/anytty`. Run the main checks with:

```sh
make test
make test-clients
npm run public:check
```

Android release validation additionally needs Java 21 and the Android SDK; run `make test-android`. iOS builds require macOS, Xcode, and the prerequisites in [clients/mobile/ios/README.md](clients/mobile/ios/README.md). Pre-release CLI archives and the unsigned Android verification APK are available on [GitHub Releases](https://github.com/anytty/anytty/releases).

## Quick start

Start the current user's daemon, then open the TUI:

```sh
./.artifacts/bin/anytty daemon start
./.artifacts/bin/anytty
```

Create and attach to a terminal from the CLI:

```sh
./.artifacts/bin/anytty terminal create --attach -- zsh
```

Use `./.artifacts/bin/anytty --help` and the [quick-start guide](https://anytty.github.io/anytty/docs/quick-start.html) for the commands implemented by the current checkout. Mobile clients add a daemon by scanning the one-time service QR code; they do not sign in or discover account devices automatically.

## Cloud and self-hosted operation

Local, SSH, and Direct routes are fully represented in this Apache-2.0 repository and do not require AnyTTY Cloud. You provide host access, network reachability, certificates, and operations for those routes.

The official AnyTTY Cloud is a managed service. This repository includes its clients, daemon agent, public protocol, and end-to-end authorization path, but not the proprietary Controller, Edge implementation, Cloud Web application, billing, database migrations, production deployment, or operational configuration. Those server components are not currently offered as a self-hostable Cloud stack. No pricing or service availability is promised here.

## Project resources

- [Documentation site](https://anytty.github.io/anytty/) and [documentation source](site/README.md)
- [Contributing](CONTRIBUTING.md), [governance](GOVERNANCE.md), and [code of conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md) and [private vulnerability reporting](https://github.com/anytty/anytty/security/advisories/new)
- [Support](SUPPORT.md), [changelog](CHANGELOG.md), and [release checklist](RELEASE_CHECKLIST.md)
- [Apache-2.0 license](LICENSE), [NOTICE](NOTICE), [third-party notices](THIRD_PARTY_NOTICES.txt), and [trademark policy](TRADEMARKS.md)

Contributions use the [Developer Certificate of Origin](DCO). AnyTTY names and logos are not granted by the Apache License 2.0.
