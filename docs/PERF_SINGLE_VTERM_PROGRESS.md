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

## Verification so far

- `go test ./vterm/...`: passed after the allocation changes.
- `go test ./core/...`: passed after the allocation changes.
- Targeted tests pass for queue order, flush, overflow gap, slow-history live
  invalidation, lifecycle ownership, CJK eviction, alt presentation overlay,
  lazy alt allocation, and boundary-only line-history deltas.
- Race and full-repository suites: pending final validation.

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
- Final clean-commit idle rerun and 64 MiB burst verification: pending.
