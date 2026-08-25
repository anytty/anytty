# Changelog

English | [简体中文](docs/zh-CN/CHANGELOG.md)

This file records user-visible changes. The current version is a prerelease and does not promise stable upgrades. See the [security boundary](docs/SECURITY_BOUNDARY.md) for security guarantees and the [documentation index](docs/README.md) for usage guides.

## Unreleased

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
