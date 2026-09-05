# Changelog

English | [简体中文](docs/zh-CN/CHANGELOG.md)

This file records user-visible changes. The current version is a prerelease and does not promise stable upgrades. See the [security boundary](docs/SECURITY_BOUNDARY.md) for security guarantees and the [documentation index](docs/README.md) for usage guides.

## Unreleased

## [0.0.1-beta.12] - 2026-09-05

### Changed

- Unified the release metadata used by the CLI, web workspace, Flutter client, and GitHub release workflow.
- Aligned the mobile build number at `17` for the Android and iOS beta artifacts.

## [0.0.1-beta.11] - 2026-09-05

### Fixed

- Android terminal input no longer advertises password-like input metadata, preventing Oppo/ColorOS devices from opening the secure keyboard for normal terminal input.

## [0.0.1-beta.10] - 2026-09-04

### Added

- Added a native browser surface for remote web sessions with isolated proxy routing, persistent session state, multi-tab snapshots, history suggestions, reader mode, desktop-site mode, and offline restore messaging.
- Added browser entry points from endpoint and terminal views, including a shared browser instance and consistent app/system back navigation.

### Changed

- Integrated browser navigation into a single compact toolbar with mobile tab switching and the existing terminal petal menu.
- Android debug and release signing configurations now enable both V1 and V2 APK signature schemes for broader OEM installer compatibility.

### Fixed

- Terminal links now open inside AnyTTY's browser while preserving `mailto:` and `tel:` external handling.
- Browser session routing no longer creates a second workspace when entering from a terminal, and leaving the browser keeps its live WebView state in memory.

## [0.0.1-beta.9] - 2026-09-02

### Changed

- macOS CLI release archives are now built on macOS and carry an explicit code signature instead of the Go linker's transient signature.

### Fixed

- The macOS installer and self-updater now apply and verify a local code signature before atomically replacing the installed executable, preventing Apple System Policy from terminating a freshly installed CLI.
- Update checks now authenticate with `GH_TOKEN`, `GITHUB_TOKEN`, or an existing authenticated GitHub CLI session, avoiding the much lower shared-IP limit for anonymous GitHub API requests.
- Update requests now retry bounded transient connection failures, HTTP 429 responses, and server errors from GitHub and its Release asset CDN.

## [0.0.1-beta.8] - 2026-09-02

### Added

- Added a Windows CI test that verifies release archive installation, checksum validation, recommended configuration installation, and preservation of existing configuration.

### Changed

- Updated the documented Windows installation command to work when the default PowerShell execution policy blocks downloaded scripts, without changing the user's persistent policy.

### Fixed

- Windows daemons now recognize Winsock `WSAECONNREFUSED` when probing an abandoned AF_UNIX socket and replace the stale socket instead of failing to start.

## [0.0.1-beta.7] - 2026-09-02

### Added

- Added daemon resource sampling settings for polling interval and per-terminal retained sample count through YAML, environment variables, and CLI configuration.

### Changed

- Zoomed TUI panes now occupy the complete terminal viewport without the header, footer, pane borders, overflow markers, or hidden chrome hit targets.
- The terminal picker now opens on running terminals by default and orders its filters as Running, Exited, and All.
- Updated the native mobile screenshots in the repository documentation.

### Fixed

- Resource sampling now uses a bounded chronological ring buffer and preserves the daemon-provided history window in the TUI instead of truncating it to 64 points.

### Security

- Updated Browserslist to 4.28.8 to address unbounded cache growth and unsafe custom-stat normalization.
- Updated gRPC-Go to 1.83.1 to prevent heap exhaustion from fragmented HTTP/2 DATA frames.

## [0.0.1-beta.6] - 2026-09-01

### Added

- Added a native Flutter client for Android and iOS, backed directly by the AnyTTY Go client engine and pinned `libghostty-vt` terminal input adapter.
- Added searchable endpoint and terminal pickers, native terminal history and file workflows, configurable quick keys, touch petal menus, appearance controls, and route management to the mobile client.
- Added automatic installation of the recommended `coralline-candy` TUI profile on first CLI install while preserving existing user configuration.

### Changed

- Replaced the Capacitor/WebView mobile application with the Flutter client while retaining the `com.anytty.app` Android package identity and release signing path.
- Made endpoint route scoring preserve device labels, made Cloud route selection measurement-driven, and automated client certificate renewal.
- Updated Android and iOS launcher assets to the AnyTTY application icon, including Android adaptive and round icons.

### Fixed

- Improved endpoint connectivity recovery and made endpoint state projections consistent across terminal pickers and device lists.
- Corrected Flutter release version metadata so beta.6 Android packages upgrade monotonically from beta.5.
- Corrected the iOS CI build order so required native XCFrameworks are generated before the Flutter simulator build.

### Upgrade notes

- The Flutter client uses a new local registry and secure credential store. Android users upgrading from the Capacitor beta must pair their endpoints again; uninstalling first is not required when the existing app was signed with the official release key.

## [0.0.1-beta.5] - 2026-08-26

### Added

- Added running, exited, all, and public-tag filters to terminal inventories on mobile and Web surfaces.
- Added explicit end-running-terminal and remove-exited-record actions with separate confirmation and history-preservation behavior.
- Added bounded, redacted Android diagnostic bundles that are created and shared only after an explicit user action.
- Added Android ARMv7 (`armeabi-v7a`) native builds for supported 32-bit devices.
- Added signed per-ABI Android APKs, a signed universal APK, and a signed Google Play AAB to the tag-driven release workflow.

### Changed

- Foreground recovery now records structured stage diagnostics and lets healthy endpoints resume without waiting for an offline endpoint to become a global app failure.
- Terminal pickers now expose result counts, keep keyboard selection visible in long lists, and render positional terminal tags as user-facing values instead of protocol keys.
- Expanded the bundled AnyTTY terminal skill with endpoint-scoped remote terminal and file workflows plus clearer authority boundaries.
- Added stable human-readable Cloud operator projections alongside numeric protocol values.

### Fixed

- Replaced the Android white screen on unsupported system WebViews with a native compatibility screen that reports the installed WebView and the Chromium 101 minimum.
- Kept terminal rendering and input gated during native generation recovery so stale sessions are not mounted while reconnecting.
- Corrected Android native recovery logging so exported diagnostics remain structured and do not include endpoint identifiers or service addresses.

## [0.0.1-beta.4] - 2026-08-26

### Fixed

- Cloud daemon admission now carries a structured daemon-limit failure, and `anytty cloud status` reports an actionable `quota_limited` state instead of waiting indefinitely for Edge readiness.

## [0.0.1-beta.3] - 2026-08-26

### Added

- Added structured Cloud subscription failure details so clients can distinguish inactive subscriptions, unavailable regions, Relay plan restrictions, and service failures.

### Changed

- Desktop, mobile, CLI, and TUI clients now present subscription-specific recovery guidance instead of a generic connection failure.
- Moved the product website and user-facing documentation to the dedicated `anytty/anytty-site` repository and updated documentation links to `anytty.com`.

### Removed

- Removed the duplicate Astro website, website npm workspace, and GitHub Pages deployment from the public source repository.

## [0.0.1-beta.2] - 2026-08-25

### Added

- Added a desktop-oriented local Web workbench with tabs, recursive multi-pane terminal layouts, drag previews, and the `Ctrl+F` terminal picker.
- Added optional password protection for externally proxied Local Web access, including Argon2id password verification, rate-limited login, and bounded browser sessions.

### Changed

- Local Web now uses a same-origin bridge so an HTTPS reverse proxy can reach the local daemon without exposing machine enrollment or pairing controls in the browser.
- Web tabs preserve independent pane layouts, and terminal drops split only the targeted pane on desktop and mobile layouts.

### Security

- Password-protected Local Web requires a secure public origin; unprotected mode remains restricted to loopback hosts.

## [0.0.1-beta.1] - 2026-08-24

### Added

- Added `anytty web`, a loopback-only local browser interface that reuses the shared terminal and file UI without restarting the daemon or terminal pool.
- Added `anytty update` to check and install verified archives from the public `anytty/anytty` GitHub Releases without restarting a running daemon.
- Added tag-driven release automation in the public source repository for CLI archives, the unsigned Android APK, checksums, build metadata, and provenance attestations.

### Changed

- Made `anytty/anytty` the sole source for future installers and release assets; the former `anytty-site` release remains only as a legacy compatibility location.

## [0.0.1-beta.0] - 2026-08-17

### Added

- Added a bilingual Markdown documentation site, GitHub Pages Actions, local build previews, and configuration for a future `anytty.com` custom domain.
- Added community governance documents, issue and pull request templates, CODEOWNERS, Dependabot, and a release checklist for the Apache-2.0 project.
- Added automated checks for HTML and Markdown links, bilingual page structure, sensitive paths, private imports, and potential credential files.
- Added Local, SSH, Direct, and Cloud routes to the endpoint registry, with tests and diagnostic commands.
- Added one-time QR pairing. The Android app adds devices only through explicit pairing and does not log in or auto-discover account devices.
- Added baseline-driven Full/Delta terminal updates, paginated history, search, range copy, and continuous switching between Live and History views.
- Added optional AnyTTY Cloud connectivity with managed device discovery, P2P negotiation, Relay fallback, and connection-path refresh without restarting the daemon.
- Added searchable Cloud product documentation with responsive navigation.
- Added complete project READMEs, stable topic guides, and repository documentation indexes.

### Changed

- PTY output now uses one bounded payload per terminal, consumed by independent Live and History cursors. Overflow behavior can be configured as `block` or `drop`.
- The TUI and mobile clients reattach long polls immediately after submitting the current renderer batch instead of using a fixed frame-rate window. Updates arriving during rendering are merged into the latest damage instead of queuing stale frames.
- App backgrounding, WebView reloads, and native session generation changes cancel stale requests and restore sessions from the local endpoint registry.
- History mode freezes its visual anchor on entry, returns to Live automatically at the newest position, and materializes large copy ranges only when confirmed.
- Cloud routes change discovery and transport only; the daemon remains the authority for terminal and file permissions.
- The repository layout guard now checks stable documents, invalid output paths, and build artifacts without restricting additional Markdown files.

### Security

- Terminal and file permissions are enforced by the daemon through client-bound capability grants.
- Pairing claims are short-lived, one-time credentials; access grants are bound to a client identity and enforced by the daemon.
- Remote connections authenticate the daemon identity. Identity, authentication, or authorization failures are rejected without weakening checks through route fallback.
- Credentials use protected storage and atomic updates. Logs do not record secrets, terminal content, or file content.

### Removed

- Removed the assumption that account devices should be auto-discovered by the mobile app. Cloud accounts and the app endpoint registry remain independent.
- Removed duplicate TUI configuration templates, completed remediation plans, obsolete design drafts, and outdated development workflow documents.
- Removed compatibility promises for unreleased legacy protocols, YAML files, and development data formats.
