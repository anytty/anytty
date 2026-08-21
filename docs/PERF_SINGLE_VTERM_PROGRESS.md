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

## Checkpoints

- `99e7c68`: combined VTerm live-damage and history-delta API plus differential
  tests.
- Core single-emulator owner and typed history queue: implementation complete,
  checkpoint commit pending final review.

## Verification so far

- `go test ./vterm/...`: passed before the core migration; new targeted VTerm
  ownership tests pass.
- `go test ./core/...`: passed after the core migration.
- Targeted tests pass for queue order, flush, overflow gap, slow-history live
  invalidation, lifecycle ownership, CJK eviction, and alt presentation overlay.

## Memory data

- Baseline reference supplied at goal start: five idle 157x79 terminals, about
  73 MB daemon physical footprint and 43 MB Go HeapAlloc.
- Reproducible before/after and 64 MB burst measurements: pending benchmark
  harness implementation.
