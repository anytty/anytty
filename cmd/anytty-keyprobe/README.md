# TUI keyboard probe

This development tool uses the production `terminalhost.Host`, `InputParser`,
and `input.RouteWithOptions`. It records raw input chunks and parsed events,
plus the default/configured normal-mode route and the bytes available when
shortcuts are bypassed. It never executes shortcut actions or forwards keys.

Build from the public repository root:

```sh
go build -o .artifacts/keyprobe/anytty-keyprobe ./cmd/anytty-keyprobe
```

Run first in the outer terminal, then run the same binary in a shell inside
AnyTTY TUI, using a different output filename:

```sh
.artifacts/keyprobe/anytty-keyprobe -output /tmp/keyprobe-outside.jsonl
.artifacts/keyprobe/anytty-keyprobe -output /tmp/keyprobe-inside.jsonl
```

Press Ctrl+/ (the reported key), Ctrl+backslash, Ctrl+C, Ctrl+], Ctrl+_, and the problematic function keys.
Three separate Esc presses (pause briefly between them) exit, or wait for the
default two-minute timeout. Ctrl+C and Ctrl+backslash are captured as data.
Output files are created exclusively with mode 0600; choose a new name on reruns.
Only this process's stdin is recorded, not global keyboard activity.

The outer capture requests exactly the TUI's keyboard enhancement (Kitty flag 1).
The inner capture observes what actually reaches the child PTY **after** the
outer TUI has processed it. A missing inner event can be an outer shortcut;
consult the outer event's `route` and `action`. `forced_output_hex` is only a
simulation of bypassing shortcuts, not evidence of actual delivery. Capture
routes model normal mode; overlays/history/prefix modes can consume keys earlier.
Pass `-config /path/to/tui-v3.yaml` to simulate the same shortcut configuration.

For Ctrl+backslash, the expected raw input is `1c` (legacy) or
`1b5b39323b3575` (`ESC [ 92 ; 5 u`). The expected routed `output_hex` is `1c`.
If the inner capture sees `1c`, investigate the target application's binding and
PTY signal settings; delivery alone does not guarantee a visible action.

## Exhaustive protocol corpus

```sh
.artifacts/keyprobe/anytty-keyprobe -mode matrix -output /tmp/keyprobe-matrix.jsonl
```

The corpus enumerates all eight Ctrl/Alt/Shift combinations for printable ASCII,
Enter/Tab/Escape/Backspace, navigation/editing keys and F1-F24; it also covers all
128 ASCII bytes and their Alt-prefixed forms, xterm modifyOtherKeys encodings,
and selected Kitty repeat/release/alternate/text-field variants. No OS shortcuts,
Super/Command/Hyper/Meta combinations, arbitrary keyboard layouts, or device
firmware are simulated. Different encodings of one key are separate samples;
these counts are **not counts of broken physical keys**. Shifted ASCII samples
exercise decoder inputs; not every sample is emitted under Kitty flag 1.

The JSONL contains the exact bytes and result for every sample:

- `forwarded`: normal-mode router produces terminal bytes.
- `shortcut`: a configured TUI shortcut owns the input; not a lost key.
- `parser-unsupported`: parser produces unknown-key or host-control.
- `no-terminal-encoding`: a key is parsed, but the router produces no PTY bytes.
- `release-ignored`: key release intentionally produces no terminal input.
- `no-event` / `multiple-events`: framing produced zero/multiple events.

The scan directly injects protocol bytes; it does not prove the user's terminal
sends them. Automated tests additionally verify arbitrary read fragmentation and
send control keys through a real raw PTY and the production Host on macOS/Linux.

## Compatibility fixes verified by the probe

- Ctrl+`/` and Ctrl+`2` through Ctrl+`8` now produce their legacy control bytes
  after CSI-u or xterm modifyOtherKeys decoding. Explicit digit shortcuts still
  require an unambiguous digit event; control-code aliases cannot trigger them.
- CSI P/Q/S encodings now recognize F1/F2/F4 with Shift/Alt/Ctrl. F3's SS3 R and
  CSI 13~ forms work. CSI R remains a cursor-position response because its bytes
  are indistinguishable from that response.
- xterm `CSI 27;modifier;codepoint~` is decoded and re-encoded for the target PTY;
  host protocol bytes are not blindly forwarded.
- CSI-u alternate-key and associated-text fields are validated. The primary key
  remains the shortcut identity; shifted/associated text is preserved for input.
  Malformed Unicode, unsupported modifiers and releases produce no PTY input.

The matrix still reports protocol limits: PUA functional keys outside the input
model (such as F13-F24), ambiguous CSI R, and combinations with no supported
legacy encoding (such as Ctrl+Enter/Tab and certain Ctrl+punctuation). Supporting
these distinctly requires target keyboard protocol negotiation; replacing them
with ordinary characters would change their meaning.

Reference wire formats: [Kitty keyboard protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/)
and [xterm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html).
