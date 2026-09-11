# Agent workbench plugin

This is an independent AnyTTY plugin with daemon and TUI components. Its daemon
service owns Agent state; one TUI component aggregates state from its connected
endpoints. Every SDK request, broadcast, mount update, and click operation travels
through the selected daemon, including self-directed TUI actions. No Agent output
scraping or model inference is used to guess authoritative status.

## Build and install

From the repository root:

```sh
mkdir -p .artifacts/plugins/agents
go build -o .artifacts/plugins/agents/anytty-agent-plugin ./cmd/anytty-agent-plugin
cp plugins/agents/anytty-plugin.toml .artifacts/plugins/agents/
anytty plugin install "$PWD/.artifacts/plugins/agents"
anytty plugin agents install codex --config-dir /explicit/codex/config
anytty plugin agents install opencode --config-dir /explicit/opencode/config
```

Install the AnyTTY plugin on each daemon machine that runs Agents, and on the
machine running the TUI. Link/install registers the package for new host
instances; it does not stop or replace an already running daemon. The executable
path in each Hook configuration must be a built AnyTTY binary supporting the
plugin commands. Plugin packages run as trusted OS processes, not in an OS sandbox.

Codex discovers `hooks.json` next to its active configuration. After installing,
open Codex `/hooks` and review/trust the new definitions through its own UI. The
installer does not edit its trust database or bypass the review. OpenCode loads
`plugins/anytty-agents.js` from its selected configuration directory at startup.

To remove only these integrations:

```sh
anytty plugin agents uninstall codex --config-dir /explicit/codex/config
anytty plugin agents uninstall opencode --config-dir /explicit/opencode/config
anytty plugin uninstall org.anytty.agents
```

Other hook groups and configuration fields are preserved. Installation is
idempotent; an unmanaged OpenCode file with the same name is not overwritten.

## UI and interaction

The plugin declares an `agents.navigator` workspace sidebar. The host resolves that
logical surface for the current TUI, so the plugin does not bind itself to a tab or
panel ID. It declares card styles, endpoint, provider, project or session title,
status, and last observed update time. Agents needing attention
(blocked/error/stale) sort first. In the existing system menu, use `a` for the next
plugin mount or `A` for the previous one. Inside the sidebar, use arrows or a mouse
click a card to open its terminal, Enter or a double-click to open a plain row, `f`
to toggle the attention filter, `x` to hide the declared surface, and Escape to
return to the previous content panel. Each terminal panel also gets a status badge
filtered by its binding. Opening a card declares `focused_panel`; filter declares
`none` and still travels through the daemon. Hide is a host-owned surface behavior
declared by the plugin, so it takes effect locally without granting a terminal
target.

The clicked row carries its stable daemon and terminal identity. The interaction
contains the original TUI, workspace, tab, panel, and binding revision; asynchronous
work does not reinterpret a later focus change. The aggregate view has one
control endpoint, while individual Agent rows can belong to other endpoints.
The host initializes real mount owners and performs the actual binding operation.

## Hook behavior and boundaries

Both adapters require `ANYTTY_TERMINAL_ID` and `ANYTTY_DAEMON_SOCKET`. Outside an
AnyTTY terminal they are no-ops. Those values are injected by the terminal-owning
daemon; hook stdin cannot override the terminal binding. Codex JSON stdin and
OpenCode JavaScript callbacks are external integration formats. The adapter
converts them into generated `PluginAgentReport` Protobuf messages before sending
anything within AnyTTY. Prompt text, tool arguments and transcript content are
not read or forwarded.

| Provider event | State |
| --- | --- |
| Codex SessionStart startup/resume/clear | idle, new source epoch |
| Codex SessionStart compact | preserve current state |
| Codex UserPromptSubmit / PreToolUse / PostToolUse | working |
| Codex PermissionRequest | blocked |
| Codex Stop / Interrupt | idle; not a claim of whole-task success |
| Codex SessionEnd | exited |
| Codex SubagentStop | ignored; never finishes the parent |
| OpenCode session.created | idle |
| OpenCode session.updated | update metadata, preserve known state |
| OpenCode session.status busy/retry/idle | working/working/idle |
| OpenCode permission.asked / permission.replied | blocked while any request remains |
| OpenCode session.error / session.deleted | error / exited |

Unknown OpenCode status values become `unknown`. Multiple pending permissions
remain tracked individually. OpenCode callbacks maintain complete per-session
snapshots and asynchronously serialize reports, coalescing pending snapshots by
session; this preserves answered permissions during bursts and does not hold up
the Agent loop. Each child reporter has a 1.5-second hard timeout. A failed
snapshot gets at most three asynchronous attempts, with 100 ms and 200 ms retry
backoff; a newer session snapshot replaces an older pending retry. The OpenCode
CLI reporter returns a failure status for the adapter to consume, while the
OpenCode event callback itself never fails or waits for that reporting work.
After all attempts fail, reporting remains best effort and the next lifecycle
event refreshes the complete state. Failed reporting never changes an approval
decision. The timeout terminates only the child reporter started by the adapter,
never the daemon it connects to.

Codex hook programs are separate concurrent processes, and official input does
not supply an event timestamp or sequence. The adapter uses a locked, persistent
per-session clock to establish **adapter observation order**, rejecting old
network deliveries by epoch and sequence. This does not reconstruct undocumented
Agent-internal ordering. Resume starts a newer epoch; compact does not. Every
Codex invocation emits `{}` and does not request cancellation, add instructions,
or approve/deny tools.

The daemon service persists Protobuf snapshots, including base state and pending
permission IDs, using the SDK's optimistic state revision. Reconnecting TUI feeds
watch an atomic initial snapshot and subsequent full state changes; disconnected
feeds retain their last rows marked stale. After a background service restart,
restored non-exited rows are also marked stale until a new hook report arrives.
Agent state is scoped to the owning
daemon. Identically named sessions or terminals on different daemons do not collide.

Each mount has at most one update awaiting the host's application acknowledgment;
new feed changes coalesce while it is pending. Lost acknowledgments and reconnects
trigger a nonce-correlated Init request for the host's authoritative revisions,
then resend a complete mount. Interactions use the exact displayed snapshot,
including when they arrive before its acknowledgment. The control endpoint stays
fixed for the TUI process: if it is offline, the displayed aggregate freezes while
other endpoint subscriptions continue, and refreshes when control reconnects.

## Validation

```sh
go test -race ./plugins/agents ./cmd/anytty-agent-plugin
ANYTTY_TEST_OPENCODE=1 go test ./plugins/agents -run TestInstalledOpenCodeLoadsPlugin -count=1 -v
ANYTTY_TEST_CODEX=1 go test ./plugins/agents -run TestInstalledCodexParsesUntrustedHooks -count=1 -v
```

The regular suite tests Codex lifecycle payloads, compact and SubagentStop,
concurrent adapter sequencing, old-epoch rejection, multiple permissions, complete
snapshot coalescing, Protobuf persistence/recovery, preservation of user hook
configuration, actual Node execution of the OpenCode plugin, and a two-endpoint
TUI process through the framed Protobuf SDK.

The opt-in integration test starts its own OpenCode loopback server using isolated
config/data/cache/state directories, installs the real JS plugin, and creates and
deletes a session through the official HTTP API. It invokes no model and cleans
up only that test server. Verified with OpenCode 1.18.20. A separate Codex 0.154.0
integration loads the installer's eight definitions as process-local configuration
overrides and queries the real app-server `hooks/list`; all are parsed and remain
untrusted. It does not modify user config or `CODEX_HOME`, create a model thread,
or run the hooks. Discovery from a user's active config layer and trusted
execution remain separate steps; actual trusted execution inside Codex requires
the user's `/hooks` review and is not claimed by the fixture tests.

Official references (checked 2026-09-11):

- [Codex hooks](https://learn.chatgpt.com/docs/hooks)
- [OpenCode plugins](https://opencode.ai/docs/plugins/)
- [OpenCode server API](https://opencode.ai/docs/server/)
