# Flutter Native App Migration

Status: migration in progress; the iOS and Android terminal vertical slice is
runnable.

This document is the feature contract for replacing the Capacitor mobile app
with a Flutter application backed by Go and libghostty-vt. The existing
TypeScript clients remain the Web implementation and the behavioral reference
during migration. They are not a source dependency of the native app.

The first supported targets are iOS and Android. The boundaries must remain
portable to macOS, Linux, and Windows, but desktop packaging and desktop-only
workflows are not part of the first delivery.

### Implementation checkpoint (2026-08-31)

- Flutter owns a native device list, QR/paste pairing flow, terminal inventory,
  active terminal route, direct canonical-cell painter, IME/hardware input, a
  libghostty-vt input adapter, and frozen-history paging.
- The Go engine runs behind FFI with event and platform-request pumps in worker
  isolates. Android packages the Go and libghostty adapters for arm64-v8a,
  armeabi-v7a, and x86_64.
- Binding operations, session-close events, and application events are accepted
  only when their full endpoint/route/generation stamp matches the active
  Flutter session. The background notification path independently tracks the
  newest successful session generation, so a late event from a replaced
  session cannot create a user-visible side effect.
- Endpoint supervisor demand is bound to one Flutter attachment identity with
  monotonic revisions; a foreign attachment cannot clear or replace the active
  snapshot even with a larger revision. Operation, session, and stream handles
  remain engine-owned, monotonic, and non-reusable, while stream records retain
  their exact session and renderer ownership until release.
- Full snapshots and revision-fenced deltas merge atomically. A rejected delta
  retains the previous valid frame while requesting a full replacement.
- History uses a separately frozen token/generation, prepends older rows at the
  head, and restores the viewport after insertion. It never substitutes live
  revision or connection generation for history identity.
- Flutter reflows width-independent logical history rows at measured local
  columns, preserving grapheme/wide-cell identity, soft-wrap coordinates,
  styled blank footprints, and fixed-grid frame geometry. A column change opens
  a replacement frozen token/generation identity, fences late responses,
  releases the previous token only after replacement acceptance, and restores
  the logical viewport anchor with virtual trailing rows where the frozen
  screen requires them.
- A frozen current alternate-screen frame uses the same bounded history pager
  as primary history. Flutter retains its segment, physical screen-row order,
  source grid width, and fixed-grid identity; older requests carry the
  alternate cursor unchanged, while the frame remains horizontally inspectable
  when it is wider than the local viewport.
- Memory remains bounded across the native pipeline. Go caps retained PTY
  output at 32 MiB per terminal and 512 MiB in aggregate, binding payloads at
  4 MiB, event/platform queues at 256/64 messages, history pages at 512 logical
  lines and 60 KiB, and file transfer credit at 1 MiB. Flutter admits one
  platform request at a time, caps pending terminal input at 128 operations and
  1 MiB, caps pending resizes at four, retains at most 8192 frozen-history
  projection rows, and publishes only the newest screen on its render timer.
- Runtime diagnostics stay in a 64-entry in-memory lifecycle ring and retain
  only aggregate render latency, output-loss/parser-epoch counters, and current
  operation/session/stream counts. The redacted JSON report exposes generation
  and supervisor attempt state but replaces endpoint/route identity and omits
  addresses, paths, terminal content, candidate IDs, and raw error messages. It
  reaches the clipboard only through the connection screen's explicit action.
- File previews resolve complete symlink chains in Go and fail on loops, reject
  unsupported MIME types and source files above 64 MiB, and return at most 4
  MiB. Raster images must fit in one complete response. Every accepted payload
  carries a SHA-256 digest that Flutter verifies before rendering; image decode
  dimensions are capped at 2048 pixels, and an offline loading test keeps the
  48dp Close action immediately available.
- Startup failures render only a classified user-safe state. Retry disposes any
  partially initialized lifecycle/background/native owners and constructs a
  fresh runtime; an eight-event startup report contains only stage, attempt,
  timestamp, and allowlisted failure category. Copy is explicit, while clearing
  the local device registry requires confirmation and never mutates a daemon.
- Frozen-history selection emits logical line/display-column ranges. Copy is
  streamed by Go in bounded chunks, while text/glob/regex search supports
  previous/next, wrap state, bounded match scans, and direct cell highlights.
- Terminal create/rename/restart/end/remove commands run through the Go engine.
  Resize uses a unique surface/view attachment, owner epochs, lifecycle
  projections, laid-out Flutter cell geometry, and explicit owner/lock/release
  controls. Followers never send automatic fit requests.
- Terminal inventory supports running/exited/all status filters, intersected
  public-tag filters, resource/activity metadata, and endpoint-scoped local pin
  order. Exited terminals bind frozen history without opening a live attachment;
  a live surface that exits transitions to the same read-only history path.
- Mobile input includes Shift/Ctrl/Alt off/once/locked latches, navigation and
  editing keys, F1-F12, and common control chords through libghostty-vt. The
  Web keybar's 400ms soft-keyboard focus lock and command-matched Claude Code /
  OpenCode shortcut groups are preserved. Per-terminal Auto/Resize/Shift modes
  either resize the authoritative PTY viewport or retain its closed-keyboard
  height and shift only enough to expose the cursor. Canvas link hit testing
  uses canonical cell widths and confirms allowlisted external schemes before
  opening another application.
- Primary-screen drag enters frozen history. Alternate-screen drag emits bounded
  bidirectional libghostty mouse-wheel reports, or application-mode cursor keys
  when alternate scroll is active without mouse tracking. Mouse geometry uses
  the same fractional cell metrics as painting.
- Terminal appearance is persisted independently from the app theme. The native
  painter supports all 17 Web terminal palettes and all six Web font choices:
  JetBrains Mono, Fira Code, Cascadia Code, Hack, Iosevka, and System Mono. The
  redistributed Nerd Font subsets retain their repository OFL notice. Settings
  provide in-family samples and real palette swatches before selection, plus a
  live terminal preview. Font changes update
  painter, hit testing, mouse encoding, and trusted viewport fit from one metrics
  object without recreating the connection.
- The in-terminal quick settings surface uses the same six-font contract as the
  full settings route, renders every menu choice in its actual family, and keeps
  a compact terminal preview synchronized with font, size, palette, and cursor
  changes. Android runtime verification caught and fixed an outdated two-item
  dropdown that crashed when the persisted font was Fira Code.
- App appearance now persists explicit dark/light selection under the same
  storage contract as Web, defaults to dark, and maps the existing zinc-neutral
  light and dark tokens into Flutter. A third system mode follows platform
  brightness without coupling the terminal palette to the app chrome.
  Status/navigation bars follow the resolved app mode.
- The native client has no xterm-style local scrollback row budget. History
  capacity and retention are server concerns; Flutter reads bounded frozen
  windows through the API and configures only the next-page prefetch distance.
- Active endpoint sessions retain a reference-counted, revisioned Go supervisor
  demand snapshot. Network changes are wake-up hints only. Foreground resume
  waits through the 1.2-second native recovery window, preserves the last frame,
  pauses input, requests `full_replace`, and releases stale frozen-history
  tokens before opening a new API window.
- A fixed terminal-tools entry keeps keyboard, function keys, paste, history,
  search, selection, resize ownership, connection size/state, and terminal
  settings/app appearance reachable without scrolling the quick-key strip. The active header
  reads the authoritative terminal name/live directory and exposes lifecycle
  actions without leaving the terminal route.
- Live input is enabled only after Flutter acknowledges presentation of the
  first canonical revision. A two-second unpresented-revision watchdog pauses
  input, retains the last painted frame, and exposes scoped reconnect instead of
  replacing terminal content with an empty error page. Ordinary recovery stays
  visually quiet for 1.2 seconds, then adds one non-blocking status over the
  retained frame; recovery success cancels that delayed state immediately.
- Client access identities support prepare, resolve, bind, sign, and delete via
  secure storage. The current Flutter implementation still materializes the
  Ed25519 seed in Dart while signing; moving signing behind native
  Keychain/Keystore handles remains a release gate.
- Android 15 emulator cold start, pairing sheet, camera permission/scanner, and
  native library loading are verified. The arm64 iOS simulator build and cold
  start, the unsigned arm64 device build, Go engine symbols, and libghostty-vt
  adapter symbols are also verified with Xcode 26.3. Repository build scripts
  select the full Xcode installation without changing the host's global
  `xcode-select` setting.
- File transfers use Go-owned resource handles and bounded data/ack frames.
  Downloads are written to atomic partial files, uploads stream from local
  files, and both directions verify size and SHA-256 before completion. Android
  system picker upload and native streamed export have passed emulator tests;
  the iOS document exporter compiles in the arm64 simulator build. Download
  acknowledgements replenish the daemon-advertised byte window instead of
  enqueueing one ACK per chunk; a 512 MiB Android transfer stayed active after
  the file route closed and reached the native destination picker. Failed
  downloads retry in place; Android verified a missing-source failure followed
  by successful retry, native save, and matching SHA-256. Download offsets and
  daemon-owned upload resume handles preserve nonzero partial progress; Android
  verified a paused 128 MiB download and paused 512 MiB upload with matching
  final SHA-256. Picker-owned upload cache copies are released after completion.
  The transfer center now follows the Web reference's full-screen layout,
  selection/bulk controls, layered Back behavior, MIME projection, and native
  open action. Android opened a verified text download in the system viewer.
- Terminal management, selection/copy/search, resize ownership, split-pane
  workflows, terminal appearance, file management, and transfers are in
  progress. Durable process-restart transfer recovery, localization, local
  discovery physical-device permission verification, iOS file-flow runtime
  verification, accessibility, and release
  hardening remain pending.

## 1. Product boundaries

| Layer | Owns | Must not own |
| --- | --- | --- |
| Flutter | Navigation, application UI, terminal composition, canonical live-screen merge cache, cell drawing, touch/keyboard adaptation, accessibility, localization, and platform-neutral presentation state | Connection retry policy, endpoint/session truth, authoritative history, credentials, or background networking |
| libghostty-vt | Key, mouse, focus, and paste encoding plus terminal-oriented Unicode utilities behind an AnyTTY C ABI | Canonical live-screen or history truth, tabs, splits, application navigation, remote sessions, notifications, or file management |
| Go Client Engine | Endpoint registry, pairing, credentials through platform ports, route planning, physical sessions, terminal/file commands, resource streams, lifecycle fencing, and background connection state | Flutter widgets, terminal drawing, OS permission prompts, or system UI |
| iOS/Android host | Secure storage primitives, notifications, background execution, network/power hints, file picker/export, status bars, haptics, camera permission, and app lifecycle | Business retry policy, terminal state, endpoint selection, or duplicate session state |

Flutter is the composition and presentation bridge. Platform messages use
serialized Proto payloads through narrow FFI/plugin APIs. Business models are
not mirrored into ad-hoc JSON contracts.

## 2. libghostty selection and constraints

The native app targets **libghostty-vt**, not Ghostty's internal embedder API.
As of 2026-08-30, Ghostty documents `include/ghostty.h` as an internal API whose
only supported consumer is the macOS app. It exposes iOS-specific embedding but
does not provide a supported Android surface. `include/ghostty/vt.h` is the
external C API intended for cross-platform embedders.

libghostty-vt provides terminal state APIs, but no windowing or drawing
implementation. It also cannot import AnyTTY's `NativeScreenResult` cell
matrix as terminal state. The daemon's canonical screen projection is already
parsed and carries richer AnyTTY revision and history boundaries, so the native
app must not encode it back to ANSI merely to parse it again.

Flutter therefore draws the canonical AnyTTY cells directly. libghostty-vt is
used for terminal input protocol behavior and Unicode utilities, configured
from the authoritative `TerminalModes` projection. A future raw-PTY mode may
use its parser and render state, but that mode cannot replace the snapshot path
without an atomic, sequence-fenced bootstrap contract.

The libghostty-vt API is explicitly work in progress. The repository must pin a
tested Ghostty commit, wrap it behind an AnyTTY-owned C ABI, record its license,
and upgrade it deliberately. Dart must never bind directly to the upstream ABI.

Official references:

- <https://github.com/ghostty-org/ghostty/blob/main/include/ghostty/vt.h>
- <https://github.com/ghostty-org/ghostty/blob/main/include/ghostty.h>
- <https://github.com/ghostty-org/ghostling>

## 3. Terminal feature contract

Every row below is required unless explicitly marked Web-only. `In progress`
means an Android implementation or partial behavior exists but has not passed
the complete iOS/Android definition of done. `Pending` means the behavior
exists in the old client or protocol but has not yet been implemented in
Flutter.

### 3.1 Inventory and lifecycle

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Terminal inventory | List created, running, exited, and removed terminals with name, command, current directory, foreground process, timestamps, attachment count, output activity, and public tags. Current running-terminal CPU/memory totals, reporting coverage, per-terminal compact trends, and bounded on-demand history are rendered from the daemon snapshot. The visible foreground inventory refreshes every two seconds without overlapping requests or polling hidden routes. Android and iPhone simulator layouts are verified; an event-driven resource stream remains. |
| In progress | Inventory filtering | Filter by running/exited state and exact public tag values; daemon-owned metadata stays hidden. |
| In progress | Pin and reorder | Pin terminals locally and reorder pinned entries without changing daemon terminal state. |
| In progress | Create | Load daemon defaults; edit name, shell-safe command arguments, working directory, environment, initial size, and size lock. Validate unfinished quoting and invalid/duplicate environment keys. |
| In progress | Edit metadata | Update name and public tags without restarting the process. |
| In progress | Restart | Restart an exited or running terminal and surface the new authoritative generation. |
| In progress | End | Confirm before stopping a running process; keep its record and saved history. |
| In progress | Remove | Confirm permanent removal of an exited record and its saved history. |
| In progress | Exited history | Keep an already displayed exited terminal mounted as read-only history and allow opening an exited terminal directly in history mode. |
| In progress | Cross-device switcher | Group saved devices, preload only the current inventory, lazily connect when another group expands, cache successful loads, expose retry locally, preserve per-device pin ordering, and route directly to the selected terminal. Automated multi-device behavior and Android single-device layout are verified; real multi-device and iOS runtime verification remain. |

### 3.2 Session and live rendering

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Attach/detach | Attach as collaborator or observer with a unique surface/view identity; detach and release all resources exactly once. |
| In progress | Canonical bootstrap | Start with `full_replace`; publish a screen only after every row, cursor, mode, size, alternate-screen flag, and revision belongs to the same response. |
| In progress | Incremental merge | Accept a delta only when terminal identity, connection generation, dimensions, and `base_revision` match the current cache. Evaluate every row copy from the unchanged base, then apply row replacements and publish once. |
| In progress | Full recovery | On generation change, invalid ranges, missing base, revision mismatch, or malformed cells, keep the previous valid frame visible and request `full_replace`. Never partially publish a rejected delta. |
| In progress | Render scheduling | Allow only one Flutter render submission at a time and coalesce pending damage to the newest state. High-rate output must not create an unbounded Dart queue. |
| In progress | Screen state | Preserve cursor position/shape/blink, primary and alternate screens, 24-bit/256 colors, bold, italic, underline, reverse, strike, links, wide characters, grapheme clusters, and tail-fill styling. |
| In progress | Terminal modes | Treat `TerminalModes` as authoritative for alternate screen/scroll, mouse tracking, bracketed paste, application cursor, and auto-wrap. Configure libghostty-vt input encoders from that projection instead of maintaining a second parsed terminal. |
| In progress | Renderer health | Detect a stalled render pipeline, stop accepting input when delivery is uncertain, and recover without displaying a blank surface. |
| In progress | Frame continuity | Keep the previous valid frame visible while a full replacement is being applied; do not flash white, recreate the whole page, or move surrounding controls. |

#### Required protocol work

The snapshot contract is the first mobile implementation. It has three
independent version domains that must never be substituted for one another:

1. Binding/session generation fences connection and handle lifetime.
2. `live_revision` and `base_revision` fence the latest-only live cell matrix.
3. Frozen-history `history_generation`, token, and cursor fence append-only
   logical history pages.

The raw PTY resource stream remains available for diagnostics and future local
parser experiments. It is not a display source until it gains an atomic VT
baseline plus parser epoch and byte sequence. Synthesizing ANSI from
`NativeScreenResult` is explicitly rejected because it loses canonical cell
identity and introduces a second parser state that can diverge from Go.

### 3.3 Input and mobile keyboard

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Ordered input | Wait for each terminal input acknowledgement before sending the next input. Never replay unacknowledged input after recovery. |
| In progress | IME and Unicode | Support composition, CJK input, multi-codepoint graphemes, emoji text input, and hardware keyboards without consuming composition events as shortcuts. Android Gboard text insertion and software Backspace are emulator-verified; deleting an empty sentinel is handled before stale IME composition ranges. |
| In progress | Modifier state | Ctrl, Alt, and Shift support `off`, one-shot, and locked states. One-shot state is consumed only after an accepted compatible key. |
| In progress | Key encoding | Encode Escape, Tab, Backspace, Delete, Home, End, arrows, Page Up/Down, F1-F24, navigation modifiers, application cursor keys, and Kitty keyboard events through libghostty-vt. |
| In progress | Quick key bar | Provide touch-sized common keys and symbols, keyboard show/hide, and a secondary Fn panel. Key layout is stable and safe-area aware. |
| Complete | Program presets | Select the Web-compatible Claude Code/OpenCode Fn groups from the terminal command while retaining the System process and line-editing groups. Submitted program commands close the panel. |
| Complete | Soft-keyboard modes | Persist per-terminal `auto`, `resize`, and `shift` values under the Web storage contract. The terminal canvas and quick-key bar share one IME-visible workspace and move together through one compositor translation. Auto shifts primary screens and resizes alternate screens; Shift retains the closed-keyboard grid. The translated child remains stable during native inset animation, live terminal painting is briefly frozen, and covered terminal-list routes stop building until they become current again. Android profile traces and screenshots verify the unified motion and visible cursor. |
| Complete | Focus lock | A 400ms keyboard-button press prevents soft-keyboard focus while every quick key and Fn shortcut remains sendable. A tap unlocks and opens the keyboard. |
| In progress | Paste safety | Read the system clipboard through the platform adapter, preserve bracketed paste, and require confirmation for large or multiline paste. Clipboard exceptions are localized. |
| In progress | Input pause | Immediately reject input while the session is recovering or delivery is uncertain. Show one scoped notice; do not queue or replay bytes. |
| Complete | Synchronized panes | Optionally send one input to all visible split targets. A successful target is never sent the same input twice when another target fails. |

### 3.4 Touch, mouse, selection, and scroll

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Primary-screen scroll | Pixel-accurate vertical touch scrolling with configurable inertia from 0 to 100 and reduced-motion support. |
| In progress | Alternate-screen scroll | Forward both touch directions to TUI applications using the active mouse/wheel protocol. Do not enter saved history while alternate screen is active. |
| In progress | Momentum cancellation | A new touch cancels the active inertia timer and prevents new reports from that gesture. Already acknowledged or queued input is never replayed; fully cancellable queued native reports still require an input-operation cancellation contract. |
| In progress | Scrollbar | Show a non-layout-shifting scrollbar with track jump and thumb drag. It reflects the currently loaded frozen-history range and expands while dragged. |
| In progress | Selection | Touch selection supports anchor, extension, auto-scroll near edges, cancel, select visible, and select all. Two-finger/system gestures must not corrupt selection state. |
| In progress | Copy | Copy by logical history range, not by scraping painted glyphs. Preserve newlines and wide-character boundaries and stream large copies in bounded chunks. |
| In progress | Links | Use canonical `ScreenCell.link_url` and `link_params`; open a link only after an explicit user action. |

### 3.5 History and search

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Frozen history | Request a latest window with token, generation, local columns, logical boundaries, and viewport anchor. Keep that generation stable while paging. |
| In progress | Lazy paging | Prefetch the first page without moving the live viewport; fetch older pages when the user reaches the threshold; coalesce repeated pulls. |
| In progress | Visual continuity | Preserve the same `logical_line_id + top_cell_offset` while prepending history. Return to live exactly once when the viewport reaches the bottom. |
| Complete | Local reflow | Reflow history using the local terminal columns, restart the frozen generation after a column change, and preserve soft-wrap and wide-character identity. |
| Complete | Alternate history | Load saved alternate-screen frames through the same bounded pager. |
| In progress | Position metadata | Display logical current/total position and timestamp without depending on painted row count. |
| In progress | Search | Support text, glob, and regex search; previous/next; wrap indication; bounded background scan; total count; current and secondary match highlights; and an overview rail. |
| In progress | Error recovery | Keep rendered content on a retryable history failure. Require explicit reload after stale token. Oversized logical lines are dismissible and not retried blindly. |
| In progress | Loading feedback | Keep fast history API requests silent and retain the rendered terminal; expose an accessible loading status only after two seconds. |
| In progress | Snapshot release | Release frozen history tokens on resume, terminal change, unmount, generation change, and process exit. |

### 3.6 Layout and controls

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Native phone navigation | Terminal list and active terminal are separate routes. Back closes the newest sheet/tool mode first, then returns to the list, then to the device list. |
| In progress | Split panes | Split left/right/above/below, target the active leaf, nest multiple splits, resize dividers, and close a leaf. The current-device terminal sheet activates an existing pane or replaces the primary pane without disturbing the split tree. In Android landscape keyboard focus mode, inactive panes stay mounted offstage while the active pane uses the available terminal height; closing the keyboard restores the full split tree and ratio. Directional placement of an arbitrary picker target and cross-device terminals remain. |
| Complete | Per-pane state | Each pane owns its attachment, history/search/selection state, keyboard target, and resize projection. Only the active pane accepts focused input unless synchronized input is explicitly enabled. |
| In progress | Unified tools | Copy, paste, selection, search, clipboard history, snippets, renderer/font controls, resize ownership, size lock, connection info, split actions, and sync-input stay reachable from one terminal tools surface. |
| In progress | Stable header | Show device, terminal name/current directory, Files, split, and tools without covering terminal content or the system safe area. Terminal lifecycle actions remain in the inventory row menu, matching the mobile reference. |
| Web-only | Browser tab workbench | Mouse drag/drop tab reordering, fuzzy keyboard picker, hover split preview, and desktop workbench tabs remain in TypeScript for Web. Native desktop can be designed later. |

### 3.7 Resize ownership

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Trusted fit | Acquire resize ownership only after Flutter has a current, laid-out viewport and measured cell geometry. Never send stale remote dimensions. |
| In progress | Owner/follower/observer | Present the authoritative `ResizeControl` reason and owner surface. Followers do not resize until ownership is acquired. |
| In progress | Lock | Size lock is independent from ownership. A locked follower can acquire ownership and then unlock. |
| In progress | Release | Explicitly release ownership and update every attached surface from lifecycle events. |
| In progress | Local errors | Resize-control failures remain local to terminal controls and are not reported as connection failures. |

### 3.8 Settings, appearance, and accessibility

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Terminal themes | Preserve AnyTTY Dark plus Tokyo Night, Dracula, One Dark, Catppuccin, Solarized, Nord, Gruvbox, GitHub, and light variants. App appearance remains independent from terminal colors. The picker previews real ANSI swatches, groups dark/light palettes, animates to the current choice with reduced-motion support, and updates the live terminal preview. |
| In progress | Font | Preserve the six Web choices using licensed mobile subsets: JetBrains Mono, Fira Code, Cascadia Code, Hack, Iosevka, and System Mono. The picker renders the same sample in every family before selection; family and size update the live terminal preview without losing session state. UI remains system sans. |
| In progress | Cursor and history behavior | Configure cursor blink, API page-prefetch distance, alternate-screen scroll inertia, keyboard mode, and automatic resize-owner acquisition. There is no client scrollback row budget. |
| In progress | Light/dark/system app theme | Use the native client's graphite/teal cross-platform tokens, synchronize status/navigation bars, and keep terminal theme independent. Explicit light/dark and automatic system modes persist through the same settings controller and are screenshot-verified on Android and iPhone simulators. Physical-device appearance transitions remain. |
| In progress | Accessibility | The live terminal is one focusable read-only element whose label includes the authoritative columns/rows and whose value contains bounded visible snapshot text: at most 32 latest nonblank rows and 4096 grapheme clusters, with no interrupting live-region announcements. Lazily built history rows expose bounded text and selected state independently. Flutter semantics tests cover spacing, complete graphemes, newest-output retention, bounds, conditional horizontal scrolling, and history selection. Android UIAutomator and iOS Accessibility runtime trees both expose the real daemon output as exactly one canvas-sized focus node without unnamed gesture/container stops; the hidden IME editor is excluded. Settings appearance choices are mutually exclusive named buttons with selected state, while font/theme preview entries and candidates expose concise labels and the current selection without repeating visual sample text. Keyboard mode combines its name/current choice, scroll inertia combines its name/value and adjustable actions, and every switch carries its visible name plus checked state in both native trees. Settings remain usable at 320 logical pixels with 200% text in widget tests, and Android portrait screenshots verify the real 200% appearance, terminal/font preview, and grouped theme palette picker without overflow. Reduced motion now prevents the settings preview cursor controller from starting, rather than merely hiding its visual output. A complete named-control/focus-order audit, hands-on VoiceOver/TalkBack reading, iOS largest Dynamic Type, contrast, and physical-device reduced-motion verification remain. |
| In progress | Touch targets | Interactive regions are at least 44x44pt on iOS and 48x48dp on Android with at least 8dp spacing where controls are adjacent. The fixed terminal key bar reserves platform-specific touch height with pixel-rounding headroom and larger labels; iPhone accessibility frames measure 44.5pt and Android frames exceed 48dp in both full and compact keyboard layouts. The Light/Dark/System settings control, font-size stepper, terminal inventory filters/resource summaries, terminal-switch title, tools actions, file breadcrumbs, transfer actions, connection diagnostics, route actions, pairing controls, Fn tabs, and Fn keys now reserve at least 48 logical pixels. Android 420dpi UIAutomator bounds verify exact 126px inventory segments and tool controls; focused widget regressions enforce the logical minimum. A complete iOS and remaining modal screen audit remains. |
| In progress | Orientation | Phone portrait and landscape remain operable; keyboard, split controls, and safe areas never occlude terminal text. The live terminal inventory switches to a two-column comparison layout at 720 logical pixels and the whole workspace respects lateral/bottom system insets. When the Android software keyboard opens in landscape, the app header yields and the two-row key bar becomes a one-row native-style accessory with 12 high-frequency controls, preserving roughly 94dp of terminal output instead of collapsing it to 8px. A split workspace temporarily focuses the active pane without detaching inactive panes; closing the keyboard restores the full tree and ratio. Real daemon snapshots are screenshot-verified in Android three-button landscape and iPhone Dynamic Island landscape. iOS software-keyboard and split-divider drag scenarios remain. |

### 3.9 Recovery and performance

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Flutter lifecycle | UI suspension does not own or terminate a demanded Go physical session. Endpoint demand is reference-counted in the runtime; resume preserves the frame, pauses input, releases frozen history, and requests a full authoritative projection. Platform background execution remains separate. |
| Complete | Generation fencing | Full session stamps fence Flutter operation results, close events, terminal/application projections, and background notifications. Go binds supervisor demand to one attachment with monotonic revisions, while engine-owned operation/session/stream handles are monotonic, renderer-scoped, session-bound, and never reused. Focused Flutter tests cover stale result/event/close rejection and notification replacement; Go tests cover foreign demand snapshots, late stream open, session close, renderer cleanup, and handle invalidation. |
| In progress | Endpoint recovery | Flutter publishes complete takeover demand snapshots and passes connectivity only as a wake-up hint. Go probes the retained winner before replacing it and owns dialing/backoff; typed recovery presentation remains incomplete. |
| In progress | Quiet recovery | Recovery under 1.2 seconds stays silent; longer recovery shows one scoped non-blocking state over the retained terminal frame, and recovery success cancels the delayed state immediately. Rendering stalls and explicit stream errors expose reconnect separately. Permanent authorization/configuration errors still need typed actions. |
| Complete | Bounded memory | Go bounds retained PTY output (32 MiB per terminal, 512 MiB aggregate), binding payloads (4 MiB), event/platform queues (256/64), history pages (512 logical lines, 60 KiB), and file transfer credit (1 MiB). Flutter's isolate pump admits one platform request at a time; terminal connections cap pending input at 128 operations/1 MiB, pending resizes at four, and frozen-history projection at 8192 rows. The canonical-screen poll keeps one request in flight and the 16ms publisher coalesces to the newest accepted frame; libghostty-vt is currently input-only and owns no mobile damage queue. Focused tests cover queue overflow/release and history truncation. |
| Complete | Diagnostics | A 64-entry in-memory lifecycle ring records generation-scoped session, connection, and stream transitions. The runtime report includes the redacted session stamp, supervisor attempt, operation/session/stream counts, aggregate render latency, dropped bytes, and parser epoch; it allowlists fields and excludes endpoint/route IDs, addresses, paths, terminal content, candidate IDs, and raw errors. The report reaches the clipboard only after the user taps `Copy redacted report`. Unit tests enforce redaction and ring eviction, while the connection-screen test enforces the explicit 48dp action. |

## 4. Device, pairing, and connection features

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Device registry | Persist endpoint projections through Go, list saved devices, show normalized OS icon, label, authorization state, reachability, Direct/SSH/Cloud availability, and observed P2P/Relay path. Remote routes without Go-projected `credential_ref` render as Authorization required and do not start Cloud presence/session probes. Terminal count without an unnecessary session remains. |
| In progress | Refresh | Pull-to-refresh and toolbar refresh update the registry and authenticated Cloud Edge presence through a dedicated Go probe without opening an interactive terminal session. Presence refreshes after foreground resume and is shown separately from endpoint enablement. Go connection planning now queries native Android NSD and iOS Bonjour for `_anytty._tcp`, validates bounded TXT metadata in Go, and accepts only pinned endpoint identities; both simulators discovered the live daemon. Explicit local-discovery projection in the device list and terminal counts without unnecessary sessions remain. |
| In progress | Pairing | Scan an AnyTTY QR code with native camera permission and exact-once cleanup; support pasted pairing command/code; reject unrelated input without transmitting it. Android camera/paste handling and an iPhone simulator denial-to-paste flow against an isolated real daemon are verified; camera success and physical-device permission behavior remain. |
| In progress | Reauthorization | Remote routes without a capability credential open a target-scoped fresh-pairing flow instead of a failing terminal session. `expected_endpoint_id` prevents a code for another endpoint from mutating the selected device; Android verified credential restoration and a successful terminal reconnect after a config-only share. iOS physical-device verification remains. |
| In progress | Endpoint sharing | The existing scan/paste entry recognizes one-time share offers, receives a Go-verified preview, shows identity, route and policy changes plus local credential omissions, and commits only after explicit confirmation. Android verified a real preview/atomic commit and Authorization required result; iPhone verified preview cancellation without registry mutation. Physical-device camera and network permission behavior remain. |
| In progress | Device editing | A compact row action and safe-area-aware bottom sheet edit the display label through Go `EndpointUpsert` and can make the endpoint the registry default. Route editing and selection policy are now endpoint-scoped full-screen workflows. Icon preset/custom image, image compression, and invalid-source handling remain. Android and iPhone layouts plus narrow widget coverage are verified; an iPhone simulator rename against an isolated real daemon confirmed authoritative registry mutation. Default mutation and physical-device behavior remain. |
| In progress | Disconnect/delete | Confirmed actions call Go `EndpointDisconnect` to end the pooled physical session and `EndpointDelete` to atomically remove saved routes and authorization. Operation-handle matching, release, and authoritative registry projection are tested; real-device success/failure behavior and active-transfer conflict presentation remain. |
| In progress | Route manager | A dedicated screen lists every Go-projected route with priority, availability, enablement, safe last-route guarding, and compact actions. User-owned Direct/SSH routes support atomic add/edit/delete/reorder, Direct advanced split ports, exact route-override tests, and SSH credential provisioning whose private key stays in platform secure storage. Cloud/local routes are visible but source-managed. Narrow widget coverage and foreground iPhone/Android layouts are verified. An isolated Android endpoint passed exact Direct-route dialing and generated a real non-exportable P-256 SSH key through Android Keystore; the iOS Keychain/Secure Enclave bridge builds for Simulator/device. Cloud route lifecycle, real SSH host/key installation, physical-device behavior, and drag reordering remain. |
| In progress | Connection policy | A dedicated mobile connection screen reads and applies the Go-owned Auto/Direct/SSH/Cloud preference, Cloud Auto/P2P/Relay, and Relay UDP/TCP policy. Per-route availability and failure reasons come only from Go. Apply explicitly disconnects the pooled endpoint so the next session uses the saved policy. Unit coverage verifies native handle matching/release; an isolated Android endpoint verified Auto to Direct persistence, reconnect, and restoration to Auto. Route creation/editing and physical-device failure handling remain. |
| In progress | Connection diagnostics | The same screen refreshes the exact ReadySession snapshot and shows the selected route/path, candidate types and protocols, selected/base addresses, same-NAT explanation, RTT, traffic, packet/loss counts, generation, selection reason, and sample time. Long candidate-pair IDs are compact with an explicit full-value copy action. A separate explicit action copies the bounded allowlisted runtime report without those private values. Live iPhone and Android screenshots are verified. Cloud Edge identity/region and stable typed failure actions remain. |

## 5. File and transfer features

| Status | Capability | Required behavior |
| --- | --- | --- |
| In progress | Navigation | List paged directories, preserve Unix/Windows drive/UNC roots, breadcrumbs, parent navigation, refresh, hidden files, and empty/error states. Symlink and hard-link traversal rules plus complete offline handling remain. |
| In progress | Sorting | Folder-first plus name, date, size, and type ascending/descending. Android is verified; iOS and locale-aware natural sorting remain. |
| In progress | Operations | Create directory, rename, recursive delete, multi-select, copy, cut/move, paste, copy paths, and retain a clear Back-stack priority for inline edits and confirmations. Android is verified; iOS remains. |
| In progress | Bookmarks | Per-device client-local bookmarks preserve the Web storage contract and support add current path, open, rename alias, and remove. Android layout, keyboard avoidance, persistence, and layered Back behavior are verified; iOS runtime verification and one-time legacy daemon import remain. |
| In progress | Upload/download | Native picker upload, native destination export, daemon-windowed resource frames, digest verification, MIME projection, progress, cancel, in-place failed-task retry, and resumable download offsets/upload handles are implemented. Android upload/download, failed-download retry, route-independent task ownership, a 128 MiB paused download, and a 512 MiB paused upload are verified end to end with matching SHA-256 digests. Picker-owned cache copies are retained while resumable and removed on completion/cancel. iOS runtime verification and background execution remain. |
| In progress | Transfer center | The Web reference's full-screen center, direction/path/device details, byte progress, active/paused/failed/done status, newest-first ordering, per-item pause/resume/retry/cancel/remove, selection mode, bulk pause/start, clear completed/failed, app-global ownership, compact workspace entry, and completed-download system open are implemented. Android verifies route-independent ownership, layered Back, selection layout, and opening a digest-verified text download in the system viewer. iOS runtime verification remains. |
| In progress | Preview | Bounded, digest-verified API text and raster-image previews are available with selectable text, pinch zoom, and decode dimensions capped at 2048 pixels. Markdown search/wrap/zoom, image rotation, streamed media, archives/documents, and any maintained 3D renderer remain. |
| Complete | Preview safety | Go resolves complete symlink chains and rejects loops, non-regular files, unsupported MIME types, sources above 64 MiB, and preview payloads above 4 MiB. Raster images must be returned whole. `FilePreviewResult.sha256` covers the exact bounded content and Flutter verifies it before rendering; a forged, oversized, unsupported, or truncated-image response fails closed. Tests cover Go type/size/symlink/digest behavior, Flutter validation, bounded image decode, and a 48dp Close action while the remote Future never completes. |

## 6. App shell and platform features

| Status | Capability | Required behavior |
| --- | --- | --- |
| Pending | Localization | Simplified Chinese and English for all user-facing states; technical identifiers remain untranslated. |
| In progress | Settings | Light/dark/system app appearance and terminal settings are implemented with a live terminal preview, 17-palette swatch picker, six-family font sample picker, font-size/cursor preview, adaptive switches/sliders, background connection/notification controls, reduced-motion-aware picker alignment, and bottom safe-area protection. The terminal-tools shortcut uses the same six-font source, renders each menu entry in-family, and updates a compact preview without leaving the terminal. Theme selection opens with the current palette and its complete group context visible, including at 200% Android text. Connection policy and route management now live in endpoint-scoped full-screen workflows; privacy policy and build/version information remain. Android and iPhone simulator layouts are screenshot-verified. |
| In progress | Native Back | One event closes exactly one latest terminal/file surface on Android. Predictive Back animation and iOS swipe-back preservation remain. |
| In progress | Haptics | Cross-device group expansion and terminal selection use bounded selection feedback. Remaining meaningful actions and a user/platform accessibility policy remain. |
| In progress | Secure storage | Device and SSH signing keys remain in Keychain/Android Keystore-backed adapters and never enter Flutter/Dart memory as private key bytes. Android SSH lookup/create/sign/delete compiles and real provisioning is emulator-verified; the iOS bridge uses Secure Enclave on devices with a non-exportable simulator Keychain fallback and builds for both targets. Physical-device verification remains. |
| In progress | Background connection | Demanded Go endpoint sessions project into a thin platform adapter. Android runs a `connectedDevice` foreground service only while the app is backgrounded and demand remains; foreground resume stops it without rebuilding the engine or losing the terminal frame. iOS uses a finite UIKit background task and deliberately does not claim an unsupported persistent background mode. Android service lifecycle, same-process resume, notification visibility, and tap return are emulator-verified. iOS channel/settings behavior is simulator-verified; real-device suspension limits, user/system termination, and durable process-restart recovery remain. |
| In progress | Notifications | Go-owned terminal-exit and completed-transfer events are deduplicated and translated into local notifications only while the app is backgrounded and the user has opted in. Permission is requested only from the settings action, denial leaves the setting off, and notification payloads carry an exact device/terminal route. Android permission, ongoing connection notification, and tap return are emulator-verified; iOS permission grant/state synchronization is simulator-verified. Real iOS event delivery and notification deep links remain. |
| In progress | Native files | Scoped upload picking, atomic partial downloads, MIME/type projection, Android streamed document export/system-open, and iOS document export/share compilation are implemented. Android system-open is verified. iOS runtime verification and durable destination persistence remain. |
| Complete | Startup recovery | Runtime construction records bounded classified startup state and closes partially initialized lifecycle, background, and native owners before surfacing failure. The device screen never renders the original exception: it offers a fresh Runtime retry, an explicit copy of an eight-event allowlisted startup report, and a confirmed reset of only the local device-registry blob. Reports and UI exclude endpoint IDs, paths, raw errors, and remote state. Widget tests inject private failure text and verify redaction, retry, 48dp actions, and reset confirmation. |
| Pending | Privacy and release | Privacy manifests, permission purpose strings, dependency notices, signing, Play 16 KiB page compatibility, App Store/Play artifact checks, and reproducible release scripts. |

## 7. Mobile design contract

The app is a work surface for developers and operators, not a marketing page.
It uses quiet full-width layouts, dense lists, and unframed terminal/file
surfaces. Cards are reserved for repeated device/terminal records, modal
content, and actionable empty/error states. Cards are never nested.

- Palette: graphite-neutral light/dark surfaces with a restrained teal
  interaction accent, green for healthy/success, amber for waiting/degraded,
  and red for blocked/destructive. It is intentionally independent from the
  Web chrome while terminal palettes retain their own colors. State always
  includes text or an icon, never color alone.
- Typography: platform system sans for UI and a bundled monospace for terminal
  cells and technical values. UI letter spacing is explicitly zero.
- Motion: 120-240 ms state transitions only when they communicate hierarchy,
  loading, or completion. Terminal painting does not use decorative animation.
  Reduced motion disables route/panel/status transitions and inertia.
- Signature: the existing five-bar connection pulse identifies bounded loading
  and recovery states; it becomes static under reduced motion. History position
  stays overlaid at the terminal edge without adding a second status card.
- Geometry: stable toolbar/keybar heights, safe-area padding, 4/8dp spacing,
  and explicit pane constraints prevent keyboard, labels, or loading states
  from resizing the terminal unexpectedly.

The 2026-08-31 visual pass exercised the settings preview, resource summary and
detail trends, terminal inventory, and terminal switcher on the foreground
Android emulator. The same settings surface was built, installed, launched by
route, and screenshot-checked on an iPhone 17 Pro simulator. These checks found
and corrected nested visual framing, a duplicated sheet handle, excessive
initial sheet height, narrow resource-detail overflow, and settings content
underlapping the Android navigation safe area.

The background controls were then checked on both foreground simulators. The
Android pass granted notification permission from the app, kept the same app
process and demanded Go terminal alive behind a `connectedDevice` foreground
service, displayed its system notification, and returned to the exact mounted
terminal when the notification was tapped. The iPhone pass exercised both the
denied and granted system permission paths and verified that the adaptive
switch reflects the authoritative authorization result without overlap at the
bottom safe area.

The endpoint connection pass then read a real Go policy and ReadySession
snapshot on both foreground simulators. It verified the compact current-path
summary, disabled route reasons, Cloud path/transport controls, fixed apply
footer, and expanded transport diagnostics in dark iOS and light Android
appearances. The first live diagnostic screenshot exposed an excessively tall
candidate-pair identifier; the mobile presentation now abbreviates it while a
copy icon preserves the full value. An isolated Android endpoint completed an
Auto to Direct apply/reconnect cycle and was restored to Auto afterward.

The route-management pass then opened the same isolated endpoints on foreground
Android and iPhone simulators. A first Android screenshot showed disabled sort
buttons and a disabled final-route switch that visually implied the route was
offline; the refined row hides irrelevant ordering controls, preserves an active
switch with an explicit last-route guard, and uses the route kind as the default
display title. Add/edit and action surfaces were screenshot-checked, and the
Android app successfully opened and closed an isolated session with the exact
`direct` route override. A subsequent native-platform pass provisioned an SSH
credential through the real Go command and Android Keystore bridge, displayed
the resulting ECDSA public key/fingerprint, and compiled the equivalent iOS
Keychain/Secure Enclave bridge. SSH form validation now rejects missing remote
AnyTTY setup/ICE addresses before calling Go.

The settings and monitoring pass used Moshi's compact preview-first selection
pattern as a reference without copying its branding. Android and iPhone
screenshots verified candidate-level font samples for all six legacy choices,
ANSI swatches for all 17 themes, automatic current-theme alignment, reduced
motion and bottom safe areas. The same pass added a terminal-list aggregate for
current daemon CPU, resident memory, and reporting coverage above the existing
per-terminal sparklines and 64-sample detail sheet. A follow-up foreground pass
verified the three-way light/dark/system selector and live two-second resource
refresh on both platforms. The device list now uses the authenticated Go Cloud
presence probe for Online/Offline state and keeps Direct/SSH enablement distinct
from reachability.

The settings accessibility follow-up made Light/Dark/System mutually exclusive
48pt/48dp buttons with an explicit selected state, while keeping the live
terminal preview fully visible in landscape. Font and theme entry controls and
their candidate rows now expose one concise label each; the selected candidate
is announced without repeating the visual sample. Keyboard mode, the adjustable
scroll-inertia value, cursor/resize toggles, background connection, and terminal
notification controls now combine their visible names with the platform role
and current value. iOS native frames and Android 420dpi UIAutomator bounds were
checked against the rendered screenshots.

The subsequent touch-target pass used the rendered Android accessibility tree,
not only widget declarations. It found and corrected 36-46dp hit regions in the
terminal filters, resource summaries, switcher title, tools panel, file path,
transfer center, connection diagnostics, pairing flow, route actions, and Fn
panel. Opening terminal tools now temporarily yields the two-row quick-key bar,
which exposes more full-size actions without adding another nested panel. An
Android 16 physical phone then loaded a real 26-terminal endpoint, displayed
the 6/6 reporting aggregate and per-terminal resource trends, and confirmed the
portrait inventory/filter layout with live daemon data. The phone was used only
for this install/read-only display check; continuing automation remains on the
foreground emulator.

The local-discovery pass completed the Go platform request on Android with NSD
and on iOS with Bonjour. Discovery starts on demand, keeps a bounded 30-second
cache, stops after idle, and returns only numeric addresses with the advertised
protocol and identity-derived key; Go remains responsible for expiry, pinning,
route selection, and final authentication. Cold start does not prompt solely
for discovery. Foreground Android and iPhone simulator runs each returned three
validated candidates and opened the real endpoint through the normal Go session
plan. Physical-device local-network permission behavior remains a release gate.

The endpoint-sharing pass then used real one-time TLS share sessions on both
foreground simulators. Android previewed and committed a config-only Direct
Route, projected the resulting missing capability as Authorization required,
and used a fresh target-constrained pairing claim to restore the credential and
open the daemon's terminal inventory. iPhone previewed the same identity, route,
policy, and credential effects and then cancelled; its existing registry entry
remained unchanged. The preview and authorization screens fit without overlap
on both platform viewports and keep the confirmation action in the lower touch
zone.

The landscape and accessibility pass then verified the real terminal inventory,
active canvas, fixed key bar, software keyboard, and split-pane restoration. The
inventory becomes two columns at 720 logical pixels and respects Android's
three-button navigation rail and the iPhone Dynamic Island safe area. With the
Android software keyboard open, the app bar yields to a single-row accessory and
a split workspace focuses its active pane without unmounting the others; closing
the keyboard restores the original tree and ratio. Native accessibility trees
on both platforms expose the bounded current terminal snapshot as exactly one
focusable read-only canvas element, after removing unnamed pane/drag semantics,
the hidden IME editor, and unnecessary non-overflow scroll nodes. Android
reported a `[128,210][2274,802]` frame for a 96x11 terminal, while iOS reported
a 750x228pt frame for an 88x11 terminal. History rows are exposed lazily and
carry their selection state. Hands-on screen reader navigation and the iOS
software-keyboard path remain.

## 8. Migration order and gates

### Phase 0: foundations

1. Install and pin Flutter, Dart, Zig, CMake, Ninja, CocoaPods, Android SDK/NDK,
   and Xcode tooling.
2. Create `clients/flutter` without changing `clients/ui` or the existing
   Capacitor project.
3. Generate Dart Proto models from the repository schemas.
4. Wrap the Go C ABI and libghostty-vt behind versioned AnyTTY-owned plugin
   APIs with deterministic ownership and isolate/thread rules.

Gate: a Flutter unit test can create/close the Go engine and a libghostty-vt
terminal 500 times with stable handle counts and no native crash.

### Phase 1: terminal vertical slice

1. Device fixture or selected endpoint opens one Go session.
2. Attach one terminal, establish the lossless bootstrap, and render it through
   libghostty-vt in Flutter.
3. Support text/IME/keybar input, resize ownership, foreground/background
   resume, full recovery, and terminal close.
4. Verify real shell, Codex-like repainting TUI, wide characters, alternate
   screen, mouse mode, and 100,000-line output.

Gate: no blank frame, lost bootstrap bytes, replayed input, unbounded frame
queue, or WebView/JavaScript runtime in the artifact.

### Phase 2: complete terminal workflow

Migrate terminal inventory/management, frozen history, search/copy, touch
selection, settings/themes, splits, snippets, clipboard history, and
accessibility.

Gate: every row in section 3 is `Done` on both iOS and Android, including
portrait/landscape, largest Dynamic Type, VoiceOver/TalkBack, and reduced
motion.

### Phase 3: device and connection workflow

Migrate registry, pairing/share, reauthorization, route management, Cloud
presence, connection policy/diagnostics, and explicit disconnect/delete.

Gate: Direct, SSH, Cloud P2P, Cloud Relay UDP/TCP, offline, revoked access,
quota, and network-switch recovery scenarios pass on real devices.

### Phase 4: files and transfers

Migrate file navigation/operations, bookmarks, native pickers, resumable
transfers, transfer center, and maintained preview formats.

Gate: upload/download interruption, background/foreground, digest mismatch,
Windows paths, symlink loops, large files, and system-open behavior pass on both
platforms.

### Phase 5: replacement and release

Complete notifications/background service, diagnostics/privacy/release
metadata, artifact boundary checks, migration of persisted user intent, and
beta distribution. Remove Capacitor from native build/release jobs only after
feature parity and rollback validation. The TypeScript Web client remains.

Gate: iOS and Android release artifacts contain Flutter, the pinned native
libraries, and Go core; they contain no bundled mobile Web application or
JavaScript terminal runtime.

## 9. Definition of done

A feature is marked `Done` only when:

1. Its behavior is implemented without importing TypeScript mobile code.
2. Unit or integration tests cover normal, cancellation, stale-generation, and
   relevant error paths.
3. It has been exercised on both an iOS simulator/device and an Android
   emulator/device as appropriate.
4. Touch, Back navigation, safe areas, localization, Dynamic Type, screen
   reader semantics, and reduced motion have been checked where applicable.
5. Native handles, streams, subscriptions, and temporary files are released on
   close, route change, engine replacement, and process exit.
6. The matching row in this document is changed from `Pending` to `Done` in the
   same change.
