# Single-VTerm Memory Optimization

Branch: `perf/single-vterm`

Baseline: checkpoint `f9eb081` in the untouched main worktree.

## Design decisions

- `SurfaceTrack` owns the only production VTerm. Its `SemanticSource` wraps the
  same VTerm and one write returns both live damage and immutable line-history
  transactions.
- Raw PTY output has one consumer. History persistence consumes typed deltas on
  a bounded nonblocking queue; overflow persists a gap boundary without
  resetting the live parser.
- History queries freeze live writes, drain the typed queue, and then snapshot
  the same emulator, preserving the cold/pending/hot fence.
- Core lifecycle rows carry persisted ownership while they remain visible, so
  explicit lifecycle history is not appended again on eviction.
- Preserved alt-screen exit frames are presentation overlays. They are never
  replayed as synthetic ANSI into the authoritative emulator.
- Restart seals the old visible tail, creates a fresh parser around the
  preserved visible frame, and clears old process modes and partial escape
  state.
- The alternate screen is allocated only on first DEC alt-screen entry.
  Line-history deltas omit converted screen frames because queries read the hot
  frame from the shared VTerm under the history fence. Used-row projections
  cache only their logical width until a caller explicitly requests a full row.

## Checkpoints

- `99e7c68`: combined VTerm live-damage and history-delta API plus differential
  tests.
- `dadafec`: core single-emulator owner, typed asynchronous history queue,
  lifecycle migration, and presentation-only preserved alt frame.
- `7e32d6f`: lazy alternate screen, compact used-row cache, and removal of
  redundant screen frames from production line-history deltas.
- `25668eb`: reproducible idle/burst benchmark and benchmark-only explicit GC
  fence shared by baseline and current binaries.
- `9508623`: chronological CLI history search starts at the oldest frozen line;
  this fixed the burst verifier's cold-history BEGIN lookup.

## Verification so far

- `go test ./vterm/...`: passed after the allocation changes.
- `go test ./core/...`: passed after the allocation changes.
- Targeted tests pass for queue order, flush, overflow gap, slow-history live
  invalidation, lifecycle ownership, CJK eviction, alt presentation overlay,
  lazy alt allocation, and boundary-only line-history deltas.
- `go test -race ./vterm/... ./core/...`: passed.
- `go test ./...`: passed with the host session's `ANYTTY` and
  `ANYTTY_TERMINAL_ID` markers removed so nested-TUI guards see a normal test
  environment.
- `git diff --check`: passed.

## Memory data

- Reproducible harness: `scripts/benchmark-single-vterm-memory.sh`. It builds
  checkpoint `f9eb081` and the current tree with the same benchmark-only SIGUSR1
  GC fence, starts isolated daemons, and excludes child-shell memory.
- Intermediate pre-frame-removal measurements showed 67,372,480 B baseline vs
  44,516,864 B current (33.92% reduction), identifying the remaining converted
  frame cache as the acceptance blocker.
- Post-frame-removal idle trial, five 157x79 `/bin/zsh -f` terminals:
  baseline physical median 66,700,672 B and HeapAlloc 48,528,128 B; current
  physical median 30,901,696 B and HeapAlloc 11,192,520 B. Physical footprint
  fell 53.67% and is 29.47 MiB, satisfying both idle thresholds.
- Final clean-code run at `9508623`, five idle 157x79 `/bin/zsh -f`
  terminals: baseline physical median 65,586,688 B, current 32,228,736 B
  (30.74 MiB), a 50.86% reduction. Current HeapAlloc/HeapInuse/HeapSys were
  17,903,808 / 20,414,464 / 23,920,640 B.
- The 64 MiB burst completed in 7 seconds with BEGIN and END found in
  authoritative history, zero dropped bytes, zero gap boundaries, and history
  available. Peak HeapAlloc was 98,838,048 B. After explicit GC and settling,
  HeapAlloc/HeapInuse were 18,609,088 / 21,094,400 B, physical median was
  51,136,128 B, the five-second NumGC delta was 1, and daemon CPU was 0.7%.
  These figures show prompt heap recovery and no sustained GC thrash; HeapSys
  remains reserved after the burst and can be reused by later output.
- Final artifacts: `.artifacts/single-vterm-memory/final-9508623` (intentionally
  ignored by git).
